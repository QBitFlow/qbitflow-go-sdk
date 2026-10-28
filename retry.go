package qbitflow

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"
)

const (
	// retryBaseDelay is the first back-off; attempt n (0-based) waits retryBaseDelay·2^n.
	retryBaseDelay = time.Second
	// maxRetryWait is the longest the SDK waits before retrying a 429: a longer Retry-After
	// returns the *RateLimitError at once.
	maxRetryWait = 60 * time.Second
)

// shouldRetry reports whether a failed attempt may be retried: a network error
// or timeout, a 5xx, a 429, or — for the idempotent creates — a 409 idempotency_key_in_use.
// Every other 4xx, a 3xx and a response-shape failure are final.
func shouldRetry(err error, idempotent bool) bool {
	var (
		netErr  *NetworkError
		srvErr  *ServerError
		rateErr *RateLimitError
		confErr *ConflictError
	)
	switch {
	case errors.As(err, &netErr), errors.As(err, &rateErr):
		return true
	case errors.As(err, &srvErr):
		return srvErr.StatusCode >= 500
	case errors.As(err, &confErr):
		return idempotent && confErr.Code == codeIdempotencyKeyInUse
	}
	return false
}

// retryDelay is the wait before the retry that follows attempt (0-based): 1 s·2^attempt, and
// for a 429 at least its Retry-After. ok is false when a 429's wait exceeds 60 s.
func retryDelay(err error, attempt int) (delay time.Duration, ok bool) {
	delay = retryBaseDelay << min(attempt, 30)
	var rateErr *RateLimitError
	if errors.As(err, &rateErr) {
		delay = max(delay, rateErr.RetryAfter)
		if delay > maxRetryWait {
			return 0, false
		}
	}
	return delay, true
}

// sleepContext waits for d, or until ctx is done.
func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// newUUIDv4 returns a random (version 4) UUID, the SDK's default Idempotency-Key.
func newUUIDv4() string {
	var b [16]byte
	_, _ = rand.Read(b[:])  // crypto/rand.Read never fails (it panics on an unusable source)
	b[6] = b[6]&0x0f | 0x40 // version 4
	b[8] = b[8]&0x3f | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
