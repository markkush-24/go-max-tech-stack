package outbound

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"pet-study/internal/outbound/profile"
)

type RetryingProfileClient struct {
	next        profile.Client
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration

	mu  sync.Mutex
	rng *rand.Rand
}

func NewRetryingProfileClient(maxAttempts int, baseDelay, maxDelay time.Duration, next profile.Client) *RetryingProfileClient {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if baseDelay < 0 {
		baseDelay = 0
	}
	if maxDelay < 0 {
		maxDelay = 0
	}
	return &RetryingProfileClient{
		next:        next,
		maxAttempts: maxAttempts,
		baseDelay:   baseDelay,
		maxDelay:    maxDelay,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (c *RetryingProfileClient) FetchProfile(ctx context.Context, userID int64, requestID string) (profile.Profile, error) {
	var lastErr error

	for attempt := 1; attempt <= c.maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return profile.Profile{}, ctxToProfileError(err)
		}

		p, err := c.next.FetchProfile(ctx, userID, requestID)
		if err == nil {
			return p, nil
		}
		lastErr = err

		if attempt == c.maxAttempts {
			break
		}
		if !isRetryableProfileError(err) {
			return profile.Profile{}, err
		}

		delay := c.backoffWithJitter(attempt)
		if delay > 0 {
			if !hasTimeForDelay(ctx, delay) {
				break
			}
			if err := sleepWithContext(ctx, delay); err != nil {
				return profile.Profile{}, ctxToProfileError(err)
			}
		}
	}

	return profile.Profile{}, lastErr
}

func isRetryableProfileError(err error) bool {
	return errors.Is(err, profile.ErrTimeout) ||
		errors.Is(err, profile.ErrNetwork) ||
		errors.Is(err, profile.ErrUpstream5xx)
}

func (c *RetryingProfileClient) backoffWithJitter(attempt int) time.Duration {
	// delay = base * 2^(attempt-1), capped by maxDelay.
	delay := c.baseDelay
	for i := 1; i < attempt; i++ {
		if delay <= 0 {
			return 0
		}
		delay *= 2
	}
	if c.maxDelay > 0 && delay > c.maxDelay {
		delay = c.maxDelay
	}
	if delay <= 0 {
		return 0
	}

	// Full jitter: random in [0, delay]
	c.mu.Lock()
	n := c.rng.Int63n(int64(delay) + 1)
	c.mu.Unlock()
	return time.Duration(n)
}

func hasTimeForDelay(ctx context.Context, d time.Duration) bool {
	deadline, ok := ctx.Deadline()
	if !ok {
		return true
	}
	return time.Until(deadline) > d
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func ctxToProfileError(err error) error {
	if errors.Is(err, context.Canceled) {
		return &profile.Error{Kind: profile.ErrCanceled, Cause: err}
	}
	return &profile.Error{Kind: profile.ErrTimeout, Cause: err}
}
