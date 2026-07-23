package audit

import "github.com/zatrano/gocore/internal/domain/shared"

// ErrNotFound, denetim kaydı bulunamadığında döner.
var ErrNotFound = shared.NewDomainError(
	shared.KindNotFound, "audit.not_found", "denetim kaydı bulunamadı")
