package goui

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/zatrano/gocore/pkg/pagination"
)

// ViewOption, select option verisi.
type ViewOption struct {
	Value    string
	Label    string
	Selected bool
}

// ViewLink, export / aksiyon linki.
type ViewLink struct {
	Href  string
	Label string
}

// ViewPageBtn, sayfalama düğmesi.
type ViewPageBtn struct {
	N      int
	Active bool
}

// ViewUploadFile, upload zone listedeki dosya.
type ViewUploadFile struct {
	ID   string
	Name string
	Size int64
}

// ViewFeature, landing feature kartı.
type ViewFeature struct {
	Key   string
	Title string
	Body  string
}

// ViewDetail, dl satırı.
type ViewDetail struct {
	Label string
	Value string
	HTML  bool // Value güvenilir HTML ise true (badge vb.)
}

func viewFieldError(errs map[string]string, name string) string {
	if errs == nil {
		return ""
	}
	msg, ok := errs[name]
	if !ok || msg == "" {
		msg = errs[strings.ToLower(name)]
	}
	return msg
}

func viewPagination(pageNum, totalPages int) []ViewPageBtn {
	if totalPages <= 1 {
		return nil
	}
	nums := pagination.VisiblePageNumbers(pageNum, totalPages, 7)
	out := make([]ViewPageBtn, 0, len(nums))
	for _, n := range nums {
		out = append(out, ViewPageBtn{N: n, Active: n == pageNum})
	}
	return out
}

func viewExportLinks(basePath string, filters map[string]string) []ViewLink {
	q := url.Values{}
	for k, v := range filters {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		switch k {
		case "page", "limit", "format":
			continue
		}
		q.Set(k, v)
	}
	mk := func(format, label string) ViewLink {
		qq := cloneURLValues(q)
		qq.Set("format", format)
		href := basePath
		if enc := qq.Encode(); enc != "" {
			href += "?" + enc
		}
		return ViewLink{Href: href, Label: label}
	}
	return []ViewLink{mk("csv", "CSV"), mk("xlsx", "Excel")}
}

func viewLimitOptions(selected int) []ViewOption {
	out := make([]ViewOption, 0, len(pagination.AllowedLimits))
	for _, n := range pagination.AllowedLimits {
		out = append(out, ViewOption{
			Value:    strconv.Itoa(n),
			Label:    strconv.Itoa(n),
			Selected: n == selected,
		})
	}
	return out
}

func viewSelectOptions(pairs [][2]string, selected string) []ViewOption {
	out := make([]ViewOption, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, ViewOption{Value: p[0], Label: p[1], Selected: p[0] == selected})
	}
	return out
}

func viewTurnstileEnabled(p *Page) bool {
	return p != nil && p.Deps.Turnstile != nil && p.Deps.Turnstile.Enabled() && p.Deps.TurnstileSiteKey != ""
}

func viewLocaleOptions(locales []string, selected string) []ViewOption {
	if len(locales) == 0 {
		locales = []string{"tr", "en"}
	}
	selected = strings.ToLower(strings.TrimSpace(selected))
	out := make([]ViewOption, 0, len(locales))
	for _, loc := range locales {
		loc = strings.ToLower(strings.TrimSpace(loc))
		if loc == "" {
			continue
		}
		out = append(out, ViewOption{Value: loc, Label: localeLabel(loc), Selected: loc == selected})
	}
	return out
}

func viewChannelOptions(selected string) []ViewOption {
	return viewSelectOptions([][2]string{
		{"inapp", "Uygulama İçi"},
		{"email", "E-posta"},
		{"sms", "SMS"},
	}, selected)
}

func viewUploadFiles(refs []uploadedRef) []ViewUploadFile {
	out := make([]ViewUploadFile, 0, len(refs))
	for _, f := range refs {
		out = append(out, ViewUploadFile{ID: f.ID, Name: f.Name, Size: f.Size})
	}
	return out
}
