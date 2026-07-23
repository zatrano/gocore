package goui

import (
	"context"
	"errors"
	"io"
	"mime"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/uuid"
	gouiupload "github.com/zatrano/goui/upload"

	appshared "github.com/zatrano/gocore/internal/application/shared"
)

// UploadStore adapts the application's storage port to GoUI upload.Storage.
type UploadStore struct {
	storage      appshared.Storage
	maxBytes     int64
	allowedMIMEs []string
}

func NewUploadStore(storage appshared.Storage, maxBytes int64, allowedMIMEs []string) *UploadStore {
	if maxBytes <= 0 {
		maxBytes = 8 << 20
	}
	return &UploadStore{storage: storage, maxBytes: maxBytes, allowedMIMEs: allowedMIMEs}
}

func (s *UploadStore) Save(name, contentType string, r io.Reader, size int64) (gouiupload.Meta, error) {
	if s == nil || s.storage == nil {
		return gouiupload.Meta{}, errors.New("upload storage is not configured")
	}
	if s.maxBytes > 0 && size > s.maxBytes {
		return gouiupload.Meta{}, errors.New("file too large")
	}
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(name))
	}
	if len(s.allowedMIMEs) > 0 && !mimeAllowed(s.allowedMIMEs, name, contentType) {
		return gouiupload.Meta{}, errors.New("file type is not allowed")
	}
	id := uuid.NewString()
	obj, err := s.storage.Put(context.Background(), id, io.LimitReader(r, s.maxBytes+1), contentType, size)
	if err != nil {
		return gouiupload.Meta{}, err
	}
	return gouiupload.Meta{
		ID:          id,
		Name:        safeFilename(name),
		ContentType: obj.ContentType,
		Size:        obj.Size,
		URL:         gouiupload.FilesPrefix + "/" + id,
	}, nil
}

func (s *UploadStore) Open(id string) (io.ReadCloser, gouiupload.Meta, error) {
	if s == nil || s.storage == nil {
		return nil, gouiupload.Meta{}, errors.New("upload storage is not configured")
	}
	r, obj, err := s.storage.Get(context.Background(), id)
	if err != nil {
		return nil, gouiupload.Meta{}, err
	}
	return r, gouiupload.Meta{
		ID:          id,
		Name:        id,
		ContentType: obj.ContentType,
		Size:        obj.Size,
		URL:         gouiupload.FilesPrefix + "/" + id,
	}, nil
}

func (s *UploadStore) Delete(id string) error {
	if s == nil || s.storage == nil {
		return errors.New("upload storage is not configured")
	}
	return s.storage.Delete(context.Background(), id)
}

func safeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." {
		return "file"
	}
	return strings.ReplaceAll(name, "..", "")
}

// mimeAllowed, yapılandırılmış MIME listesine veya bilinen uzantı eşleşmesine bakar.
func mimeAllowed(allowed []string, name, contentType string) bool {
	if slices.Contains(allowed, contentType) {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".csv":
		return slices.Contains(allowed, "text/csv") ||
			slices.Contains(allowed, "application/csv") ||
			slices.Contains(allowed, "text/plain")
	case ".xlsx":
		return slices.Contains(allowed, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	case ".xls":
		return slices.Contains(allowed, "application/vnd.ms-excel")
	default:
		return false
	}
}
