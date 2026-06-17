package accessor

type CrossUpgradeFormula struct {
	Identifie      string  `json:"identifie"`
	Title          string  `json:"title"`
	GoodsID        int32   `json:"goods_id"`
	GoodsProductID int32   `json:"goods_product_id"`
	Price          float64 `json:"price"`
	Icon           string  `json:"icon"`
}

type CrossUpgradeFormulasOption struct {
	List []CrossUpgradeFormula `json:"list"`
}
