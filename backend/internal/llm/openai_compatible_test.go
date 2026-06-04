package llm

import (
	"context"
	"io"
	"net/http"
	"strconv"
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

func TestOpenAICompatibleGeneratorNormalizesVersionAndFences(t *testing.T) {
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
			responseBody := "```yaml\nversion: 1\nsource:\n  title: \"Night Rain\"\n  author: \"Demo Author\"\n  language: \"zh-CN\"\n  chapter_count: 3\n  chapters:\n    - index: 1\n      title: \"Chapter 1\"\n      summary: \"Summary 1\"\n    - index: 2\n      title: \"Chapter 2\"\n      summary: \"Summary 2\"\n    - index: 3\n      title: \"Chapter 3\"\n      summary: \"Summary 3\"\nadaptation:\n  style: \"Suspense Drama\"\n  audience: \"General\"\n  notes: []\ncharacters:\n  - id: \"char_linqi\"\n    name: \"林琪\"\n    role: \"protagonist\"\n    description: \"Main character\"\nlocations:\n  - id: \"loc_station\"\n    name: \"车站\"\n    description: \"Key location\"\nscenes:\n  - id: \"scene_001\"\n    title: \"Chapter 1\"\n    source_chapters: [1]\n    slugline:\n      interior_exterior: \"int\"\n      location_id: \"loc_station\"\n      time: \"night\"\n    summary: \"Scene summary\"\n    objective: \"Objective\"\n    beats:\n      - type: \"action\"\n        content: \"Action beat\"\n    notes:\n      adaptation_reason: \"Reason\"\n      open_questions: []\nvalidation:\n  status: \"\"\n  warnings: []\n```"
			payload := `{"choices":[{"message":{"content":` + strconv.Quote(responseBody) + `}}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(payload)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	result, err := generator.Generate(context.Background(), GenerateRequest{
		JobID:    "job_openai_like_fenced",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err != nil {
		t.Fatalf("unexpected generator error: %v", err)
	}
	if result.Document.Version != "1.0" {
		t.Fatalf("expected normalized version 1.0, got %s", result.Document.Version)
	}
	if result.Document.Validation.Status != "passed" {
		t.Fatalf("expected normalized validation status passed, got %s", result.Document.Validation.Status)
	}
	if result.Document.Scenes[0].Slugline.InteriorExterior != "INT" {
		t.Fatalf("expected normalized int/ext INT, got %s", result.Document.Scenes[0].Slugline.InteriorExterior)
	}
}

func TestOpenAICompatibleGeneratorTransformsLooseDeepSeekSchema(t *testing.T) {
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
			responseBody := `{"choices":[{"message":{"content":"metadata:\n  title: \"夜雨疑云\"\n  author: \"示例作者\"\n  language: zh-CN\n  chapter_count: 3\n  style: 悬疑短剧\n  audience: 大众向\ncharacters:\n  - id: lin_qi\n    name: 林琪\n    description: 女主角，单身女性\nscenes:\n  - index: 1\n    chapter: 1\n    location: 公寓走廊\n    time: 深夜\n    beats:\n      - type: action\n        text: 林琪深夜回到公寓，站在走廊里，发现门锁似乎被人动过。\n      - type: dialogue\n        character_id: lin_qi\n        text: 门锁好像被撬过……\n  - index: 2\n    chapter: 2\n    location: 公寓房间内\n    time: 深夜\n    beats:\n      - type: action\n        text: 林琪在房间里找到一张陌生字条，上面写着今晚别睡。\n      - type: dialogue\n        character_id: lin_qi\n        text: 今晚别睡？有人进来过……\n  - index: 3\n    chapter: 3\n    location: 车站\n    time: 清晨\n    beats:\n      - type: action\n        text: 第二天清晨，林琪带着字条前往车站。\n      - type: dialogue\n        character_id: lin_qi\n        text: 我一定要查出是谁写的这封信。\n"}}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(responseBody)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	result, err := generator.Generate(context.Background(), GenerateRequest{
		JobID:    "job_openai_loose",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err != nil {
		t.Fatalf("unexpected generator error: %v", err)
	}
	if result.Document.Version != "1.0" {
		t.Fatalf("expected canonical version 1.0, got %s", result.Document.Version)
	}
	if result.Document.Adaptation.Style != "悬疑短剧" {
		t.Fatalf("expected adaptation style from loose schema, got %s", result.Document.Adaptation.Style)
	}
	if len(result.Document.Scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(result.Document.Scenes))
	}
	if result.Document.Scenes[2].Slugline.InteriorExterior != "EXT" {
		t.Fatalf("expected third scene ext, got %s", result.Document.Scenes[2].Slugline.InteriorExterior)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected normalization warning")
	}
}

func TestOpenAICompatibleGeneratorBackfillsMissingLooseSceneFields(t *testing.T) {
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
			responseBody := `{"choices":[{"message":{"content":"metadata:\n  title: \"夜雨疑云\"\nscenes:\n  - index: 1\n    chapter: 1\n    beats:\n      - type: action\n        text: 林琪深夜回到公寓，发现门锁似乎被人动过。\n  - index: 2\n    chapter: 2\n    beats:\n      - type: action\n        text: 她在房间里找到一张陌生字条，怀疑有人潜入。\n  - index: 3\n    chapter: 3\n    beats:\n      - type: action\n        text: 第二天清晨，她决定顺着线索前往车站。\n"}}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(responseBody)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	result, err := generator.Generate(context.Background(), GenerateRequest{
		JobID:    "job_openai_loose_backfill",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err != nil {
		t.Fatalf("unexpected generator error: %v", err)
	}
	if result.Document.Scenes[0].Slugline.LocationID != "loc_chapter_01" {
		t.Fatalf("expected fallback planned location id loc_chapter_01, got %s", result.Document.Scenes[0].Slugline.LocationID)
	}
	if result.Document.Scenes[0].Slugline.Time != "NIGHT" {
		t.Fatalf("expected fallback planned time NIGHT, got %s", result.Document.Scenes[0].Slugline.Time)
	}
	if result.Document.Scenes[2].Slugline.InteriorExterior != "EXT" {
		t.Fatalf("expected third scene ext from planned slugline, got %s", result.Document.Scenes[2].Slugline.InteriorExterior)
	}
	if len(result.Document.Scenes[1].Beats) < 2 || result.Document.Scenes[1].Beats[1].Type != "dialogue" {
		t.Fatalf("expected planned dialogue backfill, got %#v", result.Document.Scenes[1].Beats)
	}
	if result.Document.Locations[0].Name != "公寓" {
		t.Fatalf("expected planned location name 公寓, got %s", result.Document.Locations[0].Name)
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
