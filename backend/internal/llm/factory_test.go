package llm

import (
	"context"
	"testing"

	"github.com/AsperforMias/ScriptForge/backend/internal/ingest"
	"github.com/AsperforMias/ScriptForge/backend/internal/job"
	"github.com/AsperforMias/ScriptForge/backend/internal/workflow"
)

func TestNewGeneratorReturnsUnavailableWhenDisabled(t *testing.T) {
	generator := NewGenerator(ProviderConfig{Provider: "disabled"})
	if generator.Name() != "disabled" {
		t.Fatalf("expected disabled generator, got %s", generator.Name())
	}

	_, err := generator.Generate(context.Background(), GenerateRequest{})
	if err == nil {
		t.Fatal("expected unavailable error")
	}
}

func TestNewGeneratorReturnsMockProvider(t *testing.T) {
	input := validCreateJobRequest()
	source := ingest.Normalize(input)
	outline := workflow.BuildOutline(source)
	entities := workflow.ExtractEntities(source)
	plan := workflow.BuildScenePlan(source, outline, entities)

	generator := NewGenerator(ProviderConfig{Provider: "mock"})
	result, err := generator.Generate(context.Background(), GenerateRequest{
		JobID:    "job_mock",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err != nil {
		t.Fatalf("unexpected mock generator error: %v", err)
	}
	if len(result.Document.Scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(result.Document.Scenes))
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warning from mock generator")
	}
}

func validCreateJobRequest() job.CreateJobRequest {
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
