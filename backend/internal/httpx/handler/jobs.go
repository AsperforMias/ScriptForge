package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/AsperforMias/ScriptForge/backend/internal/httpx/render"
	"github.com/AsperforMias/ScriptForge/backend/internal/job"
)

type Jobs struct {
	logger  *slog.Logger
	service *job.Service
}

func NewJobs(logger *slog.Logger, service *job.Service) *Jobs {
	return &Jobs{
		logger:  logger,
		service: service,
	}
}

func (h *Jobs) Create(w http.ResponseWriter, r *http.Request) {
	var req job.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.WriteError(r.Context(), w, job.ErrInvalidInput.WithMessage("invalid json body"))
		return
	}

	createdJob, err := h.service.Create(r.Context(), req)
	if err != nil {
		render.WriteError(r.Context(), w, err)
		return
	}

	render.WriteJSON(r.Context(), w, http.StatusAccepted, map[string]any{
		"job": createdJob,
	})
}

func (h *Jobs) Get(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	details, err := h.service.Get(r.Context(), jobID)
	if err != nil {
		render.WriteError(r.Context(), w, err)
		return
	}

	render.WriteJSON(r.Context(), w, http.StatusOK, map[string]any{
		"job":    details.Job,
		"stages": details.Stages,
	})
}

func (h *Jobs) GetResult(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	result, err := h.service.GetResult(r.Context(), jobID)
	if err != nil {
		render.WriteError(r.Context(), w, err)
		return
	}

	render.WriteJSON(r.Context(), w, http.StatusOK, result)
}

func (h *Jobs) Export(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	export, err := h.service.Export(r.Context(), jobID)
	if err != nil {
		render.WriteError(r.Context(), w, err)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+jobID+".screenplay.yaml\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(export))
}
