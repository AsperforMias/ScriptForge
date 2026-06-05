package workflow

import (
	"testing"

	"github.com/AsperforMias/ScriptForge/backend/internal/ingest"
	"github.com/AsperforMias/ScriptForge/backend/internal/job"
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

	if got := plan.Scenes[0].Objective; got != "确认房间是否已经失守，并决定主角该撤离还是进入现场。" {
		t.Fatalf("unexpected scene 1 objective: %s", got)
	}
	if got := plan.Scenes[1].Beats[1].Content; got != "这张字条不是恶作剧，对方知道我今晚会回来。" {
		t.Fatalf("unexpected scene 2 dialogue: %s", got)
	}
	if got := plan.Scenes[2].Notes.OpenQuestions[0]; got != "车站线索会把主角引向谁？" {
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
	if got := plan.Scenes[0].Objective; got != "在正式汇报前确认项目风险，并决定是内部止损还是当场摊牌。" {
		t.Fatalf("unexpected workplace objective: %s", got)
	}
	if got := plan.Scenes[0].Beats[1].Content; got != "如果现在不把问题找出来，明天整个项目都会失控。" {
		t.Fatalf("unexpected workplace dialogue: %s", got)
	}
	if got := plan.Scenes[0].Beats[1].Emotion; got != "focused" {
		t.Fatalf("unexpected workplace emotion: %s", got)
	}
	if got := plan.Scenes[1].Notes.OpenQuestions[0]; got != "是谁在关键节点动了项目数据？" {
		t.Fatalf("unexpected workplace open question: %s", got)
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
	if got := plan.Scenes[2].Objective; got != "在比赛开始前稳住队伍配合，并把临场压力转成可执行的战术。" {
		t.Fatalf("unexpected sports objective: %s", got)
	}
	if got := plan.Scenes[2].Beats[1].Content; got != "就算少一个人，我们也得把这场接力跑完。" {
		t.Fatalf("unexpected sports dialogue: %s", got)
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
