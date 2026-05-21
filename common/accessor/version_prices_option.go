package accessor

import "github.com/w7panel/w7panel-zpk/common/service/w7/devcenter"

type VersionPricesOption struct {
	List []devcenter.NotAppBranchVersionPriceInfo `json:"list"`
}
