package llm

import (
	"context"

	"github.com/AsperforMias/ScriptForge/backend/internal/workflow"
)

type MockGenerator struct{}

func NewMockGenerator() Generator {
	return MockGenerator{}
}

func (MockGenerator) Name() string {
	return "mock"
}

func (MockGenerator) Generate(_ context.Context, req GenerateRequest) (GenerateResult, error) {
	document := workflow.BuildDocument(req.Input, req.Source, req.Outline, req.Entities, req.Plan)
	document.Validation.Warnings = append(document.Validation.Warnings, "generated via mock llm provider")

	return GenerateResult{
		Document: document,
		Warnings: document.Validation.Warnings,
	}, nil
}
