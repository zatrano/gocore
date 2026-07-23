package handler

import (
	"io"

	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/gocore/internal/adapters/http/render"
	adapters "github.com/zatrano/gocore/internal/adapters/shared"
	appupload "github.com/zatrano/gocore/internal/application/upload"
)

// UploadHandler, güvenli dosya yükleme sağlar.
type UploadHandler struct {
	upload *appupload.Service
}

// NewUploadHandler, handler'ı kurar.
func NewUploadHandler(upload *appupload.Service) *UploadHandler {
	return &UploadHandler{upload: upload}
}

// Upload, POST /api/v1/uploads — multipart form üzerinden bir veya birden fazla dosya yükler.
func (h *UploadHandler) Upload(c fiber.Ctx) error {
	headers, err := adapters.CollectFormFiles(c)
	if err != nil || len(headers) == 0 {
		return render.ProblemLocalized(c, 400, "upload.missing_files",
			"title.validation", "Geçersiz istek",
			"upload.missing_files", "form'da en az bir dosya bekleniyor")
	}

	files := make([]appupload.IncomingFile, 0, len(headers))
	for _, header := range headers {
		h := header
		files = append(files, appupload.IncomingFile{
			Filename: h.Filename,
			Size:     h.Size,
			Open: func() (io.ReadCloser, error) {
				return h.Open()
			},
		})
	}

	result := h.upload.UploadBatch(c.Context(), files)
	if result.Accepted == 0 {
		first := result.Invalid[0]
		switch first.Reason {
		case "upload.too_large":
			return render.ProblemLocalized(c, 413, "upload.too_large",
				"title.payload_too_large", "Dosya çok büyük",
				"upload.too_large", "izin verilen boyut aşıldı")
		case "upload.unsupported_type_detail":
			return render.ProblemLocalized(c, 415, "upload.unsupported_type",
				"title.unsupported_media", "Desteklenmeyen tür",
				"upload.unsupported_type_detail", "dosya türü ("+first.Detail+") izin verilmiyor", first.Detail)
		case "upload.invalid_name":
			return render.ProblemLocalized(c, 400, "upload.invalid_name",
				"title.validation", "Geçersiz istek",
				"upload.invalid_name", "geçersiz dosya adı")
		default:
			return render.ProblemLocalized(c, 500, "internal_error",
				"title.internal", "Sunucu hatası",
				"internal_error", "Beklenmeyen bir hata oluştu")
		}
	}

	msgKey := "success.upload.completed"
	msgFallback := "dosya yüklendi"
	if result.Total > 1 {
		msgKey = "success.upload.batch_completed"
		msgFallback = "dosyalar yüklendi"
	}

	return render.JSON(c, fiber.StatusCreated, fiber.Map{
		"message":  render.MessageText(c, msgKey, msgFallback, result.Accepted, result.Total),
		"total":    result.Total,
		"accepted": result.Accepted,
		"items":    result.Items,
		"invalid":  result.Invalid,
	})
}
