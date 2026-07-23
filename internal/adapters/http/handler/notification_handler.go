package handler

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/pkg/validation"

	"github.com/zatrano/gocore/internal/adapters/http/render"
	adapters "github.com/zatrano/gocore/internal/adapters/shared"
	appnotif "github.com/zatrano/gocore/internal/application/notification"
	"github.com/zatrano/gocore/pkg/recipients"
)

// NotificationHandler, iki grup uç nokta sağlar:
//   - Kullanıcının kendi uygulama içi bildirimleri (listeleme + okundu işaretleme).
//   - Yönetici kaynaklı elle/toplu gönderim (notifications:send izni gerektirir),
//     ayrıca CSV/Excel ile alıcı listesi yükleme.
type NotificationHandler struct {
	notif    *appnotif.Service
	realtime inboxRealtime
	validate *validator.Validate
	maxBytes int64
}

type inboxRealtime interface {
	NotifyInbox(userID string)
}

// NotificationDeps, NotificationHandler bağımlılıklarını gruplar.
type NotificationDeps struct {
	Notifications *appnotif.Service
	Realtime      inboxRealtime // can be nil
	Validate      *validator.Validate
	MaxBytes      int64
}

// NewNotificationHandler, handler'ı kurar.
func NewNotificationHandler(d NotificationDeps) *NotificationHandler {
	return &NotificationHandler{
		notif: d.Notifications, realtime: d.Realtime,
		validate: d.Validate, maxBytes: d.MaxBytes,
	}
}

// List, GET /users/me/notifications — kullanıcının bildirimleri (sayfa tabanlı sayfalama).
func (h *NotificationHandler) List(c fiber.Ctx) error {
	userID, _ := c.Locals(adapters.LocalUserID).(string)
	page, err := h.notif.List(c.Context(), appnotif.ListMyQuery{
		UserID: userID,
		Page:   adapters.ParsePage(c.Query("page")),
		Limit:  adapters.ParseLimit(c.Query("limit")),
	})
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, page)
}

// UnreadCount, GET /users/me/notifications/unread-count — okunmamış sayısı.
func (h *NotificationHandler) UnreadCount(c fiber.Ctx) error {
	userID, _ := c.Locals(adapters.LocalUserID).(string)
	count, err := h.notif.UnreadCount(c.Context(), userID)
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSON(c, fiber.StatusOK, fiber.Map{"unread": count})
}

// MarkRead, POST /users/me/notifications/:id/read — bildirimi okundu işaretler.
func (h *NotificationHandler) MarkRead(c fiber.Ctx) error {
	userID, _ := c.Locals(adapters.LocalUserID).(string)
	err := h.notif.MarkRead(c.Context(), appnotif.MarkReadCommand{
		UserID:         userID,
		NotificationID: c.Params("id"),
	})
	if err != nil {
		return render.Error(c, err)
	}
	if h.realtime != nil && userID != "" {
		h.realtime.NotifyInbox(userID)
	}
	return render.Message(c, fiber.StatusOK, "success.notification.read", "bildirim okundu işaretlendi")
}

// MarkAllRead, POST /users/me/notifications/read-all — tümünü okundu işaretler.
func (h *NotificationHandler) MarkAllRead(c fiber.Ctx) error {
	userID, _ := c.Locals(adapters.LocalUserID).(string)
	_, err := h.notif.MarkAllRead(c.Context(), appnotif.MarkAllReadCommand{UserID: userID})
	if err != nil {
		return render.Error(c, err)
	}
	if h.realtime != nil && userID != "" {
		h.realtime.NotifyInbox(userID)
	}
	return render.Message(c, fiber.StatusOK, "success.notification.read_all", "tüm bildirimler okundu işaretlendi")
}

// Delete, DELETE /users/me/notifications/:id — bildirimi siler.
func (h *NotificationHandler) Delete(c fiber.Ctx) error {
	userID, _ := c.Locals(adapters.LocalUserID).(string)
	err := h.notif.Delete(c.Context(), appnotif.DeleteCommand{
		UserID:         userID,
		NotificationID: c.Params("id"),
	})
	if err != nil {
		return render.Error(c, err)
	}
	if h.realtime != nil && userID != "" {
		h.realtime.NotifyInbox(userID)
	}
	return render.Message(c, fiber.StatusOK, "success.notification.deleted", "bildirim silindi")
}

// DeleteAll, DELETE /users/me/notifications — tüm bildirimleri siler.
func (h *NotificationHandler) DeleteAll(c fiber.Ctx) error {
	userID, _ := c.Locals(adapters.LocalUserID).(string)
	_, err := h.notif.DeleteAll(c.Context(), appnotif.DeleteAllCommand{UserID: userID})
	if err != nil {
		return render.Error(c, err)
	}
	if h.realtime != nil && userID != "" {
		h.realtime.NotifyInbox(userID)
	}
	return render.Message(c, fiber.StatusOK, "success.notification.deleted_all", "tüm bildirimler silindi")
}

// --- Yönetici gönderim uç noktaları (notifications:send izni) ---

type recipientRequest struct {
	Phone  string `json:"phone" validate:"omitempty,phone" sanitize:"phone"`
	Email  string `json:"email" validate:"omitempty,email" sanitize:"email"`
	Locale string `json:"locale"`
}

func (r recipientRequest) toDomain() appnotif.Recipient {
	return appnotif.Recipient{Phone: r.Phone, Email: r.Email, Locale: r.Locale}
}

type sendRequest struct {
	Channel   string           `json:"channel" validate:"required"`
	Audience  string           `json:"audience"` // "one" (varsayılan) | "all"
	Recipient recipientRequest `json:"recipient"`
	Title     string           `json:"title"`
	Body      string           `json:"body" validate:"required"`
	HTMLBody  string           `json:"html_body"`
	Locale    string           `json:"locale"`
}

type bulkSendRequest struct {
	Channel    string             `json:"channel" validate:"required"`
	Title      string             `json:"title"`
	Body       string             `json:"body" validate:"required"`
	HTMLBody   string             `json:"html_body"`
	Locale     string             `json:"locale"`
	Recipients []recipientRequest `json:"recipients" validate:"omitempty,dive"`
}

// Send, POST /notifications/send — tek alıcıya veya tüm kullanıcılara gönderir.
func (h *NotificationHandler) Send(c fiber.Ctx) error {
	var req sendRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	channel, err := appnotif.ParseChannel(req.Channel)
	if err != nil {
		return render.Error(c, err)
	}
	content := appnotif.MessageContent{Title: req.Title, Body: req.Body, HTMLBody: req.HTMLBody}
	if req.Audience == "all" {
		res, err := h.notif.SendToAllUsers(c.Context(), channel, content, req.Locale)
		if err != nil {
			return render.Error(c, err)
		}
		return render.JSONWithMessage(c, fiber.StatusAccepted, "success.notification.bulk_queued", "toplu gönderim kuyruğa alındı", res)
	}
	err = h.notif.SendOne(c.Context(), channel, req.Recipient.toDomain(), content)
	if err != nil {
		return render.Error(c, err)
	}
	return render.Message(c, fiber.StatusAccepted, "success.notification.sent", "bildirim gönderildi")
}

// BulkSend, POST /notifications/bulk — JSON gövdedeki alıcı listesine toplu gönderim.
func (h *NotificationHandler) BulkSend(c fiber.Ctx) error {
	var req bulkSendRequest
	if err := c.Bind().Body(&req); err != nil {
		return render.Error(c, err)
	}
	if err := validation.Check(h.validate, &req); err != nil {
		return render.Error(c, err)
	}
	channel, err := appnotif.ParseChannel(req.Channel)
	if err != nil {
		return render.Error(c, err)
	}
	content := appnotif.MessageContent{Title: req.Title, Body: req.Body, HTMLBody: req.HTMLBody}
	if len(req.Recipients) == 0 {
		return render.ProblemLocalized(c, 400, "notification.recipients_required",
			"title.validation", "Geçersiz istek",
			"notification.recipients_required", "en az bir alıcı gerekir")
	}

	recipients := make([]appnotif.Recipient, 0, len(req.Recipients))
	for _, r := range req.Recipients {
		recipients = append(recipients, r.toDomain())
	}

	res, err := h.notif.SendBulk(c.Context(), appnotif.BulkSendCommand{
		Channel:    channel,
		Content:    content,
		Recipients: recipients,
		Locale:     req.Locale,
	})
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSONWithMessage(c, fiber.StatusAccepted, "success.notification.bulk_queued", "toplu gönderim kuyruğa alındı", res)
}

// BulkUpload, POST /notifications/bulk/upload — CSV/Excel dosyasından alıcı
// listesi okuyup toplu gönderim yapar. Form alanları: channel, title, body,
// html_body, locale; dosya alanı: file.
func (h *NotificationHandler) BulkUpload(c fiber.Ctx) error {
	channel, err := appnotif.ParseChannel(c.FormValue("channel"))
	if err != nil {
		return render.Error(c, err)
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return render.ProblemLocalized(c, 400, "upload.missing_file",
			"title.validation", "Geçersiz istek",
			"upload.missing_file", "form'da 'file' alanı bekleniyor")
	}
	if fileHeader.Size > h.maxBytes {
		return render.ProblemLocalized(c, 413, "upload.too_large",
			"title.payload_too_large", "Dosya çok büyük",
			"upload.too_large", "izin verilen boyut aşıldı")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return render.Error(c, err)
	}
	defer src.Close()

	list, err := recipients.Parse(fileHeader.Filename, src)
	if err != nil {
		return render.ProblemLocalized(c, 400, "notification.parse_failed",
			"title.validation", "Geçersiz istek",
			"notification.parse_failed", "alıcı listesi ayrıştırılamadı: "+err.Error())
	}

	parsed, err := adapters.RecipientsFromParsed(list)
	if err != nil {
		return render.Error(c, err)
	}
	res, err := h.notif.SendBulk(c.Context(), adapters.BuildBulkCommand(channel, adapters.BulkMessageContent{
		Title: c.FormValue("title"), Body: c.FormValue("body"),
		HTMLBody: c.FormValue("html_body"), Locale: c.FormValue("locale"),
	}, parsed))
	if err != nil {
		return render.Error(c, err)
	}
	return render.JSONWithMessage(c, fiber.StatusAccepted, "success.notification.bulk_queued", "toplu gönderim kuyruğa alındı", res)
}
