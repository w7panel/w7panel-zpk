package zpkmarket

import (
	"context"
	"errors"
	"strings"
	"time"

	formulalogic "github.com/w7panel/w7panel-zpk/app/respo/logic/formula"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	zpk_market "github.com/w7panel/w7panel-zpk/common/service/w7/zpk-market"
)

const (
	ClusterLevelMain = "main"
	ClusterLevelSub  = "sub"

	ClusterSupportAll  = "all"
	ClusterSupportNone = "none"

	formulaCategoryRequestTimeout = 10 * time.Second
)

func GetFormulaClusterSupport(ctx context.Context, formula formulalogic.Formula) (string, error) {
	return getFormulaClusterSupport(ctx, formula, w7.ZpkMarketSdk.GetFormulaCategoriesWithContext)
}

func getFormulaClusterSupport(ctx context.Context, formula formulalogic.Formula, loadCategories func(context.Context, int) ([]zpk_market.FormulaCategory, error)) (string, error) {
	if len(formula.Tags) == 0 {
		return ClusterSupportAll, nil
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, formulaCategoryRequestTimeout)
	defer cancel()
	categories, err := loadCategories(timeoutCtx, 999)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return "", nil
		}
		return "", err
	}
	return FormulaClusterSupport(formula, categories), nil
}

func FormulaClusterSupport(formula formulalogic.Formula, categories []zpk_market.FormulaCategory) string {
	settings := make(map[string]string, len(categories))
	for _, category := range categories {
		name := strings.TrimSpace(category.Name)
		if name == "" {
			continue
		}
		settings[name] = strings.TrimSpace(category.Setting.SupportCluster)
	}

	supportMain := true
	supportSub := true
	for _, tag := range formula.Tags {
		supportCluster, exists := settings[strings.TrimSpace(tag.Name)]
		if !exists {
			continue
		}
		if supportCluster == "" {
			supportCluster = ClusterSupportAll
		}
		switch supportCluster {
		case ClusterLevelMain:
			supportSub = false
		case ClusterLevelSub:
			supportMain = false
		case ClusterSupportNone:
			supportMain = false
			supportSub = false
		}
	}

	switch {
	case supportMain && supportSub:
		return ClusterSupportAll
	case supportMain:
		return ClusterLevelMain
	case supportSub:
		return ClusterLevelSub
	default:
		return ClusterSupportNone
	}
}
