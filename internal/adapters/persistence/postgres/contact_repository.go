package postgres

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/zatrano/gocore/internal/domain/contact"
	"github.com/zatrano/gocore/internal/infrastructure/database"
	"github.com/zatrano/gocore/pkg/pagination"
)

// ContactRepository, contact.Repository PostgreSQL implementasyonudur.
type ContactRepository struct {
	tx *database.TxManager
}

// NewContactRepository, repository'yi kurar.
func NewContactRepository(tx *database.TxManager) *ContactRepository {
	return &ContactRepository{tx: tx}
}

const contactSelectCols = `id, name, email, message, locale, ip, user_agent, status, created_at, read_at`

// Save, iletişim mesajını kaydeder.
func (r *ContactRepository) Save(ctx context.Context, m *contact.Message) error {
	var ip any
	if m.IP() != "" {
		if addr, err := netip.ParseAddr(m.IP()); err == nil {
			ip = addr
		}
	}
	_, err := r.tx.DB(ctx).Exec(ctx, `
		INSERT INTO contact_messages (id, name, email, message, locale, ip, user_agent, status, created_at, read_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, uuid.MustParse(m.ID().String()), m.Name(), m.Email().String(), m.Body(), m.Locale(),
		ip, m.UserAgent(), string(m.Status()), m.CreatedAt(), m.ReadAt())
	return err
}

// FindByID, mesajı kimliğe göre getirir.
func (r *ContactRepository) FindByID(ctx context.Context, id contact.ID) (*contact.Message, error) {
	row := r.tx.DB(ctx).QueryRow(ctx, `
		SELECT `+contactSelectCols+`
		FROM contact_messages WHERE id = $1
	`, uuid.MustParse(id.String()))
	return scanContactMessage(row)
}

// List, iletişim mesajlarını sayfalar; unreadOnly ise yalnızca okunmamışları döner.
func (r *ContactRepository) List(
	ctx context.Context, page pagination.Request, unreadOnly bool,
) (pagination.Page[*contact.Message], error) {
	limit := pagination.NormalizeLimit(page.Limit)
	pageNum := page.Page
	if pageNum < 1 {
		pageNum = 1
	}

	where := "WHERE 1=1"
	var args []any
	if unreadOnly {
		where += " AND read_at IS NULL"
	}
	order := "created_at DESC, id DESC"
	if page.Ascending {
		order = "created_at ASC, id ASC"
	}

	var total int64
	if err := r.tx.DB(ctx).QueryRow(ctx, `
		SELECT COUNT(*)::bigint FROM contact_messages `+where, args...).Scan(&total); err != nil {
		return pagination.Page[*contact.Message]{}, err
	}

	listArgs := append(append([]any{}, args...), limit, pagination.Offset(pageNum, limit))
	limIdx := len(args) + 1
	offIdx := len(args) + 2
	rows, err := r.tx.DB(ctx).Query(ctx, fmt.Sprintf(`
		SELECT %s FROM contact_messages
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, contactSelectCols, where, order, limIdx, offIdx), listArgs...)
	if err != nil {
		return pagination.Page[*contact.Message]{}, err
	}
	defer rows.Close()

	var items []*contact.Message
	for rows.Next() {
		msg, err := scanContactMessage(rows)
		if err != nil {
			return pagination.Page[*contact.Message]{}, err
		}
		items = append(items, msg)
	}
	if err := rows.Err(); err != nil {
		return pagination.Page[*contact.Message]{}, err
	}
	return pagination.NewPage(items, pageNum, limit, total), nil
}

// MarkRead, mesajı okundu olarak işaretler.
func (r *ContactRepository) MarkRead(ctx context.Context, id contact.ID) error {
	uid := uuid.MustParse(id.String())
	tag, err := r.tx.DB(ctx).Exec(ctx, `
		UPDATE contact_messages
		SET read_at = COALESCE(read_at, now())
		WHERE id = $1
	`, uid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return contact.ErrNotFound
	}
	return nil
}

type contactScanner interface {
	Scan(dest ...any) error
}

func scanContactMessage(row contactScanner) (*contact.Message, error) {
	var (
		rawID                                 uuid.UUID
		name, email, body, locale, ua, status string
		ip                                    *netip.Addr
		createdAt                             time.Time
		readAt                                *time.Time
	)
	err := row.Scan(&rawID, &name, &email, &body, &locale, &ip, &ua, &status, &createdAt, &readAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, contact.ErrNotFound
		}
		return nil, err
	}
	em, err := contact.NewEmail(email)
	if err != nil {
		return nil, err
	}
	parsed, err := contact.ParseID(rawID.String())
	if err != nil {
		return nil, err
	}
	ipStr := ""
	if ip != nil {
		ipStr = ip.String()
	}
	return contact.Reconstitute(parsed, name, em, body, locale, ipStr, ua, contact.Status(status), createdAt, readAt), nil
}

var _ contact.Repository = (*ContactRepository)(nil)
