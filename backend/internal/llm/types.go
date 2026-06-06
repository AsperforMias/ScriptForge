package llm

import (
	"context"

	"github.com/AsperforMias/ScriptForge/backend/internal/ingest"
	"github.com/AsperforMias/ScriptForge/backend/internal/job"
	"github.com/AsperforMias/ScriptForge/backend/internal/screenplay"
	"github.com/AsperforMias/ScriptForge/backend/internal/workflow"
)

type ProviderConfig struct {
	Provider       string
	Model          string
	BaseURL        string
	APIKey         string
	RequestTimeout string
}

type GenerateRequest struct {
	JobID    string
	Input    job.CreateJobRequest
	Source   ingest.NormalizedSource
	Outline  workflow.OutlineBundle
	Entities workflow.EntityBundle
	Plan     workflow.ScenePlan
}

type GenerateResult struct {
	Document screenplay.Document
	Warnings []string
	Debug    *DebugSnapshot
}

type DebugSnapshot struct {
	Provider           string   `json:"provider"`
	Model              string   `json:"model,omitempty"`
	ParseMode          string   `json:"parse_mode,omitempty"`
	RawContent         string   `json:"raw_content"`
	OriginalRawContent string   `json:"original_raw_content,omitempty"`
	RetryRawContents   []string `json:"retry_raw_contents,omitempty"`
	ParseErrors        []string `json:"parse_errors,omitempty"`
}

type Generator interface {
	Name() string
	Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error)
}
