package moka

import (
	"context"
	"fmt"
	"strings"
	"unicode"
)

const pathGetBankCardInformation = "/PaymentDealer/GetBankCardInformation"

// BankCardInformationRequest, BIN sorgu isteğidir.
type BankCardInformationRequest struct {
	BinNumber string `json:"BinNumber"`
}

// BankCardInformationData, BIN sorgu yanıtıdır.
// Bazı yanıtlarda IsSuccessful false dönebilir; bu durumda ResultCode/ResultMessage dolu olur.
type BankCardInformationData struct {
	IsSuccessful    *bool  `json:"IsSuccessful,omitempty"`
	ResultCode      string `json:"ResultCode,omitempty"`
	ResultMessage   string `json:"ResultMessage,omitempty"`
	BankName        string `json:"BankName"`
	BankCode        string `json:"BankCode"`
	BinNumber       string `json:"BinNumber"`
	CardName        string `json:"CardName"`
	CardType        string `json:"CardType"`
	CreditType      string `json:"CreditType"`
	CardLogo        string `json:"CardLogo"`
	CardTemplate    string `json:"CardTemplate"`
	ProductCategory string `json:"ProductCategory"`
	GroupName       string `json:"GroupName"`
}

func (d BankCardInformationData) validate() error {
	if d.IsSuccessful != nil && !*d.IsSuccessful {
		msg := strings.TrimSpace(d.ResultMessage)
		if msg == "" {
			msg = strings.TrimSpace(d.ResultCode)
		}
		if msg == "" {
			msg = "BIN sorgusu başarısız"
		}
		return fmt.Errorf("moka: %s", msg)
	}
	return nil
}

// NormalizeBinNumber, kart numarasının ilk 8 rakamını döner (Moka spesifikasyonu).
func NormalizeBinNumber(raw string) string {
	raw = strings.TrimSpace(raw)
	var digits strings.Builder
	for _, r := range raw {
		if unicode.IsDigit(r) {
			digits.WriteRune(r)
		}
	}
	s := digits.String()
	if len(s) > 8 {
		s = s[:8]
	}
	return s
}

// GetBankCardInformation, kart BIN bilgisini sorgular.
// https://developer.mokaunited.com/home.php?page=bin-sorgu
func (c *Client) GetBankCardInformation(ctx context.Context, binNumber string) (BankCardInformationData, error) {
	bin := NormalizeBinNumber(binNumber)
	if len(bin) < 6 {
		return BankCardInformationData{}, fmt.Errorf("moka: BIN en az 6 hane olmalıdır")
	}
	var data BankCardInformationData
	err := c.postBody(ctx, pathGetBankCardInformation, map[string]any{
		"BankCardInformationRequest": BankCardInformationRequest{BinNumber: bin},
	}, &data)
	if err != nil {
		return BankCardInformationData{}, err
	}
	if err := data.validate(); err != nil {
		return data, err
	}
	return data, nil
}
