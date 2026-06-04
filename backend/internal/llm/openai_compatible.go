package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/AsperforMias/ScriptForge/backend/internal/screenplay"
)

type OpenAICompatibleGenerator struct {
	baseURL    string
	model      string
	apiKey     string
	httpClient *http.Client
}

type chatCompletionsRequest struct {
	Model       string                `json:"model"`
	Temperature float64               `json:"temperature"`
	Messages    []chatCompletionInput `json:"messages"`
}

type chatCompletionInput struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionsResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewOpenAICompatibleGenerator(cfg ProviderConfig) Generator {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	model := strings.TrimSpace(cfg.Model)
	apiKey := strings.TrimSpace(cfg.APIKey)
	if baseURL == "" || model == "" || apiKey == "" {
		return NewUnavailableGenerator("openai_compatible provider requires LLM_BASE_URL, LLM_MODEL, and LLM_API_KEY")
	}

	timeout := 45 * time.Second
	if parsed, err := time.ParseDuration(cfg.RequestTimeout); err == nil && parsed > 0 {
		timeout = parsed
	}

	return &OpenAICompatibleGenerator{
		baseURL: baseURL,
		model:   model,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (g *OpenAICompatibleGenerator) Name() string {
	return "openai_compatible"
}

func (g *OpenAICompatibleGenerator) Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error) {
	requestBody, err := g.buildRequest(req)
	if err != nil {
		return GenerateResult{}, NewInvocationError(g.Name(), err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(g.baseURL, "/")+"/chat/completions", bytes.NewReader(requestBody))
	if err != nil {
		return GenerateResult{}, NewInvocationError(g.Name(), err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return GenerateResult{}, NewInvocationError(g.Name(), err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return GenerateResult{}, NewInvocationError(g.Name(), err)
	}

	if resp.StatusCode >= 300 {
		return GenerateResult{}, NewInvocationError(g.Name(), fmt.Errorf("status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(payload))))
	}

	var completion chatCompletionsResponse
	if err := json.Unmarshal(payload, &completion); err != nil {
		return GenerateResult{}, NewInvocationError(g.Name(), err)
	}
	if completion.Error != nil {
		return GenerateResult{}, NewInvocationError(g.Name(), errors.New(completion.Error.Message))
	}
	if len(completion.Choices) == 0 {
		return GenerateResult{}, NewInvocationError(g.Name(), fmt.Errorf("empty choices"))
	}

	yamlText := strings.TrimSpace(completion.Choices[0].Message.Content)
	document, err := screenplay.ParseYAML(yamlText)
	if err != nil {
		return GenerateResult{}, NewInvocationError(g.Name(), fmt.Errorf("parse yaml response: %w", err))
	}

	document.Validation.Warnings = append(document.Validation.Warnings, "generated via openai_compatible provider")

	return GenerateResult{
		Document: document,
		Warnings: document.Validation.Warnings,
	}, nil
}

func (g *OpenAICompatibleGenerator) buildRequest(req GenerateRequest) ([]byte, error) {
	contextPayload := map[string]any{
		"source":   req.Source,
		"outline":  req.Outline,
		"entities": req.Entities,
		"plan":     req.Plan,
		"target": map[string]any{
			"style":    req.Input.Adaptation.Style,
			"audience": req.Input.Adaptation.Audience,
			"notes":    req.Input.Adaptation.Notes,
		},
	}

	contextJSON, err := json.MarshalIndent(contextPayload, "", "  ")
	if err != nil {
		return nil, err
	}

	requestBody := chatCompletionsRequest{
		Model:       g.model,
		Temperature: 0.2,
		Messages: []chatCompletionInput{
			{
				Role: "system",
				Content: "You adapt Chinese novels into structured screenplay YAML. " +
					"Return only valid YAML that matches the required screenplay schema. " +
					"Do not include markdown fences or explanations.",
			},
			{
				Role: "user",
				Content: "Generate a screenplay YAML document from this structured context:\n" +
					string(contextJSON) +
					"\nEnsure source.chapter_count matches the input, every scene references valid chapter indexes, and dialogue beats reference valid character_id values.",
			},
		},
	}

	return json.Marshal(requestBody)
}
