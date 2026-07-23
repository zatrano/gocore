package pagination_test

import (
	"testing"

	"github.com/zatrano/gocore/pkg/pagination"
)

func TestParsePage(t *testing.T) {
	if got := pagination.ParsePage(""); got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
	if got := pagination.ParsePage("3"); got != 3 {
		t.Fatalf("expected 3, got %d", got)
	}
	if got := pagination.ParsePage("-1"); got != 1 {
		t.Fatalf("expected 1, got %d", got)
	}
}

func TestNormalizeLimit(t *testing.T) {
	tests := []struct{ in, want int }{
		{0, 100},
		{100, 100},
		{500, 500},
		{1000, 1000},
		{250, 100},
		{750, 500},
		{2000, 1000},
	}
	for _, tc := range tests {
		if got := pagination.NormalizeLimit(tc.in); got != tc.want {
			t.Fatalf("NormalizeLimit(%d)=%d want %d", tc.in, got, tc.want)
		}
	}
}

func TestNewPage_meta(t *testing.T) {
	page := pagination.NewPage([]int{1, 2, 3}, 2, 100, 250)
	if page.TotalPages != 3 {
		t.Fatalf("total pages: %d", page.TotalPages)
	}
	if page.From() != 101 || page.To() != 103 {
		t.Fatalf("range: %d-%d", page.From(), page.To())
	}
	if !page.HasPrev() || !page.HasNext() {
		t.Fatal("expected prev and next")
	}
	nums := page.PageNumbers(5)
	if len(nums) != 3 {
		t.Fatalf("page numbers: %v", nums)
	}
}

func TestTotalPages_empty(t *testing.T) {
	page := pagination.NewPage([]int{}, 1, 100, 0)
	if page.From() != 0 || page.To() != 0 {
		t.Fatalf("empty range: %d-%d", page.From(), page.To())
	}
}
