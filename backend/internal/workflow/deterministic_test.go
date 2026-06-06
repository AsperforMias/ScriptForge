package workflow

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/AsperforMias/ScriptForge/backend/internal/ingest"
	"github.com/AsperforMias/ScriptForge/backend/internal/job"
	"github.com/AsperforMias/ScriptForge/backend/internal/screenplay"
	"github.com/AsperforMias/ScriptForge/backend/internal/testutil"
)

func TestBuildOutlineProducesConflictPerChapter(t *testing.T) {
	source := normalizeFixtureSource()

	outline := BuildOutline(source)
	if len(outline.Chapters) != 3 {
		t.Fatalf("expected 3 outline chapters, got %d", len(outline.Chapters))
	}

	if got := outline.Chapters[0].Conflict; got != "主角意识到私人空间可能已经被入侵，必须先判断危险是否仍在现场。" {
		t.Fatalf("unexpected chapter 1 conflict: %s", got)
	}
	if got := outline.Chapters[1].Conflict; got != "匿名警告把模糊的不安变成了明确威胁，主角必须判断这是不是针对她的布局。" {
		t.Fatalf("unexpected chapter 2 conflict: %s", got)
	}
	if got := outline.Chapters[2].Conflict; got != "主角决定主动追查线索，把被动戒备转化为现实行动。" {
		t.Fatalf("unexpected chapter 3 conflict: %s", got)
	}
}

func TestSummarizeUsesChineseFallbackCopy(t *testing.T) {
	if got := summarize(""); got != "这一章带出了新的戏剧变化。" {
		t.Fatalf("expected chinese fallback summary, got %s", got)
	}
}

func TestExtractEntitiesInfersProtagonistAndLocations(t *testing.T) {
	source := normalizeFixtureSource()

	entities := ExtractEntities(source)
	if len(entities.Characters) != 1 {
		t.Fatalf("expected 1 character, got %d", len(entities.Characters))
	}
	if entities.Characters[0].Name != "林琪" {
		t.Fatalf("expected protagonist 林琪, got %s", entities.Characters[0].Name)
	}
	if len(entities.Locations) != 3 {
		t.Fatalf("expected 3 locations, got %d", len(entities.Locations))
	}

	expectedLocations := []string{"公寓", "房间", "车站"}
	for idx, location := range entities.Locations {
		if location.Name != expectedLocations[idx] {
			t.Fatalf("expected location %s, got %s", expectedLocations[idx], location.Name)
		}
	}
}

func TestBuildScenePlanProducesMeaningfulObjectivesAndQuestions(t *testing.T) {
	source := normalizeFixtureSource()
	outline := BuildOutline(source)
	entities := ExtractEntities(source)

	plan := BuildScenePlan(source, outline, entities)
	if len(plan.Scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(plan.Scenes))
	}

	if got := plan.Scenes[0].Objective; got != "确认门锁异常是否意味着有人闯入，并判断主角能否立刻进入现场。" {
		t.Fatalf("unexpected scene 1 objective: %s", got)
	}
	if got := plan.Scenes[1].Beats[1].Content; got != "这张字条不是恶作剧，对方知道我今晚会回来。" {
		t.Fatalf("unexpected scene 2 dialogue: %s", got)
	}
	if got := plan.Scenes[2].Objective; got != "顺着字条和车站线索追查寄信人，把被动防备转成主动调查。" {
		t.Fatalf("unexpected scene 3 objective: %s", got)
	}
	if got := plan.Scenes[2].Notes.OpenQuestions[0]; got != "顺着这条车站线索，主角究竟会找到谁？" {
		t.Fatalf("unexpected scene 3 open question: %s", got)
	}
	if got := plan.Scenes[2].Beats[1].Emotion; got != "determined" {
		t.Fatalf("unexpected scene 3 emotion: %s", got)
	}
}

func TestBuildScenePlanSupportsWorkplaceScenario(t *testing.T) {
	source := normalizeWorkplaceSource()
	outline := BuildOutline(source)
	entities := ExtractEntities(source)

	plan := BuildScenePlan(source, outline, entities)
	if got := plan.Scenes[0].Objective; got != "确认是谁替换了关键数据，并决定汇报前要先止损还是直接揭穿。" {
		t.Fatalf("unexpected workplace objective: %s", got)
	}
	if got := plan.Scenes[0].Beats[1].Content; got != "如果现在不把问题找出来，明天整个项目都会失控。" {
		t.Fatalf("unexpected workplace dialogue: %s", got)
	}
	if got := plan.Scenes[0].Beats[1].Emotion; got != "focused" {
		t.Fatalf("unexpected workplace emotion: %s", got)
	}
	if got := plan.Scenes[1].Objective; got != "弄清团队猜疑是谁挑起的，并判断这次对质能否在汇报前止损。" {
		t.Fatalf("unexpected workplace scene 2 objective: %s", got)
	}
	if got := plan.Scenes[1].Notes.OpenQuestions[0]; got != "团队里的怀疑究竟是谁放出来的？" {
		t.Fatalf("unexpected workplace open question: %s", got)
	}
	if got := plan.Scenes[2].Objective; got != "在会议室摊牌前守住证据，让项目风险无法继续被掩盖。" {
		t.Fatalf("unexpected workplace scene 3 objective: %s", got)
	}
}

func TestBuildScenePlanSupportsSportsScenario(t *testing.T) {
	source := normalizeSportsSource()
	outline := BuildOutline(source)
	entities := ExtractEntities(source)

	plan := BuildScenePlan(source, outline, entities)
	if got := plan.Scenes[0].Slugline.InteriorExterior; got != "EXT" {
		t.Fatalf("unexpected sports scene 1 int/ext: %s", got)
	}
	if got := plan.Scenes[1].Slugline.Time; got != "MORNING" {
		t.Fatalf("unexpected sports scene 2 time: %s", got)
	}
	if got := plan.Scenes[2].Objective; got != "带着现有阵容完成比赛，并把临场压力转成真正的起跑动作。" {
		t.Fatalf("unexpected sports objective: %s", got)
	}
	if got := plan.Scenes[2].Beats[1].Content; got != "就算少一个人，我们也得把这场接力跑完。" {
		t.Fatalf("unexpected sports dialogue: %s", got)
	}
}

func TestBuildScenePlanSupportsFamilyScenario(t *testing.T) {
	source := normalizeFamilySource()
	outline := BuildOutline(source)
	entities := ExtractEntities(source)

	plan := BuildScenePlan(source, outline, entities)
	if got := plan.Scenes[0].Slugline.LocationID; got != "loc_chapter_01" {
		t.Fatalf("unexpected family scene 1 location id: %s", got)
	}
	if got := plan.Scenes[0].Objective; got != "确认父亲坚持回家的代价，并在家人与医生建议之间做出选择。" {
		t.Fatalf("unexpected family objective: %s", got)
	}
	if got := plan.Scenes[1].Objective; got != "接住厨房里的旧账与指责，并逼近这场家庭争执真正的症结。" {
		t.Fatalf("unexpected family scene 2 objective: %s", got)
	}
	if got := plan.Scenes[1].Beats[1].Content; got != "今晚这顿饭不是为了热闹，是为了把这些年的话说清楚。" {
		t.Fatalf("unexpected family dialogue: %s", got)
	}
	if got := plan.Scenes[2].Notes.OpenQuestions[0]; got != "这次客厅里的坦白，能不能真的让一家人把误会说开？" {
		t.Fatalf("unexpected family open question: %s", got)
	}
}

func TestBuildScenePlanSupportsComedyScenario(t *testing.T) {
	source := normalizeComedySource()
	outline := BuildOutline(source)
	entities := ExtractEntities(source)

	plan := BuildScenePlan(source, outline, entities)
	if got := plan.Scenes[0].Slugline.InteriorExterior; got != "EXT" {
		t.Fatalf("unexpected comedy scene 1 int/ext: %s", got)
	}
	if got := plan.Scenes[1].Beats[1].Emotion; got != "awkward" {
		t.Fatalf("unexpected comedy emotion: %s", got)
	}
	if got := plan.Scenes[0].Objective; got != "先止住夜市里的失控误会，再判断这次撞见会不会变成新的合作。" {
		t.Fatalf("unexpected comedy scene 1 objective: %s", got)
	}
	if got := plan.Scenes[1].Objective; got != "在餐馆里把误会说清楚，并避免朋友起哄继续推高尴尬。" {
		t.Fatalf("unexpected comedy scene 2 objective: %s", got)
	}
	if got := plan.Scenes[2].Objective; got != "把前两章的尴尬转成一次真正能落地的合作试播。" {
		t.Fatalf("unexpected comedy objective: %s", got)
	}
	if got := plan.Scenes[2].Notes.OpenQuestions[0]; got != "这次广场试播能不能把之前的尴尬真的翻篇？" {
		t.Fatalf("unexpected comedy open question: %s", got)
	}
}

func TestBuildScenePlanFallsBackForWeakEntitiesAndSparseSignals(t *testing.T) {
	source := normalizeSparseCustomSource()
	outline := BuildOutline(source)
	entities := ExtractEntities(source)
	plan := BuildScenePlan(source, outline, entities)

	if entities.Characters[0].Name != "主角" {
		t.Fatalf("expected weak entity fallback protagonist 主角, got %s", entities.Characters[0].Name)
	}
	for idx, location := range entities.Locations {
		if location.Name == "" {
			t.Fatalf("expected location name for scene %d", idx+1)
		}
		if location.Name == "Chapter 1 Main Location" || location.Name == "Chapter 2 Main Location" || location.Name == "Chapter 3 Main Location" {
			t.Fatalf("expected localized location fallback, got %s", location.Name)
		}
	}

	objectives := map[string]struct{}{}
	openQuestions := map[string]struct{}{}
	for idx, scene := range plan.Scenes {
		if scene.Objective == "" {
			t.Fatalf("expected objective for scene %d", idx+1)
		}
		objectives[scene.Objective] = struct{}{}
		if len(scene.Notes.OpenQuestions) == 0 {
			t.Fatalf("expected open question for scene %d", idx+1)
		}
		openQuestions[scene.Notes.OpenQuestions[0]] = struct{}{}
	}
	if len(objectives) != len(plan.Scenes) {
		t.Fatalf("expected unique objectives, got %d unique for %d scenes", len(objectives), len(plan.Scenes))
	}
	if len(openQuestions) != len(plan.Scenes) {
		t.Fatalf("expected unique open questions, got %d unique for %d scenes", len(openQuestions), len(plan.Scenes))
	}
}

func TestBuildDocumentMarksFailedWhenDeterministicConfidenceStaysLow(t *testing.T) {
	var req job.CreateJobRequest
	req.Source.Title = "Archive Echo"
	req.Source.Author = "Test Author"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "Chapter 1", Content: "The recorder clicks on in an empty archive room. A strange laugh leaks through the tape, but no one is named and the narrator only describes the sound."},
		{Index: 2, Title: "Chapter 2", Content: "The next afternoon, another anonymous note appears beside the machine. The narrator compares handwriting and keeps everything vague."},
		{Index: 3, Title: "Chapter 3", Content: "At dusk, the narrator reaches an old bell tower and finds only a key left behind. The voice on the tape is still unexplained."},
	}
	req.Adaptation.Style = "Suspense short drama"
	req.Generation.Mode = "deterministic"
	source := ingest.Normalize(req)
	outline := BuildOutline(source)
	entities := ExtractEntities(source)
	plan := BuildScenePlan(source, outline, entities)
	doc := BuildDocument(req, source, outline, entities, plan)

	if err := screenplay.Validate(doc); err != nil {
		t.Fatalf("expected structurally valid document: %v", err)
	}
	if doc.Characters[0].Name != "主角" {
		t.Fatalf("expected fallback protagonist 主角, got %s", doc.Characters[0].Name)
	}
	if doc.Validation.Status != "failed" {
		t.Fatalf("expected low-confidence deterministic result to fail validation status, got %s", doc.Validation.Status)
	}
}

func TestBuildScenePlanPrefersExplicitSceneEvidenceForCustomSuspenseInput(t *testing.T) {
	source := normalizeCustomSuspenseSource()
	outline := BuildOutline(source)
	entities := ExtractEntities(source)
	plan := BuildScenePlan(source, outline, entities)

	if got := plan.Scenes[0].Slugline.LocationID; got != "loc_chapter_01" {
		t.Fatalf("unexpected scene 1 location id: %s", got)
	}
	if got := entities.Locations[0].Name; got != "客厅" {
		t.Fatalf("expected scene 1 location 客厅, got %s", got)
	}
	if got := plan.Scenes[0].Objective; got == "" || !containsAny(got, "录音") {
		t.Fatalf("expected scene 1 objective to stay on recording clue, got %s", got)
	}
	if got := plan.Scenes[0].Objective; containsAny(got, "团圆饭", "误会说开", "和解") {
		t.Fatalf("expected scene 1 objective to avoid family template leakage, got %s", got)
	}
	if got := plan.Scenes[0].Beats[1].Content; !containsAny(got, "录音") {
		t.Fatalf("expected scene 1 dialogue to mention recording clue, got %s", got)
	}

	if got := entities.Locations[1].Name; got != "河堤" {
		t.Fatalf("expected scene 2 location 河堤, got %s", got)
	}
	if got := plan.Scenes[1].Objective; containsAny(got, "车站", "寄信人") {
		t.Fatalf("expected scene 2 objective to avoid station template, got %s", got)
	}
	if got := plan.Scenes[1].Beats[1].Content; containsAny(got, "车站", "寄信人") {
		t.Fatalf("expected scene 2 dialogue to avoid station template, got %s", got)
	}
	if got := plan.Scenes[1].Notes.OpenQuestions[0]; containsAny(got, "车站", "寄信人") {
		t.Fatalf("expected scene 2 open question to avoid station template, got %s", got)
	}

	if got := entities.Locations[2].Name; got != "船坞" {
		t.Fatalf("expected scene 3 location 船坞, got %s", got)
	}
	if got := plan.Scenes[2].Objective; got == "" || !containsAny(got, "钥匙", "打开") {
		t.Fatalf("expected scene 3 objective to stay on key/lock action, got %s", got)
	}
	if got := plan.Scenes[2].Notes.OpenQuestions[0]; !containsAny(got, "钥匙", "门") {
		t.Fatalf("expected scene 3 open question to stay on key/door clue, got %s", got)
	}
}

func TestDeterministicWorkflowKeepsSuspenseEvidenceAheadOfFamilyKeywords(t *testing.T) {
	source := normalizeFamilyWordSuspenseSource()
	outline := BuildOutline(source)
	entities := ExtractEntities(source)
	plan := BuildScenePlan(source, outline, entities)

	if len(plan.Scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(plan.Scenes))
	}
	if got := outline.Chapters[0].Conflict; !containsAny(got, "录音", "声音") {
		t.Fatalf("expected chapter 1 conflict to stay on recording clue, got %s", got)
	}
	if got := outline.Chapters[0].Conflict; containsAny(got, "团圆饭", "误会说开", "家人把话说开") {
		t.Fatalf("expected chapter 1 conflict to avoid family template leakage, got %s", got)
	}
	if got := plan.Scenes[0].Objective; !containsAny(got, "录音", "声音") {
		t.Fatalf("expected scene 1 objective to stay on recording clue, got %s", got)
	}
	if got := plan.Scenes[1].Objective; !containsAny(got, "编号", "钥匙") {
		t.Fatalf("expected scene 2 objective to stay on current clue, got %s", got)
	}
	if got := plan.Scenes[1].Notes.OpenQuestions[0]; containsAny(got, "团圆饭", "和解", "误会说开") {
		t.Fatalf("expected scene 2 open question to avoid family template leakage, got %s", got)
	}
	if got := plan.Scenes[2].Objective; !containsAny(got, "钥匙", "打开", "仓库") {
		t.Fatalf("expected scene 3 objective to stay on warehouse/key action, got %s", got)
	}

	expectedNames := []string{"闻溪", "郑岚", "老秦"}
	for _, expectedName := range expectedNames {
		if !characterNamesContain(entities, expectedName) {
			t.Fatalf("expected extracted characters to include %s, got %#v", expectedName, entities.Characters)
		}
	}
}

func TestDeterministicWorkflowHardensCustomGrowthFantasyInput(t *testing.T) {
	source := normalizeGrowthFantasySource()
	outline := BuildOutline(source)
	entities := ExtractEntities(source)
	plan := BuildScenePlan(source, outline, entities)
	var req job.CreateJobRequest
	req.Source.Title = source.Title
	req.Source.Author = source.Author
	req.Adaptation.Style = "异世界转生 / 贵族成长"
	req.Adaptation.Audience = "青年向"
	req.Generation.Mode = "deterministic"
	doc := BuildDocument(req, source, outline, entities, plan)

	for _, badName := range []string{"所以", "现在", "更别"} {
		if characterNamesContain(entities, badName) {
			t.Fatalf("expected filtered fragment %s to stay out of characters, got %#v", badName, entities.Characters)
		}
	}
	for _, expectedName := range []string{"艾琳", "罗莎", "维恩"} {
		if !characterNamesContain(entities, expectedName) {
			t.Fatalf("expected extracted characters to include %s, got %#v", expectedName, entities.Characters)
		}
	}

	if len(plan.Scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(plan.Scenes))
	}
	for idx, scene := range plan.Scenes {
		if len(scene.Beats) < 3 {
			t.Fatalf("expected 3 beats for scene %d, got %d", idx+1, len(scene.Beats))
		}
		if scene.Beats[0].Content == scene.Summary || strings.Contains(scene.Beats[0].Content, "...") {
			t.Fatalf("expected scene %d opening beat to be a concrete action beat, got %s", idx+1, scene.Beats[0].Content)
		}
		if containsAny(scene.Objective, "录音", "匿名", "字条", "门锁", "车站", "寄信人") {
			t.Fatalf("expected scene %d objective to avoid suspense leakage, got %s", idx+1, scene.Objective)
		}
		if containsAny(scene.Beats[len(scene.Beats)-1].Content, "录音", "匿名", "字条", "门锁", "车站", "寄信人") {
			t.Fatalf("expected scene %d dialogue to avoid suspense leakage, got %s", idx+1, scene.Beats[len(scene.Beats)-1].Content)
		}
		if len(scene.Notes.OpenQuestions) == 0 {
			t.Fatalf("expected open question for scene %d", idx+1)
		}
		if containsAny(scene.Notes.OpenQuestions[0], "录音", "匿名", "字条", "门锁", "车站", "寄信人") {
			t.Fatalf("expected scene %d open question to avoid suspense leakage, got %s", idx+1, scene.Notes.OpenQuestions[0])
		}
	}

	if len(doc.Validation.Warnings) == 0 {
		t.Fatal("expected growth-fantasy input to surface validation warnings")
	}
}

func TestDeterministicWorkflowHardensRealGrowthFantasyInput(t *testing.T) {
	req := testutil.GrowthFantasyRealInputRequest()
	source := ingest.Normalize(req)
	outline := BuildOutline(source)
	entities := ExtractEntities(source)
	plan := BuildScenePlan(source, outline, entities)
	doc := BuildDocument(req, source, outline, entities, plan)

	if err := screenplay.Validate(doc); err != nil {
		t.Fatalf("expected valid screenplay document: %v", err)
	}
	if len(doc.Scenes) <= len(doc.Source.Chapters) {
		t.Fatalf("expected at least one chapter to split into multiple scenes, got %d scenes for %d chapters", len(doc.Scenes), len(doc.Source.Chapters))
	}

	for _, badName := range testutil.GrowthFantasyRealInputForbiddenFragments() {
		if characterNamesContain(entities, badName) {
			t.Fatalf("expected fragment-like candidate %s to stay out of characters, got %#v", badName, entities.Characters)
		}
	}

	if got := doc.Characters[0].Name; got != "厄洛斯" {
		t.Fatalf("expected protagonist 厄洛斯 for real growth-fantasy input, got %s", got)
	}
	if matched := countExpectedCharacterMatches(entities, testutil.GrowthFantasyRealInputExpectedNames()); matched < 3 {
		t.Fatalf("expected real growth-fantasy input to recover core names, matched %d in %#v", matched, entities.Characters)
	}
	if countScenesForChapter(doc, 2) < 2 || countScenesForChapter(doc, 3) < 2 {
		t.Fatalf("expected chapter 2 and 3 to split into multiple scenes, got chapter2=%d chapter3=%d", countScenesForChapter(doc, 2), countScenesForChapter(doc, 3))
	}
	if !documentHasLocation(doc, "藏书馆") || !documentHasLocation(doc, "花坛") || !documentHasLocation(doc, "账房") || !documentHasLocation(doc, "议事厅") {
		t.Fatalf("expected split scenes to preserve concrete locations, got %#v", doc.Locations)
	}
	if doc.Validation.Status != "passed" {
		t.Fatalf("expected credible growth-fantasy regression to stay structurally passed, got %s", doc.Validation.Status)
	}

	for idx, scene := range doc.Scenes {
		if strings.TrimSpace(scene.Objective) == "" {
			t.Fatalf("expected objective for scene %d", idx+1)
		}
		if looksLikeNarrativeCarryover(scene.Objective) {
			t.Fatalf("expected compact scene objective for scene %d, got %s", idx+1, scene.Objective)
		}
		if len(scene.Notes.OpenQuestions) == 0 {
			t.Fatalf("expected open question for scene %d", idx+1)
		}
		if looksLikeNarrativeCarryover(scene.Notes.OpenQuestions[0]) {
			t.Fatalf("expected compact open question for scene %d, got %s", idx+1, scene.Notes.OpenQuestions[0])
		}

		actionBeatCount := 0
		for _, beat := range scene.Beats {
			if beat.Type == "dialogue" && looksLikeNarrativeCarryover(beat.Content) {
				t.Fatalf("expected compact dialogue for scene %d, got %s", idx+1, beat.Content)
			}
			if beat.Type != "action" {
				continue
			}
			actionBeatCount++
			if beat.Content == scene.Summary || looksLikeBrokenActionBeat(beat.Content) {
				t.Fatalf("expected scene %d action beat to stay shootable, got %s", idx+1, beat.Content)
			}
		}
		if actionBeatCount == 0 {
			t.Fatalf("expected at least one action beat for scene %d", idx+1)
		}
	}

	if len(doc.Validation.Warnings) == 0 {
		t.Fatal("expected real growth-fantasy input to surface concrete validation warnings")
	}
	if hasGenericGrowthWarning(doc.Validation.Warnings) {
		t.Fatalf("expected warnings to move beyond generic growth copy, got %#v", doc.Validation.Warnings)
	}
	if !hasSpecificWarning(doc.Validation.Warnings, "characters:", "fragment") &&
		!hasSpecificWarning(doc.Validation.Warnings, "protagonist confidence") &&
		!hasSpecificWarning(doc.Validation.Warnings, "objective is still derived") &&
		!hasSpecificWarning(doc.Validation.Warnings, "beat adaptation remains") &&
		!hasSpecificWarning(doc.Validation.Warnings, "location/slugline confidence is low") {
		t.Fatalf("expected specific low-confidence warnings, got %#v", doc.Validation.Warnings)
	}
}

func normalizeFixtureSource() ingest.NormalizedSource {
	var req job.CreateJobRequest
	req.Source.Title = "夜雨疑云"
	req.Source.Author = "示例作者"
	req.Adaptation.Style = "悬疑短剧"
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "第一章 深夜回家", Content: "林琪深夜回到公寓，发现门锁似乎被人动过。她停在走廊里，不确定是否应该立刻进去。"},
		{Index: 2, Title: "第二章 陌生字条", Content: "她在房间里找到一张陌生字条，上面只写着今晚别睡。林琪意识到有人提前进入过房间。"},
		{Index: 3, Title: "第三章 清晨追踪", Content: "第二天清晨，林琪带着字条前往车站，试图顺着纸上的线索找到寄信人。"},
	}
	return ingest.Normalize(req)
}

func normalizeWorkplaceSource() ingest.NormalizedSource {
	var req job.CreateJobRequest
	req.Source.Title = "交稿前夜"
	req.Source.Author = "示例作者"
	req.Adaptation.Style = "职场短剧"
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "第一章 数据被换", Content: "苏禾深夜留在办公室复核提案，发现明早汇报用的数据被人替换。她意识到项目组里有人提前动了最终版本。"},
		{Index: 2, Title: "第二章 咖啡馆对质", Content: "她约同组同事在咖啡馆见面，对方却反问她是不是想独占客户。苏禾意识到怀疑已经在团队里扩散。"},
		{Index: 3, Title: "第三章 会议室摊牌", Content: "第二天清晨，苏禾带着备份文件走进会议室，决定在正式汇报前把问题摆到台面上。"},
	}
	return ingest.Normalize(req)
}

func normalizeSportsSource() ingest.NormalizedSource {
	var req job.CreateJobRequest
	req.Source.Title = "最后一棒"
	req.Source.Author = "示例作者"
	req.Adaptation.Style = "青春运动短剧"
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "第一章 操场加练", Content: "周宁晚上独自在操场加练接力，教练突然通知主力队友可能缺席决赛。她第一次意识到最后一棒会落到自己手里。"},
		{Index: 2, Title: "第二章 教室争执", Content: "第二天一早，她在教室里听见替补队友质疑战术安排，队伍差点在比赛前先吵散。周宁只能临时站出来稳住大家。"},
		{Index: 3, Title: "第三章 跑道起跑", Content: "比赛当天清晨，周宁站上跑道，决定不再等待主力归队，而是带着现有阵容把接力跑完。"},
	}
	return ingest.Normalize(req)
}

func normalizeFamilySource() ingest.NormalizedSource {
	var req job.CreateJobRequest
	req.Source.Title = "回家吃饭"
	req.Source.Author = "示例作者"
	req.Adaptation.Style = "家庭情感短剧"
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "第一章 病房电话", Content: "程安在医院病房外接到母亲电话，得知父亲坚持出院回家吃团圆饭。她必须在家人与医生建议之间做决定。"},
		{Index: 2, Title: "第二章 厨房争执", Content: "傍晚，程安回到家里厨房准备晚饭，姐姐却指责她总拿工作当借口，家里的旧账被重新翻出来。"},
		{Index: 3, Title: "第三章 客厅和解", Content: "夜里，父亲坐在客厅里主动提起多年前的误会，程安终于决定把压在心里的话说出口。"},
	}
	return ingest.Normalize(req)
}

func normalizeComedySource() ingest.NormalizedSource {
	var req job.CreateJobRequest
	req.Source.Title = "误会直播间"
	req.Source.Author = "示例作者"
	req.Adaptation.Style = "都市轻喜剧"
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "第一章 夜市撞见", Content: "许言在夜市帮朋友看摊时，误把来取设备的摄影师当成竞争对手，当场闹出笑话。"},
		{Index: 2, Title: "第二章 餐馆圆场", Content: "第二天中午，两人在餐馆碰面试图解释误会，却因为朋友临时起哄把场面越描越乱。"},
		{Index: 3, Title: "第三章 广场开播", Content: "傍晚，他们决定在广场一起试播，把之前的误会变成一次意外成功的直播。"},
	}
	return ingest.Normalize(req)
}

func normalizeSparseCustomSource() ingest.NormalizedSource {
	var req job.CreateJobRequest
	req.Source.Title = "雾港录音带"
	req.Source.Author = "自定义作者"
	req.Adaptation.Style = "悬疑现实短剧"
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "第一章 录音失真", Content: "暴雨落了一整夜，旧录音里突然多出一段陌生笑声。叙述者不敢立刻重播，只能先把磁带锁进抽屉。"},
		{Index: 2, Title: "第二章 匿名留言", Content: "第二天下午，留言板上多出一行约见时间，没人承认写过它。叙述者决定先核对录音来源，再去找留下字的人。"},
		{Index: 3, Title: "第三章 钟楼扑空", Content: "傍晚，叙述者带着磁带赶到老城区的旧钟楼，却发现约见人已经提前离开，只留下一把钥匙。"},
	}
	return ingest.Normalize(req)
}

func normalizeCustomSuspenseSource() ingest.NormalizedSource {
	var req job.CreateJobRequest
	req.Source.Title = "回潮暗线"
	req.Source.Author = "自定义作者"
	req.Adaptation.Style = "悬疑现实短剧"
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "第一章 旧客厅录音", Content: "沈砚回到父亲留下的旧客厅整理遗物，听见录音机里多出一段夹着潮声的陌生对话。她不敢惊动家里其他人，只想先确认那段录音是不是被人动过。"},
		{Index: 2, Title: "第二章 河堤碰面", Content: "第二天傍晚，沈砚按匿名留言赶到河堤，发现纸条提到的线索指向废弃船坞，而不是任何车站。她决定先弄清是谁把钥匙塞进自己口袋，再判断这场约见是不是圈套。"},
		{Index: 3, Title: "第三章 船坞试锁", Content: "夜里，沈砚独自走进废弃船坞，用那把生锈钥匙去试库房侧门的锁孔。门后传来的撞击声让她意识到，真正想藏起来的东西还在里面。"},
	}
	return ingest.Normalize(req)
}

func normalizeFamilyWordSuspenseSource() ingest.NormalizedSource {
	var req job.CreateJobRequest
	req.Source.Title = "旧宅回声"
	req.Source.Author = "自定义作者"
	req.Adaptation.Style = "悬疑现实短剧"
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "第一章 客厅回放", Content: "闻溪回到父亲留下的家里，在旧客厅收拾遗物时听见随身听里多出一段陌生口哨。郑岚在里屋催她先吃饭，但闻溪只想先把录音倒回去，确认那段声音究竟录自哪一天。"},
		{Index: 2, Title: "第二章 楼道纸灰", Content: "第二天傍晚，闻溪在自家楼道发现烧过的纸灰和一张写着仓库编号的便签。老秦说父亲生前把备用钥匙交给过一个陌生快递员，闻溪决定先去核对编号，再查钥匙落到了谁手里。"},
		{Index: 3, Title: "第三章 仓库试锁", Content: "夜里，闻溪赶到江边旧仓库，用找到的钥匙去试开侧门。门内传来的拖拽声让她意识到，有人正赶在她之前转移父亲留下的箱子。"},
	}
	return ingest.Normalize(req)
}

func normalizeGrowthFantasySource() ingest.NormalizedSource {
	var req job.CreateJobRequest
	req.Source.Title = "转生北境"
	req.Source.Author = "自定义作者"
	req.Adaptation.Style = "异世界转生 / 贵族成长"
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "第一章 贵族次女接手北境", Content: "艾琳在伯爵府邸的账房里听完遗产分配，意识到自己这个长期被轻视的次女突然要接手最穷的北境领地。她没有争辩，只先把旧地图和欠税名册摊开，决定今晚就看清领地到底烂到什么程度。"},
		{Index: 2, Title: "第二章 巡视破败庄园", Content: "第二天清晨，艾琳带着侍女罗莎和见习骑士维恩巡视北境庄园，发现粮仓、围墙和灌渠都比账面更糟。维恩主张先裁掉无用雇工，艾琳却决定先稳住领民和春播，再去查是谁把亏空一路压到王都审计前。"},
		{Index: 3, Title: "第三章 议事厅定下新秩序", Content: "傍晚，艾琳在破旧议事厅召集管家、农务官和商会代表，准备公布新的税期与修渠顺序。罗莎提醒她，一旦让贵族亲族知道北境还能救活，原本等着看笑话的人就会立刻回来争功。"},
	}
	return ingest.Normalize(req)
}

func characterNamesContain(entities EntityBundle, expectedName string) bool {
	for _, character := range entities.Characters {
		if character.Name == expectedName {
			return true
		}
	}
	return false
}

func countExpectedCharacterMatches(entities EntityBundle, expected []string) int {
	matched := 0
	for _, name := range expected {
		if characterNamesContain(entities, name) {
			matched++
		}
	}
	return matched
}

func countScenesForChapter(doc screenplay.Document, chapterIndex int) int {
	count := 0
	for _, scene := range doc.Scenes {
		for _, sourceChapter := range scene.SourceChapters {
			if sourceChapter == chapterIndex {
				count++
				break
			}
		}
	}
	return count
}

func documentHasLocation(doc screenplay.Document, expected string) bool {
	for _, location := range doc.Locations {
		if location.Name == expected {
			return true
		}
	}
	return false
}

func looksLikeNarrativeCarryover(input string) bool {
	input = strings.TrimSpace(input)
	if input == "" {
		return true
	}
	if utf8.RuneCountInString(input) > 40 {
		return true
	}
	return containsAny(
		input,
		"因为", "随着", "直到", "只能", "所能看到", "难不成", "让人感觉", "也不知道", "这让他", "原本", "……", "...",
	)
}

func looksLikeBrokenActionBeat(input string) bool {
	input = strings.TrimSpace(input)
	if input == "" {
		return true
	}
	return containsAny(input, "【", "】", "...", "……") ||
		strings.HasPrefix(input, "“") ||
		strings.HasPrefix(input, "\"")
}

func hasGenericGrowthWarning(warnings []string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, "当前按通用成长/世界观场景规则生成") {
			return true
		}
	}
	return false
}

func hasSpecificWarning(warnings []string, keywords ...string) bool {
	for _, warning := range warnings {
		matched := true
		for _, keyword := range keywords {
			if !strings.Contains(warning, keyword) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}
