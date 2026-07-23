package rbac

import "regexp"

// RoleName, sistemde tanımlı bir rolün benzersiz adını temsil eder.
type RoleName string

// PermissionName, izin kataloğundaki atomik yetki adını temsil eder.
type PermissionName string

// Sistem rolleri: her kurulumda mevcut olmalı ve silinemez.
const (
	RoleAdmin RoleName = "admin"
	RoleUser  RoleName = "user"
)

var roleNameRe = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,31}$`)

// modül:eylem veya modül:alt:eylem (ör. users:role:change, payments:list)
var permissionNameRe = regexp.MustCompile(`^[a-z][a-z0-9_-]*(:[a-z][a-z0-9_-]*)+$`)

// ParseRoleName, rol adının biçimini doğrular.
func ParseRoleName(raw string) (RoleName, error) {
	if !roleNameRe.MatchString(raw) {
		return "", ErrInvalidRoleName
	}
	return RoleName(raw), nil
}

// ParsePermissionName, izin adının biçimini doğrular (modül:eylem).
func ParsePermissionName(raw string) (PermissionName, error) {
	if !permissionNameRe.MatchString(raw) {
		return "", ErrInvalidPermissionName
	}
	return PermissionName(raw), nil
}

func (n RoleName) String() string       { return string(n) }
func (n PermissionName) String() string { return string(n) }
