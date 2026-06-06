import type { GenerationMode, JobStatus, PipelineStageName } from "../types/api";
import type { CharacterRole, InteriorExterior } from "../types/screenplay";

const stageLabels: Record<PipelineStageName, string> = {
  ingest: "素材接收",
  outline: "章节梳理",
  entities: "角色与地点",
  scene_planning: "场景规划",
  screenplay_generation: "剧本生成",
  validation: "结构校验",
  persistence: "结果保存",
};

const statusLabels: Record<JobStatus, string> = {
  queued: "排队中",
  running: "处理中",
  succeeded: "已完成",
  failed: "失败",
};

const generationModeLabels: Record<GenerationMode, string> = {
  deterministic: "标准草稿（兜底）",
  llm: "AI 增强",
};

const characterRoleLabels: Record<CharacterRole, string> = {
  protagonist: "主角",
  supporting: "配角",
  antagonist: "对立角色",
  narrator: "叙述视角",
  other: "其他人物",
};

const interiorExteriorLabels: Record<InteriorExterior, string> = {
  INT: "室内",
  EXT: "室外",
  "INT/EXT": "室内外",
};

const sceneTimeLabels: Record<string, string> = {
  DAY: "白天",
  NIGHT: "夜晚",
  MORNING: "清晨",
  NOON: "中午",
  AFTERNOON: "下午",
  EVENING: "傍晚",
  DUSK: "黄昏",
  DAWN: "黎明",
  LATE_NIGHT: "深夜",
  UNKNOWN: "时间未明",
};

export function formatStageName(stageName: PipelineStageName) {
  return stageLabels[stageName];
}

export function formatJobStatus(status: JobStatus) {
  return statusLabels[status];
}

export function formatGenerationMode(mode: GenerationMode) {
  return generationModeLabels[mode];
}

export function formatCharacterRole(role: CharacterRole) {
  return characterRoleLabels[role] ?? "人物";
}

export function formatInteriorExterior(value: InteriorExterior) {
  return interiorExteriorLabels[value] ?? "场景";
}

export function formatSceneTime(value: string) {
  const normalized = value.trim().toUpperCase();
  if (!normalized) {
    return "时间待补充";
  }

  return sceneTimeLabels[normalized] ?? value;
}

export function formatDateTime(timestamp: string) {
  const date = new Date(timestamp);

  if (Number.isNaN(date.getTime())) {
    return timestamp;
  }

  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "numeric",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export function getErrorMessage(error: unknown) {
  if (!error) {
    return "";
  }

  if (typeof error === "string") {
    return error;
  }

  if (error instanceof Error) {
    return error.message;
  }

  if (typeof error === "object" && "message" in error && typeof error.message === "string") {
    return error.message;
  }

  return "请求失败，请稍后再试。";
}

export function formatLocationDescription(description?: string) {
  const trimmed = description?.trim() ?? "";
  if (!trimmed) {
    return "等待补充地点说明";
  }

  if (/^Primary dramatic location inferred from chapter/i.test(trimmed)) {
    return "当前章节里最主要的发生地点。";
  }

  if (/scene evidence/i.test(trimmed)) {
    return "根据章节线索整理出的主要发生地点。";
  }

  return trimmed;
}

function normalizeUserFacingText(text: string) {
  return text.replace(/\s+/g, " ").trim();
}

function formatTechnicalIssue(issue: string) {
  const trimmed = normalizeUserFacingText(issue);

  if (!trimmed) {
    return "请结合原文复核这一部分内容。";
  }

  if (trimmed.includes("llm generation via") && trimmed.includes("fell back to deterministic baseline")) {
    return "AI 生成等待过长或调用失败，系统已改用基础草稿模式生成，请重点复核人物、场景和细节。";
  }

  if (trimmed === "generated via openai_compatible provider") {
    return "本次结果由 AI 模型生成，请重点复核人物、场景和细节。";
  }

  if (trimmed === "generated via mock llm provider") {
    return "当前结果来自演示模式，不代表真实模型输出。";
  }

  if (trimmed === "generated via debug generator") {
    return "当前结果来自调试生成器，不代表真实模型输出。";
  }

  if (trimmed.includes("normalized from a loose provider scene")) {
    return "模型返回结构不够稳定，系统已自动整理成可编辑草稿，请重点复核地点、场景目标和对话。";
  }

  if (trimmed.includes("scene split confidence is low")) {
    return "场景切分把握度较低，这一章里可能还有多个转折挤在同一场里。";
  }

  if (trimmed.includes("objective is still derived from long narrative phrasing")) {
    return "场景目标仍偏概述，建议对照原文收紧成当前这一场要完成的具体动作。";
  }

  if (trimmed.includes("objective still reads like template or placeholder copy")) {
    return "场景目标仍像套话，建议改成这一场里角色当下要完成的具体任务。";
  }

  if (trimmed.includes("objective still fell back to generic copy")) {
    return "场景目标仍偏笼统，建议按这一场最直接的推进动作重写。";
  }

  if (trimmed.includes("objective duplicates")) {
    return "场景目标和其他场重复了，建议改成只属于这一场的具体任务。";
  }

  if (trimmed.includes("beat adaptation remains low confidence")) {
    return "动作整理把握度较低，原文仍以概述或内心叙述为主，请重点复核可拍片段。";
  }

  if (trimmed.includes("open_questions still fell back to generic copy")) {
    return "后续悬念仍偏笼统，请按这一场尚未解决的信息重写。";
  }

  if (trimmed.includes("open_questions duplicates")) {
    return "后续悬念和其他场重复了，建议只保留这一场独有的未解信息。";
  }

  if (trimmed.includes("open_questions is still generic")) {
    return "后续悬念仍偏空泛，没有明确证据时宁可先留空。";
  }

  if (trimmed.includes("location/slugline confidence is low")) {
    return "地点判断把握度较低，原文可能跨了多个空间，请重点复核这一场的地点。";
  }

  if (trimmed.includes("split scene still relies on inherited chapter location")) {
    return "拆分后的场景仍沿用了章节级地点，建议补细这一场的具体位置。";
  }

  if (trimmed.includes("multiple provider scenes for one chapter were merged")) {
    return "同一章节的场景边界不够稳定，系统已先合并成可编辑草稿，请人工复核场景切分。";
  }

  if (trimmed.includes("filtered fragment-like candidates")) {
    return "系统已过滤部分碎片化的人物称呼，请复核角色命名。";
  }

  if (trimmed.includes("protagonist confidence is low")) {
    const match = trimmed.match(/selected\s+(.+?)\s+from/i);
    return match
      ? `主角识别把握度较低，当前暂按“${match[1]}”整理，请复核人物命名。`
      : "主角识别把握度较低，请重点复核人物命名。";
  }

  if (trimmed.includes("scene count inferred from chapter titles")) {
    return "场景数量主要按章节标题推断，请结合正文复核切分结果。";
  }

  return trimmed;
}

export function formatUserFacingWarning(warning: string) {
  const trimmed = normalizeUserFacingText(warning);
  if (!trimmed) {
    return "请结合原文复核这一部分内容。";
  }

  const sceneMatch = trimmed.match(/^scene_(\d+):\s*(.+)$/i);
  if (sceneMatch) {
    return `场景 ${Number.parseInt(sceneMatch[1], 10)}：${formatTechnicalIssue(sceneMatch[2])}`;
  }

  const chapterMatch = trimmed.match(/^chapter_(\d+):\s*(.+)$/i);
  if (chapterMatch) {
    return `第 ${Number.parseInt(chapterMatch[1], 10)} 章：${formatTechnicalIssue(chapterMatch[2])}`;
  }

  const characterMatch = trimmed.match(/^characters:\s*(.+)$/i);
  if (characterMatch) {
    return `人物整理：${formatTechnicalIssue(characterMatch[1])}`;
  }

  return formatTechnicalIssue(trimmed);
}

export function formatUserFacingError(errorMessage: string) {
  const trimmed = normalizeUserFacingText(errorMessage);
  if (!trimmed) {
    return "";
  }

  const lower = trimmed.toLowerCase();

  if (lower.includes("context deadline exceeded") || lower.includes("client.timeout") || lower.includes("timeout")) {
    return "AI 返回较慢，这次结果暂时还没成功载入。可以稍后重试，当前输入内容不会丢失。";
  }

  if (lower.includes("job_not_ready") || lower.includes("job not ready")) {
    return "当前结果还在生成中，完成后会自动载入。";
  }

  if (lower.includes("job_not_found") || lower.includes("job not found")) {
    return "没有找到这次生成记录，请重新发起一次生成。";
  }

  if (lower.includes("networkerror") || lower.includes("failed to fetch")) {
    return "结果加载失败，请检查网络连接后重试。";
  }

  if (lower.includes("status=")) {
    return "AI 服务这次没有正常返回结果，请稍后重试。";
  }

  if (lower.includes("fell back to deterministic baseline")) {
    return "AI 生成等待过长或调用失败，系统已改用基础草稿模式生成，请重点复核人物、场景和细节。";
  }

  return trimmed;
}
