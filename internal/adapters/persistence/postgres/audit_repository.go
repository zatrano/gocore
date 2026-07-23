package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	appaudit "github.com/zatrano/gocore/internal/application/audit"
	appshared "github.com/zatrano/gocore/internal/application/shared"
	"github.com/zatrano/gocore/internal/infrastructure/database"
	"github.com/zatrano/gocore/pkg/pagination"
)

// AuditRepository, appshared.AuditLogger portunun PostgreSQL implementasyonudur.
type AuditRepository struct {
	tx *database.TxManager
}

// NewAuditRepository, repository'yi kurar.
func NewAuditRepository(tx *database.TxManager) *AuditRepository {
	return &AuditRepository{tx: tx}
}

// Log, bir denetim girdisini kalıcılaştırır. Aynı event_id varsa no-op (idempotent).
func (r *AuditRepository) Log(ctx context.Context, e appshared.AuditEntry) error {
	id := uuid.New()
	var eventID any
	if e.EventID != "" {
		if parsed, err := uuid.Parse(e.EventID); err == nil {
			eventID = parsed
			id = parsed // event_id ile aynı PK tercih edilir
		}
	}
	occurredAt := e.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	actorType := e.ActorType
	if actorType == "" {
		if e.ActorID == "" {
			actorType = appshared.ActorTypeAnonymous
		} else {
			actorType = appshared.ActorTypeUser
		}
	}

	meta := appaudit.RedactMetadata(e.Metadata)
	metaBytes := []byte("{}")
	if len(meta) > 0 {
		if b, err := json.Marshal(meta); err == nil {
			metaBytes = b
		}
	}

	var actorID any
	if e.ActorID != "" {
		if parsed, err := uuid.Parse(e.ActorID); err == nil {
			actorID = parsed
		}
	}
	var ip any
	if e.IP != "" {
		if addr, err := netip.ParseAddr(e.IP); err == nil {
			ip = addr
		}
	}

	_, err := r.tx.DB(ctx).Exec(ctx, `
		INSERT INTO audit_logs (
			id, event_id, actor_id, actor_type, actor_email, action, resource, resource_id,
			ip, user_agent, metadata, source, correlation_id, occurred_at, created_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,now()
		)
		ON CONFLICT DO NOTHING
	`, id, eventID, actorID, actorType, e.ActorEmail, e.Action, e.Resource,
		nullIfEmpty(e.ResourceID), ip, nullIfEmpty(e.UserAgent), metaBytes,
		e.Source, e.CorrelationID, occurredAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil
		}
		return err
	}
	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

const auditListCols = `
	a.id, a.actor_id, COALESCE(NULLIF(a.actor_email, ''), COALESCE(u.email, '')) AS actor_email,
	a.action, a.resource, a.resource_id,
	a.ip::text, a.user_agent, a.metadata, a.created_at,
	COALESCE(a.actor_type, 'user'), COALESCE(a.source, ''), COALESCE(a.correlation_id, ''),
	COALESCE(a.occurred_at, a.created_at)
`

// List, denetim kayıtlarını filtreler ve offset sayfalama ile döner.
func (r *AuditRepository) List(
	ctx context.Context, filter appaudit.ListFilter, page pagination.Request,
) (pagination.Page[appaudit.Log], error) {
	limit := pagination.NormalizeLimit(page.Limit)
	pageNum := page.Page
	if pageNum < 1 {
		pageNum = 1
	}

	where, args := auditListWhere(filter)
	order := "a.created_at DESC, a.id DESC"
	if page.Ascending {
		order = "a.created_at ASC, a.id ASC"
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)::bigint
		FROM audit_logs a
		LEFT JOIN users u ON u.id = a.actor_id
		%s
	`, where)
	var total int64
	if err := r.tx.DB(ctx).QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return pagination.Page[appaudit.Log]{}, err
	}

	listArgs := append(append([]any{}, args...), limit, pagination.Offset(pageNum, limit))
	listQuery := fmt.Sprintf(`
		SELECT %s
		FROM audit_logs a
		LEFT JOIN users u ON u.id = a.actor_id
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, auditListCols, where, order, len(args)+1, len(args)+2)

	rows, err := r.tx.DB(ctx).Query(ctx, listQuery, listArgs...)
	if err != nil {
		return pagination.Page[appaudit.Log]{}, err
	}
	defer rows.Close()

	var items []appaudit.Log
	for rows.Next() {
		item, err := scanAuditLog(rows)
		if err != nil {
			return pagination.Page[appaudit.Log]{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return pagination.Page[appaudit.Log]{}, err
	}
	return pagination.NewPage(items, pageNum, limit, total), nil
}

func auditListWhere(filter appaudit.ListFilter) (string, []any) {
	var (
		args  []any
		where strings.Builder
	)
	where.WriteString("WHERE 1=1")
	if action := strings.TrimSpace(filter.Action); action != "" {
		args = append(args, "%"+action+"%")
		where.WriteString(fmt.Sprintf(" AND a.action ILIKE $%d", len(args)))
	}
	if resource := strings.TrimSpace(filter.Resource); resource != "" {
		args = append(args, "%"+resource+"%")
		where.WriteString(fmt.Sprintf(" AND a.resource ILIKE $%d", len(args)))
	}
	if actor := strings.TrimSpace(filter.Actor); actor != "" {
		if id, err := uuid.Parse(actor); err == nil {
			args = append(args, id)
			where.WriteString(fmt.Sprintf(" AND a.actor_id = $%d", len(args)))
		} else {
			args = append(args, "%"+actor+"%")
			where.WriteString(fmt.Sprintf(" AND (u.email ILIKE $%d OR a.actor_email ILIKE $%d)", len(args), len(args)))
		}
	}
	return where.String(), args
}

// FindByID, tek denetim kaydını getirir.
func (r *AuditRepository) FindByID(ctx context.Context, id string) (appaudit.Log, error) {
	logID, err := uuid.Parse(id)
	if err != nil {
		return appaudit.Log{}, appaudit.ErrNotFound
	}
	row := r.tx.DB(ctx).QueryRow(ctx, `
		SELECT `+auditListCols+`
		FROM audit_logs a
		LEFT JOIN users u ON u.id = a.actor_id
		WHERE a.id = $1
	`, logID)
	log, err := scanAuditLog(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return appaudit.Log{}, appaudit.ErrNotFound
		}
		return appaudit.Log{}, err
	}
	return log, nil
}

func scanAuditLog(row pgx.Row) (appaudit.Log, error) {
	var (
		id            uuid.UUID
		actorID       *uuid.UUID
		actorEmail    string
		action        string
		resource      string
		resourceID    *string
		ip            *string
		userAgent     *string
		metadata      []byte
		createdAt     time.Time
		actorType     string
		source        string
		correlationID string
		occurredAt    time.Time
	)
	if err := row.Scan(
		&id, &actorID, &actorEmail, &action, &resource, &resourceID,
		&ip, &userAgent, &metadata, &createdAt,
		&actorType, &source, &correlationID, &occurredAt,
	); err != nil {
		return appaudit.Log{}, err
	}

	log := appaudit.Log{
		ID: id.String(), Action: action, Resource: resource,
		ActorEmail: actorEmail, CreatedAt: createdAt,
		ActorType: actorType, Source: source, CorrelationID: correlationID,
		OccurredAt: occurredAt,
	}
	if actorID != nil {
		log.ActorID = actorID.String()
	}
	if resourceID != nil {
		log.ResourceID = *resourceID
	}
	if ip != nil {
		log.IP = *ip
	}
	if userAgent != nil {
		log.UserAgent = *userAgent
	}
	if len(metadata) > 0 {
		_ = json.Unmarshal(metadata, &log.Metadata)
	}
	return log, nil
}

var (
	_ appshared.AuditLogger = (*AuditRepository)(nil)
	_ appaudit.Repository   = (*AuditRepository)(nil)
)
