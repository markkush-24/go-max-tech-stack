package outbound

import (
	"errors"
	"log/slog"

	"pet-study/internal/config"
	"pet-study/internal/outbound/profile"
)

const (
	logFieldEvent               = "event"
	logFieldOperation           = "operation"
	logFieldAttempt             = "attempt"
	logFieldMaxAttempts         = "max_attempts"
	logFieldBackoffMS           = "backoff_ms"
	logFieldOutcome             = "outcome"
	logFieldErrorKind           = "error_kind"
	logFieldRetryable           = "retryable"
	logFieldDeadlineRemainingMS = "deadline_remaining_ms"

	logOperationProfileFetch = "profile.fetch"
)

func defaultOutboundLogger() *slog.Logger {
	return slog.Default().With(config.LogFieldComponent, config.LogComponentOutboundProfile)
}

func normalizeOutboundLogger(logger *slog.Logger) *slog.Logger {
	if logger != nil {
		return logger
	}
	return defaultOutboundLogger()
}

func profileErrorKind(err error) string {
	return profile.KindLabel(err)
}

func profileErrorStatus(err error) int {
	var pe *profile.Error
	if errors.As(err, &pe) && pe.Status != 0 {
		return pe.Status
	}
	if err != nil {
		return 0
	}
	return 200
}
