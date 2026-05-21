package logic

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	copy2 "github.com/otiai10/copy"
	"github.com/w7panel/w7panel-zpk/common/function"
	"golang.org/x/sync/singleflight"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

var singleflightHandler singleflight.Group
var repositoryChartsMap sync.Map

type HelmRepository struct {
}

func (l HelmRepository) DownloadChart(repoURL, chartName, version, outputDir string) (string, error) {
	// 1. 准备环境
	settings := cli.New()
	getters := getter.All(settings)

	// 2. 查找chart URL
	chartURL, err := repo.FindChartInRepoURL(
		repoURL,
		chartName,
		version,
		"", "", "",
		getters,
	)

	if err != nil {
		return "", fmt.Errorf("failed to find chart: %v", err)
	}
	// 3. 创建下载器
	dl := downloader.ChartDownloader{
		Out:              os.Stdout,
		Getters:          getters,
		RepositoryConfig: settings.RepositoryConfig,
		RepositoryCache:  settings.RepositoryCache,
	}

	path, _, err := dl.DownloadTo(chartURL, version, outputDir)
	return path, err
}

func (l HelmRepository) GetRepositoryEntityIndex(repoURL string) (*repo.IndexFile, error) {
	depot, _ := NewDepot()
	md5 := function.GetMd5(repoURL)
	localPath := filepath.Join(depot.GetBasePath(), "helm_charts", md5+".yaml")
	forceUpdate := false
	if function.FileExists(localPath) {
		fileInfo, err := os.Stat(localPath)
		if err != nil {
			forceUpdate = true
		} else if fileInfo.ModTime().IsZero() || time.Since(fileInfo.ModTime()).Minutes() > float64(120) {
			forceUpdate = true
		}
	}
	if !function.FileExists(localPath) || forceUpdate {
		_, err, _ := singleflightHandler.Do(localPath, func() (interface{}, error) {
			repoName, err := extractRepoNameFromURL(repoURL)
			if err != nil {
				return nil, err
			}

			settings := cli.New()
			entry := &repo.Entry{
				Name: repoName,
				URL:  repoURL,
			}
			r, err := repo.NewChartRepository(entry, getter.All(settings))
			if err != nil {
				return nil, err
			}
			indexPath, err := r.DownloadIndexFile()
			if err != nil {
				return nil, err
			}
			err = copy2.Copy(indexPath, localPath)
			if err != nil {
				return nil, err
			}
			repositoryChartsMap.Delete(localPath)

			return indexPath, nil
		})
		if err != nil {
			return nil, err
		}
	}

	indexFile, exists := repositoryChartsMap.Load(localPath)
	if !exists {
		tmpIndexFile, err := repo.LoadIndexFile(localPath)
		if err != nil {
			return nil, err
		}
		repositoryChartsMap.Store(localPath, tmpIndexFile)
		indexFile = tmpIndexFile
	}

	return indexFile.(*repo.IndexFile), nil
}

func extractRepoNameFromURL(repoURL string) (string, error) {
	// 解析URL
	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("解析URL失败: %w", err)
	}

	// 获取路径的最后一部分
	repoPath := strings.TrimSuffix(parsedURL.Path, "/")
	repoName := path.Base(repoPath)

	// 如果为空，使用主机名
	if repoName == "" || repoName == "." || repoName == "/" {
		repoName = strings.Split(parsedURL.Host, ".")[0]
	}

	return repoName, nil
}
