package goui

import (
	"net/url"

	"github.com/gofiber/fiber/v3"
)

const (
	flashTypeKey = "web_flash_type"
	flashMsgKey  = "web_flash_msg"
)

// Flash, bir sonraki istekte gösterilecek mesajı cookie'ye yazar.
func Flash(c fiber.Ctx, kind, message string) {
	if c == nil || message == "" {
		return
	}
	if kind == "" {
		kind = "info"
	}
	c.Cookie(&fiber.Cookie{Name: flashTypeKey, Value: kind, Path: "/", MaxAge: 60})
	c.Cookie(&fiber.Cookie{Name: flashMsgKey, Value: url.QueryEscape(message), Path: "/", MaxAge: 60})
}

// ConsumeFlash, flash cookie'lerini okur ve temizler.
func ConsumeFlash(c fiber.Ctx) (kind, message string) {
	if c == nil {
		return "", ""
	}
	kind = c.Cookies(flashTypeKey)
	message = c.Cookies(flashMsgKey)
	if message != "" {
		if decoded, err := url.QueryUnescape(message); err == nil {
			message = decoded
		}
	}
	if kind != "" || message != "" {
		c.Cookie(&fiber.Cookie{Name: flashTypeKey, Value: "", Path: "/", MaxAge: -1})
		c.Cookie(&fiber.Cookie{Name: flashMsgKey, Value: "", Path: "/", MaxAge: -1})
	}
	return kind, message
}

func applyFlash(page *Page, kind, message string) {
	if page == nil || message == "" {
		return
	}
	switch kind {
	case "error":
		page.Error = message
	default:
		// success | info | warning | boş → Notice (SweetAlert)
		page.Notice = message
		if kind == "warning" || kind == "info" {
			page.NoticeKind = kind
		} else {
			page.NoticeKind = "success"
		}
	}
}
