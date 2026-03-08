package router

import (
	"net/http"
	"pet-study/internal/httputils"
	"pet-study/internal/service"
)

type JobHandler struct {
	jobService *service.JobService
}

func NewJobHandler(jobService *service.JobService) *JobHandler {
	return &JobHandler{jobService: jobService}
}

func (h *JobHandler) GetByID(w http.ResponseWriter, r *http.Request, id int) error {
	job, err := h.jobService.GetByID(r.Context(), int64(id))
	if err != nil {
		return err
	}
	return httputils.WriteJSON(w, http.StatusOK, job)
}
