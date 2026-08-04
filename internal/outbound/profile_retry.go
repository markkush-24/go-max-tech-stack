package outbound

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"pet-study/internal/config"
	"pet-study/internal/outbound/profile"
)

type RetryingProfileClient struct {
	next        profile.Client
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
	l           *slog.Logger

	mu  sync.Mutex
	rng *rand.Rand
}

func NewRetryingProfileClient(maxAttempts int, baseDelay, maxDelay time.Duration, next profile.Client) *RetryingProfileClient {
	return NewRetryingProfileClientWithLogger(maxAttempts, baseDelay, maxDelay, next, defaultOutboundLogger())
}

func NewRetryingProfileClientWithLogger(
	maxAttempts int,
	baseDelay time.Duration,
	maxDelay time.Duration,
	next profile.Client,
	logger *slog.Logger,
) *RetryingProfileClient {
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
		l:           normalizeOutboundLogger(logger),
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (c *RetryingProfileClient) FetchProfile(ctx context.Context, userID int64, requestID string) (profile.Profile, error) {
	var lastErr error
	start := time.Now()
	attemptsMade := 0

	for attempt := 1; attempt <= c.maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			err = ctxToProfileError(err)
			c.logRetryCompleted(ctx, requestID, attempt-1, start, "failed", err)
			return profile.Profile{}, err
		}

		attemptsMade = attempt
		p, err := c.next.FetchProfile(ctx, userID, requestID)
		if err == nil {
			c.logRetryCompleted(ctx, requestID, attempt, start, "success", nil)
			return p, nil
		}
		lastErr = err

		if attempt == c.maxAttempts {
			break
		}
		if !isRetryableProfileError(err) {
			c.logRetryCompleted(ctx, requestID, attempt, start, "failed", err)
			return profile.Profile{}, err
		}

		delay := c.backoffWithJitter(attempt)
		if delay > 0 {
			if !hasTimeForDelay(ctx, delay) {
				break
			}
		}
		c.logRetryScheduled(ctx, requestID, attempt, delay, err)
		if delay > 0 {
			if err := sleepWithContext(ctx, delay); err != nil {
				err = ctxToProfileError(err)
				c.logRetryCompleted(ctx, requestID, attempt, start, "failed", err)
				return profile.Profile{}, err
			}
		}
	}

	c.logRetryCompleted(ctx, requestID, attemptsMade, start, "failed", lastErr)
	return profile.Profile{}, lastErr
}

func (c *RetryingProfileClient) logRetryScheduled(
	ctx context.Context,
	requestID string,
	attempt int,
	delay time.Duration,
	err error,
) {
	if c.l == nil || c.maxAttempts <= 1 {
		return
	}

	attrs := []slog.Attr{
		slog.String(logFieldEvent, "profile.retry.scheduled"),
		slog.String(logFieldOperation, logOperationProfileFetch),
		slog.String(config.LogFieldRequestID, requestID),
		slog.Int(logFieldAttempt, attempt),
		slog.Int(logFieldMaxAttempts, c.maxAttempts),
		slog.Int64(logFieldBackoffMS, delay.Milliseconds()),
		slog.Bool(logFieldRetryable, true),
		slog.String(logFieldErrorKind, profileErrorKind(err)),
		slog.Int(config.LogFieldStatus, profileErrorStatus(err)),
	}
	if remaining, ok := deadlineRemainingMS(ctx); ok {
		attrs = append(attrs, slog.Int64(logFieldDeadlineRemainingMS, remaining))
	}

	c.l.LogAttrs(ctx, slog.LevelInfo, "profile retry scheduled", attrs...)
}

func (c *RetryingProfileClient) logRetryCompleted(
	ctx context.Context,
	requestID string,
	attempts int,
	start time.Time,
	outcome string,
	err error,
) {
	if c.l == nil || c.maxAttempts <= 1 || (attempts <= 1 && err == nil) {
		return
	}

	attrs := []slog.Attr{
		slog.String(logFieldEvent, "profile.retry.completed"),
		slog.String(logFieldOperation, logOperationProfileFetch),
		slog.String(config.LogFieldRequestID, requestID),
		slog.String(logFieldOutcome, outcome),
		slog.Int("attempts", attempts),
		slog.Int(logFieldMaxAttempts, c.maxAttempts),
		slog.Int64(config.LogFieldDurationMS, time.Since(start).Milliseconds()),
	}
	if err != nil {
		attrs = append(attrs,
			slog.String(logFieldErrorKind, profileErrorKind(err)),
			slog.Int(config.LogFieldStatus, profileErrorStatus(err)),
			slog.Bool(logFieldRetryable, isRetryableProfileError(err)),
		)
	}
	if remaining, ok := deadlineRemainingMS(ctx); ok {
		attrs = append(attrs, slog.Int64(logFieldDeadlineRemainingMS, remaining))
	}

	c.l.LogAttrs(ctx, slog.LevelInfo, "profile retry completed", attrs...)
}

func deadlineRemainingMS(ctx context.Context) (int64, bool) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return 0, false
	}
	remaining := time.Until(deadline)
	if remaining < 0 {
		remaining = 0
	}
	return remaining.Milliseconds(), true
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
