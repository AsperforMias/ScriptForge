import { requestJson } from "../../lib/http";
import type { JobDetailsResponse } from "../../types/api";

export function fetchJobDetails(jobId: string) {
  return requestJson<JobDetailsResponse>(`/jobs/${jobId}`);
}
