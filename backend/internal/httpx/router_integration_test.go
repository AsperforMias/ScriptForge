package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AsperforMias/ScriptForge/backend/internal/config"
	"github.com/AsperforMias/ScriptForge/backend/internal/job"
	"github.com/AsperforMias/ScriptForge/backend/internal/llm"
	"github.com/AsperforMias/ScriptForge/backend/internal/pipeline"
	"github.com/AsperforMias/ScriptForge/backend/internal/storage/artifact"
	"github.com/AsperforMias/ScriptForge/backend/internal/storage/sqlite"
)

func TestRouterJobLifecycleWithFixtures(t *testing.T) {
	router, repo := newTestHarness(t, pipeline.NewRunner(artifact.New(filepath.Join(t.TempDir(), "artifacts")), llm.NewUnavailableGenerator("deterministic mode does not use llm")))

	requestBody, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "novels", "night-rain-request.json"))
	if err != nil {
		t.Fatalf("read request fixture: %v", err)
	}

	createResp := performRequest(t, router, http.MethodPost, "/api/v1/jobs", bytes.NewReader(requestBody))

	if createResp.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", createResp.Code)
	}

	var createEnvelope struct {
		Data struct {
			Job struct {
				ID string `json:"id"`
			} `json:"job"`
		} `json:"data"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&createEnvelope); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	jobID := createEnvelope.Data.Job.ID
	if jobID == "" {
		t.Fatal("expected job id")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for {
		record, err := repo.GetJob(ctx, jobID)
		if err == nil && record.Status == "succeeded" {
			break
		}
		if err == nil && record.Status == "failed" {
			t.Fatalf("job failed unexpectedly: %s", record.ErrorMessage)
		}

		select {
		case <-ctx.Done():
			t.Fatal("job did not succeed before timeout")
		case <-time.After(50 * time.Millisecond):
		}
	}

	statusResp := performRequest(t, router, http.MethodGet, "/api/v1/jobs/"+jobID, nil)
	if statusResp.Code != http.StatusOK {
		t.Fatalf("expected 200 status response, got %d", statusResp.Code)
	}

	var statusEnvelope struct {
		Data struct {
			Job struct {
				Status          string `json:"status"`
				ProgressPercent int    `json:"progress_percent"`
			} `json:"job"`
		} `json:"data"`
	}
	if err := json.NewDecoder(statusResp.Body).Decode(&statusEnvelope); err != nil {
		t.Fatalf("decode final status response: %v", err)
	}
	if statusEnvelope.Data.Job.Status != "succeeded" {
		t.Fatalf("expected succeeded final status, got %s", statusEnvelope.Data.Job.Status)
	}
	if statusEnvelope.Data.Job.ProgressPercent != 100 {
		t.Fatalf("expected succeeded job progress to be 100, got %d", statusEnvelope.Data.Job.ProgressPercent)
	}

	resultResp := performRequest(t, router, http.MethodGet, "/api/v1/jobs/"+jobID+"/result", nil)

	if resultResp.Code != http.StatusOK {
		t.Fatalf("expected 200 result status, got %d", resultResp.Code)
	}

	var resultEnvelope struct {
		Data struct {
			YAMLText string `json:"yaml_text"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resultResp.Body).Decode(&resultEnvelope); err != nil {
		t.Fatalf("decode result response: %v", err)
	}

	expectedYAML, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "expected", "night-rain.screenplay.yaml"))
	if err != nil {
		t.Fatalf("read expected yaml fixture: %v", err)
	}

	if strings.TrimSpace(resultEnvelope.Data.YAMLText) != strings.TrimSpace(string(expectedYAML)) {
		t.Fatalf("unexpected yaml output\nexpected:\n%s\n\ngot:\n%s", string(expectedYAML), resultEnvelope.Data.YAMLText)
	}

	exportResp := performRequest(t, router, http.MethodGet, "/api/v1/jobs/"+jobID+"/export", nil)

	if exportResp.Code != http.StatusOK {
		t.Fatalf("expected 200 export status, got %d", exportResp.Code)
	}
}

func TestRouterRejectsInvalidChapterCount(t *testing.T) {
	router, _ := newTestHarness(t, pipeline.NewRunner(artifact.New(filepath.Join(t.TempDir(), "artifacts")), llm.NewUnavailableGenerator("deterministic mode does not use llm")))

	body := []byte(`{"source":{"title":"Broken","chapters":[{"index":1,"title":"One","content":"A"},{"index":2,"title":"Two","content":"B"}]},"adaptation":{"style":"Suspense"},"generation":{"mode":"deterministic"}}`)
	resp := performRequest(t, router, http.MethodPost, "/api/v1/jobs", bytes.NewReader(body))

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}

	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode invalid input response: %v", err)
	}
	if envelope.Error.Code != "invalid_input" {
		t.Fatalf("expected invalid_input code, got %s", envelope.Error.Code)
	}
}

func TestRouterReturnsConflictWhenResultIsNotReady(t *testing.T) {
	runner := &blockingRunner{release: make(chan struct{})}
	router, _ := newTestHarness(t, runner)
	defer close(runner.release)

	requestBody, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "novels", "night-rain-request.json"))
	if err != nil {
		t.Fatalf("read request fixture: %v", err)
	}

	createResp := performRequest(t, router, http.MethodPost, "/api/v1/jobs", bytes.NewReader(requestBody))
	if createResp.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", createResp.Code)
	}

	var createEnvelope struct {
		Data struct {
			Job struct {
				ID string `json:"id"`
			} `json:"job"`
		} `json:"data"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&createEnvelope); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createEnvelope.Data.Job.ID == "" {
		t.Fatal("expected job id")
	}

	time.Sleep(100 * time.Millisecond)

	resultResp := performRequest(t, router, http.MethodGet, "/api/v1/jobs/"+createEnvelope.Data.Job.ID+"/result", nil)

	if resultResp.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resultResp.Code)
	}

	statusResp := performRequest(t, router, http.MethodGet, "/api/v1/jobs/"+createEnvelope.Data.Job.ID, nil)

	if statusResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", statusResp.Code)
	}

	var statusEnvelope struct {
		Data struct {
			Job struct {
				Status          string `json:"status"`
				CurrentStage    string `json:"current_stage"`
				ProgressPercent int    `json:"progress_percent"`
			} `json:"job"`
		} `json:"data"`
	}
	if err := json.NewDecoder(statusResp.Body).Decode(&statusEnvelope); err != nil {
		t.Fatalf("decode pending status response: %v", err)
	}
	if statusEnvelope.Data.Job.Status != "running" {
		t.Fatalf("expected running status, got %s", statusEnvelope.Data.Job.Status)
	}
	if statusEnvelope.Data.Job.CurrentStage != "ingest" {
		t.Fatalf("expected current stage ingest, got %s", statusEnvelope.Data.Job.CurrentStage)
	}
	if statusEnvelope.Data.Job.ProgressPercent != 5 {
		t.Fatalf("expected ingest progress 5, got %d", statusEnvelope.Data.Job.ProgressPercent)
	}
}

func TestRouterReturnsGenerationFailedForFailedJobResult(t *testing.T) {
	router, repo := newTestHarness(t, pipeline.NewRunner(artifact.New(filepath.Join(t.TempDir(), "artifacts")), llm.NewUnavailableGenerator("deterministic mode does not use llm")))

	record := job.Job{
		ID:              "job_failed_http",
		Status:          "failed",
		CurrentStage:    "validation",
		ProgressPercent: 90,
		SourceTitle:     "Night Rain",
		GenerationMode:  "deterministic",
		ErrorMessage:    "validation failed",
		CreatedAt:       "2026-06-05T00:00:00Z",
		UpdatedAt:       "2026-06-05T00:00:10Z",
	}
	stages := []job.Stage{
		{Name: "ingest", Status: "succeeded"},
		{Name: "outline", Status: "succeeded"},
		{Name: "entities", Status: "succeeded"},
		{Name: "scene_planning", Status: "succeeded"},
		{Name: "screenplay_generation", Status: "succeeded"},
		{Name: "validation", Status: "failed", ErrorMessage: "validation failed"},
		{Name: "persistence", Status: "queued"},
	}
	if err := repo.CreateJob(context.Background(), record, stages); err != nil {
		t.Fatalf("seed failed job: %v", err)
	}

	statusResp := performRequest(t, router, http.MethodGet, "/api/v1/jobs/"+record.ID, nil)
	if statusResp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", statusResp.Code)
	}

	var statusEnvelope struct {
		Data struct {
			Job struct {
				Status          string `json:"status"`
				CurrentStage    string `json:"current_stage"`
				ProgressPercent int    `json:"progress_percent"`
			} `json:"job"`
		} `json:"data"`
	}
	if err := json.NewDecoder(statusResp.Body).Decode(&statusEnvelope); err != nil {
		t.Fatalf("decode failed job status response: %v", err)
	}
	if statusEnvelope.Data.Job.Status != "failed" {
		t.Fatalf("expected failed status, got %s", statusEnvelope.Data.Job.Status)
	}
	if statusEnvelope.Data.Job.CurrentStage != "validation" {
		t.Fatalf("expected current stage validation, got %s", statusEnvelope.Data.Job.CurrentStage)
	}
	if statusEnvelope.Data.Job.ProgressPercent != 90 {
		t.Fatalf("expected failed job progress 90, got %d", statusEnvelope.Data.Job.ProgressPercent)
	}

	resultResp := performRequest(t, router, http.MethodGet, "/api/v1/jobs/"+record.ID+"/result", nil)
	if resultResp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resultResp.Code)
	}

	var resultEnvelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resultResp.Body).Decode(&resultEnvelope); err != nil {
		t.Fatalf("decode failed result response: %v", err)
	}
	if resultEnvelope.Error.Code != "generation_failed" {
		t.Fatalf("expected generation_failed, got %s", resultEnvelope.Error.Code)
	}
	if resultEnvelope.Error.Message != "validation failed" {
		t.Fatalf("expected validation failed message, got %s", resultEnvelope.Error.Message)
	}

	exportResp := performRequest(t, router, http.MethodGet, "/api/v1/jobs/"+record.ID+"/export", nil)
	if exportResp.Code != http.StatusInternalServerError {
		t.Fatalf("expected export 500, got %d", exportResp.Code)
	}
}

func TestRouterSupportsMockLLMJobLifecycle(t *testing.T) {
	router, repo := newTestHarness(t, pipeline.NewRunner(artifact.New(filepath.Join(t.TempDir(), "artifacts")), llm.NewMockGenerator()))

	requestBody := []byte(`{"source":{"title":"Night Rain","author":"Demo Author","chapters":[{"index":1,"title":"Chapter 1","content":"林琪深夜回到公寓，发现门锁似乎被动过。"},{"index":2,"title":"Chapter 2","content":"她在房间里找到一张陌生字条，怀疑有人潜入。"},{"index":3,"title":"Chapter 3","content":"第二天清晨，她决定顺着线索前往车站。"}]},"adaptation":{"style":"Suspense Drama","audience":"General","notes":["Keep a strong hook in each scene"]},"generation":{"mode":"llm"}}`)

	createResp := performRequest(t, router, http.MethodPost, "/api/v1/jobs", bytes.NewReader(requestBody))
	if createResp.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", createResp.Code)
	}

	var createEnvelope struct {
		Data struct {
			Job struct {
				ID string `json:"id"`
			} `json:"job"`
		} `json:"data"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&createEnvelope); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for {
		record, err := repo.GetJob(ctx, createEnvelope.Data.Job.ID)
		if err == nil && record.Status == "succeeded" {
			break
		}
		if err == nil && record.Status == "failed" {
			t.Fatalf("llm mock job failed unexpectedly: %s", record.ErrorMessage)
		}

		select {
		case <-ctx.Done():
			t.Fatal("mock llm job did not succeed before timeout")
		case <-time.After(50 * time.Millisecond):
		}
	}

	resultResp := performRequest(t, router, http.MethodGet, "/api/v1/jobs/"+createEnvelope.Data.Job.ID+"/result", nil)
	if resultResp.Code != http.StatusOK {
		t.Fatalf("expected 200 result status, got %d", resultResp.Code)
	}

	var resultEnvelope struct {
		Data struct {
			Screenplay struct {
				Validation struct {
					Warnings []string `json:"warnings"`
				} `json:"validation"`
			} `json:"screenplay"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resultResp.Body).Decode(&resultEnvelope); err != nil {
		t.Fatalf("decode llm result response: %v", err)
	}
	if len(resultEnvelope.Data.Screenplay.Validation.Warnings) == 0 {
		t.Fatal("expected llm mock warning in validation block")
	}
}

func newTestHarness(t *testing.T, runner job.Runner) (http.Handler, *sqlite.Store) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "scriptforge.db")
	artifactDir := filepath.Join(tmpDir, "artifacts")

	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}

	artifactStore := artifact.New(artifactDir)
	service := job.NewService(slog.New(slog.NewTextHandler(io.Discard, nil)), repo, runner, artifactStore, 1)
	cfg := config.Load()
	cfg.SQLitePath = dbPath
	cfg.ArtifactDir = artifactDir

	router := NewRouter(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), service)
	t.Cleanup(func() {
		_ = repo.Close()
	})

	return router, repo
}

type blockingRunner struct {
	release chan struct{}
}

func (r *blockingRunner) Run(_ context.Context, _ string, _ job.CreateJobRequest) (job.ExecutionResult, error) {
	<-r.release
	return job.ExecutionResult{}, nil
}

func performRequest(t *testing.T, router http.Handler, method, path string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()

	req, err := http.NewRequest(method, path, body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}

	return performRequestRecorder(router, req)
}

func performRequestRecorder(router http.Handler, req *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}
