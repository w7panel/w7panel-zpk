package accessor

import "github.com/w7panel/w7panel-zpk/common/service/w7/devcenter"

type ServicePackagesOption struct {
	List []devcenter.NotAppServicePackage `json:"list"`
}
