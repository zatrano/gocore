package user

import (
	"strings"

	"github.com/google/uuid"

	domainrbac "github.com/zatrano/gocore/internal/domain/rbac"
	"github.com/zatrano/gocore/pkg/validation"
)

// ID, bir kullanıcının benzersiz kimliğini temsil eden value object'tir.
// Zero value geçersizdir; her zaman NewID veya ParseID ile üretilmelidir.
type ID struct {
	value uuid.UUID
}

// NewID, yeni rastgele bir kullanıcı kimliği üretir (UUIDv7 - zaman sıralı,
// index dostu). Keyset pagination ve DB index performancesı için tercih edilir.
func NewID() ID {
	v, err := uuid.NewV7()
	if err != nil {
		// NewV7 yalnızca entropi hatası verir; pratikte gerçekleşmez.
		v = uuid.New()
	}
	return ID{value: v}
}

// ParseID, string bir UUID'yi ID'ye çözer.
func ParseID(s string) (ID, error) {
	v, err := uuid.Parse(s)
	if err != nil {
		return ID{}, ErrInvalidID
	}
	return ID{value: v}, nil
}

// IDFromUUID, mevcut bir uuid.UUID'den ID üretir (repository katmanı için).
func IDFromUUID(v uuid.UUID) ID { return ID{value: v} }

func (id ID) UUID() uuid.UUID { return id.value }
func (id ID) String() string  { return id.value.String() }
func (id ID) IsZero() bool    { return id.value == uuid.Nil }

// Email, doğrulanmış bir e-posta adresini temsil eden value object'tir.
// Yalnızca NewEmail ile üretilebilir; bu da geçerliliği garanti eder.
type Email struct {
	value string
}

// NewEmail, ham girdiyi normalize eder (trim + lowercase) ve doğrular.
func NewEmail(raw string) (Email, error) {
	normalized, err := validation.NormalizeEmail(raw)
	if err != nil {
		return Email{}, ErrInvalidEmail
	}
	if normalized == "" {
		return Email{}, ErrEmailRequired
	}
	return Email{value: normalized}, nil
}

func (e Email) String() string { return e.value }

// Role, kullanıcının atanmış rol adını temsil eder. Tanım ve izinler
// rbac bounded context'inde yönetilir; user yalnızca atamayı tutar.
type Role = domainrbac.RoleName

const (
	RoleAdmin = domainrbac.RoleAdmin
	RoleUser  = domainrbac.RoleUser
)

// ParseRole, rol adının biçimini doğrular (üyelik değil). Geçerliyse Role döner.
func ParseRole(s string) (Role, error) {
	r, err := domainrbac.ParseRoleName(s)
	if err != nil {
		return "", ErrInvalidRole
	}
	return r, nil
}

// PreferredLocale, kullanıcının kalıcı dil tercihini temsil eder (ör. "tr", "en").
// Bildirim e-postaları ve arka plan işleri bu değeri kullanır.
type PreferredLocale struct {
	value string
}

// ParsePreferredLocale, ham girdiyi normalize eder ve BCP-47 temel formunu doğrular.
func ParsePreferredLocale(raw string) (PreferredLocale, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return PreferredLocale{}, ErrLocaleRequired
	}
	if len(normalized) < 2 || len(normalized) > 10 {
		return PreferredLocale{}, ErrInvalidLocale
	}
	for i, r := range normalized {
		if r == '-' {
			if i == 0 || i == len(normalized)-1 {
				return PreferredLocale{}, ErrInvalidLocale
			}
			continue
		}
		if r < 'a' || r > 'z' {
			return PreferredLocale{}, ErrInvalidLocale
		}
	}
	return PreferredLocale{value: normalized}, nil
}

func (l PreferredLocale) String() string { return l.value }

// Phone, opsiyonel E.164 telefon numarasını temsil eder. Boş değer geçerlidir
// (telefon verilmemiş); doluysa normalize edilip doğrulanır.
type Phone struct {
	value string
}

// NewPhone, ham girdiyi normalize eder. Boş girdi geçerli (sıfır) Phone döner.
func NewPhone(raw string) (Phone, error) {
	normalized, err := validation.NormalizePhone(raw)
	if err != nil {
		return Phone{}, ErrInvalidPhone
	}
	return Phone{value: normalized}, nil
}

func (p Phone) String() string { return p.value }
func (p Phone) IsZero() bool   { return p.value == "" }

// HashedPassword, argon2id ile hash'lenmiş şifreyi temsil eder. Domain katmanı
// asla düz metin şifre tutmaz; hash'leme application/infrastructure katmanında
// yapılır ve buraya yalnızca hash'lenmiş değer girer.
type HashedPassword struct {
	encoded string
}

// NewHashedPassword, önceden hash'lenmiş (PHC formatındaki) değeri sarmalar.
func NewHashedPassword(encoded string) (HashedPassword, error) {
	if !strings.HasPrefix(encoded, "$argon2id$") {
		return HashedPassword{}, ErrInvalidPasswordHash
	}
	return HashedPassword{encoded: encoded}, nil
}

func (p HashedPassword) Encoded() string { return p.encoded }
func (p HashedPassword) IsZero() bool    { return p.encoded == "" }
