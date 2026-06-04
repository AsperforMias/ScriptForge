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
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "scriptforge.db")
	artifactDir := filepath.Join(tmpDir, "artifacts")

	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer repo.Close()

	artifactStore := artifact.New(artifactDir)
	runner := pipeline.NewRunner(artifactStore)
	service := job.NewService(slog.New(slog.NewTextHandler(io.Discard, nil)), repo, runner, artifactStore, 1)
	cfg := config.Load()
	cfg.SQLitePath = dbPath
	cfg.ArtifactDir = artifactDir

	server := httptest.NewServer(NewRouter(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), service))
	defer server.Close()

	requestBody, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "novels", "night-rain-request.json"))
	if err != nil {
		t.Fatalf("read request fixture: %v", err)
	}

	createResp, err := http.Post(server.URL+"/api/v1/jobs", "application/json", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatalf("post jobs: %v", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", createResp.StatusCode)
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
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/v1/jobs/"+jobID, nil)
		statusResp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("get job status: %v", err)
		}

		var statusEnvelope struct {
			Data struct {
				Job struct {
					Status string `json:"status"`
				} `json:"job"`
			} `json:"data"`
		}
		if err := json.NewDecoder(statusResp.Body).Decode(&statusEnvelope); err != nil {
			statusResp.Body.Close()
			t.Fatalf("decode status response: %v", err)
		}
		statusResp.Body.Close()

		if statusEnvelope.Data.Job.Status == "succeeded" {
			break
		}

		select {
		case <-ctx.Done():
			t.Fatal("job did not succeed before timeout")
		case <-time.After(50 * time.Millisecond):
		}
	}

	resultResp, err := http.Get(server.URL + "/api/v1/jobs/" + jobID + "/result")
	if err != nil {
		t.Fatalf("get job result: %v", err)
	}
	defer resultResp.Body.Close()

	if resultResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 result status, got %d", resultResp.StatusCode)
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

	exportResp, err := http.Get(server.URL + "/api/v1/jobs/" + jobID + "/export")
	if err != nil {
		t.Fatalf("export job result: %v", err)
	}
	defer exportResp.Body.Close()

	if exportResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 export status, got %d", exportResp.StatusCode)
	}
}
