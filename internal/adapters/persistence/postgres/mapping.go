package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zatrano/gocore/internal/adapters/persistence/postgres/db"
	"github.com/zatrano/gocore/internal/domain/user"
	"github.com/zatrano/gocore/pkg/fieldenc"
)

// toDomain, sqlc satırını domain aggregate'ine (yeniden) oluşturur.
// Veri DB'den geldiği için geçerli kabul edilir; value object kurucuları yine de
// bütünlüğü doğrular (savunmacı programlama).
func toDomain(row db.User, fields *fieldenc.Cipher) (*user.User, error) {
	email, err := user.NewEmail(row.Email)
	if err != nil {
		return nil, err
	}
	role, err := user.ParseRole(row.Role)
	if err != nil {
		return nil, err
	}
	pass, err := user.NewHashedPassword(row.PasswordHash)
	if err != nil {
		return nil, err
	}
	locale, err := user.ParsePreferredLocale(row.PreferredLocale)
	if err != nil {
		return nil, err
	}
	phone, err := user.NewPhone(row.Phone)
	if err != nil {
		return nil, err
	}
	mfaSecret, err := fields.Decrypt(row.MfaSecret)
	if err != nil {
		return nil, err
	}
	var deletedAt *time.Time
	if row.DeletedAt.Valid {
		t := row.DeletedAt.Time
		deletedAt = &t
	}

	return user.Hydrate(
		user.IDFromUUID(row.ID),
		email, phone, row.Name, pass, role, row.Active,
		row.EmailVerified, row.MfaEnabled, mfaSecret, locale,
		row.CreatedAt.Time, row.UpdatedAt.Time, deletedAt,
	), nil
}

// toDomainSlice, satır dilimini domain aggregate dilimine çevirir.
func toDomainSlice(rows []db.User, fields *fieldenc.Cipher) ([]*user.User, error) {
	users := make([]*user.User, 0, len(rows))
	for _, row := range rows {
		u, err := toDomain(row, fields)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// ts, time.Time'ı pgtype.Timestamptz'a çevirir.
func ts(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// optText, opsiyonel bir string filtresini pgtype.Text'e çevirir (nil → NULL).
func optText[T ~string](p *T) pgtype.Text {
	if p == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: string(*p), Valid: true}
}

// optBool, opsiyonel bir bool filtresini pgtype.Bool'a çevirir.
func optBool(p *bool) pgtype.Bool {
	if p == nil {
		return pgtype.Bool{}
	}
	return pgtype.Bool{Bool: *p, Valid: true}
}

// optSearch, arama terimini pgtype.Text'e çevirir (boş → NULL).
func optSearch(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

// optDeletedFilter, silinme kapsamını pgtype.Text'e çevirir.
func optDeletedFilter(s string) pgtype.Text {
	switch s {
	case "only", "all":
		return pgtype.Text{String: s, Valid: true}
	default:
		return pgtype.Text{}
	}
}
