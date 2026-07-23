// Package config, uygulamanın tüm yapılandırmasını ortam değişkenlerinden
// tip güvenli şekilde yükler ve doğrular. Secrets yönetimi buradan geçer;
// hassas alanlar loglanırken maskelenir (String() metodları ile).
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

// Config, uygulamanın kök yapılandırmasıdır. Immutable kabul edilir:
// yükleme sonrası hiçbir alan değiştirilmez.
type Config struct {
	App      App      `envPrefix:"APP_"`
	HTTP     HTTP     `envPrefix:"HTTP_"`
	DB       DB       `envPrefix:"DB_"`
	Auth     Auth     `envPrefix:"AUTH_"`
	OAuth    OAuth    `envPrefix:"OAUTH_"`
	Security Security `envPrefix:"SEC_"`
	I18n     I18n     `envPrefix:"I18N_"`
	Notify   Notify   `envPrefix:"NOTIFY_"`
	SMTP     SMTP     `envPrefix:"SMTP_"`
	Contact  Contact  `envPrefix:"CONTACT_"`
	Payment  Payment  `envPrefix:"PAYMENT_"`
}

// App, genel uygulama meta verisi.
type App struct {
	Name        string `env:"NAME" envDefault:"enterprise" validate:"required"`
	Environment string `env:"ENV" envDefault:"development" validate:"required,oneof=development staging production"`
	Version     string `env:"VERSION" envDefault:"dev"`
	// ShutdownTimeout, graceful shutdown için maksimum bekleme süresi.
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"15s" validate:"required"`
}

// IsProduction, prod ortamında olup olmadığını döner.
func (a App) IsProduction() bool { return a.Environment == "production" }

// HTTP, HTTP sunucu yapılandırması.
type HTTP struct {
	Host            string        `env:"HOST" envDefault:"0.0.0.0"`
	Port            int           `env:"PORT" envDefault:"8080" validate:"required,min=1,max=65535"`
	ReadTimeout     time.Duration `env:"READ_TIMEOUT" envDefault:"10s" validate:"required"`
	WriteTimeout    time.Duration `env:"WRITE_TIMEOUT" envDefault:"15s" validate:"required"`
	IdleTimeout     time.Duration `env:"IDLE_TIMEOUT" envDefault:"60s" validate:"required"`
	BodyLimitBytes  int           `env:"BODY_LIMIT_BYTES" envDefault:"4194304" validate:"required,min=1024"` // 4MB
	CORSAllowOrigin string        `env:"CORS_ALLOW_ORIGIN" envDefault:"*"`
	// RequestTimeout, tek bir isteğin işlenmesi için context timeout'u.
	RequestTimeout time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s" validate:"required"`
}

// Addr, dinlenecek adresi döner.
func (h HTTP) Addr() string { return fmt.Sprintf("%s:%d", h.Host, h.Port) }

// DB, PostgreSQL bağlantı havuzu yapılandırması. pgxpool için optimize edilmiştir.
type DB struct {
	Host     string `env:"HOST" envDefault:"localhost" validate:"required"`
	Port     int    `env:"PORT" envDefault:"5432" validate:"required,min=1,max=65535"`
	User     string `env:"USER" envDefault:"postgres" validate:"required"`
	Password Secret `env:"PASSWORD" envDefault:"postgres"`
	Name     string `env:"NAME" envDefault:"enterprise" validate:"required"`
	SSLMode  string `env:"SSLMODE" envDefault:"disable" validate:"required,oneof=disable require verify-ca verify-full"`

	// Havuz optimizasyonu
	MaxConns        int32         `env:"MAX_CONNS" envDefault:"20" validate:"required,min=1"`
	MinConns        int32         `env:"MIN_CONNS" envDefault:"2" validate:"min=0"`
	MaxConnLifetime time.Duration `env:"MAX_CONN_LIFETIME" envDefault:"1h" validate:"required"`
	MaxConnIdleTime time.Duration `env:"MAX_CONN_IDLE_TIME" envDefault:"30m" validate:"required"`
	// QueryTimeout, tekil sorgular için varsayılan context timeout'u.
	QueryTimeout   time.Duration `env:"QUERY_TIMEOUT" envDefault:"5s" validate:"required"`
	ConnectTimeout time.Duration `env:"CONNECT_TIMEOUT" envDefault:"5s" validate:"required"`
}

// DSN, pgxpool için bağlantı dizesini üretir.
func (d DB) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password.Value(), d.Host, d.Port, d.Name, d.SSLMode,
	)
}

// Auth, kimlik doğrulama & JWT yapılandırması.
type Auth struct {
	JWTSecret        Secret        `env:"JWT_SECRET" envDefault:"change-me-in-production-please-32b"`
	MFAEncryptionKey Secret        `env:"MFA_ENCRYPTION_KEY" envDefault:""`
	JWTIssuer        string        `env:"JWT_ISSUER" envDefault:"enterprise"`
	JWTAudience      string        `env:"JWT_AUDIENCE" envDefault:"zatrano-api"`
	AccessTokenTTL   time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m" validate:"required"`
	RefreshTokenTTL  time.Duration `env:"REFRESH_TOKEN_TTL" envDefault:"168h" validate:"required"`
	// Brute-force koruması
	MaxLoginAttempts int           `env:"MAX_LOGIN_ATTEMPTS" envDefault:"5" validate:"required,min=1"`
	LockoutDuration  time.Duration `env:"LOCKOUT_DURATION" envDefault:"15m" validate:"required"`
	// Tek kullanımlık token ömürleri (e-posta doğrulama, şifre sıfırlama).
	VerifyTokenTTL time.Duration `env:"VERIFY_TOKEN_TTL" envDefault:"24h" validate:"required"`
	ResetTokenTTL  time.Duration `env:"RESET_TOKEN_TTL" envDefault:"1h" validate:"required"`
	// EmailLinkBaseURL, e-posta bağlantılarının (doğrulama/sıfırlama) kök adresi.
	EmailLinkBaseURL string `env:"EMAIL_LINK_BASE_URL" envDefault:"http://localhost:8080"`
}

// OAuth, OAuth/SSO sağlayıcı yapılandırmasıdır. Kimlik bilgisi boş olan
// sağlayıcı otomatik olarak devre dışı kalır.
type OAuth struct {
	// CallbackBaseURL, callback URL'lerinin kökü. Sağlayıcı callback'i:
	// {CallbackBaseURL}/{provider}/callback (ör. .../api/v1/auth/oauth).
	CallbackBaseURL string `env:"CALLBACK_BASE_URL" envDefault:"http://localhost:8080/api/v1/auth/oauth"`

	GoogleClientID     string `env:"GOOGLE_CLIENT_ID" envDefault:""`
	GoogleClientSecret Secret `env:"GOOGLE_CLIENT_SECRET" envDefault:""`

	GithubClientID     string `env:"GITHUB_CLIENT_ID" envDefault:""`
	GithubClientSecret Secret `env:"GITHUB_CLIENT_SECRET" envDefault:""`
}

// CallbackURL, verilen sağlayıcı için tam callback URL'ini üretir.
func (o OAuth) CallbackURL(provider string) string {
	return o.CallbackBaseURL + "/" + provider + "/callback"
}

// Security, güvenlik katmanı parametreleri.
type Security struct {
	// Rate limiting (IP başına)
	RateLimitMax    int           `env:"RATE_LIMIT_MAX" envDefault:"100" validate:"required,min=1"`
	RateLimitWindow time.Duration `env:"RATE_LIMIT_WINDOW" envDefault:"1m" validate:"required"`
	// Upload
	MaxUploadBytes    int64    `env:"MAX_UPLOAD_BYTES" envDefault:"10485760" validate:"required,min=1"` // 10MB
	AllowedUploadMIME []string `env:"ALLOWED_UPLOAD_MIME" envSeparator:"," envDefault:"image/jpeg,image/png,application/pdf,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/vnd.ms-excel"`
	// TrustedProxies, X-Forwarded-For'a güvenilecek proxy CIDR'leri.
	TrustedProxies []string `env:"TRUSTED_PROXIES" envSeparator:"," envDefault:""`
	// TurnstileSiteKey, Cloudflare Turnstile site anahtarı (public formlar).
	TurnstileSiteKey string `env:"TURNSTILE_SITE_KEY" envDefault:""`
	// TurnstileSecretKey, Cloudflare Turnstile secret anahtarı (siteverify).
	TurnstileSecretKey Secret `env:"TURNSTILE_SECRET_KEY" envDefault:""`
}

// TurnstileEnabled, Turnstile widget ve doğrulamasının aktif olup olmadığını döner.
func (s Security) TurnstileEnabled() bool {
	return s.TurnstileSiteKey != "" && s.TurnstileSecretKey.Value() != ""
}

// I18n, çoklu dil (uluslararasılaştırma) yapılandırması.
type I18n struct {
	// DefaultLocale, dil çözümlenemediğinde kullanılacak varsayılan dildir.
	DefaultLocale string `env:"DEFAULT_LOCALE" envDefault:"tr" validate:"required"`
	// Supported, desteklenen dillerin listesi. Her biri için pkg/i18n/locales
	// altında bir "<locale>.json" sözlüğü bulunmalıdır.
	Supported []string `env:"SUPPORTED" envSeparator:"," envDefault:"tr,en" validate:"required,min=1"`
}

// Notify, merkezi bildirim sistemi yapılandırmasıdır. Aktif SMS sağlayıcısı
// dashboard'dan seçilir; kimlik bilgileri ortam değişkenlerinden okunur.
type Notify struct {
	// SMSFrom, mesaj başlığı (Netgsm msgheader / İleti Merkezi sender).
	SMSFrom string `env:"SMS_FROM" envDefault:""`

	// Netgsm kimlik bilgileri ve gönderim parametreleri.
	NetgsmUser      string `env:"NETGSM_USER" envDefault:""`
	NetgsmPassword  Secret `env:"NETGSM_PASSWORD" envDefault:""`
	NetgsmAppName   string `env:"NETGSM_APPNAME" envDefault:"zatrano"`
	NetgsmEncoding  string `env:"NETGSM_ENCODING" envDefault:"TR" validate:"omitempty,oneof=UTF-8 TR UNICODE"`
	NetgsmIYSFilter string `env:"NETGSM_IYSFILTER" envDefault:"0" validate:"omitempty,oneof=0 11 12"`

	// İleti Merkezi kimlik bilgileri (GET API).
	IletimerkeziKey     string `env:"ILETIMERKEZI_KEY" envDefault:""`
	IletimerkeziHash    Secret `env:"ILETIMERKEZI_HASH" envDefault:""`
	IletimerkeziIYS     string `env:"ILETIMERKEZI_IYS" envDefault:"0" validate:"omitempty,oneof=0 1"`
	IletimerkeziIYSList string `env:"ILETIMERKEZI_IYS_LIST" envDefault:"BIREYSEL" validate:"omitempty,oneof=BIREYSEL TACIR"`
}

// SMTP, genel e-posta gönderim yapılandırmasıdır.
type SMTP struct {
	Host     string        `env:"HOST" envDefault:""`
	Port     int           `env:"PORT" envDefault:"587" validate:"omitempty,min=1,max=65535"`
	Username string        `env:"USERNAME" envDefault:""`
	Password Secret        `env:"PASSWORD" envDefault:""`
	From     string        `env:"FROM" envDefault:""`
	TLSMode  string        `env:"TLS_MODE" envDefault:"starttls" validate:"omitempty,oneof=none starttls tls"`
	Timeout  time.Duration `env:"TIMEOUT" envDefault:"15s"`
}

// Configured, SMTP host ve from alanlarının dolu olup olmadığını döner.
func (s SMTP) Configured() bool {
	return strings.TrimSpace(s.Host) != "" && strings.TrimSpace(s.From) != ""
}

// Contact, iletişim formu alıcı yapılandırmasıdır.
type Contact struct {
	// RecipientEmail, iletişim formundan gelen mesajların gönderileceği e-posta.
	RecipientEmail string `env:"RECIPIENT_EMAIL" envDefault:"" validate:"omitempty,email"`
}

// Payment, ödeme sağlayıcı kimlik bilgileridir. Aktif sağlayıcı dashboard'dan seçilir.
type Payment struct {
	IyzicoAPIKey       string `env:"IYZICO_API_KEY" envDefault:""`
	IyzicoSecretKey    Secret `env:"IYZICO_SECRET_KEY" envDefault:""`
	IyzicoBaseURL      string `env:"IYZICO_BASE_URL" envDefault:"https://sandbox-api.iyzipay.com"`
	CallbackURL        string `env:"CALLBACK_URL" envDefault:""`
	FieldEncryptionKey Secret `env:"FIELD_ENCRYPTION_KEY" envDefault:""`
	MokaDealerCode     string `env:"MOKA_DEALER_CODE" envDefault:""`
	MokaUsername       string `env:"MOKA_USERNAME" envDefault:""`
	MokaPassword       Secret `env:"MOKA_PASSWORD" envDefault:""`
	MokaBaseURL        string `env:"MOKA_BASE_URL" envDefault:"https://service.refmokaunited.com"`
	MokaSoftware       string `env:"MOKA_SOFTWARE" envDefault:"zatrano"`
}

// Load, ortamdan (.env dahil) yapılandırmayı yükler ve doğrular.
// .env dosyası opsiyoneldir; yoksa yalnızca gerçek ortam değişkenleri kullanılır.
func Load() (*Config, error) {
	// .env varsa yükle (mevcut env değişkenlerini ezmeden).
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config: env parse: %w", err)
	}

	if err := Validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate, yapılandırmayı struct tag'lerine göre doğrular ve
// production ortamında ek güvenlik kurallarını uygular.
func Validate(cfg *Config) error {
	v := validator.New(validator.WithRequiredStructEnabled())
	if err := v.Struct(cfg); err != nil {
		return fmt.Errorf("config: validation: %w", err)
	}

	// Production ortamında zayıf varsayılanları reddet.
	if cfg.App.IsProduction() {
		if cfg.Auth.JWTSecret.Value() == "change-me-in-production-please-32b" {
			return fmt.Errorf("config: production ortamında AUTH_JWT_SECRET değiştirilmelidir")
		}
		if len(cfg.Auth.JWTSecret.Value()) < 32 {
			return fmt.Errorf("config: production ortamında JWT secret en az 32 karakter olmalıdır")
		}
		if cfg.Auth.MFAEncryptionKey.Value() == "" {
			return fmt.Errorf("config: production ortamında AUTH_MFA_ENCRYPTION_KEY zorunludur")
		}
		if cfg.DB.SSLMode == "disable" {
			return fmt.Errorf("config: production ortamında DB_SSLMODE disable olamaz")
		}
	}

	// Varsayılan dil, desteklenen diller arasında olmalıdır.
	supported := false
	for _, l := range cfg.I18n.Supported {
		if l == cfg.I18n.DefaultLocale {
			supported = true
			break
		}
	}
	if !supported {
		return fmt.Errorf("config: I18N_DEFAULT_LOCALE (%q) I18N_SUPPORTED listesinde olmalıdır", cfg.I18n.DefaultLocale)
	}
	if cfg.App.IsProduction() && cfg.Payment.FieldEncryptionKey.Value() == "" {
		return fmt.Errorf("config: production ortamında PAYMENT_FIELD_ENCRYPTION_KEY zorunludur")
	}
	if cfg.App.IsProduction() && !cfg.Security.TurnstileEnabled() {
		return fmt.Errorf("config: production ortamında SEC_TURNSTILE_SITE_KEY ve SEC_TURNSTILE_SECRET_KEY zorunludur")
	}
	if cfg.App.IsProduction() && !cfg.SMTP.Configured() {
		return fmt.Errorf("config: production ortamında SMTP_HOST ve SMTP_FROM zorunludur")
	}
	if cfg.App.IsProduction() && strings.TrimSpace(cfg.Contact.RecipientEmail) == "" {
		return fmt.Errorf("config: production ortamında CONTACT_RECIPIENT_EMAIL zorunludur")
	}
	return nil
}
