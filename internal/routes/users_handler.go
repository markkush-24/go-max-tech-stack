package routes

import (
	"errors"
	"fmt"
	"net/http"
	"pet-study/internal/entity"
	"pet-study/internal/httputils"
	"pet-study/internal/metrics"
	"pet-study/internal/queue"
	"pet-study/internal/service"
)

type UsersHandler struct {
	userService  *service.UserService
	jobService   *service.JobService
	workerQueue  *queue.Queue
	jobsObserver metrics.JobsObserver
}

func NewUserHandler(
	userService *service.UserService,
	jobService *service.JobService,
	workerQueue *queue.Queue,
	metrics metrics.JobsObserver,
) *UsersHandler {
	return &UsersHandler{
		userService:  userService,
		jobService:   jobService,
		workerQueue:  workerQueue,
		jobsObserver: metrics}
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

		job := entity.Job{Status: entity.JobQueued}
		if err := h.jobService.Save(r.Context(), &job); err != nil {
			return err
		}
		item := queue.WorkItem{JobID: job.ID, Payload: in}
		if enqueueErr := h.workerQueue.Enqueue(r.Context(), item); enqueueErr != nil {
			deleteErr := h.jobService.Delete(r.Context(), job.ID)
			if deleteErr != nil {
				return errors.Join(enqueueErr, deleteErr)
			}
			return enqueueErr
		}
		h.jobsObserver.IncQueued()

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
	return nil
}

func (h *UsersHandler) GetByID(w http.ResponseWriter, r *http.Request, id int) error {
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
