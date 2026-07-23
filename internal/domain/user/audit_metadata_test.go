package user

import "testing"

func TestAuditMetadata_emailChanged(t *testing.T) {
	meta := AuditMetadata(EmailChangedEvent{
		OldEmail: "a@example.com",
		NewEmail: "b@example.com",
	})
	if meta["old_email"] != "a@example.com" || meta["new_email"] != "b@example.com" {
		t.Fatalf("unexpected metadata: %v", meta)
	}
}

func TestAuditMetadata_roleChanged(t *testing.T) {
	meta := AuditMetadata(RoleChangedEvent{OldRole: "user", NewRole: "admin"})
	if meta["old_role"] != "user" || meta["new_role"] != "admin" {
		t.Fatalf("unexpected metadata: %v", meta)
	}
}

func TestAuditMetadata_unknownEventNil(t *testing.T) {
	if AuditMetadata(ActivatedEvent{}) != nil {
		t.Fatal("expected nil for event without diff fields")
	}
}
