package goui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	appauth "github.com/zatrano/gocore/internal/application/auth"
	appnotif "github.com/zatrano/gocore/internal/application/notification"
	appuser "github.com/zatrano/gocore/internal/application/user"
	"github.com/zatrano/gocore/pkg/pagination"
)

// inboxItem, inbox template'ine geçirilen bildirim satırı.
type inboxItem struct {
	ID        string
	Title     string
	Content   string
	CreatedAt string
	Read      bool
}

// ---------------------------------------------------------------------------
// Account
// ---------------------------------------------------------------------------

type accountChangeNameForm struct {
	Name string `form:"name" validate:"required,min=2,max=100"`
}
type accountChangeEmailForm struct {
	Email string `form:"email" validate:"required,email" sanitize:"email"`
}
type accountChangePhoneForm struct {
	Phone string `form:"phone" validate:"omitempty,phone" sanitize:"phone"`
}
type accountChangeLocaleForm struct {
	Locale string `form:"locale" validate:"required"`
}
type accountChangePasswordForm struct {
	OldPassword string `form:"old_password" validate:"required"`
	NewPassword string `form:"new_password" validate:"required,min=8"`
}

type accountController struct {
	profile     appuser.View
	fieldErrors map[string]string
	formName    string
	formEmail   string
	formPhone   string
	formLocale  string
}

func (c *accountController) load(ctx context.Context, p *Page) error {
	if p.Deps.Users == nil {
		return errors.New("profil servisi yapılandırılmamış")
	}
	view, err := p.Deps.Users.Get(ctx, appuser.GetQuery{
		UserID: actorID(p), ActorID: actorID(p), ActorRole: actorRole(p),
	})
	if err != nil {
		return err
	}
	c.profile = view
	c.formName = view.Name
	c.formEmail = view.Email
	c.formPhone = view.Phone
	c.formLocale = view.PreferredLocale
	return nil
}

func (c *accountController) Mount(ctx context.Context, p *Page) error {
	if err := c.load(ctx, p); err != nil {
		p.Error = accountDisplayErr(err)
		return nil
	}
	return nil
}

func (c *accountController) Render(p *Page) (string, error) {
	mfaLabel := "Kapalı"
	if c.profile.MFAEnabled {
		mfaLabel = "Etkin"
	}
	emailLabel := "Bekliyor"
	if c.profile.EmailVerified {
		emailLabel = "Doğrulandı"
	}
	return p.RenderView("pages.account", map[string]any{
		"Role":               c.profile.Role,
		"MFAEnabled":         c.profile.MFAEnabled,
		"MFALabel":           mfaLabel,
		"EmailVerified":      c.profile.EmailVerified,
		"EmailVerifiedLabel": emailLabel,
		"FormName":           c.formName,
		"FormEmail":          c.formEmail,
		"FormPhone":          c.formPhone,
		"LocaleOptions":      viewLocaleOptions(p.Deps.Locales, c.formLocale),
		"ErrName":            viewFieldError(c.fieldErrors, "name"),
		"ErrEmail":           viewFieldError(c.fieldErrors, "email"),
		"ErrPhone":           viewFieldError(c.fieldErrors, "phone"),
		"ErrLocale":          viewFieldError(c.fieldErrors, "locale"),
		"ErrOldPassword":     viewFieldError(c.fieldErrors, "old_password"),
		"ErrNewPassword":     viewFieldError(c.fieldErrors, "new_password"),
	})
}

func (c *accountController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	c.fieldErrors = nil
	switch event {
	case "account.change_name":
		req := accountChangeNameForm{Name: payloadString(payload, "name")}
		c.formName = req.Name
		if err := validateDeps(p, &req); err != nil {
			c.fieldErrors = accountFieldErrors(ctx, err)
			p.Error = accountDisplayErr(err)
			return nil
		}
		if p.Deps.Users == nil {
			return errors.New("ad güncelleme servisi yapılandırılmamış")
		}
		if _, err := p.Deps.Users.ChangeName(ctx, appuser.ChangeNameCommand{
			UserID: actorID(p), Name: req.Name, ActorID: actorID(p), ActorRole: actorRole(p),
		}); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		_ = c.load(ctx, p)
		p.Notice = "ad güncellendi"
	case "account.change_email":
		req := accountChangeEmailForm{Email: payloadString(payload, "email")}
		c.formEmail = req.Email
		if err := validateDeps(p, &req); err != nil {
			c.fieldErrors = accountFieldErrors(ctx, err)
			p.Error = accountDisplayErr(err)
			return nil
		}
		if p.Deps.Users == nil {
			return errors.New("e-posta güncelleme servisi yapılandırılmamış")
		}
		if _, err := p.Deps.Users.ChangeEmail(ctx, appuser.ChangeEmailCommand{
			UserID: actorID(p), NewEmail: req.Email, ActorID: actorID(p), ActorRole: actorRole(p),
		}); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		_ = c.load(ctx, p)
		p.Notice = "e-posta adresi güncellendi"
	case "account.change_phone":
		req := accountChangePhoneForm{Phone: payloadString(payload, "phone")}
		c.formPhone = req.Phone
		if err := validateDeps(p, &req); err != nil {
			c.fieldErrors = accountFieldErrors(ctx, err)
			p.Error = accountDisplayErr(err)
			return nil
		}
		if p.Deps.Users == nil {
			return errors.New("telefon güncelleme servisi yapılandırılmamış")
		}
		if _, err := p.Deps.Users.ChangePhone(ctx, appuser.ChangePhoneCommand{
			UserID: actorID(p), Phone: req.Phone, ActorID: actorID(p), ActorRole: actorRole(p),
		}); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		_ = c.load(ctx, p)
		p.Notice = "telefon numarası güncellendi"
	case "account.change_locale":
		req := accountChangeLocaleForm{Locale: payloadString(payload, "locale")}
		c.formLocale = req.Locale
		if err := validateDeps(p, &req); err != nil {
			c.fieldErrors = accountFieldErrors(ctx, err)
			p.Error = accountDisplayErr(err)
			return nil
		}
		if p.Deps.Users == nil {
			return errors.New("dil güncelleme servisi yapılandırılmamış")
		}
		if _, err := p.Deps.Users.ChangeLocale(ctx, appuser.ChangeLocaleCommand{
			UserID: actorID(p), Locale: req.Locale,
		}); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		_ = c.load(ctx, p)
		p.Notice = "dil tercihi güncellendi"
	case "account.change_password":
		req := accountChangePasswordForm{
			OldPassword: payloadString(payload, "old_password"),
			NewPassword: payloadString(payload, "new_password"),
		}
		if err := validateDeps(p, &req); err != nil {
			c.fieldErrors = accountFieldErrors(ctx, err)
			p.Error = accountDisplayErr(err)
			return nil
		}
		if p.Deps.Auth == nil {
			return errors.New("şifre değiştirme servisi yapılandırılmamış")
		}
		if err := p.Deps.Auth.ChangePassword(ctx, appauth.ChangePasswordCommand{
			UserID: actorID(p), OldPassword: req.OldPassword, NewPassword: req.NewPassword,
		}); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		p.Notice = "şifreniz değiştirildi"
		p.Redirect = "/auth/login"
	}
	return nil
}

// ---------------------------------------------------------------------------
// MFA
// ---------------------------------------------------------------------------

type mfaCodeForm struct {
	Code string `form:"code" validate:"required"`
}

type mfaController struct {
	profile       appuser.View
	setup         *appauth.SetupResult
	recoveryCodes []string
	fieldErrors   map[string]string
}

func (c *mfaController) load(ctx context.Context, p *Page) error {
	if p.Deps.Users == nil {
		return errors.New("profil servisi yapılandırılmamış")
	}
	view, err := p.Deps.Users.Get(ctx, appuser.GetQuery{
		UserID: actorID(p), ActorID: actorID(p), ActorRole: actorRole(p),
	})
	if err != nil {
		return err
	}
	c.profile = view
	return nil
}

func (c *mfaController) Mount(ctx context.Context, p *Page) error {
	if err := c.load(ctx, p); err != nil {
		p.Error = accountDisplayErr(err)
	}
	return nil
}

func (c *mfaController) Render(p *Page) (string, error) {
	statusLabel := "Kapalı"
	if c.profile.MFAEnabled {
		statusLabel = "Etkin"
	}
	data := map[string]any{
		"MFAEnabled":        c.profile.MFAEnabled,
		"StatusLabel":       statusLabel,
		"ShowRecoveryCodes": len(c.recoveryCodes) > 0,
		"RecoveryCodes":     c.recoveryCodes,
		"ShowSetup":         c.setup != nil,
		"ShowSetupBtn":      c.setup == nil && !c.profile.MFAEnabled,
		"ErrCode":           viewFieldError(c.fieldErrors, "code"),
	}
	if c.setup != nil {
		data["QRDataURI"] = c.setup.QRDataURI
		data["Secret"] = c.setup.Secret
		data["URI"] = c.setup.URI
	}
	return p.RenderView("pages.mfa", data)
}

func (c *mfaController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	c.fieldErrors = nil
	if p.Deps.Auth == nil {
		return errors.New("MFA servisi yapılandırılmamış")
	}
	switch event {
	case "mfa.setup":
		res, err := p.Deps.Auth.MFASetup(ctx, actorID(p))
		if err != nil {
			p.Error = accountDisplayErr(err)
			_ = c.load(ctx, p)
			return nil
		}
		c.setup = &res
		_ = c.load(ctx, p)
	case "mfa.enable":
		req := mfaCodeForm{Code: payloadString(payload, "code")}
		if err := validateDeps(p, &req); err != nil {
			c.fieldErrors = accountFieldErrors(ctx, err)
			p.Error = accountDisplayErr(err)
			return nil
		}
		result, err := p.Deps.Auth.MFAEnable(ctx, appauth.EnableCommand{UserID: actorID(p), Code: req.Code})
		if err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		c.setup = nil
		c.recoveryCodes = result.RecoveryCodes
		_ = c.load(ctx, p)
		p.Notice = "iki adımlı doğrulama etkinleştirildi"
	case "mfa.disable":
		req := mfaCodeForm{Code: payloadString(payload, "code")}
		if err := validateDeps(p, &req); err != nil {
			c.fieldErrors = accountFieldErrors(ctx, err)
			p.Error = accountDisplayErr(err)
			return nil
		}
		if err := p.Deps.Auth.MFADisable(ctx, appauth.DisableCommand{UserID: actorID(p), Code: req.Code}); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		_ = c.load(ctx, p)
		p.Notice = "iki adımlı doğrulama kapatıldı"
	}
	return nil
}

// ---------------------------------------------------------------------------
// Inbox
// ---------------------------------------------------------------------------

type inboxController struct {
	page       pagination.Page[appnotif.View]
	unreadOnly bool
	unread     int64
	renderGen  int
}

func (c *inboxController) reload(ctx context.Context, p *Page) error {
	if p.Deps.Notifications == nil {
		return errors.New("bildirim listesi servisi yapılandırılmamış")
	}
	pageNum, limit := parsePageLimit(p)
	page, err := p.Deps.Notifications.List(ctx, appnotif.ListMyQuery{
		UserID: actorID(p), Page: pageNum, Limit: limit, UnreadOnly: c.unreadOnly,
	})
	if err != nil {
		return err
	}
	c.page = page
	if p.Deps.Notifications != nil {
		if n, err := p.Deps.Notifications.UnreadCount(ctx, actorID(p)); err == nil {
			c.unread = n
		}
	}
	if c.unread == 0 && c.unreadOnly {
		c.unreadOnly = false
		setQuery(p, "filter", "")
		page, err := p.Deps.Notifications.List(ctx, appnotif.ListMyQuery{
			UserID: actorID(p), Page: pageNum, Limit: limit, UnreadOnly: false,
		})
		if err != nil {
			return err
		}
		c.page = page
	}
	return nil
}

func (c *inboxController) Mount(ctx context.Context, p *Page) error {
	c.unreadOnly = pageQuery(p, "filter", "") == "unread"
	if err := c.reload(ctx, p); err != nil {
		p.Error = accountDisplayErr(err)
	}
	return nil
}

func (c *inboxController) Render(p *Page) (string, error) {
	items := make([]inboxItem, 0, len(c.page.Items))
	for _, n := range c.page.Items {
		items = append(items, inboxItem{
			ID:        n.ID,
			Title:     n.Title,
			Content:   n.Content,
			CreatedAt: formatShort(n.CreatedAt),
			Read:      n.Read,
		})
	}
	return p.RenderView("pages.inbox", map[string]any{
		"UnreadCount": c.unread,
		"UnreadOnly":  c.unreadOnly,
		"HasUnread":   c.unread > 0,
		"HasItems":    c.page.Total > 0,
		"Empty":       len(c.page.Items) == 0,
		"Items":       items,
		"RenderGen":   c.renderGen,
		"Pages":       viewPagination(c.page.Page, c.page.TotalPages),
	})
}

func (c *inboxController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	switch event {
	case "inbox.mark_read":
		id := payloadID(payload)
		if id == "" {
			p.Error = "bildirim kimliği gerekli"
			return nil
		}
		if p.Deps.Notifications == nil {
			return errors.New("bildirim okuma servisi yapılandırılmamış")
		}
		if err := p.Deps.Notifications.MarkRead(ctx, appnotif.MarkReadCommand{
			UserID: actorID(p), NotificationID: id,
		}); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		if p.Deps.InboxRealtime != nil {
			p.Deps.InboxRealtime.NotifyInbox(actorID(p))
		}
		p.Notice = "bildirim okundu işaretlendi"
		c.renderGen++
		_ = c.reload(ctx, p)
	case "inbox.mark_all_read":
		if p.Deps.Notifications == nil {
			return errors.New("bildirim okuma servisi yapılandırılmamış")
		}
		if _, err := p.Deps.Notifications.MarkAllRead(ctx, appnotif.MarkAllReadCommand{
			UserID: actorID(p),
		}); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		if p.Deps.InboxRealtime != nil {
			p.Deps.InboxRealtime.NotifyInbox(actorID(p))
		}
		p.Notice = "tüm bildirimler okundu işaretlendi"
		c.renderGen++
		_ = c.reload(ctx, p)
	case "inbox.delete":
		id := payloadID(payload)
		if id == "" {
			p.Error = "bildirim kimliği gerekli"
			return nil
		}
		if p.Deps.Notifications == nil {
			return errors.New("bildirim silme servisi yapılandırılmamış")
		}
		if err := p.Deps.Notifications.Delete(ctx, appnotif.DeleteCommand{
			UserID: actorID(p), NotificationID: id,
		}); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		if p.Deps.InboxRealtime != nil {
			p.Deps.InboxRealtime.NotifyInbox(actorID(p))
		}
		p.Notice = "bildirim silindi"
		c.renderGen++
		_ = c.reload(ctx, p)
	case "inbox.delete_all":
		if p.Deps.Notifications == nil {
			return errors.New("bildirim silme servisi yapılandırılmamış")
		}
		if _, err := p.Deps.Notifications.DeleteAll(ctx, appnotif.DeleteAllCommand{
			UserID: actorID(p),
		}); err != nil {
			p.Error = accountDisplayErr(err)
			return nil
		}
		if p.Deps.InboxRealtime != nil {
			p.Deps.InboxRealtime.NotifyInbox(actorID(p))
		}
		p.Notice = "tüm bildirimler silindi"
		setQuery(p, "page", "1")
		c.renderGen++
		_ = c.reload(ctx, p)
	case "inbox.page":
		n := payloadPage(payload, c.page.Page)
		setQuery(p, "page", strconv.Itoa(n))
		_ = c.reload(ctx, p)
	case "inbox.filter":
		v := ""
		if raw, ok := payload["value"]; ok {
			v = strings.TrimSpace(fmt.Sprint(raw))
		}
		c.unreadOnly = v == "unread"
		if c.unreadOnly {
			setQuery(p, "filter", "unread")
		} else {
			setQuery(p, "filter", "")
		}
		setQuery(p, "page", "1")
		_ = c.reload(ctx, p)
	}
	return nil
}
