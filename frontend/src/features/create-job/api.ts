import { requestJson } from "../../lib/http";
import type { CreateJobRequest, CreateJobResponse } from "../../types/api";

export function createJob(payload: CreateJobRequest) {
  return requestJson<CreateJobResponse>("/jobs", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}
