package idempotency_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/zatrano/gocore/internal/application/idempotency"
)

type memRepo struct {
	records map[string]idempotency.Record
}

func key(scope, k, actor string) string { return scope + "|" + k + "|" + actor }

func (m *memRepo) Insert(ctx context.Context, rec idempotency.Record) error {
	if m.records == nil {
		m.records = map[string]idempotency.Record{}
	}
	k := key(rec.Scope, rec.Key, rec.ActorID)
	if _, ok := m.records[k]; ok {
		return idempotency.ErrInProgress
	}
	m.records[k] = rec
	return nil
}

func (m *memRepo) Find(ctx context.Context, scope, k, actorID string) (idempotency.Record, error) {
	rec, ok := m.records[key(scope, k, actorID)]
	if !ok {
		return idempotency.Record{}, pgx.ErrNoRows
	}
	return rec, nil
}

func (m *memRepo) Complete(ctx context.Context, id string, response []byte) error {
	for k, rec := range m.records {
		if rec.ID == id {
			rec.Status = idempotency.StatusCompleted
			rec.Response = response
			m.records[k] = rec
			return nil
		}
	}
	return nil
}

func (m *memRepo) Fail(ctx context.Context, id string) error {
	for k, rec := range m.records {
		if rec.ID == id {
			rec.Status = idempotency.StatusFailed
			m.records[k] = rec
			return nil
		}
	}
	return nil
}

func TestServiceRun_Idempotent(t *testing.T) {
	repo := &memRepo{}
	svc := idempotency.NewService(repo, 0)
	calls := 0
	fn := func() (any, error) {
		calls++
		return map[string]string{"ok": "1"}, nil
	}
	out1, err := svc.Run(context.Background(), idempotency.ScopeNotificationSend, "k1", "actor", "", fn)
	if err != nil {
		t.Fatal(err)
	}
	out2, err := svc.Run(context.Background(), idempotency.ScopeNotificationSend, "k1", "actor", "", fn)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	b1, _ := json.Marshal(out1)
	b2, _ := json.Marshal(out2)
	if string(b1) != string(b2) {
		t.Fatalf("cached mismatch: %s vs %s", b1, b2)
	}
}
