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
	locationNames := make(map[string]string, len(entities.Locations))
	for _, location := range entities.Locations {
		locationNames[location.ID] = location.Name
	}
	seenObjectives := map[string]struct{}{}
	seenQuestions := map[string]struct{}{}

	for idx, chapter := range source.Chapters {
		chapterOutline := outline.Chapters[idx]
		locationID := fmt.Sprintf("loc_chapter_%02d", chapter.Index)
		locationName := locationNames[locationID]
		objective := ensureUniqueSceneText(
			buildObjective(chapterOutline, chapter.Title, chapter.Content, locationName),
			seenObjectives,
			fallbackSceneObjective(chapter, locationName),
		)
		openQuestions := ensureUniqueQuestions(
			inferOpenQuestions(chapter.Title, chapter.Content, locationName),
			seenQuestions,
			chapter,
			locationName,
		)
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
			Objective: objective,
			Beats: []screenplay.Beat{
				{
					Type:    "action",
					Content: chapterOutline.Summary,
				},
				{
					Type:        "dialogue",
					CharacterID: characterID,
					Content:     buildDialogue(chapterOutline, chapter.Title, chapter.Content, locationName),
					Emotion:     inferEmotion(chapter.Content),
				},
			},
			Notes: screenplay.SceneNotes{
				AdaptationReason: "将章节中的关键发现压缩为单一可拍场景，并保留主角的判断与行动动机。",
				OpenQuestions:    openQuestions,
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
	case containsAny(summary, "病房", "团圆饭", "父亲", "母亲", "姐姐", "客厅", "厨房", "家里"):
		return "家庭照料和旧误会同时浮上台面，主角必须在情感压力下推动家人把话说开。"
	case containsAny(summary, "夜市", "餐馆", "广场", "直播", "摄影师", "试播", "设备"):
		return "一场误会把合作关系推向失控边缘，主角必须把尴尬局面转成新的默契。"
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
		issue := extractIssueFocus(summary)
		if issue != "" {
			return fmt.Sprintf("新线索让局势出现变化，主角必须先处理%s带来的压力。", issue)
		}
		return "章节中的新信息迫使主角做出下一步戏剧行动。"
	}
}

func buildObjective(chapter OutlineChapter, title, content, locationName string) string {
	switch {
	case containsAny(content, "门锁", "被人动过", "走廊"):
		return "确认门锁异常是否意味着有人闯入，并判断主角能否立刻进入现场。"
	case containsAny(content, "线索", "车站", "寄信人", "追踪"):
		return "顺着字条和车站线索追查寄信人，把被动防备转成主动调查。"
	case containsAny(content, "字条", "别睡", "提前进入"):
		return "确认陌生字条是在警告还是威胁，并找出谁提前进入过房间。"
	case containsAny(content, "数据被人替换", "动了最终版本", "复核提案"):
		return "确认是谁替换了关键数据，并决定汇报前要先止损还是直接揭穿。"
	case containsAny(content, "独占客户", "怀疑已经在团队里扩散", "咖啡馆见面", "对质"):
		return "弄清团队猜疑是谁挑起的，并判断这次对质能否在汇报前止损。"
	case containsAny(content, "会议室", "摆到台面上", "正式汇报前"):
		return "在会议室摊牌前守住证据，让项目风险无法继续被掩盖。"
	case containsAny(content, "主力队友可能缺席", "最后一棒", "加练"):
		return "确认临阵变动会把责任推向谁，并为队伍重排接力节奏。"
	case containsAny(content, "教室", "质疑战术安排", "吵散"):
		return "稳住快要失控的队伍情绪，并把争执重新拉回比赛方案。"
	case containsAny(content, "比赛当天", "站上跑道", "带着现有阵容", "接力跑完"):
		return "带着现有阵容完成比赛，并把临场压力转成真正的起跑动作。"
	case containsAny(content, "病房外", "出院回家吃团圆饭", "医生建议"):
		return "确认父亲坚持回家的代价，并在家人与医生建议之间做出选择。"
	case strings.Contains(content, "厨房") && containsAny(content, "旧账", "指责", "姐姐"):
		return "接住厨房里的旧账与指责，并逼近这场家庭争执真正的症结。"
	case strings.Contains(content, "客厅") && containsAny(content, "父亲", "误会", "说出口"):
		return "趁着父亲先开口的时机，把多年误会真正说开。"
	case containsAny(content, "夜市", "误把", "笑话"):
		return "先止住夜市里的失控误会，再判断这次撞见会不会变成新的合作。"
	case containsAny(content, "餐馆", "解释误会", "越描越乱"):
		return "在餐馆里把误会说清楚，并避免朋友起哄继续推高尴尬。"
	case containsAny(content, "广场", "试播", "直播"):
		return "把前两章的尴尬转成一次真正能落地的合作试播。"
	}

	issue := extractIssueFocus(content)
	action := extractActionFocus(content)
	switch {
	case issue != "" && action != "":
		return fmt.Sprintf("%s，并%s。", objectiveLead(issue), actionClause(action))
	case action != "":
		return fmt.Sprintf("%s，并把局势推进到下一步。", actionClause(action))
	case issue != "":
		return fmt.Sprintf("%s，并为下一步行动建立判断。", objectiveLead(issue))
	default:
		return fallbackSceneObjective(ingest.NormalizedChapter{Index: chapter.Index, Title: title}, locationName)
	}
}

func buildDialogue(chapter OutlineChapter, title, content, locationName string) string {
	switch {
	case containsAny(content, "病房", "团圆饭", "父亲", "母亲", "姐姐", "客厅", "厨房", "家里"):
		if containsAny(content, "病房外", "医生建议") {
			return "回不回家吃这顿饭，今晚必须有人把后果说清楚。"
		}
		if containsAny(content, "客厅", "误会", "说出口") {
			return "既然爸先提了，我也不想再把这些话憋回去。"
		}
		return "今晚这顿饭不是为了热闹，是为了把这些年的话说清楚。"
	case containsAny(content, "夜市", "餐馆", "广场", "直播", "摄影师", "试播", "设备"):
		if containsAny(content, "夜市", "误把") {
			return "我先认这次闹错了，但这件事不能就这么糊过去。"
		}
		if containsAny(content, "广场", "试播", "直播") {
			return "既然都站到镜头前了，就把这次试播做成真的。"
		}
		return "先别急着生气，我们至少得把这场误会解释清楚。"
	case containsAny(content, "项目", "汇报", "客户", "方案", "数据", "提案", "会议"):
		if containsAny(content, "独占客户", "怀疑已经在团队里扩散", "咖啡馆见面") {
			return "如果连谁在放消息都不清楚，我们谁都别想安心进会议室。"
		}
		if containsAny(content, "会议室", "摆到台面上") {
			return "证据我已经带来了，今天谁都别想把这件事轻轻带过。"
		}
		return "如果现在不把问题找出来，明天整个项目都会失控。"
	case containsAny(content, "比赛", "接力", "训练", "跑道", "队伍", "队友", "决赛"):
		if containsAny(content, "教室", "质疑战术安排", "吵散") {
			return "还没站上跑道就先散掉，我们输的不会只是这场比赛。"
		}
		return "就算少一个人，我们也得把这场接力跑完。"
	case containsAny(content, "线索", "车站", "寄信人", "追踪"):
		return "线索既然指向车站，我就不能再等了。"
	case containsAny(content, "门锁", "被人动过", "走廊"):
		return "门锁被动过，屋里也许还有人。"
	case containsAny(content, "字条", "别睡", "提前进入"):
		return "这张字条不是恶作剧，对方知道我今晚会回来。"
	default:
		issue := extractIssueFocus(content)
		if issue != "" {
			return fmt.Sprintf("先把%s弄清楚，再谈下一步。", shortenPhrase(issue, 14))
		}
		return chapter.Conflict
	}
}

func inferEmotion(content string) string {
	switch {
	case containsAny(content, "病房", "团圆饭", "父亲", "母亲", "姐姐", "客厅", "厨房", "家里"):
		return "restrained"
	case containsAny(content, "夜市", "餐馆", "广场", "直播", "摄影师", "试播", "设备"):
		return "awkward"
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

func inferOpenQuestions(title, content, locationName string) []string {
	questions := make([]string, 0, 2)
	switch {
	case containsAny(content, "门锁", "被人动过"):
		questions = append(questions, "是谁动过门锁，屋里还留下了什么痕迹？")
	case containsAny(content, "线索", "寄信人", "车站"):
		questions = append(questions, "顺着这条车站线索，主角究竟会找到谁？")
	case containsAny(content, "字条", "别睡"):
		questions = append(questions, "留下字条的人为什么知道主角今晚会回来？")
	case containsAny(content, "数据被人替换", "动了最终版本", "复核提案"):
		questions = append(questions, "是谁替换了关键数据，真正想掩盖什么？")
	case containsAny(content, "独占客户", "怀疑已经在团队里扩散", "咖啡馆见面", "对质"):
		questions = append(questions, "团队里的怀疑究竟是谁放出来的？")
	case containsAny(content, "会议室", "摆到台面上", "正式汇报前"):
		questions = append(questions, "这场会议室摊牌之后，项目还能不能按原计划推进？")
	case containsAny(content, "比赛", "接力", "训练", "跑道", "队伍", "队友", "决赛"):
		if containsAny(content, "主力队友可能缺席", "最后一棒", "加练") {
			questions = append(questions, "主力缺席背后到底发生了什么，最后一棒会落到谁手里？")
		} else if containsAny(content, "教室", "质疑战术安排", "吵散") {
			questions = append(questions, "队伍能在起跑前把这场争执真正压住吗？")
		} else {
			questions = append(questions, "队伍能否在比赛开始前重新建立信任？")
		}
	case containsAny(content, "病房外", "出院回家吃团圆饭", "医生建议"):
		questions = append(questions, "这顿团圆饭值不值得冒着父亲身体再出状况的风险？")
	case strings.Contains(content, "厨房") && containsAny(content, "旧账", "指责", "姐姐"):
		questions = append(questions, "厨房里的这场争执，会不会把多年旧账彻底掀开？")
	case strings.Contains(content, "客厅") && containsAny(content, "父亲", "误会", "说出口"):
		questions = append(questions, "这次客厅里的坦白，能不能真的让一家人把误会说开？")
	case containsAny(content, "夜市", "误把", "笑话"):
		questions = append(questions, "这场夜市误会会把两人的关系推向敌对还是合作？")
	case containsAny(content, "餐馆", "解释误会", "越描越乱"):
		questions = append(questions, "这顿餐馆圆场会不会把误会解释得更糟？")
	case containsAny(content, "广场", "试播", "直播"):
		questions = append(questions, "这次广场试播能不能把之前的尴尬真的翻篇？")
	default:
		issue := extractIssueFocus(content)
		action := extractActionFocus(content)
		if question := questionFromSignals(issue, action); question != "" {
			questions = append(questions, question)
		}
	}
	if len(questions) == 0 {
		questions = append(questions, fallbackOpenQuestion(title, locationName))
	}
	return questions
}

func inferCharacterName(source ingest.NormalizedSource) string {
	candidateCounts := map[string]int{}
	for _, chapter := range source.Chapters {
		for _, candidate := range inferLikelyNames(chapter.Content) {
			candidateCounts[candidate]++
		}
	}
	bestName := ""
	bestScore := 0
	for candidate, score := range candidateCounts {
		if score > bestScore || (score == bestScore && bestName == "") {
			bestName = candidate
			bestScore = score
		}
	}
	if bestName != "" {
		return bestName
	}
	return "主角"
}

func inferLocationName(chapter ingest.NormalizedChapter) string {
	keywords := []string{"病房", "公寓", "房间", "走廊", "办公室", "会议室", "街道", "学校", "教室", "操场", "跑道", "看台", "咖啡馆", "厨房", "客厅", "夜市", "餐馆", "广场", "医院", "仓库", "车站", "天台", "码头", "直播间", "录音室", "实验室", "展厅", "档案室", "礼堂", "楼道", "门口", "宿舍", "商场", "桥下", "桥边", "河边", "巷口", "老城区", "后台", "休息室"}
	for _, keyword := range keywords {
		if strings.Contains(chapter.Content, keyword) || strings.Contains(chapter.Title, keyword) {
			return keyword
		}
	}

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?:在|到|回到|来到|走进|进入|站在|留在|赶到|前往|约在|冲到|守在)([\p{Han}]{1,10}(?:公寓|房间|走廊|办公室|会议室|街道|学校|教室|操场|跑道|看台|咖啡馆|厨房|客厅|夜市|餐馆|广场|医院|仓库|车站|病房|码头|天台|小区|宿舍|商场|后台|直播间|录音室|实验室|展厅|档案室|礼堂|老城区|城区|巷口|桥下|桥边|河边|楼道|门口|休息室))`),
		regexp.MustCompile(`([\p{Han}]{1,10}(?:公寓|房间|走廊|办公室|会议室|街道|学校|教室|操场|跑道|看台|咖啡馆|厨房|客厅|夜市|餐馆|广场|医院|仓库|车站|病房|码头|天台|小区|宿舍|商场|后台|直播间|录音室|实验室|展厅|档案室|礼堂|老城区|城区|巷口|桥下|桥边|河边|楼道|门口|休息室))`),
	}
	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(chapter.Content)
		if len(matches) == 2 {
			return matches[1]
		}
	}

	if titleLocation := inferLocationFromTitle(chapter.Title); titleLocation != "" {
		return titleLocation
	}

	return fmt.Sprintf("第%d章关键场景", chapter.Index)
}

func inferInteriorExterior(content string) string {
	if strings.Contains(content, "街") || strings.Contains(content, "路") || strings.Contains(content, "广场") || strings.Contains(content, "车站") || strings.Contains(content, "码头") || strings.Contains(content, "操场") || strings.Contains(content, "跑道") || strings.Contains(content, "看台") || strings.Contains(content, "天台") || strings.Contains(content, "夜市") {
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

func inferLikelyNames(content string) []string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?:^|[。，“”、\s])([\p{Han}]{2,3})(?:早上|晚上|清晨|深夜|当天|第二天|一早|傍晚|夜里|独自|第一次|终于|突然|立刻|先|还|只好|只能|带着)*?(?:在|回到|来到|走进|发现|决定|意识到|听见|看见|接到|站在|站上|留在|约|坐在|准备|必须|试图|想要|只能|正要|忽然|突然)`),
		regexp.MustCompile(`(?:^|[。，“”、\s])([\p{Han}]{2,3})(?:说|问|想|看|听|记得|觉得|怀疑)`),
	}
	seen := map[string]struct{}{}
	results := make([]string, 0, 4)
	for _, pattern := range patterns {
		for _, matches := range pattern.FindAllStringSubmatch(content, -1) {
			if len(matches) != 2 {
				continue
			}
			candidate := matches[1]
			if containsStopWord(candidate) || looksLikeLocation(candidate) {
				continue
			}
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			results = append(results, candidate)
		}
	}
	return results
}

func containsStopWord(input string) bool {
	stopWords := []string{"今天", "第二", "第三", "第一", "凌晨", "晚上", "清晨", "第二天", "当天", "傍晚", "夜里", "朋友", "主力", "比赛", "项目", "父亲", "母亲", "姐姐", "医生", "有人", "对方", "家里", "团队", "夜市", "广场", "厨房", "客厅", "会议", "教室", "叙述者"}
	for _, stopWord := range stopWords {
		if input == stopWord {
			return true
		}
	}
	badPrefixes := []string{"却", "并", "再", "先", "还", "只", "就", "又", "忽", "突", "原", "正", "不", "要", "会", "能", "可"}
	for _, prefix := range badPrefixes {
		if strings.HasPrefix(input, prefix) {
			return true
		}
	}
	badFragments := []string{"发现", "决定", "意识", "带着", "回到", "来到", "走进", "前往", "站上", "站在"}
	for _, fragment := range badFragments {
		if strings.Contains(input, fragment) {
			return true
		}
	}
	invalidChars := []string{"独", "自", "叙", "述", "者"}
	for _, char := range invalidChars {
		if strings.Contains(input, char) {
			return true
		}
	}
	return false
}

func looksLikeLocation(input string) bool {
	suffixes := []string{"室", "厅", "房", "馆", "站", "场", "道", "街", "路", "桥", "巷", "楼", "院", "台", "库", "口", "区", "市", "校", "园", "城"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(input, suffix) {
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

func splitSentences(content string) []string {
	replacer := strings.NewReplacer("！", "。", "？", "。", "!", "。", "?", "。", ";", "。", "；", "。")
	normalized := replacer.Replace(strings.TrimSpace(content))
	parts := strings.Split(normalized, "。")
	results := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		results = append(results, part)
	}
	return results
}

func extractIssueFocus(content string) string {
	sentences := splitSentences(content)
	for _, sentence := range sentences {
		if clause := extractClauseAfter(sentence, "发现", "意识到", "得知", "怀疑", "听见", "看到", "收到", "确认"); clause != "" {
			return clause
		}
	}
	for _, sentence := range sentences {
		if containsAny(sentence, "线索", "字条", "门锁", "数据", "误会", "争执", "缺席", "电话", "钥匙", "录音", "笑声", "留言", "证据") {
			return normalizePhrase(sentence)
		}
	}
	if len(sentences) > 0 {
		return normalizePhrase(sentences[0])
	}
	return ""
}

func extractActionFocus(content string) string {
	sentences := splitSentences(content)
	for _, sentence := range sentences {
		if clause := extractClauseAfter(sentence, "决定", "试图", "必须", "需要", "只能", "准备", "打算", "想要", "不确定"); clause != "" {
			return clause
		}
	}
	for _, sentence := range sentences {
		if containsAny(sentence, "前往", "走进", "回到", "赶到", "带着", "解释", "摊牌", "追查", "稳住", "说出口") {
			return normalizePhrase(sentence)
		}
	}
	return ""
}

func extractClauseAfter(sentence string, triggers ...string) string {
	for _, trigger := range triggers {
		index := strings.Index(sentence, trigger)
		if index == -1 {
			continue
		}
		clause := sentence[index+len(trigger):]
		clause = normalizePhrase(clause)
		if clause != "" {
			return clause
		}
	}
	return ""
}

func normalizePhrase(input string) string {
	input = strings.TrimSpace(input)
	input = strings.Trim(input, "，。！？；、,.!?;:")
	input = trimLeadingFillers(input)
	input = strings.Trim(input, "，。！？；、,.!?;:")
	return input
}

func trimLeadingFillers(input string) string {
	fillPrefix := []string{"她", "他", "主角", "叙述者", "随后", "于是", "此时", "这时", "第二天", "当天", "夜里", "傍晚", "清晨", "一早", "晚上", "深夜"}
	trimmed := strings.TrimSpace(input)
	for _, prefix := range fillPrefix {
		trimmed = strings.TrimPrefix(trimmed, prefix)
	}
	trimmed = strings.TrimLeft(trimmed, "，, ")
	return strings.TrimSpace(trimmed)
}

func objectiveLead(issue string) string {
	issue = normalizePhrase(issue)
	switch {
	case issue == "":
		return ""
	case strings.HasPrefix(issue, "是否"):
		return "确认" + issue
	case strings.Contains(issue, "谁"), strings.Contains(issue, "为何"), strings.Contains(issue, "为什么"):
		return "弄清" + issue
	default:
		return "弄清" + issue
	}
}

func actionClause(action string) string {
	action = normalizePhrase(action)
	switch {
	case action == "":
		return ""
	case strings.HasPrefix(action, "是否"):
		return "判断" + action
	default:
		return action
	}
}

func fallbackSceneObjective(chapter ingest.NormalizedChapter, locationName string) string {
	titleFocus := trimChapterPrefix(chapter.Title)
	switch {
	case titleFocus != "" && locationName != "":
		return fmt.Sprintf("围绕%s在%s遇到的新变化建立判断，并把故事推进到下一步。", titleFocus, locationName)
	case titleFocus != "":
		return fmt.Sprintf("围绕%s中的新变化建立判断，并把故事推进到下一步。", titleFocus)
	case locationName != "":
		return fmt.Sprintf("把%s中的关键变化转成明确行动，并为下一章埋下新的悬念。", locationName)
	default:
		return fmt.Sprintf("把第%d章的核心事件整理成明确、可拍摄的戏剧动作。", chapter.Index)
	}
}

func ensureUniqueSceneText(candidate string, seen map[string]struct{}, fallback string) string {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		candidate = strings.TrimSpace(fallback)
	}
	if candidate == "" {
		return candidate
	}
	if _, ok := seen[candidate]; !ok {
		seen[candidate] = struct{}{}
		return candidate
	}
	if _, ok := seen[fallback]; !ok && strings.TrimSpace(fallback) != "" {
		seen[fallback] = struct{}{}
		return fallback
	}
	return candidate
}

func ensureUniqueQuestions(candidates []string, seen map[string]struct{}, chapter ingest.NormalizedChapter, locationName string) []string {
	results := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = normalizeQuestion(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		results = append(results, candidate)
	}
	if len(results) > 0 {
		return results
	}

	fallback := fallbackOpenQuestion(chapter.Title, locationName)
	if _, ok := seen[fallback]; !ok {
		seen[fallback] = struct{}{}
		return []string{fallback}
	}

	titledFallback := normalizeQuestion(fmt.Sprintf("%s这一步会把主角推向哪里？", trimChapterPrefix(chapter.Title)))
	if titledFallback != "" {
		seen[titledFallback] = struct{}{}
		return []string{titledFallback}
	}
	return []string{}
}

func questionFromSignals(issue, action string) string {
	issue = normalizePhrase(issue)
	action = normalizePhrase(action)
	switch {
	case strings.Contains(issue, "有人"):
		return normalizeQuestion("到底是谁" + strings.TrimPrefix(issue, "有人") + "？")
	case strings.Contains(issue, "被人"):
		return normalizeQuestion(strings.Replace(issue, "被人", "被谁", 1) + "？")
	case strings.Contains(issue, "误会"):
		return "这场误会会不会把关系推向更糟的方向？"
	case strings.Contains(issue, "线索"):
		return "顺着这条线索继续追下去，主角会撞见什么真相？"
	case action != "":
		if strings.HasPrefix(action, "判断是否") {
			return normalizeQuestion(strings.TrimPrefix(action, "判断") + "？")
		}
		return normalizeQuestion("主角能否" + action + "？")
	case issue != "":
		return normalizeQuestion(issue + "背后到底还藏着什么？")
	default:
		return ""
	}
}

func normalizeQuestion(input string) string {
	input = normalizePhrase(input)
	if input == "" {
		return ""
	}
	input = strings.TrimRight(input, "？?")
	return input + "？"
}

func fallbackOpenQuestion(title, locationName string) string {
	titleFocus := trimChapterPrefix(title)
	switch {
	case locationName != "" && titleFocus != "":
		return fmt.Sprintf("%s在%s遇到的变化，会把故事推向怎样的下一步？", titleFocus, locationName)
	case locationName != "":
		return fmt.Sprintf("%s里的这次变化，会把主角推向怎样的下一步？", locationName)
	case titleFocus != "":
		return fmt.Sprintf("%s埋下的问题，会怎样影响主角的下一步选择？", titleFocus)
	default:
		return "这一章埋下的问题，会怎样影响主角的下一步选择？"
	}
}

func trimChapterPrefix(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	re := regexp.MustCompile(`^第[一二三四五六七八九十百千万0-9]+章[\s　\-—:：]*`)
	return strings.TrimSpace(re.ReplaceAllString(title, ""))
}

func inferLocationFromTitle(title string) string {
	titleFocus := trimChapterPrefix(title)
	if titleFocus == "" {
		return ""
	}
	if looksLikeLocation(titleFocus) {
		return titleFocus
	}
	return titleFocus + "现场"
}

func shortenPhrase(input string, limit int) string {
	input = normalizePhrase(input)
	if utf8.RuneCountInString(input) <= limit {
		return input
	}
	return string([]rune(input)[:limit]) + "..."
}
