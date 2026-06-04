package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/AsperforMias/ScriptForge/backend/internal/job"
)

func TestStorePersistsProgressAndWarnings(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "scriptforge.db"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer store.Close()

	record := job.Job{
		ID:              "job_progress",
		Status:          "running",
		CurrentStage:    "scene_planning",
		ProgressPercent: 55,
		SourceTitle:     "Night Rain",
		GenerationMode:  "deterministic",
		Warnings:        []string{"scene count inferred from chapter titles"},
		CreatedAt:       "2026-06-05T00:00:00Z",
		UpdatedAt:       "2026-06-05T00:00:10Z",
	}

	stages := []job.Stage{
		{Name: "ingest", Status: "succeeded"},
		{Name: "scene_planning", Status: "running"},
	}

	if err := store.CreateJob(context.Background(), record, stages); err != nil {
		t.Fatalf("create job: %v", err)
	}

	persisted, err := store.GetJob(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}

	if persisted.ProgressPercent != 55 {
		t.Fatalf("expected progress 55, got %d", persisted.ProgressPercent)
	}
	if len(persisted.Warnings) != 1 || persisted.Warnings[0] != record.Warnings[0] {
		t.Fatalf("expected warnings to round-trip, got %#v", persisted.Warnings)
	}
}

func TestStoreUpdateJobPersistsLatestProgressAndWarnings(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "scriptforge.db"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer store.Close()

	record := job.Job{
		ID:              "job_update",
		Status:          "queued",
		CurrentStage:    "ingest",
		ProgressPercent: 0,
		SourceTitle:     "Night Rain",
		GenerationMode:  "deterministic",
		Warnings:        []string{},
		CreatedAt:       "2026-06-05T00:00:00Z",
		UpdatedAt:       "2026-06-05T00:00:00Z",
	}

	if err := store.CreateJob(context.Background(), record, []job.Stage{{Name: "ingest", Status: "queued"}}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	record.Status = "failed"
	record.CurrentStage = "validation"
	record.ProgressPercent = 90
	record.Warnings = []string{"character alias normalized"}
	record.ErrorMessage = "validation failed"
	record.UpdatedAt = "2026-06-05T00:01:00Z"

	if err := store.UpdateJob(context.Background(), record); err != nil {
		t.Fatalf("update job: %v", err)
	}

	persisted, err := store.GetJob(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("get updated job: %v", err)
	}

	if persisted.Status != "failed" {
		t.Fatalf("expected failed status, got %s", persisted.Status)
	}
	if persisted.ProgressPercent != 90 {
		t.Fatalf("expected progress 90, got %d", persisted.ProgressPercent)
	}
	if len(persisted.Warnings) != 1 || persisted.Warnings[0] != "character alias normalized" {
		t.Fatalf("expected warning to persist, got %#v", persisted.Warnings)
	}
	if persisted.ErrorMessage != "validation failed" {
		t.Fatalf("expected error message to persist, got %s", persisted.ErrorMessage)
	}
}
