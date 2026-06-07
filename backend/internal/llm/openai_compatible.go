package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/AsperforMias/ScriptForge/backend/internal/job"
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

type providerChapterContext struct {
	Index               int      `json:"index"`
	Title               string   `json:"title"`
	Content             string   `json:"content"`
	EvidenceExcerpt     string   `json:"evidence_excerpt,omitempty"`
	SuggestedSceneCount int      `json:"suggested_scene_count,omitempty"`
	CharacterCandidates []string `json:"character_candidates,omitempty"`
	LocationCandidates  []string `json:"location_candidates,omitempty"`
	KeyLines            []string `json:"key_lines,omitempty"`
}

var (
	quotedLinePattern        = regexp.MustCompile(`“([^”]{1,30})”`)
	boldLinePattern          = regexp.MustCompile(`\*\*([^*]{1,30})\*\*`)
	explicitLocationPattern  = regexp.MustCompile(`[\p{Han}A-Za-z0-9]{0,12}(?:报刊亭|医院|病房|地下室|老房子|楼道|客厅|楼梯口|走廊|车站|茶馆|操场|教室|会议室|公寓|房间|花坛|藏书馆|议事厅|账房|庄园|公国|码头|仓库|跑道|便利店|十字路口|楼梯口|门口)`)
	chapterPrefixPattern     = regexp.MustCompile(`^第[一二三四五六七八九十百0-9]+章\s*`)
	genericLocationNames     = []string{"房间", "地点", "现场", "室内", "室外", "家里", "屋里"}
	suspiciousCharacterNames = []string{"像是", "没有立刻", "不是", "什么", "这里", "那里"}
	validCharacterRoles      = []string{"protagonist", "supporting", "antagonist", "narrator", "other"}
	commonSingleSurnames     = "赵钱孙李周吴郑王冯陈褚卫蒋沈韩杨朱秦尤许何吕施张孔曹严华金魏陶姜戚谢邹喻柏水窦章云苏潘葛奚范彭郎鲁韦昌马苗凤花方俞任袁柳酆鲍史唐费廉岑薛雷贺倪汤滕殷罗毕郝邬安常乐于时傅皮卞齐康伍余元卜顾孟平黄和穆萧尹姚邵湛汪祁毛禹狄米贝明臧计伏成戴谈宋茅庞熊纪舒屈项祝董梁杜阮蓝闵席季麻强贾路娄危江童颜郭梅盛林刁钟徐丘骆高夏蔡田樊胡凌霍虞万支柯昝管卢莫房裘缪干解应宗丁宣贲邓郁单杭洪包诸左石崔吉龚程邢滑裴陆荣翁荀羊於惠甄曲家封芮羿储靳汲邴糜松井段富巫乌焦巴弓牧隗山谷车侯伊宁仇武符刘景詹束龙叶幸司韶郜黎蓟薄印宿白怀蒲台从鄂索籍赖卓蔺屠蒙池乔阴郁胥能苍双闻莘党翟谭贡劳逄姬申扶堵冉宰郦雍璩桑桂濮牛寿通边扈燕冀郏浦尚农温别庄晏柴瞿阎充慕连茹习宦艾鱼容向古易慎戈廖庾终暨居衡步都耿满弘匡国文寇广禄阙东欧殳沃利蔚越夔隆师巩厍聂晁勾敖融冷訾辛阚那简饶空曾沙乜养鞠须丰巢关蒯相查后荆红游竺权逯盖益桓公"
	commonCompoundSurnames   = []string{"欧阳", "司马", "上官", "东方", "独孤", "南宫", "夏侯", "诸葛", "尉迟", "皇甫", "长孙", "宇文", "司徒", "司空", "令狐", "慕容", "公孙", "澹台", "公冶", "宗政", "濮阳", "淳于", "单于", "太叔", "申屠", "公羊", "赫连", "轩辕", "仲孙", "钟离", "闾丘", "长乐", "拓跋"}
)

func NewOpenAICompatibleGenerator(cfg ProviderConfig) Generator {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	model := strings.TrimSpace(cfg.Model)
	apiKey := strings.TrimSpace(cfg.APIKey)
	if baseURL == "" || model == "" || apiKey == "" {
		return NewUnavailableGenerator("openai_compatible provider requires LLM_BASE_URL, LLM_MODEL, and LLM_API_KEY")
	}

	timeout := 180 * time.Second
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

	messageContent, err := g.requestMessageContent(ctx, requestBody)
	if err != nil {
		return GenerateResult{}, err
	}

	yamlText := extractYAMLResponse(messageContent)
	debug := &DebugSnapshot{
		Provider:           g.Name(),
		Model:              g.model,
		RawContent:         yamlText,
		OriginalRawContent: yamlText,
	}

	document, warnings, parseMode, parseErr := parseGeneratedDocument(yamlText, req)
	if parseErr == nil {
		document.Validation.Warnings = append(document.Validation.Warnings, warnings...)
		document.Validation.Warnings = append(document.Validation.Warnings, "generated via openai_compatible provider")
		debug.ParseMode = parseMode
		return GenerateResult{
			Document: document,
			Warnings: document.Validation.Warnings,
			Debug:    debug,
		}, nil
	}

	debug.ParseErrors = append(debug.ParseErrors, parseErr.Error())
	lastYAMLText := yamlText
	lastParseErr := parseErr
	for retry := 1; retry <= 2; retry++ {
		regeneratedYAML, retryErr := g.regenerateValidYAML(ctx, req, lastYAMLText, lastParseErr, retry)
		if retryErr != nil {
			debug.ParseErrors = append(debug.ParseErrors, retryErr.Error())
			return GenerateResult{}, NewInvocationErrorWithDebug(g.Name(), fmt.Errorf("parse yaml response: %w", lastParseErr), debug)
		}
		debug.RetryRawContents = append(debug.RetryRawContents, regeneratedYAML)
		debug.RawContent = regeneratedYAML

		document, warnings, parseMode, parseErr = parseGeneratedDocument(regeneratedYAML, req)
		if parseErr == nil {
			document.Validation.Warnings = append(document.Validation.Warnings, warnings...)
			document.Validation.Warnings = append(document.Validation.Warnings, "generated via openai_compatible provider")
			document.Validation.Warnings = append(document.Validation.Warnings, fmt.Sprintf("provider output required %d format retry pass(es) before YAML parsing succeeded", retry))
			debug.ParseMode = parseMode
			return GenerateResult{
				Document: document,
				Warnings: document.Validation.Warnings,
				Debug:    debug,
			}, nil
		}

		debug.ParseErrors = append(debug.ParseErrors, parseErr.Error())
		lastYAMLText = regeneratedYAML
		lastParseErr = parseErr
	}

	return GenerateResult{}, NewInvocationErrorWithDebug(g.Name(), fmt.Errorf("parse yaml response: %w", lastParseErr), debug)
}

func (g *OpenAICompatibleGenerator) requestMessageContent(ctx context.Context, requestBody []byte) (string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(g.baseURL, "/")+"/chat/completions", bytes.NewReader(requestBody))
	if err != nil {
		return "", NewInvocationError(g.Name(), err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return "", NewInvocationError(g.Name(), err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", NewInvocationError(g.Name(), err)
	}

	if resp.StatusCode >= 300 {
		return "", NewInvocationError(g.Name(), fmt.Errorf("status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(payload))))
	}

	var completion chatCompletionsResponse
	if err := json.Unmarshal(payload, &completion); err != nil {
		return "", NewInvocationError(g.Name(), err)
	}
	if completion.Error != nil {
		return "", NewInvocationError(g.Name(), errors.New(completion.Error.Message))
	}
	if len(completion.Choices) == 0 {
		return "", NewInvocationError(g.Name(), fmt.Errorf("empty choices"))
	}

	messageContent, err := extractMessageContent(completion.Choices[0].Message.Content)
	if err != nil {
		return "", NewInvocationError(g.Name(), fmt.Errorf("extract message content: %w", err))
	}
	return messageContent, nil
}

func (g *OpenAICompatibleGenerator) regenerateValidYAML(ctx context.Context, req GenerateRequest, previousYAML string, parseErr error, retry int) (string, error) {
	requestBody, err := g.buildRetryRequest(req, previousYAML, parseErr, retry)
	if err != nil {
		return "", err
	}
	messageContent, err := g.requestMessageContent(ctx, requestBody)
	if err != nil {
		return "", err
	}
	return extractYAMLResponse(messageContent), nil
}

func (g *OpenAICompatibleGenerator) buildRequest(req GenerateRequest) ([]byte, error) {
	return g.buildGenerationRequest(req, "", "", 0.2)
}

func (g *OpenAICompatibleGenerator) buildRetryRequest(req GenerateRequest, previousYAML string, parseErr error, retry int) ([]byte, error) {
	extraSystem := "Your previous attempt produced malformed YAML. Regenerate the full screenplay YAML from scratch using the same context. Re-decide characters, locations, scenes, sluglines, objectives, beats, notes, evidence, review, and validation together as one coherent document, and return only valid parseable YAML."
	extraUser := fmt.Sprintf(
		"Previous attempt %d failed to parse with this error: %s\nReturn the full screenplay YAML again with correct indentation, quoting, list markers, and escaped content. Rebuild the entire YAML document instead of patching one field in isolation. Do not explain the fix.\nPrevious malformed attempt:\n%s",
		retry,
		parseErr.Error(),
		previousYAML,
	)
	return g.buildGenerationRequest(req, extraSystem, extraUser, 0)
}

func (g *OpenAICompatibleGenerator) buildGenerationRequest(req GenerateRequest, extraSystem, extraUser string, temperature float64) ([]byte, error) {
	contextPayload := buildProviderContext(req)

	contextJSON, err := json.MarshalIndent(contextPayload, "", "  ")
	if err != nil {
		return nil, err
	}

	requestBody := chatCompletionsRequest{
		Model:       g.model,
		Temperature: temperature,
		Messages: []chatCompletionInput{
			{
				Role: "system",
				Content: "You adapt Chinese novels into structured screenplay YAML. " +
					"Return only valid YAML that matches this exact root schema: version, source, adaptation, characters, locations, scenes, validation. " +
					"Do not invent alternative top-level keys such as metadata. " +
					"The provided character_candidates, location_candidates, and key lines are only low-trust grounding hints; always prefer the raw chapter text when they conflict. " +
					"Keep the total scene count close to the provided scene_count_target, and default to the chapter-level suggested_scene_count unless the raw chapter text clearly contains a separate second dramatic turn that needs its own shootable scene. " +
					"To keep YAML parseable, quote every non-empty string scalar value with double quotes. " +
					"Keep evidence excerpts short and plain; avoid multiline prose blocks, markdown markers, or decorative punctuation inside YAML values. " +
					"Ignore suspicious candidate names or generic locations if the chapter text does not support them. " +
					"Every scene must include slugline.interior_exterior, slugline.location_id, slugline.time, summary, beats, evidence, and review. " +
					"objective and notes.open_questions are optional: leave them empty or omit them when the source evidence is insufficient. " +
					"An objective must be an immediate scene task, not atmosphere-setting or broad truth-seeking copy. " +
					"All scenes must reference a valid location_id declared in locations. " +
					"Do not invent characters, locations, dialogue, objectives, or open questions that are not grounded in the provided context. " +
					"Prefer omission over fabrication when evidence is weak. " +
					"Do not repeat the same dialogue, beat, objective, or open question across scenes. " +
					"Use Chinese content values where appropriate, but keep schema keys exactly as requested. " +
					"Do not include markdown fences or explanations." +
					conditionalSuffix(" "+strings.TrimSpace(extraSystem)),
			},
			{
				Role: "user",
				Content: "Generate a screenplay YAML document from this structured context. Use chapter text as the primary source of truth; use candidates and hints only when they are explicitly supported by the chapter text:\n" +
					string(contextJSON) +
					"\nFollow this skeleton exactly:\n" +
					"version: \"1.0\"\n" +
					"source:\n  title: ...\n  author: ...\n  language: zh-CN\n  chapter_count: ...\n  chapters:\n    - index: 1\n      title: ...\n      summary: ...\n" +
					"adaptation:\n  style: ...\n  audience: ...\n  notes: []\n" +
					"characters:\n  - id: char_xxx\n    name: ...\n    role: protagonist\n    description: ...\n" +
					"locations:\n  - id: loc_xxx\n    name: ...\n    description: ...\n" +
					"scenes:\n  - id: scene_001\n    title: ...\n    source_chapters: [1]\n    slugline:\n      interior_exterior: INT\n      location_id: loc_xxx\n      time: NIGHT\n    summary: ...\n    objective: ...\n    beats:\n      - type: action\n        content: ...\n      - type: dialogue\n        character_id: char_xxx\n        content: ...\n        emotion: tense\n    notes:\n      adaptation_reason: ...\n      open_questions:\n        - ...\n" +
					"    evidence:\n      chapter_indexes: [1]\n      excerpt: ...\n      cues:\n        - ...\n    review:\n      confidence: medium\n      issues:\n        - ...\n" +
					"validation:\n  status: passed\n  warnings: []\n" +
					"Ensure source.chapter_count matches the input, every scene references valid chapter indexes, and dialogue beats reference valid character_id values. " +
					"Treat scene_count_target and each chapter's suggested_scene_count as the default structural plan; only exceed them when the source text itself shows a clearly separated extra dramatic turn. " +
					"Bad objective patterns include atmosphere-setting or generic truth-seeking copy such as 建立悬疑氛围, 建立悬疑基调, 制造紧迫感, 引入某人的警告, 了解发生了什么, 弄清楚真相. " +
					"If a scene detail is uncertain, lower review.confidence and record the issue instead of fabricating a polished answer. " +
					"Keep evidence.excerpt to one short sentence or phrase, and avoid any unescaped colon-heavy or quote-heavy text copied verbatim from the novel." +
					conditionalSuffix("\n"+strings.TrimSpace(extraUser)),
			},
		},
	}

	return json.Marshal(requestBody)
}

func conditionalSuffix(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return value
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

	for idx := range doc.Characters {
		doc.Characters[idx].Role = normalizeGeneratedCharacterRole(doc.Characters[idx].Role, idx)
	}

	for idx := range doc.Scenes {
		doc.Scenes[idx].Slugline.InteriorExterior = strings.ToUpper(strings.TrimSpace(doc.Scenes[idx].Slugline.InteriorExterior))
		doc.Scenes[idx].Slugline.Time = strings.ToUpper(strings.TrimSpace(doc.Scenes[idx].Slugline.Time))
		for beatIdx := range doc.Scenes[idx].Beats {
			doc.Scenes[idx].Beats[beatIdx].Type = normalizeGeneratedBeatType(doc.Scenes[idx].Beats[beatIdx].Type)
			if doc.Scenes[idx].Beats[beatIdx].Type != "dialogue" {
				doc.Scenes[idx].Beats[beatIdx].CharacterID = ""
			}
		}
	}

	return doc
}

func parseGeneratedDocument(yamlText string, req GenerateRequest) (screenplay.Document, []string, string, error) {
	document, err := screenplay.ParseYAML(yamlText)
	if err == nil {
		document = normalizeGeneratedDocument(document)
		document = enrichGeneratedDocument(document, req)
		if validateErr := screenplay.Validate(document); validateErr == nil {
			return document, nil, "canonical", nil
		}
	}

	fallbackDocument, err := parseLooseDocument(yamlText, req)
	if err != nil {
		return screenplay.Document{}, nil, "", err
	}

	fallbackDocument = normalizeGeneratedDocument(fallbackDocument)
	fallbackDocument = enrichGeneratedDocument(fallbackDocument, req)
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
			Evidence: buildProviderSceneEvidence(req, chapterIndex, idx),
			Review: &screenplay.Review{
				Confidence: "low",
				Issues: []string{
					"normalized from a loose provider scene; verify location, objective, and dialogue against the source chapters.",
				},
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

func enrichGeneratedDocument(doc screenplay.Document, req GenerateRequest) screenplay.Document {
	doc = repairGeneratedDocument(doc, req)
	doc = compressGeneratedScenes(doc, req)
	doc = repairGeneratedDocument(doc, req)
	for idx := range doc.Scenes {
		scene := &doc.Scenes[idx]
		chapterIndex := firstSceneChapter(scene.SourceChapters, idx, req)
		if scene.Evidence == nil {
			scene.Evidence = buildProviderSceneEvidence(req, chapterIndex, idx)
		}
		if scene.Review == nil {
			scene.Review = &screenplay.Review{
				Confidence: "medium",
			}
		}
		if strings.TrimSpace(scene.Review.Confidence) == "" {
			scene.Review.Confidence = "medium"
		}
	}
	return doc
}

func buildProviderSceneEvidence(req GenerateRequest, chapterIndex, fallbackIndex int) *screenplay.Evidence {
	chapter := lookupInputChapter(req, chapterIndex, fallbackIndex)
	excerpt := truncateRunes(normalizeWhitespace(chapter.Content), 220)
	if excerpt == "" {
		excerpt = lookupChapterSummary(req, chapterIndex, "")
	}
	if excerpt == "" {
		excerpt = chapter.Title
	}

	cues := []string{sanitizeChapterTitle(chapter.Title)}
	cues = append(cues, extractKeyLines(chapter.Content)...)
	cues = append(cues, extractExplicitLocationCandidates(chapter.Content)...)
	cues = append(cues, extractSupportedCharacterNames(req, chapter.Content)...)

	return &screenplay.Evidence{
		ChapterIndexes: []int{chapterIndex},
		Excerpt:        excerpt,
		Cues:           uniqueNonEmptyStrings(cues),
	}
}

func buildProviderContext(req GenerateRequest) map[string]any {
	chapters := make([]providerChapterContext, 0, len(req.Input.Source.Chapters))
	suggestedSceneCountByChapter := plannedSceneCountByChapter(req)

	for _, chapter := range req.Input.Source.Chapters {
		chapterContent := normalizeWhitespace(chapter.Content)
		chapters = append(chapters, providerChapterContext{
			Index:               chapter.Index,
			Title:               chapter.Title,
			Content:             chapterContent,
			EvidenceExcerpt:     truncateRunes(chapterContent, 220),
			SuggestedSceneCount: suggestedSceneCountByChapter[chapter.Index],
			CharacterCandidates: extractSupportedCharacterNames(req, chapter.Content),
			LocationCandidates:  extractExplicitLocationCandidates(chapter.Content),
			KeyLines:            extractKeyLines(chapter.Content),
		})
	}

	return map[string]any{
		"source": map[string]any{
			"title":         req.Input.Source.Title,
			"author":        req.Input.Source.Author,
			"language":      req.Source.Language,
			"chapter_count": len(req.Input.Source.Chapters),
		},
		"adaptation": map[string]any{
			"style":    req.Input.Adaptation.Style,
			"audience": req.Input.Adaptation.Audience,
			"notes":    req.Input.Adaptation.Notes,
		},
		"scene_count_target": len(req.Plan.Scenes),
		"chapters":           chapters,
	}
}

func plannedSceneCountByChapter(req GenerateRequest) map[int]int {
	counts := make(map[int]int, len(req.Input.Source.Chapters))
	for _, chapter := range req.Input.Source.Chapters {
		counts[chapter.Index] = 1
	}
	for _, scene := range req.Plan.Scenes {
		chapterIndex := firstSceneChapter(scene.SourceChapters, 0, req)
		counts[chapterIndex]++
	}
	for chapterIndex, count := range counts {
		if count <= 1 {
			counts[chapterIndex] = 1
			continue
		}
		counts[chapterIndex] = count - 1
	}
	return counts
}

func firstSceneChapter(sourceChapters []int, fallbackIndex int, req GenerateRequest) int {
	if len(sourceChapters) > 0 && sourceChapters[0] > 0 {
		return sourceChapters[0]
	}
	if fallbackIndex < len(req.Input.Source.Chapters) {
		return req.Input.Source.Chapters[fallbackIndex].Index
	}
	return fallbackIndex + 1
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

func uniqueNonEmptyStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func repairGeneratedDocument(doc screenplay.Document, req GenerateRequest) screenplay.Document {
	preferredCharacters := buildPreferredCharacters(req)
	doc.Characters = sanitizeGeneratedCharacters(doc.Characters, preferredCharacters, req)
	validCharacterIDs := make(map[string]struct{}, len(doc.Characters))
	for _, character := range doc.Characters {
		validCharacterIDs[character.ID] = struct{}{}
	}

	locationByID := make(map[string]screenplay.Location, len(doc.Locations))
	for _, location := range doc.Locations {
		locationByID[location.ID] = location
	}
	usedLocations := map[string]screenplay.Location{}
	fallbackCharacterID := ""
	if len(doc.Characters) > 0 {
		fallbackCharacterID = doc.Characters[0].ID
	}

	for idx := range doc.Scenes {
		scene := &doc.Scenes[idx]
		chapterIndex := firstSceneChapter(scene.SourceChapters, idx, req)
		chapter := lookupInputChapter(req, chapterIndex, idx)
		chapterText := sanitizeChapterTitle(chapter.Title) + " " + normalizeWhitespace(chapter.Content)
		scene.Objective = sanitizeSceneObjective(scene.Objective, chapterText, scene.Review)
		scene.Notes.OpenQuestions = sanitizeOpenQuestions(scene.Notes.OpenQuestions, chapterText, scene.Review)

		for beatIdx := range scene.Beats {
			beat := &scene.Beats[beatIdx]
			if beat.Type == "dialogue" {
				if _, ok := validCharacterIDs[beat.CharacterID]; !ok {
					beat.CharacterID = fallbackCharacterID
				}
			}
		}

		repairedLocation := chooseSceneLocation(scene, chapter, locationByID[scene.Slugline.LocationID], idx)
		scene.Slugline.LocationID = repairedLocation.ID
		if scene.Slugline.InteriorExterior != "INT" && scene.Slugline.InteriorExterior != "EXT" {
			scene.Slugline.InteriorExterior = inferLooseInteriorExterior(repairedLocation.Name)
		}
		if strings.TrimSpace(scene.Slugline.Time) == "" {
			scene.Slugline.Time = normalizeLooseTime(chapter.Content)
		}
		usedLocations[repairedLocation.ID] = repairedLocation
	}

	doc.Locations = orderedUsedLocations(doc.Scenes, usedLocations)
	return doc
}

func buildPreferredCharacters(req GenerateRequest) []screenplay.Character {
	sourceText := normalizeWhitespace(joinChapterContents(req.Input.Source.Chapters))
	characters := make([]screenplay.Character, 0, len(req.Entities.Characters))
	for _, character := range req.Entities.Characters {
		if !isSupportedPreferredCharacterName(character.Name, sourceText) {
			continue
		}
		characters = append(characters, character)
	}
	return characters
}

func sanitizeGeneratedCharacters(generated, preferred []screenplay.Character, req GenerateRequest) []screenplay.Character {
	sourceText := normalizeWhitespace(joinChapterContents(req.Input.Source.Chapters))
	result := make([]screenplay.Character, 0, len(generated)+len(preferred))
	seenNames := map[string]struct{}{}

	appendIfSupported := func(character screenplay.Character, validator func(string, string) bool) {
		name := strings.TrimSpace(character.Name)
		if !validator(name, sourceText) {
			return
		}
		if _, ok := seenNames[name]; ok {
			return
		}
		if strings.TrimSpace(character.ID) == "" {
			character.ID = "char_" + looseSlug(name)
		}
		if strings.TrimSpace(character.Role) == "" {
			character.Role = "supporting"
		}
		seenNames[name] = struct{}{}
		result = append(result, character)
	}

	for _, character := range generated {
		appendIfSupported(character, isSupportedGeneratedCharacterName)
	}
	if len(result) == 0 {
		for _, character := range preferred {
			appendIfSupported(character, isSupportedPreferredCharacterName)
		}
	}
	if len(result) == 0 {
		for _, fallback := range append(append([]screenplay.Character{}, preferred...), generated...) {
			if strings.TrimSpace(fallback.Name) == "" {
				continue
			}
			if strings.TrimSpace(fallback.ID) == "" {
				fallback.ID = "char_" + looseSlug(fallback.Name)
			}
			if strings.TrimSpace(fallback.Role) == "" {
				fallback.Role = "supporting"
			}
			if len(result) == 0 {
				fallback.Role = "protagonist"
			}
			result = append(result, fallback)
			if len(result) >= 2 {
				break
			}
		}
	}
	return result
}

func chooseSceneLocation(scene *screenplay.Scene, chapter job.ChapterBody, current screenplay.Location, sceneIndex int) screenplay.Location {
	candidates := extractExplicitLocationCandidates(chapter.Content)
	chapterText := normalizeWhitespace(chapter.Content)
	current = sanitizeLocationCandidate(current)
	if current.ID != "" &&
		isSupportedLocationName(current.Name) &&
		(strings.Contains(chapterText, strings.TrimSpace(current.Name)) || hasMeaningfulSourceOverlap(current.Name, chapterText)) {
		return current
	}

	bestCandidate := ""
	bestScore := -1
	sceneText := normalizeWhitespace(scene.Summary)
	if scene.Evidence != nil {
		sceneText = strings.TrimSpace(sceneText + " " + scene.Evidence.Excerpt + " " + strings.Join(scene.Evidence.Cues, " "))
	}
	for _, candidate := range candidates {
		if isGenericLocationName(candidate) {
			continue
		}
		score := locationCandidateScore(candidate)
		if strings.Contains(sceneText, candidate) || hasMeaningfulSourceOverlap(candidate, sceneText) {
			score += 25
		}
		if score > bestScore {
			bestScore = score
			bestCandidate = candidate
		}
	}
	if bestCandidate != "" {
		return screenplay.Location{
			ID:          "loc_" + looseSlug(bestCandidate),
			Name:        bestCandidate,
			Description: "Location grounded directly from the source chapter.",
		}
	}

	if current.ID != "" && strings.TrimSpace(current.Name) != "" {
		return current
	}

	fallbackName := fmt.Sprintf("chapter_%02d_location", sceneIndex+1)
	if len(candidates) > 0 {
		fallbackName = candidates[0]
	}
	return screenplay.Location{
		ID:          "loc_" + looseSlug(fallbackName),
		Name:        fallbackName,
		Description: "Location fallback derived from the source chapter.",
	}
}

func sanitizeSceneObjective(objective, chapterText string, review *screenplay.Review) string {
	objective = strings.TrimSpace(objective)
	if objective == "" {
		return ""
	}
	if looksLikeLLMTemplateText(objective) {
		return ""
	}
	return objective
}

func sanitizeOpenQuestions(questions []string, chapterText string, review *screenplay.Review) []string {
	result := make([]string, 0, len(questions))
	for _, question := range questions {
		trimmed := strings.TrimSpace(question)
		if trimmed == "" {
			continue
		}
		if looksLikeLLMTemplateText(trimmed) {
			continue
		}
		result = append(result, trimmed)
	}
	return uniqueNonEmptyStrings(result)
}

func buildProviderCharacterCandidates(req GenerateRequest, sourceText string) []string {
	names := extractSupportedCharacterNames(req, sourceText)
	return uniqueNonEmptyStrings(names)
}

func buildProviderLocationCandidates(req GenerateRequest) []string {
	locations := []string{}
	for _, chapter := range req.Input.Source.Chapters {
		locations = append(locations, extractExplicitLocationCandidates(chapter.Content)...)
	}
	return uniqueNonEmptyStrings(locations)
}

func extractSupportedCharacterNames(req GenerateRequest, sourceText string) []string {
	names := []string{}
	for _, character := range req.Entities.Characters {
		if isSupportedPreferredCharacterName(character.Name, sourceText) {
			names = append(names, character.Name)
		}
	}
	return uniqueNonEmptyStrings(names)
}

func extractExplicitLocationCandidates(content string) []string {
	matches := explicitLocationPattern.FindAllString(normalizeWhitespace(content), -1)
	filtered := make([]string, 0, len(matches))
	for _, match := range matches {
		location := normalizeLocationCandidate(match)
		if !isSupportedLocationName(location) {
			continue
		}
		filtered = append(filtered, location)
	}
	filtered = uniqueNonEmptyStrings(filtered)
	sort.SliceStable(filtered, func(i, j int) bool {
		return locationCandidateScore(filtered[i]) > locationCandidateScore(filtered[j])
	})
	return filtered
}

func extractKeyLines(content string) []string {
	lines := []string{}
	for _, match := range quotedLinePattern.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}
		lines = append(lines, truncateRunes(strings.TrimSpace(match[1]), 40))
	}
	for _, match := range boldLinePattern.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}
		lines = append(lines, strings.TrimSpace(match[1]))
	}
	return uniqueNonEmptyStrings(lines)
}

func hasMeaningfulSourceOverlap(text, sourceText string) bool {
	textRunes := []rune(normalizeWhitespace(text))
	sourceText = normalizeWhitespace(sourceText)
	for n := 4; n >= 2; n-- {
		for idx := 0; idx+n <= len(textRunes); idx++ {
			fragment := strings.TrimSpace(string(textRunes[idx : idx+n]))
			if utf8.RuneCountInString(fragment) < 2 || isCommonChineseFragment(fragment) {
				continue
			}
			if strings.Contains(sourceText, fragment) {
				return true
			}
		}
	}
	return false
}

func isCommonChineseFragment(fragment string) bool {
	common := []string{"先弄", "弄清", "再决定", "决定", "下一步", "接下来", "问题", "主角", "到底", "哪里", "什么", "是否", "能否", "继续", "追查", "真实", "意图"}
	return slices.Contains(common, fragment)
}

func looksLikeLLMTemplateText(text string) bool {
	normalized := normalizeWhitespace(text)
	if containsAny(normalized, "先弄清", "再决定", "下一步", "继续追查下去", "真实意图", "匿名留言", "接下来会发生什么") {
		return true
	}
	if containsAny(normalized, "建立悬疑氛围", "建立悬疑基调", "制造紧迫感", "推动剧情继续", "引入苏雯的警告", "引入警告", "了解发生了什么", "弄清楚真相") {
		return true
	}
	if strings.Contains(normalized, "展示") && containsAny(normalized, "归来动机", "异常态度", "人物状态") {
		return true
	}
	return false
}

func compressGeneratedScenes(doc screenplay.Document, req GenerateRequest) screenplay.Document {
	plannedCounts := plannedSceneCountByChapter(req)
	if len(plannedCounts) == 0 || len(doc.Scenes) == 0 {
		return doc
	}

	docCounts := map[int]int{}
	for idx, scene := range doc.Scenes {
		chapterIndex := firstSceneChapter(scene.SourceChapters, idx, req)
		docCounts[chapterIndex]++
	}

	needsCompression := false
	for chapterIndex, actualCount := range docCounts {
		if actualCount > 1 && plannedCounts[chapterIndex] <= 1 {
			needsCompression = true
			break
		}
	}
	if !needsCompression {
		return doc
	}

	grouped := make(map[int][]screenplay.Scene, len(req.Input.Source.Chapters))
	extras := make([]screenplay.Scene, 0)
	for idx, scene := range doc.Scenes {
		chapterIndex := firstSceneChapter(scene.SourceChapters, idx, req)
		if _, ok := plannedCounts[chapterIndex]; !ok {
			extras = append(extras, scene)
			continue
		}
		grouped[chapterIndex] = append(grouped[chapterIndex], scene)
	}

	compressed := make([]screenplay.Scene, 0, len(req.Input.Source.Chapters)+len(extras))
	for _, chapter := range req.Input.Source.Chapters {
		chapterScenes := grouped[chapter.Index]
		if len(chapterScenes) == 0 {
			continue
		}
		if plannedCounts[chapter.Index] > 1 || len(chapterScenes) == 1 {
			compressed = append(compressed, chapterScenes...)
			continue
		}
		compressed = append(compressed, mergeScenesForChapter(chapterScenes, chapter, req))
	}
	compressed = append(compressed, extras...)

	for idx := range compressed {
		compressed[idx].ID = fmt.Sprintf("scene_%03d", idx+1)
	}
	doc.Scenes = compressed
	doc.Locations = orderedUsedLocations(doc.Scenes, locationsByID(doc.Locations))
	return doc
}

func mergeScenesForChapter(scenes []screenplay.Scene, chapter job.ChapterBody, req GenerateRequest) screenplay.Scene {
	base := scenes[0]
	chapterText := sanitizeChapterTitle(chapter.Title) + " " + normalizeWhitespace(chapter.Content)
	merged := base
	merged.SourceChapters = []int{chapter.Index}
	merged.Summary = firstNonEmpty(base.Summary, truncateRunes(normalizeWhitespace(chapter.Content), 140), chapter.Title)
	merged.Objective = chooseMergedObjective(scenes, chapterText)
	merged.Beats = mergeSceneBeats(scenes)
	if len(merged.Beats) == 0 {
		merged.Beats = []screenplay.Beat{{Type: "action", Content: merged.Summary}}
	}
	merged.Notes.OpenQuestions = mergeOpenQuestionsForChapter(scenes, chapterText)
	merged.Evidence = mergeSceneEvidence(scenes, chapter.Index, chapter.Content, req)
	merged.Review = mergeChapterReview(scenes)
	return merged
}

func chooseMergedObjective(scenes []screenplay.Scene, chapterText string) string {
	for _, scene := range scenes {
		objective := strings.TrimSpace(scene.Objective)
		if objective == "" || looksLikeLLMTemplateText(objective) {
			continue
		}
		if hasMeaningfulSourceOverlap(objective, chapterText) {
			return objective
		}
	}
	return ""
}

func mergeSceneBeats(scenes []screenplay.Scene) []screenplay.Beat {
	seen := map[string]struct{}{}
	beats := make([]screenplay.Beat, 0, 6)
	for _, scene := range scenes {
		for _, beat := range scene.Beats {
			content := strings.TrimSpace(beat.Content)
			if content == "" {
				continue
			}
			key := strings.ToLower(strings.TrimSpace(beat.Type)) + "|" + normalizeWhitespace(content)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			beats = append(beats, beat)
			if len(beats) >= 5 {
				return beats
			}
		}
	}
	return beats
}

func mergeOpenQuestionsForChapter(scenes []screenplay.Scene, chapterText string) []string {
	result := make([]string, 0, 2)
	seen := map[string]struct{}{}
	for _, scene := range scenes {
		for _, question := range scene.Notes.OpenQuestions {
			trimmed := strings.TrimSpace(question)
			if trimmed == "" || looksLikeLLMTemplateText(trimmed) || !hasMeaningfulSourceOverlap(trimmed, chapterText) {
				continue
			}
			if _, ok := seen[trimmed]; ok {
				continue
			}
			seen[trimmed] = struct{}{}
			result = append(result, trimmed)
			if len(result) >= 2 {
				return result
			}
		}
	}
	return result
}

func mergeSceneEvidence(scenes []screenplay.Scene, chapterIndex int, chapterContent string, req GenerateRequest) *screenplay.Evidence {
	merged := &screenplay.Evidence{
		ChapterIndexes: []int{chapterIndex},
		Cues:           []string{},
	}
	for _, scene := range scenes {
		if scene.Evidence == nil {
			continue
		}
		if merged.Excerpt == "" && strings.TrimSpace(scene.Evidence.Excerpt) != "" {
			merged.Excerpt = strings.TrimSpace(scene.Evidence.Excerpt)
		}
		merged.Cues = append(merged.Cues, scene.Evidence.Cues...)
	}
	if merged.Excerpt == "" {
		fallback := buildProviderSceneEvidence(req, chapterIndex, 0)
		if fallback != nil {
			merged.Excerpt = fallback.Excerpt
			merged.Cues = append(merged.Cues, fallback.Cues...)
		}
	}
	if merged.Excerpt == "" {
		merged.Excerpt = truncateRunes(normalizeWhitespace(chapterContent), 220)
	}
	merged.Cues = uniqueNonEmptyStrings(merged.Cues)
	return merged
}

func mergeChapterReview(scenes []screenplay.Scene) *screenplay.Review {
	issues := []string{"multiple provider scenes for one chapter were merged to keep a stable editable draft; review scene boundaries manually."}
	confidence := "medium"
	for _, scene := range scenes {
		if scene.Review == nil {
			continue
		}
		issues = append(issues, scene.Review.Issues...)
		if scene.Review.Confidence == "low" {
			confidence = "low"
		}
	}
	return &screenplay.Review{
		Confidence: confidence,
		Issues:     uniqueNonEmptyStrings(issues),
	}
}

func locationsByID(locations []screenplay.Location) map[string]screenplay.Location {
	result := make(map[string]screenplay.Location, len(locations))
	for _, location := range locations {
		result[location.ID] = location
	}
	return result
}

func normalizeGeneratedCharacterRole(input string, idx int) string {
	role := strings.TrimSpace(strings.ToLower(input))
	if slices.Contains(validCharacterRoles, role) {
		return role
	}

	switch role {
	case "", "lead", "main", "hero", "主角":
		if idx == 0 {
			return "protagonist"
		}
		return "supporting"
	case "support", "supporting_role", "side", "extra", "extras", "minor", "guest", "配角":
		return "supporting"
	case "villain", "反派":
		return "antagonist"
	case "narration", "旁白":
		return "narrator"
	case "路人", "群众":
		return "other"
	default:
		if idx == 0 {
			return "protagonist"
		}
		return "supporting"
	}
}

func normalizeGeneratedBeatType(input string) string {
	beatType := strings.TrimSpace(strings.ToLower(input))
	switch beatType {
	case "dialogue", "dialog", "speech", "line":
		return "dialogue"
	case "transition":
		return "transition"
	case "note":
		return "note"
	case "action", "memory", "flashback", "narration", "narrative", "inner_voice", "inner voice", "voice_over", "voice over", "monologue", "":
		return "action"
	default:
		return "action"
	}
}

func isLowConfidenceReview(review *screenplay.Review) bool {
	return review != nil && strings.TrimSpace(strings.ToLower(review.Confidence)) == "low"
}

func isSupportedGeneratedCharacterName(name, sourceText string) bool {
	name = strings.TrimSpace(name)
	if name == "" || !strings.Contains(sourceText, name) {
		return false
	}
	if slices.Contains(suspiciousCharacterNames, name) {
		return false
	}
	runes := []rune(name)
	if len(runes) < 2 || len(runes) > 6 {
		return false
	}
	for _, r := range runes {
		if !unicode.Is(unicode.Han, r) {
			return false
		}
	}
	if looksLikeCharacterFragment(name) {
		return false
	}
	return true
}

func isSupportedPreferredCharacterName(name, sourceText string) bool {
	if !isSupportedGeneratedCharacterName(name, sourceText) {
		return false
	}
	return looksLikeChinesePersonalName(name)
}

func looksLikeCharacterFragment(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return true
	}

	if strings.Contains(name, "一个人") ||
		strings.Contains(name, "只出现") ||
		strings.Contains(name, "对话越") {
		return true
	}

	if strings.HasPrefix(name, "问题") ||
		strings.HasPrefix(name, "在") ||
		strings.HasPrefix(name, "从") ||
		strings.HasPrefix(name, "向") ||
		strings.HasPrefix(name, "往") ||
		strings.HasPrefix(name, "于") ||
		strings.HasPrefix(name, "反而") ||
		strings.HasPrefix(name, "而是") ||
		strings.HasPrefix(name, "如果") ||
		strings.HasPrefix(name, "因为") ||
		strings.HasPrefix(name, "但是") ||
		strings.HasPrefix(name, "而且") ||
		strings.HasPrefix(name, "于是") ||
		strings.HasPrefix(name, "然后") ||
		strings.HasPrefix(name, "只是") ||
		strings.HasPrefix(name, "对话") {
		return true
	}

	return false
}

func looksLikeChinesePersonalName(name string) bool {
	name = strings.TrimSpace(name)
	runes := []rune(name)
	if len(runes) < 2 || len(runes) > 4 {
		return false
	}
	for _, r := range runes {
		if !unicode.Is(unicode.Han, r) {
			return false
		}
	}

	if len(runes) >= 3 {
		prefix := string(runes[:2])
		if slices.Contains(commonCompoundSurnames, prefix) {
			return true
		}
	}

	return strings.ContainsRune(commonSingleSurnames, runes[0])
}

func isSupportedLocationName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	runes := []rune(name)
	if len(runes) < 2 || len(runes) > 12 {
		return false
	}
	for _, r := range runes {
		if !(unicode.Is(unicode.Han, r) || unicode.IsDigit(r) || unicode.IsLetter(r)) {
			return false
		}
	}
	if looksLikeNarrativeLocationText(name) {
		return false
	}
	return true
}

func isGenericLocationName(name string) bool {
	return slices.Contains(genericLocationNames, strings.TrimSpace(name))
}

func normalizeLocationCandidate(input string) string {
	input = strings.TrimSpace(input)
	suffixes := []string{"报刊亭", "报亭", "医院", "病房", "地下室", "老房子", "楼道", "客厅", "楼梯口", "走廊", "车站", "茶馆", "操场", "教室", "会议室", "公寓", "房间", "花坛", "藏书馆", "议事厅", "账房", "庄园", "公国", "码头", "仓库", "跑道", "便利店", "十字路口", "门口"}
	for _, suffix := range suffixes {
		if idx := strings.Index(input, suffix); idx >= 0 {
			start := idx
			for start > 0 {
				r, size := utf8.DecodeLastRuneInString(input[:start])
				if r == utf8.RuneError || unicode.IsSpace(r) || strings.ContainsRune("，。！？、“”‘’（）()：:；;,.!?*", r) {
					break
				}
				if idx-start >= 12 {
					break
				}
				start -= size
			}
			input = strings.TrimSpace(input[start : idx+len(suffix)])
			break
		}
	}
	trimPrefixes := []string{"站在", "走到", "回到", "来到", "先去", "别信", "刚走到", "看到", "看见", "回头看", "她在", "他在", "在", "到"}
	for _, prefix := range trimPrefixes {
		input = strings.TrimPrefix(input, prefix)
	}
	if core := bestLocationCore(input); core != "" {
		return core
	}
	if idx := strings.LastIndex(input, "的"); idx >= 0 && idx < len(input)-len("的") {
		input = input[idx+len("的"):]
	}
	if core := bestLocationCore(input); core != "" {
		return core
	}
	return strings.TrimSpace(input)
}

func bestLocationCore(input string) string {
	best := ""
	bestScore := -1
	for _, suffix := range []string{"报刊亭", "报亭", "医院", "病房", "地下室", "老房子", "楼道", "客厅", "楼梯口", "走廊", "公寓", "房间", "便利店", "十字路口", "门口"} {
		searchFrom := 0
		for {
			idx := strings.Index(input[searchFrom:], suffix)
			if idx < 0 {
				break
			}
			idx += searchFrom
			start := idx
			limit := 8
			for start > 0 {
				r, size := utf8.DecodeLastRuneInString(input[:start])
				if r == utf8.RuneError || unicode.IsSpace(r) || strings.ContainsRune("，。！？、“”‘’（）()：:；;,.!?*", r) {
					break
				}
				if utf8.RuneCountInString(input[start:idx]) >= limit {
					break
				}
				start -= size
			}

			candidate := strings.TrimSpace(input[start : idx+len(suffix)])
			candidate = trimNarrativeLocationCandidate(candidate, suffix)
			if candidate == "" {
				searchFrom = idx + len(suffix)
				continue
			}

			score := locationCandidateScore(candidate)
			if score > bestScore || (score == bestScore && utf8.RuneCountInString(candidate) > utf8.RuneCountInString(best)) {
				best = candidate
				bestScore = score
			}
			searchFrom = idx + len(suffix)
		}
	}
	return best
}

func trimNarrativeLocationCandidate(candidate, suffix string) string {
	candidate = strings.TrimSpace(candidate)
	narrativeFragments := []string{"回到", "走到", "刚走到", "来到", "站在", "停在", "看见", "看到", "见过", "整理", "送来", "找到", "传来", "想起", "意识到", "发现", "进入", "检查", "观察", "收到", "拿起", "打开", "决定", "试图", "顺着", "带着"}
	for _, fragment := range narrativeFragments {
		if idx := strings.LastIndex(candidate, fragment); idx >= 0 {
			candidate = strings.TrimSpace(candidate[idx+len(fragment):])
		}
	}
	if idx := strings.LastIndex(candidate, "的"); idx >= 0 && idx < len(candidate)-len("的") {
		candidate = strings.TrimSpace(candidate[idx+len("的"):])
	}
	if strings.HasSuffix(candidate, suffix) {
		prefix := strings.TrimSpace(strings.TrimSuffix(candidate, suffix))
		if prefix != "" && allRunesInSet(prefix, "过了着在到进出入") {
			candidate = suffix
		}
	}
	prefix := strings.TrimSuffix(candidate, suffix)
	switch strings.TrimSpace(prefix) {
	case "但", "人", "己", "渡", "门", "许言", "父亲", "昏暗":
		return suffix
	}
	if candidate == "" {
		return suffix
	}
	return candidate
}

func allRunesInSet(input, allowed string) bool {
	for _, r := range input {
		if !strings.ContainsRune(allowed, r) {
			return false
		}
	}
	return input != ""
}

func sanitizeLocationCandidate(location screenplay.Location) screenplay.Location {
	name := normalizeLocationCandidate(location.Name)
	if !isSupportedLocationName(name) {
		return screenplay.Location{}
	}
	location.Name = name
	location.ID = "loc_" + looseSlug(name)
	return location
}

func looksLikeNarrativeLocationText(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return true
	}

	if containsAny(name,
		"意识到", "发现", "确认", "决定", "试图", "观察", "检查", "看到", "看见",
		"进入", "回到", "来到", "走到", "走向", "站在", "停在", "留在", "找到",
		"拿起", "打开", "收到", "提醒", "警告", "带着", "顺着", "有人", "别睡",
	) {
		return true
	}

	if strings.HasPrefix(name, "她") ||
		strings.HasPrefix(name, "他") {
		return true
	}

	return false
}

func locationCandidateScore(name string) int {
	score := utf8.RuneCountInString(name)
	if isGenericLocationName(name) {
		score -= 10
	}
	switch {
	case containsAny(name, "报刊亭", "报亭", "医院", "老房子", "地下室", "病房", "客厅", "楼道"):
		score += 20
	case containsAny(name, "十字路口", "便利店", "楼梯口", "走廊"):
		score += 10
	}
	return score
}

func orderedUsedLocations(scenes []screenplay.Scene, locations map[string]screenplay.Location) []screenplay.Location {
	result := make([]screenplay.Location, 0, len(locations))
	seen := map[string]struct{}{}
	for _, scene := range scenes {
		location, ok := locations[scene.Slugline.LocationID]
		if !ok {
			continue
		}
		if _, exists := seen[location.ID]; exists {
			continue
		}
		seen[location.ID] = struct{}{}
		result = append(result, location)
	}
	return result
}

func lookupInputChapter(req GenerateRequest, chapterIndex, fallbackIndex int) job.ChapterBody {
	for _, chapter := range req.Input.Source.Chapters {
		if chapter.Index == chapterIndex {
			return chapter
		}
	}
	if fallbackIndex >= 0 && fallbackIndex < len(req.Input.Source.Chapters) {
		return req.Input.Source.Chapters[fallbackIndex]
	}
	return job.ChapterBody{Index: chapterIndex}
}

func joinChapterContents(chapters []job.ChapterBody) string {
	parts := make([]string, 0, len(chapters))
	for _, chapter := range chapters {
		parts = append(parts, chapter.Title)
		parts = append(parts, chapter.Content)
	}
	return strings.Join(parts, "\n")
}

func sanitizeChapterTitle(title string) string {
	title = strings.TrimSpace(title)
	return chapterPrefixPattern.ReplaceAllString(title, "")
}

func normalizeWhitespace(input string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(input)), " ")
}

func truncateRunes(input string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(input))
	if len(runes) <= limit {
		return string(runes)
	}
	return strings.TrimSpace(string(runes[:limit])) + "..."
}
