package goui

import (
	"embed"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	gouitemplate "github.com/zatrano/goui/template"
	"github.com/zatrano/goui/ws"
)

//go:embed all:views
var embeddedViews embed.FS

// ShellLabels, layout/partial şablonlarında kullanılan çeviri metinleri.
type ShellLabels struct {
	OpenMenu      string
	CloseMenu     string
	Logout        string
	Contact       string
	Dashboard     string
	Login         string
	Register      string
	Notifications string
}

// ShellLocale, dil seçici seçenekleri.
type ShellLocale struct {
	Code     string
	Label    string
	Selected bool
}

// ShellNavItem, sidebar bağlantısı veya ayarlar grubu.
type ShellNavItem struct {
	Href     string
	Icon     string
	Label    string
	Active   bool
	Group    bool
	Open     bool
	Children []ShellNavItem
}

// ShellData, layouts.shell ve partial'lara verilen veri modeli.
type ShellData struct {
	Protected      bool
	AuthShell      bool
	LoggedIn       bool
	Title          string
	Redirect       string
	Notice         string
	NoticeKind     string
	Error          string
	Body           htmltemplate.HTML
	Locales        []ShellLocale
	ContactPath    string
	Labels         ShellLabels
	NavItems       []ShellNavItem
	ActorRole      string
	ActorEmail     string
	ActorUserID    string
	UnreadCount    int64
	UnreadBadge    string
	HasUnread      bool
	NotifAriaLabel string
}

func openViews(deps Deps, hub *ws.Hub) (*gouitemplate.Registry, func(), error) {
	root, cleanup, watch, err := resolveViewsRoot(deps)
	if err != nil {
		return nil, nil, err
	}
	cfg := gouitemplate.Config{
		Root:            root,
		StrictProps:     deps.Secure,
		WatchForChanges: watch,
		ExtraFuncs: htmltemplate.FuncMap{
			"icon": func(name string) htmltemplate.HTML {
				return htmltemplate.HTML(navIcon(name)) // #nosec G203 -- SVG ikonları sabit güvenilir içerik
			},
		},
	}
	if watch && hub != nil {
		cfg.OnReload = func() {
			hub.Broadcast(ws.PushMessage{Kind: "reload", Text: "templates updated"})
		}
		cfg.OnReloadError = func(err error) {
			log.Printf("goui templates reload: %v", err)
		}
	}
	reg, err := gouitemplate.NewRegistry(cfg)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, nil, fmt.Errorf("goui templates: %w", err)
	}
	for _, w := range reg.Warnings() {
		log.Printf("goui template warning: %s", w)
	}
	return reg, cleanup, nil
}

func resolveViewsRoot(deps Deps) (root string, cleanup func(), watch bool, err error) {
	if custom := strings.TrimSpace(deps.ViewsRoot); custom != "" {
		if st, stErr := os.Stat(custom); stErr != nil || !st.IsDir() {
			return "", nil, false, fmt.Errorf("views root %q: %w", custom, stErr)
		}
		return custom, nil, !deps.Secure, nil
	}
	if src := sourceViewsDir(); src != "" {
		return src, nil, !deps.Secure, nil
	}
	dir, err := os.MkdirTemp("", "gocore-goui-views-*")
	if err != nil {
		return "", nil, false, err
	}
	if err := copyEmbeddedViews(dir); err != nil {
		_ = os.RemoveAll(dir)
		return "", nil, false, err
	}
	return dir, func() { _ = os.RemoveAll(dir) }, false, nil
}

func sourceViewsDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	dir := filepath.Join(filepath.Dir(file), "views")
	if st, err := os.Stat(dir); err == nil && st.IsDir() {
		return dir
	}
	return ""
}

func copyEmbeddedViews(dst string) error {
	return fs.WalkDir(embeddedViews, "views", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel("views", path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		data, err := embeddedViews.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o600)
	})
}

var (
	defaultViewsOnce sync.Once
	defaultViewsReg  *gouitemplate.Registry
	defaultViewsErr  error
)

// testViews, birim testlerinde Page.Views verilmediğinde gömülü şablonları yükler.
func testViews() (*gouitemplate.Registry, error) {
	defaultViewsOnce.Do(func() {
		defaultViewsReg, _, defaultViewsErr = openViews(Deps{}, nil)
	})
	return defaultViewsReg, defaultViewsErr
}
