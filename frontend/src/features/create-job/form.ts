import type { GenerationMode } from "../../types/api";

export interface WorkspaceChapterDraft {
  title: string;
  content: string;
}

export interface WorkspaceFormValues {
  title: string;
  author: string;
  style: string;
  audience: string;
  notesText: string;
  generationMode: GenerationMode;
  chapters: WorkspaceChapterDraft[];
}

export function createEmptyChapterDraft(index: number): WorkspaceChapterDraft {
  return {
    title: `第${index}章`,
    content: "",
  };
}

export const defaultWorkspaceFormValues: WorkspaceFormValues = {
  title: "",
  author: "",
  style: "",
  audience: "",
  notesText: "",
  generationMode: "deterministic",
  chapters: [createEmptyChapterDraft(1), createEmptyChapterDraft(2), createEmptyChapterDraft(3)],
};

export const sampleWorkspaceFormValues: WorkspaceFormValues = {
  title: "夜雨疑云",
  author: "示例作者",
  style: "悬疑网剧",
  audience: "大众向",
  notesText: "强化外部冲突\n保留第一人称压迫感\n优先可拍摄动作",
  generationMode: "deterministic",
  chapters: [
    {
      title: "第一章 深夜回家",
      content:
        "林琪深夜回到旧公寓，发现门锁似乎被人动过。她停在走廊里，不确定是否应该立刻进去。",
    },
    {
      title: "第二章 陌生字条",
      content:
        "她在房间里找到一张陌生字条，上面只写着今晚别睡。林琪意识到有人提前进入过房间。",
    },
    {
      title: "第三章 清晨追踪",
      content:
        "第二天清晨，林琪带着字条前往车站，试图顺着纸上的线索找到寄信人，却先一步撞见了楼里的邻居宋舟。",
    },
  ],
};

export function parseNotes(notesText: string) {
  return notesText
    .split(/\r?\n|,/)
    .map((note) => note.trim())
    .filter(Boolean);
}
