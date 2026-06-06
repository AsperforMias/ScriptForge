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
	"gopkg.in/yaml.v3"
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
			Content json.RawMessage `json:"content"`
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

	messageContent, err := extractMessageContent(completion.Choices[0].Message.Content)
	if err != nil {
		return GenerateResult{}, NewInvocationError(g.Name(), fmt.Errorf("extract message content: %w", err))
	}

	yamlText := extractYAMLResponse(messageContent)
	document, warnings, parseMode, err := parseGeneratedDocument(yamlText, req)
	if err != nil {
		return GenerateResult{}, NewInvocationError(g.Name(), fmt.Errorf("parse yaml response: %w", err))
	}
	document.Validation.Warnings = append(document.Validation.Warnings, warnings...)
	document.Validation.Warnings = append(document.Validation.Warnings, "generated via openai_compatible provider")

	return GenerateResult{
		Document: document,
		Warnings: document.Validation.Warnings,
		Debug: &DebugSnapshot{
			Provider:   g.Name(),
			Model:      g.model,
			ParseMode:  parseMode,
			RawContent: yamlText,
		},
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
					"Return only valid YAML that matches this exact root schema: version, source, adaptation, characters, locations, scenes, validation. " +
					"Do not invent alternative top-level keys such as metadata. " +
					"Every scene must include slugline.interior_exterior, slugline.location_id, slugline.time, summary, objective, beats, and notes.open_questions. " +
					"All scenes must reference a valid location_id declared in locations. " +
					"Use Chinese content values where appropriate, but keep schema keys exactly as requested. " +
					"Do not include markdown fences or explanations.",
			},
			{
				Role: "user",
				Content: "Generate a screenplay YAML document from this structured context:\n" +
					string(contextJSON) +
					"\nFollow this skeleton exactly:\n" +
					"version: \"1.0\"\n" +
					"source:\n  title: ...\n  author: ...\n  language: zh-CN\n  chapter_count: ...\n  chapters:\n    - index: 1\n      title: ...\n      summary: ...\n" +
					"adaptation:\n  style: ...\n  audience: ...\n  notes: []\n" +
					"characters:\n  - id: char_xxx\n    name: ...\n    role: protagonist\n    description: ...\n" +
					"locations:\n  - id: loc_xxx\n    name: ...\n    description: ...\n" +
					"scenes:\n  - id: scene_001\n    title: ...\n    source_chapters: [1]\n    slugline:\n      interior_exterior: INT\n      location_id: loc_xxx\n      time: NIGHT\n    summary: ...\n    objective: ...\n    beats:\n      - type: action\n        content: ...\n      - type: dialogue\n        character_id: char_xxx\n        content: ...\n        emotion: tense\n    notes:\n      adaptation_reason: ...\n      open_questions:\n        - ...\n" +
					"validation:\n  status: passed\n  warnings: []\n" +
					"Ensure source.chapter_count matches the input, every scene references valid chapter indexes, and dialogue beats reference valid character_id values.",
			},
		},
	}

	return json.Marshal(requestBody)
}

type looseDocument struct {
	Metadata struct {
		Title        string `yaml:"title"`
		Author       string `yaml:"author"`
		Language     string `yaml:"language"`
		ChapterCount int    `yaml:"chapter_count"`
		Style        string `yaml:"style"`
		Audience     string `yaml:"audience"`
	} `yaml:"metadata"`
	Characters []struct {
		ID          string `yaml:"id"`
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	} `yaml:"characters"`
	Scenes []struct {
		Index    int    `yaml:"index"`
		Chapter  int    `yaml:"chapter"`
		Location string `yaml:"location"`
		Time     string `yaml:"time"`
		Beats    []struct {
			Type        string `yaml:"type"`
			Text        string `yaml:"text"`
			CharacterID string `yaml:"character_id"`
		} `yaml:"beats"`
	} `yaml:"scenes"`
}

type contentPart struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Content string `json:"content"`
	Value   string `json:"value"`
}

func extractMessageContent(raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return "", fmt.Errorf("missing message content")
	}

	if trimmed[0] == '"' {
		var content string
		if err := json.Unmarshal(trimmed, &content); err != nil {
			return "", err
		}
		return strings.TrimSpace(content), nil
	}

	if trimmed[0] == '[' {
		var parts []json.RawMessage
		if err := json.Unmarshal(trimmed, &parts); err != nil {
			return "", err
		}

		textParts := make([]string, 0, len(parts))
		for _, partRaw := range parts {
			partText := extractContentPartText(partRaw)
			if partText == "" {
				continue
			}
			textParts = append(textParts, partText)
		}

		if len(textParts) == 0 {
			return "", fmt.Errorf("no text parts found in message content array")
		}

		return strings.TrimSpace(strings.Join(textParts, "\n")), nil
	}

	return "", fmt.Errorf("unsupported message content shape")
}

func extractContentPartText(raw json.RawMessage) string {
	var direct string
	if err := json.Unmarshal(raw, &direct); err == nil {
		return strings.TrimSpace(direct)
	}

	var part contentPart
	if err := json.Unmarshal(raw, &part); err == nil {
		if text := firstNonEmpty(part.Text, part.Content, part.Value); text != "" {
			return strings.TrimSpace(text)
		}
	}

	var nested struct {
		Type string `json:"type"`
		Text struct {
			Value string `json:"value"`
		} `json:"text"`
	}
	if err := json.Unmarshal(raw, &nested); err == nil {
		return strings.TrimSpace(nested.Text.Value)
	}

	return ""
}

func extractYAMLResponse(content string) string {
	content = strings.TrimSpace(content)
	if idx := strings.Index(content, "```"); idx >= 0 {
		content = content[idx+3:]
		content = strings.TrimPrefix(content, "yaml")
		content = strings.TrimPrefix(content, "yml")
		content = strings.TrimLeft(content, "\r\n")
		if end := strings.Index(content, "```"); end >= 0 {
			content = content[:end]
		}
		return strings.TrimSpace(content)
	}

	lines := strings.Split(content, "\n")
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "version:"),
			strings.HasPrefix(trimmed, "source:"),
			strings.HasPrefix(trimmed, "adaptation:"),
			strings.HasPrefix(trimmed, "metadata:"),
			strings.HasPrefix(trimmed, "characters:"),
			strings.HasPrefix(trimmed, "locations:"),
			strings.HasPrefix(trimmed, "scenes:"),
			strings.HasPrefix(trimmed, "validation:"):
			return strings.TrimSpace(strings.Join(lines[idx:], "\n"))
		}
	}

	return content
}

func normalizeGeneratedDocument(doc screenplay.Document) screenplay.Document {
	version := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(doc.Version), "v"))
	switch {
	case version == "1", version == "1.0", version == "1.0.0":
		doc.Version = "1.0"
	}

	doc.Validation.Status = strings.TrimSpace(strings.ToLower(doc.Validation.Status))
	if doc.Validation.Status == "" {
		doc.Validation.Status = "passed"
	}
	if doc.Validation.Warnings == nil {
		doc.Validation.Warnings = []string{}
	}

	for idx := range doc.Scenes {
		doc.Scenes[idx].Slugline.InteriorExterior = strings.ToUpper(strings.TrimSpace(doc.Scenes[idx].Slugline.InteriorExterior))
		doc.Scenes[idx].Slugline.Time = strings.ToUpper(strings.TrimSpace(doc.Scenes[idx].Slugline.Time))
	}

	return doc
}

func parseGeneratedDocument(yamlText string, req GenerateRequest) (screenplay.Document, []string, string, error) {
	document, err := screenplay.ParseYAML(yamlText)
	if err == nil {
		document = normalizeGeneratedDocument(document)
		if validateErr := screenplay.Validate(document); validateErr == nil {
			return document, nil, "canonical", nil
		}
	}

	fallbackDocument, err := parseLooseDocument(yamlText, req)
	if err != nil {
		return screenplay.Document{}, nil, "", err
	}

	fallbackDocument = normalizeGeneratedDocument(fallbackDocument)
	if err := screenplay.Validate(fallbackDocument); err != nil {
		return screenplay.Document{}, nil, "", err
	}
	return fallbackDocument, []string{"normalized from loose openai-compatible yaml"}, "loose_normalized", nil
}

func parseLooseDocument(yamlText string, req GenerateRequest) (screenplay.Document, error) {
	var loose looseDocument
	if err := yaml.Unmarshal([]byte(yamlText), &loose); err != nil {
		return screenplay.Document{}, err
	}
	if len(loose.Scenes) == 0 {
		return screenplay.Document{}, fmt.Errorf("no scenes found in loose yaml")
	}

	characters := make([]screenplay.Character, 0, max(len(loose.Characters), len(req.Entities.Characters)))
	if len(loose.Characters) > 0 {
		for idx, character := range loose.Characters {
			characterID := strings.TrimSpace(character.ID)
			if characterID == "" {
				characterID = "char_" + looseSlug(character.Name)
			}
			role := "supporting"
			if idx == 0 {
				role = "protagonist"
			}
			characters = append(characters, screenplay.Character{
				ID:          characterID,
				Name:        strings.TrimSpace(character.Name),
				Role:        role,
				Description: strings.TrimSpace(character.Description),
			})
		}
	} else {
		characters = append(characters, req.Entities.Characters...)
	}

	locationIDs := map[string]string{}
	locations := make([]screenplay.Location, 0, len(loose.Scenes))
	scenes := make([]screenplay.Scene, 0, len(loose.Scenes))

	for idx, looseScene := range loose.Scenes {
		plannedScene, hasPlannedScene := lookupPlannedScene(req, looseScene.Chapter, idx)
		plannedLocation, hasPlannedLocation := lookupPlannedLocation(req, looseScene.Chapter, idx)
		locationName := strings.TrimSpace(looseScene.Location)
		if locationName == "" && hasPlannedLocation {
			locationName = plannedLocation.Name
		}
		if locationName == "" {
			locationName = fmt.Sprintf("Scene %d Location", idx+1)
		}

		locationID, ok := locationIDs[locationName]
		if !ok {
			location := screenplay.Location{
				ID:          "loc_" + looseSlug(locationName),
				Name:        locationName,
				Description: "Location normalized from openai-compatible provider output.",
			}
			if hasPlannedLocation && (strings.TrimSpace(looseScene.Location) == "" || locationName == plannedLocation.Name) {
				location = plannedLocation
			}
			locationID = location.ID
			locationIDs[locationName] = locationID
			locations = append(locations, location)
		}

		chapterIndex := looseScene.Chapter
		if chapterIndex == 0 && idx < len(req.Input.Source.Chapters) {
			chapterIndex = req.Input.Source.Chapters[idx].Index
		}

		title := lookupChapterTitle(req, chapterIndex, idx)
		summary := lookupChapterSummary(req, chapterIndex, firstActionBeat(looseScene.Beats))
		beats := make([]screenplay.Beat, 0, len(looseScene.Beats))
		for _, beat := range looseScene.Beats {
			content := strings.TrimSpace(beat.Text)
			if content == "" {
				continue
			}
			beatType := normalizeLooseBeatType(beat.Type)
			characterID := normalizeLooseCharacterID(strings.TrimSpace(beat.CharacterID), beatType, characters)
			beats = append(beats, screenplay.Beat{
				Type:        beatType,
				CharacterID: characterID,
				Content:     content,
				Emotion:     inferLooseEmotion(content),
			})
		}
		if len(beats) == 0 {
			beats = append(beats, screenplay.Beat{Type: "action", Content: summary})
		}
		if hasPlannedScene {
			beats = ensureDialogueBeat(beats, plannedScene)
		}

		interiorExterior := inferLooseInteriorExterior(locationName)
		if strings.TrimSpace(looseScene.Location) == "" && hasPlannedScene {
			interiorExterior = plannedScene.Slugline.InteriorExterior
		}
		timeValue := normalizeLooseTime(looseScene.Time)
		if strings.TrimSpace(looseScene.Time) == "" && hasPlannedScene {
			timeValue = plannedScene.Slugline.Time
		}

		sceneID := fmt.Sprintf("scene_%03d", idx+1)
		scene := screenplay.Scene{
			ID:             sceneID,
			Title:          title,
			SourceChapters: []int{chapterIndex},
			Slugline: screenplay.Slugline{
				InteriorExterior: interiorExterior,
				LocationID:       locationID,
				Time:             timeValue,
			},
			Summary:   summary,
			Objective: lookupPlannedObjective(req, chapterIndex, idx),
			Beats:     beats,
			Notes: screenplay.SceneNotes{
				AdaptationReason: "Normalized from a looser openai-compatible screenplay response into the canonical project schema.",
				OpenQuestions:    lookupPlannedOpenQuestions(req, chapterIndex, idx),
			},
		}
		scenes = append(scenes, scene)
	}

	sourceChapters := make([]screenplay.SourceChapter, 0, len(req.Input.Source.Chapters))
	for idx, chapter := range req.Input.Source.Chapters {
		summary := ""
		if idx < len(req.Outline.Chapters) {
			summary = req.Outline.Chapters[idx].Summary
		}
		sourceChapters = append(sourceChapters, screenplay.SourceChapter{
			Index:   chapter.Index,
			Title:   chapter.Title,
			Summary: summary,
		})
	}

	document := screenplay.Document{
		Version: "1.0",
		Source: screenplay.Source{
			Title:        firstNonEmpty(loose.Metadata.Title, req.Source.Title, req.Input.Source.Title),
			Author:       firstNonEmpty(loose.Metadata.Author, req.Source.Author, req.Input.Source.Author),
			Language:     firstNonEmpty(loose.Metadata.Language, req.Source.Language, "zh-CN"),
			ChapterCount: len(req.Input.Source.Chapters),
			Chapters:     sourceChapters,
		},
		Adaptation: screenplay.Adaptation{
			Style:    firstNonEmpty(loose.Metadata.Style, req.Input.Adaptation.Style),
			Audience: firstNonEmpty(loose.Metadata.Audience, req.Input.Adaptation.Audience),
			Notes:    req.Input.Adaptation.Notes,
		},
		Characters: characters,
		Locations:  locations,
		Scenes:     scenes,
		Validation: screenplay.Validation{
			Status:   "passed",
			Warnings: []string{},
		},
	}

	return document, nil
}

func firstActionBeat(beats []struct {
	Type        string `yaml:"type"`
	Text        string `yaml:"text"`
	CharacterID string `yaml:"character_id"`
}) string {
	for _, beat := range beats {
		if strings.TrimSpace(beat.Text) != "" {
			return strings.TrimSpace(beat.Text)
		}
	}
	return ""
}

func lookupChapterTitle(req GenerateRequest, chapterIndex, fallbackIndex int) string {
	for _, chapter := range req.Input.Source.Chapters {
		if chapter.Index == chapterIndex {
			return chapter.Title
		}
	}
	if fallbackIndex < len(req.Input.Source.Chapters) {
		return req.Input.Source.Chapters[fallbackIndex].Title
	}
	return fmt.Sprintf("Scene %03d", fallbackIndex+1)
}

func lookupChapterSummary(req GenerateRequest, chapterIndex int, fallback string) string {
	for _, chapter := range req.Outline.Chapters {
		if chapter.Index == chapterIndex {
			return chapter.Summary
		}
	}
	if strings.TrimSpace(fallback) != "" {
		return strings.TrimSpace(fallback)
	}
	return "Scene summary generated from provider output."
}

func lookupPlannedObjective(req GenerateRequest, chapterIndex, fallbackIndex int) string {
	if fallbackIndex < len(req.Plan.Scenes) {
		return req.Plan.Scenes[fallbackIndex].Objective
	}
	for _, scene := range req.Plan.Scenes {
		if len(scene.SourceChapters) > 0 && scene.SourceChapters[0] == chapterIndex {
			return scene.Objective
		}
	}
	return "Drive the chapter conflict into a filmable dramatic action."
}

func lookupPlannedOpenQuestions(req GenerateRequest, chapterIndex, fallbackIndex int) []string {
	if fallbackIndex < len(req.Plan.Scenes) {
		return req.Plan.Scenes[fallbackIndex].Notes.OpenQuestions
	}
	for _, scene := range req.Plan.Scenes {
		if len(scene.SourceChapters) > 0 && scene.SourceChapters[0] == chapterIndex {
			return scene.Notes.OpenQuestions
		}
	}
	return []string{}
}

func lookupPlannedScene(req GenerateRequest, chapterIndex, fallbackIndex int) (screenplay.Scene, bool) {
	if fallbackIndex < len(req.Plan.Scenes) {
		return req.Plan.Scenes[fallbackIndex], true
	}
	for _, scene := range req.Plan.Scenes {
		if len(scene.SourceChapters) > 0 && scene.SourceChapters[0] == chapterIndex {
			return scene, true
		}
	}
	return screenplay.Scene{}, false
}

func lookupPlannedLocation(req GenerateRequest, chapterIndex, fallbackIndex int) (screenplay.Location, bool) {
	scene, ok := lookupPlannedScene(req, chapterIndex, fallbackIndex)
	if !ok {
		return screenplay.Location{}, false
	}
	for _, location := range req.Plan.Locations {
		if location.ID == scene.Slugline.LocationID {
			return location, true
		}
	}
	for _, location := range req.Entities.Locations {
		if location.ID == scene.Slugline.LocationID {
			return location, true
		}
	}
	if fallbackIndex < len(req.Entities.Locations) {
		return req.Entities.Locations[fallbackIndex], true
	}
	return screenplay.Location{}, false
}

func normalizeLooseBeatType(input string) string {
	switch strings.TrimSpace(strings.ToLower(input)) {
	case "dialogue":
		return "dialogue"
	default:
		return "action"
	}
}

func normalizeLooseCharacterID(input, beatType string, characters []screenplay.Character) string {
	if beatType != "dialogue" {
		return ""
	}
	if input != "" {
		for _, character := range characters {
			if character.ID == input {
				return input
			}
		}
	}
	if len(characters) > 0 {
		return characters[0].ID
	}
	return ""
}

func ensureDialogueBeat(beats []screenplay.Beat, plannedScene screenplay.Scene) []screenplay.Beat {
	for _, beat := range beats {
		if beat.Type == "dialogue" && strings.TrimSpace(beat.Content) != "" {
			return beats
		}
	}
	for _, beat := range plannedScene.Beats {
		if beat.Type == "dialogue" && strings.TrimSpace(beat.Content) != "" {
			return append(beats, beat)
		}
	}
	return beats
}

func inferLooseInteriorExterior(locationName string) string {
	if containsAny(locationName, "站", "街", "路", "广场", "码头", "外") {
		return "EXT"
	}
	return "INT"
}

func normalizeLooseTime(input string) string {
	switch {
	case containsAny(input, "深夜", "夜", "晚上", "凌晨"):
		return "NIGHT"
	case containsAny(input, "清晨", "早晨", "早上"):
		return "MORNING"
	default:
		return "DAY"
	}
}

func inferLooseEmotion(content string) string {
	switch {
	case containsAny(content, "撬过", "有人进来", "别睡", "不能再等"):
		return "tense"
	case containsAny(content, "一定要", "查出", "追"):
		return "determined"
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func looseSlug(input string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	input = strings.ReplaceAll(input, " ", "_")
	input = strings.ReplaceAll(input, "-", "_")
	if input == "" {
		return "generated"
	}
	return input
}

func containsAny(input string, fragments ...string) bool {
	for _, fragment := range fragments {
		if strings.Contains(input, fragment) {
			return true
		}
	}
	return false
}
