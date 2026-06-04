package job

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
)

func TestCreateRejectsLessThanThreeChapters(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(slog.New(slog.NewTextHandler(io.Discard, nil)), repo, fakeRunner{}, fakeYAMLReader{}, 1)
	req := validCreateJobRequest()
	req.Source.Chapters = req.Source.Chapters[:2]

	_, err := service.Create(context.Background(), req)
	if err == nil {
		t.Fatal("expected validation error")
	}

	var appErr AppError
	if !AsAppError(err, &appErr) {
		t.Fatal("expected app error")
	}
	if appErr.Code != "invalid_input" {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}

func TestGetResultReturnsNotReadyForQueuedJob(t *testing.T) {
	repo := newFakeRepository()
	service := NewService(slog.New(slog.NewTextHandler(io.Discard, nil)), repo, fakeRunner{}, fakeYAMLReader{}, 1)
	req := validCreateJobRequest()

	created, err := service.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	_, err = service.GetResult(context.Background(), created.ID)
	if err == nil {
		t.Fatal("expected job not ready error")
	}

	var appErr AppError
	if !AsAppError(err, &appErr) {
		t.Fatal("expected app error")
	}
	if appErr.Code != "job_not_ready" {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}

type fakeRepository struct {
	mu        sync.RWMutex
	jobs      map[string]Job
	stages    map[string][]Stage
	artifacts map[string]Artifact
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		jobs:      map[string]Job{},
		stages:    map[string][]Stage{},
		artifacts: map[string]Artifact{},
	}
}

func (r *fakeRepository) CreateJob(_ context.Context, record Job, stages []Stage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs[record.ID] = record
	r.stages[record.ID] = stages
	return nil
}

func (r *fakeRepository) GetJob(_ context.Context, jobID string) (Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.jobs[jobID]
	if !ok {
		return Job{}, ErrJobNotFound
	}
	return record, nil
}

func (r *fakeRepository) GetStages(_ context.Context, jobID string) ([]Stage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stages[jobID], nil
}

func (r *fakeRepository) UpdateJob(_ context.Context, record Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs[record.ID] = record
	return nil
}

func (r *fakeRepository) UpdateStages(_ context.Context, jobID string, stages []Stage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stages[jobID] = stages
	return nil
}

func (r *fakeRepository) SaveArtifact(_ context.Context, artifact Artifact) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.artifacts[artifact.JobID] = artifact
	return nil
}

func (r *fakeRepository) GetArtifact(_ context.Context, jobID string) (Artifact, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	artifact, ok := r.artifacts[jobID]
	if !ok {
		return Artifact{}, ErrJobNotReady
	}
	return artifact, nil
}

type fakeRunner struct{}

func (fakeRunner) Run(_ context.Context, _ string, _ CreateJobRequest) (ExecutionResult, error) {
	return ExecutionResult{}, nil
}

type fakeYAMLReader struct{}

func (fakeYAMLReader) ReadYAML(_ string) (string, error) {
	return "", nil
}

func validCreateJobRequest() CreateJobRequest {
	var req CreateJobRequest
	req.Source.Title = "Night Rain"
	req.Adaptation.Style = "Suspense Drama"
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []ChapterBody{
		{Index: 1, Title: "Chapter 1", Content: "A"},
		{Index: 2, Title: "Chapter 2", Content: "B"},
		{Index: 3, Title: "Chapter 3", Content: "C"},
	}
	return req
}
