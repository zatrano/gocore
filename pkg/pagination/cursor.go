package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidCursor, çözülemeyen veya bozuk keyset imlecini bildirir.
var ErrInvalidCursor = errors.New("pagination: invalid cursor")

// cursorPayload, keyset imlecinin JSON gövdesidir (base64url ile kodlanır).
type cursorPayload struct {
	T string `json:"t"` // RFC3339Nano
	I string `json:"i"` // kayıt kimliği (UUID)
}

// EncodeCursor, (createdAt, id) çiftinden opak base64url imleci üretir.
func EncodeCursor(createdAt time.Time, id string) string {
	if createdAt.IsZero() || id == "" {
		return ""
	}
	raw, err := json.Marshal(cursorPayload{
		T: createdAt.UTC().Format(time.RFC3339Nano),
		I: id,
	})
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

// DecodeCursor, imleci (createdAt, id) çiftine çözer.
func DecodeCursor(s string) (createdAt time.Time, id string, err error) {
	if s == "" {
		return time.Time{}, "", ErrInvalidCursor
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}
	var p cursorPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}
	createdAt, err = time.Parse(time.RFC3339Nano, p.T)
	if err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}
	if _, err := uuid.Parse(p.I); err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}
	return createdAt, p.I, nil
}

// EncodeNextCursor, tam dolu sayfada son kayıttan sonraki imleci üretir.
func EncodeNextCursor[T any](items []T, limit int, key func(T) (time.Time, string)) string {
	if limit <= 0 || len(items) < limit {
		return ""
	}
	last := items[len(items)-1]
	at, id := key(last)
	return EncodeCursor(at, id)
}
