package payment

import (
	"context"
	"log/slog"
	"strings"
	"time"

	domainpayment "github.com/zatrano/gocore/internal/domain/payment"
	domainsettings "github.com/zatrano/gocore/internal/domain/settings"
	"github.com/zatrano/gocore/internal/infrastructure/payment/iyzico"
)

const (
	defaultReconcileMinAge  = 5 * time.Minute
	defaultReconcileBatch   = 50
	defaultReconcileTimeout = 30 * time.Second
)

// ReconcileResult, tek reconciliation turunun özetidir.
type ReconcileResult struct {
	Scanned int `json:"scanned"`
	Updated int `json:"updated"`
	Failed  int `json:"failed"`
}

// ReconcileStale, bekleyen ödemeleri sağlayıcıyla hizalar (webhook/timeout emniyet ağı).
func (s *ThreeDSService) ReconcileStale(ctx context.Context, minAge time.Duration, batchSize int) (ReconcileResult, error) {
	if minAge <= 0 {
		minAge = defaultReconcileMinAge
	}
	if batchSize <= 0 {
		batchSize = defaultReconcileBatch
	}
	payments, err := s.repo.ListReconcileCandidates(ctx, minAge, batchSize)
	if err != nil {
		return ReconcileResult{}, err
	}
	result := ReconcileResult{Scanned: len(payments)}
	for _, p := range payments {
		itemCtx, cancel := context.WithTimeout(ctx, defaultReconcileTimeout)
		updated, recErr := s.reconcileOne(itemCtx, p)
		cancel()
		if recErr != nil {
			result.Failed++
			slog.Warn("payment reconcile failed",
				slog.String("reference", p.Reference()),
				slog.String("provider", p.Provider()),
				slog.Any("error", recErr),
			)
			continue
		}
		if updated {
			result.Updated++
		}
	}
	return result, nil
}

func (s *ThreeDSService) reconcileOne(ctx context.Context, p *domainpayment.Payment) (bool, error) {
	if p.Status() != domainpayment.StatusPending {
		return false, nil
	}
	switch p.Provider() {
	case domainsettings.ProviderIyzico.String():
		return s.reconcileIyzico(ctx, p)
	default:
		return false, nil
	}
}

func (s *ThreeDSService) reconcileIyzico(ctx context.Context, p *domainpayment.Payment) (bool, error) {
	if p.Stage() == domainpayment.StageCallbackOK {
		_, err := s.completeIyzico3DS(ctx, p.Reference(), true)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	if strings.TrimSpace(p.ProviderPaymentID()) == "" {
		return false, nil
	}
	detail, err := s.iyzico.PaymentDetail(ctx, iyzico.PaymentDetailRequest{
		Locale:                "tr",
		ConversationID:        p.Reference(),
		PaymentConversationID: p.Reference(),
		PaymentID:             p.ProviderPaymentID(),
	})
	if err != nil {
		return false, err
	}
	switch strings.ToUpper(strings.TrimSpace(detail.PaymentStatus)) {
	case "SUCCESS":
		if p.Status() == domainpayment.StatusSuccess {
			return false, nil
		}
		p.ApplyIyzicoAuth(
			detail.PaymentID, detail.AuthCode, domainpayment.FormatPaidPrice(detail.PaidPrice),
			detail.Installment, detail.BinNumber, detail.LastFourDigits, detail.CardAssociation,
		)
		p.EmitReconciled()
		if err := s.updateAndPublish(ctx, p); err != nil {
			return false, err
		}
		return true, nil
	case "FAILURE":
		if p.Status() == domainpayment.StatusFailed {
			return false, nil
		}
		p.MarkFailed()
		p.EmitReconciled()
		if err := s.updateAndPublish(ctx, p); err != nil {
			return false, err
		}
		return true, nil
	default:
		return false, nil
	}
}
