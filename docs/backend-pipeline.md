# Backend Pipeline

## 目标

本文件把“小说转剧本”后端管线从概念描述收紧为阶段合同。后续实现必须遵循这些阶段输入输出，而不是把所有逻辑压进 handler。

## 总流程

```text
job created
  -> ingest
  -> outline
  -> entities
  -> scene_planning
  -> screenplay_generation
  -> validation
  -> persistence
  -> job succeeded
```

失败时：
- 任一阶段失败，job 状态置为 `failed`
- 记录失败阶段和错误信息
- 已产出的中间结果可选保留为调试快照

## Stage Contract

### 1. `ingest`

输入：
- `CreateJobRequest`

输出：
- `NormalizedSource`

职责：
- 校验章节数不少于 3
- 校验章节 index 连续
- 清洗空白字符
- 生成内部 source metadata

失败条件：
- 缺少章节
- 空章节
- 结构非法

### 2. `outline`

输入：
- `NormalizedSource`

输出：
- `OutlineBundle`

职责：
- 为每章生成摘要
- 抽取章节主冲突
- 识别跨章节主线

deterministic 首版要求：
- 至少能为每章生成一个简短 summary 占位结果

### 3. `entities`

输入：
- `NormalizedSource`
- `OutlineBundle`

输出：
- `EntityBundle`

职责：
- 抽人物
- 抽地点
- 建立人物/地点 ID

deterministic 首版要求：
- 可以从规则或占位逻辑生成最小合法人物和地点集合

### 4. `scene_planning`

输入：
- `NormalizedSource`
- `OutlineBundle`
- `EntityBundle`

输出：
- `ScenePlan`

职责：
- 把章节映射为若干 scene
- 为每个 scene 建立 `source_chapters`
- 给出 scene title、summary、objective、slugline 基础信息

首版最低要求：
- 至少每章映射到 1 个 scene
- 每个 scene 都必须可追溯到来源章节

### 5. `screenplay_generation`

输入：
- `ScenePlan`
- `EntityBundle`

输出：
- `ScreenplayDocument`

职责：
- 为每个 scene 生成 `beats`
- 组装完整 YAML 对象

首版最低要求：
- 每个 scene 至少 1 个 `action` beat
- 若出现 `dialogue` beat，必须带 `character_id`

### 6. `validation`

输入：
- `ScreenplayDocument`

输出：
- `ValidatedScreenplay`

职责：
- 运行 Schema 校验
- 填写 `validation.status` 与 `validation.warnings`
- 生成 YAML 字符串

失败条件：
- 缺顶层字段
- `chapter_count < 3`
- 外键引用无效
- scene / beat 结构不合法

### 7. `persistence`

输入：
- `ValidatedScreenplay`
- job metadata

输出：
- `PersistedArtifact`

职责：
- 写入 YAML 文件
- 更新 job 状态为 `succeeded`
- 存储产物元数据

## 内部接口建议

推荐后续实现以下接口，而不是直接耦合具体组件：

```text
type JobStore interface
type ArtifactStore interface
type Generator interface
type Validator interface
type PipelineRunner interface
```

其中：
- `Generator` 支持 `deterministic` 和 `llm`
- `Validator` 负责结构校验和 YAML 序列化

## 并发与执行模型

首版约束：
- 单服务进程内执行
- 同时允许有限数量 job 运行
- 使用简单 semaphore 或 worker pool 控制并发

原因：
- 足够展示后端调度意识
- 不引入消息队列或分布式复杂度

## 调试与观测

建议每个阶段记录：
- stage name
- started_at
- finished_at
- status
- warning_count
- error message

这些字段用于：
- 前端进度展示
- README / demo 说明
- 本地调试
