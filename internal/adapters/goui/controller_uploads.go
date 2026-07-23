package goui

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	appshared "github.com/zatrano/gocore/internal/application/shared"
	appupload "github.com/zatrano/gocore/internal/application/upload"
	"github.com/zatrano/gocore/pkg/rbac"
)

// uploadedRef, GoUI /goui/upload tamamlandıktan sonra WS ile gelen dosya meta bilgisidir.
type uploadedRef struct {
	ID          string
	Name        string
	URL         string
	ContentType string
	Size        int64
}

func parseUploadRef(payload map[string]any) uploadedRef {
	fields := payloadFields(payload)
	id := firstNonEmpty(payloadString(payload, "id"), payloadString(payload, "value"))
	name := payloadString(payload, "name")
	url := payloadString(payload, "url")
	contentType := firstNonEmpty(payloadString(payload, "contentType"), payloadString(payload, "type"))

	var size int64
	sizeStr := payloadString(payload, "size")
	if sizeStr != "" {
		size, _ = strconv.ParseInt(sizeStr, 10, 64)
	} else if v, ok := fields["size"]; ok {
		switch n := v.(type) {
		case float64:
			size = int64(n)
		case int64:
			size = n
		case int:
			size = int64(n)
		}
	}
	return uploadedRef{
		ID:          id,
		Name:        name,
		URL:         url,
		ContentType: contentType,
		Size:        size,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func appendUploadRef(list []uploadedRef, ref uploadedRef, multiple bool) []uploadedRef {
	if ref.ID == "" {
		return list
	}
	if !multiple {
		return []uploadedRef{ref}
	}
	for _, existing := range list {
		if existing.ID == ref.ID {
			return list
		}
	}
	return append(list, ref)
}

func removeUploadRef(list []uploadedRef, id string) []uploadedRef {
	if id == "" {
		return list
	}
	out := make([]uploadedRef, 0, len(list))
	for _, f := range list {
		if f.ID != id {
			out = append(out, f)
		}
	}
	return out
}

func incomingFromStorage(ctx context.Context, storage appshared.Storage, refs []uploadedRef) ([]appupload.IncomingFile, error) {
	if storage == nil {
		return nil, errStorageRequired
	}
	files := make([]appupload.IncomingFile, 0, len(refs))
	for _, ref := range refs {
		ref := ref
		files = append(files, appupload.IncomingFile{
			Filename: ref.Name,
			Size:     ref.Size,
			Open: func() (io.ReadCloser, error) {
				r, _, err := storage.Get(ctx, ref.ID)
				return r, err
			},
		})
	}
	return files, nil
}

func cleanupUploadRefs(ctx context.Context, storage appshared.Storage, refs []uploadedRef) {
	if storage == nil {
		return
	}
	for _, ref := range refs {
		_ = storage.Delete(ctx, ref.ID)
	}
}

func maxUploadBytes(p *Page) int64 {
	if p.Deps.Upload != nil {
		return p.Deps.Upload.MaxBytes()
	}
	if p.Deps.MaxUpload > 0 {
		return p.Deps.MaxUpload
	}
	return 10 << 20
}

// ---------------------------------------------------------------------------
// uploads
// ---------------------------------------------------------------------------

type uploadsController struct {
	files   []uploadedRef
	result  *appupload.BatchResult
	warning string
}

func (c *uploadsController) Mount(_ context.Context, p *Page) error {
	_ = p
	return nil
}

func (c *uploadsController) Render(p *Page) (string, error) {
	maxBytes := maxUploadBytes(p)
	type resultItem struct {
		Key         string
		Filename    string
		ContentType string
		SizeStr     string
	}
	var resultItems []resultItem
	hasResult := c.result != nil && len(c.result.Items) > 0
	if hasResult {
		resultItems = make([]resultItem, 0, len(c.result.Items))
		for _, item := range c.result.Items {
			resultItems = append(resultItems, resultItem{
				Key:         item.Key,
				Filename:    item.Filename,
				ContentType: item.ContentType,
				SizeStr:     strconv.FormatInt(item.Size, 10),
			})
		}
	}
	return p.RenderView("pages.uploads", map[string]any{
		"HasResult":   hasResult,
		"ResultItems": resultItems,
		"Warning":     c.warning,
		"MaxBytes":    strconv.FormatInt(maxBytes, 10),
		"Accept":      strings.Join(p.Deps.AllowedMIMEs, ","),
		"Files":       viewUploadFiles(c.files),
	})
}

func uploadBatchErrorMessage(result appupload.BatchResult) string {
	if len(result.Invalid) == 0 {
		return "Beklenmeyen bir hata oluştu"
	}
	first := result.Invalid[0]
	switch first.Reason {
	case "upload.too_large":
		return "izin verilen boyut aşıldı"
	case "upload.unsupported_type_detail":
		if first.Detail != "" {
			return fmt.Sprintf("dosya türü (%s) izin verilmiyor", first.Detail)
		}
		return "dosya türü izin verilmiyor"
	case "upload.invalid_name":
		return "geçersiz dosya adı"
	default:
		if first.Detail != "" {
			return first.Detail
		}
		return first.Reason
	}
}

func (c *uploadsController) HandleEvent(ctx context.Context, p *Page, event string, payload map[string]any) error {
	if err := requirePagePerm(ctx, p, rbac.PermUploadsCreate); err != nil {
		return err
	}
	switch event {
	case "uploads.file.uploaded":
		c.files = appendUploadRef(c.files, parseUploadRef(payload), true)
		return nil
	case "uploads.file.remove":
		c.files = removeUploadRef(c.files, firstNonEmpty(payloadString(payload, "value"), payloadString(payload, "id")))
		return nil
	case "uploads.submit":
		c.warning = ""
		c.result = nil
		if len(c.files) == 0 {
			p.Error = errMissingUpload.Error()
			return nil
		}
		if p.Deps.Upload == nil {
			return errUploadRequired
		}
		files, err := incomingFromStorage(ctx, p.Deps.Storage, c.files)
		if err != nil {
			return err
		}
		result := p.Deps.Upload.UploadBatch(ctx, files)
		c.result = &result
		cleanupUploadRefs(ctx, p.Deps.Storage, c.files)
		c.files = nil
		if result.Accepted == 0 {
			p.Error = uploadBatchErrorMessage(result)
			return nil
		}
		if result.Total > 1 {
			p.Notice = fmt.Sprintf("dosyalar yüklendi (%d/%d)", result.Accepted, result.Total)
		} else {
			p.Notice = "dosya yüklendi"
		}
		if len(result.Invalid) > 0 {
			c.warning = fmt.Sprintf("bazı dosyalar yüklenemedi (%d/%d)", len(result.Invalid), result.Total)
		}
		return nil
	}
	return nil
}
