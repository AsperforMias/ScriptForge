import type { GenerationMode, JobStatus, PipelineStageName } from "../types/api";

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
  deterministic: "标准草稿",
  llm: "AI 增强",
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
