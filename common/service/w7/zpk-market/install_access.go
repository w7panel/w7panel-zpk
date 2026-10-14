package zpk_market

import (
	"errors"

	"github.com/w7panel/w7panel-zpk/common/service/w7/base"
)

const (
	AccessReasonPurchaseRequired = "purchase_required"
	AccessReasonVIPRequired      = "vip_required"
	AccessReasonVIPLevel         = "vip_level_insufficient"
	AccessReasonVIPExpired       = "vip_expired"
	InstallConflictRefundPending = "refund_pending"
)

// InstallAccessError is an explicit market denial, rather than a transport or signing failure.
type InstallAccessError struct {
	Reason  string
	Message string
}

func (e *InstallAccessError) Error() string { return e.Message }

// Only errors decoded from the market response receive this wrapper.
type marketResponseError struct{ base.ApiError }

func (e *marketResponseError) Unwrap() error { return e.ApiError }

// Shared with the market installation/upgrade denial protocol.
const (
	InstallAccessPurchaseRequired = 40301
	InstallAccessVIPRequired      = 40302
	InstallAccessVIPLevel         = 40303
	InstallAccessVIPExpired       = 40304
)

func classifyInstallAccessError(err error) error {
	var responseErr *marketResponseError
	if !errors.As(err, &responseErr) {
		return err
	}
	reason := ""
	message := responseErr.Error()
	switch responseErr.Code {
	case InstallAccessPurchaseRequired:
		reason, message = AccessReasonPurchaseRequired, "未找到有效的已购订单，请先购买后再安装"
	case InstallAccessVIPRequired:
		reason = AccessReasonVIPRequired
	case InstallAccessVIPLevel:
		reason = AccessReasonVIPLevel
	case InstallAccessVIPExpired:
		reason = AccessReasonVIPExpired
	default:
		return err
	}
	return &InstallAccessError{Reason: reason, Message: message}
}
