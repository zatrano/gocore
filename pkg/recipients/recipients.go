// Package recipients, toplu bildirim/SMS/e-posta için alıcı listelerini
// CSV ve Excel (xlsx) dosyalarından ayrıştırır.
//
// Bu paket iletişim formu modülü değildir. Form mesajları
// internal/domain/contact + application/contact altındadır.
package recipients

import (
	"errors"
	"strings"
)

// Row, dosyadan ayrıştırılmış tek alıcı satırıdır. Hangi alanların dolu olması
// gerektiği kullanım kanalına (inapp/sms/email) göre çağıran tarafça belirlenir.
type Row struct {
	UserID string
	Name   string
	Email  string
	Phone  string
	Locale string
	// Line, kaynak dosyadaki 1 tabanlı satır numarasıdır (başlık = 1).
	Line int
}

var (
	// ErrEmptyFile, dosyada veri satırı yok (yalnız başlık veya tamamen boş).
	ErrEmptyFile = errors.New("recipients: dosya boş veya yalnızca başlık içeriyor")
	// ErrNoRecognizedColumns, başlık satırında bilinen sütun bulunamadı.
	ErrNoRecognizedColumns = errors.New("recipients: tanınan sütun yok (user_id, name, email, phone, locale)")
	// ErrUnsupportedFormat, dosya uzantısı desteklenmiyor.
	ErrUnsupportedFormat = errors.New("recipients: desteklenmeyen dosya biçimi (yalnızca .csv, .xlsx)")
	// ErrTooManyRows, satır sayısı üst sınırı aştı.
	ErrTooManyRows = errors.New("recipients: satır sayısı üst sınırı aşıldı")
)

// MaxRows, tek bir dosyadan ayrıştırılacak azami veri satırı sayısıdır.
const MaxRows = 50_000

type columnIndex struct {
	userID int
	name   int
	email  int
	phone  int
	locale int
}

func newColumnIndex(header []string) (columnIndex, bool) {
	idx := columnIndex{userID: -1, name: -1, email: -1, phone: -1, locale: -1}
	found := false
	for i, raw := range header {
		switch normalizeHeader(raw) {
		case "user_id", "userid", "id", "kullanici_id":
			idx.userID = i
			found = true
		case "name", "ad", "isim", "ad_soyad", "adsoyad":
			idx.name = i
			found = true
		case "email", "e-mail", "e_mail", "eposta", "e-posta", "mail":
			idx.email = i
			found = true
		case "phone", "telefon", "gsm", "tel", "phone_number", "telefon_no":
			idx.phone = i
			found = true
		case "locale", "dil", "lang", "language":
			idx.locale = i
			found = true
		}
	}
	return idx, found
}

func (idx columnIndex) row(cells []string, line int) Row {
	return Row{
		UserID: at(cells, idx.userID),
		Name:   at(cells, idx.name),
		Email:  at(cells, idx.email),
		Phone:  at(cells, idx.phone),
		Locale: at(cells, idx.locale),
		Line:   line,
	}
}

func at(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func normalizeHeader(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimPrefix(s, "\ufeff")
	return s
}

func isEmptyRow(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}
