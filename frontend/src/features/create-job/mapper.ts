import type { CreateJobRequest } from "../../types/api";
import type { WorkspaceFormValues } from "./form";
import { parseNotes } from "./form";

export function mapDraftToCreateJobRequest(draft: WorkspaceFormValues): CreateJobRequest {
  return {
    source: {
      title: draft.title,
      author: draft.author,
      chapters: draft.chapters.map((chapter, index) => ({
        index: index + 1,
        title: chapter.title,
        content: chapter.content,
      })),
    },
    adaptation: {
      style: draft.style,
      audience: draft.audience,
      notes: parseNotes(draft.notesText),
    },
    generation: {
      mode: draft.generationMode,
    },
  };
}
