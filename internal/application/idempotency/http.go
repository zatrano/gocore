package idempotency

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RunHTTP, API mutasyonlarını idempotent çalıştırır. cached=true ise fn çağrılmadı.
func (s *Service) RunHTTP(
	ctx context.Context, scope, key, actorID, requestHash string,
	fn func() (*HTTPStoredResponse, error),
) (cached bool, resp *HTTPStoredResponse, err error) {
	key = trim(key)
	if key == "" {
		out, fnErr := fn()
		return false, out, fnErr
	}
	actorID = trim(actorID)

	if existing, findErr := s.repo.Find(ctx, scope, key, actorID); findErr == nil {
		switch existing.Status {
		case StatusCompleted:
			if requestHash != "" && existing.RequestHash != "" && existing.RequestHash != requestHash {
				return false, nil, ErrConflict
			}
			h, uErr := UnmarshalHTTPStoredResponse(existing.Response)
			if uErr != nil {
				return false, nil, uErr
			}
			return true, &h, nil
		case StatusProcessing, StatusFailed:
			return false, nil, ErrInProgress
		}
	} else if !errors.Is(findErr, pgx.ErrNoRows) {
		return false, nil, findErr
	}

	id := uuid.NewString()
	expires := time.Now().UTC().Add(s.ttl)
	insertErr := s.repo.Insert(ctx, Record{
		ID: id, Scope: scope, Key: key, ActorID: actorID,
		RequestHash: requestHash, Status: StatusProcessing, ExpiresAt: expires,
	})
	if insertErr != nil {
		if existing, findErr := s.repo.Find(ctx, scope, key, actorID); findErr == nil && existing.Status == StatusCompleted {
			if requestHash != "" && existing.RequestHash != "" && existing.RequestHash != requestHash {
				return false, nil, ErrConflict
			}
			h, uErr := UnmarshalHTTPStoredResponse(existing.Response)
			if uErr != nil {
				return false, nil, uErr
			}
			return true, &h, nil
		}
		return false, nil, ErrInProgress
	}

	out, fnErr := fn()
	if fnErr != nil {
		_ = s.repo.Fail(ctx, id)
		return false, nil, fnErr
	}
	if out == nil {
		_ = s.repo.Fail(ctx, id)
		return false, nil, errors.New("idempotency: boş HTTP yanıtı")
	}
	if out.StatusCode >= 500 {
		_ = s.repo.Fail(ctx, id)
		return false, out, nil
	}
	raw, mErr := out.Marshal()
	if mErr != nil {
		_ = s.repo.Fail(ctx, id)
		return false, nil, mErr
	}
	if cErr := s.repo.Complete(ctx, id, raw); cErr != nil {
		return false, nil, cErr
	}
	return false, out, nil
}
