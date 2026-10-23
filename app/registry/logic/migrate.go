package logic

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/sync/errgroup"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// MigrationEndpoint describes a registry, with credentials supplied separately
// from its address so credentials never enter references or reports.
type MigrationEndpoint struct {
	Address, Username, Password, CAFile string
	PlainHTTP                           bool
}

type MigrationOptions struct {
	Source, Destination   MigrationEndpoint
	DryRun                bool
	Attempts, Concurrency int
	Workers               int
	ItemTimeout           time.Duration
	Report                io.Writer
	Progress              io.Writer
}

type migrationRecord struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag,omitempty"`
	Digest     string `json:"digest,omitempty"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

func newMigrationRegistry(endpoint MigrationEndpoint) (*remote.Registry, *http.Transport, error) {
	address := strings.TrimSpace(endpoint.Address)
	if address == "" || address != strings.ToLower(address) || strings.ContainsAny(address, "/@?#") || strings.ContainsRune(address, '\\') || strings.IndexFunc(address, unicode.IsSpace) >= 0 {
		// Registry references are hosts, without schemes, paths or credentials.
		return nil, nil, errors.New("仓库地址须为小写 host[:port]，不带协议、路径或凭据")
	}
	registry, err := remote.NewRegistry(address)
	if err != nil {
		return nil, nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = time.Minute
	if endpoint.CAFile != "" {
		data, err := os.ReadFile(endpoint.CAFile)
		if err != nil {
			return nil, nil, err
		}
		pool, err := x509.SystemCertPool()
		if err != nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(data) {
			return nil, nil, errors.New("CA 文件没有有效证书")
		}
		transport.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	}
	registry.PlainHTTP = endpoint.PlainHTTP
	registry.Client = &auth.Client{
		Client: &http.Client{Transport: transport}, Cache: auth.NewCache(),
		Credential: auth.StaticCredential(address, auth.Credential{Username: endpoint.Username, Password: endpoint.Password}),
	}
	return registry, transport, nil
}

func migrationRetry(ctx context.Context, attempts int, timeout time.Duration, fn func(context.Context) error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		err = fn(attemptCtx)
		cancel()
		if err == nil {
			return nil
		}
		if i+1 < attempts {
			timer := time.NewTimer(time.Duration(i+1) * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return err
}

// copyMigrationTag never selects a platform: the entire manifest/index DAG and
// discoverable referrers are copied, including all architectures and variants.
func copyMigrationTag(ctx context.Context, src oras.ReadOnlyGraphTarget, dst oras.Target, digest, tag string, concurrency int) error {
	opts := oras.DefaultExtendedCopyOptions
	opts.Concurrency = concurrency
	desc, err := oras.ExtendedCopy(ctx, src, digest, dst, tag, opts)
	if err != nil {
		return err
	}
	actual, err := dst.Resolve(ctx, tag)
	if err != nil {
		return err
	}
	if actual.Digest != desc.Digest || actual.Digest.String() != digest {
		return fmt.Errorf("目标 tag digest 校验失败: %s", tag)
	}
	return nil
}

// MigrateRegistry migrates every visible tagged graph. Untagged, unreachable
// manifests cannot be discovered through the standard catalog/tags APIs.
func MigrateRegistry(ctx context.Context, options MigrationOptions) error {
	if options.Workers == 0 {
		options.Workers = 3
	}
	if options.Attempts < 1 || options.Concurrency < 1 || options.Workers < 1 || options.ItemTimeout <= 0 {
		return errors.New("重试次数、镜像并发数、层并发数和单项超时必须大于 0")
	}
	if strings.EqualFold(strings.TrimSpace(options.Source.Address), strings.TrimSpace(options.Destination.Address)) {
		return errors.New("源和目标仓库地址必须不同")
	}
	src, srcTransport, err := newMigrationRegistry(options.Source)
	if err != nil {
		return fmt.Errorf("源仓库配置: %w", err)
	}
	defer srcTransport.CloseIdleConnections()
	dst, dstTransport, err := newMigrationRegistry(options.Destination)
	if err != nil {
		return fmt.Errorf("目标仓库配置: %w", err)
	}
	defer dstTransport.CloseIdleConnections()
	if options.Progress == nil {
		options.Progress = io.Discard
	}
	if options.Report == nil {
		return errors.New("必须提供迁移报告输出")
	}
	encoder := json.NewEncoder(options.Report)
	repositories := map[string]bool{}
	err = migrationRetry(ctx, options.Attempts, options.ItemTimeout, func(ctx context.Context) error {
		return src.Repositories(ctx, "", func(names []string) error {
			for _, name := range names {
				repositories[name] = true
			}
			return nil
		})
	})
	if err != nil {
		return fmt.Errorf("枚举源仓库失败: %w", err)
	}
	names := make([]string, 0, len(repositories))
	for name := range repositories {
		names = append(names, name)
	}
	sort.Strings(names)
	fmt.Fprintf(options.Progress, "发现 %d 个可见仓库；请确认账号具备全量 catalog/读取权限。\n", len(names))
	failed, succeeded := 0, 0
	type migrationTask struct {
		source, target *remote.Repository
		name, tag      string
	}
	var tasks []migrationTask
	for _, name := range names {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		source, err := remote.NewRepository(src.Reference.Registry + "/" + name)
		if err != nil {
			return err
		}
		source.Client, source.PlainHTTP = src.Client, src.PlainHTTP
		target, err := remote.NewRepository(dst.Reference.Registry + "/" + name)
		if err != nil {
			return err
		}
		target.Client, target.PlainHTTP = dst.Client, dst.PlainHTTP
		tags := map[string]bool{}
		err = migrationRetry(ctx, options.Attempts, options.ItemTimeout, func(ctx context.Context) error {
			return source.Tags(ctx, "", func(page []string) error {
				for _, tag := range page {
					tags[tag] = true
				}
				return nil
			})
		})
		if err != nil {
			failed++
			if e := encoder.Encode(migrationRecord{Repository: name, Status: "failed", Error: err.Error()}); e != nil {
				return e
			}
			continue
		}
		ordered := make([]string, 0, len(tags))
		for tag := range tags {
			ordered = append(ordered, tag)
		}
		sort.Strings(ordered)
		for _, tag := range ordered {
			tasks = append(tasks, migrationTask{source: source, target: target, name: name, tag: tag})
		}
	}
	var outputMu sync.Mutex
	group, workerCtx := errgroup.WithContext(ctx)
	group.SetLimit(options.Workers)
	for _, task := range tasks {
		if workerCtx.Err() != nil {
			break
		}
		group.Go(func() error {
			if workerCtx.Err() != nil {
				return workerCtx.Err()
			}
			outputMu.Lock()
			fmt.Fprintf(options.Progress, "处理 %s:%s（全部平台）\n", task.name, task.tag)
			outputMu.Unlock()
			record := migrationRecord{Repository: task.name, Tag: task.tag, Status: "planned"}
			err := migrationRetry(workerCtx, options.Attempts, options.ItemTimeout, func(ctx context.Context) error {
				desc, err := task.source.Resolve(ctx, task.tag)
				if err == nil {
					record.Digest = desc.Digest.String()
				}
				return err
			})
			if err == nil && !options.DryRun {
				err = migrationRetry(workerCtx, options.Attempts, options.ItemTimeout, func(ctx context.Context) error {
					return copyMigrationTag(ctx, task.source, task.target, record.Digest, task.tag, options.Concurrency)
				})
				if err == nil {
					record.Status = "success"
				}
			}
			outputMu.Lock()
			defer outputMu.Unlock()
			if err != nil {
				failed++
				record.Status = "failed"
				record.Error = err.Error()
			} else {
				succeeded++
			}
			if err := encoder.Encode(record); err != nil {
				return fmt.Errorf("写入报告失败: %w", err)
			}
			fmt.Fprintf(options.Progress, "%s %s:%s %s\n", record.Status, task.name, task.tag, record.Digest)
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	fmt.Fprintf(options.Progress, "完成：成功/计划 %d，失败 %d\n", succeeded, failed)
	if failed > 0 {
		return fmt.Errorf("迁移存在 %d 个失败项，请查看报告并重新运行", failed)
	}
	return nil
}
