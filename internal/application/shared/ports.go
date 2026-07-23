// Package shared, uygulama (use-case) katmanının tüm bounded context'lerde
// paylaştığı portları (arayüzleri) tanımlar. Bu arayüzlerin implementasyonları
// infrastructure/adapters katmanındadır (Dependency Inversion).
package shared

import (
	"context"

	"github.com/zatrano/gocore/internal/domain/shared"
)

// PasswordHasher, şifre hash'leme ve doğrulama portu. Implementasyon Argon2id
// kullanır (infrastructure/security).
type PasswordHasher interface {
	// Hash, düz metin şifreyi PHC formatlı hash'e çevirir.
	Hash(ctx context.Context, plain string) (string, error)
	// Verify, düz metin şifreyi hash ile karşılaştırır. Sabit zamanlı olmalıdır.
	Verify(ctx context.Context, plain, encoded string) (bool, error)
	// NeedsRehash, hash parametreleri güncel politikadan zayıfsa true döner.
	NeedsRehash(encoded string) bool
}

// EventPublisher, domain event'lerini yayınlama portu. Şimdilik in-memory/log
// implementasyonu var; ileride Kafka/RabbitMQ adaptörüyle değiştirilebilir.
type EventPublisher interface {
	Publish(ctx context.Context, events ...shared.DomainEvent) error
}

// TxManager, use-case'lerin birden fazla repository çağrısını tek bir atomik
// transaction içinde çalıştırmasını sağlar (Unit of Work). fn içindeki tüm
// repository işlemleri, context aracılığıyla aynı transaction'ı kullanır.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Clock, zamanı soyutlar; testlerde deterministik zaman sağlar.
type Clock interface {
	Now() int64 // unix nano
}

// SessionRevoker, kullanıcının tüm oturumlarını iptal eder (rol/şifre değişimi).
type SessionRevoker interface {
	RevokeAll(ctx context.Context, userID string) error
}
