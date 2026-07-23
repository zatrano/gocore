package goui

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	adaptershared "github.com/zatrano/gocore/internal/adapters/shared"
	apppayment "github.com/zatrano/gocore/internal/application/payment"
	domainsettings "github.com/zatrano/gocore/internal/domain/settings"
	"github.com/zatrano/gocore/internal/infrastructure/payment/iyzico"
	"github.com/zatrano/gocore/internal/infrastructure/payment/moka"
	"github.com/zatrano/gocore/pkg/pagination"
	"github.com/zatrano/gocore/pkg/rbac"
	"github.com/zatrano/gocore/pkg/validation"
)

func decodeIyzicoHTML(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("3DS HTML içeriği boş")
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "<!doctype") || strings.HasPrefix(lower, "<html") {
		return raw, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// --- Checkout ---

type checkoutFormState struct {
	Amount, Currency, CardHolder, CardNumber, ExpMonth, ExpYear, Cvc string
	BuyerName, BuyerSurname, BuyerEmail, BuyerPhone                  string
	BuyerIdentity, BuyerAddress, BuyerCity, BuyerCountry, BuyerZip   string
	Installment                                                      int
	BinNumber, BinPrice                                              string
}

type checkoutCtrl struct {
	activeProvider string
	iyzicoActive   bool
	mokaActive     bool
	configured     bool
	form           checkoutFormState
	binResult      string
	calcResult     string
}

func (c *checkoutCtrl) Mount(ctx context.Context, p *Page) error {
	if err := requireAnyPerm(ctx, p, rbac.PermPaymentsCharge); err != nil {
		return err
	}
	st := adaptershared.PaymentIntegrationStatus(p.Deps.Payment)
	view, err := p.Deps.Settings.GetPaymentView(ctx, st)
	if err != nil {
		return err
	}
	c.activeProvider = view.ActiveProvider
	c.iyzicoActive = view.ActiveProvider == domainsettings.ProviderIyzico.String()
	c.mokaActive = view.ActiveProvider == domainsettings.ProviderMoka.String()
	c.configured = false
	for _, row := range view.Providers {
		if row.Active {
			c.configured = row.Configured
			break
		}
	}
	if c.form.Currency == "" {
		if c.mokaActive {
			c.form.Currency = "TL"
		} else {
			c.form.Currency = "TRY"
		}
	}
	if c.form.Installment <= 0 {
		c.form.Installment = 1
	}
	if c.form.BuyerCountry == "" {
		c.form.BuyerCountry = "Turkey"
	}
	if c.form.BinPrice == "" && c.form.Amount != "" {
		c.form.BinPrice = c.form.Amount
	}
	if c.form.BinPrice == "" {
		c.form.BinPrice = "100"
	}
	return nil
}

func (c *checkoutCtrl) Render(p *Page) (string, error) {
	binPriceLabel := "Tutar (opsiyonel)"
	if c.iyzicoActive {
		binPriceLabel = "Tutar (iyzico taksit için zorunlu)"
	}
	return p.RenderView("pages.checkout", map[string]any{
		"Configured":     c.configured,
		"ActiveProvider": c.activeProvider,
		"MokaActive":     c.mokaActive,
		"IyzicoActive":   c.iyzicoActive,
		"BinNumber":      c.form.BinNumber,
		"BinPrice":       c.form.BinPrice,
		"BinPriceLabel":  binPriceLabel,
		"BinResult":      c.binResult,
		"CalcResult":     c.calcResult,
		"Amount":         c.form.Amount,
		"Currency":       c.form.Currency,
		"Installment":    strconv.Itoa(c.form.Installment),
		"CardHolder":     c.form.CardHolder,
		"CardNumber":     c.form.CardNumber,
		"ExpMonth":       c.form.ExpMonth,
		"ExpYear":        c.form.ExpYear,
		"Cvc":            c.form.Cvc,
		"BuyerName":      c.form.BuyerName,
		"BuyerSurname":   c.form.BuyerSurname,
		"BuyerEmail":     c.form.BuyerEmail,
		"BuyerPhone":     c.form.BuyerPhone,
		"BuyerIdentity":  c.form.BuyerIdentity,
		"BuyerCity":      c.form.BuyerCity,
		"BuyerAddress":   c.form.BuyerAddress,
		"BuyerCountry":   c.form.BuyerCountry,
		"BuyerZip":       c.form.BuyerZip,
	})
}

func (c *checkoutCtrl) applyField(event, value string) {
	switch event {
	case "field.amount":
		c.form.Amount = value
	case "field.currency":
		c.form.Currency = value
	case "field.installment":
		n, _ := strconv.Atoi(value)
		if n > 0 {
			c.form.Installment = n
		}
	case "field.card_holder":
		c.form.CardHolder = value
	case "field.card_number":
		c.form.CardNumber = value
	case "field.exp_month":
		c.form.ExpMonth = value
	case "field.exp_year":
		c.form.ExpYear = value
	case "field.cvc":
		c.form.Cvc = value
	case "field.buyer_name":
		c.form.BuyerName = value
	case "field.buyer_surname":
		c.form.BuyerSurname = value
	case "field.buyer_email":
		c.form.BuyerEmail = value
	case "field.buyer_phone":
		c.form.BuyerPhone = value
	case "field.buyer_identity":
		c.form.BuyerIdentity = value
	case "field.buyer_address":
		c.form.BuyerAddress = value
	case "field.buyer_city":
		c.form.BuyerCity = value
	case "field.buyer_country":
		c.form.BuyerCountry = value
	case "field.buyer_zip":
		c.form.BuyerZip = value
	case "field.bin_number":
		c.form.BinNumber = value
	case "field.bin_price":
		c.form.BinPrice = value
	}
}

func (c *checkoutCtrl) syncFormFromPayload(payload map[string]any) {
	fields := payloadFields(payload)
	set := func(key string, dst *string) {
		if v, ok := fields[key]; ok {
			if s, ok := v.(string); ok {
				*dst = strings.TrimSpace(s)
			}
		}
	}
	set("amount", &c.form.Amount)
	set("currency", &c.form.Currency)
	set("card_holder", &c.form.CardHolder)
	set("card_number", &c.form.CardNumber)
	set("exp_month", &c.form.ExpMonth)
	set("exp_year", &c.form.ExpYear)
	set("cvc", &c.form.Cvc)
	set("buyer_name", &c.form.BuyerName)
	set("buyer_surname", &c.form.BuyerSurname)
	set("buyer_email", &c.form.BuyerEmail)
	set("buyer_phone", &c.form.BuyerPhone)
	set("buyer_identity", &c.form.BuyerIdentity)
	set("buyer_address", &c.form.BuyerAddress)
	set("buyer_city", &c.form.BuyerCity)
	set("buyer_country", &c.form.BuyerCountry)
	set("buyer_zip", &c.form.BuyerZip)
	set("bin_number", &c.form.BinNumber)
	set("bin_price", &c.form.BinPrice)
	if v, ok := fields["installment"]; ok {
		switch n := v.(type) {
		case string:
			if i, err := strconv.Atoi(strings.TrimSpace(n)); err == nil && i > 0 {
				c.form.Installment = i
			}
		case float64:
			if int(n) > 0 {
				c.form.Installment = int(n)
			}
		}
	}
}

type checkoutSubmitDTO struct {
	Amount             string `validate:"required"`
	Currency           string
	Installment        int    `validate:"min=1,max=12"`
	CardHolderFullName string `validate:"required"`
	CardNumber         string `validate:"required"`
	ExpMonth           string `validate:"required"`
	ExpYear            string `validate:"required"`
	CvcNumber          string `validate:"required"`
	BuyerName          string `validate:"required"`
	BuyerSurname       string `validate:"required"`
	BuyerEmail         string `validate:"required,email" sanitize:"email"`
	BuyerPhone         string `validate:"omitempty,phone" sanitize:"phone"`
	BuyerIdentity      string
	BuyerAddress       string `validate:"required"`
	BuyerCity          string `validate:"required"`
	BuyerCountry       string
	BuyerZip           string
}

func (c *checkoutCtrl) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	if err := requireAnyPerm(ctx, p, rbac.PermPaymentsCharge); err != nil {
		return err
	}
	if strings.HasPrefix(event, "field.") {
		c.applyField(event, payloadValue(payload))
		return nil
	}
	switch event {
	case "checkout.bin_check":
		c.syncFormFromPayload(payload)
		return c.binCheck(ctx, p)
	case "checkout.calc_amount":
		c.syncFormFromPayload(payload)
		return c.calcAmount(ctx, p)
	case "checkout.submit":
		c.syncFormFromPayload(payload)
		return c.submit(ctx, p)
	case "checkout.complete":
		ref := payloadString(payload, "reference")
		if ref == "" {
			ref = payloadValue(payload)
		}
		if p.Deps.ThreeDSSvc == nil {
			return errors.New("3DS servisi yapılandırılmamış")
		}
		result, err := p.Deps.ThreeDSSvc.Complete3DS(ctx, ref)
		if err != nil {
			return err
		}
		raw, _ := json.MarshalIndent(result, "", "  ")
		p.Notice = "3DS tamamlandı"
		c.binResult = string(raw)
		return nil
	}
	return nil
}

func (c *checkoutCtrl) binCheck(ctx context.Context, p *Page) error {
	if p.Deps.ThreeDSSvc == nil {
		return errors.New("3DS servisi yapılandırılmamış")
	}
	req := struct {
		BinNumber string `validate:"required,min=6,max=19"`
		Price     string
		Locale    string
	}{BinNumber: c.form.BinNumber, Price: c.form.BinPrice, Locale: "tr"}
	if err := validateDeps(p, &req); err != nil {
		return err
	}
	if c.iyzicoActive && strings.TrimSpace(req.Price) == "" {
		return errors.New("iyzico BIN sorgusu için tutar zorunludur")
	}
	result, err := p.Deps.ThreeDSSvc.BinCheck(ctx, apppayment.BinCheckQuery{
		Locale: req.Locale, Price: req.Price, BinNumber: req.BinNumber,
	})
	if err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(result, "", "  ")
	c.binResult = string(raw)
	return nil
}

func (c *checkoutCtrl) calcAmount(ctx context.Context, p *Page) error {
	if p.Deps.ThreeDSSvc == nil {
		return errors.New("3DS servisi yapılandırılmamış")
	}
	amount, err := strconv.ParseFloat(strings.TrimSpace(c.form.Amount), 64)
	if err != nil || amount <= 0 {
		return errors.New("geçerli bir sipariş tutarı giriniz")
	}
	req := struct {
		BinNumber         string  `validate:"required,min=6,max=19"`
		OrderAmount       float64 `validate:"required,gt=0"`
		InstallmentNumber int     `validate:"min=0,max=12"`
	}{BinNumber: c.form.BinNumber, OrderAmount: amount, InstallmentNumber: c.form.Installment}
	if err := validation.Check(p.Deps.Validate, &req); err != nil {
		return err
	}
	result, err := p.Deps.ThreeDSSvc.CalcPaymentAmount(ctx, apppayment.CalcPaymentAmountCommand{
		BinNumber: req.BinNumber, OrderAmount: req.OrderAmount,
		InstallmentNumber: req.InstallmentNumber, Currency: c.form.Currency, IsThreeD: 1,
	})
	if err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(result, "", "  ")
	c.calcResult = string(raw)
	return nil
}

func (c *checkoutCtrl) submit(ctx context.Context, p *Page) error {
	if p.Deps.ThreeDSSvc == nil {
		return errors.New("3DS servisi yapılandırılmamış")
	}
	dto := checkoutSubmitDTO{
		Amount: c.form.Amount, Currency: c.form.Currency, Installment: c.form.Installment,
		CardHolderFullName: c.form.CardHolder, CardNumber: c.form.CardNumber,
		ExpMonth: c.form.ExpMonth, ExpYear: c.form.ExpYear, CvcNumber: c.form.Cvc,
		BuyerName: c.form.BuyerName, BuyerSurname: c.form.BuyerSurname,
		BuyerEmail: c.form.BuyerEmail, BuyerPhone: c.form.BuyerPhone,
		BuyerIdentity: c.form.BuyerIdentity, BuyerAddress: c.form.BuyerAddress,
		BuyerCity: c.form.BuyerCity, BuyerCountry: c.form.BuyerCountry, BuyerZip: c.form.BuyerZip,
	}
	if dto.Installment <= 0 {
		dto.Installment = 1
	}
	if err := validateDeps(p, &dto); err != nil {
		return err
	}
	defer func() {
		c.form.CardNumber = ""
		c.form.Cvc = ""
		c.form.ExpMonth = ""
		c.form.ExpYear = ""
		c.form.BinNumber = ""
	}()
	c.form.BuyerEmail = dto.BuyerEmail
	c.form.BuyerPhone = dto.BuyerPhone

	provider, err := p.Deps.ThreeDSSvc.ActiveProvider(ctx)
	if err != nil {
		return err
	}
	ip := actorClientIP(ctx)
	if ip == "" {
		return errClientIPMissing
	}
	switch provider {
	case domainsettings.ProviderIyzico.String():
		if strings.TrimSpace(dto.BuyerIdentity) == "" {
			return errors.New("TC kimlik numarası zorunludur")
		}
		return c.submitIyzico(ctx, p, dto, ip)
	case domainsettings.ProviderMoka.String():
		return c.submitMoka(ctx, p, dto, ip)
	default:
		return domainsettings.ErrInvalidPaymentProvider
	}
}

func (c *checkoutCtrl) submitIyzico(ctx context.Context, p *Page, form checkoutSubmitDTO, ip string) error {
	price := strings.TrimSpace(form.Amount)
	currency := form.Currency
	if currency == "" {
		currency = "TRY"
	}
	now := time.Now().UTC()
	basketID := "WEB-" + now.Format("20060102150405")
	result, err := p.Deps.ThreeDSSvc.InitializeIyzico3DS(ctx, apppayment.InitializeIyzicoCommand{
		Locale: "tr", Price: price, PaidPrice: price, Currency: currency,
		Installment: form.Installment, PaymentChannel: "WEB",
		BasketID: basketID, PaymentGroup: "PRODUCT",
		PaymentCard: iyzico.PaymentCard{
			CardHolderName: form.CardHolderFullName,
			CardNumber:     strings.ReplaceAll(form.CardNumber, " ", ""),
			ExpireMonth:    form.ExpMonth, ExpireYear: form.ExpYear, CVC: form.CvcNumber,
		},
		Buyer: iyzico.Buyer{
			ID: "BY-" + basketID, Name: form.BuyerName, Surname: form.BuyerSurname,
			IdentityNumber: form.BuyerIdentity, Email: form.BuyerEmail,
			GsmNumber: form.BuyerPhone, RegistrationDate: now.Format("2006-01-02 15:04:05"),
			LastLoginDate: now.Format("2006-01-02 15:04:05"), RegistrationAddress: form.BuyerAddress,
			City: form.BuyerCity, Country: defaultStrSPA(form.BuyerCountry, "Turkey"),
			ZipCode: form.BuyerZip, IP: ip,
		},
		ShippingAddress: iyzico.Address{
			Address: form.BuyerAddress, ZipCode: form.BuyerZip,
			ContactName: form.BuyerName + " " + form.BuyerSurname,
			City:        form.BuyerCity, Country: defaultStrSPA(form.BuyerCountry, "Turkey"),
		},
		BillingAddress: iyzico.Address{
			Address: form.BuyerAddress, ZipCode: form.BuyerZip,
			ContactName: form.BuyerName + " " + form.BuyerSurname,
			City:        form.BuyerCity, Country: defaultStrSPA(form.BuyerCountry, "Turkey"),
		},
		BasketItems: []iyzico.BasketItem{{
			ID: "BI1", Price: price, Name: "Ödeme", Category1: "Payment", Category2: "Web", ItemType: "VIRTUAL",
		}},
	})
	if err != nil {
		return err
	}
	p.Redirect = "/dashboard/payments/3ds/start?reference=" + result.Reference
	p.Notice = "3DS başlatıldı"
	return nil
}

func (c *checkoutCtrl) submitMoka(ctx context.Context, p *Page, form checkoutSubmitDTO, ip string) error {
	amount, err := strconv.ParseFloat(strings.TrimSpace(form.Amount), 64)
	if err != nil || amount <= 0 {
		return errors.New("geçerli bir tutar giriniz")
	}
	currency := form.Currency
	if currency == "" {
		currency = "TL"
	}
	installment := form.Installment
	if installment <= 0 {
		installment = 1
	}
	result, err := p.Deps.ThreeDSSvc.InitializeMoka3DS(ctx, apppayment.InitializeMokaCommand{
		CardHolderFullName: form.CardHolderFullName,
		CardNumber:         strings.ReplaceAll(form.CardNumber, " ", ""),
		ExpMonth:           form.ExpMonth, ExpYear: form.ExpYear, CvcNumber: form.CvcNumber,
		Amount: amount, Currency: currency, InstallmentNumber: installment,
		ClientIP: ip, Description: "Web checkout",
		BuyerInformation: &moka.BuyerInformation{
			BuyerFullName: form.BuyerName + " " + form.BuyerSurname,
			BuyerEmail:    form.BuyerEmail, BuyerGsmNumber: form.BuyerPhone,
			BuyerAddress: form.BuyerAddress,
		},
	})
	if err != nil {
		return err
	}
	p.Redirect = "/dashboard/payments/3ds/start?reference=" + result.Reference
	p.Notice = "3DS başlatıldı"
	return nil
}

func defaultStrSPA(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

// --- Payments list ---

type paymentsListCtrl struct {
	status, provider, order string
	pageNum, limit          int
	page                    pagination.Page[apppayment.PaymentView]
}

func (c *paymentsListCtrl) Mount(ctx context.Context, p *Page) error {
	if err := requireAnyPerm(ctx, p, rbac.PermPaymentsList); err != nil {
		return err
	}
	c.status = pageQuery(p, "status", c.status)
	c.provider = pageQuery(p, "provider", c.provider)
	c.order = pageQuery(p, "order", c.order)
	c.pageNum, c.limit = parsePageLimit(p)
	return c.reload(ctx, p)
}

func (c *paymentsListCtrl) reload(ctx context.Context, p *Page) error {
	if p.Deps.ThreeDSSvc == nil {
		return errors.New("3DS servisi yapılandırılmamış")
	}
	page, err := p.Deps.ThreeDSSvc.ListPayments(ctx, apppayment.ListPaymentsQuery{
		Status: c.status, Provider: c.provider,
		Page: c.pageNum, Limit: c.limit,
		Ascending: c.order == "asc",
	})
	if err != nil {
		return err
	}
	c.page = page
	c.pageNum, c.limit = page.Page, page.Limit
	return nil
}

type paymentRow struct {
	Reference     string
	Provider      string
	StatusLabel   string
	BuyerDisplay  string
	BuyerEmail    string
	AmountDisplay string
	Installment   string
	CardDisplay   string
	CreatedAt     string
	DetailHref    string
}

func (c *paymentsListCtrl) Render(p *Page) (string, error) {
	items := make([]paymentRow, 0, len(c.page.Items))
	for _, item := range c.page.Items {
		name := strings.TrimSpace(item.BuyerName + " " + item.BuyerSurname)
		if name == "" {
			name = "—"
		}
		amt := item.Amount
		if item.PaidAmount != "" {
			amt = item.PaidAmount
		}
		card := item.CardDisplay
		if card == "" {
			card = "—"
		}
		items = append(items, paymentRow{
			Reference:     item.Reference,
			Provider:      item.Provider,
			StatusLabel:   item.StatusLabel,
			BuyerDisplay:  name,
			BuyerEmail:    item.BuyerEmail,
			AmountDisplay: amt + " " + item.Currency,
			Installment:   strconv.Itoa(item.Installment),
			CardDisplay:   card,
			CreatedAt:     formatShort(item.CreatedAt),
			DetailHref:    "/dashboard/payments/transactions/" + item.Reference,
		})
	}
	return p.RenderView("pages.payments", map[string]any{
		"ExportLinks":     viewExportLinks("/dashboard/payments/transactions/export", map[string]string{"status": c.status, "provider": c.provider, "order": c.order}),
		"Status":          c.status,
		"Provider":        c.provider,
		"Order":           c.order,
		"Limit":           strconv.Itoa(c.limit),
		"StatusOptions":   viewSelectOptions([][2]string{{"", "Tümü"}, {"pending", "Beklemede"}, {"success", "Başarılı"}, {"failed", "Başarısız"}}, c.status),
		"ProviderOptions": viewSelectOptions([][2]string{{"", "Tümü"}, {"iyzico", "Iyzico"}, {"moka", "Moka"}}, c.provider),
		"OrderOptions":    viewSelectOptions([][2]string{{"", "Yeniden eskiye"}, {"asc", "Eskiden yeniye"}}, c.order),
		"LimitOptions":    viewLimitOptions(c.limit),
		"Items":           items,
		"Pages":           viewPagination(c.pageNum, c.page.TotalPages),
	})
}

func (c *paymentsListCtrl) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	if err := requireAnyPerm(ctx, p, rbac.PermPaymentsList); err != nil {
		return err
	}
	switch event {
	case "field.status":
		c.status = payloadValue(payload)
	case "field.provider":
		c.provider = payloadValue(payload)
	case "field.order":
		c.order = payloadValue(payload)
	case "field.limit":
		c.limit = adaptershared.ParseLimit(payloadValue(payload))
	case "payments.filter":
		c.status = payloadString(payload, "status")
		c.provider = payloadString(payload, "provider")
		c.order = payloadString(payload, "order")
		c.limit = adaptershared.ParseLimit(payloadString(payload, "limit"))
		c.pageNum = 1
		return c.reload(ctx, p)
	case "payments.clear":
		c.status, c.provider, c.order = "", "", ""
		c.pageNum = 1
		c.limit = 0
		return c.reload(ctx, p)
	case "payments.page":
		c.pageNum = payloadPage(payload, c.pageNum)
		return c.reload(ctx, p)
	}
	return nil
}

// --- Payment show ---

type paymentShowCtrl struct {
	payment apppayment.PaymentView
}

func (c *paymentShowCtrl) Mount(ctx context.Context, p *Page) error {
	if err := requireAnyPerm(ctx, p, rbac.PermPaymentsCharge, rbac.PermPaymentsList); err != nil {
		return err
	}
	if p.Deps.ThreeDSSvc == nil {
		return errors.New("3DS servisi yapılandırılmamış")
	}
	ref := ""
	if p.Params != nil {
		ref = p.Params["reference"]
	}
	view, err := p.Deps.ThreeDSSvc.GetPayment(ctx, ref)
	if err != nil {
		return err
	}
	c.payment = view
	return nil
}

func (c *paymentShowCtrl) Render(p *Page) (string, error) {
	pay := c.payment
	rows := []ViewDetail{
		{Label: "Referans", Value: pay.Reference},
		{Label: "Ödeme ID", Value: pay.ID},
		{Label: "Sağlayıcı", Value: pay.Provider},
		{Label: "Durum", Value: pay.StatusLabel},
		{Label: "Tutar", Value: pay.Amount + " " + pay.Currency},
	}
	if pay.PaidAmount != "" {
		rows = append(rows, ViewDetail{Label: "Tahsil edilen", Value: pay.PaidAmount + " " + pay.Currency})
	}
	rows = append(rows, ViewDetail{Label: "Taksit", Value: strconv.Itoa(pay.Installment)})
	if pay.BuyerName != "" || pay.BuyerSurname != "" {
		rows = append(rows, ViewDetail{Label: "Alıcı", Value: strings.TrimSpace(pay.BuyerName + " " + pay.BuyerSurname)})
	}
	if pay.BuyerEmail != "" {
		rows = append(rows, ViewDetail{Label: "E-posta", Value: pay.BuyerEmail})
	}
	if pay.BuyerPhone != "" {
		rows = append(rows, ViewDetail{Label: "Telefon", Value: pay.BuyerPhone})
	}
	if pay.CardHolder != "" {
		rows = append(rows, ViewDetail{Label: "Kart sahibi", Value: pay.CardHolder})
	}
	if pay.CardDisplay != "" {
		card := pay.CardDisplay
		if pay.CardAssociation != "" {
			card += " (" + pay.CardAssociation + ")"
		}
		rows = append(rows, ViewDetail{Label: "Kart", Value: card})
	}
	if pay.ProviderPaymentID != "" {
		rows = append(rows, ViewDetail{Label: "Sağlayıcı işlem ID", Value: pay.ProviderPaymentID})
	}
	if pay.AuthCode != "" {
		rows = append(rows, ViewDetail{Label: "Onay kodu", Value: pay.AuthCode})
	}
	if pay.ResultCode != "" {
		rows = append(rows, ViewDetail{Label: "Sonuç kodu", Value: pay.ResultCode})
	}
	if pay.ResultMessage != "" {
		rows = append(rows, ViewDetail{Label: "Sonuç mesajı", Value: pay.ResultMessage})
	}
	rows = append(rows,
		ViewDetail{Label: "Oluşturulma", Value: formatShort(pay.CreatedAt)},
		ViewDetail{Label: "Güncellenme", Value: formatShort(pay.UpdatedAt)},
	)
	if pay.CompletedAt != nil {
		rows = append(rows, ViewDetail{Label: "Tamamlanma", Value: formatShortPtr(pay.CompletedAt)})
	}
	return p.RenderView("pages.payment_show", map[string]any{
		"Reference": pay.Reference,
		"Rows":      rows,
	})
}

func (c *paymentShowCtrl) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	if err := requireAnyPerm(ctx, p, rbac.PermPaymentsCharge, rbac.PermPaymentsList); err != nil {
		return err
	}
	if event != "payment.complete" {
		return nil
	}
	if err := requireAnyPerm(ctx, p, rbac.PermPaymentsCharge); err != nil {
		return err
	}
	ref := payloadString(payload, "reference")
	if ref == "" && p.Params != nil {
		ref = p.Params["reference"]
	}
	if p.Deps.ThreeDSSvc == nil {
		return errors.New("3DS servisi yapılandırılmamış")
	}
	if _, err := p.Deps.ThreeDSSvc.Complete3DS(ctx, ref); err != nil {
		return err
	}
	p.Notice = "3DS tamamlandı"
	return c.Mount(ctx, p)
}
