package image

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"

	clicommon "github.com/w7panel/w7panel-zpk/cli/app/application/logic/common"
)

func imagePushReference(source, host, username string) (string, error) {
	if strings.Contains(source, "@") {
		return "", fmt.Errorf("本地镜像请使用 tag 引用")
	}
	if username == "" {
		return "", fmt.Errorf("请先执行 zpk login")
	}
	name := username + "/" + path.Base(source)
	reference, err := clicommon.ResolveTarget(name, host, "", false)
	if err != nil {
		return "", err
	}
	if reference.Version == "" {
		reference.Version = "latest"
	}
	name = reference.Reference() + ":" + reference.Version

	return name, nil
}

// PushImage delegates standard image layer deduplication to Docker. A private
// temporary config keeps the developer's existing Docker credentials intact.
func PushImage(ctx context.Context, source string, session clicommon.Session, stdout, stderr io.Writer) error {
	credential, err := session.RegistryCredential(session.Host)
	if err != nil {
		return err
	}
	if credential.Username == "" {
		return fmt.Errorf("请先执行 zpk login")
	}
	target, err := imagePushReference(source, session.Host, credential.Username)
	if err != nil {
		return err
	}
	if _, err = exec.LookPath("docker"); err != nil {
		return fmt.Errorf("未找到 docker，请先安装 Docker: %w", err)
	}
	directory, err := os.MkdirTemp("", "zpk-docker-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	run := func(stdin io.Reader, args ...string) error {
		args = append([]string{"--config", directory}, args...)
		command := exec.CommandContext(ctx, "docker", args...)
		command.Stdin, command.Stdout, command.Stderr = stdin, stdout, stderr
		return command.Run()
	}
	if err = run(strings.NewReader(credential.Password+"\n"), "login", session.Host, "--username", credential.Username, "--password-stdin"); err != nil {
		return fmt.Errorf("Docker 仓库认证失败: %w", err)
	}
	if err = run(nil, "tag", source, target); err != nil {
		return fmt.Errorf("Docker 镜像标记失败: %w", err)
	}
	if err = run(nil, "push", target); err != nil {
		return fmt.Errorf("Docker 镜像推送失败: %w", err)
	}
	if stdout != nil {
		fmt.Fprintf(stdout, "镜像已推送 %s\n", target)
	}
	return nil
}
