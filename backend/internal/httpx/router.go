package httpx

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/AsperforMias/ScriptForge/backend/internal/config"
	"github.com/AsperforMias/ScriptForge/backend/internal/httpx/handler"
	appmiddleware "github.com/AsperforMias/ScriptForge/backend/internal/httpx/middleware"
	"github.com/AsperforMias/ScriptForge/backend/internal/job"
)

func NewRouter(cfg config.Config, logger *slog.Logger, jobService *job.Service) http.Handler {
	r := chi.NewRouter()

	r.Use(appmiddleware.RequestID)
	r.Use(appmiddleware.Recoverer(logger))
	r.Use(appmiddleware.Timeout(cfg.HTTPWriteTimeout))
	r.Use(appmiddleware.AccessLog(logger))
	r.Use(appmiddleware.BodyLimit(cfg.RequestBodyLimitBytes))
	r.Use(appmiddleware.CORS(cfg.CORSAllowOrigin))

	jobHandler := handler.NewJobs(logger, jobService)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api/v1", func(api chi.Router) {
		api.Post("/jobs", jobHandler.Create)
		api.Get("/jobs/{jobID}", jobHandler.Get)
		api.Get("/jobs/{jobID}/result", jobHandler.GetResult)
		api.Get("/jobs/{jobID}/export", jobHandler.Export)
	})

	return r
}
