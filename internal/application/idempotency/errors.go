package idempotency

import "github.com/zatrano/gocore/internal/domain/shared"

var (
	// ErrInProgress, aynı anahtarla işlem hâlâ devam ediyor.
	ErrInProgress = shared.NewDomainError(
		shared.KindConflict, "idempotency.in_progress", "işlem devam ediyor, lütfen tekrar deneyin")

	// ErrConflict, aynı anahtar farklı istek gövdesiyle kullanıldı.
	ErrConflict = shared.NewDomainError(
		shared.KindConflict, "idempotency.conflict", "idempotency anahtarı farklı istekle çakışıyor")

	// ErrDuplicateSubmit, form nonce zaten tüketildi (çift gönderim).
	ErrDuplicateSubmit = shared.NewDomainError(
		shared.KindConflict, "form.duplicate_submit", "bu form zaten gönderildi")
)
