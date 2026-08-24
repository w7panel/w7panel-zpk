package logic

import (
	"encoding/base64"
	"encoding/json"

	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
)

type TicketInfo struct {
	FormulaId       int32  `json:"formula_id"`
	ConsoleUid      int32  `json:"console_uid"`
	FormulaVersion  string `json:"formula_version"`
	FormulaType     string `json:"formula_type"`
	FormulaIsPlugin bool   `json:"formula_is_plugin"`
	OrderSn         string `json:"order_sn"`
	IsUpgrade       bool   `json:"is_upgrade"`
	Reinstall       bool   `json:"reinstall"`
	Domain          string `json:"domain"`
	AppIdentify     string `json:"app_identify"`
}

type Ticket struct {
}

func IsFormulaPlugin(formulaType, traditionInstallType string) bool {
	return formulaType == "gateway-plugin" ||
		(formulaType == "tradition" && traditionInstallType == "extension")
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
	return encodeURLSafeTicket(ticket)
}

func (l Ticket) ParseTicket(ticket string) (*TicketInfo, error) {
	key := function.GetMd5(facade.GetConfig().GetString("setting.secret"))
	aesTicket, err := decodeURLSafeTicket(ticket)
	if err != nil {
		aesTicket = ticket
	}
	info, err := function.AesDecrypt(aesTicket, key)
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

func encodeURLSafeTicket(ticket string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ticket)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func decodeURLSafeTicket(ticket string) (string, error) {
	ciphertext, err := base64.RawURLEncoding.DecodeString(ticket)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
