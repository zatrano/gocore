package handler

import (
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/api"
)

// DocsHandler, OpenAPI spesifikasyonunu ve Swagger UI'ı sunar.
type DocsHandler struct{}

// NewDocsHandler, handler'ı kurar.
func NewDocsHandler() *DocsHandler { return &DocsHandler{} }

// Spec, GET /openapi.yaml — gömülü OpenAPI 3.1 spesifikasyonunu döner.
func (h *DocsHandler) Spec(c fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, "application/yaml")
	return c.Send(api.OpenAPISpec)
}

// SwaggerUI, GET /docs — CDN üzerinden Swagger UI'ı yükleyen HTML sayfası döner.
func (h *DocsHandler) SwaggerUI(c fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return c.SendString(swaggerHTML)
}

const swaggerHTML = `<!DOCTYPE html>
<html lang="tr">
<head>
  <meta charset="UTF-8">
  <title>ZATRANO API - Swagger UI</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({
        url: '/openapi.yaml',
        dom_id: '#swagger-ui',
        presets: [SwaggerUIBundle.presets.apis],
        layout: 'BaseLayout'
      });
    };
  </script>
</body>
</html>`
