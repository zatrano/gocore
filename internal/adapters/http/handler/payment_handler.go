package handler

import (
	"encoding/base64"
	"html"
	"strings"

	"github.com/zatrano/gocore/pkg/validation"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/internal/adapters/http/render"
	adapters "github.com/zatrano/gocore/internal/adapters/shared"
	apppayment "github.com/zatrano/gocore/internal/application/payment"
	domainpayment "github.com/zatrano/gocore/internal/domain/payment"
	domainsettings "github.com/zatrano/gocore/internal/domain/settings"
	"github.com/zatrano/gocore/internal/infrastructure/payment/iyzico"
	"github.com/zatrano/gocore/internal/infrastructure/payment/moka"
)

// PaymentHandler, 3DS ödeme uç noktalarını sunar.
type PaymentHandler struct {
	threeds  *apppayment.ThreeDSService
	validate *validator.Validate
}

// NewPaymentHandler, handler'ı kurar.
func NewPaymentHandler(threeds *apppayment.ThreeDSService, validate *validator.Validate) *PaymentHandler {
	return &PaymentHandler{threeds: threeds, validate: validate}
}

type binCheckRequest struct {
	Locale    string `json:"locale"`
	Price     string `json:"price"`
	BinNumber string `json:"binNumber" validate:"required,min=6,max=19"`
}

// BinCheck, POST /payments/bin-check — kart BIN sorgusu (aktif sağlayıcı).
func (h *PaymentHandler) BinCheck(c fiber.Ctx) error {
	var req binCheckRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	provider, err := h.threeds.ActiveProvider(c.Context())
	if err != nil {
		return render.Error(c, err)
	}
	if provider == domainsettings.ProviderIyzico.String() && strings.TrimSpace(req.Price) == "" {
		return render.Error(c, domainpayment.ErrBinPriceRequired)
	}
	resp, err := h.threeds.BinCheck(c.Context(), apppayment.BinCheckQuery{
		Locale: req.Locale, Price: req.Price, BinNumber: req.BinNumber,
	})
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, resp)
}

// GetPayment, GET /api/v1/payments/transactions/:reference — ödeme detayı.
func (h *PaymentHandler) GetPayment(c fiber.Ctx) error {
	view, err := h.threeds.GetPayment(c.Context(), c.Params("reference"))
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, view)
}

type initIyzico3DSRequest struct {
	Locale          string              `json:"locale"`
	Reference       string              `json:"reference"`
	Price           string              `json:"price" validate:"required"`
	PaidPrice       string              `json:"paidPrice" validate:"required"`
	Currency        string              `json:"currency" validate:"required"`
	Installment     int                 `json:"installment" validate:"required,min=1"`
	PaymentChannel  string              `json:"paymentChannel" validate:"required"`
	BasketID        string              `json:"basketId" validate:"required"`
	PaymentGroup    string              `json:"paymentGroup" validate:"required"`
	PaymentCard     iyzico.PaymentCard  `json:"paymentCard" validate:"required"`
	Buyer           iyzico.Buyer        `json:"buyer" validate:"required"`
	ShippingAddress iyzico.Address      `json:"shippingAddress" validate:"required"`
	BillingAddress  iyzico.Address      `json:"billingAddress" validate:"required"`
	BasketItems     []iyzico.BasketItem `json:"basketItems" validate:"required,min=1,dive"`
}

type initMoka3DSRequest struct {
	OtherTrxCode       string                 `json:"otherTrxCode"`
	CardHolderFullName string                 `json:"cardHolderFullName" validate:"required"`
	CardNumber         string                 `json:"cardNumber" validate:"required"`
	ExpMonth           string                 `json:"expMonth" validate:"required"`
	ExpYear            string                 `json:"expYear" validate:"required"`
	CvcNumber          string                 `json:"cvcNumber" validate:"required"`
	Amount             float64                `json:"amount" validate:"required,gt=0"`
	Currency           string                 `json:"currency"`
	InstallmentNumber  int                    `json:"installmentNumber" validate:"min=0,max=12"`
	Description        string                 `json:"description"`
	BuyerInformation   *moka.BuyerInformation `json:"buyerInformation"`
	IsPoolPayment      int                    `json:"isPoolPayment"`
	IsPreAuth          int                    `json:"isPreAuth"`
	IsTokenized        int                    `json:"isTokenized"`
	RedirectType       int                    `json:"redirectType"`
}

// Initialize3DS, POST /payments/3ds/initialize — aktif sağlayıcıya göre 3DS başlatır.
func (h *PaymentHandler) Initialize3DS(c fiber.Ctx) error {
	provider, err := h.threeds.ActiveProvider(c.Context())
	if err != nil {
		return render.Error(c, err)
	}
	switch provider {
	case domainsettings.ProviderIyzico.String():
		var req initIyzico3DSRequest
		if err := c.Bind().Body(&req); err != nil {
			return render.Error(c, err)
		}
		if err := validation.Check(h.validate, &req); err != nil {
			return render.Error(c, err)
		}
		result, err := h.threeds.InitializeIyzico3DS(c.Context(), apppayment.InitializeIyzicoCommand{
			Locale: req.Locale, Reference: req.Reference,
			Price: req.Price, PaidPrice: req.PaidPrice, Currency: req.Currency,
			Installment: req.Installment, PaymentChannel: req.PaymentChannel,
			BasketID: req.BasketID, PaymentGroup: req.PaymentGroup,
			PaymentCard: req.PaymentCard, Buyer: req.Buyer,
			ShippingAddress: req.ShippingAddress, BillingAddress: req.BillingAddress,
			BasketItems: req.BasketItems,
		})
		if err != nil {
			return render.Error(c, err)
		}
		return render.JSON(c, fiber.StatusOK, result)
	case domainsettings.ProviderMoka.String():
		var req initMoka3DSRequest
		if err := c.Bind().Body(&req); err != nil {
			return render.Error(c, err)
		}
		if err := validation.Check(h.validate, &req); err != nil {
			return render.Error(c, err)
		}
		// İstemci IP'si body'den alınmaz (spoof riski); güvenilir proxy sonrası c.IP().
		result, err := h.threeds.InitializeMoka3DS(c.Context(), apppayment.InitializeMokaCommand{
			OtherTrxCode: req.OtherTrxCode, CardHolderFullName: req.CardHolderFullName,
			CardNumber: req.CardNumber, ExpMonth: req.ExpMonth, ExpYear: req.ExpYear,
			CvcNumber: req.CvcNumber, Amount: req.Amount, Currency: req.Currency,
			InstallmentNumber: req.InstallmentNumber, ClientIP: c.IP(),
			Description: req.Description, BuyerInformation: req.BuyerInformation,
			IsPoolPayment: req.IsPoolPayment, IsPreAuth: req.IsPreAuth,
			IsTokenized: req.IsTokenized, RedirectType: req.RedirectType,
		})
		if err != nil {
			return render.Error(c, err)
		}
		return render.JSON(c, fiber.StatusOK, result)
	default:
		return render.Error(c, domainpayment.ErrProviderNotActive)
	}
}

type complete3DSRequest struct {
	Reference string `json:"reference" validate:"required"`
}

// Complete3DS, POST /payments/3ds/auth — 3DS tamamlama.
func (h *PaymentHandler) Complete3DS(c fiber.Ctx) error {
	var req complete3DSRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	resp, err := h.threeds.Complete3DS(c.Context(), req.Reference)
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, resp)
}

type calcPaymentAmountRequest struct {
	BinNumber          string  `json:"binNumber" validate:"required,min=6,max=19"`
	OrderAmount        float64 `json:"orderAmount" validate:"required,gt=0"`
	InstallmentNumber  int     `json:"installmentNumber" validate:"min=0,max=12"`
	Currency           string  `json:"currency"`
	IsThreeD           int     `json:"isThreeD" validate:"oneof=0 1"`
	GroupRevenueRate   float64 `json:"groupRevenueRate"`
	GroupRevenueAmount float64 `json:"groupRevenueAmount"`
}

// CalcPaymentAmount, POST /payments/calc-amount — Moka taksit tutarı hesaplama.
func (h *PaymentHandler) CalcPaymentAmount(c fiber.Ctx) error {
	var req calcPaymentAmountRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	if req.IsThreeD == 0 {
		req.IsThreeD = 1
	}
	resp, err := h.threeds.CalcPaymentAmount(c.Context(), apppayment.CalcPaymentAmountCommand{
		BinNumber: req.BinNumber, OrderAmount: req.OrderAmount,
		InstallmentNumber: req.InstallmentNumber, Currency: req.Currency,
		IsThreeD: req.IsThreeD, GroupRevenueRate: req.GroupRevenueRate,
		GroupRevenueAmount: req.GroupRevenueAmount,
	})
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, resp)
}

// ListPayments, GET /api/v1/payments/transactions — ödeme listesi.
func (h *PaymentHandler) ListPayments(c fiber.Ctx) error {
	page, err := h.threeds.ListPayments(c.Context(), apppayment.ListPaymentsQuery{
		Status: c.Query("status"), Provider: c.Query("provider"),
		Page: adapters.ParsePage(c.Query("page")), Limit: adapters.ParseLimit(c.Query("limit")),
		Ascending: c.Query("order") == "asc",
	})
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, page)
}

// IyzicoWebhook, POST /payments/webhook/iyzico — imzalı iyzico bildirimi (public).
func (h *PaymentHandler) IyzicoWebhook(c fiber.Ctx) error {
	var payload iyzico.WebhookPayload
	if err := c.Bind().Body(&payload); err != nil {
		return render.Error(c, err)
	}
	result, err := h.threeds.HandleIyzicoWebhook(c.Context(), payload, c.Get("X-IYZ-SIGNATURE-V3"))
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, result)
}

// Callback3DS, POST /payments/3ds/callback — iyzico veya Moka yönlendirme (public).
func (h *PaymentHandler) Callback3DS(c fiber.Ctx) error {
	if c.FormValue("hashValue") != "" || c.FormValue("OtherTrxCode") != "" {
		payload := apppayment.MokaCallbackPayload{
			HashValue: c.FormValue("hashValue"), ResultCode: c.FormValue("resultCode"),
			ResultMessage: c.FormValue("resultMessage"), TrxCode: c.FormValue("trxCode"),
			OtherTrxCode: c.FormValue("OtherTrxCode"),
		}
		result, err := h.threeds.HandleMokaCallback(c.Context(), payload)
		if err != nil {
			return c.Type("html").Status(fiber.StatusOK).SendString(callbackFailureHTML(payload.OtherTrxCode, err.Error()))
		}
		return c.Type("html").Status(fiber.StatusOK).SendString(callbackSuccessHTML(result))
	}

	payload := apppayment.IyzicoCallbackPayload{
		Status: c.FormValue("status"), PaymentID: c.FormValue("paymentId"),
		ConversationData: c.FormValue("conversationData"), Reference: c.FormValue("conversationId"),
		MDStatus: c.FormValue("mdStatus"),
	}
	result, err := h.threeds.HandleIyzicoCallback(c.Context(), payload)
	if err != nil {
		return c.Type("html").Status(fiber.StatusOK).SendString(callbackFailureHTML(payload.Reference, err.Error()))
	}
	return c.Type("html").Status(fiber.StatusOK).SendString(callbackSuccessHTML(result))
}

func callbackSuccessHTML(result apppayment.CallbackResult) string {
	return `<!doctype html><html lang="tr"><head><meta charset="utf-8"><title>Ödeme başarılı</title></head><body>
<h1>Ödeme tamamlandı</h1>
<p>Sağlayıcı: ` + html.EscapeString(result.Provider) + `</p>
<p>Referans: ` + html.EscapeString(result.Reference) + `</p>
<p>İşlem kodu: ` + html.EscapeString(result.PaymentID) + `</p>
<p><a href="/dashboard/payments/checkout">Yeni ödeme</a> · <a href="/dashboard">Panel</a></p>
</body></html>`
}

func callbackFailureHTML(reference, msg string) string {
	return `<!doctype html><html lang="tr"><head><meta charset="utf-8"><title>Ödeme başarısız</title></head><body>
<h1>Ödeme tamamlanamadı</h1>
<p>Referans: ` + html.EscapeString(reference) + `</p>
<p>` + html.EscapeString(msg) + `</p>
<p><a href="/dashboard/payments/checkout">Tekrar dene</a></p>
</body></html>`
}

// DecodeIyzicoHTML, base64 threeDSHtmlContent değerini çözer.
func DecodeIyzicoHTML(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", domainpayment.ErrInvalidCallback
	}
	if strings.HasPrefix(strings.ToLower(raw), "<!doctype") || strings.HasPrefix(strings.ToLower(raw), "<html") {
		return raw, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
