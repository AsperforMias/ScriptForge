package llm

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/AsperforMias/ScriptForge/backend/internal/ingest"
	"github.com/AsperforMias/ScriptForge/backend/internal/job"
	"github.com/AsperforMias/ScriptForge/backend/internal/workflow"
)

func TestNewGeneratorReturnsUnavailableForIncompleteOpenAICompatibleConfig(t *testing.T) {
	generator := NewGenerator(ProviderConfig{Provider: "openai_compatible", BaseURL: "https://example.com/v1"})
	if generator.Name() != "disabled" {
		t.Fatalf("expected unavailable generator fallback, got %s", generator.Name())
	}
}

func TestOpenAICompatibleGeneratorParsesYAMLResponse(t *testing.T) {
	input := validOpenAICompatibleCreateJobRequest()
	source := ingest.Normalize(input)
	outline := workflow.BuildOutline(source)
	entities := workflow.ExtractEntities(source)
	plan := workflow.BuildScenePlan(source, outline, entities)

	generator := NewOpenAICompatibleGenerator(ProviderConfig{
		Provider:       "openai_compatible",
		BaseURL:        "https://example.com/v1",
		Model:          "demo-model",
		APIKey:         "demo-key",
		RequestTimeout: "5s",
	}).(*OpenAICompatibleGenerator)
	generator.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("Authorization"); got != "Bearer demo-key" {
				t.Fatalf("unexpected auth header: %s", got)
			}
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			if !strings.Contains(string(body), "\"model\":\"demo-model\"") {
				t.Fatalf("expected model in request body, got %s", string(body))
			}

			responseBody := `{"choices":[{"message":{"content":"version: \"1.0\"\nsource:\n  title: \"Night Rain\"\n  author: \"Demo Author\"\n  language: \"zh-CN\"\n  chapter_count: 3\n  chapters:\n    - index: 1\n      title: \"Chapter 1\"\n      summary: \"Summary 1\"\n    - index: 2\n      title: \"Chapter 2\"\n      summary: \"Summary 2\"\n    - index: 3\n      title: \"Chapter 3\"\n      summary: \"Summary 3\"\nadaptation:\n  style: \"Suspense Drama\"\n  audience: \"General\"\n  notes: []\ncharacters:\n  - id: \"char_linqi\"\n    name: \"林琪\"\n    role: \"protagonist\"\n    description: \"Main character\"\nlocations:\n  - id: \"loc_station\"\n    name: \"车站\"\n    description: \"Key location\"\nscenes:\n  - id: \"scene_001\"\n    title: \"Chapter 1\"\n    source_chapters: [1]\n    slugline:\n      interior_exterior: \"INT\"\n      location_id: \"loc_station\"\n      time: \"NIGHT\"\n    summary: \"Scene summary\"\n    objective: \"Objective\"\n    beats:\n      - type: \"action\"\n        content: \"Action beat\"\n    notes:\n      adaptation_reason: \"Reason\"\n      open_questions: []\nvalidation:\n  status: \"passed\"\n  warnings: []"}}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(responseBody)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	result, err := generator.Generate(context.Background(), GenerateRequest{
		JobID:    "job_openai_like",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err != nil {
		t.Fatalf("unexpected generator error: %v", err)
	}
	if result.Document.Source.Title != "Night Rain" {
		t.Fatalf("unexpected source title: %s", result.Document.Source.Title)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected provider warning")
	}
}

func validOpenAICompatibleCreateJobRequest() job.CreateJobRequest {
	var req job.CreateJobRequest
	req.Source.Title = "Night Rain"
	req.Source.Author = "Demo Author"
	req.Adaptation.Style = "Suspense Drama"
	req.Adaptation.Audience = "General"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "Chapter 1", Content: "林琪深夜回到公寓，发现门锁似乎被动过。"},
		{Index: 2, Title: "Chapter 2", Content: "她在房间里找到一张陌生字条，怀疑有人潜入。"},
		{Index: 3, Title: "Chapter 3", Content: "第二天清晨，她决定顺着线索前往车站。"},
	}
	return req
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
