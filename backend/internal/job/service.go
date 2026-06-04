package job

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/AsperforMias/ScriptForge/backend/internal/screenplay"
)

type YAMLReader interface {
	ReadYAML(path string) (string, error)
}

type Service struct {
	logger     *slog.Logger
	repo       Repository
	runner     Runner
	yamlReader YAMLReader
	semaphore  chan struct{}
}

func NewService(logger *slog.Logger, repo Repository, runner Runner, yamlReader YAMLReader, maxConcurrency int) *Service {
	if maxConcurrency <= 0 {
		maxConcurrency = 1
	}

	return &Service{
		logger:     logger,
		repo:       repo,
		runner:     runner,
		yamlReader: yamlReader,
		semaphore:  make(chan struct{}, maxConcurrency),
	}
}

func (s *Service) Create(ctx context.Context, req CreateJobRequest) (Job, error) {
	if err := validateCreateJobRequest(req); err != nil {
		return Job{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())

	created := Job{
		ID:                jobID,
		Status:            "queued",
		CurrentStage:      "ingest",
		ProgressPercent:   0,
		SourceTitle:       req.Source.Title,
		GenerationMode:    req.Generation.Mode,
		Warnings:          []string{},
		ErrorMessage:      "",
		CreatedAt:         now,
		UpdatedAt:         now,
		InputSnapshotPath: "",
		ResultYAMLPath:    "",
	}

	stageList := []Stage{
		{Name: "ingest", Status: "queued"},
		{Name: "outline", Status: "queued"},
		{Name: "entities", Status: "queued"},
		{Name: "scene_planning", Status: "queued"},
		{Name: "screenplay_generation", Status: "queued"},
		{Name: "validation", Status: "queued"},
		{Name: "persistence", Status: "queued"},
	}

	if err := s.repo.CreateJob(ctx, created, stageList); err != nil {
		return Job{}, err
	}

	s.logger.Info("job created",
		slog.String("component", "job"),
		slog.String("job_id", jobID),
		slog.String("status", created.Status),
		slog.String("source_title", created.SourceTitle),
	)

	go s.execute(jobID, req)

	return created, nil
}

func (s *Service) Get(ctx context.Context, jobID string) (Details, error) {
	record, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		return Details{}, err
	}

	stages, err := s.repo.GetStages(ctx, jobID)
	if err != nil {
		return Details{}, err
	}

	return Details{Job: record, Stages: stages}, nil
}

func (s *Service) GetResult(ctx context.Context, jobID string) (ResultPayload, error) {
	record, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		return ResultPayload{}, err
	}
	if record.Status != "succeeded" {
		return ResultPayload{}, ErrJobNotReady
	}

	artifact, err := s.repo.GetArtifact(ctx, jobID)
	if err != nil {
		return ResultPayload{}, err
	}

	yamlText, err := s.yamlReader.ReadYAML(artifact.YAMLPath)
	if err != nil {
		return ResultPayload{}, err
	}

	document, err := screenplay.ParseYAML(yamlText)
	if err != nil {
		return ResultPayload{}, NewAppError("internal_error", "failed to parse persisted screenplay", map[string]any{
			"job_id": jobID,
		})
	}

	return ResultPayload{
		JobID:      record.ID,
		Screenplay: document,
		YAMLText:   yamlText,
	}, nil
}

func (s *Service) Export(ctx context.Context, jobID string) (string, error) {
	result, err := s.GetResult(ctx, jobID)
	if err != nil {
		return "", err
	}
	return result.YAMLText, nil
}

func (s *Service) execute(jobID string, req CreateJobRequest) {
	s.semaphore <- struct{}{}
	defer func() { <-s.semaphore }()

	ctx := context.Background()

	record, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		s.logger.Error("failed to load job for execution", slog.String("job_id", jobID), slog.String("error", err.Error()))
		return
	}

	record.Status = "running"
	record.CurrentStage = "ingest"
	record.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := s.repo.UpdateJob(ctx, record); err != nil {
		s.logger.Error("failed to mark job running", slog.String("job_id", jobID), slog.String("error", err.Error()))
		return
	}

	result, runErr := s.runner.Run(ctx, jobID, req)
	if err := s.repo.UpdateStages(ctx, jobID, result.Stages); err != nil {
		s.logger.Error("failed to update job stages", slog.String("job_id", jobID), slog.String("error", err.Error()))
	}

	record, _ = s.repo.GetJob(ctx, jobID)
	record.CurrentStage = result.CurrentStage
	record.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if runErr != nil {
		record.Status = "failed"
		record.ErrorMessage = runErr.Error()
		record.Warnings = nil
		_ = s.repo.UpdateJob(ctx, record)
		s.logger.Error("job execution failed",
			slog.String("component", "job"),
			slog.String("job_id", jobID),
			slog.String("stage", result.CurrentStage),
			slog.String("error", runErr.Error()),
		)
		return
	}

	record.Status = "succeeded"
	record.ErrorMessage = ""
	record.Warnings = result.Warnings
	record.InputSnapshotPath = result.InputSnapshotPath
	record.ResultYAMLPath = result.YAMLPath
	record.ProgressPercent = 100
	if err := s.repo.SaveArtifact(ctx, Artifact{
		JobID:         jobID,
		YAMLPath:      result.YAMLPath,
		YAMLSizeBytes: len(result.YAMLText),
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		s.logger.Error("failed to save artifact", slog.String("job_id", jobID), slog.String("error", err.Error()))
		return
	}

	if err := s.repo.UpdateJob(ctx, record); err != nil {
		s.logger.Error("failed to mark job succeeded", slog.String("job_id", jobID), slog.String("error", err.Error()))
		return
	}

	s.logger.Info("job execution succeeded",
		slog.String("component", "job"),
		slog.String("job_id", jobID),
		slog.String("stage", result.CurrentStage),
	)
}

func validateCreateJobRequest(req CreateJobRequest) error {
	if strings.TrimSpace(req.Source.Title) == "" {
		return ErrInvalidInput.WithMessage("source.title is required")
	}
	if len(req.Source.Chapters) < 3 {
		return ErrInvalidInput.WithMessage("at least 3 chapters are required")
	}
	if strings.TrimSpace(req.Adaptation.Style) == "" {
		return ErrInvalidInput.WithMessage("adaptation.style is required")
	}
	if req.Generation.Mode != "deterministic" && req.Generation.Mode != "llm" {
		return ErrInvalidInput.WithMessage("generation.mode must be deterministic or llm")
	}

	for idx, chapter := range req.Source.Chapters {
		expected := idx + 1
		if chapter.Index != expected {
			return ErrInvalidInput.WithMessage("source.chapters index must be continuous from 1")
		}
		if strings.TrimSpace(chapter.Title) == "" {
			return ErrInvalidInput.WithMessage("source.chapters title is required")
		}
		if strings.TrimSpace(chapter.Content) == "" {
			return ErrInvalidInput.WithMessage("source.chapters content is required")
		}
	}

	return nil
}
