package llm

import "strings"

func NewGenerator(cfg ProviderConfig) Generator {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "", "disabled":
		return NewUnavailableGenerator("LLM_PROVIDER is disabled")
	case "mock":
		return NewMockGenerator()
	default:
		return NewUnavailableGenerator("provider adapter is not implemented yet")
	}
}
