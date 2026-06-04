package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/AsperforMias/ScriptForge/backend/internal/job"
	"github.com/AsperforMias/ScriptForge/backend/internal/llm"
	"github.com/AsperforMias/ScriptForge/backend/internal/storage/artifact"
)

func TestRunnerRunProducesArtifactsAndYAML(t *testing.T) {
	tmpDir := t.TempDir()
	store := artifact.New(tmpDir)
	runner := NewRunner(store, llm.NewUnavailableGenerator("deterministic mode does not use llm"))

	req := validCreateJobRequest()
	result, err := runner.Run(context.Background(), "job_test_runner", req)
	if err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}

	if result.YAMLText == "" {
		t.Fatal("expected yaml output")
	}
	if len(result.Document.Scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(result.Document.Scenes))
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "job_test_runner", "screenplay.yaml")); err != nil {
		t.Fatalf("expected screenplay artifact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "job_test_runner", "input.json")); err != nil {
		t.Fatalf("expected input artifact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "job_test_runner", "normalized_source.json")); err != nil {
		t.Fatalf("expected normalized source artifact: %v", err)
	}
}

func TestRunnerRunSupportsMockLLMMode(t *testing.T) {
	tmpDir := t.TempDir()
	store := artifact.New(tmpDir)
	runner := NewRunner(store, llm.NewMockGenerator())

	req := validCreateJobRequest()
	req.Generation.Mode = "llm"

	result, err := runner.Run(context.Background(), "job_test_runner_llm", req)
	if err != nil {
		t.Fatalf("unexpected llm run error: %v", err)
	}
	if len(result.Document.Scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(result.Document.Scenes))
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected llm warnings")
	}
}

func TestRunnerRunFailsWhenLLMProviderIsUnavailable(t *testing.T) {
	tmpDir := t.TempDir()
	store := artifact.New(tmpDir)
	runner := NewRunner(store, llm.NewUnavailableGenerator("provider not configured"))

	req := validCreateJobRequest()
	req.Generation.Mode = "llm"

	result, err := runner.Run(context.Background(), "job_test_runner_llm_disabled", req)
	if err == nil {
		t.Fatal("expected llm provider error")
	}
	if result.CurrentStage != "screenplay_generation" {
		t.Fatalf("expected screenplay_generation failure, got %s", result.CurrentStage)
	}
}

func validCreateJobRequest() job.CreateJobRequest {
	var req job.CreateJobRequest
	req.Source.Title = "Night Rain"
	req.Source.Author = "Demo Author"
	req.Adaptation.Style = "Suspense Drama"
	req.Adaptation.Audience = "General"
	req.Adaptation.Notes = []string{"Keep a strong hook in each scene"}
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "Chapter 1", Content: "林琪深夜回到公寓，发现门锁似乎被动过。"},
		{Index: 2, Title: "Chapter 2", Content: "她在房间里找到一张陌生字条，怀疑有人潜入。"},
		{Index: 3, Title: "Chapter 3", Content: "第二天清晨，她决定顺着线索前往车站。"},
	}
	return req
}
