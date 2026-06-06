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
	contextJSON := userContent[contextStart:contextEnd+1]
	var contextPayload map[string]any
	if err := json.Unmarshal([]byte(contextJSON), &contextPayload); err != nil {
		t.Fatalf("unmarshal embedded context json: %v", err)
	}

	charactersAny, ok := contextPayload["character_candidates"].([]any)
	if !ok {
		t.Fatalf("expected character_candidates array, got %#v", contextPayload["character_candidates"])
	}
	for _, forbidden := range testutil.FogHarborEchoForbiddenCharacterNames() {
		for _, value := range charactersAny {
			if value == forbidden {
				t.Fatalf("expected sanitized character candidates to exclude %s, got %#v", forbidden, charactersAny)
			}
		}
	}

	locationsAny, ok := contextPayload["location_candidates"].([]any)
	if !ok {
		t.Fatalf("expected location_candidates array, got %#v", contextPayload["location_candidates"])
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

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
