package shared

import (
	"net/url"
	"strconv"

	"github.com/zatrano/gocore/pkg/pagination"
)

// ParsePage, query string'ten sayfa numarasını çözer.
func ParsePage(s string) int {
	return pagination.ParsePage(s)
}

// ParseLimit, query string'ten sayfa boyutunu güvenle çözer (hata → 0, default).
func ParseLimit(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return pagination.NormalizeLimit(n)
}

// ListURL, filtre + sayfalama parametreleriyle liste URL'si üretir.
func ListURL(path string, params map[string]string, page, limit int) string {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = pagination.DefaultLimit
	}
	q := url.Values{}
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	q.Set("page", strconv.Itoa(page))
	q.Set("limit", strconv.Itoa(limit))
	if enc := q.Encode(); enc != "" {
		return path + "?" + enc
	}
	return path
}
