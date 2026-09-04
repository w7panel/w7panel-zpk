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
	OrderSn     string `json:"order_sn"`
}

func ResolveManifestDependencyReleaseNames(manifest *commonlogic.Manifest, consoleUID int32, orderSn string) error {
	bindings := make(map[string]DependencyOrderBinding)
	if manifest == nil {
		return nil
	}
	hasExternalDependency := false
	for _, dependency := range manifest.Platform.Depends {
		if isExternalDependency(dependency) {
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
			bindings[identify] = DependencyOrderBinding{
				AppIdentify: binding.AppIdentify,
				OrderSn:     binding.OrderSn,
			}
		}
	}
	return ResolveDependencyReleaseNames(manifest, bindings)
}

func ResolveDependencyReleaseNames(manifest *commonlogic.Manifest, bindings map[string]DependencyOrderBinding) error {
	if manifest == nil {
		return nil
	}
	bindingsByIdentifie := make(map[string]DependencyOrderBinding, len(bindings))
	for identify, binding := range bindings {
		if identify != "" {
			bindingsByIdentifie[identify] = binding
		}
	}
	manifest.Platform.Depends = append([]commonlogic.Depend(nil), manifest.Platform.Depends...)
	for index := range manifest.Platform.Depends {
		dependency := &manifest.Platform.Depends[index]
		if !isExternalDependency(*dependency) {
			continue
		}

		identify := dependency.Identifie
		binding, hasBinding := bindingsByIdentifie[identify]
		if hasBinding {
			dependency.OrderSn = strings.TrimSpace(binding.OrderSn)
		}
		if hasBinding && binding.AppIdentify != "" {
			dependency.ReleaseName = binding.AppIdentify
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
		if dependency.ReleaseName != "" {
			dependency.ReleaseName = dependency.ReleaseName
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
	PopulateManifestStartParamsWithDependencyReleaseNames(manifest)
	return nil
}

func GenerateDependencyReleaseName(identify string) (string, error) {
	if identify == "" {
		return "", fmt.Errorf("依赖应用标识不能为空")
	}
	suffix, err := secureLowercaseString(dependencyReleaseSuffixLen)
	if err != nil {
		return "", fmt.Errorf("生成依赖 releaseName 失败: %w", err)
	}
	maxBaseLen := 63 - len(suffix) - 1
	if len(identify) > maxBaseLen {
		identify = strings.TrimRight(identify[:maxBaseLen], "-")
	}
	if identify == "" {
		return "", fmt.Errorf("依赖应用标识无法生成 releaseName")
	}
	return identify + "-" + suffix, nil
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
