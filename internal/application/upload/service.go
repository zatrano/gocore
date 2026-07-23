// Package upload, güvenli çoklu dosya yükleme use-case'ini sağlar.
package upload

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/google/uuid"

	appshared "github.com/zatrano/gocore/internal/application/shared"
	domainupload "github.com/zatrano/gocore/internal/domain/upload"
	"github.com/zatrano/gocore/pkg/safefs"
)

// IncomingFile, adaptörden gelen tek bir yükleme girdisidir.
type IncomingFile struct {
	Filename string
	Size     int64
	Open     func() (io.ReadCloser, error)
}

// UploadedItem, başarıyla depolanan dosyayı temsil eder.
type UploadedItem struct {
	Key         string `json:"key"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

// FailedItem, doğrulama veya depolama hatası olan dosyayı kaydeder.
type FailedItem struct {
	Index    int    `json:"index"`
	Filename string `json:"filename,omitempty"`
	Reason   string `json:"reason"`
	Detail   string `json:"detail,omitempty"`
}

// BatchResult, çoklu yükleme özetidir.
type BatchResult struct {
	Total    int            `json:"total"`
	Accepted int            `json:"accepted"`
	Items    []UploadedItem `json:"items"`
	Invalid  []FailedItem   `json:"invalid,omitempty"`
}

// Service, dosya yükleme iş kurallarını uygular.
type Service struct {
	storage      appshared.Storage
	maxBytes     int64
	allowedMIMEs map[string]struct{}
	publisher    appshared.EventPublisher
}

// NewService, servisi kurar. publisher nil olabilir.
func NewService(storage appshared.Storage, maxBytes int64, allowedMIMEs []string, publisher appshared.EventPublisher) *Service {
	set := make(map[string]struct{}, len(allowedMIMEs))
	for _, m := range allowedMIMEs {
		set[m] = struct{}{}
	}
	return &Service{storage: storage, maxBytes: maxBytes, allowedMIMEs: set, publisher: publisher}
}

// UploadBatch, dosya listesini işler; kısmi başarı desteklenir.
func (s *Service) UploadBatch(ctx context.Context, files []IncomingFile) BatchResult {
	batchID := uuid.NewString()
	res := BatchResult{Total: len(files), Items: make([]UploadedItem, 0, len(files))}
	mimeStats := make(map[string]*domainupload.MIMEStat)
	for i, f := range files {
		item, reason, detail, err := s.uploadOne(ctx, f)
		if err != nil {
			res.Invalid = append(res.Invalid, FailedItem{
				Index: i, Filename: f.Filename, Reason: "upload.storage_failed", Detail: err.Error(),
			})
			continue
		}
		if reason != "" {
			res.Invalid = append(res.Invalid, FailedItem{
				Index: i, Filename: f.Filename, Reason: reason, Detail: detail,
			})
			continue
		}
		res.Items = append(res.Items, item)
		res.Accepted++
		stat := mimeStats[item.ContentType]
		if stat == nil {
			stat = &domainupload.MIMEStat{MIME: item.ContentType}
			mimeStats[item.ContentType] = stat
		}
		stat.Count++
		stat.Bytes += item.Size
	}
	if s.publisher != nil && res.Accepted > 0 {
		summary := make([]domainupload.MIMEStat, 0, len(mimeStats))
		for _, st := range mimeStats {
			summary = append(summary, *st)
		}
		rejected := res.Total - res.Accepted
		_ = s.publisher.Publish(ctx, domainupload.NewBatchCompletedEvent(batchID, res.Total, res.Accepted, rejected, summary))
	}
	return res
}

func (s *Service) uploadOne(ctx context.Context, f IncomingFile) (UploadedItem, string, string, error) {
	if f.Size > s.maxBytes {
		return UploadedItem{}, "upload.too_large", "", nil
	}
	src, err := f.Open()
	if err != nil {
		return UploadedItem{}, "", "", err
	}
	defer src.Close()

	head := make([]byte, 512)
	n, _ := io.ReadFull(src, head)
	head = head[:n]
	detected := http.DetectContentType(head)
	if !s.isAllowed(detected) {
		return UploadedItem{}, "upload.unsupported_type_detail", detected, nil
	}

	safeName, err := safefs.SanitizeFilename(f.Filename)
	if err != nil {
		return UploadedItem{}, "upload.invalid_name", "", nil
	}
	key := uuid.NewString() + "_" + safeName

	obj, err := s.storage.Put(ctx, key, io.MultiReader(bytes.NewReader(head), src), detected, f.Size)
	if err != nil {
		return UploadedItem{}, "", "", err
	}
	return UploadedItem{
		Key: obj.Key, Filename: f.Filename, ContentType: detected, Size: obj.Size,
	}, "", "", nil
}

func (s *Service) isAllowed(mime string) bool {
	if i := bytes.IndexByte([]byte(mime), ';'); i >= 0 {
		mime = mime[:i]
	}
	_, ok := s.allowedMIMEs[mime]
	return ok
}

// MaxBytes, tek dosya için izin verilen maksimum boyutu döner.
func (s *Service) MaxBytes() int64 { return s.maxBytes }
