package goui

//go:generate go run assets_generate.go

import (
	"path"
	"strings"

	"github.com/gofiber/fiber/v3"
)

func registerAssetRoutes(app *fiber.App) {
	app.Get("/client/*", serveAsset)
	app.Get("/forms/*", serveAsset)
	app.Get("/goui/assets/*", serveAsset)
}

func serveAsset(c fiber.Ctx) error {
	name := path.Clean(c.Path())
	content, ok := assetFiles[name]
	if !ok {
		return fiber.ErrNotFound
	}
	switch {
	case strings.HasSuffix(name, ".js"):
		c.Type("js", "utf-8")
	case strings.HasSuffix(name, ".css"):
		c.Type("css", "utf-8")
	}
	c.Set("Cache-Control", "public, max-age=3600")
	return c.SendString(content)
}
