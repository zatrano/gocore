// Package datetime, kullanıcıya gösterilen tarih/saat biçimlendirmesini merkezileştirir.
// Veritabanı ve iş mantığı UTC/timestamptz kullanmaya devam eder; yalnızca
// sunum katmanında Europe/Istanbul saat dilimine çevrilir.
package datetime

import (
	"encoding/json"
	"time"
)

const (
	LayoutDateTime      = "02.01.2006 15:04:05"
	LayoutDateTimeShort = "02.01.2006 15:04"
	displayTZ           = "Europe/Istanbul"
)

var displayLoc = mustLoadLocation(displayTZ)

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("TRT", 3*3600)
	}
	return loc
}

// DisplayLocation, panel ve API gösterimlerinde kullanılan saat dilimidir.
func DisplayLocation() *time.Location { return displayLoc }

// InDisplay, anı Türkiye saat dilimine çevirir.
func InDisplay(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return t.In(displayLoc)
}

// FormatDateTime, tam tarih-saat döner (02.01.2006 15:04:05).
func FormatDateTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return InDisplay(t).Format(LayoutDateTime)
}

// FormatDateTimeShort, kısa tarih-saat döner (02.01.2006 15:04).
func FormatDateTimeShort(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return InDisplay(t).Format(LayoutDateTimeShort)
}

// FormatRFC3339, API yanıtları için Türkiye saat diliminde RFC3339 üretir.
func FormatRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return InDisplay(t).Format(time.RFC3339)
}

// JSONTime, JSON API'de zamanı Türkiye saat dilimiyle serileştirir.
type JSONTime struct {
	time.Time
}

// FromTime, JSONTime oluşturur.
func FromTime(t time.Time) JSONTime { return JSONTime{Time: t} }

// PtrFromTime, nil güvenli *JSONTime üretir.
func PtrFromTime(t *time.Time) *JSONTime {
	if t == nil || t.IsZero() {
		return nil
	}
	jt := FromTime(*t)
	return &jt
}

// MarshalJSON, zamanı RFC3339 (+03:00) olarak yazar.
func (t JSONTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(FormatRFC3339(t.Time))
}

// UnmarshalJSON, RFC3339 veya panel formatını okur.
func (t *JSONTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		t.Time = time.Time{}
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if parsed, err := time.Parse(time.RFC3339, s); err == nil {
		t.Time = parsed.UTC()
		return nil
	}
	if parsed, err := time.Parse(LayoutDateTime, s); err == nil {
		t.Time = parsed.In(displayLoc).UTC()
		return nil
	}
	parsed, err := time.Parse(LayoutDateTimeShort, s)
	if err != nil {
		return err
	}
	t.Time = parsed.In(displayLoc).UTC()
	return nil
}
