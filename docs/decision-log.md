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
- 决策：先实现 deterministic/rule-based 生成链路，再接真实 LLM
- 原因：
- 避免模型供应商未定导致主链路阻塞
- 先保证“合法 YAML 输出”可演示
- 为回归测试保留稳定基线

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

- 状态：`deferred`
- 决策：由前端 teammate 主导选择
- 约束：只要满足 `frontend.md` 和 `api-contract.md`，框架不是当前 blocker

### D-015 前端默认落地路径

- 状态：`locked`
- 决策：在前端 teammate 未明确要求其他框架前，默认采用 `Vite + React + TypeScript + TanStack Query + React Hook Form` 的单页工作台方案
- 原因：
- 对任务化 API 轮询足够直接
- 对 Codex session 足够明确，可直接执行
- 不会因为过重 UI 技术选型拖慢 MVP

### D-014 公网部署

- 状态：`deferred`
- 决策：优先保证本地演示，公网部署作为加分项而非首个 blocker

## Human-Needed

当前无必须立即阻塞开发的人类决策项。

如果后续出现以下情况，必须升级为 `human-needed`：
- 前端 teammate 已明确要求固定某个框架或组件体系
- 需要接入付费模型服务并涉及成本/账号
- 需要把单体服务改为多进程或多服务
