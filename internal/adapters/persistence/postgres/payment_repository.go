package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domainpayment "github.com/zatrano/gocore/internal/domain/payment"
	"github.com/zatrano/gocore/internal/infrastructure/database"
	"github.com/zatrano/gocore/pkg/fieldenc"
	"github.com/zatrano/gocore/pkg/pagination"
)

// PaymentRepository, ödeme kayıtları için PostgreSQL implementasyonudur.
type PaymentRepository struct {
	tx     *database.TxManager
	fields *fieldenc.Cipher
}

// NewPaymentRepository, repository'yi kurar.
func NewPaymentRepository(tx *database.TxManager, fields *fieldenc.Cipher) *PaymentRepository {
	return &PaymentRepository{tx: tx, fields: fields}
}

const paymentCols = `
	id, reference, provider, status, stage,
	amount, paid_amount, currency, installment,
	buyer_name, buyer_surname, buyer_email, buyer_phone,
	card_holder, card_bin, card_last4, card_association,
	provider_payment_id, result_code, result_message, auth_code,
	conversation_data, init_payload,
	created_at, updated_at, completed_at
`

func (r *PaymentRepository) Save(ctx context.Context, p *domainpayment.Payment) error {
	args, err := r.paymentArgs(p)
	if err != nil {
		return err
	}
	_, err = r.tx.DB(ctx).Exec(ctx, `
		INSERT INTO payments (`+paymentCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26)
	`, args...)
	return err
}

func (r *PaymentRepository) FindByReference(ctx context.Context, reference string) (*domainpayment.Payment, error) {
	row := r.tx.DB(ctx).QueryRow(ctx, `
		SELECT `+paymentCols+`
		FROM payments
		WHERE reference = $1
	`, reference)
	return r.scanPayment(row)
}

func (r *PaymentRepository) Update(ctx context.Context, p *domainpayment.Payment) error {
	cardHolder, cardBin, cardLast4, err := r.protectCard(p.CardHolder(), p.CardBin(), p.CardLast4())
	if err != nil {
		return err
	}
	_, err = r.tx.DB(ctx).Exec(ctx, `
		UPDATE payments SET
			status = $2, stage = $3,
			amount = $4, paid_amount = $5, currency = $6, installment = $7,
			buyer_name = $8, buyer_surname = $9, buyer_email = $10, buyer_phone = $11,
			card_holder = $12, card_bin = $13, card_last4 = $14, card_association = $15,
			provider_payment_id = $16, result_code = $17, result_message = $18, auth_code = $19,
			conversation_data = $20, init_payload = $21,
			updated_at = $22, completed_at = $23
		WHERE id = $1
	`, p.ID(), string(p.Status()), string(p.Stage()),
		p.Amount(), p.PaidAmount(), p.Currency(), p.Installment(),
		p.BuyerName(), p.BuyerSurname(), p.BuyerEmail(), p.BuyerPhone(),
		cardHolder, cardBin, cardLast4, p.CardAssociation(),
		p.ProviderPaymentID(), p.ResultCode(), p.ResultMessage(), p.AuthCode(),
		p.ConversationData(), p.InitPayload(),
		p.UpdatedAt(), p.CompletedAt())
	return err
}

func (r *PaymentRepository) List(
	ctx context.Context, filter domainpayment.ListFilter, page pagination.Request,
) (pagination.Page[*domainpayment.Payment], error) {
	limit := pagination.NormalizeLimit(page.Limit)
	pageNum := page.Page
	if pageNum < 1 {
		pageNum = 1
	}

	where, args := paymentListWhere(filter)
	order := "created_at DESC, id DESC"
	if page.Ascending {
		order = "created_at ASC, id ASC"
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*)::bigint FROM payments %s`, where)
	var total int64
	if err := r.tx.DB(ctx).QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return pagination.Page[*domainpayment.Payment]{}, err
	}

	listArgs := append(append([]any{}, args...), limit, pagination.Offset(pageNum, limit))
	query := fmt.Sprintf(`
		SELECT %s FROM payments %s ORDER BY %s LIMIT $%d OFFSET $%d
	`, paymentCols, where, order, len(args)+1, len(args)+2)

	rows, err := r.tx.DB(ctx).Query(ctx, query, listArgs...)
	if err != nil {
		return pagination.Page[*domainpayment.Payment]{}, err
	}
	defer rows.Close()

	var items []*domainpayment.Payment
	for rows.Next() {
		item, err := r.scanPayment(rows)
		if err != nil {
			return pagination.Page[*domainpayment.Payment]{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return pagination.Page[*domainpayment.Payment]{}, err
	}
	return pagination.NewPage(items, pageNum, limit, total), nil
}

func paymentListWhere(filter domainpayment.ListFilter) (string, []any) {
	var (
		args  []any
		where strings.Builder
	)
	where.WriteString("WHERE 1=1")
	if st := strings.TrimSpace(filter.Status); st != "" {
		args = append(args, st)
		where.WriteString(fmt.Sprintf(" AND status = $%d", len(args)))
	}
	if p := strings.TrimSpace(filter.Provider); p != "" {
		args = append(args, p)
		where.WriteString(fmt.Sprintf(" AND provider = $%d", len(args)))
	}
	return where.String(), args
}

func (r *PaymentRepository) ListReconcileCandidates(ctx context.Context, minAge time.Duration, limit int) ([]*domainpayment.Payment, error) {
	if limit <= 0 {
		limit = 50
	}
	if minAge <= 0 {
		minAge = 5 * time.Minute
	}
	cutoff := time.Now().UTC().Add(-minAge)
	rows, err := r.tx.DB(ctx).Query(ctx, `
		SELECT `+paymentCols+`
		FROM payments
		WHERE status = $1
		  AND updated_at < $2
		  AND (
		    (provider = 'iyzico' AND stage = $3)
		    OR (provider = 'iyzico' AND provider_payment_id <> '' AND stage NOT IN ($4, $5))
		  )
		ORDER BY updated_at ASC
		LIMIT $6
	`, string(domainpayment.StatusPending), cutoff,
		string(domainpayment.StageCallbackOK),
		string(domainpayment.StageFailed), string(domainpayment.StageCompleted),
		limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*domainpayment.Payment
	for rows.Next() {
		item, err := r.scanPayment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PaymentRepository) paymentArgs(p *domainpayment.Payment) ([]any, error) {
	cardHolder, cardBin, cardLast4, err := r.protectCard(p.CardHolder(), p.CardBin(), p.CardLast4())
	if err != nil {
		return nil, err
	}
	return []any{
		p.ID(), p.Reference(), p.Provider(), string(p.Status()), string(p.Stage()),
		p.Amount(), p.PaidAmount(), p.Currency(), p.Installment(),
		p.BuyerName(), p.BuyerSurname(), p.BuyerEmail(), p.BuyerPhone(),
		cardHolder, cardBin, cardLast4, p.CardAssociation(),
		p.ProviderPaymentID(), p.ResultCode(), p.ResultMessage(), p.AuthCode(),
		p.ConversationData(), p.InitPayload(),
		p.CreatedAt(), p.UpdatedAt(), p.CompletedAt(),
	}, nil
}

func (r *PaymentRepository) protectCard(holder, bin, last4 string) (string, string, string, error) {
	h, err := r.fields.Encrypt(holder)
	if err != nil {
		return "", "", "", err
	}
	b, err := r.fields.Encrypt(bin)
	if err != nil {
		return "", "", "", err
	}
	l, err := r.fields.Encrypt(last4)
	if err != nil {
		return "", "", "", err
	}
	return h, b, l, nil
}

func (r *PaymentRepository) revealCard(holder, bin, last4 string) (string, string, string, error) {
	h, err := r.fields.Decrypt(holder)
	if err != nil {
		return "", "", "", err
	}
	b, err := r.fields.Decrypt(bin)
	if err != nil {
		return "", "", "", err
	}
	l, err := r.fields.Decrypt(last4)
	if err != nil {
		return "", "", "", err
	}
	return h, b, l, nil
}

func (r *PaymentRepository) scanPayment(row pgx.Row) (*domainpayment.Payment, error) {
	var (
		id, reference, provider, status, stage                 string
		amount, paidAmount, currency                           string
		installment                                            int
		buyerName, buyerSurname, buyerEmail, buyerPhone        string
		cardHolder, cardBin, cardLast4, cardAssociation        string
		providerPaymentID, resultCode, resultMessage, authCode string
		conversationData, initPayload                          string
		createdAt, updatedAt                                   time.Time
		completedAt                                            *time.Time
	)
	err := row.Scan(
		&id, &reference, &provider, &status, &stage,
		&amount, &paidAmount, &currency, &installment,
		&buyerName, &buyerSurname, &buyerEmail, &buyerPhone,
		&cardHolder, &cardBin, &cardLast4, &cardAssociation,
		&providerPaymentID, &resultCode, &resultMessage, &authCode,
		&conversationData, &initPayload,
		&createdAt, &updatedAt, &completedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainpayment.ErrPaymentNotFound
		}
		return nil, err
	}
	cardHolder, cardBin, cardLast4, err = r.revealCard(cardHolder, cardBin, cardLast4)
	if err != nil {
		return nil, err
	}
	return domainpayment.RehydratePayment(
		id, reference, provider, status, stage,
		amount, paidAmount, currency, installment,
		buyerName, buyerSurname, buyerEmail, buyerPhone,
		cardHolder, cardBin, cardLast4, cardAssociation,
		providerPaymentID, resultCode, resultMessage, authCode,
		conversationData, initPayload,
		createdAt, updatedAt, completedAt,
	), nil
}

// ParsePaymentUUID, ödeme kimliği doğrular.
func ParsePaymentUUID(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	id, err := uuid.Parse(raw)
	if err != nil {
		return "", domainpayment.ErrPaymentNotFound
	}
	return id.String(), nil
}

var _ domainpayment.Repository = (*PaymentRepository)(nil)
