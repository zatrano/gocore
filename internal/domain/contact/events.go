package contact

import "github.com/zatrano/gocore/internal/domain/shared"

const EventSubmitted = "contact.submitted"

// SubmittedEvent, iletişim formu gönderildiğinde üretilir.
type SubmittedEvent struct {
	shared.BaseEvent
	Name  string
	Email string
}

// AuditedEventNames, denetim kapsamındaki iletişim olayları.
func AuditedEventNames() []string { return []string{EventSubmitted} }
