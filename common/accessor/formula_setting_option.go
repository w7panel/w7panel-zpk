package accessor

type FormulaSettingOption struct {
	SupportCrossUpgrade           bool `json:"support_cross_upgrade"`
	SupportAutoPublishToZpkMarket bool `json:"support_auto_publish_to_zpk_market"`
	EnableServicePackageFee       bool `json:"enable_service_package_fee"`
}
