package handler

import (
	"context"
	"log/slog"
	"strings"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"

	adapters "github.com/zatrano/gocore/internal/adapters/shared"
	"github.com/zatrano/gocore/internal/application/auth"
	"github.com/zatrano/gocore/internal/infrastructure/logger"
	"github.com/zatrano/gocore/internal/infrastructure/realtime"
)

const cookieAccessToken = "access_token"

// RealtimeHandler, genel uygulama WebSocket uç noktasını yönetir (/api/v1/ws).
// GoUI /goui/ws'ten bağımsızdır; mobil ve API istemcileri de bağlanabilir.
type RealtimeHandler struct {
	hub      *realtime.Hub
	sessions wsTokenVerifier
	log      *slog.Logger
}

type wsTokenVerifier interface {
	Verify(ctx context.Context, token string) (auth.Claims, error)
}

// NewRealtimeHandler, canlı olay WS handler'ını kurar.
func NewRealtimeHandler(hub *realtime.Hub, sessions wsTokenVerifier, log *slog.Logger) *RealtimeHandler {
	return &RealtimeHandler{hub: hub, sessions: sessions, log: log}
}

// Authenticate, upgrade öncesi kimlik doğrular.
// Sıra: Authorization Bearer → ?access_token= / ?token= → access_token cookie.
func (h *RealtimeHandler) Authenticate(c fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}
	token := extractWSToken(c)
	if token == "" {
		return fiber.ErrUnauthorized
	}
	claims, err := h.sessions.Verify(c.Context(), token)
	if err != nil || claims.UserID == "" {
		return fiber.ErrUnauthorized
	}
	c.Locals(adapters.LocalUserID, claims.UserID)
	c.Locals(adapters.LocalRole, claims.Role)
	c.Locals(adapters.LocalEmail, claims.Email)
	c.SetContext(logger.WithUserID(c.Context(), claims.UserID))
	return c.Next()
}

// Upgrade, Fiber websocket handler'ını döner.
func (h *RealtimeHandler) Upgrade() fiber.Handler {
	return websocket.New(h.serve, websocket.Config{
		AllowEmptyOrigin: true, // mobil / native istemciler Origin göndermeyebilir
	})
}

func (h *RealtimeHandler) serve(c *websocket.Conn) {
	userID, _ := c.Locals(adapters.LocalUserID).(string)
	userID = strings.TrimSpace(userID)
	if userID == "" || h.hub == nil {
		_ = c.Close()
		return
	}

	client := h.hub.Subscribe(userID)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for msg := range client.Outbound() {
			if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}()
	defer func() {
		h.hub.Unsubscribe(userID, client)
		<-done
	}()

	// Bağlantı açılışında güncel rozet durumu (mobil / panel senkron).
	h.hub.NotifyInbox(userID)

	for {
		if _, _, err := c.ReadMessage(); err != nil {
			break
		}
	}
}

func extractWSToken(c fiber.Ctx) string {
	if header := c.Get(fiber.HeaderAuthorization); header != "" {
		parts := strings.SplitN(header, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}
	if t := strings.TrimSpace(c.Query("access_token")); t != "" {
		return t
	}
	if t := strings.TrimSpace(c.Query("token")); t != "" {
		return t
	}
	return strings.TrimSpace(c.Cookies(cookieAccessToken))
}
