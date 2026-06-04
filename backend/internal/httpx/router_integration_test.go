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
	"github.com/AsperforMias/ScriptForge/backend/internal/pipeline"
	"github.com/AsperforMias/ScriptForge/backend/internal/storage/artifact"
	"github.com/AsperforMias/ScriptForge/backend/internal/storage/sqlite"
)

func TestRouterJobLifecycleWithFixtures(t *testing.T) {
	router, _ := newTestHarness(t, pipeline.NewRunner(artifact.New(filepath.Join(t.TempDir(), "artifacts"))))

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
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/jobs/"+jobID, nil)
		if err != nil {
			t.Fatalf("build status request: %v", err)
		}
		statusResp := performRequestRecorder(router, req)

		var statusEnvelope struct {
			Data struct {
				Job struct {
					Status          string `json:"status"`
					ProgressPercent int    `json:"progress_percent"`
					ErrorMessage    string `json:"error_message"`
				} `json:"job"`
			} `json:"data"`
		}
		if err := json.NewDecoder(statusResp.Body).Decode(&statusEnvelope); err != nil {
			t.Fatalf("decode status response: %v", err)
		}

		if statusEnvelope.Data.Job.Status == "succeeded" {
			if statusEnvelope.Data.Job.ProgressPercent != 100 {
				t.Fatalf("expected succeeded job progress to be 100, got %d", statusEnvelope.Data.Job.ProgressPercent)
			}
			break
		}
		if statusEnvelope.Data.Job.Status == "failed" {
			t.Fatalf("job failed unexpectedly: %s", statusEnvelope.Data.Job.ErrorMessage)
		}

		select {
		case <-ctx.Done():
			t.Fatal("job did not succeed before timeout")
		case <-time.After(50 * time.Millisecond):
		}
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
	router, _ := newTestHarness(t, pipeline.NewRunner(artifact.New(filepath.Join(t.TempDir(), "artifacts"))))

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
