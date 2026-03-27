package routes

import (
	"fmt"
	"net/http"
	"pet-study/internal/httputils"
	"pet-study/internal/security"
	"pet-study/internal/service"
	"pet-study/internal/stream"
	"time"
)

type JobHandler struct {
	jobService   *service.JobService
	eventHub     *stream.Hub
	heartbeat    time.Duration
	writeTimeout time.Duration
}

func NewJobHandler(
	jobService *service.JobService,
	eventHub *stream.Hub,
	heartbeat time.Duration,
	writeTimeout time.Duration,
) *JobHandler {
	return &JobHandler{
		jobService:   jobService,
		eventHub:     eventHub,
		heartbeat:    heartbeat,
		writeTimeout: writeTimeout}
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
