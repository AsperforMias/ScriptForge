# Backend Architecture

## 目标

后端不是简单的“调一下模型然后返回字符串”，而是本项目的核心竞争力展示区。首版后端必须同时满足：
- 把 3 章以上小说稳定转成结构化 YAML
- 展示 Go 服务设计能力
- 展示中间件、任务编排、校验、持久化和可观测性
- 在 72h 内保持足够轻量和可交付

## 选定架构

采用单体 Go HTTP API + 进程内异步任务执行器：

```text
HTTP Client
  -> Router
  -> Middleware Chain
  -> Handler
  -> Job Service
  -> Job Store (SQLite)
  -> Pipeline Runner
  -> Artifact Store (local files)
  -> YAML Validator
```

这不是“伪微服务”，而是为 72h MVP 设计的务实架构：
- 接口清楚
- 状态可追踪
- 容易本地启动
- 便于评委理解
- 能体现工程设计而不是脚本拼接

## 目录建议

建议在当前骨架基础上按以下方向实现：

```text
backend/
  cmd/api/                  service entrypoint
  internal/config/          env/config loading
  internal/httpx/           handlers, middleware, response helpers
  internal/job/             job service, job state machine
  internal/ingest/          chapter validation and normalization
  internal/workflow/        outline/entities/scene planning rules
  internal/pipeline/        pipeline orchestration
  internal/screenplay/      YAML domain models and validation
  internal/storage/         SQLite + file artifact repositories
  internal/llm/             provider abstraction and adapters
  pkg/                      stable shared helpers only if needed
```

## Go 中间件栈

首版必须具备的中间件：
1. `RequestID`
2. `Recoverer`
3. `Timeout`
4. `AccessLog`
5. `BodyLimit`
6. `CORS`

建议顺序：

```text
RequestID -> Recoverer -> Timeout -> AccessLog -> BodyLimit -> CORS -> Handler
```

每个中间件的价值：
- `RequestID`：串联 job 创建、状态查询、错误日志
- `Recoverer`：避免 panic 直接打断演示
- `Timeout`：防止异常请求卡死
- `AccessLog`：展示结构化日志与调试能力
- `BodyLimit`：控制超长请求，避免小说全文直接拖垮服务
- `CORS`：保证本地前后端分离联调顺畅

## 任务模型

采用异步 job，而不是把生成链路塞进一个同步接口。

原因：
- 小说转剧本天然是长任务
- 前端需要展示阶段状态
- 任务模型更能体现后端工程能力
- 即使首版 worker 仍在单进程内，也比同步接口更接近真实系统

Job 状态：
- `queued`
- `running`
- `succeeded`
- `failed`

Pipeline 阶段：
- `ingest`
- `outline`
- `entities`
- `scene_planning`
- `screenplay_generation`
- `validation`
- `persistence`

## 数据与存储

SQLite 存：
- jobs
- job events / stage status
- artifact metadata

本地文件存：
- 原始章节输入
- 中间产物快照
- 最终 YAML

这样做的理由：
- 比纯内存更可靠
- 比接外部数据库更省时间
- 能展示“结果可追溯”和“产物可回放”

## LLM 与 deterministic 双模式

必须支持两种生成模式：
- `deterministic`
- `llm`

默认开发顺序：
1. 先做 `deterministic`
2. 再挂 `llm`

理由：
- 先把结构化合同跑通
- 先具备稳定回归样例
- 降低供应商和网络依赖对交付节奏的影响

## 为什么这套后端有竞争力

相对“直接调用大模型返回一段文本”的方案，本架构更有竞争力的点在于：
- 有明确任务状态与长任务处理模型
- 有阶段化 pipeline，而不是单函数黑盒
- 有 YAML Schema 校验闭环
- 有 Go 中间件和结构化日志，能体现服务端工程素养
- 有 deterministic 基线，降低 demo 风险
- 有 SQLite + artifact 持久化，便于回放、调试和展示

## 明确不做

首版不做：
- 外部消息队列
- 分布式 worker
- 鉴权系统
- 复杂租户模型
- 复杂缓存集群

这些对 72h 不是最优收益点，且会稀释本题核心价值。
