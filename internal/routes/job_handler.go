package routes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"pet-study/internal/config"
	"pet-study/internal/entity"
	"pet-study/internal/httputils"
	"pet-study/internal/requestid"
	"pet-study/internal/security"
	"pet-study/internal/service"
	"pet-study/internal/stream"
	"pet-study/internal/transport/pb"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"
)

type JobHandler struct {
	jobService    *service.JobService
	eventHub      *stream.Hub
	heartbeat     time.Duration
	writeTimeout  time.Duration
	jobGRPCClient pb.JobsServiceClient
	logger        *slog.Logger
}

func NewJobHandler(
	jobService *service.JobService,
	eventHub *stream.Hub,
	heartbeat time.Duration,
	writeTimeout time.Duration,
	jobGRPCClient pb.JobsServiceClient,
) *JobHandler {
	return NewJobHandlerWithLogger(jobService, eventHub, heartbeat, writeTimeout, jobGRPCClient, defaultSSELogger())
}

func NewJobHandlerWithLogger(
	jobService *service.JobService,
	eventHub *stream.Hub,
	heartbeat time.Duration,
	writeTimeout time.Duration,
	jobGRPCClient pb.JobsServiceClient,
	logger *slog.Logger,
) *JobHandler {
	return &JobHandler{
		jobService:    jobService,
		eventHub:      eventHub,
		heartbeat:     heartbeat,
		writeTimeout:  writeTimeout,
		jobGRPCClient: jobGRPCClient,
		logger:        normalizeJobHandlerLogger(logger),
	}
}

func (h *JobHandler) GetByID(w http.ResponseWriter, r *http.Request, id int) error {

	job, err := h.jobService.GetByID(r.Context(), int64(id))
	if err != nil {
		return err
	}

	principal, ok := security.FromContext(r.Context())
	if !ok {
		return security.NewUnauthorized(security.AuthNMissing, nil)
	}

	readJob := security.CanReadJob(principal, job.OwnerUserID)
	if !readJob {
		return security.NewForbidden(security.AuthZForbidden, nil)
	}

	return httputils.WriteJSON(w, http.StatusOK, job)
}

func (h *JobHandler) GetByIDViaGRPC(w http.ResponseWriter, r *http.Request, id int64) error {
	ctx, cancel := context.WithTimeout(r.Context(), httputils.GRPCBridgeCallTimeout)
	defer cancel()

	if reqID, ok := requestid.RequestID(ctx); ok && reqID != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "request-id", reqID)
	}
	if authorization := strings.TrimSpace(r.Header.Get("Authorization")); authorization != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", authorization)
	}

	if h.jobGRPCClient == nil {
		return httputils.ErrGRPCBridgeUnavailable
	}

	resp, err := h.jobGRPCClient.GetJob(ctx, &pb.GetJobRequest{Id: id})
	if err != nil {
		return httputils.MapGRPCBridgeError(err)
	}

	out := entity.JobBridgeDTO{
		ID:     resp.GetId(),
		Status: mapPBJobStatus(resp.GetStatus()),
		Source: "grpc",
	}

	return httputils.WriteJSON(w, http.StatusOK, out)
}

func (h *JobHandler) Events(w http.ResponseWriter, r *http.Request, id int) error {
	job, err := h.jobService.GetByID(r.Context(), int64(id))
	if err != nil {
		return err
	}

	principal, ok := security.FromContext(r.Context())
	if !ok {
		return security.NewUnauthorized(security.AuthNMissing, nil)
	}

	readJob := security.CanReadJob(principal, job.OwnerUserID)
	if !readJob {
		return security.NewForbidden(security.AuthZForbidden, nil)
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	rc := http.NewResponseController(w)

	writeWithTimeout := func(writeFn func() error) error {
		if h.writeTimeout > 0 {
			if err := rc.SetWriteDeadline(time.Now().Add(h.writeTimeout)); err != nil {
				return err
			}
			defer rc.SetWriteDeadline(time.Time{})
		}
		return writeFn()
	}

	_, ok = w.(http.Flusher)

	if !ok {
		return httputils.ErrStreamingUnsupported

	}
	subscription, unsubscribe, err := h.eventHub.Subscribe(int64(id))
	if err != nil {
		return err
	}
	defer unsubscribe()

	rid, _ := requestid.RequestID(r.Context())
	start := time.Now()
	closeReason := "completed"
	writeFailures := 0
	h.logger.Info(
		"sse connection opened",
		"event", "sse.connection.opened",
		config.LogFieldRequestID, rid,
		"job_id", int64(id),
	)
	defer func() {
		h.logger.Info(
			"sse connection closed",
			"event", "sse.connection.closed",
			config.LogFieldRequestID, rid,
			"job_id", int64(id),
			"reason", closeReason,
			"write_failures", writeFailures,
			config.LogFieldDurationMS, time.Since(start).Milliseconds(),
		)
	}()

	ticker := time.NewTicker(h.heartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			closeReason = sseCloseReason(r.Context().Err())
			return nil
		case <-ticker.C:
			err := writeWithTimeout(func() error {
				if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
					return err
				}
				return rc.Flush()
			})
			if err != nil {
				writeFailures++
				closeReason = "write_failed"
				h.logSSEWriteFailure(r.Context(), rid, int64(id), "heartbeat", err)
				return err
			}
		case event, subOk := <-subscription.C:
			if !subOk {
				closeReason = "subscription_closed"
				return nil
			}

			err := writeWithTimeout(func() error {
				if err := stream.WriteSSE(w, stream.Event{
					Data: event.Data,
					Type: event.Type,
				}); err != nil {
					return err
				}
				return rc.Flush()
			})
			if err != nil {
				writeFailures++
				closeReason = "write_failed"
				h.logSSEWriteFailure(r.Context(), rid, int64(id), "event", err)
				return err
			}
		}
	}
}

func (h *JobHandler) logSSEWriteFailure(ctx context.Context, requestID string, jobID int64, phase string, err error) {
	h.logger.Warn(
		"sse write failed",
		"event", "sse.write.failed",
		config.LogFieldRequestID, requestID,
		"job_id", jobID,
		"phase", phase,
		"error_kind", sseErrorKind(err),
	)
}

func defaultSSELogger() *slog.Logger {
	return slog.Default().With(config.LogFieldComponent, "sse")
}

func normalizeJobHandlerLogger(logger *slog.Logger) *slog.Logger {
	if logger != nil {
		return logger
	}
	return defaultSSELogger()
}

func sseCloseReason(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "client_closed"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	default:
		return "context_done"
	}
}

func sseErrorKind(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "context_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	}

	var timeoutErr interface{ Timeout() bool }
	if errors.As(err, &timeoutErr) && timeoutErr.Timeout() {
		return "timeout"
	}
	return "write_error"
}

func mapPBJobStatus(s pb.JobStatus) string {
	switch s {
	case pb.JobStatus_JOB_STATUS_QUEUED:
		return "queued"
	case pb.JobStatus_JOB_STATUS_RUNNING:
		return "running"
	case pb.JobStatus_JOB_STATUS_SUCCEEDED:
		return "succeeded"
	case pb.JobStatus_JOB_STATUS_FAILED:
		return "failed"
	default:
		return "unknown"
	}
}
