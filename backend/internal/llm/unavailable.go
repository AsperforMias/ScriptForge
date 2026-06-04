package llm

import (
	"context"
	"fmt"
)

type UnavailableGenerator struct {
	reason string
}

func NewUnavailableGenerator(reason string) Generator {
	return UnavailableGenerator{reason: reason}
}

func (g UnavailableGenerator) Name() string {
	return "disabled"
}

func (g UnavailableGenerator) Generate(_ context.Context, _ GenerateRequest) (GenerateResult, error) {
	return GenerateResult{}, NewUnavailableError(fmt.Sprintf("llm provider is unavailable: %s", g.reason))
}
