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
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/AsperforMias/ScriptForge/backend/internal/config"
	"github.com/AsperforMias/ScriptForge/backend/internal/job"
	"github.com/AsperforMias/ScriptForge/backend/internal/llm"
	"github.com/AsperforMias/ScriptForge/backend/internal/pipeline"
	"github.com/AsperforMias/ScriptForge/backend/internal/screenplay"
	"github.com/AsperforMias/ScriptForge/backend/internal/storage/artifact"
	"github.com/AsperforMias/ScriptForge/backend/internal/storage/sqlite"
	"gopkg.in/yaml.v3"
)

func TestRouterJobLifecycleWithFixtures(t *testing.T) {
	router, repo, _ := newTestHarness(t, pipeline.NewRunner(artifact.New(filepath.Join(t.TempDir(), "artifacts")), llm.NewUnavailableGenerator("deterministic mode does not use llm")))

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

	if !yamlDocumentsEqual(resultEnvelope.Data.YAMLText, string(expectedYAML)) {
		t.Fatalf("unexpected yaml output\nexpected:\n%s\n\ngot:\n%s", string(expectedYAML), resultEnvelope.Data.YAMLText)
	}

	exportResp := performRequest(t, router, http.MethodGet, "/api/v1/jobs/"+jobID+"/export", nil)

	if exportResp.Code != http.StatusOK {
		t.Fatalf("expected 200 export status, got %d", exportResp.Code)
	}
}

func TestRouterJobLifecycleWithCustomChineseInput(t *testing.T) {
	router, repo, _ := newTestHarness(t, pipeline.NewRunner(artifact.New(filepath.Join(t.TempDir(), "artifacts")), llm.NewUnavailableGenerator("deterministic mode does not use llm")))

	requestBody, err := json.Marshal(customChineseCreateJobRequest())
	if err != nil {
		t.Fatalf("marshal custom request: %v", err)
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

	waitForSucceededJob(t, repo, jobID)

	resultResp := performRequest(t, router, http.MethodGet, "/api/v1/jobs/"+jobID+"/result", nil)
	if resultResp.Code != http.StatusOK {
		t.Fatalf("expected 200 result status, got %d", resultResp.Code)
	}

	var resultEnvelope struct {
		Data struct {
			Screenplay screenplay.Document `json:"screenplay"`
			YAMLText   string              `json:"yaml_text"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resultResp.Body).Decode(&resultEnvelope); err != nil {
		t.Fatalf("decode result response: %v", err)
	}
	if err := screenplay.Validate(resultEnvelope.Data.Screenplay); err != nil {
		t.Fatalf("expected valid screenplay from custom request: %v", err)
	}
	if len(resultEnvelope.Data.Screenplay.Scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(resultEnvelope.Data.Screenplay.Scenes))
	}
	if resultEnvelope.Data.Screenplay.Characters[0].Name != "主角" {
		t.Fatalf("expected weak entity fallback protagonist 主角, got %s", resultEnvelope.Data.Screenplay.Characters[0].Name)
	}

	objectives := map[string]struct{}{}
	openQuestions := map[string]struct{}{}
	for idx, location := range resultEnvelope.Data.Screenplay.Locations {
		if strings.Contains(location.Name, "Chapter ") {
			t.Fatalf("expected localized location fallback for chapter %d, got %s", idx+1, location.Name)
		}
	}
	for idx, scene := range resultEnvelope.Data.Screenplay.Scenes {
		if strings.TrimSpace(scene.Objective) == "" {
			t.Fatalf("expected objective for custom scene %d", idx+1)
		}
		objectives[scene.Objective] = struct{}{}
		if len(scene.Notes.OpenQuestions) == 0 {
			t.Fatalf("expected open question for custom scene %d", idx+1)
		}
		openQuestions[scene.Notes.OpenQuestions[0]] = struct{}{}
	}
	if len(objectives) != len(resultEnvelope.Data.Screenplay.Scenes) {
		t.Fatalf("expected unique objectives, got %d unique for %d scenes", len(objectives), len(resultEnvelope.Data.Screenplay.Scenes))
	}
	if len(openQuestions) != len(resultEnvelope.Data.Screenplay.Scenes) {
		t.Fatalf("expected unique open questions, got %d unique for %d scenes", len(openQuestions), len(resultEnvelope.Data.Screenplay.Scenes))
	}

	var yamlDoc screenplay.Document
	if err := yaml.Unmarshal([]byte(resultEnvelope.Data.YAMLText), &yamlDoc); err != nil {
		t.Fatalf("unmarshal yaml text: %v", err)
	}
	if !reflect.DeepEqual(yamlDoc, resultEnvelope.Data.Screenplay) {
		t.Fatalf("expected api screenplay json and yaml_text to describe the same document")
	}
}

func TestRouterJobLifecycleWithRealisticCustomSuspenseInput(t *testing.T) {
	router, repo, _ := newTestHarness(t, pipeline.NewRunner(artifact.New(filepath.Join(t.TempDir(), "artifacts")), llm.NewUnavailableGenerator("deterministic mode does not use llm")))

	requestBody, err := json.Marshal(customSuspenseCreateJobRequest())
	if err != nil {
		t.Fatalf("marshal suspense request: %v", err)
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

	waitForSucceededJob(t, repo, createEnvelope.Data.Job.ID)

	resultResp := performRequest(t, router, http.MethodGet, "/api/v1/jobs/"+createEnvelope.Data.Job.ID+"/result", nil)
	if resultResp.Code != http.StatusOK {
		t.Fatalf("expected 200 result status, got %d", resultResp.Code)
	}

	var resultEnvelope struct {
		Data struct {
			Screenplay screenplay.Document `json:"screenplay"`
			YAMLText   string              `json:"yaml_text"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resultResp.Body).Decode(&resultEnvelope); err != nil {
		t.Fatalf("decode result response: %v", err)
	}
	if err := screenplay.Validate(resultEnvelope.Data.Screenplay); err != nil {
		t.Fatalf("expected valid screenplay from suspense request: %v", err)
	}
	assertCustomSuspenseScreenplay(t, resultEnvelope.Data.Screenplay)

	var yamlDoc screenplay.Document
	if err := yaml.Unmarshal([]byte(resultEnvelope.Data.YAMLText), &yamlDoc); err != nil {
		t.Fatalf("unmarshal yaml text: %v", err)
	}
	if !reflect.DeepEqual(yamlDoc, resultEnvelope.Data.Screenplay) {
		t.Fatalf("expected api screenplay json and yaml_text to describe the same document")
	}
}

func TestRouterJobLifecycleWithFamilyWordSuspenseInput(t *testing.T) {
	router, repo, _ := newTestHarness(t, pipeline.NewRunner(artifact.New(filepath.Join(t.TempDir(), "artifacts")), llm.NewUnavailableGenerator("deterministic mode does not use llm")))

	requestBody, err := json.Marshal(familyWordSuspenseCreateJobRequest())
	if err != nil {
		t.Fatalf("marshal suspense request: %v", err)
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

	waitForSucceededJob(t, repo, createEnvelope.Data.Job.ID)

	resultResp := performRequest(t, router, http.MethodGet, "/api/v1/jobs/"+createEnvelope.Data.Job.ID+"/result", nil)
	if resultResp.Code != http.StatusOK {
		t.Fatalf("expected 200 result status, got %d", resultResp.Code)
	}

	var resultEnvelope struct {
		Data struct {
			Screenplay screenplay.Document `json:"screenplay"`
			YAMLText   string              `json:"yaml_text"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resultResp.Body).Decode(&resultEnvelope); err != nil {
		t.Fatalf("decode result response: %v", err)
	}
	if err := screenplay.Validate(resultEnvelope.Data.Screenplay); err != nil {
		t.Fatalf("expected valid screenplay from suspense request: %v", err)
	}
	assertFamilyWordSuspenseScreenplay(t, resultEnvelope.Data.Screenplay)

	var yamlDoc screenplay.Document
	if err := yaml.Unmarshal([]byte(resultEnvelope.Data.YAMLText), &yamlDoc); err != nil {
		t.Fatalf("unmarshal yaml text: %v", err)
	}
	if !reflect.DeepEqual(yamlDoc, resultEnvelope.Data.Screenplay) {
		t.Fatalf("expected api screenplay json and yaml_text to describe the same document")
	}
}

func TestRouterRejectsInvalidChapterCount(t *testing.T) {
	router, _, _ := newTestHarness(t, pipeline.NewRunner(artifact.New(filepath.Join(t.TempDir(), "artifacts")), llm.NewUnavailableGenerator("deterministic mode does not use llm")))

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
	router, _, _ := newTestHarness(t, runner)
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

	exportResp := performRequest(t, router, http.MethodGet, "/api/v1/jobs/"+createEnvelope.Data.Job.ID+"/export", nil)
	if exportResp.Code != http.StatusConflict {
		t.Fatalf("expected export 409, got %d", exportResp.Code)
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

func TestRouterReturnsNotFoundForUnknownJobEndpoints(t *testing.T) {
	router, _, _ := newTestHarness(t, pipeline.NewRunner(artifact.New(filepath.Join(t.TempDir(), "artifacts")), llm.NewUnavailableGenerator("deterministic mode does not use llm")))

	paths := []string{
		"/api/v1/jobs/job_missing",
		"/api/v1/jobs/job_missing/result",
		"/api/v1/jobs/job_missing/export",
	}

	for _, path := range paths {
		resp := performRequest(t, router, http.MethodGet, path, nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404 for %s, got %d", path, resp.Code)
		}

		var envelope struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
			t.Fatalf("decode not found response for %s: %v", path, err)
		}
		if envelope.Error.Code != "job_not_found" {
			t.Fatalf("expected job_not_found for %s, got %s", path, envelope.Error.Code)
		}
	}
}

func TestRouterReturnsGenerationFailedForFailedJobResult(t *testing.T) {
	router, repo, _ := newTestHarness(t, pipeline.NewRunner(artifact.New(filepath.Join(t.TempDir(), "artifacts")), llm.NewUnavailableGenerator("deterministic mode does not use llm")))

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

func TestRouterReturnsWarningsForMockLLMResult(t *testing.T) {
	router, repo, artifactDir := newTestHarness(t, pipeline.NewRunner(artifact.New(filepath.Join(t.TempDir(), "artifacts")), llm.NewMockGenerator()))

	request := validMockLLMCreateJobRequest()
	mockArtifacts := artifact.New(filepath.Join(t.TempDir(), "mock-provider"))
	runner := pipeline.NewRunner(mockArtifacts, llm.NewMockGenerator())
	runResult, err := runner.Run(context.Background(), "job_llm_seed", request)
	if err != nil {
		t.Fatalf("run mock llm pipeline: %v", err)
	}

	jobArtifactStore := artifact.New(artifactDir)
	yamlPath, err := jobArtifactStore.WriteYAML("job_llm_seed", runResult.YAMLText)
	if err != nil {
		t.Fatalf("write seeded yaml artifact: %v", err)
	}

	record := job.Job{
		ID:                "job_llm_seed",
		Status:            "succeeded",
		CurrentStage:      "persistence",
		ProgressPercent:   100,
		SourceTitle:       request.Source.Title,
		GenerationMode:    "llm",
		Warnings:          runResult.Warnings,
		InputSnapshotPath: runResult.InputSnapshotPath,
		ResultYAMLPath:    yamlPath,
		CreatedAt:         "2026-06-05T00:00:00Z",
		UpdatedAt:         "2026-06-05T00:00:10Z",
	}
	stages := []job.Stage{
		{Name: "ingest", Status: "succeeded"},
		{Name: "outline", Status: "succeeded"},
		{Name: "entities", Status: "succeeded"},
		{Name: "scene_planning", Status: "succeeded"},
		{Name: "screenplay_generation", Status: "succeeded"},
		{Name: "validation", Status: "succeeded"},
		{Name: "persistence", Status: "succeeded"},
	}
	if err := repo.CreateJob(context.Background(), record, stages); err != nil {
		t.Fatalf("seed llm job: %v", err)
	}
	if err := repo.SaveArtifact(context.Background(), job.Artifact{
		JobID:         record.ID,
		YAMLPath:      yamlPath,
		YAMLSizeBytes: len(runResult.YAMLText),
		CreatedAt:     "2026-06-05T00:00:10Z",
	}); err != nil {
		t.Fatalf("seed llm artifact: %v", err)
	}

	resultResp := performRequest(t, router, http.MethodGet, "/api/v1/jobs/"+record.ID+"/result", nil)
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

func newTestHarness(t *testing.T, runner job.Runner) (http.Handler, *sqlite.Store, string) {
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

	return router, repo, artifactDir
}

func waitForSucceededJob(t *testing.T, repo *sqlite.Store, jobID string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for {
		record, err := repo.GetJob(ctx, jobID)
		if err == nil && record.Status == "succeeded" {
			return
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

func validMockLLMCreateJobRequest() job.CreateJobRequest {
	var req job.CreateJobRequest
	req.Source.Title = "Night Rain"
	req.Source.Author = "Demo Author"
	req.Adaptation.Style = "Suspense Drama"
	req.Adaptation.Audience = "General"
	req.Adaptation.Notes = []string{"Keep a strong hook in each scene"}
	req.Generation.Mode = "llm"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "Chapter 1", Content: "林琪深夜回到公寓，发现门锁似乎被动过。"},
		{Index: 2, Title: "Chapter 2", Content: "她在房间里找到一张陌生字条，怀疑有人潜入。"},
		{Index: 3, Title: "Chapter 3", Content: "第二天清晨，她决定顺着线索前往车站。"},
	}
	return req
}

func familyWordSuspenseCreateJobRequest() job.CreateJobRequest {
	var req job.CreateJobRequest
	req.Source.Title = "旧宅回声"
	req.Source.Author = "自定义作者"
	req.Adaptation.Style = "悬疑现实短剧"
	req.Adaptation.Audience = "青年向"
	req.Adaptation.Notes = []string{"家庭词不能盖过当前线索", "同章多线索时围绕主证据推进"}
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "第一章 客厅回放", Content: "闻溪回到父亲留下的家里，在旧客厅收拾遗物时听见随身听里多出一段陌生口哨。郑岚在里屋催她先吃饭，但闻溪只想先把录音倒回去，确认那段声音究竟录自哪一天。"},
		{Index: 2, Title: "第二章 楼道纸灰", Content: "第二天傍晚，闻溪在自家楼道发现烧过的纸灰和一张写着仓库编号的便签。老秦说父亲生前把备用钥匙交给过一个陌生快递员，闻溪决定先去核对编号，再查钥匙落到了谁手里。"},
		{Index: 3, Title: "第三章 仓库试锁", Content: "夜里，闻溪赶到江边旧仓库，用找到的钥匙去试开侧门。门内传来的拖拽声让她意识到，有人正赶在她之前转移父亲留下的箱子。"},
	}
	return req
}

func customChineseCreateJobRequest() job.CreateJobRequest {
	var req job.CreateJobRequest
	req.Source.Title = "雾港录音带"
	req.Source.Author = "自定义作者"
	req.Adaptation.Style = "悬疑现实短剧"
	req.Adaptation.Audience = "青年向"
	req.Adaptation.Notes = []string{"保留迟疑感", "突出线索递进"}
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "第一章 录音失真", Content: "暴雨落了一整夜，旧录音里突然多出一段陌生笑声。叙述者不敢立刻重播，只能先把磁带锁进抽屉。"},
		{Index: 2, Title: "第二章 匿名留言", Content: "第二天下午，留言板上多出一行约见时间，没人承认写过它。叙述者决定先核对录音来源，再去找留下字的人。"},
		{Index: 3, Title: "第三章 钟楼扑空", Content: "傍晚，叙述者带着磁带赶到老城区的旧钟楼，却发现约见人已经提前离开，只留下一把钥匙。"},
	}
	return req
}

func customSuspenseCreateJobRequest() job.CreateJobRequest {
	var req job.CreateJobRequest
	req.Source.Title = "回潮暗线"
	req.Source.Author = "自定义作者"
	req.Adaptation.Style = "悬疑现实短剧"
	req.Adaptation.Audience = "青年向"
	req.Adaptation.Notes = []string{"以当前章节证据驱动场景目标", "避免凭空补车站线索"}
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "第一章 旧客厅录音", Content: "沈砚回到父亲留下的旧客厅整理遗物，听见录音机里多出一段夹着潮声的陌生对话。她不敢惊动家里其他人，只想先确认那段录音是不是被人动过。"},
		{Index: 2, Title: "第二章 河堤碰面", Content: "第二天傍晚，沈砚按匿名留言赶到河堤，发现纸条提到的线索指向废弃船坞，而不是任何车站。她决定先弄清是谁把钥匙塞进自己口袋，再判断这场约见是不是圈套。"},
		{Index: 3, Title: "第三章 船坞试锁", Content: "夜里，沈砚独自走进废弃船坞，用那把生锈钥匙去试库房侧门的锁孔。门后传来的撞击声让她意识到，真正想藏起来的东西还在里面。"},
	}
	return req
}

func assertCustomSuspenseScreenplay(t *testing.T, doc screenplay.Document) {
	t.Helper()

	if len(doc.Scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(doc.Scenes))
	}
	if len(doc.Locations) != 3 {
		t.Fatalf("expected 3 locations, got %d", len(doc.Locations))
	}

	expectedLocations := []string{"客厅", "河堤", "船坞"}
	for idx, location := range doc.Locations {
		if location.Name != expectedLocations[idx] {
			t.Fatalf("expected location %s for chapter %d, got %s", expectedLocations[idx], idx+1, location.Name)
		}
	}

	if got := doc.Scenes[0].Objective; !containsAnyText(got, "录音") {
		t.Fatalf("expected scene 1 objective to mention recording clue, got %s", got)
	}
	if got := doc.Scenes[0].Objective; containsAnyText(got, "团圆饭", "误会说开", "和解") {
		t.Fatalf("expected scene 1 objective to avoid family template leakage, got %s", got)
	}
	if got := doc.Scenes[0].Beats[1].Content; !containsAnyText(got, "录音") {
		t.Fatalf("expected scene 1 dialogue to mention recording clue, got %s", got)
	}

	if got := doc.Scenes[1].Objective; containsAnyText(got, "车站", "寄信人") {
		t.Fatalf("expected scene 2 objective to avoid station template, got %s", got)
	}
	if got := doc.Scenes[1].Beats[1].Content; containsAnyText(got, "车站", "寄信人") {
		t.Fatalf("expected scene 2 dialogue to avoid station template, got %s", got)
	}
	if got := doc.Scenes[1].Notes.OpenQuestions[0]; containsAnyText(got, "车站", "寄信人") {
		t.Fatalf("expected scene 2 open question to avoid station template, got %s", got)
	}

	if got := doc.Scenes[2].Objective; !containsAnyText(got, "钥匙", "打开") {
		t.Fatalf("expected scene 3 objective to stay on key clue, got %s", got)
	}
	if got := doc.Scenes[2].Notes.OpenQuestions[0]; !containsAnyText(got, "钥匙", "门") {
		t.Fatalf("expected scene 3 open question to stay on key/door clue, got %s", got)
	}
}

func assertFamilyWordSuspenseScreenplay(t *testing.T, doc screenplay.Document) {
	t.Helper()

	if len(doc.Scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(doc.Scenes))
	}
	if len(doc.Characters) < 3 {
		t.Fatalf("expected multiple extracted characters, got %#v", doc.Characters)
	}

	expectedNames := []string{"闻溪", "郑岚", "老秦"}
	for _, expectedName := range expectedNames {
		found := false
		for _, character := range doc.Characters {
			if character.Name == expectedName {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected extracted characters to include %s, got %#v", expectedName, doc.Characters)
		}
	}

	if got := doc.Scenes[0].Objective; !containsAnyText(got, "录音", "声音") {
		t.Fatalf("expected scene 1 objective to stay on recording clue, got %s", got)
	}
	if got := doc.Scenes[0].Objective; containsAnyText(got, "团圆饭", "误会说开", "和解") {
		t.Fatalf("expected scene 1 objective to avoid family template leakage, got %s", got)
	}
	if got := doc.Scenes[1].Objective; !containsAnyText(got, "编号", "钥匙") {
		t.Fatalf("expected scene 2 objective to stay on current clue, got %s", got)
	}
	if got := doc.Scenes[1].Notes.OpenQuestions[0]; containsAnyText(got, "团圆饭", "和解", "误会说开") {
		t.Fatalf("expected scene 2 open question to avoid family template leakage, got %s", got)
	}
	if got := doc.Scenes[2].Objective; !containsAnyText(got, "钥匙", "打开", "仓库") {
		t.Fatalf("expected scene 3 objective to stay on key/warehouse action, got %s", got)
	}
}

func containsAnyText(input string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}
	return false
}

func yamlDocumentsEqual(actualYAML, expectedYAML string) bool {
	var actualDoc screenplay.Document
	if err := yaml.Unmarshal([]byte(actualYAML), &actualDoc); err != nil {
		return false
	}
	var expectedDoc screenplay.Document
	if err := yaml.Unmarshal([]byte(expectedYAML), &expectedDoc); err != nil {
		return false
	}
	return reflect.DeepEqual(actualDoc, expectedDoc)
}
