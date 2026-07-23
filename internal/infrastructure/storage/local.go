// Package storage, appshared.Storage portunun implementasyonlarını içerir.
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	appshared "github.com/zatrano/gocore/internal/application/shared"
	"github.com/zatrano/gocore/pkg/safefs"
)

// Local, yerel dosya sistemi tabanlı depolama. Tüm yollar taban dizine
// hapsedilir (path traversal / zip slip koruması). Üretimde S3/GCS adaptörüyle
// değiştirilebilir (port aynı kalır).
type Local struct {
	root *os.Root
}

// NewLocal, taban dizini oluşturarak storage'ı kurar.
func NewLocal(baseDir string) (*Local, error) {
	if err := os.MkdirAll(baseDir, 0o750); err != nil {
		return nil, fmt.Errorf("storage: taban dizin oluşturulamadı: %w", err)
	}
	root, err := os.OpenRoot(baseDir)
	if err != nil {
		return nil, fmt.Errorf("storage: root açılamadı: %w", err)
	}
	return &Local{root: root}, nil
}

func relativeKey(key string) (string, error) {
	if key == "" {
		return "", safefs.ErrEmptyName
	}
	key = strings.ReplaceAll(key, `\`, "/")
	clean := path.Clean(key)
	if clean == "." || strings.HasPrefix(clean, "..") || strings.HasPrefix(clean, "/") {
		return "", safefs.ErrPathTraversal
	}
	return clean, nil
}

// Put, veriyi güvenli bir yola yazar.
func (l *Local) Put(_ context.Context, key string, r io.Reader, contentType string, size int64) (appshared.FileObject, error) {
	rel, err := relativeKey(key)
	if err != nil {
		return appshared.FileObject{}, err
	}
	if dir := path.Dir(rel); dir != "." {
		if err := l.root.Mkdir(dir, 0o750); err != nil && !os.IsExist(err) {
			return appshared.FileObject{}, err
		}
	}

	f, err := l.root.OpenFile(rel, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return appshared.FileObject{}, err
	}
	defer f.Close()

	written, err := io.Copy(f, r)
	if err != nil {
		return appshared.FileObject{}, err
	}
	return appshared.FileObject{Key: key, ContentType: contentType, Size: written}, nil
}

// Get, veriyi güvenli yoldan okur.
func (l *Local) Get(_ context.Context, key string) (io.ReadCloser, appshared.FileObject, error) {
	rel, err := relativeKey(key)
	if err != nil {
		return nil, appshared.FileObject{}, err
	}
	f, err := l.root.Open(rel)
	if err != nil {
		return nil, appshared.FileObject{}, err
	}
	stat, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, appshared.FileObject{}, err
	}
	return f, appshared.FileObject{Key: key, Size: stat.Size()}, nil
}

// Delete, dosyayı güvenli yoldan siler.
func (l *Local) Delete(_ context.Context, key string) error {
	rel, err := relativeKey(key)
	if err != nil {
		return err
	}
	if err := l.root.Remove(rel); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
