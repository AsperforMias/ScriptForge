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

export type WorkspaceSamplePresetId = "suspense" | "workplace" | "campus_relay";

export interface WorkspaceSamplePreset {
  id: WorkspaceSamplePresetId;
  label: string;
  description: string;
  values: WorkspaceFormValues;
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

export const workspaceSamplePresets: WorkspaceSamplePreset[] = [
  {
    id: "suspense",
    label: "悬疑",
    description: "夜归、字条与追踪线索的悬疑短剧样例。",
    values: {
      title: "夜雨疑云",
      author: "示例作者",
      style: "悬疑短剧",
      audience: "大众向",
      notesText: "强化悬疑氛围\n保留主角主动调查的动机",
      generationMode: "deterministic",
      chapters: [
        {
          title: "第一章 深夜回家",
          content:
            "林琪深夜回到公寓，发现门锁似乎被人动过。她停在走廊里，不确定是否应该立刻进去。",
        },
        {
          title: "第二章 陌生字条",
          content:
            "她在房间里找到一张陌生字条，上面只写着今晚别睡。林琪意识到有人提前进入过房间。",
        },
        {
          title: "第三章 清晨追踪",
          content:
            "第二天清晨，林琪带着字条前往车站，试图顺着纸上的线索找到寄信人。",
        },
      ],
    },
  },
  {
    id: "workplace",
    label: "职场",
    description: "汇报前夜的数据异常与团队猜疑样例。",
    values: {
      title: "交稿前夜",
      author: "示例作者",
      style: "职场短剧",
      audience: "都市向",
      notesText: "突出时间压力\n保留团队猜疑",
      generationMode: "deterministic",
      chapters: [
        {
          title: "第一章 数据被换",
          content:
            "苏禾深夜留在办公室复核提案，发现明早汇报用的数据被人替换。她意识到项目组里有人提前动了最终版本。",
        },
        {
          title: "第二章 咖啡馆对质",
          content:
            "她约同组同事在咖啡馆见面，对方却反问她是不是想独占客户。苏禾意识到怀疑已经在团队里扩散。",
        },
        {
          title: "第三章 会议室摊牌",
          content:
            "第二天清晨，苏禾带着备份文件走进会议室，决定在正式汇报前把问题摆到台面上。",
        },
      ],
    },
  },
  {
    id: "campus_relay",
    label: "校园运动",
    description: "接力决赛前的队伍压力与成长样例。",
    values: {
      title: "最后一棒",
      author: "示例作者",
      style: "青春运动短剧",
      audience: "校园向",
      notesText: "保留队伍压力\n突出临场成长",
      generationMode: "deterministic",
      chapters: [
        {
          title: "第一章 操场加练",
          content:
            "周宁晚上独自在操场加练接力，教练突然通知主力队友可能缺席决赛。她第一次意识到最后一棒会落到自己手里。",
        },
        {
          title: "第二章 教室争执",
          content:
            "第二天一早，她在教室里听见替补队友质疑战术安排，队伍差点在比赛前先吵散。周宁只能临时站出来稳住大家。",
        },
        {
          title: "第三章 跑道起跑",
          content:
            "比赛当天清晨，周宁站上跑道，决定不再等待主力归队，而是带着现有阵容把接力跑完。",
        },
      ],
    },
  },
];

export const sampleWorkspaceFormValues: WorkspaceFormValues = workspaceSamplePresets[0].values;

export function parseNotes(notesText: string) {
  return notesText
    .split(/\r?\n|,/)
    .map((note) => note.trim())
    .filter(Boolean);
}
