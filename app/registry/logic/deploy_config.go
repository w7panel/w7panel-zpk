package logic

import (
	"errors"
	"strings"

	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

const encryptedK8sConfigPrefix = "aes:"

// EncryptK8sConfig leaves the empty in-cluster configuration unchanged.
func (l Deploy) EncryptK8sConfig(content string) (string, error) {
	if content == "" {
		return "", nil
	}
	// Normalize an encrypted input before saving to avoid double encryption.
	plain, err := l.DecryptK8sConfig(content)
	if err != nil {
		return "", err
	}
	key := function.GetMd5(facade.GetConfig().GetString("setting.secret"))
	encrypted, err := function.AesEncrypt(plain, key)
	if err != nil {
		return "", errors.New("加密部署规则 KUBECONFIG 失败")
	}
	return encryptedK8sConfigPrefix + encrypted, nil
}

// DecryptK8sConfig accepts historical plaintext; marked ciphertext must decrypt successfully.
func (l Deploy) DecryptK8sConfig(content string) (string, error) {
	if !strings.HasPrefix(content, encryptedK8sConfigPrefix) {
		return content, nil
	}
	key := function.GetMd5(facade.GetConfig().GetString("setting.secret"))
	plain, err := function.AesDecrypt(strings.TrimPrefix(content, encryptedK8sConfigPrefix), key)
	if err != nil || plain == "" {
		return "", errors.New("解密部署规则 KUBECONFIG 失败，请检查 setting.secret 或重新保存配置")
	}
	return plain, nil
}
