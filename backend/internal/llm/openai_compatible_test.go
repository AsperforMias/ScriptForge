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
