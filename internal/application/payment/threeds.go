package payment

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	appidempotency "github.com/zatrano/gocore/internal/application/idempotency"
	appsettings "github.com/zatrano/gocore/internal/application/settings"
	appshared "github.com/zatrano/gocore/internal/application/shared"
	domainpayment "github.com/zatrano/gocore/internal/domain/payment"
	domainsettings "github.com/zatrano/gocore/internal/domain/settings"
	dshared "github.com/zatrano/gocore/internal/domain/shared"
	"github.com/zatrano/gocore/internal/infrastructure/config"
	infrapayment "github.com/zatrano/gocore/internal/infrastructure/payment"
	"github.com/zatrano/gocore/internal/infrastructure/payment/iyzico"
	"github.com/zatrano/gocore/internal/infrastructure/payment/moka"
	"github.com/zatrano/gocore/pkg/datetime"
	"github.com/zatrano/gocore/pkg/pagination"
)

// ThreeDSService, 3DS ödeme use-case'lerini yönetir.
type ThreeDSService struct {
	settings   *appsettings.Service
	repo       domainpayment.Repository
	iyzico     *iyzico.Client
	moka       *moka.Client
	paymentCfg config.Payment
	baseURL    string
	idem       *appidempotency.Service
	publisher  appshared.EventPublisher
	tx         appshared.TxManager
}

// ThreeDSDeps, ThreeDSService bağımlılıklarını gruplar.
type ThreeDSDeps struct {
	Settings   *appsettings.Service
	Repo       domainpayment.Repository
	Iyzico     *iyzico.Client
	Moka       *moka.Client
	PaymentCfg config.Payment
	BaseURL    string
	Idem       *appidempotency.Service
	Publisher  appshared.EventPublisher
	Tx         appshared.TxManager
}

// NewThreeDSService, servisi kurar.
func NewThreeDSService(d ThreeDSDeps) *ThreeDSService {
	return &ThreeDSService{
		settings:   d.Settings,
		repo:       d.Repo,
		iyzico:     d.Iyzico,
		moka:       d.Moka,
		paymentCfg: d.PaymentCfg,
		baseURL:    strings.TrimSuffix(d.BaseURL, "/"),
		idem:       d.Idem,
		publisher:  d.Publisher,
		tx:         d.Tx,
	}
}

func (s *ThreeDSService) publish(ctx context.Context, events ...dshared.DomainEvent) error {
	if s.publisher == nil || len(events) == 0 {
		return nil
	}
	return s.publisher.Publish(ctx, events...)
}

func (s *ThreeDSService) persistPayment(ctx context.Context, payment *domainpayment.Payment, op func(context.Context) error) error {
	run := func(ctx context.Context) error {
		if err := op(ctx); err != nil {
			return err
		}
		return s.publish(ctx, payment.PullEvents()...)
	}
	if s.tx != nil {
		return s.tx.WithinTx(ctx, run)
	}
	return run(ctx)
}

func (s *ThreeDSService) saveAndPublish(ctx context.Context, payment *domainpayment.Payment) error {
	return s.persistPayment(ctx, payment, func(ctx context.Context) error {
		return s.repo.Save(ctx, payment)
	})
}

func (s *ThreeDSService) updateAndPublish(ctx context.Context, payment *domainpayment.Payment) error {
	return s.persistPayment(ctx, payment, func(ctx context.Context) error {
		return s.repo.Update(ctx, payment)
	})
}

func (s *ThreeDSService) integrationStatus() appsettings.PaymentIntegrationStatus {
	return infrapayment.IntegrationStatusFromConfig(s.paymentCfg).ToApplication()
}

// ActiveProvider, dashboard'da seçili ödeme sağlayıcısını döner.
func (s *ThreeDSService) ActiveProvider(ctx context.Context) (string, error) {
	view, err := s.settings.GetPaymentView(ctx, s.integrationStatus())
	if err != nil {
		return "", err
	}
	return view.ActiveProvider, nil
}

func (s *ThreeDSService) ensureProvider(ctx context.Context, want domainsettings.PaymentProvider) error {
	st := s.integrationStatus()
	switch want {
	case domainsettings.ProviderIyzico:
		if !st.IyzicoConfigured {
			return domainpayment.ErrProviderNotConfigured
		}
	case domainsettings.ProviderMoka:
		if !st.MokaConfigured {
			return domainpayment.ErrProviderNotConfigured
		}
	}
	view, err := s.settings.GetPaymentView(ctx, st)
	if err != nil {
		return err
	}
	if view.ActiveProvider != want.String() {
		return domainpayment.ErrProviderNotActive
	}
	return nil
}

func (s *ThreeDSService) callbackURL() string {
	if u := strings.TrimSpace(s.paymentCfg.CallbackURL); u != "" {
		return u
	}
	return s.baseURL + "/api/v1/payments/3ds/callback"
}

// BinCheckQuery, BIN sorgu girdisidir.
type BinCheckQuery struct {
	Locale    string
	Price     string
	BinNumber string
}

// BinCheck, aktif sağlayıcıya göre kart BIN bilgisini sorgular.
func (s *ThreeDSService) BinCheck(ctx context.Context, q BinCheckQuery) (BinCheckResult, error) {
	bin := moka.NormalizeBinNumber(q.BinNumber)
	if len(bin) < 6 {
		return BinCheckResult{}, domainpayment.ErrInvalidBin
	}
	provider, err := s.ActiveProvider(ctx)
	if err != nil {
		return BinCheckResult{}, err
	}
	switch provider {
	case domainsettings.ProviderIyzico.String():
		if err := s.ensureProvider(ctx, domainsettings.ProviderIyzico); err != nil {
			return BinCheckResult{}, err
		}
		if strings.TrimSpace(q.Price) == "" {
			return BinCheckResult{}, domainpayment.ErrBinPriceRequired
		}
		resp, err := s.iyzico.BinCheck(ctx, iyzico.BinCheckRequest{
			Locale: q.Locale, Price: q.Price, BinNumber: bin,
		})
		if err != nil {
			return BinCheckResult{}, err
		}
		return BinCheckResult{Provider: provider, BinNumber: bin, Price: q.Price, Iyzico: &resp}, nil
	case domainsettings.ProviderMoka.String():
		if err := s.ensureProvider(ctx, domainsettings.ProviderMoka); err != nil {
			return BinCheckResult{}, err
		}
		resp, err := s.moka.GetBankCardInformation(ctx, bin)
		if err != nil {
			return BinCheckResult{}, err
		}
		return BinCheckResult{Provider: provider, BinNumber: bin, Moka: &resp}, nil
	default:
		return BinCheckResult{}, domainpayment.ErrProviderNotActive
	}
}

// GetPayment, referans ile ödeme kaydını döner.
func (s *ThreeDSService) GetPayment(ctx context.Context, reference string) (PaymentView, error) {
	payment, err := s.repo.FindByReference(ctx, reference)
	if err != nil {
		return PaymentView{}, err
	}
	return paymentView(payment), nil
}

// ListPaymentsQuery, ödeme listesi girdisidir.
type ListPaymentsQuery struct {
	Status    string
	Provider  string
	Page      int
	Limit     int
	Ascending bool
}

// ListPayments, ödeme kayıtlarını sayfalar.
func (s *ThreeDSService) ListPayments(ctx context.Context, q ListPaymentsQuery) (pagination.Page[PaymentView], error) {
	page, err := s.repo.List(ctx, domainpayment.ListFilter{
		Status: q.Status, Provider: q.Provider,
	}, pagination.Request{Page: q.Page, Limit: q.Limit, Ascending: q.Ascending})
	if err != nil {
		return pagination.Page[PaymentView]{}, err
	}
	views := make([]PaymentView, 0, len(page.Items))
	for _, payment := range page.Items {
		views = append(views, paymentView(payment))
	}
	return pagination.NewPage(views, page.Page, page.Limit, page.Total), nil
}

// CalcPaymentAmountCommand, Moka taksit tutarı hesaplama girdisidir.
type CalcPaymentAmountCommand struct {
	BinNumber          string
	OrderAmount        float64
	InstallmentNumber  int
	Currency           string
	IsThreeD           int
	GroupRevenueRate   float64
	GroupRevenueAmount float64
}

// CalcPaymentAmount, aktif sağlayıcı Moka iken karttan çekilecek tutarı hesaplar.
func (s *ThreeDSService) CalcPaymentAmount(ctx context.Context, cmd CalcPaymentAmountCommand) (CalcPaymentAmountResult, error) {
	if err := s.ensureProvider(ctx, domainsettings.ProviderMoka); err != nil {
		return CalcPaymentAmountResult{}, err
	}
	data, err := s.moka.DoCalcPaymentAmount(ctx, moka.CalcPaymentAmountRequest{
		BinNumber: cmd.BinNumber, OrderAmount: cmd.OrderAmount,
		InstallmentNumber: cmd.InstallmentNumber, Currency: cmd.Currency,
		IsThreeD: cmd.IsThreeD, GroupRevenueRate: cmd.GroupRevenueRate,
		GroupRevenueAmount: cmd.GroupRevenueAmount,
	})
	if err != nil {
		return CalcPaymentAmountResult{}, err
	}
	return CalcPaymentAmountResult{
		Provider: domainsettings.ProviderMoka.String(),
		Moka:     data,
	}, nil
}

// HandleIyzicoWebhook, imzalı iyzico webhook bildirimini doğrular ve oturumu günceller.
func (s *ThreeDSService) HandleIyzicoWebhook(ctx context.Context, payload iyzico.WebhookPayload, signatureV3 string) (WebhookHandleResult, error) {
	secret := strings.TrimSpace(s.paymentCfg.IyzicoSecretKey.Value())
	if secret == "" {
		return WebhookHandleResult{}, domainpayment.ErrProviderNotConfigured
	}
	if strings.TrimSpace(signatureV3) == "" {
		return WebhookHandleResult{}, domainpayment.ErrWebhookSignatureRequired
	}
	if !iyzico.VerifyWebhookSignatureV3(secret, signatureV3, payload) {
		return WebhookHandleResult{}, domainpayment.ErrWebhookInvalidSignature
	}
	result := WebhookHandleResult{
		Verified:      true,
		Acknowledged:  true,
		Status:        payload.Status,
		IyziReference: payload.IyziReferenceCode,
	}
	reference := strings.TrimSpace(payload.PaymentConversationID)
	if reference == "" {
		return result, nil
	}
	result.Reference = reference
	session, err := s.repo.FindByReference(ctx, reference)
	if err != nil {
		if errors.Is(err, domainpayment.ErrPaymentNotFound) {
			return result, nil
		}
		return WebhookHandleResult{}, err
	}
	if session.Provider() != domainsettings.ProviderIyzico.String() {
		return result, nil
	}
	if payload.PaymentID != "" && session.ProviderPaymentID() == "" {
		session.ApplyWebhookPaymentID(payload.PaymentID)
		result.PaymentUpdated = true
	}
	switch strings.ToUpper(strings.TrimSpace(payload.Status)) {
	case "SUCCESS":
		if session.Status() != domainpayment.StatusSuccess {
			session.ApplyWebhookStatus(true)
			result.PaymentUpdated = true
		}
	case "FAILURE":
		if session.Status() != domainpayment.StatusFailed {
			session.ApplyWebhookStatus(false)
			result.PaymentUpdated = true
		}
	}
	if result.PaymentUpdated {
		session.EmitWebhookApplied()
		if err := s.updateAndPublish(ctx, session); err != nil {
			return WebhookHandleResult{}, err
		}
	}
	return result, nil
}

func paymentView(payment *domainpayment.Payment) PaymentView {
	return PaymentView{
		ID:                payment.ID(),
		Reference:         payment.Reference(),
		Provider:          payment.Provider(),
		Status:            string(payment.Status()),
		StatusLabel:       domainpayment.StatusLabel(payment.Status()),
		Stage:             string(payment.Stage()),
		Amount:            payment.Amount(),
		PaidAmount:        payment.PaidAmount(),
		Currency:          payment.Currency(),
		Installment:       payment.Installment(),
		BuyerName:         payment.BuyerName(),
		BuyerSurname:      payment.BuyerSurname(),
		BuyerEmail:        payment.BuyerEmail(),
		BuyerPhone:        payment.BuyerPhone(),
		CardHolder:        domainpayment.MaskCardHolder(payment.CardHolder()),
		CardDisplay:       payment.CardDisplay(),
		CardAssociation:   payment.CardAssociation(),
		ProviderPaymentID: payment.ProviderPaymentID(),
		ResultCode:        payment.ResultCode(),
		ResultMessage:     payment.ResultMessage(),
		AuthCode:          payment.AuthCode(),
		CreatedAt:         datetime.FromTime(payment.CreatedAt()),
		UpdatedAt:         datetime.FromTime(payment.UpdatedAt()),
		CompletedAt:       datetime.PtrFromTime(payment.CompletedAt()),
	}
}

// InitializeIyzicoCommand, iyzico 3DS başlatma girdisidir.
type InitializeIyzicoCommand struct {
	Locale          string
	Reference       string
	Price           string
	PaidPrice       string
	Currency        string
	Installment     int
	PaymentChannel  string
	BasketID        string
	PaymentGroup    string
	PaymentCard     iyzico.PaymentCard
	Buyer           iyzico.Buyer
	ShippingAddress iyzico.Address
	BillingAddress  iyzico.Address
	BasketItems     []iyzico.BasketItem
}

// InitializeMokaCommand, Moka 3DS başlatma girdisidir.
type InitializeMokaCommand struct {
	OtherTrxCode       string
	CardHolderFullName string
	CardNumber         string
	ExpMonth           string
	ExpYear            string
	CvcNumber          string
	Amount             float64
	Currency           string
	InstallmentNumber  int
	ClientIP           string
	Description        string
	BuyerInformation   *moka.BuyerInformation
	IsPoolPayment      int
	IsPreAuth          int
	IsTokenized        int
	RedirectType       int
}

// Initialize3DSResult, 3DS başlatma çıktısıdır.
type Initialize3DSResult struct {
	Provider           string `json:"provider"`
	PaymentID          string `json:"payment_id"`
	Reference          string `json:"reference"`
	ThreeDSHtmlContent string `json:"threeDSHtmlContent,omitempty"`
	RedirectURL        string `json:"redirectUrl,omitempty"`
	CodeForHash        string `json:"codeForHash,omitempty"`
}

// InitializeIyzico3DS, iyzico 3DS akışını başlatır.
func (s *ThreeDSService) InitializeIyzico3DS(ctx context.Context, cmd InitializeIyzicoCommand) (Initialize3DSResult, error) {
	if err := s.ensureProvider(ctx, domainsettings.ProviderIyzico); err != nil {
		return Initialize3DSResult{}, err
	}
	reference := resolvePaymentReference(cmd.Reference, ctx)
	if existing, err := s.repo.FindByReference(ctx, reference); err == nil {
		if res, ok := initResultFromPayment(existing); ok {
			return res, nil
		}
		return Initialize3DSResult{}, appidempotency.ErrConflict
	} else if !errors.Is(err, domainpayment.ErrPaymentNotFound) {
		return Initialize3DSResult{}, err
	}
	resp, err := s.iyzico.Initialize3DS(ctx, iyzico.Init3DSRequest{
		Locale: cmd.Locale, ConversationID: reference,
		Price: cmd.Price, PaidPrice: cmd.PaidPrice, Currency: cmd.Currency,
		Installment: cmd.Installment, PaymentChannel: cmd.PaymentChannel,
		BasketID: cmd.BasketID, PaymentGroup: cmd.PaymentGroup,
		PaymentCard: cmd.PaymentCard, Buyer: cmd.Buyer,
		ShippingAddress: cmd.ShippingAddress, BillingAddress: cmd.BillingAddress,
		BasketItems: cmd.BasketItems, CallbackURL: s.callbackURL(),
	})
	if err != nil {
		return Initialize3DSResult{}, err
	}
	payment := domainpayment.NewPayment(
		uuid.NewString(), reference, domainsettings.ProviderIyzico.String(), cmd.Price, cmd.Currency, cmd.Installment,
	)
	payment.SetPaidAmount(cmd.PaidPrice)
	payment.SetBuyer(cmd.Buyer.Name, cmd.Buyer.Surname, cmd.Buyer.Email, cmd.Buyer.GsmNumber)
	payment.SetCard(cmd.PaymentCard.CardHolderName, cmd.PaymentCard.CardNumber)
	payment.SetInitPayload(resp.ThreeDSHtmlContent)
	payment.EmitInitialized()
	if err := s.saveAndPublish(ctx, payment); err != nil {
		return Initialize3DSResult{}, err
	}
	return Initialize3DSResult{
		Provider:           domainsettings.ProviderIyzico.String(),
		PaymentID:          payment.ID(),
		Reference:          resp.ConversationID,
		ThreeDSHtmlContent: resp.ThreeDSHtmlContent,
	}, nil
}

// InitializeMoka3DS, Moka 3DS akışını başlatır.
func (s *ThreeDSService) InitializeMoka3DS(ctx context.Context, cmd InitializeMokaCommand) (Initialize3DSResult, error) {
	if err := s.ensureProvider(ctx, domainsettings.ProviderMoka); err != nil {
		return Initialize3DSResult{}, err
	}
	otherTrxCode := resolvePaymentReference(cmd.OtherTrxCode, ctx)
	if existing, err := s.repo.FindByReference(ctx, otherTrxCode); err == nil {
		if res, ok := initResultFromPayment(existing); ok {
			return res, nil
		}
		return Initialize3DSResult{}, appidempotency.ErrConflict
	} else if !errors.Is(err, domainpayment.ErrPaymentNotFound) {
		return Initialize3DSResult{}, err
	}
	priceStr := strconv.FormatFloat(cmd.Amount, 'f', -1, 64)
	currency := cmd.Currency
	if currency == "" {
		currency = "TL"
	}
	resp, err := s.moka.DoDirectPaymentThreeD(ctx, moka.DirectPaymentThreeDRequest{
		CardHolderFullName: cmd.CardHolderFullName,
		CardNumber:         cmd.CardNumber,
		ExpMonth:           cmd.ExpMonth,
		ExpYear:            cmd.ExpYear,
		CvcNumber:          cmd.CvcNumber,
		Amount:             cmd.Amount,
		Currency:           currency,
		InstallmentNumber:  cmd.InstallmentNumber,
		ClientIP:           cmd.ClientIP,
		OtherTrxCode:       otherTrxCode,
		IsPoolPayment:      cmd.IsPoolPayment,
		IsPreAuth:          cmd.IsPreAuth,
		IsTokenized:        cmd.IsTokenized,
		RedirectType:       cmd.RedirectType,
		Description:        cmd.Description,
		ReturnHash:         1,
		RedirectUrl:        s.callbackURL(),
		BuyerInformation:   cmd.BuyerInformation,
	})
	if err != nil {
		return Initialize3DSResult{}, err
	}
	installment := cmd.InstallmentNumber
	if installment < 1 {
		installment = 1
	}
	payment := domainpayment.NewPayment(
		uuid.NewString(), otherTrxCode, domainsettings.ProviderMoka.String(), priceStr, currency, installment,
	)
	payment.SetCard(cmd.CardHolderFullName, cmd.CardNumber)
	if cmd.BuyerInformation != nil {
		payment.SetBuyer(cmd.BuyerInformation.BuyerFullName, "", cmd.BuyerInformation.BuyerEmail, cmd.BuyerInformation.BuyerGsmNumber)
	}
	payment.SetCodeForHash(resp.CodeForHash)
	payment.SetInitPayload(resp.URL)
	payment.EmitInitialized()
	if err := s.saveAndPublish(ctx, payment); err != nil {
		return Initialize3DSResult{}, err
	}
	return Initialize3DSResult{
		Provider:    domainsettings.ProviderMoka.String(),
		PaymentID:   payment.ID(),
		Reference:   otherTrxCode,
		RedirectURL: resp.URL,
		CodeForHash: resp.CodeForHash,
	}, nil
}

// IyzicoCallbackPayload, iyzico callback POST gövdesidir.
type IyzicoCallbackPayload struct {
	Status           string
	PaymentID        string
	ConversationData string
	Reference        string
	MDStatus         string
}

// MokaCallbackPayload, Moka redirect POST gövdesidir.
type MokaCallbackPayload struct {
	HashValue     string
	ResultCode    string
	ResultMessage string
	TrxCode       string
	OtherTrxCode  string
}

// CallbackResult, callback işleme sonucudur.
type CallbackResult struct {
	Provider      string `json:"provider"`
	Reference     string `json:"reference"`
	Status        string `json:"status"`
	PaymentID     string `json:"payment_id,omitempty"`
	ResultCode    string `json:"result_code,omitempty"`
	ResultMessage string `json:"result_message,omitempty"`
}

// HandleIyzicoCallback, iyzico callback'ini işler ve auth çağrısı yapar.
func (s *ThreeDSService) HandleIyzicoCallback(ctx context.Context, payload IyzicoCallbackPayload) (CallbackResult, error) {
	session, err := s.repo.FindByReference(ctx, payload.Reference)
	if err != nil {
		return CallbackResult{}, err
	}
	if res, ok := callbackResultFromPayment(session); ok {
		return res, nil
	}
	if session.Provider() != domainsettings.ProviderIyzico.String() {
		return CallbackResult{}, domainpayment.ErrInvalidCallback
	}
	if err := session.ApplyIyzicoCallback(payload.Status, payload.PaymentID, payload.MDStatus, payload.ConversationData); err != nil {
		_ = s.updateAndPublish(ctx, session)
		return CallbackResult{
			Provider: domainsettings.ProviderIyzico.String(), Reference: payload.Reference, Status: "failure",
		}, err
	}
	if err := s.updateAndPublish(ctx, session); err != nil {
		return CallbackResult{}, err
	}
	authResp, err := s.CompleteIyzico3DS(ctx, session.Reference())
	if err != nil {
		return CallbackResult{
			Provider: domainsettings.ProviderIyzico.String(), Reference: payload.Reference, Status: "failure",
		}, err
	}
	return CallbackResult{
		Provider:  domainsettings.ProviderIyzico.String(),
		Reference: payload.Reference,
		Status:    "success",
		PaymentID: authResp.PaymentID,
	}, nil
}

// HandleMokaCallback, Moka redirect callback'ini işler.
func (s *ThreeDSService) HandleMokaCallback(ctx context.Context, payload MokaCallbackPayload) (CallbackResult, error) {
	session, err := s.repo.FindByReference(ctx, payload.OtherTrxCode)
	if err != nil {
		return CallbackResult{}, err
	}
	if res, ok := callbackResultFromPayment(session); ok {
		return res, nil
	}
	if session.Provider() != domainsettings.ProviderMoka.String() {
		return CallbackResult{}, domainpayment.ErrInvalidCallback
	}
	success, valid := moka.VerifyHashValue(payload.HashValue, session.ConversationData())
	if err := session.ApplyMokaCallback(success, valid, payload.TrxCode, payload.ResultCode, payload.ResultMessage); err != nil {
		_ = s.updateAndPublish(ctx, session)
		return CallbackResult{
			Provider: domainsettings.ProviderMoka.String(), Reference: payload.OtherTrxCode,
			Status: "failure", ResultCode: payload.ResultCode, ResultMessage: payload.ResultMessage,
		}, err
	}
	if err := s.updateAndPublish(ctx, session); err != nil {
		return CallbackResult{}, err
	}
	return CallbackResult{
		Provider:      domainsettings.ProviderMoka.String(),
		Reference:     payload.OtherTrxCode,
		Status:        "success",
		PaymentID:     payload.TrxCode,
		ResultCode:    payload.ResultCode,
		ResultMessage: payload.ResultMessage,
	}, nil
}

// CompleteIyzico3DS, iyzico ödemesi için auth çağrısı yapar.
func (s *ThreeDSService) CompleteIyzico3DS(ctx context.Context, reference string) (iyzico.Auth3DSResponse, error) {
	return s.completeIyzico3DS(ctx, reference, false)
}

func (s *ThreeDSService) completeIyzico3DS(ctx context.Context, reference string, reconciled bool) (iyzico.Auth3DSResponse, error) {
	if err := s.ensureProvider(ctx, domainsettings.ProviderIyzico); err != nil {
		return iyzico.Auth3DSResponse{}, err
	}
	session, err := s.repo.FindByReference(ctx, reference)
	if err != nil {
		return iyzico.Auth3DSResponse{}, err
	}
	if session.Provider() != domainsettings.ProviderIyzico.String() {
		return iyzico.Auth3DSResponse{}, domainpayment.ErrInvalidCallback
	}
	if session.Stage() != domainpayment.StageCallbackOK {
		return iyzico.Auth3DSResponse{}, fmt.Errorf("%w: ödeme callback bekliyor", domainpayment.ErrInvalidCallback)
	}
	resp, err := s.iyzico.Auth3DS(ctx, iyzico.Auth3DSRequest{
		PaymentID: session.ProviderPaymentID(), ConversationData: session.ConversationData(),
	})
	if err != nil {
		session.MarkFailed()
		_ = s.updateAndPublish(ctx, session)
		return iyzico.Auth3DSResponse{}, err
	}
	session.ApplyIyzicoAuth(
		resp.PaymentID, resp.AuthCode, domainpayment.FormatPaidPrice(resp.PaidPrice),
		resp.Installment, resp.BinNumber, resp.LastFourDigits, resp.CardAssociation,
	)
	if reconciled {
		session.EmitReconciled()
	}
	if err := s.updateAndPublish(ctx, session); err != nil {
		return resp, err
	}
	return resp, nil
}

// Complete3DS, aktif sağlayıcıya göre tamamlama yapar.
func (s *ThreeDSService) Complete3DS(ctx context.Context, reference string) (Complete3DSResult, error) {
	session, err := s.repo.FindByReference(ctx, reference)
	if err != nil {
		return Complete3DSResult{}, err
	}
	if session.Status() == domainpayment.StatusSuccess {
		return completeResultFromPayment(session), nil
	}
	if session.Status() == domainpayment.StatusFailed {
		return Complete3DSResult{}, domainpayment.ErrThreeDSFailed
	}
	switch session.Provider() {
	case domainsettings.ProviderIyzico.String():
		resp, err := s.CompleteIyzico3DS(ctx, reference)
		if err != nil {
			return Complete3DSResult{}, err
		}
		return Complete3DSResult{
			Provider: domainsettings.ProviderIyzico.String(), Reference: reference,
			Status: "success", PaymentID: resp.PaymentID, AuthCode: resp.AuthCode, Iyzico: &resp,
		}, nil
	case domainsettings.ProviderMoka.String():
		if session.Status() != domainpayment.StatusSuccess {
			return Complete3DSResult{}, fmt.Errorf("%w: Moka ödemesi henüz tamamlanmadı", domainpayment.ErrInvalidCallback)
		}
		return Complete3DSResult{
			Provider: domainsettings.ProviderMoka.String(), Reference: reference,
			Status: "success", PaymentID: session.ProviderPaymentID(), ResultCode: session.ResultCode(),
			ResultMessage: session.ResultMessage(),
		}, nil
	default:
		return Complete3DSResult{}, domainpayment.ErrProviderNotActive
	}
}

// Start3DSView, web 3DS başlatma sayfası için ödeme yükünü döner.
func (s *ThreeDSService) Start3DSView(ctx context.Context, reference string) (Initialize3DSResult, error) {
	session, err := s.repo.FindByReference(ctx, reference)
	if err != nil {
		return Initialize3DSResult{}, err
	}
	switch session.Provider() {
	case domainsettings.ProviderIyzico.String():
		return Initialize3DSResult{
			Provider: session.Provider(), PaymentID: session.ID(), Reference: session.Reference(),
			ThreeDSHtmlContent: session.InitPayload(),
		}, nil
	case domainsettings.ProviderMoka.String():
		return Initialize3DSResult{
			Provider: session.Provider(), PaymentID: session.ID(), Reference: session.Reference(),
			RedirectURL: session.InitPayload(), CodeForHash: session.ConversationData(),
		}, nil
	default:
		return Initialize3DSResult{}, domainpayment.ErrProviderNotActive
	}
}

func resolvePaymentReference(explicit string, ctx context.Context) string {
	if r := strings.TrimSpace(explicit); r != "" {
		return r
	}
	if k := appidempotency.KeyFromContext(ctx); k != "" {
		return k
	}
	return uuid.NewString()
}

func initResultFromPayment(p *domainpayment.Payment) (Initialize3DSResult, bool) {
	if p.Status() != domainpayment.StatusPending || p.Stage() != domainpayment.StageInitialized {
		return Initialize3DSResult{}, false
	}
	switch p.Provider() {
	case domainsettings.ProviderIyzico.String():
		return Initialize3DSResult{
			Provider: p.Provider(), PaymentID: p.ID(), Reference: p.Reference(),
			ThreeDSHtmlContent: p.InitPayload(),
		}, true
	case domainsettings.ProviderMoka.String():
		return Initialize3DSResult{
			Provider: p.Provider(), PaymentID: p.ID(), Reference: p.Reference(),
			RedirectURL: p.InitPayload(), CodeForHash: p.ConversationData(),
		}, true
	default:
		return Initialize3DSResult{}, false
	}
}

func callbackResultFromPayment(p *domainpayment.Payment) (CallbackResult, bool) {
	switch p.Status() {
	case domainpayment.StatusSuccess:
		return CallbackResult{
			Provider: p.Provider(), Reference: p.Reference(), Status: "success",
			PaymentID: p.ProviderPaymentID(), ResultCode: p.ResultCode(), ResultMessage: p.ResultMessage(),
		}, true
	case domainpayment.StatusFailed:
		return CallbackResult{
			Provider: p.Provider(), Reference: p.Reference(), Status: "failure",
			ResultCode: p.ResultCode(), ResultMessage: p.ResultMessage(),
		}, true
	default:
		return CallbackResult{}, false
	}
}

func completeResultFromPayment(session *domainpayment.Payment) Complete3DSResult {
	switch session.Provider() {
	case domainsettings.ProviderIyzico.String():
		return Complete3DSResult{
			Provider: session.Provider(), Reference: session.Reference(),
			Status: "success", PaymentID: session.ProviderPaymentID(), AuthCode: session.AuthCode(),
		}
	default:
		return Complete3DSResult{
			Provider: session.Provider(), Reference: session.Reference(),
			Status: "success", PaymentID: session.ProviderPaymentID(),
			ResultCode: session.ResultCode(), ResultMessage: session.ResultMessage(),
		}
	}
}
