package goui

import (
	"context"
	"errors"
	"html"
	"strings"

	appshared "github.com/zatrano/gocore/internal/application/shared"
	"github.com/zatrano/gocore/pkg/rbac"
)

var (
	errForbiddenSPA    = errors.New("bu işlem için yetkiniz yok")
	errClientIPMissing = errors.New("istemci IP adresi oturum bağlamında yok; 3DS başlatılamaz")
)

func requireAnyPerm(ctx context.Context, p *Page, perms ...rbac.Permission) error {
	for _, perm := range perms {
		if p.Allowed(ctx, perm) {
			return nil
		}
	}
	return errForbiddenSPA
}

func actorClientIP(ctx context.Context) string {
	return strings.TrimSpace(appshared.ActorFromContext(ctx).IP)
}

func payloadValue(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	if v, ok := payload["value"]; ok && v != nil {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func escapeHTML(s string) string { return html.EscapeString(s) }
