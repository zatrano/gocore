// Package safefs, dosya sistemi güvenliği yardımcıları sağlar: path traversal,
// zip slip ve tehlikeli dosya adlarına karşı koruma.
package safefs

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// ErrPathTraversal, hedef yol taban dizinin dışına çıkarsa döner.
	ErrPathTraversal = errors.New("safefs: path traversal tespit edildi")
	// ErrEmptyName, sanitize sonrası ad boş kalırsa döner.
	ErrEmptyName = errors.New("safefs: geçersiz dosya adı")

	// unsafeChars, dosya adında izin verilmeyen karakterler.
	unsafeChars = regexp.MustCompile(`[^\w.\-]+`)
)

// SanitizeFilename, kullanıcıdan gelen dosya adını güvenli hale getirir:
// dizin bileşenlerini atar, tehlikeli karakterleri temizler ve gizli/çift
// nokta durumlarını engeller. Zip slip ve path traversal'a karşı ilk savunmadır.
func SanitizeFilename(name string) (string, error) {
	// Yol bileşenlerini at (a/b/../c → c). Hem / hem \ ele alınır.
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)
	name = strings.TrimSpace(name)

	// Baştaki noktaları temizle (".." ve gizli dosyalar).
	name = strings.TrimLeft(name, ".")

	// Tehlikeli karakterleri alt çizgiyle değiştir.
	name = unsafeChars.ReplaceAllString(name, "_")

	if name == "" || name == "." || name == ".." {
		return "", ErrEmptyName
	}
	return name, nil
}

// SafeJoin, baseDir ile untrusted relative path'i güvenle birleştirir. Sonuç
// baseDir'in DIŞINA çıkıyorsa ErrPathTraversal döner. Arşiv çıkarırken (zip
// slip) ve upload yollarında kullanılmalıdır.
func SafeJoin(baseDir, untrusted string) (string, error) {
	cleanBase, err := filepath.Abs(filepath.Clean(baseDir))
	if err != nil {
		return "", err
	}
	// filepath.Join, ".." bileşenlerini çözer; sonuç taban dizinin dışına
	// çıkarsa aşağıdaki prefix kontrolü bunu yakalar (traversal → hata).
	absJoined, err := filepath.Abs(filepath.Join(cleanBase, untrusted))
	if err != nil {
		return "", err
	}

	// Hedef, taban dizinin altında mı? (separator eklenerek prefix aldatması önlenir)
	if absJoined != cleanBase && !strings.HasPrefix(absJoined, cleanBase+string(filepath.Separator)) {
		return "", ErrPathTraversal
	}
	return absJoined, nil
}
