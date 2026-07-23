package user_test

import (
	"context"
	"testing"

	appuser "github.com/zatrano/gocore/internal/application/user"
)

func TestListQuery_acceptsCursor(t *testing.T) {
	q := appuser.ListQuery{Cursor: "opaque-cursor", Page: 99}
	if q.Cursor != "opaque-cursor" {
		t.Fatalf("cursor field missing")
	}
	if q.Page != 99 {
		t.Fatal("page should remain set at query layer")
	}
}

func TestService_ListMethodExists(t *testing.T) {
	t.Parallel()
	var svc *appuser.Service
	if svc == nil {
		// Derleme zamanı yüzey kontrolü; nil receiver çağrısı yapılmaz.
		_ = (*appuser.Service).List
	}
	_ = context.Background()
}
