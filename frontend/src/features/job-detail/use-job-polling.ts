import { useQuery } from "@tanstack/react-query";
import { fetchJobDetails } from "./api";

export function useJobPolling(jobId: string | null) {
  return useQuery({
    queryKey: ["job-details", jobId],
    queryFn: async () => {
      if (!jobId) {
        throw new Error("job id is required");
      }

      return fetchJobDetails(jobId);
    },
    enabled: Boolean(jobId),
    refetchInterval: (query) => {
      const status = query.state.data?.data?.job.status;

      if (!status || status === "queued" || status === "running") {
        return 2000;
      }

      return false;
    },
  });
}
