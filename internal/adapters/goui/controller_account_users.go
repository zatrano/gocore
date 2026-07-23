package goui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"

	adapters "github.com/zatrano/gocore/internal/adapters/shared"
	"github.com/zatrano/gocore/internal/domain/shared"
	"github.com/zatrano/gocore/pkg/datetime"
	"github.com/zatrano/gocore/pkg/pagination"
	"github.com/zatrano/gocore/pkg/validation"
)

// accountUsersController, panel / hesap / gelen kutusu / kullanıcı ekranları için Controller üretir.
func accountUsersController(screen string) Controller {
	switch screen {
	case "dashboard":
		return &dashboardController{}
	case "account":
		return &accountController{}
	case "mfa":
		return &mfaController{}
	case "inbox":
		return &inboxController{}
	case "users":
		return &usersListController{}
	case "user-new":
		return &userNewController{}
	case "user-show":
		return &userShowController{}
	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// Ortak yardımcılar
// ---------------------------------------------------------------------------

func accountDisplayErr(err error) string {
	if err == nil {
		return ""
	}
	var inv validation.InvalidFields
	if errors.As(err, &inv) {
		return "Bir veya daha fazla alan geçersiz"
	}
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		return "Bir veya daha fazla alan geçersiz"
	}
	if de, ok := shared.AsDomainError(err); ok {
		return de.Message
	}
	return err.Error()
}

func accountFieldErrors(ctx context.Context, err error) map[string]string {
	if err == nil {
		return nil
	}
	var inv validation.InvalidFields
	if errors.As(err, &inv) {
		return inv.FieldMap(ctx)
	}
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return nil
	}
	out := make(map[string]string, len(verrs))
	for _, fe := range verrs {
		key := strings.ToLower(fe.Field())
		out[key] = validation.FieldMessage(ctx, fe)
	}
	return out
}

func validateDeps(p *Page, dst any) error {
	if p.Deps.Validate == nil {
		return nil
	}
	return validation.Check(p.Deps.Validate, dst)
}

func requireAccess(err error) error {
	if err != nil {
		return err
	}
	return nil
}

func actorID(p *Page) string   { return p.Actor.UserID }
func actorRole(p *Page) string { return p.Actor.Role }

func preferredLocale(p *Page) string {
	if p == nil {
		return "tr"
	}
	if loc := strings.TrimSpace(p.Locale); loc != "" {
		return loc
	}
	return "tr"
}

func pageQuery(p *Page, key, fallback string) string {
	if p.Query == nil {
		return fallback
	}
	if v, ok := p.Query[key]; ok && v != "" {
		return v
	}
	return fallback
}

func paramID(p *Page) string {
	if p.Params == nil {
		return ""
	}
	return p.Params["id"]
}

func parsePageLimit(p *Page) (page, limit int) {
	page = adapters.ParsePage(pageQuery(p, "page", "1"))
	limit = adapters.ParseLimit(pageQuery(p, "limit", ""))
	if limit <= 0 {
		limit = pagination.DefaultLimit
	}
	return page, limit
}

func setQuery(p *Page, key, value string) {
	if p.Query == nil {
		p.Query = map[string]string{}
	}
	if value == "" {
		delete(p.Query, key)
		return
	}
	p.Query[key] = value
}

func formatShort(t datetime.JSONTime) string {
	return datetime.FormatDateTimeShort(t.Time)
}

func formatShortPtr(t *datetime.JSONTime) string {
	if t == nil {
		return ""
	}
	return datetime.FormatDateTimeShort(t.Time)
}

func renderPagination(prefix string, pageNum, totalPages int) string {
	btns := viewPagination(pageNum, totalPages)
	if len(btns) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<nav class="pagination" aria-label="Sayfalama">`)
	for _, btn := range btns {
		cls := "page-btn"
		if btn.Active {
			cls += " active"
		}
		fmt.Fprintf(&b, `<button type="button" class="%s" g-click="%s.page" data-goui-value="%d" data-key="page-%d">%d</button>`,
			cls, prefix, btn.N, btn.N, btn.N)
	}
	b.WriteString(`</nav>`)
	return b.String()
}

func payloadPage(payload map[string]any, fallback int) int {
	raw := ""
	if v, ok := payload["value"]; ok {
		raw = fmt.Sprint(v)
	}
	if raw == "" {
		raw = payloadString(payload, "page")
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

func payloadID(payload map[string]any) string {
	if v, ok := payload["id"]; ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	if v, ok := payload["value"]; ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return payloadString(payload, "id")
}
