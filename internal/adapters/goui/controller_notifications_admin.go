package goui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	adapters "github.com/zatrano/gocore/internal/adapters/shared"
	appnotif "github.com/zatrano/gocore/internal/application/notification"
	"github.com/zatrano/gocore/pkg/rbac"
	"github.com/zatrano/gocore/pkg/recipients"
	"github.com/zatrano/gocore/pkg/validation"
)

func bulkSendNotice(channel appnotif.Channel, accepted int) string {
	if channel == appnotif.ChannelInApp {
		return fmt.Sprintf("bildirim gönderildi (%d alıcı)", accepted)
	}
	return fmt.Sprintf("toplu gönderim kuyruğa alındı (%d alıcı)", accepted)
}
func defaultNotifChannel(s string) string {
	if s == "" {
		return string(appnotif.ChannelInApp)
	}
	return s
}

func defaultAudience(s string) string {
	if s == "all" {
		return "all"
	}
	return "one"
}

func localeOptions(deps Deps, selected string) []string {
	locales := deps.Locales
	if len(locales) == 0 {
		locales = []string{"tr", "en"}
	}
	if selected == "" && len(locales) > 0 {
		_ = locales[0]
	}
	return locales
}

// ---------------------------------------------------------------------------
// notification-send
// ---------------------------------------------------------------------------

type notificationSendController struct {
	channel   string
	audience  string
	email     string
	phone     string
	locale    string
	title     string
	body      string
	htmlBody  string
	fieldErrs map[string]string
}

func (c *notificationSendController) Mount(_ context.Context, p *Page) error {
	c.channel = defaultNotifChannel("")
	c.audience = "one"
	if p.Query != nil {
		if ch := p.Query["channel"]; ch != "" {
			c.channel = defaultNotifChannel(ch)
		}
		if aud := p.Query["audience"]; aud != "" {
			c.audience = defaultAudience(aud)
		}
	}
	locales := localeOptions(p.Deps, "")
	if c.locale == "" && len(locales) > 0 {
		c.locale = locales[0]
	}
	return nil
}

func (c *notificationSendController) Render(p *Page) (string, error) {
	locales := localeOptions(p.Deps, c.locale)
	return p.RenderView("pages.notification_send", map[string]any{
		"Channel":        c.channel,
		"Audience":       c.audience,
		"Email":          c.email,
		"Phone":          c.phone,
		"Title":          c.title,
		"Body":           c.body,
		"HTMLBody":       c.htmlBody,
		"Locale":         c.locale,
		"EmailError":     viewFieldError(c.fieldErrs, "email"),
		"PhoneError":     viewFieldError(c.fieldErrs, "phone"),
		"ChannelOptions": viewChannelOptions(c.channel),
		"LocaleOptions":  viewLocaleOptions(locales, c.locale),
	})
}

type sendNotifForm struct {
	Channel  string `form:"channel" validate:"required"`
	Audience string `form:"audience"`
	Email    string `form:"email" validate:"omitempty,email" sanitize:"email"`
	Phone    string `form:"phone" validate:"omitempty,phone" sanitize:"phone"`
	Locale   string `form:"locale"`
	Title    string `form:"title"`
	Body     string `form:"body" validate:"required"`
	HTMLBody string `form:"html_body"`
}

func (c *notificationSendController) applyForm(req sendNotifForm) {
	c.channel = defaultNotifChannel(req.Channel)
	c.audience = defaultAudience(req.Audience)
	c.email = req.Email
	c.phone = req.Phone
	c.locale = req.Locale
	c.title = req.Title
	c.body = req.Body
	c.htmlBody = req.HTMLBody
}

func (c *notificationSendController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	if err := requirePagePerm(ctx, p, rbac.PermNotificationsSend); err != nil {
		return err
	}
	switch event {
	case "notification.channel":
		c.channel = defaultNotifChannel(payloadString(payload, "value"))
		return nil
	case "notification.audience":
		c.audience = defaultAudience(payloadString(payload, "value"))
		return nil
	case "notification.body":
		c.body = payloadString(payload, "value")
		return nil
	case "notification.send":
		req := sendNotifForm{
			Channel:  payloadString(payload, "channel"),
			Audience: payloadString(payload, "audience"),
			Email:    payloadString(payload, "email"),
			Phone:    payloadString(payload, "phone"),
			Locale:   payloadString(payload, "locale"),
			Title:    payloadString(payload, "title"),
			Body:     payloadString(payload, "body"),
			HTMLBody: payloadString(payload, "html_body"),
		}
		c.applyForm(req)
		c.fieldErrs = nil
		if p.Deps.Validate != nil {
			if err := validation.Check(p.Deps.Validate, &req); err != nil {
				c.fieldErrs = fieldErrorsFrom(err)
				c.applyForm(req)
				p.Error = userFacingError(err)
				return nil
			}
		}
		c.applyForm(req)
		if p.Deps.Notifications == nil {
			return errSenderRequired
		}
		channel, err := appnotif.ParseChannel(req.Channel)
		if err != nil {
			p.Error = userFacingError(err)
			return nil
		}
		content := appnotif.MessageContent{Title: req.Title, Body: req.Body, HTMLBody: req.HTMLBody}
		if req.Audience == "all" {
			res, err := p.Deps.Notifications.SendToAllUsers(ctx, channel, content, req.Locale)
			if err != nil {
				p.Error = userFacingError(err)
				return nil
			}
			if res.Accepted == 0 {
				p.Error = "hiçbir alıcıya gönderilemedi"
				return nil
			}
			p.Notice = bulkSendNotice(channel, res.Accepted)
			p.Redirect = "/dashboard/notifications/send?channel=" + string(channel) + "&audience=all"
			return nil
		}
		if err := p.Deps.Notifications.SendOne(ctx, channel, appnotif.Recipient{
			Email: req.Email, Phone: req.Phone, Locale: req.Locale,
		}, content); err != nil {
			p.Error = userFacingError(err)
			return nil
		}
		p.Notice = "bildirim gönderildi"
		p.Redirect = "/dashboard/notifications/send?channel=" + string(channel)
		return nil
	}
	return nil
}

// ---------------------------------------------------------------------------
// notification-bulk
// ---------------------------------------------------------------------------

type notificationBulkController struct {
	channel    string
	title      string
	body       string
	htmlBody   string
	locale     string
	recipients string
	files      []uploadedRef
	result     *appnotif.BulkResult
	fieldErrs  map[string]string
}

func (c *notificationBulkController) Mount(_ context.Context, p *Page) error {
	c.channel = defaultNotifChannel("")
	if p.Query != nil {
		if ch := p.Query["channel"]; ch != "" {
			c.channel = defaultNotifChannel(ch)
		}
	}
	locales := localeOptions(p.Deps, "")
	if c.locale == "" && len(locales) > 0 {
		c.locale = locales[0]
	}
	return nil
}

type bulkInvalidRow struct {
	Index  string
	Reason string
}

func (c *notificationBulkController) Render(p *Page) (string, error) {
	locales := localeOptions(p.Deps, c.locale)
	data := map[string]any{
		"Channel":        c.channel,
		"Title":          c.title,
		"Body":           c.body,
		"HTMLBody":       c.htmlBody,
		"Locale":         c.locale,
		"Recipients":     c.recipients,
		"ChannelOptions": viewChannelOptions(c.channel),
		"LocaleOptions":  viewLocaleOptions(locales, c.locale),
		"TemplateLinks":  viewExportLinks("/dashboard/notifications/recipients/template", nil),
		"Files":          viewUploadFiles(c.files),
		"HasResult":      c.result != nil,
	}
	if c.result != nil {
		data["ResultTotal"] = strconv.Itoa(c.result.Total)
		data["ResultAccepted"] = strconv.Itoa(c.result.Accepted)
		data["ResultInvalid"] = strconv.Itoa(len(c.result.Invalid))
		if len(c.result.Invalid) > 0 {
			inv := make([]bulkInvalidRow, 0, len(c.result.Invalid))
			for _, item := range c.result.Invalid {
				inv = append(inv, bulkInvalidRow{Index: strconv.Itoa(item.Index), Reason: item.Reason})
			}
			data["InvalidItems"] = inv
		}
	}
	return p.RenderView("pages.notification_bulk", data)
}

type bulkNotifForm struct {
	Channel    string `form:"channel" validate:"required"`
	Title      string `form:"title"`
	Body       string `form:"body" validate:"required"`
	HTMLBody   string `form:"html_body"`
	Locale     string `form:"locale"`
	Recipients string `form:"recipients" validate:"required"`
}

func (c *notificationBulkController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	if err := requirePagePerm(ctx, p, rbac.PermNotificationsSend); err != nil {
		return err
	}
	switch event {
	case "notification.bulk_channel":
		c.channel = defaultNotifChannel(payloadString(payload, "value"))
		return nil
	case "notification.file.uploaded":
		c.files = appendUploadRef(c.files, parseUploadRef(payload), false)
		return nil
	case "notification.file.remove":
		c.files = removeUploadRef(c.files, payloadString(payload, "id"))
		return nil
	case "notification.bulk_file":
		return c.sendFromFile(ctx, p, payload)
	case "notification.bulk":
		req := bulkNotifForm{
			Channel:    payloadString(payload, "channel"),
			Title:      payloadString(payload, "title"),
			Body:       payloadString(payload, "body"),
			HTMLBody:   payloadString(payload, "html_body"),
			Locale:     payloadString(payload, "locale"),
			Recipients: payloadString(payload, "recipients"),
		}
		c.channel = defaultNotifChannel(req.Channel)
		c.title = req.Title
		c.body = req.Body
		c.htmlBody = req.HTMLBody
		c.locale = req.Locale
		c.recipients = req.Recipients
		c.fieldErrs = nil
		c.result = nil
		if p.Deps.Validate != nil {
			if err := validation.Check(p.Deps.Validate, &req); err != nil {
				c.fieldErrs = fieldErrorsFrom(err)
				p.Error = userFacingError(err)
				return nil
			}
		}
		if p.Deps.Notifications == nil {
			return errSenderRequired
		}
		channel, err := appnotif.ParseChannel(req.Channel)
		if err != nil {
			p.Error = userFacingError(err)
			return nil
		}
		recipients := adapters.RecipientsFromTextLines(req.Recipients, req.Locale, channel)
		res, err := p.Deps.Notifications.SendBulk(ctx, adapters.BuildBulkCommand(channel, adapters.BulkMessageContent{
			Title: req.Title, Body: req.Body, HTMLBody: req.HTMLBody, Locale: req.Locale,
		}, recipients))
		if err != nil {
			p.Error = userFacingError(err)
			return nil
		}
		c.result = &res
		if res.Accepted == 0 {
			p.Error = "hiçbir alıcıya gönderilemedi"
			return nil
		}
		p.Notice = bulkSendNotice(channel, res.Accepted)
		return nil
	}
	return nil
}

func (c *notificationBulkController) sendFromFile(ctx context.Context, p *Page, payload map[string]any) error {
	c.channel = defaultNotifChannel(payloadString(payload, "channel"))
	c.title = payloadString(payload, "title")
	c.body = payloadString(payload, "body")
	c.htmlBody = payloadString(payload, "html_body")
	c.locale = payloadString(payload, "locale")
	c.result = nil
	if len(c.files) == 0 {
		p.Error = "önce CSV veya Excel dosyası yükleyin"
		return nil
	}
	if p.Deps.Notifications == nil || p.Deps.Storage == nil {
		return errSenderRequired
	}
	channel, err := appnotif.ParseChannel(c.channel)
	if err != nil {
		p.Error = userFacingError(err)
		return nil
	}
	if strings.TrimSpace(c.body) == "" {
		p.Error = "içerik zorunludur"
		return nil
	}
	ref := c.files[0]
	src, _, err := p.Deps.Storage.Get(ctx, ref.ID)
	if err != nil {
		p.Error = userFacingError(err)
		return nil
	}
	defer src.Close()
	list, err := recipients.Parse(ref.Name, src)
	if err != nil {
		p.Error = userFacingError(err)
		return nil
	}
	parsed, err := adapters.RecipientsFromParsed(list)
	if err != nil {
		p.Error = userFacingError(err)
		return nil
	}
	res, err := p.Deps.Notifications.SendBulk(ctx, adapters.BuildBulkCommand(channel, adapters.BulkMessageContent{
		Title: c.title, Body: c.body, HTMLBody: c.htmlBody, Locale: c.locale,
	}, parsed))
	if err != nil {
		p.Error = userFacingError(err)
		return nil
	}
	cleanupUploadRefs(ctx, p.Deps.Storage, c.files)
	c.files = nil
	c.result = &res
	if res.Accepted == 0 {
		p.Error = "hiçbir alıcıya gönderilemedi"
		return nil
	}
	p.Notice = bulkSendNotice(channel, res.Accepted)
	return nil
}

// ---------------------------------------------------------------------------
// notification-upload
// ---------------------------------------------------------------------------

type notificationUploadController struct {
	channel  string
	title    string
	body     string
	htmlBody string
	locale   string
	files    []uploadedRef
	result   *appnotif.BulkResult
}

func (c *notificationUploadController) Mount(_ context.Context, p *Page) error {
	// Dosya yükleme artık /dashboard/notifications/bulk içinde.
	p.Redirect = "/dashboard/notifications/bulk"
	return nil
}

func (c *notificationUploadController) Render(p *Page) (string, error) {
	locales := localeOptions(p.Deps, c.locale)
	data := map[string]any{
		"Channel":        c.channel,
		"Title":          c.title,
		"Body":           c.body,
		"HTMLBody":       c.htmlBody,
		"Locale":         c.locale,
		"ChannelOptions": viewChannelOptions(c.channel),
		"LocaleOptions":  viewLocaleOptions(locales, c.locale),
		"TemplateLinks":  viewExportLinks("/dashboard/notifications/recipients/template", nil),
		"Files":          viewUploadFiles(c.files),
		"HasResult":      c.result != nil,
	}
	if c.result != nil {
		data["ResultTotal"] = strconv.Itoa(c.result.Total)
		data["ResultAccepted"] = strconv.Itoa(c.result.Accepted)
		data["ResultInvalid"] = strconv.Itoa(len(c.result.Invalid))
	}
	return p.RenderView("pages.notification_upload", data)
}

func (c *notificationUploadController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	if err := requirePagePerm(ctx, p, rbac.PermNotificationsSend); err != nil {
		return err
	}
	switch event {
	case "notification.upload_channel":
		c.channel = defaultNotifChannel(payloadString(payload, "value"))
		return nil
	case "notification.file.uploaded":
		c.files = appendUploadRef(c.files, parseUploadRef(payload), false)
		return nil
	case "notification.file.remove":
		c.files = removeUploadRef(c.files, firstNonEmpty(payloadString(payload, "value"), payloadString(payload, "id")))
		return nil
	case "notification.upload_send":
		c.channel = defaultNotifChannel(payloadString(payload, "channel"))
		c.title = payloadString(payload, "title")
		c.body = payloadString(payload, "body")
		c.htmlBody = payloadString(payload, "html_body")
		c.locale = payloadString(payload, "locale")
		c.result = nil
		if len(c.files) == 0 {
			p.Error = errMissingUpload.Error()
			return nil
		}
		ref := c.files[0]
		maxBytes := maxUploadBytes(p)
		if ref.Size > maxBytes {
			p.Error = errFileTooLarge.Error()
			return nil
		}
		if p.Deps.Storage == nil {
			return errStorageRequired
		}
		if p.Deps.Notifications == nil {
			return errSenderRequired
		}
		channel, err := appnotif.ParseChannel(c.channel)
		if err != nil {
			p.Error = userFacingError(err)
			return nil
		}
		src, _, err := p.Deps.Storage.Get(ctx, ref.ID)
		if err != nil {
			p.Error = userFacingError(err)
			return nil
		}
		defer src.Close()
		list, err := recipients.Parse(ref.Name, src)
		if err != nil {
			p.Error = userFacingError(err)
			return nil
		}
		parsed, err := adapters.RecipientsFromParsed(list)
		if err != nil {
			p.Error = userFacingError(err)
			return nil
		}
		res, err := p.Deps.Notifications.SendBulk(ctx, adapters.BuildBulkCommand(channel, adapters.BulkMessageContent{
			Title: c.title, Body: c.body, HTMLBody: c.htmlBody, Locale: c.locale,
		}, parsed))
		if err != nil {
			p.Error = userFacingError(err)
			return nil
		}
		cleanupUploadRefs(ctx, p.Deps.Storage, c.files)
		c.files = nil
		c.result = &res
		p.Notice = bulkSendNotice(channel, res.Accepted)
		return nil
	}
	return nil
}
