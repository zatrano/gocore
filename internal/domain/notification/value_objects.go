package notification

import "github.com/google/uuid"

// ID, uygulama içi bir bildirimin benzersiz kimliğidir (value object).
type ID struct {
	value uuid.UUID
}

// NewID, yeni zaman-sıralı (UUIDv7) bir kimlik üretir.
func NewID() ID {
	v, err := uuid.NewV7()
	if err != nil {
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
