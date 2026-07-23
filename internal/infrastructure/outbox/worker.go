// Package outbox, kalıcı iş kuyruğu worker ve e-posta/bildirim/audit işleyicilerini içerir.
package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	appnotif "github.com/zatrano/gocore/internal/application/notification"
	appoutbox "github.com/zatrano/gocore/internal/application/outbox"
	appshared "github.com/zatrano/gocore/internal/application/shared"
)

// Handler, tek bir outbox işini işler.
type Handler func(ctx context.Context, job appoutbox.Job) error

// Worker, outbox_jobs tablosundan iş claim edip işler.
type Worker struct {
	repo     appoutbox.Repository
	handlers map[string]Handler
	log      *slog.Logger
	batch    int
	lease    time.Duration
	interval time.Duration
}

// NewWorker, worker'ı kurar.
func NewWorker(repo appoutbox.Repository, log *slog.Logger) *Worker {
	return &Worker{
		repo:     repo,
		handlers: make(map[string]Handler),
		log:      log,
		batch:    20,
		lease:    45 * time.Second,
		interval: 500 * time.Millisecond,
	}
}

// Register, kind için handler kaydeder.
func (w *Worker) Register(kind string, h Handler) {
	w.handlers[kind] = h
}

// Run, bgCtx iptal edilene kadar poll eder.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.tick(ctx); err != nil && ctx.Err() == nil {
				w.log.WarnContext(ctx, "outbox tick failed", slog.String("error", err.Error()))
			}
		}
	}
}

func (w *Worker) tick(ctx context.Context) error {
	jobs, err := w.repo.Claim(ctx, w.batch, w.lease)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		w.process(ctx, job)
	}
	return nil
}

func (w *Worker) process(ctx context.Context, job appoutbox.Job) {
	h, ok := w.handlers[job.Kind]
	if !ok {
		_ = w.repo.MarkDead(ctx, job.ID, job.Attempts, "unknown job kind: "+job.Kind)
		return
	}
	if err := h(ctx, job); err != nil {
		w.onFailure(ctx, job, err)
		return
	}
	if err := w.repo.MarkCompleted(ctx, job.ID); err != nil {
		w.log.ErrorContext(ctx, "outbox mark completed failed",
			slog.String("id", job.ID), slog.String("error", err.Error()))
	}
}

func (w *Worker) onFailure(ctx context.Context, job appoutbox.Job, err error) {
	msg := err.Error()
	if job.Attempts >= job.MaxAttempts {
		_ = w.repo.MarkDead(ctx, job.ID, job.Attempts, msg)
		w.log.ErrorContext(ctx, "outbox job dead-lettered",
			slog.String("id", job.ID), slog.String("kind", job.Kind), slog.String("error", msg))
		return
	}
	delay := backoff(job.Attempts)
	next := time.Now().UTC().Add(delay)
	_ = w.repo.MarkRetryable(ctx, job.ID, job.Attempts, next, msg)
	w.log.WarnContext(ctx, "outbox job retry scheduled",
		slog.String("id", job.ID), slog.String("kind", job.Kind),
		slog.Int("attempts", job.Attempts), slog.Duration("delay", delay),
		slog.String("error", msg))
}

func backoff(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	// 1s, 2s, 4s ... max 5m
	sec := math.Pow(2, float64(attempts-1))
	d := time.Duration(sec) * time.Second
	if d > 5*time.Minute {
		return 5 * time.Minute
	}
	return d
}

// EmailHandler, kind=email.send işlerini Mailer ile gönderir.
func EmailHandler(mailer appshared.Mailer) Handler {
	return func(ctx context.Context, job appoutbox.Job) error {
		var p appoutbox.EmailPayload
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return fmt.Errorf("email payload: %w", err)
		}
		return mailer.Send(ctx, appshared.Email{
			To: p.To, Subject: p.Subject, HTMLBody: p.HTMLBody, TextBody: p.TextBody,
			From: p.From, ReplyTo: p.ReplyTo,
		})
	}
}

// DispatchHandler, kind=notification.dispatch işlerini Dispatcher ile gönderir.
func DispatchHandler(dispatcher *appnotif.Dispatcher) Handler {
	return func(ctx context.Context, job appoutbox.Job) error {
		var p appoutbox.DispatchPayload
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return fmt.Errorf("dispatch payload: %w", err)
		}
		args := make([]any, len(p.Args))
		for i, a := range p.Args {
			args[i] = a
		}
		cmd := appnotif.SendCommand{
			Channel:          appnotif.Channel(p.Channel),
			UserID:           p.UserID,
			Email:            p.Email,
			Phone:            p.Phone,
			Locale:           p.Locale,
			TitleKey:         p.TitleKey,
			ContentKey:       p.ContentKey,
			HTMLContentKey:   p.HTMLContentKey,
			TitleFallback:    p.TitleFallback,
			BodyFallback:     p.BodyFallback,
			HTMLBodyFallback: p.HTMLBodyFallback,
			Args:             args,
		}
		if p.LiteralTitle != "" || p.LiteralBody != "" {
			cmd.TitleKey = ""
			cmd.ContentKey = ""
			cmd.HTMLContentKey = ""
			cmd.TitleFallback = p.LiteralTitle
			cmd.BodyFallback = p.LiteralBody
			cmd.HTMLBodyFallback = p.LiteralHTML
		}
		return dispatcher.Send(ctx, cmd)
	}
}
