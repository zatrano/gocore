package shared

import (
	"mime/multipart"

	"github.com/gofiber/fiber/v3"
)

// CollectFormFiles, multipart formdan yüklenecek dosyaları toplar.
func CollectFormFiles(c fiber.Ctx) ([]*multipart.FileHeader, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, err
	}
	if form == nil {
		return nil, nil
	}
	return form.File["files"], nil
}
