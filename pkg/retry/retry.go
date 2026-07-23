// Package retry, geçici hatalarda üstel geri çekilme ve jitter ile yeniden deneme sağlar.
package retry

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// Options, yeniden deneme davranışını yapılandırır.
type Options struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// DefaultOptions, ödeme sağlayıcı HTTP çağrıları için makul varsayılanları döner.
func DefaultOptions() Options {
	return Options{
		MaxAttempts: 4,
		BaseDelay:   200 * time.Millisecond,
		MaxDelay:    10 * time.Second,
	}
}

type retryableError struct {
	err error
}

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }

// MarkRetryable, hatayı yeniden denenebilir olarak işaretler.
func MarkRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &retryableError{err: err}
}

// HTTPStatusRetryable, HTTP durum kodunun geçici hata olup olmadığını döner.
func HTTPStatusRetryable(status int) bool {
	return status >= 500 || status == 429
}

// Retryable, hatanın yeniden denenebilir olup olmadığını döner.
func Retryable(err error) bool {
	if err == nil {
		return false
	}
	var re *retryableError
	if errors.As(err, &re) {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	if isNetworkErr(err) {
		return true
	}
	return httpStatusFromMessage(err.Error()) >= 500 || httpStatusFromMessage(err.Error()) == 429
}

// Do, fn'i geçici hatalarda backoff+jitter ile yeniden dener.
func Do(ctx context.Context, opts Options, fn func() error) error {
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 1
	}
	if opts.BaseDelay <= 0 {
		opts.BaseDelay = 200 * time.Millisecond
	}
	if opts.MaxDelay <= 0 {
		opts.MaxDelay = 10 * time.Second
	}

	var lastErr error
	for attempt := range opts.MaxAttempts {
		if err := ctx.Err(); err != nil {
			return err
		}
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if !Retryable(lastErr) || attempt == opts.MaxAttempts-1 {
			return lastErr
		}
		delay := backoff(opts, attempt)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return lastErr
}

func backoff(opts Options, attempt int) time.Duration {
	d := opts.BaseDelay << attempt
	if d > opts.MaxDelay {
		d = opts.MaxDelay
	}
	return d + jitter(d)
}

func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	span := d / 5
	if span <= 0 {
		return 0
	}
	var buf [8]byte
	if _, err := crand.Read(buf[:]); err != nil {
		return 0
	}
	// span, MaxDelay (10s) ile sınırlıdır; mod güvenli aralıkta kalır.
	// #nosec G115 -- jitter aralığı pratik backoff tavanı içinde.
	return time.Duration(binary.BigEndian.Uint64(buf[:]) % uint64(span))
}

func isNetworkErr(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "tls handshake timeout")
}

func httpStatusFromMessage(msg string) int {
	idx := strings.Index(msg, "http ")
	if idx < 0 {
		return 0
	}
	rest := msg[idx+5:]
	end := strings.IndexByte(rest, ':')
	if end < 0 {
		end = strings.IndexByte(rest, ' ')
	}
	if end < 0 {
		return 0
	}
	var status int
	if _, err := fmt.Sscanf(rest[:end], "%d", &status); err != nil {
		return 0
	}
	return status
}
