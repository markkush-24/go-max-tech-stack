package routes

import (
	"fmt"
	"net/http"
	"pet-study/internal/entity"
	"pet-study/internal/httputils"
	"pet-study/internal/requestid"
	"pet-study/internal/security"
	"pet-study/internal/service"
	"pet-study/internal/stream"
	"pet-study/internal/transport/pb"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type JobHandler struct {
	jobService    *service.JobService
	eventHub      *stream.Hub
	heartbeat     time.Duration
	writeTimeout  time.Duration
	jobGRPCClient pb.JobsServiceClient
}

func NewJobHandler(
	jobService *service.JobService,
	eventHub *stream.Hub,
	heartbeat time.Duration,
	writeTimeout time.Duration,
	jobGRPCClient pb.JobsServiceClient,
) *JobHandler {
	return &JobHandler{
		jobService:    jobService,
		eventHub:      eventHub,
		heartbeat:     heartbeat,
		writeTimeout:  writeTimeout,
		jobGRPCClient: jobGRPCClient}
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
	ctx := r.Context()

	if reqID, ok := requestid.RequestID(ctx); ok && reqID != "" {
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("request-id", reqID))
	}

	if h.jobGRPCClient == nil {
		return httputils.ErrGRPCBridgeUnavailable
	}

	resp, err := h.jobGRPCClient.GetJob(ctx, &pb.GetJobRequest{Id: id})
	if err != nil {
		return mapGRPCError(err)
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
	subscription, unsubscribe := h.eventHub.Subscribe(int64(id))
	defer unsubscribe()

	ticker := time.NewTicker(h.heartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return nil
		case <-ticker.C:
			err := writeWithTimeout(func() error {
				if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
					return err
				}
				return rc.Flush()
			})
			if err != nil {
				return err
			}
		case event, subOk := <-subscription.C:
			if !subOk {
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
				return err
			}
		}
	}
}

func mapGRPCError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	switch st.Code() {
	case codes.InvalidArgument:
		return &httputils.BadRequestError{Detail: st.Message()}

	case codes.NotFound:
		return entity.ErrJobNotFound

	case codes.PermissionDenied:
		return security.NewForbidden(security.AuthZForbidden, nil)

	case codes.Unauthenticated:
		return security.NewUnauthorized(security.AuthNInvalid, nil)

	default:
		return err
	}
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
