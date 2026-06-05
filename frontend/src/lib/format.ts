import type { JobStatus, PipelineStageName } from "../types/api";

const stageLabels: Record<PipelineStageName, string> = {
  ingest: "素材接收",
  outline: "章节梳理",
  entities: "人物 / 地点",
  scene_planning: "场景规划",
  screenplay_generation: "剧本生成",
  validation: "Schema 校验",
  persistence: "结果落盘",
};

const statusLabels: Record<JobStatus, string> = {
  queued: "排队中",
  running: "处理中",
  succeeded: "已完成",
  failed: "失败",
};

export function formatStageName(stageName: PipelineStageName) {
  return stageLabels[stageName];
}

export function formatJobStatus(status: JobStatus) {
  return statusLabels[status];
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
