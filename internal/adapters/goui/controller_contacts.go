package goui

import (
	"context"
	"errors"
	"strconv"

	adaptershared "github.com/zatrano/gocore/internal/adapters/shared"
	appcontact "github.com/zatrano/gocore/internal/application/contact"
	"github.com/zatrano/gocore/pkg/datetime"
	"github.com/zatrano/gocore/pkg/pagination"
	"github.com/zatrano/gocore/pkg/rbac"
)

func contactsController(screen string) Controller {
	switch screen {
	case "contacts":
		return &contactsListCtrl{}
	case "contact-show":
		return &contactShowCtrl{}
	default:
		return nil
	}
}

type contactsListCtrl struct {
	unreadOnly     bool
	pageNum, limit int
	page           pagination.Page[appcontact.View]
}

func (c *contactsListCtrl) Mount(ctx context.Context, p *Page) error {
	if err := requireAnyPerm(ctx, p, rbac.PermContactsList); err != nil {
		return err
	}
	c.unreadOnly = pageQuery(p, "unread", "") == "1" || pageQuery(p, "unread", "") == "true"
	c.pageNum, c.limit = parsePageLimit(p)
	return c.reload(ctx, p)
}

func (c *contactsListCtrl) reload(ctx context.Context, p *Page) error {
	if p.Deps.Contacts == nil {
		return errors.New("iletişim listesi servisi yapılandırılmamış")
	}
	page, err := p.Deps.Contacts.List(ctx, appcontact.ListQuery{
		UnreadOnly: c.unreadOnly,
		Page:       c.pageNum,
		Limit:      c.limit,
	})
	if err != nil {
		return err
	}
	c.page = page
	c.pageNum, c.limit = page.Page, page.Limit
	return nil
}

func (c *contactsListCtrl) Render(p *Page) (string, error) {
	contactFilters := map[string]string{}
	if c.unreadOnly {
		contactFilters["unread"] = "1"
	}
	type contactItem struct {
		ID         string
		Date       string
		Name       string
		Email      string
		Unread     bool
		DetailHref string
	}
	items := make([]contactItem, 0, len(c.page.Items))
	for _, item := range c.page.Items {
		items = append(items, contactItem{
			ID:         item.ID,
			Date:       datetime.FormatDateTime(item.CreatedAt.Time),
			Name:       item.Name,
			Email:      item.Email,
			Unread:     !item.Read,
			DetailHref: "/dashboard/contacts/" + item.ID,
		})
	}
	return p.RenderView("pages.contacts", map[string]any{
		"ExportLinks":   viewExportLinks("/dashboard/contacts/export", contactFilters),
		"UnreadValue":   unreadFilterValue(c.unreadOnly),
		"UnreadOptions": viewSelectOptions([][2]string{{"", "Tümü"}, {"1", "Okunmamış"}}, unreadFilterValue(c.unreadOnly)),
		"LimitValue":    strconv.Itoa(c.limit),
		"LimitOptions":  viewLimitOptions(c.limit),
		"HasItems":      len(c.page.Items) > 0,
		"Items":         items,
		"Pages":         viewPagination(c.pageNum, c.page.TotalPages),
	})
}

func unreadFilterValue(unreadOnly bool) string {
	if unreadOnly {
		return "1"
	}
	return ""
}

func (c *contactsListCtrl) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	if err := requireAnyPerm(ctx, p, rbac.PermContactsList); err != nil {
		return err
	}
	switch event {
	case "field.unread":
		c.unreadOnly = payloadValue(payload) == "1"
	case "field.limit":
		c.limit = adaptershared.ParseLimit(payloadValue(payload))
	case "contacts.filter":
		c.unreadOnly = payloadString(payload, "unread") == "1"
		c.limit = adaptershared.ParseLimit(payloadString(payload, "limit"))
		c.pageNum = 1
		return c.reload(ctx, p)
	case "contacts.clear":
		c.unreadOnly = false
		c.pageNum = 1
		c.limit = 0
		return c.reload(ctx, p)
	case "contacts.page":
		c.pageNum = payloadPage(payload, c.pageNum)
		return c.reload(ctx, p)
	}
	return nil
}

type contactShowCtrl struct {
	msg appcontact.View
}

func (c *contactShowCtrl) Mount(ctx context.Context, p *Page) error {
	if err := requireAnyPerm(ctx, p, rbac.PermContactsList); err != nil {
		return err
	}
	return c.reload(ctx, p)
}

func (c *contactShowCtrl) reload(ctx context.Context, p *Page) error {
	if p.Deps.Contacts == nil {
		return errors.New("iletişim get servisi yapılandırılmamış")
	}
	id := ""
	if p.Params != nil {
		id = p.Params["id"]
	}
	view, err := p.Deps.Contacts.Get(ctx, id)
	if err != nil {
		return err
	}
	c.msg = view
	return nil
}

func (c *contactShowCtrl) Render(p *Page) (string, error) {
	msg := c.msg
	readAt := ""
	if msg.ReadAt != nil {
		readAt = datetime.FormatDateTime(msg.ReadAt.Time)
	}
	return p.RenderView("pages.contact_show", map[string]any{
		"ID":              msg.ID,
		"Name":            msg.Name,
		"Email":           msg.Email,
		"Locale":          msg.Locale,
		"Status":          msg.Status,
		"Message":         msg.Message,
		"IsRead":          msg.Read,
		"ShowMarkRead":    !msg.Read,
		"CreatedAt":       datetime.FormatDateTime(msg.CreatedAt.Time),
		"ReadAtFormatted": readAt,
	})
}

func (c *contactShowCtrl) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	if err := requireAnyPerm(ctx, p, rbac.PermContactsList); err != nil {
		return err
	}
	if event != "contacts.mark_read" {
		return nil
	}
	if p.Deps.Contacts == nil {
		return errors.New("iletişim okundu servisi yapılandırılmamış")
	}
	id := ""
	if p.Params != nil {
		id = p.Params["id"]
	}
	view, err := p.Deps.Contacts.MarkRead(ctx, appcontact.MarkReadCommand{ID: id})
	if err != nil {
		return err
	}
	c.msg = view
	p.Notice = "Mesaj okundu olarak işaretlendi"
	return nil
}
