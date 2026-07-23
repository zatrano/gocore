// Package pagination, liste uçları için sayfa tabanlı sayfalama sağlar.
package pagination

import (
	"math"
	"strconv"
)

// Varsayılan ve izin verilen sayfa boyutları.
const (
	DefaultLimit = 100
	MaxLimit     = 1000
)

// AllowedLimits, filtrelerde seçilebilir sayfa boyutlarıdır.
var AllowedLimits = []int{100, 500, 1000}

// Request, sayfalama isteği parametreleridir.
//
// Panel UI Page/Limit (offset) kullanır. REST API istemcileri isteğe bağlı
// Cursor ile keyset sayfalama yapabilir; Cursor doluysa Page yok sayılır (1 kabul edilir).
type Request struct {
	Page      int
	Limit     int
	Ascending bool
	Cursor    string // opak keyset imleci (API)
}

// Page, sayfalanmış bir sonucu ve meta bilgisini taşır.
type Page[T any] struct {
	Items      []T    `json:"items"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	Total      int64  `json:"total"`
	TotalPages int    `json:"total_pages"`
	NextCursor string `json:"next_cursor,omitempty"` // keyset: sonraki sayfa imleci
}

// ParsePage, query string'ten sayfa numarasını çözer (1 tabanlı).
func ParsePage(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 1
	}
	return n
}

// NormalizeLimit, limiti izin verilen değerlere çeker.
func NormalizeLimit(limit int) int {
	if limit <= 0 {
		return DefaultLimit
	}
	for _, allowed := range AllowedLimits {
		if limit == allowed {
			return allowed
		}
	}
	// En yakın izin verilen değere yuvarla (küçükten büyüğe).
	chosen := DefaultLimit
	for _, allowed := range AllowedLimits {
		if limit >= allowed {
			chosen = allowed
		}
	}
	return chosen
}

// Offset, SQL OFFSET değerini hesaplar.
func Offset(page, limit int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * limit
}

// LimitInt32, NormalizeLimit sonucunu sqlc int32 alanları için döner.
func LimitInt32(limit int) int32 {
	return clampInt32(NormalizeLimit(limit))
}

// OffsetInt32, sayfa offset'ini sqlc int32 alanları için döner.
func OffsetInt32(page, limit int) int32 {
	return clampInt32(Offset(page, limit))
}

func clampInt32(n int) int32 {
	if n < 0 {
		return 0
	}
	if n > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(n)
}

// TotalPages, toplam sayfa sayısını hesaplar.
func TotalPages(total int64, limit int) int {
	if limit <= 0 || total <= 0 {
		return 0
	}
	return int(math.Ceil(float64(total) / float64(limit)))
}

// NewPage, öğe listesi ve meta bilgisinden sayfa üretir.
func NewPage[T any](items []T, page, limit int, total int64) Page[T] {
	if page < 1 {
		page = 1
	}
	limit = NormalizeLimit(limit)
	return Page[T]{
		Items:      items,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: TotalPages(total, limit),
	}
}

// From, gösterilen ilk kaydın 1 tabanlı sırasını döner (0 = kayıt yok).
func (p Page[T]) From() int64 {
	if p.Total == 0 || len(p.Items) == 0 {
		return 0
	}
	return int64(Offset(p.Page, p.Limit)) + 1
}

// To, gösterilen son kaydın sırasını döner.
func (p Page[T]) To() int64 {
	if p.Total == 0 {
		return 0
	}
	n := int64(Offset(p.Page, p.Limit)) + int64(len(p.Items))
	if n > p.Total {
		return p.Total
	}
	return n
}

// HasPrev, önceki sayfa olup olmadığını döner.
func (p Page[T]) HasPrev() bool { return p.Page > 1 }

// HasNext, sonraki sayfa olup olmadığını döner.
func (p Page[T]) HasNext() bool { return p.Page < p.TotalPages }

// PageNumbers, sayfa numarası düğmeleri için görünür aralığı döner.
func (p Page[T]) PageNumbers(maxVisible int) []int {
	if maxVisible <= 0 {
		maxVisible = 7
	}
	if p.TotalPages <= 0 {
		return nil
	}
	if p.TotalPages <= maxVisible {
		out := make([]int, p.TotalPages)
		for i := range out {
			out[i] = i + 1
		}
		return out
	}
	half := maxVisible / 2
	start := p.Page - half
	if start < 1 {
		start = 1
	}
	end := start + maxVisible - 1
	if end > p.TotalPages {
		end = p.TotalPages
		start = end - maxVisible + 1
	}
	out := make([]int, 0, end-start+1)
	for i := start; i <= end; i++ {
		out = append(out, i)
	}
	return out
}

// VisiblePageNumbers, meta bilgisi olmadan sayfa numarası aralığı üretir.
func VisiblePageNumbers(current, totalPages, maxVisible int) []int {
	return Page[struct{}]{Page: current, TotalPages: totalPages}.PageNumbers(maxVisible)
}
