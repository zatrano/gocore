package moka

import (
	"context"
	"fmt"
)

const pathDoCalcPaymentAmount = "/PaymentDealer/DoCalcPaymentAmount"

// CalcPaymentAmountRequest, taksit/komisyon dahil tahsil tutarı hesaplama isteğidir.
// https://developer.mokaunited.com/home.php?page=karttan-cekilecek-tutar
type CalcPaymentAmountRequest struct {
	BinNumber          string  `json:"BinNumber"`
	Currency           string  `json:"Currency,omitempty"`
	OrderAmount        float64 `json:"OrderAmount"`
	InstallmentNumber  int     `json:"InstallmentNumber"`
	GroupRevenueRate   float64 `json:"GroupRevenueRate,omitempty"`
	GroupRevenueAmount float64 `json:"GroupRevenueAmount,omitempty"`
	IsThreeD           int     `json:"IsThreeD"`
}

// CalcPaymentAmountData, DoCalcPaymentAmount yanıtıdır.
type CalcPaymentAmountData struct {
	PaymentAmount                    float64                  `json:"PaymentAmount"`
	DealerDepositAmount              float64                  `json:"DealerDepositAmount"`
	DealerCommissionRate             float64                  `json:"DealerCommissionRate"`
	DealerCommissionAmount           float64                  `json:"DealerCommissionAmount"`
	DealerCommissionFixedAmount      float64                  `json:"DealerCommissionFixedAmount"`
	DealerGroupCommissionRate        float64                  `json:"DealerGroupCommissionRate"`
	DealerGroupCommissionAmount      float64                  `json:"DealerGroupCommissionAmount"`
	DealerGroupCommissionFixedAmount float64                  `json:"DealerGroupCommissionFixedAmount"`
	GroupRevenueRate                 float64                  `json:"GroupRevenueRate"`
	GroupRevenueAmount               float64                  `json:"GroupRevenueAmount"`
	BankCard                         *BankCardInformationData `json:"BankCard,omitempty"`
}

// DoCalcPaymentAmount, sepet tutarı ve taksit için karttan çekilecek tutarı hesaplar.
func (c *Client) DoCalcPaymentAmount(ctx context.Context, req CalcPaymentAmountRequest) (CalcPaymentAmountData, error) {
	bin := NormalizeBinNumber(req.BinNumber)
	if len(bin) < 6 {
		return CalcPaymentAmountData{}, fmt.Errorf("moka: BIN en az 6 hane olmalıdır")
	}
	if req.OrderAmount <= 0 {
		return CalcPaymentAmountData{}, fmt.Errorf("moka: sepet tutarı pozitif olmalıdır")
	}
	if req.InstallmentNumber < 0 {
		return CalcPaymentAmountData{}, fmt.Errorf("moka: geçersiz taksit sayısı")
	}
	req.BinNumber = bin
	if req.Currency == "" {
		req.Currency = "TL"
	}
	var data CalcPaymentAmountData
	if err := c.post(ctx, pathDoCalcPaymentAmount, req, &data); err != nil {
		return CalcPaymentAmountData{}, err
	}
	return data, nil
}
