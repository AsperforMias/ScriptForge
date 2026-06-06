package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/AsperforMias/ScriptForge/backend/internal/job"
	"github.com/AsperforMias/ScriptForge/backend/internal/llm"
	"github.com/AsperforMias/ScriptForge/backend/internal/screenplay"
	"github.com/AsperforMias/ScriptForge/backend/internal/storage/artifact"
	"github.com/AsperforMias/ScriptForge/backend/internal/testutil"
	"github.com/AsperforMias/ScriptForge/backend/internal/workflow"
	"gopkg.in/yaml.v3"
)

func TestRunnerRunProducesArtifactsAndYAML(t *testing.T) {
	tmpDir := t.TempDir()
	store := artifact.New(tmpDir)
	runner := NewRunner(store, llm.NewUnavailableGenerator("deterministic mode does not use llm"))

	req := validCreateJobRequest()
	result, err := runner.Run(context.Background(), "job_test_runner", req)
	if err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}

	if result.YAMLText == "" {
		t.Fatal("expected yaml output")
	}
	if len(result.Document.Scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(result.Document.Scenes))
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "job_test_runner", "screenplay.yaml")); err != nil {
		t.Fatalf("expected screenplay artifact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "job_test_runner", "input.json")); err != nil {
		t.Fatalf("expected input artifact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "job_test_runner", "normalized_source.json")); err != nil {
		t.Fatalf("expected normalized source artifact: %v", err)
	}
}

func TestRunnerRunMatchesDeterministicFixture(t *testing.T) {
	testCases := []struct {
		name         string
		request      job.CreateJobRequest
		expectedPath string
		jobID        string
	}{
		{
			name:         "night rain suspense",
			request:      fixtureCreateJobRequest(),
			expectedPath: filepath.Join("..", "..", "..", "testdata", "expected", "night-rain.screenplay.yaml"),
			jobID:        "job_fixture_runner_night_rain",
		},
		{
			name:         "workplace crisis",
			request:      mustLoadFixtureRequest(t, "workplace-crisis-request.json"),
			expectedPath: filepath.Join("..", "..", "..", "testdata", "expected", "workplace-crisis.screenplay.yaml"),
			jobID:        "job_fixture_runner_workplace",
		},
		{
			name:         "campus relay",
			request:      mustLoadFixtureRequest(t, "campus-relay-request.json"),
			expectedPath: filepath.Join("..", "..", "..", "testdata", "expected", "campus-relay.screenplay.yaml"),
			jobID:        "job_fixture_runner_campus",
		},
		{
			name:         "family dinner",
			request:      mustLoadFixtureRequest(t, "family-dinner-request.json"),
			expectedPath: filepath.Join("..", "..", "..", "testdata", "expected", "family-dinner.screenplay.yaml"),
			jobID:        "job_fixture_runner_family",
		},
		{
			name:         "comedy live mixup",
			request:      mustLoadFixtureRequest(t, "comedy-live-mixup-request.json"),
			expectedPath: filepath.Join("..", "..", "..", "testdata", "expected", "comedy-live-mixup.screenplay.yaml"),
			jobID:        "job_fixture_runner_comedy",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			store := artifact.New(tmpDir)
			runner := NewRunner(store, llm.NewUnavailableGenerator("deterministic mode does not use llm"))

			result, err := runner.Run(context.Background(), tc.jobID, tc.request)
			if err != nil {
				t.Fatalf("unexpected run error: %v", err)
			}

			expectedYAML, err := os.ReadFile(tc.expectedPath)
			if err != nil {
				t.Fatalf("read expected fixture: %v", err)
			}

			if !yamlDocumentsEqual(result.YAMLText, string(expectedYAML)) {
				t.Fatalf("unexpected yaml output\nexpected:\n%s\n\ngot:\n%s", string(expectedYAML), result.YAMLText)
			}
		})
	}
}

func TestRunnerRunSupportsCustomChineseInputRegression(t *testing.T) {
	tmpDir := t.TempDir()
	store := artifact.New(tmpDir)
	runner := NewRunner(store, llm.NewUnavailableGenerator("deterministic mode does not use llm"))

	req := customChineseCreateJobRequest()
	result, err := runner.Run(context.Background(), "job_test_runner_custom_cn", req)
	if err != nil {
		t.Fatalf("unexpected custom run error: %v", err)
	}

	if err := screenplay.Validate(result.Document); err != nil {
		t.Fatalf("expected valid screenplay document: %v", err)
	}
	if result.Document.Characters[0].Name != "主角" {
		t.Fatalf("expected weak entity fallback protagonist name 主角, got %s", result.Document.Characters[0].Name)
	}
	if len(result.Document.Scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(result.Document.Scenes))
	}

	objectives := map[string]struct{}{}
	openQuestions := map[string]struct{}{}
	for idx, location := range result.Document.Locations {
		if strings.Contains(location.Name, "Chapter ") {
			t.Fatalf("expected localized location fallback for chapter %d, got %s", idx+1, location.Name)
		}
		if strings.TrimSpace(location.Name) == "" {
			t.Fatalf("expected non-empty location for chapter %d", idx+1)
		}
	}
	for idx, scene := range result.Document.Scenes {
		if strings.TrimSpace(scene.Objective) == "" {
			t.Fatalf("expected non-empty scene objective for chapter %d", idx+1)
		}
		objectives[scene.Objective] = struct{}{}
		if len(scene.Notes.OpenQuestions) == 0 {
			t.Fatalf("expected non-empty open questions for scene %d", idx+1)
		}
		openQuestions[scene.Notes.OpenQuestions[0]] = struct{}{}
	}
	if len(objectives) != len(result.Document.Scenes) {
		t.Fatalf("expected unique objectives across custom scenes, got %d unique for %d scenes", len(objectives), len(result.Document.Scenes))
	}
	if len(openQuestions) != len(result.Document.Scenes) {
		t.Fatalf("expected unique open questions across custom scenes, got %d unique for %d scenes", len(openQuestions), len(result.Document.Scenes))
	}

	var yamlDoc screenplay.Document
	if err := yaml.Unmarshal([]byte(result.YAMLText), &yamlDoc); err != nil {
		t.Fatalf("expected yaml output to be parseable: %v", err)
	}
	if !reflect.DeepEqual(yamlDoc, result.Document) {
		t.Fatal("expected yaml_text and in-memory document to describe the same screenplay")
	}
}

func TestRunnerRunSupportsRealisticCustomSuspenseRegression(t *testing.T) {
	tmpDir := t.TempDir()
	store := artifact.New(tmpDir)
	runner := NewRunner(store, llm.NewUnavailableGenerator("deterministic mode does not use llm"))

	req := customSuspenseCreateJobRequest()
	result, err := runner.Run(context.Background(), "job_test_runner_custom_suspense", req)
	if err != nil {
		t.Fatalf("unexpected suspense run error: %v", err)
	}

	if err := screenplay.Validate(result.Document); err != nil {
		t.Fatalf("expected valid screenplay document: %v", err)
	}
	assertCustomSuspenseDocument(t, result.Document)

	var yamlDoc screenplay.Document
	if err := yaml.Unmarshal([]byte(result.YAMLText), &yamlDoc); err != nil {
		t.Fatalf("expected yaml output to be parseable: %v", err)
	}
	if !reflect.DeepEqual(yamlDoc, result.Document) {
		t.Fatal("expected yaml_text and in-memory document to describe the same screenplay")
	}
}

func TestRunnerRunSupportsFamilyWordSuspenseRegression(t *testing.T) {
	tmpDir := t.TempDir()
	store := artifact.New(tmpDir)
	runner := NewRunner(store, llm.NewUnavailableGenerator("deterministic mode does not use llm"))

	req := familyWordSuspenseCreateJobRequest()
	result, err := runner.Run(context.Background(), "job_test_runner_family_word_suspense", req)
	if err != nil {
		t.Fatalf("unexpected suspense run error: %v", err)
	}

	if err := screenplay.Validate(result.Document); err != nil {
		t.Fatalf("expected valid screenplay document: %v", err)
	}
	assertFamilyWordSuspenseDocument(t, result.Document)

	var yamlDoc screenplay.Document
	if err := yaml.Unmarshal([]byte(result.YAMLText), &yamlDoc); err != nil {
		t.Fatalf("expected yaml output to be parseable: %v", err)
	}
	if !reflect.DeepEqual(yamlDoc, result.Document) {
		t.Fatal("expected yaml_text and in-memory document to describe the same screenplay")
	}
}

func TestRunnerRunSupportsGrowthFantasyRegression(t *testing.T) {
	tmpDir := t.TempDir()
	store := artifact.New(tmpDir)
	runner := NewRunner(store, llm.NewUnavailableGenerator("deterministic mode does not use llm"))

	req := growthFantasyCreateJobRequest()
	result, err := runner.Run(context.Background(), "job_test_runner_growth_fantasy", req)
	if err != nil {
		t.Fatalf("unexpected growth-fantasy run error: %v", err)
	}

	if err := screenplay.Validate(result.Document); err != nil {
		t.Fatalf("expected valid screenplay document: %v", err)
	}
	assertGrowthFantasyDocument(t, result.Document)

	var yamlDoc screenplay.Document
	if err := yaml.Unmarshal([]byte(result.YAMLText), &yamlDoc); err != nil {
		t.Fatalf("expected yaml output to be parseable: %v", err)
	}
	if !reflect.DeepEqual(yamlDoc, result.Document) {
		t.Fatal("expected yaml_text and in-memory document to describe the same screenplay")
	}
}

func TestRunnerRunSupportsRealGrowthFantasyRegression(t *testing.T) {
	tmpDir := t.TempDir()
	store := artifact.New(tmpDir)
	runner := NewRunner(store, llm.NewUnavailableGenerator("deterministic mode does not use llm"))

	req := testutil.GrowthFantasyRealInputRequest()
	result, err := runner.Run(context.Background(), "job_test_runner_real_growth_fantasy", req)
	if err != nil {
		t.Fatalf("unexpected real growth-fantasy run error: %v", err)
	}

	if err := screenplay.Validate(result.Document); err != nil {
		t.Fatalf("expected valid screenplay document: %v", err)
	}
	assertRealGrowthFantasyDocument(t, result.Document)

	var yamlDoc screenplay.Document
	if err := yaml.Unmarshal([]byte(result.YAMLText), &yamlDoc); err != nil {
		t.Fatalf("expected yaml output to be parseable: %v", err)
	}
	if !reflect.DeepEqual(yamlDoc, result.Document) {
		t.Fatal("expected yaml_text and in-memory document to describe the same screenplay")
	}
}

func mustLoadFixtureRequest(t *testing.T, name string) job.CreateJobRequest {
	t.Helper()

	path := filepath.Join("..", "..", "..", "testdata", "novels", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read request fixture %s: %v", name, err)
	}

	var req job.CreateJobRequest
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("unmarshal request fixture %s: %v", name, err)
	}
	return req
}

func TestRunnerRunSupportsMockLLMMode(t *testing.T) {
	tmpDir := t.TempDir()
	store := artifact.New(tmpDir)
	runner := NewRunner(store, llm.NewMockGenerator())

	req := validCreateJobRequest()
	req.Generation.Mode = "llm"

	result, err := runner.Run(context.Background(), "job_test_runner_llm", req)
	if err != nil {
		t.Fatalf("unexpected llm run error: %v", err)
	}
	if len(result.Document.Scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(result.Document.Scenes))
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected llm warnings")
	}
}

func TestRunnerRunPersistsProviderDebugSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	store := artifact.New(tmpDir)
	runner := NewRunner(store, debugGenerator{})

	req := validCreateJobRequest()
	req.Generation.Mode = "llm"

	result, err := runner.Run(context.Background(), "job_test_runner_llm_debug", req)
	if err != nil {
		t.Fatalf("unexpected llm run error: %v", err)
	}
	if result.ProviderDebugPath == "" {
		t.Fatal("expected provider debug path")
	}

	data, err := os.ReadFile(result.ProviderDebugPath)
	if err != nil {
		t.Fatalf("read provider debug artifact: %v", err)
	}

	var snapshot llm.DebugSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("unmarshal provider debug artifact: %v", err)
	}
	if snapshot.Provider != "debug-generator" {
		t.Fatalf("expected provider debug-generator, got %s", snapshot.Provider)
	}
	if snapshot.ParseMode != "canonical" {
		t.Fatalf("expected parse mode canonical, got %s", snapshot.ParseMode)
	}
}

func TestRunnerRunFailsWhenLLMProviderIsUnavailable(t *testing.T) {
	tmpDir := t.TempDir()
	store := artifact.New(tmpDir)
	runner := NewRunner(store, llm.NewUnavailableGenerator("provider not configured"))

	req := validCreateJobRequest()
	req.Generation.Mode = "llm"

	result, err := runner.Run(context.Background(), "job_test_runner_llm_disabled", req)
	if err == nil {
		t.Fatal("expected llm provider error")
	}
	if result.CurrentStage != "screenplay_generation" {
		t.Fatalf("expected screenplay_generation failure, got %s", result.CurrentStage)
	}
}

func validCreateJobRequest() job.CreateJobRequest {
	var req job.CreateJobRequest
	req.Source.Title = "Night Rain"
	req.Source.Author = "Demo Author"
	req.Adaptation.Style = "Suspense Drama"
	req.Adaptation.Audience = "General"
	req.Adaptation.Notes = []string{"Keep a strong hook in each scene"}
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "Chapter 1", Content: "林琪深夜回到公寓，发现门锁似乎被动过。"},
		{Index: 2, Title: "Chapter 2", Content: "她在房间里找到一张陌生字条，怀疑有人潜入。"},
		{Index: 3, Title: "Chapter 3", Content: "第二天清晨，她决定顺着线索前往车站。"},
	}
	return req
}

func fixtureCreateJobRequest() job.CreateJobRequest {
	var req job.CreateJobRequest
	req.Source.Title = "夜雨疑云"
	req.Source.Author = "示例作者"
	req.Adaptation.Style = "悬疑短剧"
	req.Adaptation.Audience = "大众向"
	req.Adaptation.Notes = []string{"强化悬疑氛围", "保留主角主动调查的动机"}
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "第一章 深夜回家", Content: "林琪深夜回到公寓，发现门锁似乎被人动过。她停在走廊里，不确定是否应该立刻进去。"},
		{Index: 2, Title: "第二章 陌生字条", Content: "她在房间里找到一张陌生字条，上面只写着今晚别睡。林琪意识到有人提前进入过房间。"},
		{Index: 3, Title: "第三章 清晨追踪", Content: "第二天清晨，林琪带着字条前往车站，试图顺着纸上的线索找到寄信人。"},
	}
	return req
}

func customChineseCreateJobRequest() job.CreateJobRequest {
	var req job.CreateJobRequest
	req.Source.Title = "雾港录音带"
	req.Source.Author = "自定义作者"
	req.Adaptation.Style = "悬疑现实短剧"
	req.Adaptation.Audience = "青年向"
	req.Adaptation.Notes = []string{"保留迟疑感", "突出线索递进"}
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "第一章 录音失真", Content: "暴雨落了一整夜，旧录音里突然多出一段陌生笑声。叙述者不敢立刻重播，只能先把磁带锁进抽屉。"},
		{Index: 2, Title: "第二章 匿名留言", Content: "第二天下午，留言板上多出一行约见时间，没人承认写过它。叙述者决定先核对录音来源，再去找留下字的人。"},
		{Index: 3, Title: "第三章 钟楼扑空", Content: "傍晚，叙述者带着磁带赶到老城区的旧钟楼，却发现约见人已经提前离开，只留下一把钥匙。"},
	}
	return req
}

func customSuspenseCreateJobRequest() job.CreateJobRequest {
	var req job.CreateJobRequest
	req.Source.Title = "回潮暗线"
	req.Source.Author = "自定义作者"
	req.Adaptation.Style = "悬疑现实短剧"
	req.Adaptation.Audience = "青年向"
	req.Adaptation.Notes = []string{"以当前章节证据驱动场景目标", "避免凭空补车站线索"}
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "第一章 旧客厅录音", Content: "沈砚回到父亲留下的旧客厅整理遗物，听见录音机里多出一段夹着潮声的陌生对话。她不敢惊动家里其他人，只想先确认那段录音是不是被人动过。"},
		{Index: 2, Title: "第二章 河堤碰面", Content: "第二天傍晚，沈砚按匿名留言赶到河堤，发现纸条提到的线索指向废弃船坞，而不是任何车站。她决定先弄清是谁把钥匙塞进自己口袋，再判断这场约见是不是圈套。"},
		{Index: 3, Title: "第三章 船坞试锁", Content: "夜里，沈砚独自走进废弃船坞，用那把生锈钥匙去试库房侧门的锁孔。门后传来的撞击声让她意识到，真正想藏起来的东西还在里面。"},
	}
	return req
}

func familyWordSuspenseCreateJobRequest() job.CreateJobRequest {
	var req job.CreateJobRequest
	req.Source.Title = "旧宅回声"
	req.Source.Author = "自定义作者"
	req.Adaptation.Style = "悬疑现实短剧"
	req.Adaptation.Audience = "青年向"
	req.Adaptation.Notes = []string{"家庭词不能盖过当前线索", "同章多线索时围绕主证据推进"}
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "第一章 客厅回放", Content: "闻溪回到父亲留下的家里，在旧客厅收拾遗物时听见随身听里多出一段陌生口哨。郑岚在里屋催她先吃饭，但闻溪只想先把录音倒回去，确认那段声音究竟录自哪一天。"},
		{Index: 2, Title: "第二章 楼道纸灰", Content: "第二天傍晚，闻溪在自家楼道发现烧过的纸灰和一张写着仓库编号的便签。老秦说父亲生前把备用钥匙交给过一个陌生快递员，闻溪决定先去核对编号，再查钥匙落到了谁手里。"},
		{Index: 3, Title: "第三章 仓库试锁", Content: "夜里，闻溪赶到江边旧仓库，用找到的钥匙去试开侧门。门内传来的拖拽声让她意识到，有人正赶在她之前转移父亲留下的箱子。"},
	}
	return req
}

func growthFantasyCreateJobRequest() job.CreateJobRequest {
	var req job.CreateJobRequest
	req.Source.Title = "转生北境"
	req.Source.Author = "自定义作者"
	req.Adaptation.Style = "异世界转生 / 贵族成长"
	req.Adaptation.Audience = "青年向"
	req.Adaptation.Notes = []string{"先把处境变化写实，再考虑题材扩展"}
	req.Generation.Mode = "deterministic"
	req.Source.Chapters = []job.ChapterBody{
		{Index: 1, Title: "第一章 贵族次女接手北境", Content: "艾琳在伯爵府邸的账房里听完遗产分配，意识到自己这个长期被轻视的次女突然要接手最穷的北境领地。她没有争辩，只先把旧地图和欠税名册摊开，决定今晚就看清领地到底烂到什么程度。"},
		{Index: 2, Title: "第二章 巡视破败庄园", Content: "第二天清晨，艾琳带着侍女罗莎和见习骑士维恩巡视北境庄园，发现粮仓、围墙和灌渠都比账面更糟。维恩主张先裁掉无用雇工，艾琳却决定先稳住领民和春播，再去查是谁把亏空一路压到王都审计前。"},
		{Index: 3, Title: "第三章 议事厅定下新秩序", Content: "傍晚，艾琳在破旧议事厅召集管家、农务官和商会代表，准备公布新的税期与修渠顺序。罗莎提醒她，一旦让贵族亲族知道北境还能救活，原本等着看笑话的人就会立刻回来争功。"},
	}
	return req
}

func assertCustomSuspenseDocument(t *testing.T, doc screenplay.Document) {
	t.Helper()

	if len(doc.Scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(doc.Scenes))
	}
	if len(doc.Locations) != 3 {
		t.Fatalf("expected 3 locations, got %d", len(doc.Locations))
	}

	expectedLocations := []string{"客厅", "河堤", "船坞"}
	for idx, location := range doc.Locations {
		if location.Name != expectedLocations[idx] {
			t.Fatalf("expected location %s for chapter %d, got %s", expectedLocations[idx], idx+1, location.Name)
		}
	}

	if got := doc.Scenes[0].Objective; !containsAnyText(got, "录音") {
		t.Fatalf("expected scene 1 objective to mention recording clue, got %s", got)
	}
	if got := doc.Scenes[0].Objective; containsAnyText(got, "团圆饭", "误会说开", "和解") {
		t.Fatalf("expected scene 1 objective to avoid family template leakage, got %s", got)
	}
	if got := doc.Scenes[0].Beats[1].Content; !containsAnyText(got, "录音") {
		t.Fatalf("expected scene 1 dialogue to mention recording clue, got %s", got)
	}

	if got := doc.Scenes[1].Objective; containsAnyText(got, "车站", "寄信人") {
		t.Fatalf("expected scene 2 objective to avoid station template, got %s", got)
	}
	if got := doc.Scenes[1].Beats[1].Content; containsAnyText(got, "车站", "寄信人") {
		t.Fatalf("expected scene 2 dialogue to avoid station template, got %s", got)
	}
	if got := doc.Scenes[1].Notes.OpenQuestions[0]; containsAnyText(got, "车站", "寄信人") {
		t.Fatalf("expected scene 2 open question to avoid station template, got %s", got)
	}

	if got := doc.Scenes[2].Objective; !containsAnyText(got, "钥匙", "打开") {
		t.Fatalf("expected scene 3 objective to stay on key clue, got %s", got)
	}
	if got := doc.Scenes[2].Notes.OpenQuestions[0]; !containsAnyText(got, "钥匙", "门") {
		t.Fatalf("expected scene 3 open question to stay on key/door clue, got %s", got)
	}
}

func assertFamilyWordSuspenseDocument(t *testing.T, doc screenplay.Document) {
	t.Helper()

	if len(doc.Scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(doc.Scenes))
	}

	if len(doc.Characters) < 3 {
		t.Fatalf("expected multiple extracted characters, got %#v", doc.Characters)
	}
	expectedNames := []string{"闻溪", "郑岚", "老秦"}
	for _, expectedName := range expectedNames {
		found := false
		for _, character := range doc.Characters {
			if character.Name == expectedName {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected extracted characters to include %s, got %#v", expectedName, doc.Characters)
		}
	}

	if got := doc.Scenes[0].Summary; strings.Contains(got, "This chapter") {
		t.Fatalf("expected chinese summary fallback path, got %s", got)
	}
	if got := doc.Scenes[0].Objective; !containsAnyText(got, "录音", "声音") {
		t.Fatalf("expected scene 1 objective to stay on recording clue, got %s", got)
	}
	if got := doc.Scenes[0].Objective; containsAnyText(got, "团圆饭", "误会说开", "和解") {
		t.Fatalf("expected scene 1 objective to avoid family template leakage, got %s", got)
	}
	if got := doc.Scenes[1].Objective; !containsAnyText(got, "编号", "钥匙") {
		t.Fatalf("expected scene 2 objective to stay on current clue, got %s", got)
	}
	if got := doc.Scenes[1].Beats[1].Content; !containsAnyText(got, "编号", "钥匙", "线索") {
		t.Fatalf("expected scene 2 dialogue to stay on current clue, got %s", got)
	}
	if got := doc.Scenes[2].Objective; !containsAnyText(got, "钥匙", "打开", "仓库") {
		t.Fatalf("expected scene 3 objective to stay on key/warehouse action, got %s", got)
	}
}

func assertGrowthFantasyDocument(t *testing.T, doc screenplay.Document) {
	t.Helper()

	if len(doc.Scenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(doc.Scenes))
	}
	if len(doc.Validation.Warnings) == 0 {
		t.Fatal("expected growth-fantasy regression to surface validation warnings")
	}

	for _, badName := range []string{"所以", "现在", "更别"} {
		for _, character := range doc.Characters {
			if character.Name == badName {
				t.Fatalf("expected filtered fragment %s to stay out of characters, got %#v", badName, doc.Characters)
			}
		}
	}
	for _, expectedName := range []string{"艾琳", "罗莎", "维恩"} {
		found := false
		for _, character := range doc.Characters {
			if character.Name == expectedName {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected extracted characters to include %s, got %#v", expectedName, doc.Characters)
		}
	}

	for idx, scene := range doc.Scenes {
		if len(scene.Beats) < 3 {
			t.Fatalf("expected scene %d to have at least 3 beats, got %d", idx+1, len(scene.Beats))
		}
		if scene.Beats[0].Content == scene.Summary || strings.Contains(scene.Beats[0].Content, "...") {
			t.Fatalf("expected scene %d opening beat to stay concrete, got %s", idx+1, scene.Beats[0].Content)
		}
		if containsAnyText(scene.Objective, "录音", "匿名", "字条", "门锁", "车站", "寄信人") {
			t.Fatalf("expected scene %d objective to avoid suspense leakage, got %s", idx+1, scene.Objective)
		}
		if containsAnyText(scene.Notes.OpenQuestions[0], "录音", "匿名", "字条", "门锁", "车站", "寄信人") {
			t.Fatalf("expected scene %d open question to avoid suspense leakage, got %s", idx+1, scene.Notes.OpenQuestions[0])
		}
	}
}

func assertRealGrowthFantasyDocument(t *testing.T, doc screenplay.Document) {
	t.Helper()

	if len(doc.Scenes) <= len(doc.Source.Chapters) {
		t.Fatalf("expected at least one chapter to split into multiple scenes, got %d scenes for %d chapters", len(doc.Scenes), len(doc.Source.Chapters))
	}
	if len(doc.Validation.Warnings) == 0 {
		t.Fatal("expected real growth-fantasy regression to surface validation warnings")
	}

	for _, badName := range testutil.GrowthFantasyRealInputForbiddenFragments() {
		for _, character := range doc.Characters {
			if character.Name == badName {
				t.Fatalf("expected filtered fragment %s to stay out of characters, got %#v", badName, doc.Characters)
			}
		}
	}
	if got := doc.Characters[0].Name; got != "厄洛斯" {
		t.Fatalf("expected protagonist 厄洛斯, got %s", got)
	}
	matched := 0
	for _, expected := range testutil.GrowthFantasyRealInputExpectedNames() {
		for _, character := range doc.Characters {
			if character.Name == expected {
				matched++
				break
			}
		}
	}
	if matched < 3 {
		t.Fatalf("expected core names to be recovered, matched %d in %#v", matched, doc.Characters)
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
			t.Fatalf("expected compact objective for scene %d, got %s", idx+1, scene.Objective)
		}
		if len(scene.Notes.OpenQuestions) == 0 || looksLikeNarrativeCarryover(scene.Notes.OpenQuestions[0]) {
			t.Fatalf("expected compact open question for scene %d, got %#v", idx+1, scene.Notes.OpenQuestions)
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
			t.Fatalf("expected action beat for scene %d", idx+1)
		}
	}

	for _, warning := range doc.Validation.Warnings {
		if strings.Contains(warning, "当前按通用成长/世界观场景规则生成") {
			t.Fatalf("expected warnings to move beyond generic growth copy, got %#v", doc.Validation.Warnings)
		}
	}
	if !containsSpecificWarning(doc.Validation.Warnings, "characters:", "fragment") &&
		!containsSpecificWarning(doc.Validation.Warnings, "protagonist confidence") &&
		!containsSpecificWarning(doc.Validation.Warnings, "objective is still derived") &&
		!containsSpecificWarning(doc.Validation.Warnings, "beat adaptation remains") &&
		!containsSpecificWarning(doc.Validation.Warnings, "location/slugline confidence is low") {
		t.Fatalf("expected concrete low-confidence warnings, got %#v", doc.Validation.Warnings)
	}
}

func containsAnyText(input string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}
	return false
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
	return containsAnyText(input, "因为", "随着", "直到", "只能", "所能看到", "难不成", "让人感觉", "也不知道", "这让他", "原本", "……", "...")
}

func looksLikeBrokenActionBeat(input string) bool {
	input = strings.TrimSpace(input)
	if input == "" {
		return true
	}
	return containsAnyText(input, "【", "】", "...", "……") || strings.HasPrefix(input, "“") || strings.HasPrefix(input, "\"")
}

func containsSpecificWarning(warnings []string, keywords ...string) bool {
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

func yamlDocumentsEqual(actualYAML, expectedYAML string) bool {
	var actualDoc screenplay.Document
	if err := yaml.Unmarshal([]byte(actualYAML), &actualDoc); err != nil {
		return false
	}
	var expectedDoc screenplay.Document
	if err := yaml.Unmarshal([]byte(expectedYAML), &expectedDoc); err != nil {
		return false
	}
	return reflect.DeepEqual(actualDoc, expectedDoc)
}

type debugGenerator struct{}

func (debugGenerator) Name() string {
	return "debug-generator"
}

func (debugGenerator) Generate(_ context.Context, req llm.GenerateRequest) (llm.GenerateResult, error) {
	document := workflow.BuildDocument(req.Input, req.Source, req.Outline, req.Entities, req.Plan)
	document.Validation.Warnings = append(document.Validation.Warnings, "generated via debug generator")

	return llm.GenerateResult{
		Document: document,
		Warnings: document.Validation.Warnings,
		Debug: &llm.DebugSnapshot{
			Provider:   "debug-generator",
			Model:      "fixture-model",
			ParseMode:  "canonical",
			RawContent: "version: \"1.0\"",
		},
	}, nil
}
