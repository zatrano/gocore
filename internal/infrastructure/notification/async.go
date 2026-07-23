// AsyncRunner, bildirim ve benzeri yan etkileri HTTP isteğinden ayırarak worker
// pool üzerinde fire-and-forget çalıştırır. İstek gövdesi gönderimin bitmesini
// beklemez; hatalar yalnızca loglanır.
package notification

import (
	"context"
	"log/slog"

	"github.com/zatrano/gocore/pkg/worker"
)

// TaskRunner, bir işi senkron veya asenkron yürütme sözleşmesidir. Üretimde
// AsyncRunner, testlerde SyncRunner kullanılır.
type TaskRunner interface {
	Go(ctx context.Context, fn func(context.Context) error) error
}

// AsyncRunner, görevleri worker pool'a kuyruklar. Kuyruk doluysa görev düşürülür
// ve uyarı loglanır.
type AsyncRunner struct {
	pool *worker.Pool
	log  *slog.Logger
}

// NewAsyncRunner, asenkron görev yürütücüsünü kurar.
func NewAsyncRunner(pool *worker.Pool, log *slog.Logger) *AsyncRunner {
	return &AsyncRunner{pool: pool, log: log}
}

// Go, fn'i arka planda çalıştırır ve hemen nil döner. fn hata verirse loglanır.
func (r *AsyncRunner) Go(ctx context.Context, fn func(context.Context) error) error {
	// İstek iptal olsa bile arka plan görevi tamamlanabilsin (trace/correlation korunur).
	detached := context.WithoutCancel(ctx)

	if !r.pool.Submit(func(_ context.Context) {
		if err := fn(detached); err != nil {
			r.log.ErrorContext(detached, "async notification task failed",
				slog.String("error", err.Error()),
			)
		}
	}) {
		r.log.WarnContext(ctx, "notification worker queue full, task dropped")
	}
	return nil
}

// SyncRunner, görevi aynı goroutine'de çalıştırır (unit testler için).
type SyncRunner struct{}

// Go, fn'i senkron yürütür.
func (SyncRunner) Go(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
