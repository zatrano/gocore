package user

import "context"

// RoleChecker, bir rol adının rbac domain'inde tanımlı olup olmadığını doğrulayan
// porttur. authz.RoleExistsChecker bu arayüzü gerçekler.
type RoleChecker interface {
	RoleExists(ctx context.Context, role string) (bool, error)
}
