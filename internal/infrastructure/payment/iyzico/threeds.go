package iyzico

import "context"

// PaymentCard, kart bilgileridir.
type PaymentCard struct {
	CardHolderName string `json:"cardHolderName"`
	CardNumber     string `json:"cardNumber"`
	ExpireYear     string `json:"expireYear"`
	ExpireMonth    string `json:"expireMonth"`
	CVC            string `json:"cvc"`
}

// Buyer, alıcı bilgileridir.
type Buyer struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Surname             string `json:"surname"`
	IdentityNumber      string `json:"identityNumber"`
	Email               string `json:"email"`
	GsmNumber           string `json:"gsmNumber"`
	RegistrationDate    string `json:"registrationDate"`
	LastLoginDate       string `json:"lastLoginDate"`
	RegistrationAddress string `json:"registrationAddress"`
	City                string `json:"city"`
	Country             string `json:"country"`
	ZipCode             string `json:"zipCode"`
	IP                  string `json:"ip"`
}

// Address, fatura/teslimat adresidir.
type Address struct {
	Address     string `json:"address"`
	ZipCode     string `json:"zipCode"`
	ContactName string `json:"contactName"`
	City        string `json:"city"`
	Country     string `json:"country"`
}

// BasketItem, sepet kalemidir.
type BasketItem struct {
	ID        string `json:"id"`
	Price     string `json:"price"`
	Name      string `json:"name"`
	Category1 string `json:"category1"`
	Category2 string `json:"category2"`
	ItemType  string `json:"itemType"`
}

// BinCheckRequest, BIN sorgu isteğidir.
type BinCheckRequest struct {
	Locale    string `json:"locale,omitempty"`
	Price     string `json:"price"`
	BinNumber string `json:"binNumber"`
}

// InstallmentPrice, taksit fiyat satırıdır.
type InstallmentPrice struct {
	InstallmentPrice  float64 `json:"installmentPrice"`
	TotalPrice        float64 `json:"totalPrice"`
	InstallmentNumber int     `json:"installmentNumber"`
}

// InstallmentDetail, BIN sorgu yanıt satırıdır.
type InstallmentDetail struct {
	BinNumber         string             `json:"binNumber"`
	Price             float64            `json:"price"`
	CardType          string             `json:"cardType"`
	CardAssociation   string             `json:"cardAssociation"`
	CardFamilyName    string             `json:"cardFamilyName"`
	Force3DS          int                `json:"force3ds"`
	BankCode          int                `json:"bankCode"`
	BankName          string             `json:"bankName"`
	InstallmentPrices []InstallmentPrice `json:"installmentPrices"`
}

// BinCheckResponse, BIN sorgu yanıtıdır.
type BinCheckResponse struct {
	APIResponse
	InstallmentDetails []InstallmentDetail `json:"installmentDetails"`
}

// BinCheck, kart BIN bilgisini ve taksit seçeneklerini sorgular.
func (c *Client) BinCheck(ctx context.Context, req BinCheckRequest) (BinCheckResponse, error) {
	var resp BinCheckResponse
	if err := c.post(ctx, pathBinCheck, req, &resp); err != nil {
		return BinCheckResponse{}, err
	}
	if err := resp.apiError(); err != nil {
		return resp, err
	}
	return resp, nil
}

// Init3DSRequest, 3DS başlatma isteğidir.
type Init3DSRequest struct {
	Locale          string       `json:"locale,omitempty"`
	ConversationID  string       `json:"conversationId"`
	Price           string       `json:"price"`
	PaidPrice       string       `json:"paidPrice"`
	Currency        string       `json:"currency"`
	Installment     int          `json:"installment"`
	PaymentChannel  string       `json:"paymentChannel"`
	BasketID        string       `json:"basketId"`
	PaymentGroup    string       `json:"paymentGroup"`
	PaymentCard     PaymentCard  `json:"paymentCard"`
	Buyer           Buyer        `json:"buyer"`
	ShippingAddress Address      `json:"shippingAddress"`
	BillingAddress  Address      `json:"billingAddress"`
	BasketItems     []BasketItem `json:"basketItems"`
	CallbackURL     string       `json:"callbackUrl"`
}

// Init3DSResponse, 3DS başlatma yanıtıdır.
type Init3DSResponse struct {
	APIResponse
	ConversationID     string `json:"conversationId"`
	ThreeDSHtmlContent string `json:"threeDSHtmlContent"`
}

// Initialize3DS, 3DS ödeme akışını başlatır.
func (c *Client) Initialize3DS(ctx context.Context, req Init3DSRequest) (Init3DSResponse, error) {
	var resp Init3DSResponse
	if err := c.post(ctx, pathInit3DS, req, &resp); err != nil {
		return Init3DSResponse{}, err
	}
	if err := resp.apiError(); err != nil {
		return resp, err
	}
	return resp, nil
}

// Auth3DSRequest, 3DS tamamlama isteğidir.
type Auth3DSRequest struct {
	Locale           string `json:"locale,omitempty"`
	PaymentID        string `json:"paymentId"`
	ConversationData string `json:"conversationData,omitempty"`
}

// ItemTransaction, sepet kalemi işlem sonucudur.
type ItemTransaction struct {
	ItemID               string  `json:"itemId"`
	PaymentTransactionID string  `json:"paymentTransactionId"`
	TransactionStatus    int     `json:"transactionStatus"`
	Price                float64 `json:"price"`
	PaidPrice            float64 `json:"paidPrice"`
}

// Auth3DSResponse, 3DS tamamlama yanıtıdır.
type Auth3DSResponse struct {
	APIResponse
	Price            float64           `json:"price"`
	PaidPrice        float64           `json:"paidPrice"`
	Installment      int               `json:"installment"`
	PaymentID        string            `json:"paymentId"`
	FraudStatus      int               `json:"fraudStatus"`
	CardType         string            `json:"cardType"`
	CardAssociation  string            `json:"cardAssociation"`
	CardFamilyName   string            `json:"cardFamilyName"`
	BinNumber        string            `json:"binNumber"`
	LastFourDigits   string            `json:"lastFourDigits"`
	BasketID         string            `json:"basketId"`
	Currency         string            `json:"currency"`
	ItemTransactions []ItemTransaction `json:"itemTransactions"`
	AuthCode         string            `json:"authCode"`
	Phase            string            `json:"phase"`
	MDStatus         int               `json:"mdStatus"`
	HostReference    string            `json:"hostReference"`
}

// Auth3DS, 3DS doğrulaması sonrası ödemeyi tamamlar.
func (c *Client) Auth3DS(ctx context.Context, req Auth3DSRequest) (Auth3DSResponse, error) {
	var resp Auth3DSResponse
	if err := c.post(ctx, pathAuth3DS, req, &resp); err != nil {
		return Auth3DSResponse{}, err
	}
	if err := resp.apiError(); err != nil {
		return resp, err
	}
	return resp, nil
}

// PaymentDetailRequest, ödeme sorgu isteğidir.
type PaymentDetailRequest struct {
	Locale                string `json:"locale,omitempty"`
	ConversationID        string `json:"conversationId,omitempty"`
	PaymentID             string `json:"paymentId,omitempty"`
	PaymentConversationID string `json:"paymentConversationId,omitempty"`
}

// PaymentDetailResponse, ödeme sorgu yanıtıdır.
type PaymentDetailResponse struct {
	APIResponse
	PaymentStatus   string  `json:"paymentStatus"`
	PaymentID       string  `json:"paymentId"`
	PaidPrice       float64 `json:"paidPrice"`
	Installment     int     `json:"installment"`
	BinNumber       string  `json:"binNumber"`
	LastFourDigits  string  `json:"lastFourDigits"`
	CardAssociation string  `json:"cardAssociation"`
	AuthCode        string  `json:"authCode"`
}

// PaymentDetail, sağlayıcıdaki ödeme durumunu sorgular (reconciliation).
func (c *Client) PaymentDetail(ctx context.Context, req PaymentDetailRequest) (PaymentDetailResponse, error) {
	var resp PaymentDetailResponse
	if err := c.post(ctx, pathPaymentDetail, req, &resp); err != nil {
		return PaymentDetailResponse{}, err
	}
	if err := resp.apiError(); err != nil {
		return resp, err
	}
	return resp, nil
}
