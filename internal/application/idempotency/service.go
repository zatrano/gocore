package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Service, idempotent işlem yürütme sağlar.
type Service struct {
	repo Repository
	ttl  time.Duration
}

// NewService, Service'i kurar.
func NewService(repo Repository, ttl time.Duration) *Service {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &Service{repo: repo, ttl: ttl}
}

// Run, anahtar verilmişse idempotent çalıştırır; boş anahtarda doğrudan fn çağrılır.
func (s *Service) Run(ctx context.Context, scope, key, actorID, requestHash string, fn func() (any, error)) (any, error) {
	key = trim(key)
	if key == "" {
		return fn()
	}
	actorID = trim(actorID)

	existing, err := s.repo.Find(ctx, scope, key, actorID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		switch existing.Status {
		case StatusCompleted:
			if requestHash != "" && existing.RequestHash != "" && existing.RequestHash != requestHash {
				return nil, ErrConflict
			}
			return unmarshalResponse(existing.Response)
		case StatusProcessing:
			return nil, ErrInProgress
		case StatusFailed:
			// Önceki başarısız deneme — yeni denemeye izin ver (kayıt güncellenmez, yeni insert conflict verir).
			return nil, ErrInProgress
		}
	}

	id := uuid.NewString()
	expires := time.Now().UTC().Add(s.ttl)
	insertErr := s.repo.Insert(ctx, Record{
		ID: id, Scope: scope, Key: key, ActorID: actorID,
		RequestHash: requestHash, Status: StatusProcessing, ExpiresAt: expires,
	})
	if insertErr != nil {
		existing, findErr := s.repo.Find(ctx, scope, key, actorID)
		if findErr != nil {
			return nil, insertErr
		}
		if existing.Status == StatusCompleted {
			if requestHash != "" && existing.RequestHash != "" && existing.RequestHash != requestHash {
				return nil, ErrConflict
			}
			return unmarshalResponse(existing.Response)
		}
		return nil, ErrInProgress
	}

	result, fnErr := fn()
	if fnErr != nil {
		_ = s.repo.Fail(ctx, id)
		return nil, fnErr
	}
	raw, err := json.Marshal(result)
	if err != nil {
		_ = s.repo.Fail(ctx, id)
		return nil, err
	}
	if err := s.repo.Complete(ctx, id, raw); err != nil {
		return nil, err
	}
	return result, nil
}

// HashRequest, istek gövdesinden SHA-256 özeti üretir.
func HashRequest(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func unmarshalResponse(raw []byte) (any, error) {
	if len(raw) == 0 {
		return nil, errors.New("idempotency: boş yanıt")
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
