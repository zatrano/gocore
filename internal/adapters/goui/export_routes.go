package goui

import (
	"bytes"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	appaudit "github.com/zatrano/gocore/internal/application/audit"
	appcontact "github.com/zatrano/gocore/internal/application/contact"
	apppayment "github.com/zatrano/gocore/internal/application/payment"
	appuser "github.com/zatrano/gocore/internal/application/user"
	"github.com/zatrano/gocore/pkg/datetime"
	"github.com/zatrano/gocore/pkg/rbac"
	"github.com/zatrano/gocore/pkg/tabular"
)

const exportMaxRows = 1000 // pagination.MaxLimit ile hizalı

func (ui *UI) registerExportRoutes(app *fiber.App) {
	app.Get("/dashboard/users/export", ui.requireExport(rbac.PermUsersList, ui.exportUsers))
	app.Get("/dashboard/contacts/export", ui.requireExport(rbac.PermContactsList, ui.exportContacts))
	app.Get("/dashboard/payments/transactions/export", ui.requireExport(rbac.PermPaymentsList, ui.exportPayments))
	app.Get("/dashboard/audit/logs/export", ui.requireExport(rbac.PermAuditList, ui.exportAudit))
	app.Get("/dashboard/notifications/recipients/template", ui.requireExport(rbac.PermNotificationsSend, ui.exportRecipientTemplate))
}

func (ui *UI) requireExport(perm rbac.Permission, next fiber.Handler) fiber.Handler {
	return func(c fiber.Ctx) error {
		claims, _, _, err := ui.authenticate(c, c.Cookies(cookieAccess), c.Cookies(cookieRefresh))
		if err != nil {
			return c.Redirect().To("/auth/login?next=" + url.QueryEscape(c.OriginalURL()))
		}
		if !ui.anyPermission(c, claims.Role, []rbac.Permission{perm}) {
			return fiber.ErrForbidden
		}
		c.Locals("export_role", claims.Role)
		return next(c)
	}
}

func (ui *UI) writeExport(c fiber.Ctx, filename string, headers []string, rows [][]string) error {
	format := tabular.ParseFormat(c.Query("format"))
	var buf bytes.Buffer
	if err := tabular.Write(&buf, format, "export", headers, rows); err != nil {
		return err
	}
	c.Set("Content-Type", format.ContentType())
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, filename, format.Extension()))
	return c.Send(buf.Bytes())
}

func (ui *UI) exportUsers(c fiber.Ctx) error {
	if ui.deps.Users == nil {
		return fiber.ErrInternalServerError
	}
	role, _ := c.Locals("export_role").(string)
	q := appuser.ListQuery{
		ActorRole: role,
		Role:      c.Query("role"),
		Search:    c.Query("search"),
		Deleted:   c.Query("deleted"),
		Page:      1,
		Limit:     exportMaxRows,
		Ascending: c.Query("order") == "asc",
	}
	if activeStr := c.Query("active"); activeStr != "" {
		active := activeStr == "true"
		q.Active = &active
	}
	page, err := ui.deps.Users.List(c.Context(), q)
	if err != nil {
		return err
	}
	rows := make([][]string, 0, len(page.Items))
	for _, u := range page.Items {
		status := "passive"
		if u.Deleted {
			status = "deleted"
		} else if u.Active {
			status = "active"
		}
		rows = append(rows, []string{u.ID, u.Name, u.Email, u.Role, status, u.PreferredLocale})
	}
	return ui.writeExport(c, "users", []string{"id", "name", "email", "role", "status", "locale"}, rows)
}

func (ui *UI) exportContacts(c fiber.Ctx) error {
	if ui.deps.Contacts == nil {
		return fiber.ErrInternalServerError
	}
	unreadOnly := c.Query("unread") == "1"
	page, err := ui.deps.Contacts.List(c.Context(), appcontact.ListQuery{
		UnreadOnly: unreadOnly,
		Page:       1,
		Limit:      exportMaxRows,
	})
	if err != nil {
		return err
	}
	rows := make([][]string, 0, len(page.Items))
	for _, item := range page.Items {
		read := "1"
		if !item.Read {
			read = "0"
		}
		rows = append(rows, []string{
			item.ID, item.Name, item.Email, item.Message, item.Locale, item.Status, read,
			datetime.FormatDateTime(item.CreatedAt.Time),
		})
	}
	return ui.writeExport(c, "contacts",
		[]string{"id", "name", "email", "message", "locale", "status", "read", "created_at"}, rows)
}

func (ui *UI) exportPayments(c fiber.Ctx) error {
	if ui.deps.ThreeDSSvc == nil {
		return fiber.ErrInternalServerError
	}
	page, err := ui.deps.ThreeDSSvc.ListPayments(c.Context(), apppayment.ListPaymentsQuery{
		Status:    c.Query("status"),
		Provider:  c.Query("provider"),
		Page:      1,
		Limit:     exportMaxRows,
		Ascending: c.Query("order") == "asc",
	})
	if err != nil {
		return err
	}
	rows := make([][]string, 0, len(page.Items))
	for _, item := range page.Items {
		amt := item.Amount
		if item.PaidAmount != "" {
			amt = item.PaidAmount
		}
		rows = append(rows, []string{
			item.Reference, item.Provider, item.Status, item.StatusLabel,
			strings.TrimSpace(item.BuyerName + " " + item.BuyerSurname),
			item.BuyerEmail, amt, item.Currency, strconv.Itoa(item.Installment),
			item.CardDisplay, formatShort(item.CreatedAt),
		})
	}
	return ui.writeExport(c, "payments",
		[]string{"reference", "provider", "status", "status_label", "buyer", "email", "amount", "currency", "installment", "card", "created_at"},
		rows)
}

func (ui *UI) exportAudit(c fiber.Ctx) error {
	if ui.deps.Audit == nil {
		return fiber.ErrInternalServerError
	}
	page, err := ui.deps.Audit.List(c.Context(), appaudit.ListQuery{
		Action:    c.Query("action"),
		Resource:  c.Query("resource"),
		Actor:     c.Query("actor"),
		Page:      1,
		Limit:     exportMaxRows,
		Ascending: c.Query("order") == "asc",
	})
	if err != nil {
		return err
	}
	rows := make([][]string, 0, len(page.Items))
	for _, item := range page.Items {
		rows = append(rows, []string{
			item.ID, item.CreatedAt, item.ActorID, item.ActorEmail, item.ActorType,
			item.Action, item.Resource, item.ResourceID, item.Source, item.CorrelationID,
		})
	}
	return ui.writeExport(c, "audit_logs",
		[]string{"id", "created_at", "actor_id", "actor_email", "actor_type", "action", "resource", "resource_id", "source", "correlation_id"},
		rows)
}

func (ui *UI) exportRecipientTemplate(c fiber.Ctx) error {
	headers := []string{"email", "phone", "name", "locale", "user_id"}
	sample := [][]string{
		{"ornek@firma.com", "+905551112233", "Ali Veli", "tr", ""},
	}
	return ui.writeExport(c, "recipients_template", headers, sample)
}
