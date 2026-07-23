package moka

import "context"

// BuyerInformation, alıcı bilgileridir (opsiyonel).
type BuyerInformation struct {
	BuyerFullName  string `json:"BuyerFullName,omitempty"`
	BuyerGsmNumber string `json:"BuyerGsmNumber,omitempty"`
	BuyerEmail     string `json:"BuyerEmail,omitempty"`
	BuyerAddress   string `json:"BuyerAddress,omitempty"`
}

// DirectPaymentThreeDRequest, 3DS ödeme isteğidir.
type DirectPaymentThreeDRequest struct {
	CardHolderFullName string            `json:"CardHolderFullName"`
	CardNumber         string            `json:"CardNumber"`
	ExpMonth           string            `json:"ExpMonth"`
	ExpYear            string            `json:"ExpYear"`
	CvcNumber          string            `json:"CvcNumber"`
	CardToken          string            `json:"CardToken,omitempty"`
	Amount             float64           `json:"Amount"`
	Currency           string            `json:"Currency,omitempty"`
	InstallmentNumber  int               `json:"InstallmentNumber"`
	ClientIP           string            `json:"ClientIP"`
	OtherTrxCode       string            `json:"OtherTrxCode"`
	SubMerchantName    string            `json:"SubMerchantName,omitempty"`
	IsPoolPayment      int               `json:"IsPoolPayment"`
	IsPreAuth          int               `json:"IsPreAuth"`
	IsTokenized        int               `json:"IsTokenized"`
	IntegratorId       int               `json:"IntegratorId,omitempty"`
	Software           string            `json:"Software"`
	Description        string            `json:"Description,omitempty"`
	ReturnHash         int               `json:"ReturnHash"`
	RedirectUrl        string            `json:"RedirectUrl"`
	RedirectType       int               `json:"RedirectType,omitempty"`
	BuyerInformation   *BuyerInformation `json:"BuyerInformation,omitempty"`
}

// DirectPaymentThreeDData, başarılı 3DS başlatma yanıtıdır.
type DirectPaymentThreeDData struct {
	URL         string `json:"Url"`
	CodeForHash string `json:"CodeForHash"`
}

// DoDirectPaymentThreeD, 3D Secure ödeme akışını başlatır.
func (c *Client) DoDirectPaymentThreeD(ctx context.Context, req DirectPaymentThreeDRequest) (DirectPaymentThreeDData, error) {
	if req.ReturnHash == 0 {
		req.ReturnHash = 1
	}
	if req.Software == "" {
		req.Software = c.software
	}
	if req.Currency == "" {
		req.Currency = "TL"
	}
	var data DirectPaymentThreeDData
	if err := c.post(ctx, pathDoDirectPaymentThreeD, req, &data); err != nil {
		return DirectPaymentThreeDData{}, err
	}
	return data, nil
}
