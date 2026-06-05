package workflow

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/AsperforMias/ScriptForge/backend/internal/ingest"
	"github.com/AsperforMias/ScriptForge/backend/internal/job"
	"github.com/AsperforMias/ScriptForge/backend/internal/screenplay"
)

type OutlineChapter struct {
	Index    int
	Title    string
	Summary  string
	Conflict string
}

type OutlineBundle struct {
	Chapters []OutlineChapter
}

type EntityBundle struct {
	Characters []screenplay.Character
	Locations  []screenplay.Location
}

type ScenePlan struct {
	Scenes []screenplay.Scene
}

func BuildOutline(source ingest.NormalizedSource) OutlineBundle {
	chapters := make([]OutlineChapter, 0, len(source.Chapters))
	for _, chapter := range source.Chapters {
		summary := summarize(chapter.Content)
		chapters = append(chapters, OutlineChapter{
			Index:    chapter.Index,
			Title:    chapter.Title,
			Summary:  summary,
			Conflict: buildConflict(summary),
		})
	}

	return OutlineBundle{Chapters: chapters}
}

func ExtractEntities(source ingest.NormalizedSource) EntityBundle {
	mainCharacterName := inferCharacterName(source)
	characters := []screenplay.Character{
		{
			ID:          "char_" + slugify(mainCharacterName),
			Name:        mainCharacterName,
			Role:        "protagonist",
			Description: "从章节行动线中推断出的主视角人物，负责把发现转化为调查行动。",
		},
	}

	locations := make([]screenplay.Location, 0, len(source.Chapters))
	for _, chapter := range source.Chapters {
		locationName := inferLocationName(chapter)
		locations = append(locations, screenplay.Location{
			ID:          fmt.Sprintf("loc_chapter_%02d", chapter.Index),
			Name:        locationName,
			Description: fmt.Sprintf("Primary dramatic location inferred from chapter %d.", chapter.Index),
		})
	}

	return EntityBundle{
		Characters: characters,
		Locations:  locations,
	}
}

func BuildScenePlan(source ingest.NormalizedSource, outline OutlineBundle, entities EntityBundle) ScenePlan {
	characterID := entities.Characters[0].ID
	scenes := make([]screenplay.Scene, 0, len(source.Chapters))

	for idx, chapter := range source.Chapters {
		chapterOutline := outline.Chapters[idx]
		locationID := fmt.Sprintf("loc_chapter_%02d", chapter.Index)
		scene := screenplay.Scene{
			ID:             fmt.Sprintf("scene_%03d", idx+1),
			Title:          chapter.Title,
			SourceChapters: []int{chapter.Index},
			Slugline: screenplay.Slugline{
				InteriorExterior: inferInteriorExterior(chapter.Content),
				LocationID:       locationID,
				Time:             inferTime(chapter.Content),
			},
			Summary:   chapterOutline.Summary,
			Objective: buildObjective(chapterOutline, chapter.Content),
			Beats: []screenplay.Beat{
				{
					Type:    "action",
					Content: chapterOutline.Summary,
				},
				{
					Type:        "dialogue",
					CharacterID: characterID,
					Content:     buildDialogue(chapterOutline, chapter.Content),
					Emotion:     inferEmotion(chapter.Content),
				},
			},
			Notes: screenplay.SceneNotes{
				AdaptationReason: "将章节中的关键发现压缩为单一可拍场景，并保留主角的判断与行动动机。",
				OpenQuestions:    inferOpenQuestions(chapter.Content),
			},
		}
		scenes = append(scenes, scene)
	}

	return ScenePlan{Scenes: scenes}
}

func BuildDocument(req job.CreateJobRequest, source ingest.NormalizedSource, outline OutlineBundle, entities EntityBundle, plan ScenePlan) screenplay.Document {
	sourceChapters := make([]screenplay.SourceChapter, 0, len(outline.Chapters))
	for _, chapter := range outline.Chapters {
		sourceChapters = append(sourceChapters, screenplay.SourceChapter{
			Index:   chapter.Index,
			Title:   chapter.Title,
			Summary: chapter.Summary,
		})
	}

	return screenplay.Document{
		Version: "1.0",
		Source: screenplay.Source{
			Title:        source.Title,
			Author:       source.Author,
			Language:     source.Language,
			ChapterCount: len(source.Chapters),
			Chapters:     sourceChapters,
		},
		Adaptation: screenplay.Adaptation{
			Style:    req.Adaptation.Style,
			Audience: req.Adaptation.Audience,
			Notes:    req.Adaptation.Notes,
		},
		Characters: entities.Characters,
		Locations:  entities.Locations,
		Scenes:     plan.Scenes,
		Validation: screenplay.Validation{
			Status:   "passed",
			Warnings: []string{},
		},
	}
}

func summarize(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return "This chapter introduces a new dramatic development."
	}
	if len([]rune(content)) <= 80 {
		return content
	}
	return string([]rune(content)[:80]) + "..."
}

func buildConflict(summary string) string {
	switch {
	case containsAny(summary, "项目", "汇报", "客户", "方案", "数据", "提案", "会议"):
		return "项目推进进入关键节点，主角必须在时间压力下查清内部失误或背叛。"
	case containsAny(summary, "比赛", "接力", "训练", "跑道", "队伍", "队友", "决赛"):
		return "比赛临近，主角必须在团队压力和胜负期待之间做出选择。"
	case containsAny(summary, "线索", "车站", "寄信人", "追踪"):
		return "主角决定主动追查线索，把被动戒备转化为现实行动。"
	case containsAny(summary, "门锁", "被人动过", "停在走廊"):
		return "主角意识到私人空间可能已经被入侵，必须先判断危险是否仍在现场。"
	case containsAny(summary, "字条", "别睡", "提前进入"):
		return "匿名警告把模糊的不安变成了明确威胁，主角必须判断这是不是针对她的布局。"
	default:
		return "章节中的新信息迫使主角做出下一步戏剧行动。"
	}
}

func buildObjective(chapter OutlineChapter, content string) string {
	switch {
	case containsAny(content, "项目", "汇报", "客户", "方案", "数据", "提案", "会议"):
		return "在正式汇报前确认项目风险，并决定是内部止损还是当场摊牌。"
	case containsAny(content, "比赛", "接力", "训练", "跑道", "队伍", "队友", "决赛"):
		return "在比赛开始前稳住队伍配合，并把临场压力转成可执行的战术。"
	case containsAny(content, "线索", "车站", "寄信人", "追踪"):
		return "顺着已有线索主动追查寄信人，让故事从受威胁转向反向调查。"
	case containsAny(content, "门锁", "被人动过", "走廊"):
		return "确认房间是否已经失守，并决定主角该撤离还是进入现场。"
	case containsAny(content, "字条", "别睡", "提前进入"):
		return "弄清匿名警告的可信度，并把威胁来源从猜测推进到具体目标。"
	default:
		return fmt.Sprintf("把第 %d 章的核心事件整理成明确、可拍摄的戏剧动作。", chapter.Index)
	}
}

func buildDialogue(chapter OutlineChapter, content string) string {
	switch {
	case containsAny(content, "项目", "汇报", "客户", "方案", "数据", "提案", "会议"):
		return "如果现在不把问题找出来，明天整个项目都会失控。"
	case containsAny(content, "比赛", "接力", "训练", "跑道", "队伍", "队友", "决赛"):
		return "就算少一个人，我们也得把这场接力跑完。"
	case containsAny(content, "线索", "车站", "寄信人", "追踪"):
		return "线索既然指向车站，我就不能再等了。"
	case containsAny(content, "门锁", "被人动过", "走廊"):
		return "门锁被动过，屋里也许还有人。"
	case containsAny(content, "字条", "别睡", "提前进入"):
		return "这张字条不是恶作剧，对方知道我今晚会回来。"
	default:
		return chapter.Conflict
	}
}

func inferEmotion(content string) string {
	switch {
	case containsAny(content, "比赛", "接力", "训练", "跑道", "队伍", "队友", "决赛"):
		return "determined"
	case containsAny(content, "项目", "汇报", "客户", "方案", "数据", "提案", "会议"):
		return "focused"
	case containsAny(content, "门锁", "别睡", "被人动过", "危险"):
		return "tense"
	case containsAny(content, "线索", "前往", "追踪", "决定"):
		return "determined"
	default:
		return "focused"
	}
}

func inferOpenQuestions(content string) []string {
	questions := make([]string, 0, 2)
	switch {
	case containsAny(content, "项目", "汇报", "客户", "方案", "数据", "提案", "会议"):
		questions = append(questions, "是谁在关键节点动了项目数据？")
	case containsAny(content, "比赛", "接力", "训练", "跑道", "队伍", "队友", "决赛"):
		questions = append(questions, "队伍能否在比赛开始前重新建立信任？")
	case containsAny(content, "线索", "寄信人", "车站"):
		questions = append(questions, "车站线索会把主角引向谁？")
	case containsAny(content, "门锁", "被人动过"):
		questions = append(questions, "是谁在主角回家前动过门锁？")
	case containsAny(content, "字条", "别睡"):
		questions = append(questions, "留下字条的人为什么知道主角的作息？")
	}
	return questions
}

func inferCharacterName(source ingest.NormalizedSource) string {
	for _, chapter := range source.Chapters {
		if candidate := inferLeadingName(chapter.Content); candidate != "" {
			return candidate
		}
	}

	candidates := extractCJKPhrases(strings.Join(chapterContents(source.Chapters), " "))
	if len(candidates) > 0 {
		if utf8.RuneCountInString(candidates[0]) >= 2 {
			return string([]rune(candidates[0])[:2])
		}
		return candidates[0]
	}
	return "主角"
}

func inferLocationName(chapter ingest.NormalizedChapter) string {
	keywords := []string{"公寓", "房间", "走廊", "办公室", "会议室", "街道", "学校", "教室", "操场", "跑道", "看台", "咖啡馆", "医院", "仓库", "车站"}
	for _, keyword := range keywords {
		if strings.Contains(chapter.Content, keyword) {
			return keyword
		}
	}
	return fmt.Sprintf("Chapter %d Main Location", chapter.Index)
}

func inferInteriorExterior(content string) string {
	if strings.Contains(content, "街") || strings.Contains(content, "路") || strings.Contains(content, "广场") || strings.Contains(content, "车站") || strings.Contains(content, "码头") || strings.Contains(content, "操场") || strings.Contains(content, "跑道") || strings.Contains(content, "看台") || strings.Contains(content, "天台") {
		return "EXT"
	}
	return "INT"
}

func inferTime(content string) string {
	switch {
	case strings.Contains(content, "夜"), strings.Contains(content, "凌晨"), strings.Contains(content, "今晚"), strings.Contains(content, "晚上"), strings.Contains(content, "傍晚"):
		return "NIGHT"
	case strings.Contains(content, "早"), strings.Contains(content, "清晨"), strings.Contains(content, "一早"):
		return "MORNING"
	default:
		return "DAY"
	}
}

func extractCJKPhrases(input string) []string {
	re := regexp.MustCompile(`[\p{Han}]{2,4}`)
	seen := map[string]struct{}{}
	results := []string{}
	for _, candidate := range re.FindAllString(input, -1) {
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		results = append(results, candidate)
		if len(results) == 3 {
			break
		}
	}
	return results
}

func inferLeadingName(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	runes := []rune(content)
	if len(runes) < 2 {
		return ""
	}

	candidate := string(runes[:2])
	if containsStopWord(candidate) {
		return ""
	}

	return candidate
}

func containsStopWord(input string) bool {
	stopWords := []string{"今天", "第二", "第三", "第一", "凌晨", "晚上", "清晨"}
	for _, stopWord := range stopWords {
		if input == stopWord {
			return true
		}
	}
	return false
}

func containsAny(input string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}
	return false
}

func chapterContents(chapters []ingest.NormalizedChapter) []string {
	results := make([]string, 0, len(chapters))
	for _, chapter := range chapters {
		results = append(results, chapter.Content)
	}
	return results
}

func slugify(input string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return "protagonist"
	}
	input = strings.ReplaceAll(input, " ", "_")
	return input
}
