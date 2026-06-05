import { requestJson } from "../../lib/http";
import type { JobResultResponse } from "../../types/api";

export function fetchJobResult(jobId: string) {
  return requestJson<JobResultResponse>(`/jobs/${jobId}/result`);
}
