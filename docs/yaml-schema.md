# Screenplay YAML Schema

## 设计目标

该 YAML Schema 不是为了追求影视工业标准的完整性，而是为了满足本题的 4 个核心要求：
- 能承载 3 章以上小说的改编结果
- 足够结构化，便于程序生成和校验
- 足够可读，便于作者手工编辑
- 能清晰体现“章节 -> 场景 -> 剧本片段”的生成链路

因此 Schema 采用“元信息 + 索引表 + 场景列表”的形式，而不是纯自由文本。
同时，为了避免模型在证据不足时硬填 `objective`、`open_questions`、`dialogue`，当前 hardening 版本额外引入了可选的 `evidence` 与 `review` 层：前者回答“这场戏依据哪段章节证据成立”，后者回答“这场戏当前有多可信、哪些点需要人工复核”。

## 顶层结构

```yaml
version: "1.0"
source:
  title: ""
  author: ""
  language: "zh-CN"
  chapter_count: 3
adaptation:
  style: ""
  audience: ""
  notes: []
characters: []
locations: []
scenes: []
validation:
  status: "passed"
  warnings: []
```

## 字段定义

本节中：
- “必填”表示顶层合法 YAML 必须出现该字段
- “可选”表示字段可以省略，但若出现则必须满足约束

### `version`

作用：
- 标记 Schema 版本，便于未来演进和兼容

约束：
- 字符串
- 首版固定为 `1.0`

### `source`

作用：
- 保留原始小说来源信息
- 使生成产物能追溯回输入文本

建议字段：

```yaml
source:
  title: "作品标题"
  author: "作者名"
  language: "zh-CN"
  chapter_count: 3
  chapters:
    - index: 1
      title: "第一章"
      summary: "章节摘要"
    - index: 2
      title: "第二章"
      summary: "章节摘要"
```

规范要求：
- `title`：必填，非空字符串
- `author`：可选，字符串
- `language`：必填，首版默认 `zh-CN`
- `chapter_count`：必填，整数，且必须大于等于 3
- `chapters`：必填，数组长度必须等于 `chapter_count`
- `chapters[].index`：必填，正整数，且从 1 开始连续
- `chapters[].title`：必填，非空字符串
- `chapters[].summary`：可选，字符串

### `adaptation`

作用：
- 记录本次改编的生成约束
- 让相同小说可以在不同风格下重复生成并对比

建议字段：

```yaml
adaptation:
  style: "悬疑网剧"
  audience: "大众向"
  notes:
    - "保留第一人称心理张力"
    - "优先强化冲突场景"
```

规范要求：
- `style`：必填，非空字符串
- `audience`：可选，字符串
- `notes`：可选，字符串数组

### `characters`

作用：
- 把人物从场景正文中抽离，降低重复
- 便于前端做人物面板和引用检查

建议字段：

```yaml
characters:
  - id: "char_lin_qi"
    name: "林琪"
    aliases: ["阿琪"]
    role: "protagonist"
    description: "年轻悬疑小说作者，观察敏锐但多疑。"
```

约束建议：
- `id` 全局唯一
- `role` 可枚举：`protagonist` `supporting` `antagonist` `narrator` `other`

规范要求：
- `id`：必填，非空字符串，全局唯一
- `name`：必填，非空字符串
- `aliases`：可选，字符串数组
- `role`：必填，枚举值之一：`protagonist` `supporting` `antagonist` `narrator` `other`
- `description`：可选，字符串

### `locations`

作用：
- 复用场景地点
- 便于后端做场景统计和一致性检查

建议字段：

```yaml
locations:
  - id: "loc_old_apartment"
    name: "旧公寓"
    description: "灯光昏暗的老式居民楼。"
```

规范要求：
- `id`：必填，非空字符串，全局唯一
- `name`：必填，非空字符串
- `description`：可选，字符串

### `scenes`

作用：
- 这是剧本主体
- 每个 scene 必须能映射回来源章节，且具备可编辑的剧本片段结构

建议字段：

```yaml
scenes:
  - id: "scene_001"
    title: "深夜回家"
    source_chapters: [1]
    slugline:
      interior_exterior: "INT"
      location_id: "loc_old_apartment"
      time: "NIGHT"
    summary: "林琪回到旧公寓，察觉门锁被动过。"
    objective: "确认门锁是否真的被人动过。"
    beats:
      - type: "action"
        content: "走廊尽头的声控灯忽明忽暗，林琪停在门前。"
      - type: "dialogue"
        character_id: "char_lin_qi"
        content: "我明明早上锁好了。"
        emotion: "uneasy"
    notes:
      adaptation_reason: "将原文内心描写转化为可拍摄动作与短对白。"
      open_questions:
        - "门锁异常是否需要在下一场直接揭示原因？"
    evidence:
      chapter_indexes: [1]
      excerpt: "林琪回到旧公寓，察觉门锁被动过。"
      cues:
        - "门锁异常"
        - "深夜回家"
    review:
      confidence: "medium"
      issues:
        - "location 仍需人工复核"
```

约束建议：
- `source_chapters` 至少包含一个章节编号
- `beats` 至少包含一个元素
- `type` 首版限定为：`action` `dialogue` `transition` `note`
- `dialogue` 类型应携带 `character_id`

规范要求：
- `id`：必填，非空字符串，全局唯一
- `title`：必填，非空字符串
- `source_chapters`：必填，整数数组，且每个值都必须存在于 `source.chapters[].index`
- `slugline.interior_exterior`：必填，枚举 `INT` `EXT` `INT/EXT`
- `slugline.location_id`：必填，且必须引用已定义的 `locations[].id`
- `slugline.time`：必填，非空字符串
- `summary`：必填，非空字符串
- `objective`：可选，字符串
- `beats`：必填，非空数组
- `beats[].type`：必填，枚举 `action` `dialogue` `transition` `note`
- `beats[].content`：必填，非空字符串
- `beats[].character_id`：当 `type=dialogue` 时必填，且必须引用已定义的 `characters[].id`
- `notes.adaptation_reason`：可选，字符串
- `notes.open_questions`：可选，字符串数组
- `evidence.chapter_indexes`：可选，整数数组；若出现，其值必须存在于 `source.chapters[].index`
- `evidence.excerpt`：可选，字符串；建议放短证据摘录或高度贴近原文的摘要句，不要求长引用
- `evidence.cues`：可选，字符串数组；建议放本 scene 的关键证据线索词
- `review.confidence`：可选，枚举 `high` `medium` `low`
- `review.issues`：可选，字符串数组；用于记录“角色识别不稳 / objective 模板化 / 地点低置信度 / beat 重复”等待人工复核问题

MVP hardening 规则：
- `objective` 与 `notes.open_questions` 不再被视为“必须填满”的字段；证据不足时可以留空或省略
- `evidence` 与 `review` 不是工业级 provenance 系统，而是首版最小证据绑定与人工复核锚点
- 首版不强制每个 beat 逐条绑定证据，避免 Schema 过重；先把追溯粒度控制在 scene 级别

### `validation`

作用：
- 把结构校验结果显式写进产物
- 便于前端和评委快速判断该 YAML 是否可靠

建议字段：

```yaml
validation:
  status: "passed"
  warnings:
    - "scene_004 缺少明确时间标记"
```

规范要求：
- `status`：必填，枚举 `passed` `failed`
- `warnings`：必填，字符串数组

说明：
- `passed` 只表示“当前通过结构校验，且未触发明显的内容级高风险规则”，不等于“语义质量已经可靠”
- 若 scene 出现明显重复、模板化占位、低置信度堆积等问题，后端应把具体风险写入 `warnings`，必要时降为 `failed`

## 为什么这样设计

### 1. 适合多阶段后端管线

该结构天然适合拆分为多个生成阶段：
- 先做 `source` 标准化
- 再抽 `characters` 和 `locations`
- 再规划 `scenes`
- 最后生成 `beats`

这比一次性输出整篇自由文本更稳定，也更能体现后端工程设计。

### 2. 同时适合程序校验和人工编辑

若结构过于扁平：
- 程序难以验证完整性
- 前端难以做局部展示

若结构过于复杂：
- 首版实现成本过高
- 用户修改成本上升

当前设计在“结构清晰”和“编辑友好”之间取平衡。

### 2.1 用 `evidence` 和 `review` 把“结构完整”与“内容可信”拆开

过去只有 `validation.status` 和 `warnings`，会让结果出现“字段齐全、YAML 合法，但人物 / 地点 / objective 明显不稳”时仍然看起来像通过。

加入 scene 级 `evidence` 与 `review` 后，MVP 可以更诚实地表达：
- 这一场戏主要依据哪章、哪条线索建立
- 哪些字段是低置信度推断，不该被误读为稳定事实
- 人工编辑时优先从哪里开始修

### 3. 保留来源章节映射

题目强调输入为 3 章以上小说文本，因此输出需要能回答：
- 这一场戏来自哪些章节？
- 改编时是否跳跃或合并了章节内容？

`source_chapters` 是首版必须保留的可追溯信息。

### 4. 用 `beats` 承载可拍摄片段

剧本不是章节摘要。用 `beats` 表达动作、对白、转场，有两个好处：
- 更贴近剧本可拍摄单元
- 便于后续扩展为卡片视图、场景编辑器或导出其他格式

但首版也明确限制：
- `beats` 不要求假装完整覆盖全部叙述
- 如果章节主要由内心戏或概述性叙述组成，允许 beats 较少，同时把风险写进 `review.issues` / `validation.warnings`

## 首版校验规则建议

最小校验规则：
- 顶层字段必须存在：`version` `source` `adaptation` `scenes` `validation`
- `source.chapter_count >= 3`
- `len(source.chapters) == source.chapter_count`
- `scenes` 非空
- 每个 `scene.id` 唯一
- 每个 `scene.source_chapters` 非空
- 每个 `scene.beats` 非空
- `dialogue` beat 必须包含 `character_id`
- 所有外键引用必须有效：`location_id`、`character_id`、`source_chapters`
- 若出现 `evidence.chapter_indexes`，其引用必须有效
- 若出现 `review.confidence`，其值必须为 `high` `medium` `low`
- validation 除结构校验外，还应补最小内容级审计：至少覆盖 scene objective / open question / beat 的明显重复与占位文案

## 示例片段

```yaml
version: "1.0"
source:
  title: "夜雨疑云"
  author: "示例作者"
  language: "zh-CN"
  chapter_count: 3
adaptation:
  style: "悬疑网剧"
  audience: "大众向"
  notes: ["强化外部冲突"]
characters:
  - id: "char_lin_qi"
    name: "林琪"
    aliases: []
    role: "protagonist"
    description: "年轻作者。"
locations:
  - id: "loc_old_apartment"
    name: "旧公寓"
    description: "陈旧、安静、带压迫感。"
scenes:
  - id: "scene_001"
    title: "深夜回家"
    source_chapters: [1]
    slugline:
      interior_exterior: "INT"
      location_id: "loc_old_apartment"
      time: "NIGHT"
    summary: "主角发现门锁异常。"
    objective: "确认门锁异常不是自己的错觉。"
    beats:
      - type: "action"
        content: "林琪站在门前，钥匙停在半空。"
      - type: "dialogue"
        character_id: "char_lin_qi"
        content: "不对。"
        emotion: "uneasy"
    notes:
      adaptation_reason: "压缩心理描写，增强可视动作。"
      open_questions: []
    evidence:
      chapter_indexes: [1]
      excerpt: "主角发现门锁异常。"
      cues: ["门锁异常", "夜归"]
    review:
      confidence: medium
      issues: []
validation:
  status: "passed"
  warnings: []
```
