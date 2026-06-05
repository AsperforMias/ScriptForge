import type { CreateJobRequest, GenerationMode } from "../../types/api";

export interface WorkspaceChapterDraft {
  title: string;
  content: string;
}

export interface WorkspaceDraft {
  title: string;
  author: string;
  style: string;
  audience: string;
  notes: string[];
  generationMode: GenerationMode;
  chapters: WorkspaceChapterDraft[];
}

export function mapDraftToCreateJobRequest(draft: WorkspaceDraft): CreateJobRequest {
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
      notes: draft.notes,
    },
    generation: {
      mode: draft.generationMode,
    },
  };
}
