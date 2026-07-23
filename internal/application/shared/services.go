package shared

import (
	"context"
	"io"
	"time"
)

// Cache, anahtar-değer önbellek portudur. Implementasyon in-memory veya Redis
// olabilir; use-case'ler yalnızca bu arayüzü bilir (Storage abstraction).
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	// Take, değeri atomik olarak okuyup siler. Tek kullanımlık token/state
	// tüketimlerinde eşzamanlı tekrar kullanımını engeller.
	Take(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// FileObject, depolamaya yazılacak/okunacak dosya meta verisidir.
type FileObject struct {
	Key         string
	ContentType string
	Size        int64
}

// Storage, nesne/dosya depolama portudur. Yerel disk, S3, GCS gibi backend'ler
// bu arayüzü uygular (Storage abstraction).
type Storage interface {
	Put(ctx context.Context, key string, r io.Reader, contentType string, size int64) (FileObject, error)
	Get(ctx context.Context, key string) (io.ReadCloser, FileObject, error)
	Delete(ctx context.Context, key string) error
}

// Email, gönderilecek e-postayı temsil eder.
type Email struct {
	To       []string
	Subject  string
	HTMLBody string
	TextBody string
	From     string // boşsa mailer varsayılan From kullanır
	ReplyTo  string
}

// Mailer, e-posta gönderme portudur. SMTP, SES, SendGrid vb. ile uygulanabilir
// (Email abstraction).
type Mailer interface {
	Send(ctx context.Context, email Email) error
}

// AuditEntry, güvenlik/uyumluluk denetim kaydıdır (kim, ne, ne zaman, nerede).
type AuditEntry struct {
	EventID       string // domain event id; idempotent yazım için
	OccurredAt    time.Time
	ActorID       string // boş olabilir (anonim işlem)
	ActorType     string
	ActorEmail    string
	Action        string // ör. "user.login"
	Resource      string // ör. "user"
	ResourceID    string
	Source        string
	CorrelationID string
	IP            string
	UserAgent     string
	Metadata      map[string]any
}

// AuditLogger, denetim kaydı yazma portudur. Değiştirilemez denetim izi için
// kalıcı bir depoya (audit_logs tablosu) yazar.
type AuditLogger interface {
	Log(ctx context.Context, entry AuditEntry) error
}
