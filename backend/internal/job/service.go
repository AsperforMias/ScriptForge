package job

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

type Service struct {
	logger *slog.Logger

	mu     sync.RWMutex
	jobs   map[string]Job
	stages map[string][]Stage
}

func NewService(logger *slog.Logger) *Service {
	return &Service{
		logger: logger,
		jobs:   make(map[string]Job),
		stages: make(map[string][]Stage),
	}
}

func (s *Service) Create(_ context.Context, req CreateJobRequest) (Job, error) {
	if err := validateCreateJobRequest(req); err != nil {
		return Job{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())

	created := Job{
		ID:              jobID,
		Status:          "queued",
		CurrentStage:    "ingest",
		ProgressPercent: 0,
		SourceTitle:     req.Source.Title,
		GenerationMode:  req.Generation.Mode,
		Warnings:        []string{},
		ErrorMessage:    "",
		CreatedAt:       now,
		UpdatedAt:       now,
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

	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[jobID] = created
	s.stages[jobID] = stageList

	s.logger.Info("job created",
		slog.String("component", "job"),
		slog.String("job_id", jobID),
		slog.String("status", created.Status),
		slog.String("source_title", created.SourceTitle),
	)

	return created, nil
}

func (s *Service) Get(_ context.Context, jobID string) (Details, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	created, ok := s.jobs[jobID]
	if !ok {
		return Details{}, ErrJobNotFound
	}

	return Details{
		Job:    created,
		Stages: s.stages[jobID],
	}, nil
}

func (s *Service) GetResult(_ context.Context, jobID string) (map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	created, ok := s.jobs[jobID]
	if !ok {
		return nil, ErrJobNotFound
	}

	if created.Status != "succeeded" {
		return nil, ErrJobNotReady
	}

	return map[string]any{
		"job_id":     created.ID,
		"screenplay": map[string]any{},
		"yaml_text":  "",
	}, nil
}

func (s *Service) Export(ctx context.Context, jobID string) (string, error) {
	result, err := s.GetResult(ctx, jobID)
	if err != nil {
		return "", err
	}

	yamlText, _ := result["yaml_text"].(string)
	return yamlText, nil
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
