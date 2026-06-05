import { ChapterList } from "../components/input/chapter-list";
import { SourceForm } from "../components/input/source-form";
import { JobStatusPanel } from "../components/jobs/job-status-panel";
import { ExportActions } from "../components/result/export-actions";
import { ScreenplaySummary } from "../components/result/screenplay-summary";
import { YamlEditor } from "../components/result/yaml-editor";
import type { JobStage, JobSummary } from "../types/api";
import type { ScreenplayDocument } from "../types/screenplay";

const demoJob: JobSummary = {
  id: "job_demo_20260605",
  status: "running",
  current_stage: "scene_planning",
  progress_percent: 62,
  source_title: "夜雨疑云",
  generation_mode: "deterministic",
  warnings: [],
  error_message: "",
  created_at: "2026-06-05T09:00:00Z",
  updated_at: "2026-06-05T09:01:24Z",
};

const demoStages: JobStage[] = [
  {
    name: "ingest",
    status: "succeeded",
    started_at: "2026-06-05T09:00:00Z",
    finished_at: "2026-06-05T09:00:05Z",
  },
  {
    name: "outline",
    status: "succeeded",
    started_at: "2026-06-05T09:00:05Z",
    finished_at: "2026-06-05T09:00:11Z",
  },
  {
    name: "entities",
    status: "succeeded",
    started_at: "2026-06-05T09:00:11Z",
    finished_at: "2026-06-05T09:00:16Z",
  },
  {
    name: "scene_planning",
    status: "running",
    started_at: "2026-06-05T09:00:16Z",
  },
  {
    name: "screenplay_generation",
    status: "queued",
  },
  {
    name: "validation",
    status: "queued",
  },
  {
    name: "persistence",
    status: "queued",
  },
];

const demoScreenplay: ScreenplayDocument = {
  version: "1.0",
  source: {
    title: "夜雨疑云",
    author: "示例作者",
    language: "zh-CN",
    chapter_count: 3,
    chapters: [
      { index: 1, title: "门锁", summary: "回家时发现门锁被动过。" },
      { index: 2, title: "录音", summary: "楼道监控与录音揭出新的嫌疑人。" },
      { index: 3, title: "对峙", summary: "主角在旧公寓里完成第一次正面对峙。" },
    ],
  },
  adaptation: {
    style: "悬疑网剧",
    audience: "大众向",
    notes: ["强化外部冲突", "保留第一人称压迫感"],
  },
  characters: [
    {
      id: "char_lin_qi",
      name: "林琪",
      aliases: ["阿琪"],
      role: "protagonist",
      description: "敏感而克制的年轻作者。",
    },
    {
      id: "char_song_zhou",
      name: "宋舟",
      aliases: [],
      role: "supporting",
      description: "表面冷静，实则隐瞒信息的邻居。",
    },
  ],
  locations: [
    {
      id: "loc_old_apartment",
      name: "旧公寓",
      description: "走廊狭窄、声控灯反应迟钝的老式公寓。",
    },
    {
      id: "loc_rooftop",
      name: "楼顶平台",
      description: "潮湿、带风，适合对峙与揭露。",
    },
  ],
  scenes: [
    {
      id: "scene_001",
      title: "深夜回家",
      source_chapters: [1],
      slugline: {
        interior_exterior: "INT",
        location_id: "loc_old_apartment",
        time: "NIGHT",
      },
      summary: "林琪回到旧公寓，察觉门锁被动过。",
      objective: "建立悬疑氛围并埋下异常线索。",
      beats: [
        { type: "action", content: "声控灯忽明忽暗，林琪停在门前。", emotion: "uneasy" },
        {
          type: "dialogue",
          character_id: "char_lin_qi",
          content: "我出门前明明上了锁。",
          emotion: "uneasy",
        },
      ],
      notes: {
        adaptation_reason: "将内心描写转为可拍摄动作与短对白。",
        open_questions: ["门锁异常是否要在下一场直接揭示原因？"],
      },
    },
    {
      id: "scene_002",
      title: "楼顶试探",
      source_chapters: [2, 3],
      slugline: {
        interior_exterior: "EXT",
        location_id: "loc_rooftop",
        time: "LATE NIGHT",
      },
      summary: "林琪与宋舟在楼顶试探彼此，怀疑升级。",
      objective: "把嫌疑人关系正式拉到台前。",
      beats: [
        { type: "action", content: "风吹起晾衣绳，宋舟先一步挡住出口。" },
        {
          type: "dialogue",
          character_id: "char_song_zhou",
          content: "你要找的真相，也许根本不在屋里。",
        },
      ],
      notes: {
        adaptation_reason: "把分散在线索章节中的对峙提前合并，增强中段张力。",
        open_questions: ["是否需要增加一场监控回放来解释宋舟动机？"],
      },
    },
  ],
  validation: {
    status: "passed",
    warnings: [],
  },
};

const demoYaml = `version: "1.0"
source:
  title: "夜雨疑云"
  author: "示例作者"
  language: "zh-CN"
  chapter_count: 3
adaptation:
  style: "悬疑网剧"
  audience: "大众向"
  notes:
    - "强化外部冲突"
characters:
  - id: "char_lin_qi"
    name: "林琪"
    role: "protagonist"
locations:
  - id: "loc_old_apartment"
    name: "旧公寓"
scenes:
  - id: "scene_001"
    title: "深夜回家"
    source_chapters: [1]
    slugline:
      interior_exterior: "INT"
      location_id: "loc_old_apartment"
      time: "NIGHT"
    summary: "林琪回到旧公寓，察觉门锁被动过。"
validation:
  status: "passed"
  warnings: []`;

export function WorkspacePage() {
  return (
    <main className="workspace-shell">
      <section className="page-intro">
        <p className="eyebrow">ScriptForge Editorial Studio</p>
        <div className="page-intro__heading">
          <div>
            <h1>小说转剧本工作台</h1>
            <p>
              按文档锁定的单页工作区架构，先把输入、任务状态与 YAML
              结果三条主线并列落地。
            </p>
          </div>
          <div className="page-intro__aside">
            <span>Vite + React + TypeScript</span>
            <span>TanStack Query + React Hook Form</span>
            <span>YAML-first Result Workspace</span>
          </div>
        </div>
      </section>

      <section className="workspace-grid" aria-label="ScriptForge workspace">
        <div className="panel panel--input">
          <div className="panel__header">
            <div>
              <p className="panel__eyebrow">Input Workspace</p>
              <h2>创作素材板</h2>
            </div>
            <span className="panel__badge">至少 3 章</span>
          </div>
          <SourceForm />
          <ChapterList />
        </div>

        <div className="panel panel--status">
          <div className="panel__header">
            <div>
              <p className="panel__eyebrow">Job Status</p>
              <h2>任务进度带</h2>
            </div>
            <span className="panel__badge panel__badge--muted">2s polling</span>
          </div>
          <JobStatusPanel job={demoJob} stages={demoStages} />
        </div>

        <div className="panel panel--result">
          <div className="panel__header">
            <div>
              <p className="panel__eyebrow">Result Workspace</p>
              <h2>剧本初稿与结构摘要</h2>
            </div>
            <span className="panel__badge panel__badge--accent">YAML 核心结果</span>
          </div>
          <ExportActions jobId={demoJob.id} />
          <YamlEditor yamlText={demoYaml} />
          <ScreenplaySummary screenplay={demoScreenplay} />
        </div>
      </section>
    </main>
  );
}
