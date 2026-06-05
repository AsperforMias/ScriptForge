import type { ScreenplayDocument } from "./screenplay";

export type JobStatus = "queued" | "running" | "succeeded" | "failed";

export type PipelineStageName =
  | "ingest"
  | "outline"
  | "entities"
  | "scene_planning"
  | "screenplay_generation"
  | "validation"
  | "persistence";

export type GenerationMode = "deterministic" | "llm";

export type ApiErrorCode =
  | "invalid_input"
  | "job_not_found"
  | "job_not_ready"
  | "generation_failed"
  | "internal_error";

export interface ApiMeta {
  request_id: string;
}

export interface ApiError {
  code: ApiErrorCode | string;
  message: string;
  details?: Record<string, unknown>;
}

export interface ApiEnvelope<T> {
  data: T | null;
  error: ApiError | null;
  meta: ApiMeta;
}

export interface SourceChapterInput {
  index: number;
  title: string;
  content: string;
}

export interface CreateJobRequest {
  source: {
    title: string;
    author: string;
    chapters: SourceChapterInput[];
  };
  adaptation: {
    style: string;
    audience: string;
    notes: string[];
  };
  generation: {
    mode: GenerationMode;
  };
}

export interface JobSummary {
  id: string;
  status: JobStatus;
  current_stage: PipelineStageName;
  progress_percent: number;
  source_title?: string;
  generation_mode?: GenerationMode;
  warnings: string[];
  error_message: string;
  created_at: string;
  updated_at: string;
}

export interface JobStage {
  name: PipelineStageName;
  status: JobStatus;
  warning_count?: number;
  error_message?: string;
  started_at?: string;
  finished_at?: string;
}

export interface CreateJobResponse {
  job: JobSummary;
}

export interface JobDetailsResponse {
  job: JobSummary;
  stages: JobStage[];
}

export interface JobResultResponse {
  job_id: string;
  screenplay: ScreenplayDocument;
  yaml_text: string;
}
