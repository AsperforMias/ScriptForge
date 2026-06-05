import { useQuery } from "@tanstack/react-query";
import { fetchJobResult } from "./api";

export function useJobResult(jobId: string | null, enabled = true) {
  return useQuery({
    queryKey: ["job-result", jobId],
    queryFn: async () => {
      if (!jobId) {
        throw new Error("job id is required");
      }

      return fetchJobResult(jobId);
    },
    enabled: Boolean(jobId) && enabled,
  });
}
