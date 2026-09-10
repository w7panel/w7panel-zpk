package zpkmarket

import (
	"fmt"
	"strings"

	formulalogic "github.com/w7panel/w7panel-zpk/app/respo/logic/formula"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
)

func GetFormulaInfoDependencyOrderBindings(
	consoleUID int32,
	orderSN string,
	responseManifest commonlogic.Manifest,
	allManifests []*commonlogic.Manifest,
	includeChildManifests bool,
) ([]formulalogic.DependencyOrderBinding, error) {
	if formulalogic.HasExternalDependencies(responseManifest) {
		return GetDependencyOrderBindings(consoleUID, orderSN)
	}
	if includeChildManifests {
		for _, manifest := range allManifests {
			if formulalogic.HasExternalDependencies(*manifest) {
				return GetDependencyOrderBindings(consoleUID, orderSN)
			}
		}
	}
	return []formulalogic.DependencyOrderBinding{}, nil
}

func GetDependencyOrderBindings(consoleUID int32, orderSN string) ([]formulalogic.DependencyOrderBinding, error) {
	bindings := make([]formulalogic.DependencyOrderBinding, 0)
	if strings.TrimSpace(orderSN) == "" {
		return bindings, nil
	}

	marketBindings, err := w7.ZpkMarketSdk.GetDependencyOrders(consoleUID, orderSN)
	if err != nil {
		return nil, fmt.Errorf("查询依赖订单失败: %w", err)
	}
	for identify, binding := range marketBindings {
		bindings = append(bindings, formulalogic.DependencyOrderBinding{
			Identify:       identify,
			AppReleaseName: binding.AppIdentify,
			OrderSn:        binding.OrderSn,
		})
	}

	return bindings, nil
}
