package function

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
)

func ArrUnique[T comparable](slice []T) []T {
	encountered := map[T]bool{}
	result := []T{}

	for _, v := range slice {
		if !encountered[v] {
			encountered[v] = true
			result = append(result, v)
		}
	}
	return result
}

func GetRandomString(n int) string {
	str := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz123456789"
	bytes := []byte(str)
	var result []byte
	for i := 0; i < n; i++ {
		result = append(result, bytes[rand.Intn(len(bytes))])
	}
	return string(result)
}

func GetRandomStringNotContainerNumber(n int) string {
	str := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	var result []byte
	for i := 0; i < n; i++ {
		result = append(result, bytes[rand.Intn(len(bytes))])
	}
	return string(result)
}

func GetMd5(str string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(str)))
}

func ConvertDigitsToLetters(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			// 核心算法：'0'变成'a', '1'变成'b', ..., '9'变成'j'
			return r - '0' + 'a'
		}
		return r // 非数字字符（如原本的 a-f）保持不变
	}, s)
}

func EncodeURIComponent(s string, excluded ...[]byte) string {
	var b bytes.Buffer
	written := 0
	for i, n := 0, len(s); i < n; i++ {
		c := s[i]
		switch c {
		case '-', '_', '.', '!', '~', '*', '\'', '(', ')':
			continue
		default:
			// Unreserved according to RFC 3986 sec 2.3
			if 'a' <= c && c <= 'z' {
				continue
			}
			if 'A' <= c && c <= 'Z' {
				continue
			}
			if '0' <= c && c <= '9' {
				continue
			}
			if len(excluded) > 0 {
				conti := false
				for _, ch := range excluded[0] {
					if ch == c {
						conti = true
						break
					}
				}
				if conti {
					continue
				}
			}
		}
		b.WriteString(s[written:i])
		fmt.Fprintf(&b, "%%%02X", c)
		written = i + 1
	}
	if written == 0 {
		return s
	}
	b.WriteString(s[written:])
	return b.String()
}

// Copy-pasted from libtrust where it is private.
func JoseBase64UrlEncode(b []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "=")
}

func SortTagsDesc(tags []string) {
	sort.Slice(tags, func(i, j int) bool {
		// 提取版本号部分（去掉可能的"v"前缀）
		v1 := strings.TrimPrefix(tags[i], "v")
		v2 := strings.TrimPrefix(tags[j], "v")

		// 分割版本号组件
		parts1 := strings.Split(v1, ".")
		parts2 := strings.Split(v2, ".")

		// 比较每个组件
		for k := 0; k < len(parts1) && k < len(parts2); k++ {
			num1, err1 := strconv.Atoi(parts1[k])
			num2, err2 := strconv.Atoi(parts2[k])

			// 如果都能转换为数字，则进行数值比较
			if err1 == nil && err2 == nil {
				if num1 != num2 {
					return num1 > num2 // 降序排列
				}
			} else {
				// 不能转换时按字符串比较
				if parts1[k] != parts2[k] {
					return parts1[k] > parts2[k]
				}
			}
		}

		// 如果共同部分相同，长度更长的版本号更大
		return len(parts1) > len(parts2)
	})
}

func LooksLikeRegistryReference(name string) bool {
	first, _, ok := strings.Cut(name, "/")
	if !ok {
		return false
	}
	return first == "localhost" || strings.ContainsAny(first, ".:")
}
