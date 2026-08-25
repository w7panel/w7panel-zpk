package logic

import (
	"crypto/rand"
	"fmt"
	"strings"

	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
)

const dependencyReleaseSuffixLen = 12

type DependencyOrderBinding struct {
	AppIdentify string `json:"app_identify"`
}

func ResolveManifestDependencyReleaseNames(manifest *commonlogic.Manifest, consoleUID int32, orderSn string) error {
	bindings := make(map[string]DependencyOrderBinding)
	if manifest == nil {
		return nil
	}
	hasExternalDependency := false
	for _, dependency := range manifest.Platform.Depends {
		if dependency.Type == "out" {
			hasExternalDependency = true
			break
		}
	}
	if strings.TrimSpace(orderSn) != "" && hasExternalDependency {
		marketBindings, err := w7.ZpkMarketSdk.GetDependencyOrders(consoleUID, orderSn)
		if err != nil {
			return fmt.Errorf("查询依赖订单失败: %w", err)
		}
		for identify, binding := range marketBindings {
			bindings[identify] = DependencyOrderBinding{AppIdentify: binding.AppIdentify}
		}
	}
	return ResolveDependencyReleaseNames(manifest, bindings)
}

func ResolveDependencyReleaseNames(manifest *commonlogic.Manifest, bindings map[string]DependencyOrderBinding) error {
	if manifest == nil {
		return nil
	}
	normalizedBindings := make(map[string]DependencyOrderBinding, len(bindings))
	for identify, binding := range bindings {
		normalizedBindings[normalizeDependencyName(identify)] = binding
	}
	manifest.Platform.Depends = append([]commonlogic.Depend(nil), manifest.Platform.Depends...)
	for index := range manifest.Platform.Depends {
		dependency := &manifest.Platform.Depends[index]
		if dependency.Type != "out" {
			continue
		}

		identify := normalizeDependencyName(dependency.Identifie)
		binding, hasBinding := normalizedBindings[identify]
		if hasBinding && strings.TrimSpace(binding.AppIdentify) != "" {
			dependency.ReleaseName = normalizeDependencyName(binding.AppIdentify)
			if dependency.ReleaseName == "" {
				return fmt.Errorf("依赖 %s 的 app_identify 无法作为 releaseName", dependency.Identifie)
			}
			dependency.ReleaseNameFixed = true
			continue
		}
		if hasBinding && dependency.MultipleInstances {
			var err error
			dependency.ReleaseName, err = GenerateDependencyReleaseName(identify)
			if err != nil {
				return err
			}
			dependency.ReleaseNameFixed = false
			continue
		}
		if strings.TrimSpace(dependency.ReleaseName) != "" {
			dependency.ReleaseName = normalizeDependencyName(dependency.ReleaseName)
			if dependency.ReleaseName == "" {
				return fmt.Errorf("依赖 %s 的 releaseName 无效", dependency.Identifie)
			}
			dependency.ReleaseNameFixed = true
			continue
		}
		if dependency.MultipleInstances {
			var err error
			dependency.ReleaseName, err = GenerateDependencyReleaseName(identify)
			if err != nil {
				return err
			}
			dependency.ReleaseNameFixed = false
			continue
		}

		dependency.ReleaseName = identify
		dependency.ReleaseNameFixed = true
	}
	ApplyDependencyReleaseStartParams(manifest)
	return nil
}

func GenerateDependencyReleaseName(identify string) (string, error) {
	base := normalizeDependencyName(identify)
	if base == "" {
		return "", fmt.Errorf("依赖应用标识不能为空")
	}
	suffix, err := secureLowercaseString(dependencyReleaseSuffixLen)
	if err != nil {
		return "", fmt.Errorf("生成依赖 releaseName 失败: %w", err)
	}
	maxBaseLen := 63 - len(suffix) - 1
	if len(base) > maxBaseLen {
		base = strings.TrimRight(base[:maxBaseLen], "-")
	}
	if base == "" {
		return "", fmt.Errorf("依赖应用标识无法生成 releaseName")
	}
	return base + "-" + suffix, nil
}

func normalizeDependencyName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		valid := char >= 'a' && char <= 'z' || char >= '0' && char <= '9'
		if valid {
			builder.WriteRune(char)
			lastDash = false
			continue
		}
		if !lastDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func secureLowercaseString(length int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	result := make([]byte, length)
	for index, value := range randomBytes {
		result[index] = alphabet[int(value)%len(alphabet)]
	}
	return string(result), nil
}
