// Package worker, sınırlı eşzamanlılıkla arka plan işlerini yürüten bir worker
// pool ve periyodik job zamanlayıcı sağlar. Graceful shutdown destekler.
package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Task, worker pool tarafından yürütülecek iş birimidir.
type Task func(ctx context.Context)

// Pool, sabit sayıda worker ile görevleri kuyruğa alıp yürütür. Kuyruk dolu
// olduğunda Submit bloklanır (backpressure); bu, sınırsız bellek büyümesini
// (allocation patlaması) önler.
type Pool struct {
	tasks chan Task
	wg    sync.WaitGroup
	log   *slog.Logger
	once  sync.Once
}

// NewPool, workerCount worker ve queueSize kapasiteli kuyrukla pool oluşturur
// ve worker'ları başlatır.
func NewPool(ctx context.Context, workerCount, queueSize int, log *slog.Logger) *Pool {
	if workerCount <= 0 {
		workerCount = 1
	}
	p := &Pool{
		tasks: make(chan Task, queueSize),
		log:   log,
	}
	for range workerCount {
		p.wg.Add(1)
		go p.worker(ctx)
	}
	return p
}

func (p *Pool) worker(ctx context.Context) {
	defer p.wg.Done()
	for task := range p.tasks {
		p.runSafely(ctx, task)
	}
}

// runSafely, tek bir görevi panic'e karşı koruyarak çalıştırır (bir görevin
// paniklemesi tüm worker'ı düşürmesin).
func (p *Pool) runSafely(ctx context.Context, task Task) {
	defer func() {
		if r := recover(); r != nil {
			p.log.Error("worker task panicked", slog.Any("panic", r))
		}
	}()
	task(ctx)
}

// Submit, bir görevi kuyruğa ekler. Kuyruk kapalıysa false döner.
func (p *Pool) Submit(task Task) (ok bool) {
	defer func() {
		// Kapalı kanala göndermeyi güvenli hale getir.
		if recover() != nil {
			ok = false
		}
	}()
	p.tasks <- task
	return true
}

// Shutdown, yeni görev kabulünü durdurur ve kuyruktaki görevlerin bitmesini
// (veya ctx iptalini) bekler. Graceful shutdown için çağrılır.
func (p *Pool) Shutdown(ctx context.Context) error {
	p.once.Do(func() { close(p.tasks) })

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Scheduler, bir fonksiyonu belirli aralıklarla çalıştıran basit zamanlayıcıdır
// (ör. cache/guard temizliği). ctx iptal edilince durur.
func Scheduler(ctx context.Context, interval time.Duration, fn func(context.Context)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fn(ctx)
		}
	}
}
