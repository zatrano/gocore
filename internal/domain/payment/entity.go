package payment

import (
	"strconv"
	"strings"
	"time"

	"github.com/zatrano/gocore/internal/domain/shared"
)

// Payment, tüm sağlayıcılarda ortak ödeme kaydıdır (3DS tahsilat).
type Payment struct {
	shared.EventRecorder

	id                string
	reference         string
	provider          string
	status            PaymentStatus
	stage             PaymentStage
	amount            string
	paidAmount        string
	currency          string
	installment       int
	buyerName         string
	buyerSurname      string
	buyerEmail        string
	buyerPhone        string
	cardHolder        string
	cardBin           string
	cardLast4         string
	cardAssociation   string
	providerPaymentID string
	resultCode        string
	resultMessage     string
	authCode          string
	conversationData  string
	initPayload       string
	createdAt         time.Time
	updatedAt         time.Time
	completedAt       *time.Time
}

// NewPayment, yeni ödeme kaydı oluşturur.
func NewPayment(id, reference, provider, amount, currency string, installment int) *Payment {
	now := time.Now().UTC()
	return &Payment{
		id:          id,
		reference:   reference,
		provider:    provider,
		status:      StatusPending,
		stage:       StageInitialized,
		amount:      amount,
		currency:    currency,
		installment: installment,
		createdAt:   now,
		updatedAt:   now,
	}
}

// RehydratePayment, persistence'tan ödeme kaydını yeniden oluşturur.
func RehydratePayment(
	id, reference, provider, status, stage string,
	amount, paidAmount, currency string,
	installment int,
	buyerName, buyerSurname, buyerEmail, buyerPhone string,
	cardHolder, cardBin, cardLast4, cardAssociation string,
	providerPaymentID, resultCode, resultMessage, authCode string,
	conversationData, initPayload string,
	createdAt, updatedAt time.Time,
	completedAt *time.Time,
) *Payment {
	return &Payment{
		id: id, reference: reference, provider: provider,
		status: PaymentStatus(status), stage: PaymentStage(stage),
		amount: amount, paidAmount: paidAmount, currency: currency, installment: installment,
		buyerName: buyerName, buyerSurname: buyerSurname, buyerEmail: buyerEmail, buyerPhone: buyerPhone,
		cardHolder: cardHolder, cardBin: cardBin, cardLast4: cardLast4, cardAssociation: cardAssociation,
		providerPaymentID: providerPaymentID, resultCode: resultCode, resultMessage: resultMessage,
		authCode: authCode, conversationData: conversationData, initPayload: initPayload,
		createdAt: createdAt, updatedAt: updatedAt, completedAt: completedAt,
	}
}

func (p *Payment) ID() string                { return p.id }
func (p *Payment) Reference() string         { return p.reference }
func (p *Payment) Provider() string          { return p.provider }
func (p *Payment) Status() PaymentStatus     { return p.status }
func (p *Payment) Stage() PaymentStage       { return p.stage }
func (p *Payment) Amount() string            { return p.amount }
func (p *Payment) PaidAmount() string        { return p.paidAmount }
func (p *Payment) Currency() string          { return p.currency }
func (p *Payment) Installment() int          { return p.installment }
func (p *Payment) BuyerName() string         { return p.buyerName }
func (p *Payment) BuyerSurname() string      { return p.buyerSurname }
func (p *Payment) BuyerEmail() string        { return p.buyerEmail }
func (p *Payment) BuyerPhone() string        { return p.buyerPhone }
func (p *Payment) CardHolder() string        { return p.cardHolder }
func (p *Payment) CardBin() string           { return p.cardBin }
func (p *Payment) CardLast4() string         { return p.cardLast4 }
func (p *Payment) CardAssociation() string   { return p.cardAssociation }
func (p *Payment) ProviderPaymentID() string { return p.providerPaymentID }
func (p *Payment) ResultCode() string        { return p.resultCode }
func (p *Payment) ResultMessage() string     { return p.resultMessage }
func (p *Payment) AuthCode() string          { return p.authCode }
func (p *Payment) ConversationData() string  { return p.conversationData }
func (p *Payment) InitPayload() string       { return p.initPayload }
func (p *Payment) CreatedAt() time.Time      { return p.createdAt }
func (p *Payment) UpdatedAt() time.Time      { return p.updatedAt }
func (p *Payment) CompletedAt() *time.Time   { return p.completedAt }
func (p *Payment) CardDisplay() string       { return CardDisplay(p.cardBin, p.cardLast4) }

// SetBuyer, alıcı bilgilerini kaydeder.
func (p *Payment) SetBuyer(name, surname, email, phone string) {
	p.buyerName = strings.TrimSpace(name)
	p.buyerSurname = strings.TrimSpace(surname)
	p.buyerEmail = strings.TrimSpace(email)
	p.buyerPhone = strings.TrimSpace(phone)
	p.touch()
}

// SetCard, kart sahibi ve maskeli kart bilgisini kaydeder.
func (p *Payment) SetCard(holder, cardNumber string) {
	p.cardHolder = strings.TrimSpace(holder)
	p.cardBin, p.cardLast4 = CardMask(cardNumber)
	p.touch()
}

// SetPaidAmount, tahsil edilen tutarı yazar.
func (p *Payment) SetPaidAmount(amount string) {
	p.paidAmount = strings.TrimSpace(amount)
	p.touch()
}

// SetInitPayload, 3DS başlatma yanıtını saklar.
func (p *Payment) SetInitPayload(payload string) {
	p.initPayload = payload
	p.touch()
}

// EmitInitialized, 3DS başlatma olayını kaydeder.
func (p *Payment) EmitInitialized() {
	p.Record(NewThreeDSInitialized(p))
}

// EmitWebhookApplied, webhook ile güncelleme olayını kaydeder.
func (p *Payment) EmitWebhookApplied() {
	p.Record(NewThreeDSWebhookApplied(p))
}

// EmitReconciled, reconciliation olayını kaydeder.
func (p *Payment) EmitReconciled() {
	p.Record(NewThreeDSReconciled(p))
}

// SetCodeForHash, Moka init yanıtındaki CodeForHash değerini saklar.
func (p *Payment) SetCodeForHash(codeForHash string) {
	p.conversationData = strings.ToUpper(strings.TrimSpace(codeForHash))
	p.touch()
}

// ApplyIyzicoCallback, iyzico callback sonucunu uygular.
func (p *Payment) ApplyIyzicoCallback(status, paymentID, mdStatus, conversationData string) error {
	if status != "success" || mdStatus != "1" {
		p.markFailed(mdStatus, "")
		return ErrThreeDSFailed
	}
	p.providerPaymentID = paymentID
	p.resultCode = mdStatus
	p.conversationData = conversationData
	p.stage = StageCallbackOK
	p.status = StatusPending
	p.touch()
	return nil
}

// ApplyMokaCallback, Moka redirect callback sonucunu uygular.
func (p *Payment) ApplyMokaCallback(success bool, validHash bool, trxCode, resultCode, resultMessage string) error {
	if !validHash {
		p.markFailed("", "geçersiz callback imzası")
		return ErrInvalidCallback
	}
	if !success {
		p.markFailed(resultCode, resultMessage)
		return ErrThreeDSFailed
	}
	p.providerPaymentID = trxCode
	p.resultCode = resultCode
	p.resultMessage = strings.TrimSpace(resultMessage)
	p.markSuccess()
	return nil
}

// ApplyIyzicoAuth, iyzico auth yanıtıyla ödemeyi tamamlar.
func (p *Payment) ApplyIyzicoAuth(paymentID, authCode, paidPrice string, installment int, bin, last4, association string) {
	if paymentID != "" {
		p.providerPaymentID = paymentID
	}
	if authCode != "" {
		p.authCode = authCode
	}
	if paidPrice != "" {
		p.paidAmount = paidPrice
	}
	if installment > 0 {
		p.installment = installment
	}
	if bin != "" {
		p.cardBin = bin
	}
	if last4 != "" {
		p.cardLast4 = last4
	}
	if association != "" {
		p.cardAssociation = association
	}
	p.markSuccess()
}

// MarkSuccess, ödemeyi başarılı olarak kapatır.
func (p *Payment) MarkSuccess() { p.markSuccess() }

// MarkFailed, ödemeyi başarısız olarak işaretler.
func (p *Payment) MarkFailed() { p.markFailed("", "") }

// ApplyWebhookPaymentID, iyzico webhook paymentId değerini yazar.
func (p *Payment) ApplyWebhookPaymentID(paymentID string) {
	p.providerPaymentID = strings.TrimSpace(paymentID)
	p.touch()
}

// ApplyWebhookStatus, iyzico webhook durumunu uygular.
func (p *Payment) ApplyWebhookStatus(success bool) {
	if success {
		if p.status != StatusSuccess {
			p.markSuccess()
		}
		return
	}
	if p.status != StatusFailed {
		p.markFailed("", "")
	}
}

func (p *Payment) markSuccess() {
	p.status = StatusSuccess
	p.stage = StageCompleted
	now := time.Now().UTC()
	p.completedAt = &now
	p.updatedAt = now
	p.Record(NewThreeDSCompleted(p))
}

func (p *Payment) markFailed(code, message string) {
	p.status = StatusFailed
	p.stage = StageFailed
	if code != "" {
		p.resultCode = code
	}
	if message != "" {
		p.resultMessage = message
	}
	now := time.Now().UTC()
	p.completedAt = &now
	p.updatedAt = now
	p.Record(NewThreeDSFailed(p))
}

func (p *Payment) touch() {
	p.updatedAt = time.Now().UTC()
}

// FormatPaidPrice, float tutarı stringe çevirir.
func FormatPaidPrice(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
