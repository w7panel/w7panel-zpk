package accessor

type FormulaSettingOption struct {
	SupportCrossUpgrade           bool                   `json:"support_cross_upgrade"`
	SupportAutoPublishToZpkMarket bool                   `json:"support_auto_publish_to_zpk_market"`
	EnableServicePackageFee       bool                   `json:"enable_service_package_fee"`
	BaseInfo                      *FormulaBaseInfoOption `json:"base_info,omitempty"`
}

// FormulaBaseInfoOption contains metadata shared by every version of a formula.
// Version manifests still contain these fields for package compatibility, but
// this option is their source of truth once it has been initialized.
type FormulaBaseInfoOption struct {
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Annotation        map[string]interface{} `json:"annotation"`
	InstallOnlyOnce   bool                   `json:"once"`
	ClusterPrivileged bool                   `json:"cluster_privileges"`
	RegisterSite      bool                   `json:"register_site"`
}
