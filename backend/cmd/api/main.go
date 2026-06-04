package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/AsperforMias/ScriptForge/backend/internal/config"
	"github.com/AsperforMias/ScriptForge/backend/internal/httpx"
	"github.com/AsperforMias/ScriptForge/backend/internal/job"
	"github.com/AsperforMias/ScriptForge/backend/internal/llm"
	"github.com/AsperforMias/ScriptForge/backend/internal/pipeline"
	"github.com/AsperforMias/ScriptForge/backend/internal/storage/artifact"
	"github.com/AsperforMias/ScriptForge/backend/internal/storage/sqlite"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("starting scriptforge backend",
		slog.String("http_addr", cfg.HTTPAddr),
		slog.String("app_env", cfg.AppEnv),
		slog.String("sqlite_path", cfg.SQLitePath),
		slog.String("artifact_dir", cfg.ArtifactDir),
		slog.Int64("request_body_limit_bytes", cfg.RequestBodyLimitBytes),
		slog.Int("job_max_concurrency", cfg.JobMaxConcurrency),
		slog.String("generation_mode_default", cfg.GenerationModeDefault),
		slog.String("llm_provider", cfg.LLMProvider),
		slog.String("llm_model", cfg.LLMModel),
	)

	repo, err := sqlite.Open(cfg.SQLitePath)
	if err != nil {
		logger.Error("failed to open sqlite store", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer repo.Close()

	artifactStore := artifact.New(cfg.ArtifactDir)
	llmGenerator := llm.NewGenerator(llm.ProviderConfig{
		Provider:       cfg.LLMProvider,
		Model:          cfg.LLMModel,
		BaseURL:        cfg.LLMBaseURL,
		APIKey:         cfg.LLMAPIKey,
		RequestTimeout: cfg.LLMRequestTimeout.String(),
	})
	runner := pipeline.NewRunner(artifactStore, llmGenerator)
	jobService := job.NewService(logger, repo, runner, artifactStore, cfg.JobMaxConcurrency)
	router := httpx.NewRouter(cfg, logger, jobService)

	server := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTPWriteTimeout)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
