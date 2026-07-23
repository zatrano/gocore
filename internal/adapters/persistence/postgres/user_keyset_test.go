package postgres_test

import (
	"testing"

	"github.com/zatrano/gocore/internal/adapters/persistence/postgres/db"
)

// Keyset sorgularının sqlc tarafından üretildiğini derleme zamanında doğrular.
func TestUsersKeysetQueriesGenerated(t *testing.T) {
	t.Parallel()
	var q db.Queries
	_ = q.ListUsersDescKeyset
	_ = q.ListUsersAscKeyset
}
