// Package security, güvenlikle ilgili altyapı implementasyonlarını içerir:
// Argon2id şifre hash'leme, JWT üretimi ve brute-force koruması.
package security

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2Params, Argon2id hash parametreleridir. OWASP önerileri baz alınmıştır;
// donanıma göre ayarlanabilir.
type Argon2Params struct {
	Memory      uint32 // KiB cinsinden
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultArgon2Params, üretim için makul varsayılanlar (OWASP 2024+).
func DefaultArgon2Params() Argon2Params {
	return Argon2Params{
		Memory:      64 * 1024, // 64 MB
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// Argon2Hasher, appshared.PasswordHasher portunun Argon2id implementasyonudur.
type Argon2Hasher struct {
	params Argon2Params
}

// NewArgon2Hasher, verilen parametrelerle hasher üretir.
func NewArgon2Hasher(p Argon2Params) *Argon2Hasher { return &Argon2Hasher{params: p} }

var (
	errInvalidHash         = errors.New("security: geçersiz hash formatı")
	errIncompatibleVersion = errors.New("security: uyumsuz argon2 sürümü")
)

// Hash, düz metin şifreyi PHC ($argon2id$...) formatında hash'ler.
func (h *Argon2Hasher) Hash(_ context.Context, plain string) (string, error) {
	salt := make([]byte, h.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("security: salt üretilemedi: %w", err)
	}

	key := argon2.IDKey(
		[]byte(plain), salt,
		h.params.Iterations, h.params.Memory, h.params.Parallelism, h.params.KeyLength,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Key := base64.RawStdEncoding.EncodeToString(key)

	// PHC string formatı.
	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.params.Memory, h.params.Iterations, h.params.Parallelism,
		b64Salt, b64Key,
	)
	return encoded, nil
}

// Verify, düz metin şifreyi PHC formatlı hash ile sabit zamanlı karşılaştırır.
func (h *Argon2Hasher) Verify(_ context.Context, plain, encoded string) (bool, error) {
	p, salt, key, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}

	other := argon2.IDKey([]byte(plain), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)

	// subtle.ConstantTimeCompare, timing attack'lara karşı koruma sağlar.
	if subtle.ConstantTimeCompare(key, other) == 1 {
		return true, nil
	}
	return false, nil
}

// NeedsRehash, mevcut hash parametreleri güncel politikadan zayıfsa true döner.
func (h *Argon2Hasher) NeedsRehash(encoded string) bool {
	p, _, _, err := decodeHash(encoded)
	if err != nil {
		return true
	}
	return p.Memory < h.params.Memory ||
		p.Iterations < h.params.Iterations ||
		p.Parallelism < h.params.Parallelism
}

// decodeHash, PHC formatlı hash'i bileşenlerine ayırır.
func decodeHash(encoded string) (Argon2Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return Argon2Params{}, nil, nil, errInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return Argon2Params{}, nil, nil, errInvalidHash
	}
	if version != argon2.Version {
		return Argon2Params{}, nil, nil, errIncompatibleVersion
	}

	var p Argon2Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Iterations, &p.Parallelism); err != nil {
		return Argon2Params{}, nil, nil, errInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Argon2Params{}, nil, nil, errInvalidHash
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return Argon2Params{}, nil, nil, errInvalidHash
	}

	p.SaltLength, err = u32Len(salt)
	if err != nil {
		return Argon2Params{}, nil, nil, err
	}
	p.KeyLength, err = u32Len(key)
	if err != nil {
		return Argon2Params{}, nil, nil, err
	}
	return p, salt, key, nil
}

func u32Len(b []byte) (uint32, error) {
	if len(b) > math.MaxUint32 {
		return 0, errInvalidHash
	}
	return uint32(len(b)), nil // #nosec G115 -- len(b) <= math.MaxUint32 kontrolü yukarıda.
}
