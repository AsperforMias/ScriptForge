package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/AsperforMias/ScriptForge/backend/internal/ingest"
	"github.com/AsperforMias/ScriptForge/backend/internal/job"
	"github.com/AsperforMias/ScriptForge/backend/internal/screenplay"
	"github.com/AsperforMias/ScriptForge/backend/internal/testutil"
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

			return fixtureResponse(t, "canonical_night_rain.yaml"), nil
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

func TestOpenAICompatibleBuildRequestIncludesEvidenceAndReviewGuidance(t *testing.T) {
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

	body, err := generator.buildRequest(GenerateRequest{
		JobID:    "job_openai_guidance",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err != nil {
		t.Fatalf("unexpected build request error: %v", err)
	}

	payload := string(body)
	if !strings.Contains(payload, "evidence") {
		t.Fatalf("expected evidence guidance in request payload, got %s", payload)
	}
	if !strings.Contains(payload, "review") {
		t.Fatalf("expected review guidance in request payload, got %s", payload)
	}
	if !strings.Contains(payload, "Prefer omission over fabrication") {
		t.Fatalf("expected anti-fabrication guidance in request payload, got %s", payload)
	}
	if !strings.Contains(payload, "quote every non-empty string scalar value with double quotes") {
		t.Fatalf("expected yaml quoting guidance in request payload, got %s", payload)
	}
	if !strings.Contains(payload, "Keep evidence.excerpt to one short sentence or phrase") {
		t.Fatalf("expected short evidence guidance in request payload, got %s", payload)
	}
}

func TestOpenAICompatibleBuildRequestSanitizesFogHarborEchoContext(t *testing.T) {
	input := testutil.FogHarborEchoRequest()
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

	body, err := generator.buildRequest(GenerateRequest{
		JobID:    "job_openai_fog_harbor_context",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err != nil {
		t.Fatalf("unexpected build request error: %v", err)
	}

	var requestPayload struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &requestPayload); err != nil {
		t.Fatalf("unmarshal request payload: %v", err)
	}
	if len(requestPayload.Messages) < 2 {
		t.Fatalf("expected user message in request payload, got %#v", requestPayload.Messages)
	}
	userContent := requestPayload.Messages[1].Content
	contextStart := strings.Index(userContent, "{")
	contextEnd := strings.LastIndex(userContent, "}\nFollow this skeleton exactly:")
	if contextStart < 0 || contextEnd < 0 {
		t.Fatalf("expected embedded context json in request payload, got %s", userContent)
	}
	contextJSON := userContent[contextStart : contextEnd+1]
	var contextPayload map[string]any
	if err := json.Unmarshal([]byte(contextJSON), &contextPayload); err != nil {
		t.Fatalf("unmarshal embedded context json: %v", err)
	}

	if _, exists := contextPayload["character_candidates"]; exists {
		t.Fatalf("expected top-level character_candidates to be omitted to reduce prompt pollution, got %#v", contextPayload["character_candidates"])
	}
	if _, exists := contextPayload["location_candidates"]; exists {
		t.Fatalf("expected top-level location_candidates to be omitted to reduce prompt pollution, got %#v", contextPayload["location_candidates"])
	}
	if got, ok := contextPayload["scene_count_target"].(float64); !ok || int(got) != 3 {
		t.Fatalf("expected scene_count_target=3 in request payload, got %#v", contextPayload["scene_count_target"])
	}

	chaptersAny, ok := contextPayload["chapters"].([]any)
	if !ok || len(chaptersAny) < 3 {
		t.Fatalf("expected chapters array in request payload, got %#v", contextPayload["chapters"])
	}
	firstChapter, ok := chaptersAny[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first chapter context object, got %#v", chaptersAny[0])
	}
	if _, exists := firstChapter["summary"]; exists {
		t.Fatalf("expected chapter summary to be omitted from provider context, got %#v", firstChapter["summary"])
	}
	if got, ok := firstChapter["suggested_scene_count"].(float64); !ok || int(got) != 1 {
		t.Fatalf("expected first chapter suggested_scene_count=1, got %#v", firstChapter["suggested_scene_count"])
	}

	charactersAny, ok := firstChapter["character_candidates"].([]any)
	if !ok {
		t.Fatalf("expected chapter-level character_candidates array, got %#v", firstChapter["character_candidates"])
	}
	for _, forbidden := range testutil.FogHarborEchoForbiddenCharacterNames() {
		for _, value := range charactersAny {
			if value == forbidden {
				t.Fatalf("expected sanitized character candidates to exclude %s, got %#v", forbidden, charactersAny)
			}
		}
	}

	locationsAny := make([]any, 0)
	for _, chapterAny := range chaptersAny {
		chapterContext, ok := chapterAny.(map[string]any)
		if !ok {
			t.Fatalf("expected chapter context object, got %#v", chapterAny)
		}
		if chapterLocations, ok := chapterContext["location_candidates"].([]any); ok {
			locationsAny = append(locationsAny, chapterLocations...)
		}
	}
	if len(locationsAny) == 0 {
		t.Fatalf("expected aggregated chapter-level location_candidates array, got %#v", chaptersAny)
	}
	for _, value := range locationsAny {
		if value == "房间" {
			t.Fatalf("expected sanitized location candidates to exclude generic room fallback, got %#v", locationsAny)
		}
	}

	for _, expected := range testutil.FogHarborEchoExpectedLocationFragments() {
		found := false
		for _, value := range locationsAny {
			if text, ok := value.(string); ok && strings.Contains(text, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected request payload to keep source-grounded location fragment %s, got %#v", expected, locationsAny)
		}
	}
	for _, forbiddenFragment := range []string{"回到", "整理", "刚走到", "送来", "见过", "停在"} {
		for _, value := range locationsAny {
			if text, ok := value.(string); ok && strings.Contains(text, forbiddenFragment) {
				t.Fatalf("expected sanitized location candidates to exclude narrative fragment %s, got %#v", forbiddenFragment, locationsAny)
			}
		}
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
			return fixtureResponse(t, "fenced_version1_night_rain.txt"), nil
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

func TestOpenAICompatibleGeneratorExtractsFencedYAMLAfterProviderPreface(t *testing.T) {
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
			return fixtureResponse(t, "preface_fenced_night_rain.txt"), nil
		}),
	}

	result, err := generator.Generate(context.Background(), GenerateRequest{
		JobID:    "job_openai_preface_fenced",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err != nil {
		t.Fatalf("unexpected generator error: %v", err)
	}
	if result.Debug == nil || result.Debug.ParseMode != "canonical" {
		t.Fatalf("expected canonical parse mode for fenced yaml after preface, got %#v", result.Debug)
	}
	if result.Document.Source.Title != "Night Rain" {
		t.Fatalf("unexpected source title after fenced extraction: %s", result.Document.Source.Title)
	}
}

func TestOpenAICompatibleGeneratorSupportsArrayMessageContent(t *testing.T) {
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
			body := `{"choices":[{"message":{"content":[{"type":"text","text":"Here is the screenplay draft you requested."},{"type":"text","text":` +
				strconv.Quote("```yaml\n"+providerFixture(t, "canonical_night_rain.yaml")+"\n```") +
				`}]}}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	result, err := generator.Generate(context.Background(), GenerateRequest{
		JobID:    "job_openai_content_array",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err != nil {
		t.Fatalf("unexpected generator error: %v", err)
	}
	if result.Debug == nil || result.Debug.ParseMode != "canonical" {
		t.Fatalf("expected canonical parse mode, got %#v", result.Debug)
	}
	if result.Document.Source.Title != "Night Rain" {
		t.Fatalf("unexpected source title after content array parsing: %s", result.Document.Source.Title)
	}
}

func TestOpenAICompatibleGeneratorRegeneratesMalformedYAMLBeforeFallback(t *testing.T) {
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

	attempts := 0
	generator.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			attempts++
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			payload := string(body)
			if attempts == 1 {
				if strings.Contains(payload, "Previous attempt 1 failed to parse") {
					t.Fatalf("did not expect retry wording in first request, got %s", payload)
				}
				content := `version: "1.0"
source:
  title: "Night Rain"
  author: "Demo Author"
  language: "zh-CN"
  chapter_count: 3
  chapters:
    - index: 1
      title: "Chapter 1"
      summary: "林琪发现门锁异常。"
adaptation:
  style: "Suspense Drama"
  audience: "General"
  notes: []
characters:
  - id: "char_lin_qi"
    name: "林琪"
    role: "protagonist"
    description: "主角。"
locations:
  - id: "loc_apartment"
    name: "公寓"
    description: "主场景。"
scenes:
  - id: "scene_001"
    title: "深夜回家"
    source_chapters: [1]
    slugline:
      interior_exterior: "INT"
      location_id: "loc_apartment"
      time: "NIGHT"
    summary: "林琪回到公寓。"
    beats:
      - type: "action"
        content: "林琪走到门前。"
    notes:
      adaptation_reason: "保留回家异样。"
      open_questions:
        - "是谁动过门锁？"
    evidence:
      chapter_indexes: [1]
      excerpt: "门锁似乎被动过。"
      cues:
        - "门锁"
    review:
      confidence: "high
      issues: []
validation:
  status: "passed"
  warnings: []`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":` + strconv.Quote(content) + `}}]}`)),
					Header:     make(http.Header),
				}, nil
			}
			if !strings.Contains(payload, "Previous attempt 1 failed to parse") {
				t.Fatalf("expected retry wording in second request, got %s", payload)
			}
			return fixtureResponse(t, "canonical_night_rain.yaml"), nil
		}),
	}

	result, err := generator.Generate(context.Background(), GenerateRequest{
		JobID:    "job_openai_retry_parse",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err != nil {
		t.Fatalf("unexpected generator error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 provider attempts, got %d", attempts)
	}
	if result.Debug == nil {
		t.Fatal("expected debug snapshot")
	}
	if len(result.Debug.RetryRawContents) != 1 {
		t.Fatalf("expected one retry raw content entry, got %#v", result.Debug)
	}
	if len(result.Debug.ParseErrors) == 0 {
		t.Fatalf("expected initial parse error to be recorded, got %#v", result.Debug)
	}
	if !warningsContainSubstring(result.Warnings, "format retry pass") {
		t.Fatalf("expected retry warning, got %#v", result.Warnings)
	}
}

func TestOpenAICompatibleGeneratorRejectsArrayContentWithoutTextParts(t *testing.T) {
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
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":[{"type":"image_url","image_url":{"url":"https://example.com/demo.png"}}]}}]}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	_, err := generator.Generate(context.Background(), GenerateRequest{
		JobID:    "job_openai_content_array_without_text",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err == nil {
		t.Fatal("expected array-without-text failure")
	}
	if !strings.Contains(err.Error(), "no text parts found in message content array") {
		t.Fatalf("expected array content extraction error, got %v", err)
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
			return fixtureResponse(t, "loose_deepseek_schema.yaml"), nil
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
	validated, err := screenplay.ValidateAndSerialize(result.Document)
	if err != nil {
		t.Fatalf("unexpected validation error after loose normalization: %v", err)
	}
	if validated.Document.Validation.Status != "failed" {
		t.Fatalf("expected loose-normalized output to remain validation-failed for honesty, got %s", validated.Document.Validation.Status)
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
			return fixtureResponse(t, "loose_missing_scene_fields.yaml"), nil
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
	if result.Document.Scenes[0].Slugline.LocationID != "loc_公寓" {
		t.Fatalf("expected repaired planned location id loc_公寓, got %s", result.Document.Scenes[0].Slugline.LocationID)
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

func TestOpenAICompatibleGeneratorFallsBackToPlannedCharactersAndMetadata(t *testing.T) {
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
			return fixtureResponse(t, "loose_missing_metadata_and_characters.yaml"), nil
		}),
	}

	result, err := generator.Generate(context.Background(), GenerateRequest{
		JobID:    "job_openai_loose_entity_fallback",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err != nil {
		t.Fatalf("unexpected generator error: %v", err)
	}
	if result.Document.Source.Author != input.Source.Author {
		t.Fatalf("expected source author fallback %s, got %s", input.Source.Author, result.Document.Source.Author)
	}
	if result.Document.Adaptation.Style != input.Adaptation.Style {
		t.Fatalf("expected adaptation style fallback %s, got %s", input.Adaptation.Style, result.Document.Adaptation.Style)
	}
	if len(result.Document.Characters) != len(entities.Characters) {
		t.Fatalf("expected fallback characters from entities, got %d want %d", len(result.Document.Characters), len(entities.Characters))
	}
	if len(result.Document.Characters) == 0 || result.Document.Characters[0].Name != entities.Characters[0].Name {
		t.Fatalf("expected first fallback character %s, got %#v", entities.Characters[0].Name, result.Document.Characters)
	}
	if result.Debug == nil || result.Debug.ParseMode != "loose_normalized" {
		t.Fatalf("expected loose_normalized parse mode, got %#v", result.Debug)
	}
}

func TestOpenAICompatibleGeneratorRepairsFogHarborEchoCanonicalOutput(t *testing.T) {
	input := testutil.FogHarborEchoRequest()
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
			return fixtureResponse(t, "canonical_fog_harbor_bad.yaml"), nil
		}),
	}

	result, err := generator.Generate(context.Background(), GenerateRequest{
		JobID:    "job_openai_fog_harbor_repair",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err != nil {
		t.Fatalf("unexpected generator error: %v", err)
	}

	for _, forbidden := range testutil.FogHarborEchoForbiddenCharacterNames() {
		for _, character := range result.Document.Characters {
			if character.Name == forbidden {
				t.Fatalf("expected repaired characters to exclude %s, got %#v", forbidden, result.Document.Characters)
			}
		}
	}
	for _, forbiddenFragment := range testutil.FogHarborEchoForbiddenObjectiveFragments() {
		for _, scene := range result.Document.Scenes {
			if strings.Contains(scene.Objective, forbiddenFragment) {
				t.Fatalf("expected repaired objectives to exclude %s, got %#v", forbiddenFragment, result.Document.Scenes)
			}
			for _, question := range scene.Notes.OpenQuestions {
				if strings.Contains(question, forbiddenFragment) {
					t.Fatalf("expected repaired open questions to exclude %s, got %#v", forbiddenFragment, result.Document.Scenes)
				}
			}
		}
	}
	locationNames := []string{}
	for _, location := range result.Document.Locations {
		locationNames = append(locationNames, location.Name)
	}
	if len(locationNames) == 0 {
		t.Fatal("expected repaired locations")
	}
	for _, expected := range testutil.FogHarborEchoExpectedLocationFragments() {
		if !containsFragment(locationNames, expected) {
			t.Fatalf("expected repaired locations to include fragment %s, got %#v", expected, locationNames)
		}
	}
	lowConfidenceScenes := 0
	for _, scene := range result.Document.Scenes {
		if scene.Review != nil && scene.Review.Confidence == "low" {
			lowConfidenceScenes++
		}
	}
	if lowConfidenceScenes < 2 {
		t.Fatalf("expected repaired fog harbor output to keep low-confidence review markers, got %#v", result.Document.Scenes)
	}
}

func TestOpenAICompatibleGeneratorKeepsCanonicalOutputWithExtraRolesAndBeatAliases(t *testing.T) {
	input := testutil.FogHarborEchoRequest()
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
			content := `version: "1.0"
source:
  title: "雾港回声"
  author: "自定义输入作者"
  language: zh-CN
  chapter_count: 3
  chapters:
    - index: 1
      title: "第一章 归港的人"
      summary: "林渡在十字路口报刊亭前回城，苏雯递伞并示警。"
    - index: 2
      title: "第二章 旧楼里的纸条"
      summary: "林渡在市立第三医院得知父亲和地下室的异常线索。"
    - index: 3
      title: "第三章 地下室的第二把钥匙"
      summary: "林渡回到老房子，发现镜室地图和神秘持钥匙者。"
adaptation:
  style: "悬疑网剧"
  audience: "大众向"
  notes: []
characters:
  - id: char_lin_du
    name: "林渡"
    role: protagonist
    description: "主角。"
  - id: char_worker
    name: "工装男"
    role: minor
    description: "楼梯口出现的男人。"
locations:
  - id: loc_street_cross
    name: "十字路口报刊亭"
    description: "回城街口。"
  - id: loc_hospital
    name: "市立第三医院"
    description: "医院走廊与病房。"
  - id: loc_old_house
    name: "老房子"
    description: "林渡返家的旧楼。"
scenes:
  - id: scene_001
    title: "夜雨归人"
    source_chapters: [1]
    slugline:
      interior_exterior: EXT
      location_id: loc_street_cross
      time: NIGHT
    summary: "林渡回城并收到苏雯警告。"
    objective: "建立悬疑基调，引入苏雯的警告。"
    beats:
      - type: action
        content: "林渡站在十字路口报刊亭前。"
      - type: dialogue
        character_id: char_lin_du
        content: "你怎么在这？"
    notes:
      adaptation_reason: "保留回城和示警。"
    evidence:
      chapter_indexes: [1]
      excerpt: "林渡站在报刊亭前，苏雯递伞并警告他。"
      cues: ["报刊亭", "苏雯"]
    review:
      confidence: high
      issues: []
  - id: scene_002
    title: "医院纸条"
    source_chapters: [2]
    slugline:
      interior_exterior: INT
      location_id: loc_hospital
      time: NIGHT
    summary: "林渡在医院拿到父亲留下的纸条。"
    beats:
      - type: action
        content: "许言在医院走廊拦住林渡。"
      - type: dialogue
        character_id: char_lin_du
        content: "我爸怎么样？"
    notes:
      adaptation_reason: "保留纸条线索。"
    evidence:
      chapter_indexes: [2]
      excerpt: "许言转交纸条。"
      cues: ["医院", "纸条"]
    review:
      confidence: high
      issues: []
  - id: scene_003
    title: "老房子与钥匙"
    source_chapters: [3]
    slugline:
      interior_exterior: INT
      location_id: loc_old_house
      time: NIGHT
    summary: "林渡在老房子发现镜室和神秘来客。"
    beats:
      - type: action
        content: "林渡回到老房子，发现门锁被动过。"
      - type: memory
        character_id: char_lin_du
        content: "门锁被动过。"
    notes:
      adaptation_reason: "保留返家调查。"
    evidence:
      chapter_indexes: [3]
      excerpt: "林渡回到老房子，楼下有人持相同钥匙。"
      cues: ["老房子", "钥匙"]
    review:
      confidence: high
      issues: []
validation:
  status: passed
  warnings: []`
			payload := `{"choices":[{"message":{"content":` + strconv.Quote(content) + `}}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(payload)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	result, err := generator.Generate(context.Background(), GenerateRequest{
		JobID:    "job_openai_canonical_extra_roles",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err != nil {
		t.Fatalf("unexpected generator error: %v", err)
	}
	if result.Debug == nil || result.Debug.ParseMode != "canonical" {
		t.Fatalf("expected canonical parse mode, got %#v", result.Debug)
	}
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "normalized from loose openai-compatible yaml") {
			t.Fatalf("expected canonical output to avoid loose normalization warning, got %#v", result.Warnings)
		}
	}
	foundWorker := false
	for _, character := range result.Document.Characters {
		if character.Name == "工装男" {
			foundWorker = true
			if character.Role != "supporting" {
				t.Fatalf("expected minor role to normalize to supporting, got %#v", character)
			}
		}
	}
	if !foundWorker {
		t.Fatalf("expected canonical output to keep source-grounded extra-role character, got %#v", result.Document.Characters)
	}
	locationNames := []string{}
	locationByID := map[string]string{}
	for _, location := range result.Document.Locations {
		locationNames = append(locationNames, location.Name)
		locationByID[location.ID] = location.Name
	}
	if !containsFragment(locationNames, "医院") || !containsFragment(locationNames, "老房子") {
		t.Fatalf("expected canonical locations to survive parsing, got %#v", locationNames)
	}
	if got := locationByID[result.Document.Scenes[0].Slugline.LocationID]; got != "十字路口报刊亭" {
		t.Fatalf("expected canonical scene location to keep the specific street kiosk, got %s with locations %#v", got, result.Document.Locations)
	}
	if got := result.Document.Scenes[0].Slugline.InteriorExterior; got != "EXT" {
		t.Fatalf("expected canonical scene int/ext to preserve EXT, got %s", got)
	}
	if got := result.Document.Scenes[0].Objective; got != "" {
		t.Fatalf("expected template objective to be cleared for honesty, got %q", got)
	}
	if result.Document.Scenes[2].Beats[1].Type != "action" {
		t.Fatalf("expected memory beat alias to normalize to action, got %#v", result.Document.Scenes[2].Beats)
	}
}

func TestOpenAICompatibleGeneratorCompressesFogHarborOverSplitCanonicalOutput(t *testing.T) {
	input := testutil.FogHarborEchoRequest()
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
			content := `version: "1.0"
source:
  title: "雾港回声"
  author: "自定义输入作者"
  language: "zh-CN"
  chapter_count: 3
  chapters:
    - index: 1
      title: "第一章 归港的人"
      summary: "林渡归港，苏雯递伞并警告。"
    - index: 2
      title: "第二章 旧楼里的纸条"
      summary: "林渡在医院拿到纸条。"
    - index: 3
      title: "第三章 地下室的第二把钥匙"
      summary: "林渡在老房子发现地图与持钥匙者。"
adaptation:
  style: "悬疑网剧"
  audience: "大众向"
  notes: []
characters:
  - id: "char_lin_du"
    name: "林渡"
    role: "protagonist"
    description: "主角。"
  - id: "char_su_wen"
    name: "苏雯"
    role: "supporting"
    description: "旧同学。"
  - id: "char_xu_yan"
    name: "许言"
    role: "supporting"
    description: "医生。"
locations:
  - id: "loc_cross"
    name: "十字路口报刊亭"
    description: "街口报刊亭。"
  - id: "loc_hospital_corridor"
    name: "市立第三医院走廊"
    description: "医院走廊。"
  - id: "loc_hospital_room"
    name: "病房"
    description: "医院病房。"
  - id: "loc_old_house_living_room"
    name: "老房子客厅"
    description: "返家后的客厅。"
  - id: "loc_old_house_stairway"
    name: "楼梯口"
    description: "楼下楼梯口。"
scenes:
  - id: "scene_001"
    title: "夜雨归港"
    source_chapters: [1]
    slugline:
      interior_exterior: "EXT"
      location_id: "loc_cross"
      time: "NIGHT"
    summary: "林渡回城，苏雯递伞并警告。"
    objective: "建立悬疑基调，引入苏雯的警告。"
    beats:
      - type: "action"
        content: "林渡站在十字路口报刊亭前。"
      - type: "dialogue"
        character_id: "char_lin_du"
        content: "你怎么在这？"
    notes:
      adaptation_reason: "保留回城和示警。"
      open_questions:
        - "接下来会发生什么？"
    evidence:
      chapter_indexes: [1]
      excerpt: "林渡站在报刊亭前，苏雯递伞并警告他。"
      cues: ["报刊亭", "苏雯"]
    review:
      confidence: "high"
      issues: []
  - id: "scene_002"
    title: "医院走廊"
    source_chapters: [2]
    slugline:
      interior_exterior: "INT"
      location_id: "loc_hospital_corridor"
      time: "NIGHT"
    summary: "许言在走廊拦住林渡。"
    objective: "林渡要确认父亲为什么突然住院。"
    beats:
      - type: "action"
        content: "林渡刚到病房门口，就被许言拦住。"
      - type: "dialogue"
        character_id: "char_lin_du"
        content: "我爸怎么样？"
    notes:
      adaptation_reason: "单独强调走廊对话。"
    evidence:
      chapter_indexes: [2]
      excerpt: "林渡刚走到病房门口，就被拦住。"
      cues: ["病房门口", "许言"]
    review:
      confidence: "medium"
      issues: []
  - id: "scene_003"
    title: "病房纸条"
    source_chapters: [2]
    slugline:
      interior_exterior: "INT"
      location_id: "loc_hospital_room"
      time: "NIGHT"
    summary: "许言转交纸条，林建国手指微动。"
    objective: "弄清楚父亲昏迷的原因。"
    beats:
      - type: "action"
        content: "许言把纸条递给林渡。"
      - type: "action"
        content: "病床上的林建国手指蜷了一下。"
    notes:
      adaptation_reason: "单独强调病房线索。"
      open_questions:
        - "纸条为什么会写给林渡？"
    evidence:
      chapter_indexes: [2]
      excerpt: "纸条上写着别让他们找到地下室的第二把钥匙。"
      cues: ["纸条", "第二把钥匙"]
    review:
      confidence: "medium"
      issues: []
  - id: "scene_004"
    title: "返家镜室"
    source_chapters: [3]
    slugline:
      interior_exterior: "INT"
      location_id: "loc_old_house_living_room"
      time: "NIGHT"
    summary: "林渡回到老房子，在客厅发现镜室地图。"
    objective: "林渡要确认镜室和第二把钥匙之间的联系。"
    beats:
      - type: "action"
        content: "林渡发现桌上的牛皮纸袋和地图。"
      - type: "action"
        content: "地图上用红笔圈出镜室。"
    notes:
      adaptation_reason: "保留返家调查。"
    evidence:
      chapter_indexes: [3]
      excerpt: "地图在地下一层的位置圈出镜室。"
      cues: ["镜室", "地图"]
    review:
      confidence: "medium"
      issues: []
  - id: "scene_005"
    title: "楼下持钥匙者"
    source_chapters: [3]
    slugline:
      interior_exterior: "INT"
      location_id: "loc_old_house_stairway"
      time: "NIGHT"
    summary: "楼梯口出现两个男人，其中一人拿着相同铜钥匙。"
    objective: "林渡想知道楼下的人为什么也有同样的钥匙。"
    beats:
      - type: "action"
        content: "两个男人停在楼梯口，像在确认门牌。"
      - type: "action"
        content: "鸭舌帽男掏出旧铜钥匙，在指尖转了一圈。"
    notes:
      adaptation_reason: "保留门外威胁。"
      open_questions:
        - "楼下的人是谁派来的？"
    evidence:
      chapter_indexes: [3]
      excerpt: "鸭舌帽男拿着与林渡手中相似的铜钥匙。"
      cues: ["楼梯口", "铜钥匙"]
    review:
      confidence: "medium"
      issues: []
validation:
  status: "passed"
  warnings: []`
			payload := `{"choices":[{"message":{"content":` + strconv.Quote(content) + `}}]}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(payload)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	result, err := generator.Generate(context.Background(), GenerateRequest{
		JobID:    "job_openai_fog_harbor_over_split",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err != nil {
		t.Fatalf("unexpected generator error: %v", err)
	}
	if got := len(result.Document.Scenes); got != 3 {
		t.Fatalf("expected over-split fog harbor canonical output to compress to 3 scenes, got %d", got)
	}
	for idx, scene := range result.Document.Scenes {
		if len(scene.SourceChapters) != 1 || scene.SourceChapters[0] != idx+1 {
			t.Fatalf("expected compressed scene %d to stay aligned to chapter %d, got %#v", idx+1, idx+1, scene.SourceChapters)
		}
	}
	if got := result.Document.Scenes[0].Objective; got != "" {
		t.Fatalf("expected scene 1 template objective to be cleared after compression, got %q", got)
	}
	if result.Document.Scenes[1].Review == nil || result.Document.Scenes[1].Review.Confidence != "medium" {
		t.Fatalf("expected merged hospital scene to keep medium confidence review, got %#v", result.Document.Scenes[1].Review)
	}
	if !containsIssue(result.Document.Scenes[1].Review, "merged to keep a stable editable draft") {
		t.Fatalf("expected merged hospital scene review to disclose scene-boundary merge, got %#v", result.Document.Scenes[1].Review)
	}
	if len(result.Document.Scenes[2].Beats) < 3 {
		t.Fatalf("expected merged old-house scene to retain multiple beats, got %#v", result.Document.Scenes[2].Beats)
	}
}

func TestOpenAICompatibleGeneratorRejectsLooseSchemaWithoutScenes(t *testing.T) {
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
			return fixtureResponse(t, "invalid_no_scenes.yaml"), nil
		}),
	}

	_, err := generator.Generate(context.Background(), GenerateRequest{
		JobID:    "job_openai_invalid_loose",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "no scenes found in loose yaml") {
		t.Fatalf("expected no scenes error, got %v", err)
	}
}

func containsIssue(review *screenplay.Review, fragment string) bool {
	if review == nil {
		return false
	}
	for _, issue := range review.Issues {
		if strings.Contains(issue, fragment) {
			return true
		}
	}
	return false
}

func TestOpenAICompatibleGeneratorSurfacesProviderHTTPFailure(t *testing.T) {
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
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"rate limit exceeded"}}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	_, err := generator.Generate(context.Background(), GenerateRequest{
		JobID:    "job_openai_http_failure",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err == nil {
		t.Fatal("expected provider http failure")
	}
	if !strings.Contains(err.Error(), "status=429") {
		t.Fatalf("expected status code in error, got %v", err)
	}
}

func TestOpenAICompatibleGeneratorSurfacesProviderErrorField(t *testing.T) {
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
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"provider internal failure"}}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	_, err := generator.Generate(context.Background(), GenerateRequest{
		JobID:    "job_openai_error_field",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err == nil {
		t.Fatal("expected provider error field failure")
	}
	if !strings.Contains(err.Error(), "provider internal failure") {
		t.Fatalf("expected provider message in error, got %v", err)
	}
}

func TestSanitizeGeneratedCharactersPrefersCanonicalLLMCharactersOverWeakPreferredFragments(t *testing.T) {
	req := validOpenAICompatibleCreateJobRequest()
	req.Source.Title = "交稿前夜"
	req.Source.Author = "示例作者"
	req.Source.Chapters[0].Title = "第一章 数据被换"
	req.Source.Chapters[0].Content = "晚上十点半，苏栀一个人留在办公室复核提案。她翻出本地备份后确认，问题只出现在共享盘上的最终文件。"
	req.Source.Chapters[1].Title = "第二章 咖啡馆对质"
	req.Source.Chapters[1].Content = "第二天下午，苏栀把顾屿约到咖啡馆。顾屿没有正面回答，反而质问她这几天为什么一直单独和客户助理沟通。对话越说越僵。"
	req.Source.Chapters[2].Title = "第三章 会议室摊牌"
	req.Source.Chapters[2].Content = "清晨，苏栀提前到会议室，项目负责人已经在场。"

	source := ingest.Normalize(req)
	generated := []screenplay.Character{
		{ID: "char_suzhi", Name: "苏栀", Role: "protagonist"},
		{ID: "char_guyu", Name: "顾屿", Role: "supporting"},
		{ID: "char_project_lead", Name: "项目负责人", Role: "supporting"},
	}
	preferred := []screenplay.Character{
		{ID: "char_frag_1", Name: "苏栀一个人", Role: "protagonist"},
		{ID: "char_frag_2", Name: "问题只出现", Role: "supporting"},
		{ID: "char_frag_3", Name: "反而", Role: "supporting"},
		{ID: "char_frag_4", Name: "对话越", Role: "supporting"},
	}

	got := sanitizeGeneratedCharacters(generated, preferred, GenerateRequest{
		Input:  req,
		Source: source,
	})

	if len(got) != 3 {
		t.Fatalf("expected only canonical generated characters to remain, got %#v", got)
	}
	for _, forbidden := range []string{"苏栀一个人", "问题只出现", "反而", "对话越"} {
		for _, character := range got {
			if character.Name == forbidden {
				t.Fatalf("expected fragment-like preferred character %s to be removed, got %#v", forbidden, got)
			}
		}
	}
}

func TestCharacterNameValidatorsRejectFragmentLikeChinesePhrases(t *testing.T) {
	sourceText := normalizeWhitespace("晚上十点半，苏栀一个人留在办公室。问题只出现在共享盘。顾屿没有正面回答，反而质问她。对话越说越僵。她检查卧室时，在抽屉底层发现字条，而是没有立刻报警。")

	for _, bad := range []string{"苏栀一个人", "问题只出现", "反而", "对话越", "在抽屉底层", "而是"} {
		if isSupportedGeneratedCharacterName(bad, sourceText) {
			t.Fatalf("expected fragment-like phrase %s to be rejected as a generated character name", bad)
		}
		if isSupportedPreferredCharacterName(bad, sourceText) {
			t.Fatalf("expected fragment-like phrase %s to be rejected as a preferred character name", bad)
		}
	}

	for _, good := range []string{"苏栀", "顾屿"} {
		if !isSupportedGeneratedCharacterName(good, sourceText) {
			t.Fatalf("expected normal character name %s to stay supported for generated characters", good)
		}
		if !isSupportedPreferredCharacterName(good, sourceText) {
			t.Fatalf("expected normal character name %s to stay supported for preferred characters", good)
		}
	}
}

func TestSanitizeGeneratedCharactersKeepsSingleValidLLMCharacterWithoutPreferredBackfill(t *testing.T) {
	req := validOpenAICompatibleCreateJobRequest()
	req.Source.Chapters[0].Title = "第一章 夜雨"
	req.Source.Chapters[0].Content = "林琬回到旧公寓，在抽屉底层发现一张写着地址的纸条。"
	req.Source.Chapters[1].Title = "第二章 旧友"
	req.Source.Chapters[1].Content = "她联系苏雯确认纸条来历，但苏雯只是让她先别报警。"
	req.Source.Chapters[2].Title = "第三章 追查"
	req.Source.Chapters[2].Content = "林琬决定先去纸条上的地址看看。"

	source := ingest.Normalize(req)
	generated := []screenplay.Character{
		{ID: "char_linwan", Name: "林琬", Role: "protagonist"},
	}
	preferred := []screenplay.Character{
		{ID: "char_frag_1", Name: "在抽屉底层", Role: "supporting"},
		{ID: "char_frag_2", Name: "而是", Role: "supporting"},
	}

	got := sanitizeGeneratedCharacters(generated, preferred, GenerateRequest{
		Input:  req,
		Source: source,
	})

	if len(got) != 1 || got[0].Name != "林琬" {
		t.Fatalf("expected single llm character to stay untouched, got %#v", got)
	}
}

func TestIsSupportedPreferredCharacterNameRequiresHumanLikeChineseNames(t *testing.T) {
	sourceText := normalizeWhitespace("林琬在卧室抽屉发现字条。项目负责人准备开始彩排。苏雯站在门口。")

	if !isSupportedPreferredCharacterName("林琬", sourceText) {
		t.Fatal("expected 林琬 to be accepted as a preferred character name")
	}
	if !isSupportedPreferredCharacterName("苏雯", sourceText) {
		t.Fatal("expected 苏雯 to be accepted as a preferred character name")
	}
	if isSupportedPreferredCharacterName("项目负责人", sourceText) {
		t.Fatal("expected role-like title 项目负责人 to stay out of preferred character candidates")
	}
	if !isSupportedGeneratedCharacterName("项目负责人", sourceText) {
		t.Fatal("expected generated role-like title 项目负责人 to remain allowed")
	}
}

func TestSanitizeSceneObjectiveKeepsNonTemplateTextEvenWhenReviewIsLowConfidence(t *testing.T) {
	objective := "迫使项目组承认数据被调包"
	chapterText := normalizeWhitespace("周宁在操场热身，教练提醒她注意接力棒交接。")
	review := &screenplay.Review{Confidence: "low"}

	got := sanitizeSceneObjective(objective, chapterText, review)
	if got != objective {
		t.Fatalf("expected low-confidence objective to be preserved when it is not obvious template text, got %q", got)
	}
}

func TestSanitizeOpenQuestionsKeepsNonTemplateTextEvenWhenReviewIsLowConfidence(t *testing.T) {
	questions := []string{
		"是谁提前改了签到表？",
		"接下来会发生什么？",
	}
	chapterText := normalizeWhitespace("周宁在操场边和教练交谈，准备开始接力训练。")
	review := &screenplay.Review{Confidence: "low"}

	got := sanitizeOpenQuestions(questions, chapterText, review)
	if len(got) != 1 || got[0] != "是谁提前改了签到表？" {
		t.Fatalf("expected non-template open question to stay while template filler is removed, got %#v", got)
	}
}

func TestIsSupportedLocationNameRejectsNarrativeSentenceLikeText(t *testing.T) {
	for _, bad := range []string{
		"林琪意识到有人提前进入过房间",
		"她在房间里找到一张陌生字条",
		"回到旧公寓",
	} {
		if isSupportedLocationName(bad) {
			t.Fatalf("expected narrative-like location candidate %q to be rejected", bad)
		}
	}

	for _, good := range []string{
		"旧公寓",
		"市立第三医院",
		"旧城区十字路口",
		"车站",
	} {
		if !isSupportedLocationName(good) {
			t.Fatalf("expected normal location candidate %q to stay supported", good)
		}
	}
}

func TestChooseSceneLocationRejectsSentenceLikeGeneratedLocation(t *testing.T) {
	scene := &screenplay.Scene{
		Title:   "陌生字条",
		Summary: "林琪在房间内发现字条，意识到有人提前进入过房间。",
	}
	chapter := job.ChapterBody{
		Index:   2,
		Title:   "第二章 陌生字条",
		Content: "她在房间里找到一张陌生字条，上面只写着今晚别睡。林琪意识到有人提前进入过房间。",
	}
	current := screenplay.Location{
		ID:          "loc_bad_sentence",
		Name:        "林琪意识到有人提前进入过房间",
		Description: "bad generated location",
	}

	got := chooseSceneLocation(scene, chapter, current, 1)
	if got.Name != "房间" {
		t.Fatalf("expected sentence-like generated location to fall back to source-grounded 房间, got %#v", got)
	}
}

func TestOpenAICompatibleGeneratorRejectsEmptyChoices(t *testing.T) {
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
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"choices":[]}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	_, err := generator.Generate(context.Background(), GenerateRequest{
		JobID:    "job_openai_empty_choices",
		Input:    input,
		Source:   source,
		Outline:  outline,
		Entities: entities,
		Plan:     plan,
	})
	if err == nil {
		t.Fatal("expected empty choices failure")
	}
	if !strings.Contains(err.Error(), "empty choices") {
		t.Fatalf("expected empty choices error, got %v", err)
	}
}

func fixtureResponse(t *testing.T, name string) *http.Response {
	t.Helper()

	content := providerFixture(t, name)
	payload := `{"choices":[{"message":{"content":` + strconv.Quote(content) + `}}]}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(payload)),
		Header:     make(http.Header),
	}
}

func providerFixture(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "..", "..", "testdata", "provider-fixtures", "openai-compatible", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read provider fixture %s: %v", name, err)
	}
	return string(data)
}

func containsFragment(values []string, fragment string) bool {
	for _, value := range values {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
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

func warningsContainSubstring(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
