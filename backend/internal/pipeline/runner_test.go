package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AsperforMias/ScriptForge/backend/internal/job"
	"github.com/AsperforMias/ScriptForge/backend/internal/llm"
	"github.com/AsperforMias/ScriptForge/backend/internal/storage/artifact"
	"github.com/AsperforMias/ScriptForge/backend/internal/workflow"
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

func TestRunnerRunMatchesDeterministicFixture(t *testing.T) {
	testCases := []struct {
		name         string
		request      job.CreateJobRequest
		expectedPath string
		jobID        string
	}{
		{
			name:         "night rain suspense",
			request:      fixtureCreateJobRequest(),
			expectedPath: filepath.Join("..", "..", "..", "testdata", "expected", "night-rain.screenplay.yaml"),
			jobID:        "job_fixture_runner_night_rain",
		},
		{
			name:         "workplace crisis",
			request:      mustLoadFixtureRequest(t, "workplace-crisis-request.json"),
			expectedPath: filepath.Join("..", "..", "..", "testdata", "expected", "workplace-crisis.screenplay.yaml"),
			jobID:        "job_fixture_runner_workplace",
		},
		{
			name:         "campus relay",
			request:      mustLoadFixtureRequest(t, "campus-relay-request.json"),
			expectedPath: filepath.Join("..", "..", "..", "testdata", "expected", "campus-relay.screenplay.yaml"),
			jobID:        "job_fixture_runner_campus",
		},
		{
			name:         "family dinner",
			request:      mustLoadFixtureRequest(t, "family-dinner-request.json"),
			expectedPath: filepath.Join("..", "..", "..", "testdata", "expected", "family-dinner.screenplay.yaml"),
			jobID:        "job_fixture_runner_family",
		},
		{
			name:         "comedy live mixup",
			request:      mustLoadFixtureRequest(t, "comedy-live-mixup-request.json"),
			expectedPath: filepath.Join("..", "..", "..", "testdata", "expected", "comedy-live-mixup.screenplay.yaml"),
			jobID:        "job_fixture_runner_comedy",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			store := artifact.New(tmpDir)
			runner := NewRunner(store, llm.NewUnavailableGenerator("deterministic mode does not use llm"))

			result, err := runner.Run(context.Background(), tc.jobID, tc.request)
			if err != nil {
				t.Fatalf("unexpected run error: %v", err)
			}

			expectedYAML, err := os.ReadFile(tc.expectedPath)
			if err != nil {
				t.Fatalf("read expected fixture: %v", err)
			}

			if strings.TrimSpace(result.YAMLText) != strings.TrimSpace(string(expectedYAML)) {
				t.Fatalf("unexpected yaml output\nexpected:\n%s\n\ngot:\n%s", string(expectedYAML), result.YAMLText)
			}
		})
	}
}

func mustLoadFixtureRequest(t *testing.T, name string) job.CreateJobRequest {
	t.Helper()

	path := filepath.Join("..", "..", "..", "testdata", "novels", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read request fixture %s: %v", name, err)
	}

	var req job.CreateJobRequest
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("unmarshal request fixture %s: %v", name, err)
	}
	return req
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

func TestRunnerRunPersistsProviderDebugSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	store := artifact.New(tmpDir)
	runner := NewRunner(store, debugGenerator{})

	req := validCreateJobRequest()
	req.Generation.Mode = "llm"

	result, err := runner.Run(context.Background(), "job_test_runner_llm_debug", req)
	if err != nil {
		t.Fatalf("unexpected llm run error: %v", err)
	}
	if result.ProviderDebugPath == "" {
		t.Fatal("expected provider debug path")
	}

	data, err := os.ReadFile(result.ProviderDebugPath)
	if err != nil {
		t.Fatalf("read provider debug artifact: %v", err)
	}

	var snapshot llm.DebugSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("unmarshal provider debug artifact: %v", err)
	}
	if snapshot.Provider != "debug-generator" {
		t.Fatalf("expected provider debug-generator, got %s", snapshot.Provider)
	}
	if snapshot.ParseMode != "canonical" {
		t.Fatalf("expected parse mode canonical, got %s", snapshot.ParseMode)
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

func fixtureCreateJobRequest() job.CreateJobRequest {
	var req job.CreateJobRequest
	req.Source.Title = "夜雨疑云"
	req.Source.Author = "示例作者"
	req.Adaptation.Style = "悬疑短剧"
	req.Adaptation.Audience = "大众向"
	req.Adaptation.Notes = []string{"强化悬疑氛围", "保留主角主动调查的动机"}
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "第一章 深夜回家", Content: "林琪深夜回到公寓，发现门锁似乎被人动过。她停在走廊里，不确定是否应该立刻进去。"},
		{Index: 2, Title: "第二章 陌生字条", Content: "她在房间里找到一张陌生字条，上面只写着今晚别睡。林琪意识到有人提前进入过房间。"},
		{Index: 3, Title: "第三章 清晨追踪", Content: "第二天清晨，林琪带着字条前往车站，试图顺着纸上的线索找到寄信人。"},
	}
	return req
}

type debugGenerator struct{}

func (debugGenerator) Name() string {
	return "debug-generator"
}

func (debugGenerator) Generate(_ context.Context, req llm.GenerateRequest) (llm.GenerateResult, error) {
	document := workflow.BuildDocument(req.Input, req.Source, req.Outline, req.Entities, req.Plan)
	document.Validation.Warnings = append(document.Validation.Warnings, "generated via debug generator")

	return llm.GenerateResult{
		Document: document,
		Warnings: document.Validation.Warnings,
		Debug: &llm.DebugSnapshot{
			Provider:   "debug-generator",
			Model:      "fixture-model",
			ParseMode:  "canonical",
			RawContent: "version: \"1.0\"",
		},
	}, nil
}
