package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/AsperforMias/ScriptForge/backend/internal/ingest"
	"github.com/AsperforMias/ScriptForge/backend/internal/job"
	"github.com/AsperforMias/ScriptForge/backend/internal/llm"
	"github.com/AsperforMias/ScriptForge/backend/internal/screenplay"
	"github.com/AsperforMias/ScriptForge/backend/internal/storage/artifact"
	"github.com/AsperforMias/ScriptForge/backend/internal/workflow"
)

var stageOrder = []string{
	"ingest",
	"outline",
	"entities",
	"scene_planning",
	"screenplay_generation",
	"validation",
	"persistence",
}

type Runner struct {
	artifacts    *artifact.Store
	llmGenerator llm.Generator
}

func NewRunner(artifacts *artifact.Store, llmGenerator llm.Generator) *Runner {
	if llmGenerator == nil {
		llmGenerator = llm.NewUnavailableGenerator("no llm generator configured")
	}

	return &Runner{
		artifacts:    artifacts,
		llmGenerator: llmGenerator,
	}
}

func (r *Runner) Run(ctx context.Context, jobID string, req job.CreateJobRequest) (job.ExecutionResult, error) {
	stages := queuedStages()

	inputPath, err := r.artifacts.WriteInputSnapshot(jobID, req)
	if err != nil {
		return failResult(stages, "ingest", err), err
	}

	markStageRunning(stages, "ingest")
	source := ingest.Normalize(req)
	normalizedPath, err := r.artifacts.WriteNormalizedSource(jobID, source)
	if err != nil {
		return failResult(stages, "ingest", err), err
	}
	markStageSucceeded(stages, "ingest")

	select {
	case <-ctx.Done():
		return failResult(stages, "outline", ctx.Err()), ctx.Err()
	default:
	}

	markStageRunning(stages, "outline")
	outline := workflow.BuildOutline(source)
	markStageSucceeded(stages, "outline")

	markStageRunning(stages, "entities")
	entities := workflow.ExtractEntities(source)
	markStageSucceeded(stages, "entities")

	markStageRunning(stages, "scene_planning")
	plan := workflow.BuildScenePlan(source, outline, entities)
	markStageSucceeded(stages, "scene_planning")

	markStageRunning(stages, "screenplay_generation")
	document, providerDebug, generationWarnings, err := r.generateDocument(ctx, jobID, req, source, outline, entities, plan)
	if err != nil {
		return failResult(stages, "screenplay_generation", err), err
	}
	markStageSucceeded(stages, "screenplay_generation")

	markStageRunning(stages, "validation")
	document.Validation.Warnings = mergeWarnings(document.Validation.Warnings, generationWarnings)
	validated, err := screenplay.ValidateAndSerialize(document)
	if err != nil {
		return failResult(stages, "validation", err), err
	}
	markStageSucceeded(stages, "validation")

	markStageRunning(stages, "persistence")
	providerDebugPath := ""
	if providerDebug != nil {
		providerDebugPath, err = r.artifacts.WriteProviderDebug(jobID, providerDebug)
		if err != nil {
			return failResult(stages, "persistence", err), err
		}
	}
	yamlPath, err := r.artifacts.WriteYAML(jobID, validated.YAMLText)
	if err != nil {
		return failResult(stages, "persistence", err), err
	}
	markStageSucceeded(stages, "persistence")

	return job.ExecutionResult{
		Document:             validated.Document,
		YAMLText:             validated.YAMLText,
		InputSnapshotPath:    inputPath,
		NormalizedSourcePath: normalizedPath,
		ProviderDebugPath:    providerDebugPath,
		YAMLPath:             yamlPath,
		Warnings:             mergeWarnings(generationWarnings, validated.Warnings),
		Stages:               stages,
		CurrentStage:         "persistence",
	}, nil
}

func queuedStages() []job.Stage {
	stages := make([]job.Stage, 0, len(stageOrder))
	for _, name := range stageOrder {
		stages = append(stages, job.Stage{Name: name, Status: "queued"})
	}
	return stages
}

func markStageRunning(stages []job.Stage, stageName string) {
	for idx := range stages {
		if stages[idx].Name == stageName {
			stages[idx].Status = "running"
			stages[idx].StartedAt = time.Now().UTC().Format(time.RFC3339)
			return
		}
	}
}

func (r *Runner) generateDocument(ctx context.Context, jobID string, req job.CreateJobRequest, source ingest.NormalizedSource, outline workflow.OutlineBundle, entities workflow.EntityBundle, plan workflow.ScenePlan) (screenplay.Document, *llm.DebugSnapshot, []string, error) {
	if req.Generation.Mode != "llm" {
		return workflow.BuildDocument(req, source, outline, entities, plan), nil, nil, nil
	}

	result, err := r.llmGenerator.Generate(ctx, llm.GenerateRequest{
		JobID:    jobID,
		Input:    req,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err != nil {
		fallbackDocument := workflow.BuildDocument(req, source, outline, entities, plan)
		warnings := []string{
			fmt.Sprintf("llm generation via %s failed; fell back to deterministic baseline: %s", r.llmGenerator.Name(), err.Error()),
		}
		return fallbackDocument, nil, warnings, nil
	}
	if len(result.Warnings) > 0 && len(result.Document.Validation.Warnings) == 0 {
		result.Document.Validation.Warnings = append(result.Document.Validation.Warnings, result.Warnings...)
	}
	return result.Document, result.Debug, result.Warnings, nil
}

func mergeWarnings(parts ...[]string) []string {
	seen := map[string]struct{}{}
	merged := make([]string, 0)
	for _, values := range parts {
		for _, value := range values {
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			merged = append(merged, value)
		}
	}
	return merged
}

func markStageSucceeded(stages []job.Stage, stageName string) {
	for idx := range stages {
		if stages[idx].Name == stageName {
			stages[idx].Status = "succeeded"
			if stages[idx].StartedAt == "" {
				stages[idx].StartedAt = time.Now().UTC().Format(time.RFC3339)
			}
			stages[idx].FinishedAt = time.Now().UTC().Format(time.RFC3339)
			return
		}
	}
}

func failResult(stages []job.Stage, stageName string, err error) job.ExecutionResult {
	for idx := range stages {
		if stages[idx].Name == stageName {
			if stages[idx].StartedAt == "" {
				stages[idx].StartedAt = time.Now().UTC().Format(time.RFC3339)
			}
			stages[idx].Status = "failed"
			stages[idx].ErrorMessage = err.Error()
			stages[idx].FinishedAt = time.Now().UTC().Format(time.RFC3339)
			break
		}
	}
	return job.ExecutionResult{
		Stages:       stages,
		CurrentStage: stageName,
	}
}
