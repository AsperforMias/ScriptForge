# Decision Log

## 用途

本文件记录已经锁定的实现决策，避免后续 human/agent session 反复重选方案。

规则：
- 标记为“locked”的内容，默认不得绕过
- 标记为“deferred”的内容，可以暂缓，但不能阻塞 MVP
- 标记为“human-needed”的内容，agent 必须停下询问

## Locked

### D-001 仓库形态

- 状态：`locked`
- 决策：单仓，目录分为 `backend/`、`frontend/`、`docs/`
- 原因：满足双人协作、统一评审入口、降低仓库管理成本

### D-002 后端语言

- 状态：`locked`
- 决策：后端主实现使用 Go
- 原因：你的主栈是 Go，且本题更适合展示服务化、管线编排、校验、任务状态和中间件能力

### D-003 后端部署形态

- 状态：`locked`
- 决策：首版采用单体 HTTP API 服务，不拆多服务
- 原因：72 小时内需要优先保证可交付性与可复现性，单体更适合

### D-004 任务模型

- 状态：`locked`
- 决策：采用“HTTP API + 进程内异步 job runner”
- 原因：
- 保留任务状态、阶段进度和结果查询能力
- 比同步单接口更能体现后端设计
- 比外部 MQ/队列更轻量，适合 72h MVP

### D-005 数据持久化

- 状态：`locked`
- 决策：SQLite 持久化任务元数据，本地文件系统保存输入文本与 YAML 产物
- 原因：本地可运行、无需额外基础设施、足够支撑演示和回归

### D-006 核心输出

- 状态：`locked`
- 决策：YAML 是唯一权威产物格式，JSON 仅用于 API 包装
- 原因：赛题强制 YAML，且 YAML 更适合人工编辑和展示结构化价值

### D-007 首版生成策略

- 状态：`locked`
- 决策：首版主生成链路必须是 `llm-first`，deterministic 仅保留为 fallback / mock / smoke baseline
- 原因：
- 赛题要的是“小说 -> 可编辑 YAML 初稿”，不是“跨题材规则改编引擎”
- 继续把 deterministic 扩成主生成器，会在 72h 范围内快速演化成高成本、低泛化的模板系统
- LLM 更适合承担长文本抽取与结构化改写；规则层应聚焦 grounding、normalize、validate 和 fallback
- deterministic 仍保留有价值：可做离线 smoke、失败演示和最小结构回退，但不应主导产品叙事

### D-008 Go HTTP 路由与中间件

- 状态：`locked`
- 决策：必须实现显式中间件链，至少包括：
- request ID
- panic recovery
- timeout
- structured logging
- request body size limit
- CORS
- 原因：这部分是后端竞争力展示点，且对调试、演示和可靠性直接有价值

### D-009 Pipeline 阶段

- 状态：`locked`
- 决策：固定为以下顺序：
1. ingest
2. outline
3. entities
4. scene_planning
5. screenplay_generation
6. validation
7. persistence

### D-010 前后端接口形态

- 状态：`locked`
- 决策：前端对接任务化接口，不以同步 `generate` 接口为正式协议
- 原因：更符合“查看进度、获取结果、导出 YAML”的产品路径

### D-011 后端主库与实现约束

- 状态：`locked`
- 决策：后端库选型、配置项、SQLite schema、artifact 目录、错误映射、测试矩阵以 `backend-tech-stack.md` 为准
- 原因：避免不同 human/agent session 在实现时各自重选主库或重写落地规则

## Deferred

### D-012 模型供应商

- 状态：`deferred`
- 决策：保持接口抽象，待接入阶段再定具体供应商
- 约束：在确定前，不得让主链路依赖某一云厂商 SDK

### D-013 前端框架

- 状态：`locked`
- 决策：首版前端默认方案固定为 `Vite + React + TypeScript + TanStack Query + React Hook Form` 的单页工作台
- 原因：
- 该方案已被文档和当前实现共同采用
- 对任务化 API、轮询和 YAML-first 结果区足够直接
- 能避免 72h 项目在前端技术选型上继续反复摇摆

### D-015 前端默认落地路径

- 状态：`locked`
- 决策：继续沿用 `Vite + React + TypeScript + TanStack Query + React Hook Form` 的单页工作台方案，并以当前 `frontend/` 骨架为后续 PR 基线
- 原因：
- 对任务化 API 轮询足够直接
- 对 Codex session 足够明确，可直接执行
- 不会因为过重 UI 技术选型拖慢 MVP

### D-016 自定义章节输入验收路径

- 状态：`locked`
- 决策：首版必须支持用户直接粘贴 / 手工录入自己的小说章节完成完整链路；sample preset 仅作为演示加速器，不能成为唯一可验证路径
- 原因：
- 赛题要求是“输入不少于 3 个章节的小说文本”，而不是“只能运行仓库自带样例”
- 若只对 fixture / preset 友好，会削弱作品在真实输入下的可信度，也不利于评委快速判断系统是否真的可用
- 后续 smoke-check、README 自检步骤与前后端文案都必须围绕这一验收路径保持一致

### D-017 文档优先纠偏

- 状态：`locked`
- 决策：当实现已经明显偏向“规则引擎化”而偏离赛题目标时，必须先修正文档基线，再继续写代码
- 原因：
- 本仓以 `docs/` 为最高优先级项目上下文
- 若不先修正文档，后续 human/agent session 会继续沿错误方向补 deterministic 规则、fixture 和 demo 文案
- 这类偏题会直接消耗比赛时长，却不提升“小说 -> 可编辑 YAML 初稿”的核心完成度

### D-018 真实三章节输入的主判断策略

- 状态：`locked`
- 决策：在真实三章节小说输入的主链路上，人物理解、场景边界判断和剧本初稿生成以 `llm` 为主；本地规则层只保留最小职责，不再继续做重规则提取
- 原因：
- 当前比赛时间紧张，继续扩人物/地点/scene 的本地规则库，ROI 明显低于直接提升真实 `llm` 主链路质量
- 赛题要的是“真实小说文本 -> 可编辑 YAML 初稿”的可演示完成度，而不是在 72h 内做出可泛化的规则改编器
- 本地规则层仍然有价值，但职责应收缩到：输入校验、轻量 grounding hints、schema 校验、normalize、质量告警和 provider 失败时的 fallback
- 当前不引入重量级 RAG / 向量检索链路；主证据来源仍是用户提供的原始章节文本本身

### D-014 公网部署

- 状态：`deferred`
- 决策：优先保证本地演示，公网部署作为可选加分项而非硬性要求
- 依据：主办方最新说明仅要求公开可访问的代码仓库、demo 视频和 README；对开发语言、部署方式和产品形态均不做限制，且评审会实际试用产品但不限定必须提供公网 Web 地址

## Human-Needed

当前无必须立即阻塞开发的人类决策项。

如果后续出现以下情况，必须升级为 `human-needed`：
- 前端 teammate 已明确要求固定某个框架或组件体系
- 需要接入付费模型服务并涉及成本/账号
- 需要把单体服务改为多进程或多服务
