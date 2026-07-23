package goui

import (
	"context"
	"html"
	"strings"

	"github.com/zatrano/goui/core"
	"github.com/zatrano/goui/diff"
	gouipage "github.com/zatrano/goui/page"
	"github.com/zatrano/goui/ws"
)

// renderSSR, ModeSEO / ModeStatic için ilk HTTP yanıtında tam HTML gövdesi üretir.
// WS hydrate aynı factory ile yeni instance açar; flash SSR örneğinde tüketilir.
func (ui *UI) renderSSR(ctx context.Context, name, locale string) (string, core.Head, error) {
	comp, err := ui.registry.Create(name)
	if err != nil {
		return "", core.Head{}, err
	}
	ws.PrepareComponent(comp, gouipage.SSRComponentID, locale, ui.translator)
	if err := comp.Mount(ctx); err != nil {
		return "", core.Head{}, err
	}
	defer func() { _ = comp.Unmount(ctx) }()

	frag, err := comp.Render()
	if err != nil {
		return "", core.Head{}, err
	}
	frag, err = ws.DecorateHTML(frag, gouipage.SSRComponentID)
	if err != nil {
		return "", core.Head{}, err
	}
	if mode, ok := ui.registry.Mode(name); ok && mode == core.ModeSEO {
		frag, err = markSSR(frag)
		if err != nil {
			return "", core.Head{}, err
		}
	}

	head := core.Head{Title: name, Lang: locale}
	if hp, ok := comp.(core.HeadProvider); ok {
		head = mergeHead(head, hp.Head(), locale)
	}
	return frag, head, nil
}

func markSSR(htmlFrag string) (string, error) {
	tree, err := diff.ParseHTML(htmlFrag)
	if err != nil {
		return "", err
	}
	if len(tree.Children) == 0 {
		return `<div data-goui-ssr="1"></div>`, nil
	}
	root := tree.Children[0]
	if root.Attrs == nil {
		root.Attrs = make(map[string]string)
	}
	root.Attrs["data-goui-ssr"] = "1"
	return diff.Serialize(tree), nil
}

func mergeHead(base, override core.Head, locale string) core.Head {
	if override.Title != "" {
		base.Title = override.Title
	}
	if override.Description != "" {
		base.Description = override.Description
	}
	if override.Canonical != "" {
		base.Canonical = override.Canonical
	}
	if override.Lang != "" {
		base.Lang = override.Lang
	} else if base.Lang == "" {
		base.Lang = locale
	}
	if override.Robots != "" {
		base.Robots = override.Robots
	}
	if override.OGTitle != "" {
		base.OGTitle = override.OGTitle
	}
	if override.OGDescription != "" {
		base.OGDescription = override.OGDescription
	}
	if override.OGImage != "" {
		base.OGImage = override.OGImage
	}
	if override.OGType != "" {
		base.OGType = override.OGType
	}
	if len(override.Extra) > 0 {
		base.Extra = override.Extra
	}
	return base
}

func renderHeadMeta(head core.Head) string {
	var b strings.Builder
	if head.Description != "" {
		b.WriteString(`<meta name="description" content="` + html.EscapeString(head.Description) + `">`)
	}
	if head.Canonical != "" {
		b.WriteString(`<link rel="canonical" href="` + html.EscapeString(head.Canonical) + `">`)
	}
	if head.Robots != "" {
		b.WriteString(`<meta name="robots" content="` + html.EscapeString(head.Robots) + `">`)
	}
	ogType := head.OGType
	if ogType == "" && (head.OGTitle != "" || head.OGDescription != "" || head.OGImage != "") {
		ogType = "website"
	}
	if head.OGTitle != "" {
		b.WriteString(`<meta property="og:title" content="` + html.EscapeString(head.OGTitle) + `">`)
	}
	if head.OGDescription != "" {
		b.WriteString(`<meta property="og:description" content="` + html.EscapeString(head.OGDescription) + `">`)
	}
	if head.OGImage != "" {
		b.WriteString(`<meta property="og:image" content="` + html.EscapeString(head.OGImage) + `">`)
	}
	if ogType != "" {
		b.WriteString(`<meta property="og:type" content="` + html.EscapeString(ogType) + `">`)
	}
	for _, m := range head.Extra {
		b.WriteString("<meta")
		if m.Name != "" {
			b.WriteString(` name="` + html.EscapeString(m.Name) + `"`)
		}
		if m.Property != "" {
			b.WriteString(` property="` + html.EscapeString(m.Property) + `"`)
		}
		b.WriteString(` content="` + html.EscapeString(m.Content) + `">`)
	}
	return b.String()
}
