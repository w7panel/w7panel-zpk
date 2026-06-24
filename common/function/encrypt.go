package function

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

func pad(buf []byte, blockSize int) []byte {
	padding := blockSize - (len(buf) % blockSize)
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(buf, padText...)
}

func unpad(src []byte) []byte {
	padding := src[len(src)-1]
	return src[:len(src)-int(padding)]
}

func AesEncrypt(plaintext string, key string) (string, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	plainBytes := []byte(plaintext)
	// 对于aes加密，密文长度必须等长于明文长度，所以需要对明文进行填充
	plainBytes = pad(plainBytes, aes.BlockSize)
	ciphertext := make([]byte, aes.BlockSize+len(plainBytes))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plainBytes)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func AesDecrypt(ciphertextBase64 string, key string) (string, error) {
	// 解码base64编码的密文
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	// 密文长度应该至少是块大小加上实际密文长度
	if len(ciphertext) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}

	// 获取初始向量
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	// 创建一个CFB解密器对象
	stream := cipher.NewCFBDecrypter(block, iv)

	// 解密
	stream.XORKeyStream(ciphertext, ciphertext)

	// 对解密后的数据去填充
	unpaddedBytes := unpad(ciphertext)

	return string(unpaddedBytes), nil
}
