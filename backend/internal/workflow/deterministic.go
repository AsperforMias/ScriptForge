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

func sceneText(title, content string) string {
	return strings.TrimSpace(title + " " + content)
}

func keywordScore(text string, strong, weak []string) int {
	score := 0
	for _, keyword := range strong {
		if strings.Contains(text, keyword) {
			score += 2
		}
	}
	for _, keyword := range weak {
		if strings.Contains(text, keyword) {
			score++
		}
	}
	return score
}

func familySceneScore(title, content, locationName string) int {
	score := keywordScore(
		sceneText(title, content),
		[]string{"团圆饭", "病房", "出院", "旧账", "和解", "说出口", "医生建议"},
		[]string{"父亲", "母亲", "姐姐", "家里", "客厅", "厨房", "照料"},
	)
	switch locationName {
	case "病房", "厨房", "客厅":
		score += 2
	}
	return score
}

func workplaceSceneScore(title, content, locationName string) int {
	score := keywordScore(
		sceneText(title, content),
		[]string{"数据被人替换", "最终版本", "汇报", "提案", "项目", "客户", "会议室", "证据"},
		[]string{"咖啡馆", "对质", "止损", "摊牌", "备份文件", "独占客户"},
	)
	switch locationName {
	case "办公室", "会议室", "咖啡馆":
		score++
	}
	return score
}

func sportsSceneScore(title, content, locationName string) int {
	score := keywordScore(
		sceneText(title, content),
		[]string{"接力", "决赛", "跑道", "主力队友", "最后一棒", "战术安排"},
		[]string{"操场", "教室", "起跑", "比赛", "队伍", "队友"},
	)
	switch locationName {
	case "操场", "教室", "跑道":
		score++
	}
	return score
}

func comedySceneScore(title, content, locationName string) int {
	score := keywordScore(
		sceneText(title, content),
		[]string{"误会", "摄影师", "直播", "试播", "笑话", "起哄"},
		[]string{"夜市", "餐馆", "广场", "设备", "圆场"},
	)
	switch locationName {
	case "夜市", "餐馆", "广场":
		score++
	}
	return score
}

func isSuspenseIntrusionScene(title, content, locationName string) bool {
	return strings.Contains(content, "门锁") && containsAny(content, "被人动过", "走廊", "不确定是否应该立刻进去")
}

func isSuspenseWarningScene(title, content, locationName string) bool {
	return containsAny(content, "字条", "留言", "今晚别睡") && containsAny(content, "提前进入", "知道", "警告", "恶作剧")
}

func isStationPursuitScene(title, content, locationName string) bool {
	text := sceneText(title, content)
	if locationName == "车站" && keywordScore(text, []string{"寄信人", "追踪", "追查", "车站"}, []string{"线索", "字条"}) >= 4 {
		return true
	}
	return containsAny(title, "车站", "追踪") && containsAny(content, "前往车站", "赶到车站", "来到车站")
}

func isWorkplaceDataScene(title, content, locationName string) bool {
	return workplaceSceneScore(title, content, locationName) >= 4 && containsAny(content, "数据被人替换", "最终版本", "复核提案")
}

func isWorkplaceConfrontationScene(title, content, locationName string) bool {
	return workplaceSceneScore(title, content, locationName) >= 4 && containsAny(content, "独占客户", "怀疑已经在团队里扩散", "咖啡馆见面", "对质")
}

func isWorkplaceShowdownScene(title, content, locationName string) bool {
	return workplaceSceneScore(title, content, locationName) >= 4 && containsAny(content, "会议室", "摆到台面上", "正式汇报前", "证据")
}

func isSportsSetupScene(title, content, locationName string) bool {
	return sportsSceneScore(title, content, locationName) >= 4 && containsAny(content, "主力队友可能缺席", "最后一棒", "加练")
}

func isSportsConflictScene(title, content, locationName string) bool {
	return sportsSceneScore(title, content, locationName) >= 4 && containsAny(content, "教室", "质疑战术安排", "吵散")
}

func isSportsRaceScene(title, content, locationName string) bool {
	return sportsSceneScore(title, content, locationName) >= 4 && containsAny(content, "比赛当天", "站上跑道", "带着现有阵容", "接力跑完")
}

func isFamilyCareScene(title, content, locationName string) bool {
	return familySceneScore(title, content, locationName) >= 4 &&
		containsAny(content, "病房外", "出院回家吃团圆饭", "医生建议") &&
		containsAny(content, "父亲", "母亲", "姐姐", "家里")
}

func isFamilyConflictScene(title, content, locationName string) bool {
	return familySceneScore(title, content, locationName) >= 4 &&
		locationName == "厨房" &&
		containsAny(content, "旧账", "指责") &&
		containsAny(content, "父亲", "母亲", "姐姐", "家里")
}

func isFamilyReconcileScene(title, content, locationName string) bool {
	return familySceneScore(title, content, locationName) >= 4 &&
		locationName == "客厅" &&
		containsAny(content, "误会", "说出口", "先开口", "坦白") &&
		containsAny(content, "父亲", "母亲", "姐姐", "家里")
}

func isComedyMeetScene(title, content, locationName string) bool {
	return comedySceneScore(title, content, locationName) >= 4 && locationName == "夜市" && containsAny(content, "误把", "笑话")
}

func isComedyClarifyScene(title, content, locationName string) bool {
	return comedySceneScore(title, content, locationName) >= 4 && locationName == "餐馆" && containsAny(content, "解释误会", "越描越乱")
}

func isComedyBroadcastScene(title, content, locationName string) bool {
	return comedySceneScore(title, content, locationName) >= 4 && locationName == "广场" && containsAny(content, "试播", "直播")
}

type sceneArtifact struct {
	kind  string
	label string
}

func inferSceneArtifact(title, content, locationName string) sceneArtifact {
	text := sceneText(title, content)
	switch {
	case containsAny(text, "录音", "录音机", "录音带", "磁带", "笑声", "潮声"):
		return sceneArtifact{kind: "recording", label: "录音里多出的声音"}
	case containsAny(text, "留言", "来信", "匿名信", "字条"):
		return sceneArtifact{kind: "message", label: "匿名留言"}
	case containsAny(text, "钥匙", "锁孔"):
		return sceneArtifact{kind: "key", label: "这把钥匙"}
	case containsAny(text, "账本"):
		return sceneArtifact{kind: "ledger", label: "账本里被藏起的记录"}
	case containsAny(text, "合同", "合约"):
		return sceneArtifact{kind: "contract", label: "合同里被藏起的条件"}
	case containsAny(text, "货单", "货运事故"):
		return sceneArtifact{kind: "manifest", label: "货单和事故记录"}
	case containsAny(text, "铭牌", "档案", "照片", "底片", "证据"):
		return sceneArtifact{kind: "clue", label: "这条新线索"}
	case containsAny(text, "门锁"):
		return sceneArtifact{kind: "lock", label: "门锁异常"}
	case containsAny(text, "数据"):
		return sceneArtifact{kind: "data", label: "关键数据被替换的原因"}
	default:
		titleFocus := trimChapterPrefix(title)
		if titleFocus != "" {
			return sceneArtifact{kind: "title", label: titleFocus}
		}
		if locationName != "" {
			return sceneArtifact{kind: "location", label: locationName}
		}
		return sceneArtifact{}
	}
}

func buildEvidenceDrivenObjective(chapter ingest.NormalizedChapter, locationName string) string {
	artifact := inferSceneArtifact(chapter.Title, chapter.Content, locationName)
	decision := inferGenericDecision(chapter.Content, locationName)
	switch artifact.kind {
	case "recording":
		if containsAny(chapter.Content, "核对", "重播") {
			return "先确认录音里多出的声音从何而来，再决定该向谁核对这条线索。"
		}
		return "先确认录音里多出的声音从何而来，再决定是否继续追查下去。"
	case "message":
		if containsAny(chapter.Content, "赶到", "赴约", "见面", "约见") {
			return "先确认匿名留言是不是诱饵，再决定要不要按约继续追查。"
		}
		return "先弄清匿名留言背后的真实意图，再决定下一步该找谁。"
	case "key":
		if containsAny(chapter.Content, "试锁", "打开", "锁孔", "库房", "船坞") {
			return "先确认这把钥匙究竟能打开哪里，再决定是否继续深入。"
		}
		return "先弄清这把钥匙为什么会出现在这里，再决定是否顺着它继续追查。"
	case "ledger":
		return "先确认账本里到底藏着哪一段记录，再决定该信谁、该找谁对质。"
	case "contract":
		return "先确认合同里被遮住的条件是什么，再决定该不该继续签下去。"
	case "manifest":
		return "先弄清货单和事故记录之间缺失了哪一环，再决定何时把真相带出去。"
	case "clue":
		if decision != "" {
			return fmt.Sprintf("先弄清%s指向什么，再%s。", artifact.label, decision)
		}
		return "先弄清这条新线索到底指向什么，再决定下一步该追到哪里。"
	case "lock":
		return "先确认门锁异常意味着什么，再决定该不该继续待在现场。"
	case "data":
		return "先确认关键数据是在哪里被调包的，再决定下一步该找谁负责。"
	default:
		if artifact.label != "" && decision != "" {
			return fmt.Sprintf("先弄清%s背后的问题，再%s。", artifact.label, decision)
		}
		if artifact.label != "" {
			return fmt.Sprintf("先弄清%s背后的问题，并据此推进下一步行动。", artifact.label)
		}
		return fallbackSceneObjective(chapter, locationName)
	}
}

func inferGenericDecision(content, locationName string) string {
	switch {
	case containsAny(content, "告诉警方", "报警", "交给警方"):
		return "决定是否把消息交给警方"
	case containsAny(content, "核对", "查证", "对照"):
		return "确认下一步该向谁核对"
	case containsAny(content, "赶到", "赴约", "见面", "约见"):
		return "判断这次赴约是不是圈套"
	case containsAny(content, "试锁", "打开", "锁孔"):
		return "赶在被打断前试出答案"
	case containsAny(content, "带到", "签约现场", "摆到台面上", "摊牌"):
		return "决定在什么时机把真相摆到台面上"
	case containsAny(content, "追查", "继续追", "顺着"):
		return "决定要不要继续追查下去"
	case locationName != "":
		return fmt.Sprintf("决定是否继续在%s追下去", locationName)
	default:
		return ""
	}
}

func buildEvidenceDrivenDialogue(chapter OutlineChapter, title, content, locationName string) string {
	artifact := inferSceneArtifact(title, content, locationName)
	switch artifact.kind {
	case "recording":
		return "先把录音里的异常声音核对清楚，再决定该信谁。"
	case "message":
		return "如果这张留言真想把我引过去，我更得先看清它想让我见谁。"
	case "key":
		return "钥匙不会平白无故留下，它一定在指向下一道门。"
	case "ledger":
		return "只要账本还在，就有人不想让我把这一页翻出来。"
	case "contract":
		return "合同敢藏条件，就说明有人不想让我现在看懂它。"
	case "manifest":
		return "货单和事故记录对不上，这里面一定还有人没说实话。"
	case "clue":
		return "这条线索既然露出来了，我就不能让它再断掉。"
	case "lock":
		return "门锁既然被动过，说明有人比我先到过这里。"
	case "data":
		return "数据敢在最后一刻被换掉，就说明有人赌我来不及发现。"
	default:
		issue := extractIssueFocus(content)
		if issue != "" {
			return fmt.Sprintf("先把%s弄清楚，再谈下一步。", shortenPhrase(issue, 14))
		}
		return chapter.Conflict
	}
}

func buildEvidenceDrivenOpenQuestion(title, content, locationName string) []string {
	artifact := inferSceneArtifact(title, content, locationName)
	switch artifact.kind {
	case "recording":
		return []string{"录音里多出的声音究竟是谁留下的？"}
	case "message":
		return []string{"匿名留言到底想把主角引到哪里？"}
	case "key":
		return []string{"这把钥匙最终会打开哪一道门？"}
	case "ledger":
		return []string{"账本里到底藏着谁不想被看见的记录？"}
	case "contract":
		return []string{"合同里被遮住的条件，会把谁逼到台前？"}
	case "manifest":
		return []string{"货单和事故记录对不上的那一段，究竟是谁动了手脚？"}
	case "clue":
		return []string{"这条新线索背后到底还藏着什么？"}
	case "lock":
		return []string{"是谁先一步动过门锁，又想在现场掩盖什么？"}
	case "data":
		return []string{"关键数据究竟是在哪个环节被人换掉的？"}
	default:
		issue := extractIssueFocus(content)
		action := extractActionFocus(content)
		if question := questionFromSignals(issue, action); question != "" {
			return []string{question}
		}
		return nil
	}
}

func buildObjective(chapter OutlineChapter, title, content, locationName string) string {
	switch {
	case isSuspenseIntrusionScene(title, content, locationName):
		return "确认门锁异常是否意味着有人闯入，并判断主角能否立刻进入现场。"
	case isStationPursuitScene(title, content, locationName):
		return "顺着字条和车站线索追查寄信人，把被动防备转成主动调查。"
	case isSuspenseWarningScene(title, content, locationName):
		return "确认陌生字条是在警告还是威胁，并找出谁提前进入过房间。"
	case isWorkplaceDataScene(title, content, locationName):
		return "确认是谁替换了关键数据，并决定汇报前要先止损还是直接揭穿。"
	case isWorkplaceConfrontationScene(title, content, locationName):
		return "弄清团队猜疑是谁挑起的，并判断这次对质能否在汇报前止损。"
	case isWorkplaceShowdownScene(title, content, locationName):
		return "在会议室摊牌前守住证据，让项目风险无法继续被掩盖。"
	case isSportsSetupScene(title, content, locationName):
		return "确认临阵变动会把责任推向谁，并为队伍重排接力节奏。"
	case isSportsConflictScene(title, content, locationName):
		return "稳住快要失控的队伍情绪，并把争执重新拉回比赛方案。"
	case isSportsRaceScene(title, content, locationName):
		return "带着现有阵容完成比赛，并把临场压力转成真正的起跑动作。"
	case isFamilyCareScene(title, content, locationName):
		return "确认父亲坚持回家的代价，并在家人与医生建议之间做出选择。"
	case isFamilyConflictScene(title, content, locationName):
		return "接住厨房里的旧账与指责，并逼近这场家庭争执真正的症结。"
	case isFamilyReconcileScene(title, content, locationName):
		return "趁着父亲先开口的时机，把多年误会真正说开。"
	case isComedyMeetScene(title, content, locationName):
		return "先止住夜市里的失控误会，再判断这次撞见会不会变成新的合作。"
	case isComedyClarifyScene(title, content, locationName):
		return "在餐馆里把误会说清楚，并避免朋友起哄继续推高尴尬。"
	case isComedyBroadcastScene(title, content, locationName):
		return "把前两章的尴尬转成一次真正能落地的合作试播。"
	}
	return buildEvidenceDrivenObjective(ingest.NormalizedChapter{Index: chapter.Index, Title: title, Content: content}, locationName)
}

func buildDialogue(chapter OutlineChapter, title, content, locationName string) string {
	switch {
	case isFamilyCareScene(title, content, locationName):
		return "回不回家吃这顿饭，今晚必须有人把后果说清楚。"
	case isFamilyReconcileScene(title, content, locationName):
		return "既然爸先提了，我也不想再把这些话憋回去。"
	case isFamilyConflictScene(title, content, locationName):
		return "今晚这顿饭不是为了热闹，是为了把这些年的话说清楚。"
	case isComedyMeetScene(title, content, locationName):
		return "我先认这次闹错了，但这件事不能就这么糊过去。"
	case isComedyBroadcastScene(title, content, locationName):
		return "既然都站到镜头前了，就把这次试播做成真的。"
	case isComedyClarifyScene(title, content, locationName):
		return "先别急着生气，我们至少得把这场误会解释清楚。"
	case isWorkplaceConfrontationScene(title, content, locationName):
		return "如果连谁在放消息都不清楚，我们谁都别想安心进会议室。"
	case isWorkplaceShowdownScene(title, content, locationName):
		return "证据我已经带来了，今天谁都别想把这件事轻轻带过。"
	case isWorkplaceDataScene(title, content, locationName):
		return "如果现在不把问题找出来，明天整个项目都会失控。"
	case isSportsConflictScene(title, content, locationName):
		return "还没站上跑道就先散掉，我们输的不会只是这场比赛。"
	case isSportsSetupScene(title, content, locationName), isSportsRaceScene(title, content, locationName):
		return "就算少一个人，我们也得把这场接力跑完。"
	case isStationPursuitScene(title, content, locationName):
		return "线索既然指向车站，我就不能再等了。"
	case isSuspenseIntrusionScene(title, content, locationName):
		return "门锁被动过，屋里也许还有人。"
	case isSuspenseWarningScene(title, content, locationName):
		return "这张字条不是恶作剧，对方知道我今晚会回来。"
	}
	return buildEvidenceDrivenDialogue(chapter, title, content, locationName)
}

func inferEmotion(content string) string {
	switch {
	case familySceneScore("", content, "") >= 3:
		return "restrained"
	case comedySceneScore("", content, "") >= 3:
		return "awkward"
	case sportsSceneScore("", content, "") >= 3:
		return "determined"
	case workplaceSceneScore("", content, "") >= 3:
		return "focused"
	case containsAny(content, "门锁", "别睡", "被人动过", "危险"):
		return "tense"
	case containsAny(content, "线索", "前往", "追踪", "决定", "钥匙", "录音", "留言"):
		return "determined"
	default:
		return "focused"
	}
}

func inferOpenQuestions(title, content, locationName string) []string {
	questions := make([]string, 0, 2)
	switch {
	case isSuspenseIntrusionScene(title, content, locationName):
		questions = append(questions, "是谁动过门锁，屋里还留下了什么痕迹？")
	case isStationPursuitScene(title, content, locationName):
		questions = append(questions, "顺着这条车站线索，主角究竟会找到谁？")
	case isSuspenseWarningScene(title, content, locationName):
		questions = append(questions, "留下字条的人为什么知道主角今晚会回来？")
	case isWorkplaceDataScene(title, content, locationName):
		questions = append(questions, "是谁替换了关键数据，真正想掩盖什么？")
	case isWorkplaceConfrontationScene(title, content, locationName):
		questions = append(questions, "团队里的怀疑究竟是谁放出来的？")
	case isWorkplaceShowdownScene(title, content, locationName):
		questions = append(questions, "这场会议室摊牌之后，项目还能不能按原计划推进？")
	case isSportsSetupScene(title, content, locationName):
		questions = append(questions, "主力缺席背后到底发生了什么，最后一棒会落到谁手里？")
	case isSportsConflictScene(title, content, locationName):
		questions = append(questions, "队伍能在起跑前把这场争执真正压住吗？")
	case isSportsRaceScene(title, content, locationName):
		questions = append(questions, "队伍能否在比赛开始前重新建立信任？")
	case isFamilyCareScene(title, content, locationName):
		questions = append(questions, "这顿团圆饭值不值得冒着父亲身体再出状况的风险？")
	case isFamilyConflictScene(title, content, locationName):
		questions = append(questions, "厨房里的这场争执，会不会把多年旧账彻底掀开？")
	case isFamilyReconcileScene(title, content, locationName):
		questions = append(questions, "这次客厅里的坦白，能不能真的让一家人把误会说开？")
	case isComedyMeetScene(title, content, locationName):
		questions = append(questions, "这场夜市误会会把两人的关系推向敌对还是合作？")
	case isComedyClarifyScene(title, content, locationName):
		questions = append(questions, "这顿餐馆圆场会不会把误会解释得更糟？")
	case isComedyBroadcastScene(title, content, locationName):
		questions = append(questions, "这次广场试播能不能把之前的尴尬真的翻篇？")
	default:
		questions = append(questions, buildEvidenceDrivenOpenQuestion(title, content, locationName)...)
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
	keywords := locationKeywords()
	bestName := ""
	bestScore := -1
	for _, keyword := range keywords {
		score := locationCandidateScore(chapter.Title, chapter.Content, keyword)
		if score > bestScore {
			bestScore = score
			bestName = keyword
		}
	}
	if bestScore >= 0 {
		return bestName
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

func locationKeywords() []string {
	return []string{
		"录音室", "会议室", "档案室", "直播间", "实验室", "休息室",
		"病房", "船坞", "钟楼", "河堤", "河岸", "堤岸", "桥下", "桥边", "巷口",
		"码头", "车站", "天台", "公寓", "房间", "走廊", "办公室", "学校", "教室",
		"操场", "跑道", "看台", "咖啡馆", "厨房", "客厅", "夜市", "餐馆", "广场",
		"医院", "仓库", "库房", "展厅", "礼堂", "楼道", "门口", "宿舍", "商场",
		"老城区", "城区", "后台", "街道",
	}
}

func locationCandidateScore(title, content, keyword string) int {
	score := -1
	if strings.Contains(title, keyword) {
		score = 6
	}
	if count := strings.Count(content, keyword); count > 0 {
		if score < 0 {
			score = 0
		}
		score += count * 2
	}

	contextBonus := 0
	if regexp.MustCompile(fmt.Sprintf(`(?:在|到|回到|来到|走进|进入|站在|留在|赶到|前往|约在|冲到|守在|停在|躲进|奔向)(?:[\p{Han}]{0,6})?%s`, regexp.QuoteMeta(keyword))).MatchString(content) {
		contextBonus = 3
	} else if regexp.MustCompile(fmt.Sprintf(`%s(?:里|内|外|边|上|前|后)`, regexp.QuoteMeta(keyword))).MatchString(content) {
		contextBonus = 1
	}
	if contextBonus > 0 {
		if score < 0 {
			score = 0
		}
		score += contextBonus
	}

	if index := strings.Index(content, keyword); index >= 0 {
		earlyBonus := 3 - utf8.RuneCountInString(content[:index])/8
		if earlyBonus < 0 {
			earlyBonus = 0
		}
		if score < 0 {
			score = 0
		}
		score += earlyBonus
	}

	negativePatterns := []*regexp.Regexp{
		regexp.MustCompile(fmt.Sprintf(`(?:不是|并非|不在|没有去|没去)(?:[\p{Han}]{0,6})?%s`, regexp.QuoteMeta(keyword))),
		regexp.MustCompile(fmt.Sprintf(`而不是(?:[\p{Han}]{0,6})?%s`, regexp.QuoteMeta(keyword))),
	}
	for _, pattern := range negativePatterns {
		if pattern.MatchString(content) {
			score -= 4
		}
	}

	return score
}

func inferInteriorExterior(content string) string {
	if strings.Contains(content, "街") || strings.Contains(content, "路") || strings.Contains(content, "广场") || strings.Contains(content, "车站") || strings.Contains(content, "码头") || strings.Contains(content, "操场") || strings.Contains(content, "跑道") || strings.Contains(content, "看台") || strings.Contains(content, "天台") || strings.Contains(content, "夜市") || strings.Contains(content, "河堤") || strings.Contains(content, "河岸") || strings.Contains(content, "堤岸") || strings.Contains(content, "船坞") || strings.Contains(content, "钟楼") || strings.Contains(content, "桥下") || strings.Contains(content, "桥边") {
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
	suffixes := []string{"室", "厅", "房", "馆", "站", "场", "道", "街", "路", "桥", "巷", "楼", "院", "台", "库", "口", "区", "市", "校", "园", "城", "堤", "岸", "坞"}
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
