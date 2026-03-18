package routes

import (
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
	"time"
)

type UsersV2Handler struct {
	userService  *service.UserService
	jobService   *service.JobService
	workerQueue  *queue.Queue
	jobsObserver metrics.JobsObserver
	eventHub     *stream.Hub
}

func NewUserV2Handler(
	userService *service.UserService,
	jobService *service.JobService,
	workerQueue *queue.Queue,
	metrics metrics.JobsObserver,
	eventHub *stream.Hub,
) *UsersV2Handler {
	return &UsersV2Handler{
		userService:  userService,
		jobService:   jobService,
		workerQueue:  workerQueue,
		jobsObserver: metrics,
		eventHub:     eventHub}
}

func (h *UsersV2Handler) Create(w http.ResponseWriter, r *http.Request) error {
	httputils.LimitBody(w, r, 64<<10)

	if err := httputils.RequireJSONContentType(r); err != nil {
		return err
	}
	qs := r.URL.Query().Get("async")
	if qs == "1" {
		var in entity.CreateUserInputV2
		if err := httputils.ParseJSON(r.Body, &in); err != nil {
			return err
		}

		if details := httputils.ValidateCreateUserInputV2(in); len(details) > 0 {
			return &httputils.ValidationError{
				InvalidParams: httputils.ToInvalidParams(details),
			}
		}

		inV1 := entity.MapCreateV2ToV1(in)
		job := entity.Job{Status: entity.JobQueued}
		if err := h.jobService.Save(r.Context(), &job); err != nil {
			return err
		}
		item := queue.WorkItem{JobID: job.ID, Payload: inV1}
		if enqueueErr := h.workerQueue.Enqueue(r.Context(), item); enqueueErr != nil {
			deleteErr := h.jobService.Delete(r.Context(), job.ID)
			if deleteErr != nil {
				return errors.Join(enqueueErr, deleteErr)
			}
			return enqueueErr
		}
		h.jobsObserver.IncQueued()

		h.eventHub.Publish(job.ID, stream.Event{
			Type:  string(entity.JobQueued),
			JobID: job.ID,
			At:    time.Now(),
		})

		w.Header().Set("Location", fmt.Sprintf("/api/v1/jobs/%d", job.ID))
		return httputils.WriteJSON(w, http.StatusAccepted, job)
	} else {
		var in entity.CreateUserInputV2
		if err := httputils.ParseJSON(r.Body, &in); err != nil {
			return err
		}

		if details := httputils.ValidateCreateUserInputV2(in); len(details) > 0 {
			return &httputils.ValidationError{InvalidParams: httputils.ToInvalidParams(details)}
		}

		v1in := entity.MapCreateV2ToV1(in)

		created, err := h.userService.CreateUser(r.Context(), &v1in)
		if err != nil {
			return err
		}

		w.Header().Set("Location", fmt.Sprintf("/api/v2/users/%d", created.ID))
		return httputils.WriteJSON(w, http.StatusCreated, entity.MapUserDTOToV2(created))
	}
}

func (h *UsersV2Handler) GetByID(w http.ResponseWriter, r *http.Request, id int) error {
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

	return httputils.WriteJSON(w, http.StatusOK, entity.UserDTO{
		ID:    u.ID,
		Name:  u.Name,
		Email: u.Email,
	})
}

func (h *UsersV2Handler) List(w http.ResponseWriter, r *http.Request) error {
	users, err := h.userService.GetAllUsers(r.Context())
	if err != nil {
		return err
	}
	return httputils.WriteJSON(w, http.StatusOK, entity.MapUsersToV2(users))
}
