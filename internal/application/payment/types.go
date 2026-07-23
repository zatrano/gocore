package payment

import (
	"github.com/zatrano/gocore/internal/infrastructure/payment/iyzico"
	"github.com/zatrano/gocore/internal/infrastructure/payment/moka"
	"github.com/zatrano/gocore/pkg/datetime"
)

// BinCheckResult, sağlayıcıdan bağımsız BIN sorgu yanıt özetidir.
type BinCheckResult struct {
	Provider  string                        `json:"provider"`
	BinNumber string                        `json:"binNumber"`
	Price     string                        `json:"price,omitempty"`
	Iyzico    *iyzico.BinCheckResponse      `json:"iyzico,omitempty"`
	Moka      *moka.BankCardInformationData `json:"moka,omitempty"`
}

// PaymentView, ödeme listesi ve detay DTO'sudur.
type PaymentView struct {
	ID                string             `json:"id"`
	Reference         string             `json:"reference"`
	Provider          string             `json:"provider"`
	Status            string             `json:"status"`
	StatusLabel       string             `json:"status_label"`
	Stage             string             `json:"stage,omitempty"`
	Amount            string             `json:"amount"`
	PaidAmount        string             `json:"paid_amount,omitempty"`
	Currency          string             `json:"currency"`
	Installment       int                `json:"installment"`
	BuyerName         string             `json:"buyer_name,omitempty"`
	BuyerSurname      string             `json:"buyer_surname,omitempty"`
	BuyerEmail        string             `json:"buyer_email,omitempty"`
	BuyerPhone        string             `json:"buyer_phone,omitempty"`
	CardHolder        string             `json:"card_holder,omitempty"`
	CardDisplay       string             `json:"card_display,omitempty"`
	CardAssociation   string             `json:"card_association,omitempty"`
	ProviderPaymentID string             `json:"provider_payment_id,omitempty"`
	ResultCode        string             `json:"result_code,omitempty"`
	ResultMessage     string             `json:"result_message,omitempty"`
	AuthCode          string             `json:"auth_code,omitempty"`
	CreatedAt         datetime.JSONTime  `json:"created_at"`
	UpdatedAt         datetime.JSONTime  `json:"updated_at"`
	CompletedAt       *datetime.JSONTime `json:"completed_at,omitempty"`
}

// CalcPaymentAmountResult, Moka taksit tutarı hesaplama yanıtıdır.
type CalcPaymentAmountResult struct {
	Provider string                     `json:"provider"`
	Moka     moka.CalcPaymentAmountData `json:"moka"`
}

// WebhookHandleResult, iyzico webhook işleme sonucudur.
type WebhookHandleResult struct {
	Verified       bool   `json:"verified"`
	Acknowledged   bool   `json:"acknowledged"`
	Reference      string `json:"reference,omitempty"`
	PaymentUpdated bool   `json:"payment_updated"`
	Status         string `json:"status,omitempty"`
	IyziReference  string `json:"iyzi_reference_code,omitempty"`
}

// Complete3DSResult, tamamlama yanıtıdır (sağlayıcıya göre alanlar dolabilir).
type Complete3DSResult struct {
	Provider      string                  `json:"provider"`
	Reference     string                  `json:"reference"`
	Status        string                  `json:"status"`
	PaymentID     string                  `json:"payment_id,omitempty"`
	AuthCode      string                  `json:"auth_code,omitempty"`
	ResultCode    string                  `json:"result_code,omitempty"`
	ResultMessage string                  `json:"result_message,omitempty"`
	Iyzico        *iyzico.Auth3DSResponse `json:"iyzico,omitempty"`
}
