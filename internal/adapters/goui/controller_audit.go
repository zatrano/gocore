package goui

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	adaptershared "github.com/zatrano/gocore/internal/adapters/shared"
	appaudit "github.com/zatrano/gocore/internal/application/audit"
	"github.com/zatrano/gocore/pkg/pagination"
	"github.com/zatrano/gocore/pkg/rbac"
)

// --- Audit ---

type auditListCtrl struct {
	action, resource, actor, order string
	pageNum, limit                 int
	page                           pagination.Page[appaudit.View]
}

func (c *auditListCtrl) Mount(ctx context.Context, p *Page) error {
	if err := requireAnyPerm(ctx, p, rbac.PermAuditList); err != nil {
		return err
	}
	c.action = pageQuery(p, "action", c.action)
	c.resource = pageQuery(p, "resource", c.resource)
	c.actor = pageQuery(p, "actor", c.actor)
	c.order = pageQuery(p, "order", c.order)
	c.pageNum, c.limit = parsePageLimit(p)
	return c.reload(ctx, p)
}

func (c *auditListCtrl) reload(ctx context.Context, p *Page) error {
	if p.Deps.Audit == nil {
		return errors.New("audit list servisi yapılandırılmamış")
	}
	page, err := p.Deps.Audit.List(ctx, appaudit.ListQuery{
		Action: c.action, Resource: c.resource, Actor: c.actor,
		Page: c.pageNum, Limit: c.limit, Ascending: c.order == "asc",
	})
	if err != nil {
		return err
	}
	c.page = page
	c.pageNum, c.limit = page.Page, page.Limit
	return nil
}

type auditRow struct {
	ID         string
	CreatedAt  string
	ActorEmail string
	ActorID    string
	ActorType  string
	Action     string
	Resource   string
	ResourceID string
	DetailHref string
}

func (c *auditListCtrl) Render(p *Page) (string, error) {
	items := make([]auditRow, 0, len(c.page.Items))
	for _, item := range c.page.Items {
		items = append(items, auditRow{
			ID:         item.ID,
			CreatedAt:  item.CreatedAt,
			ActorEmail: item.ActorEmail,
			ActorID:    item.ActorID,
			ActorType:  item.ActorType,
			Action:     item.Action,
			Resource:   item.Resource,
			ResourceID: item.ResourceID,
			DetailHref: "/dashboard/audit/logs/" + item.ID,
		})
	}
	return p.RenderView("pages.audit", map[string]any{
		"ExportLinks":     viewExportLinks("/dashboard/audit/logs/export", map[string]string{"action": c.action, "resource": c.resource, "actor": c.actor, "order": c.order}),
		"Action":          c.action,
		"Resource":        c.resource,
		"Actor":           c.actor,
		"Order":           c.order,
		"Limit":           strconv.Itoa(c.limit),
		"ResourceOptions": viewSelectOptions(auditResourceOptions(), c.resource),
		"OrderOptions":    viewSelectOptions([][2]string{{"", "Yeniden eskiye"}, {"asc", "Eskiden yeniye"}}, c.order),
		"LimitOptions":    viewLimitOptions(c.limit),
		"Items":           items,
		"Pages":           viewPagination(c.pageNum, c.page.TotalPages),
	})
}

func auditResourceOptions() [][2]string {
	return [][2]string{
		{"", "Tüm kaynaklar"},
		{"auth", "Kimlik doğrulama"},
		{"user", "Kullanıcı"},
		{"rbac", "Yetki / rol"},
		{"payment", "Ödeme"},
		{"settings", "Ayarlar"},
		{"notification", "Bildirim"},
		{"upload", "Yükleme"},
		{"contact", "İletişim"},
	}
}

func (c *auditListCtrl) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	if err := requireAnyPerm(ctx, p, rbac.PermAuditList); err != nil {
		return err
	}
	switch event {
	case "field.action":
		c.action = payloadValue(payload)
	case "field.resource":
		c.resource = payloadValue(payload)
	case "field.actor":
		c.actor = payloadValue(payload)
	case "field.order":
		c.order = payloadValue(payload)
	case "field.limit":
		c.limit = adaptershared.ParseLimit(payloadValue(payload))
	case "audit.filter":
		c.action = payloadString(payload, "action")
		c.resource = payloadString(payload, "resource")
		c.actor = payloadString(payload, "actor")
		c.order = payloadString(payload, "order")
		c.limit = adaptershared.ParseLimit(payloadString(payload, "limit"))
		c.pageNum = 1
		return c.reload(ctx, p)
	case "audit.clear":
		c.action, c.resource, c.actor, c.order = "", "", "", ""
		c.pageNum = 1
		c.limit = 0
		return c.reload(ctx, p)
	case "audit.page":
		c.pageNum = payloadPage(payload, c.pageNum)
		return c.reload(ctx, p)
	}
	return nil
}

type auditShowCtrl struct {
	log appaudit.View
}

func (c *auditShowCtrl) Mount(ctx context.Context, p *Page) error {
	if err := requireAnyPerm(ctx, p, rbac.PermAuditList); err != nil {
		return err
	}
	if p.Deps.Audit == nil {
		return errors.New("audit get servisi yapılandırılmamış")
	}
	id := ""
	if p.Params != nil {
		id = p.Params["id"]
	}
	view, err := p.Deps.Audit.Get(ctx, id)
	if err != nil {
		return err
	}
	c.log = view
	return nil
}

func (c *auditShowCtrl) Render(p *Page) (string, error) {
	log := c.log
	resourceID := log.ResourceID
	if resourceID == "" {
		resourceID = "—"
	}
	actor := "—"
	if log.ActorEmail != "" {
		actor = log.ActorEmail
	} else if log.ActorID != "" {
		actor = log.ActorID
	}
	ip := log.IP
	if ip == "" {
		ip = "—"
	}
	ua := log.UserAgent
	if ua == "" {
		ua = "—"
	}
	rows := []ViewDetail{
		{Label: "Kayıt ID", Value: log.ID},
		{Label: "Tarih", Value: log.CreatedAt},
		{Label: "Aksiyon", Value: log.Action},
		{Label: "Kaynak", Value: log.Resource},
		{Label: "Kaynak ID", Value: resourceID},
		{Label: "Aktör", Value: actor},
	}
	if log.ActorID != "" {
		rows = append(rows, ViewDetail{Label: "Aktör ID", Value: log.ActorID})
	}
	if log.ActorType != "" {
		rows = append(rows, ViewDetail{Label: "Aktör türü", Value: log.ActorType})
	}
	if log.Source != "" {
		rows = append(rows, ViewDetail{Label: "Kaynak yüzey", Value: log.Source})
	}
	if log.CorrelationID != "" {
		rows = append(rows, ViewDetail{Label: "Correlation ID", Value: log.CorrelationID})
	}
	if log.ChangeSummary != "" {
		rows = append(rows, ViewDetail{Label: "Değişiklik", Value: log.ChangeSummary})
	}
	rows = append(rows,
		ViewDetail{Label: "IP", Value: ip},
		ViewDetail{Label: "User-Agent", Value: ua},
	)

	var metaRows []ViewDetail
	if len(log.Metadata) > 0 {
		keys := make([]string, 0, len(log.Metadata))
		for k := range log.Metadata {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			metaRows = append(metaRows, ViewDetail{Label: k, Value: fmt.Sprint(log.Metadata[k])})
		}
	}
	return p.RenderView("pages.audit_show", map[string]any{
		"Rows":     rows,
		"MetaRows": metaRows,
	})
}

func (c *auditShowCtrl) HandleEvent(ctx context.Context, p *Page, _ string, _ map[string]any) error {
	return requireAnyPerm(ctx, p, rbac.PermAuditList)
}
