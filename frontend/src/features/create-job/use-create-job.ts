import { useMutation } from "@tanstack/react-query";
import { createJob } from "./api";

export function useCreateJob() {
  return useMutation({
    mutationFn: createJob,
  });
}
