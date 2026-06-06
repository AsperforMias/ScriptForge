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
  recommended?: boolean;
  values: WorkspaceFormValues;
}

export function createEmptyChapterDraft(index: number): WorkspaceChapterDraft {
  return {
    title: `第${index}章`,
    content: "",
  };
}

const emptyWorkspaceFormValues: WorkspaceFormValues = {
  title: "",
  author: "",
  style: "",
  audience: "",
  notesText: "",
  generationMode: "llm",
  chapters: [createEmptyChapterDraft(1), createEmptyChapterDraft(2), createEmptyChapterDraft(3)],
};

export const workspaceSamplePresets: WorkspaceSamplePreset[] = [
  {
    id: "suspense",
    label: "悬疑",
    description: "夜归、字条与车站追查的三章悬疑示例。",
    values: {
      title: "夜雨疑云",
      author: "示例作者",
      style: "悬疑短剧",
      audience: "大众向",
      notesText: "强化悬疑氛围\n保留主角主动调查的动机",
      generationMode: "llm",
      chapters: [
        {
          title: "第一章 深夜回家",
          content:
            "夜里十一点，林琬拖着行李回到旧公寓，刚走到三楼就发现自家门锁有被人拧动过的细小刮痕。走廊尽头的声控灯忽明忽暗，她站在门前没有立刻进去，只先把耳朵贴近门板，确认屋里是不是还有人。\n\n门终于被她推开时，客厅里一切都像离开前那样整齐，只有茶几上的杯垫被人换了方向。林琬想起三天前失联的前室友，忽然怀疑对方回来过，却又找不到更直接的证据。",
        },
        {
          title: "第二章 陌生字条",
          content:
            "她检查卧室和书桌时，在抽屉底层发现一张折得很小的字条，上面只写着四个字: 今晚别睡。纸角沾着雨水，像是不久前才被塞进来。\n\n林琬顺着字条往下查，发现衣柜后方的插座有被人拔插过的痕迹，而床头原本关着的旧收音机不知什么时候停在了凌晨两点十七分。她意识到闯入者不是随手翻动，而是在屋里找过某样特定的东西。",
        },
        {
          title: "第三章 清晨追查",
          content:
            "天刚亮，林琬带着字条和收音机里的异常时间赶到南站。她记得前室友离开这座城时，就是在凌晨两点多从这里上车。\n\n车站值班窗口还没完全开门，她只能先去翻看寄存柜区域的监控提示牌，试着从字条笔迹和昨夜的闯入痕迹之间找联系。就在她准备拨通报警电话时，身后忽然有人低声叫出了她的名字。",
        },
      ],
    },
  },
  {
    id: "workplace",
    label: "职场",
    description: "汇报前夜的数据异常、团队猜疑与会议室摊牌。",
    recommended: true,
    values: {
      title: "交稿前夜",
      author: "示例作者",
      style: "职场短剧",
      audience: "都市向",
      notesText: "突出时间压力\n保留团队猜疑",
      generationMode: "llm",
      chapters: [
        {
          title: "第一章 数据被换",
          content:
            "晚上十点半，苏栀一个人留在办公室复核第二天要向客户汇报的提案。她本来只是想再确认一遍预算页，却发现终稿里的核心数据被悄悄换成了两周前的旧版本，连图表颜色和注释顺序都被人刻意整理得像是她自己提交过的一样。\n\n她翻出本地备份后确认，问题只出现在共享盘上的最终文件。更麻烦的是，项目组里只有三个人有编辑权限，而明早九点客户就会带着法务一起到场。如果她现在直接发消息质问，只会提前惊动真正动手的人。",
        },
        {
          title: "第二章 咖啡馆对质",
          content:
            "第二天下午，苏栀把同组同事顾屿约到公司楼下的咖啡馆，想先确认是谁在项目组里放出了“她准备独占客户”的风声。顾屿没有正面回答，反而质问她这几天为什么一直单独和客户助理沟通，像是早就认定她会把责任推给别人。\n\n对话越说越僵，苏栀才意识到，数据被换只是表面问题，真正危险的是团队内部已经有人提前布好了怀疑链。她如果处理失误，明天的汇报现场就不会只是解释数据，而会变成一次公开甩锅。",
        },
        {
          title: "第三章 会议室摊牌",
          content:
            "第二天清晨八点四十，苏栀带着打印好的备份文件和修改记录提前走进会议室。客户还没到，项目负责人和顾屿已经坐在里面，桌上摆着那份被替换过数据的终稿。\n\n她原本只想在会前私下说明问题，但看到负责人试图直接开始彩排，她决定不再等下去，而是当场把版本记录投到屏幕上。无论最后责任落在谁身上，她都必须先把错误拦在正式汇报开始之前。",
        },
      ],
    },
  },
  {
    id: "campus_relay",
    label: "校园运动",
    description: "接力决赛前的队伍压力、教室争执与跑道决断。",
    values: {
      title: "最后一棒",
      author: "示例作者",
      style: "青春运动短剧",
      audience: "校园向",
      notesText: "保留队伍压力\n突出临场成长",
      generationMode: "llm",
      chapters: [
        {
          title: "第一章 操场加练",
          content:
            "晚自习结束后，周宁独自在操场加练接棒。她原本只是队里的第三替补，却在最后一轮冲刺时被教练叫停，得知主力最后一棒扭伤脚踝，极有可能赶不上两天后的决赛。\n\n夜风一阵阵刮过看台，周宁拿着接力棒站在跑道尽头，第一次真正意识到自己可能要顶上最关键的位置。可她这学期一直跑得不稳，队里不少人都默认最后一棒不会轮到她。",
        },
        {
          title: "第二章 教室争执",
          content:
            "第二天一早，周宁刚走进教室，就听见两名队友在后排争论战术安排。有人觉得应该临时换人，有人认为干脆放弃争冠保住名次，连一向最稳的第二棒都开始怀疑现在的排兵是不是在赌运气。\n\n争执很快从谁跑最后一棒，升级成谁该为前几场失利负责。周宁原本想装作没听见，可当大家把目光投向她时，她知道自己再沉默下去，队伍可能会在上赛道前就先散掉。",
        },
        {
          title: "第三章 跑道起跑",
          content:
            "比赛当天清晨，周宁和队友站上跑道热身时，主力最后一棒仍然没有出现。看台上的广播已经开始报项目顺序，教练只能把最终名单交上去，不再给任何人留下犹豫的时间。\n\n周宁接过号码贴时手心全是汗。她知道这场比赛未必能跑赢最强的对手，但如果她还在等主力归队，队伍就连正常起跑都做不到。她决定带着现有阵容把这场接力完整跑完，哪怕最后一棒必须由她来扛。",
        },
      ],
    },
  },
];

export const recommendedWorkspaceSamplePreset: WorkspaceSamplePreset =
  workspaceSamplePresets.find((preset) => preset.recommended) ?? workspaceSamplePresets[0];

export function cloneWorkspaceFormValues(values: WorkspaceFormValues): WorkspaceFormValues {
  return {
    ...values,
    chapters: values.chapters.map((chapter) => ({ ...chapter })),
  };
}

export const blankWorkspaceFormValues: WorkspaceFormValues = cloneWorkspaceFormValues(
  emptyWorkspaceFormValues,
);

export const defaultWorkspaceFormValues: WorkspaceFormValues = cloneWorkspaceFormValues(
  recommendedWorkspaceSamplePreset?.values ?? emptyWorkspaceFormValues,
);

export const sampleWorkspaceFormValues: WorkspaceFormValues = cloneWorkspaceFormValues(
  workspaceSamplePresets[0]?.values ?? emptyWorkspaceFormValues,
);

function normalizeWorkspaceFormValues(values: WorkspaceFormValues) {
  return {
    title: values.title.trim(),
    author: values.author.trim(),
    style: values.style.trim(),
    audience: values.audience.trim(),
    notesText: values.notesText.trim(),
    generationMode: values.generationMode,
    chapters: values.chapters.map((chapter) => ({
      title: chapter.title.trim(),
      content: chapter.content.trim(),
    })),
  };
}

export function findMatchingWorkspaceSamplePresetId(
  values: WorkspaceFormValues,
): WorkspaceSamplePresetId | null {
  const normalizedValues = JSON.stringify(normalizeWorkspaceFormValues(values));

  return (
    workspaceSamplePresets.find((preset) => {
      return JSON.stringify(normalizeWorkspaceFormValues(preset.values)) === normalizedValues;
    })?.id ?? null
  );
}

export function parseNotes(notesText: string) {
  return notesText
    .split(/\r?\n|,/)
    .map((note) => note.trim())
    .filter(Boolean);
}
