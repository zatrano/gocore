// Package logger, slog tabanlı yapısal loglama sağlar. Context içinden
// correlation_id gibi alanları otomatik olarak loglara ekler.
package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// ctxKey, context içinde log alanlarını taşımak için özel anahtar tipi.
type ctxKey struct{}

// fields, context'e iliştirilen log alanları.
type fields struct {
	correlationID string
	requestID     string
	userID        string
	clientIP      string
	userAgent     string
}

// New, ortam ve seviyeye göre yapılandırılmış bir *slog.Logger üretir.
// production'da JSON, development'ta daha okunabilir text handler kullanılır.
func New(env, level string) *slog.Logger {
	lvl := parseLevel(level)
	opts := &slog.HandlerOptions{
		Level:     lvl,
		AddSource: env != "production",
	}

	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	// contextHandler, her log kaydına context alanlarını enjekte eder.
	return slog.New(&contextHandler{Handler: handler})
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// contextHandler, slog.Handler'ı sarmalayıp context'ten log alanlarını çeker.
type contextHandler struct {
	slog.Handler
}

func (h *contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if f, ok := ctx.Value(ctxKey{}).(fields); ok {
		if f.requestID != "" {
			r.AddAttrs(slog.String("request_id", f.requestID))
		}
		if f.correlationID != "" {
			r.AddAttrs(slog.String("correlation_id", f.correlationID))
		}
		if f.userID != "" {
			r.AddAttrs(slog.String("user_id", f.userID))
		}
	}
	return h.Handler.Handle(ctx, r)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{Handler: h.Handler.WithGroup(name)}
}

// --- Context yardımcıları ---

func getFields(ctx context.Context) fields {
	if f, ok := ctx.Value(ctxKey{}).(fields); ok {
		return f
	}
	return fields{}
}

// WithCorrelationID, context'e correlation id ekler.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	f := getFields(ctx)
	f.correlationID = id
	return context.WithValue(ctx, ctxKey{}, f)
}

// WithRequestID, context'e istek kimliği ekler.
func WithRequestID(ctx context.Context, id string) context.Context {
	f := getFields(ctx)
	f.requestID = id
	return context.WithValue(ctx, ctxKey{}, f)
}

// WithUserID, context'e authenticated user id ekler.
func WithUserID(ctx context.Context, id string) context.Context {
	f := getFields(ctx)
	f.userID = id
	return context.WithValue(ctx, ctxKey{}, f)
}

// WithRequestClient, context'e istemci IP ve User-Agent bilgisini ekler.
func WithRequestClient(ctx context.Context, ip, userAgent string) context.Context {
	f := getFields(ctx)
	f.clientIP = ip
	f.userAgent = userAgent
	return context.WithValue(ctx, ctxKey{}, f)
}

// CorrelationID, context'teki correlation id'yi döner.
func CorrelationID(ctx context.Context) string { return getFields(ctx).correlationID }

// UserID, context'teki authenticated kullanıcı kimliğini döner.
func UserID(ctx context.Context) string { return getFields(ctx).userID }

// ClientIP, context'teki istemci IP adresini döner.
func ClientIP(ctx context.Context) string { return getFields(ctx).clientIP }

// UserAgent, context'teki istemci User-Agent başlığını döner.
func UserAgent(ctx context.Context) string { return getFields(ctx).userAgent }
