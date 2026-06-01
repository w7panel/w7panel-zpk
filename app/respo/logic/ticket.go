package logic

import (
	"encoding/json"

	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

type TicketInfo struct {
	FormulaId      int32  `json:"formula_id"`
	ConsoleUid     int32  `json:"console_uid"`
	FormulaVersion string `json:"formula_version"`
	OrderSn        string `json:"order_sn"`
	IsUpgrade      bool   `json:"is_upgrade"`
}

type Ticket struct {
}

func (l Ticket) GetTicket(ticketInfo TicketInfo) (string, error) {
	key := function.GetMd5(facade.GetConfig().GetString("setting.secret"))
	content, err := json.Marshal(ticketInfo)
	if err != nil {
		return "", err
	}
	ticket, err := function.AesEncrypt(string(content), key)
	if err != nil {
		return "", err
	}
	return ticket, nil
}

func (l Ticket) ParseTicket(ticket string) (*TicketInfo, error) {
	key := function.GetMd5(facade.GetConfig().GetString("setting.secret"))
	info, err := function.AesDecrypt(ticket, key)
	if err != nil {
		return nil, err
	}

	ticketInfo := &TicketInfo{}
	err = json.Unmarshal([]byte(info), ticketInfo)
	if err != nil {
		return nil, err
	}

	return ticketInfo, nil
}
