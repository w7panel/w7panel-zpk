package logic

import (
	"github.com/w7panel/w7panel-zpk/common/accessor"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
)

const yamlCopyAnnotationKey = "w7.cc/yaml_copy"

func filterFormulaAnnotations(annotation map[string]interface{}) map[string]interface{} {
	if annotation == nil {
		return nil
	}
	filtered := make(map[string]interface{}, len(annotation))
	for key, value := range annotation {
		if key != yamlCopyAnnotationKey {
			filtered[key] = value
		}
	}
	return filtered
}

// GetBaseInfo returns formula-level base information. Existing formulas that
// have not saved formula-level data yet fall back to their current manifest.
func (f *Formula) GetBaseInfo() *accessor.FormulaBaseInfoOption {
	if f.Setting != nil && f.Setting.BaseInfo != nil {
		baseInfo := *f.Setting.BaseInfo
		baseInfo.Annotation = filterFormulaAnnotations(baseInfo.Annotation)
		return &baseInfo
	}

	baseInfo := &accessor.FormulaBaseInfoOption{}
	if f.Manifest == nil {
		baseInfo.Name = f.Title
		return baseInfo
	}

	baseInfo.Name = f.Manifest.Application.Name
	if baseInfo.Name == "" {
		baseInfo.Name = f.Manifest.Platform.BaseInfo.Name
	}
	if baseInfo.Name == "" {
		baseInfo.Name = f.Title
	}
	baseInfo.Description = f.Manifest.Application.Description
	if baseInfo.Description == "" {
		baseInfo.Description = f.Manifest.Platform.BaseInfo.Description
	}
	baseInfo.Annotation = filterFormulaAnnotations(f.Manifest.Application.Annotation)
	baseInfo.InstallOnlyOnce = f.Manifest.Application.InstallOnlyOnce
	baseInfo.ClusterPrivileged = f.Manifest.Application.ClusterPrivileged
	baseInfo.RegisterSite = f.Manifest.Application.RegisterSite
	return baseInfo
}

// ApplyBaseInfo overlays formula-level metadata onto a version manifest. This
// keeps exported manifests compatible while preventing per-version drift.
func (f *Formula) ApplyBaseInfo(manifest *logic2.Manifest) {
	if manifest == nil || f.Setting == nil || f.Setting.BaseInfo == nil {
		return
	}

	baseInfo := f.Setting.BaseInfo
	manifest.Application.Name = baseInfo.Name
	manifest.Application.Description = baseInfo.Description
	manifest.Application.Annotation = filterFormulaAnnotations(baseInfo.Annotation)
	manifest.Application.InstallOnlyOnce = baseInfo.InstallOnlyOnce
	manifest.Application.ClusterPrivileged = baseInfo.ClusterPrivileged
	manifest.Application.RegisterSite = baseInfo.RegisterSite

	if manifest.Platform.BaseInfo.Name == "" {
		manifest.Platform.BaseInfo.Name = baseInfo.Name
	}
	if manifest.Platform.BaseInfo.Description == "" {
		manifest.Platform.BaseInfo.Description = baseInfo.Description
	}
}
