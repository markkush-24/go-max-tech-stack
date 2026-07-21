package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"pet-study/internal/entity"
	"pet-study/internal/httputils"
	"pet-study/internal/metrics"
	"pet-study/internal/queue"
	"pet-study/internal/security"
	"pet-study/internal/service"
	"pet-study/internal/stream"
	"pet-study/internal/transport/pb"
	"time"
)

const asyncEnqueueRollbackTimeout = 2 * time.Second

type UsersHandler struct {
	userService  *service.UserService
	jobService   *service.JobService
	workerQueue  *queue.Queue
	jobsObserver metrics.JobsObserver
	eventHub     *stream.Hub
}

func NewUserHandler(
	userService *service.UserService,
	jobService *service.JobService,
	workerQueue *queue.Queue,
	metrics metrics.JobsObserver,
	eventHub *stream.Hub,
) *UsersHandler {
	return &UsersHandler{
		userService:  userService,
		jobService:   jobService,
		workerQueue:  workerQueue,
		jobsObserver: metrics,
		eventHub:     eventHub}
}

func (h *UsersHandler) Create(w http.ResponseWriter, r *http.Request) error {
	httputils.LimitBody(w, r, 64<<10)

	if err := httputils.RequireJSONContentType(r); err != nil {
		return err
	}

	qs := r.URL.Query().Get("async")
	if qs == "1" {
		var in entity.CreateUserInput
		if err := httputils.ParseJSON(r.Body, &in); err != nil {
			return err
		}

		if details := httputils.ValidateCreateUserInput(in); len(details) > 0 {
			return &httputils.ValidationError{
				InvalidParams: httputils.ToInvalidParams(details),
			}
		}

		principal, ok := security.FromContext(r.Context())
		if !ok {
			return security.NewUnauthorized(security.AuthNMissing, nil)
		}

		job := entity.Job{Status: entity.JobQueued, OwnerUserID: principal.UserID}
		if err := h.jobService.Save(r.Context(), &job); err != nil {
			return err
		}

		publishQueuedJob(h.eventHub, h.jobsObserver, job.ID)

		if enqueueErr := h.workerQueue.Enqueue(r.Context(), queue.WorkItem{JobID: job.ID, Payload: in}); enqueueErr != nil {
			deleteErr := deleteJobAfterFailedEnqueue(r.Context(), h.jobService, job.ID)
			if deleteErr != nil {
				return errors.Join(enqueueErr, deleteErr)
			}
			return enqueueErr
		}

		w.Header().Set("Location", fmt.Sprintf("/api/v1/jobs/%d", job.ID))
		return httputils.WriteJSON(w, http.StatusAccepted, job)
	} else {
		var in entity.CreateUserInput
		if err := httputils.ParseJSON(r.Body, &in); err != nil {
			return err
		}

		if details := httputils.ValidateCreateUserInput(in); len(details) > 0 {
			return &httputils.ValidationError{
				InvalidParams: httputils.ToInvalidParams(details),
			}
		}

		u, err := h.userService.CreateUser(r.Context(), &in)
		if err != nil {
			return err
		}

		w.Header().Set("Location", fmt.Sprintf("/api/v1/users/%d", u.ID))
		return httputils.WriteJSON(w, http.StatusCreated, u)
	}
}

func (h *UsersHandler) GetByID(w http.ResponseWriter, r *http.Request, id int) error {
	principal, ok := security.FromContext(r.Context())
	if !ok {
		return security.NewUnauthorized(security.AuthNMissing, nil)
	}

	readUser := security.CanReadUser(principal, int64(id))
	if !readUser {
		return security.NewForbidden(security.AuthZForbidden, nil)
	}
	u, err := h.userService.GetByID(r.Context(), id)
	if err != nil {
		return err
	}

	etag := httputils.UserETag(u.ID, u.Version)
	w.Header().Set("ETag", etag)

	if httputils.IfNoneMatchMatches(r.Header.Get("If-None-Match"), etag) {
		httputils.AddVary(w, "Accept")
		w.Header().Set("Cache-Control", httputils.CacheControlParams)
		w.WriteHeader(http.StatusNotModified)
		return nil
	}

	w.Header().Set("Cache-Control", httputils.CacheControlParams)
	return httputils.WriteNegotiated(w, r, http.StatusOK,
		entity.UserDTO{
			ID:    u.ID,
			Name:  u.Name,
			Email: u.Email,
		},
		&pb.User{
			Id:    int64(u.ID),
			Name:  u.Name,
			Email: u.Email,
		},
	)
}

func (h *UsersHandler) Export(w http.ResponseWriter, r *http.Request, id int) error {
	principal, ok := security.FromContext(r.Context())
	if !ok {
		return security.NewUnauthorized(security.AuthNMissing, nil)
	}

	readUser := security.CanReadUser(principal, int64(id))
	if !readUser {
		return security.NewForbidden(security.AuthZForbidden, nil)
	}

	u, err := h.userService.GetByID(r.Context(), id)
	if err != nil {
		return err
	}

	out := entity.UserExport{
		ID:    int64(u.ID),
		Name:  u.Name,
		Email: u.Email,
		Age:   u.Age,
	}

	data, err := json.Marshal(out)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="user-%d-export.json"`, id))
	w.Header().Set("Content-Type", "application/json")

	reader := bytes.NewReader(data)
	http.ServeContent(w, r, fmt.Sprintf("user-%d-export.json", id), time.Time{}, reader)
	return nil
}

func (h *UsersHandler) List(w http.ResponseWriter, r *http.Request) error {
	users, err := h.userService.GetAllUsers(r.Context())
	if err != nil {
		return err
	}

	usersDtos := make([]entity.UserDTO, 0, len(users))
	for _, u := range users {
		usersDtos = append(usersDtos, entity.UserDTO{
			ID:    u.ID,
			Name:  u.Name,
			Email: u.Email,
		})
	}

	return httputils.WriteJSON(w, http.StatusOK, usersDtos)
}

func deleteJobAfterFailedEnqueue(ctx context.Context, jobService *service.JobService, id int64) error {
	rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), asyncEnqueueRollbackTimeout)
	defer cancel()

	return jobService.Delete(rollbackCtx, id)
}

func publishQueuedJob(eventHub *stream.Hub, jobsObserver metrics.JobsObserver, jobID int64) {
	jobsObserver.IncQueued()
	eventHub.Publish(jobID, stream.Event{
		Type:  string(entity.JobQueued),
		JobID: jobID,
		At:    time.Now(),
	})
}
