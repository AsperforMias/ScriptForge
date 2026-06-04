# Backend Tech Stack

## 目标

本文件把后端从“架构方向”收紧到“实现约束”。后续 human/agent session 在没有充分理由更新文档前，应按本文档落地，不再自行重选主库。

## Go 版本与模块

锁定要求：
- Go 版本：`1.25.x`
- 模块根目录：`backend/`
- module path：后续以仓库实际远端为准，但必须在 `backend/go.mod` 中单独初始化

原因：
- 使用较新的标准库能力
- 避免前后端混在同一 module
- 当前锁定的 `modernc.org/sqlite v1.51.0` 会把 module 的 `go` 指令提升到 `1.25.0`，因此文档与实现统一到 `1.25.x`

## 基础库选型

### HTTP Router

- 选型：`github.com/go-chi/chi/v5`
- 用途：路由、中间件组合、URL 参数提取

原因：
- 轻量
- 中间件组合自然
- 对当前单体 API 足够
- 比重量级框架更适合在 72h 内展示“自己搭的服务结构”

### Logging

- 选型：Go 标准库 `log/slog`
- 输出格式：JSON

原因：
- 标准库依赖少
- 足够支撑 request/job/stage 级结构化日志
- 比额外引入复杂日志生态更稳

### YAML

- 选型：`gopkg.in/yaml.v3`
- 用途：YAML 序列化与反序列化

原因：
- 生态稳定
- 足以支撑首版 Schema

### SQLite Driver

- 首选：`modernc.org/sqlite`
- 备选：`github.com/mattn/go-sqlite3`

约束：
- 首版默认优先 `modernc.org/sqlite`，避免 CGO 依赖增加本地演示复杂度
- 若因兼容性改用 `mattn/go-sqlite3`，必须更新本文档和 README 的依赖说明

### Database Access

- 选型：标准库 `database/sql`
- 不引入 ORM

原因：
- 表结构简单
- 更利于展示对 schema 和 SQL 的掌控
- 避免 72h 项目为 ORM 适配消耗时间

### Config Loading

- 选型：标准库 `os` + `strconv` + 自定义 `internal/config`
- 不引入 `viper`

原因：
- 配置项数量有限
- 减少隐藏行为
- 更可控

### UUID / ID Strategy

- 首版建议：自定义前缀 ID
- 形式：
  - `job_<suffix>`
  - `req_<suffix>`
  - `scene_<index>`
  - `char_<slug>`
  - `loc_<slug>`

约束：
- 不强制引入 UUID 库作为阶段 1 blocker
- 若后续需要全局唯一随机 ID，可补充轻量实现，但命名格式仍保持可读

## 目录落地约束

建议落地目录：

```text
backend/
  cmd/api/main.go
  internal/config/
  internal/httpx/
    handler/
    middleware/
    render/
  internal/job/
  internal/ingest/
  internal/workflow/
  internal/pipeline/
  internal/screenplay/
  internal/storage/
    sqlite/
    artifact/
  internal/llm/
```

约束：
- `httpx` 只负责 HTTP 相关逻辑，不放业务流程
- `job` 负责任务状态机与 job service
- `pipeline` 负责 orchestration，不直接处理 SQLite 细节
- `storage` 负责 DB 和文件系统实现

## 配置项

首版必须支持以下环境变量：

| Name | Required | Default | Purpose |
| --- | --- | --- | --- |
| `APP_ENV` | no | `development` | 运行环境标识 |
| `HTTP_ADDR` | no | `:8080` | HTTP 监听地址 |
| `SQLITE_PATH` | no | `./tmp/scriptforge.db` | SQLite 数据库路径 |
| `ARTIFACT_DIR` | no | `./tmp/artifacts` | YAML 与输入快照目录 |
| `HTTP_READ_TIMEOUT` | no | `10s` | 读超时 |
| `HTTP_WRITE_TIMEOUT` | no | `30s` | 写超时 |
| `REQUEST_BODY_LIMIT_BYTES` | no | `4194304` | 请求体大小限制，默认 4MB |
| `JOB_MAX_CONCURRENCY` | no | `2` | 同时运行 job 数 |
| `CORS_ALLOW_ORIGIN` | no | `*` | 本地联调用 |
| `GENERATION_MODE_DEFAULT` | no | `deterministic` | 默认生成模式 |
| `LLM_PROVIDER` | no | `disabled` | LLM provider selector，首版支持 `disabled` / `mock` / `openai_compatible` |
| `LLM_MODEL` | no | `` | 预留模型名 |
| `LLM_BASE_URL` | no | `` | 预留 provider base URL |
| `LLM_API_KEY` | no | `` | 预留 provider API key |
| `LLM_REQUEST_TIMEOUT` | no | `45s` | 预留 LLM 调用超时 |

约束：
- 所有配置必须在启动时打印结构化摘要日志，但不得输出敏感信息
- 若缺省值生效，应在日志中可见

### 本地凭证约定

本地真实 provider 调试时：
- 先执行 `cp .env.local.example .env.local`
- 在仓库根目录维护 `.env.local`
- 通过 `set -a && source .env.local && set +a` 导出环境变量
- `.env.local` 必须保持 gitignored
- 真实 `LLM_API_KEY` 不得进入 commit、PR 描述、README 示例或日志

当前已验证的兼容接入方式：
- `LLM_PROVIDER=openai_compatible`
- `LLM_BASE_URL=https://api.deepseek.com`
- `LLM_MODEL=deepseek-v4-flash`

说明：
- 以上模型选择仅用于低成本功能链路验证
- 最终 demo 模型可在不改后端接口的前提下切换

## SQLite Schema

首版最小表结构：

### `jobs`

字段：
- `id TEXT PRIMARY KEY`
- `source_title TEXT NOT NULL`
- `status TEXT NOT NULL`
- `current_stage TEXT NOT NULL`
- `progress_percent INTEGER NOT NULL DEFAULT 0`
- `generation_mode TEXT NOT NULL`
- `warning_count INTEGER NOT NULL DEFAULT 0`
- `warnings_json TEXT NOT NULL DEFAULT '[]'`
- `error_message TEXT NOT NULL DEFAULT ''`
- `input_snapshot_path TEXT NOT NULL`
- `result_yaml_path TEXT NOT NULL DEFAULT ''`
- `created_at TEXT NOT NULL`
- `updated_at TEXT NOT NULL`

约束：
- `status` 枚举必须与 `api-contract.md` 一致
- `current_stage` 枚举必须与 `api-contract.md` 一致

### `job_stages`

字段：
- `job_id TEXT NOT NULL`
- `stage_name TEXT NOT NULL`
- `status TEXT NOT NULL`
- `warning_count INTEGER NOT NULL DEFAULT 0`
- `error_message TEXT NOT NULL DEFAULT ''`
- `started_at TEXT NOT NULL DEFAULT ''`
- `finished_at TEXT NOT NULL DEFAULT ''`

约束：
- 复合唯一键：`(job_id, stage_name)`

### `artifacts`

字段：
- `job_id TEXT PRIMARY KEY`
- `yaml_path TEXT NOT NULL`
- `yaml_size_bytes INTEGER NOT NULL`
- `created_at TEXT NOT NULL`

### Migration Strategy

首版要求：
- 使用项目内 SQL 文件或内嵌 SQL 常量
- 不引入复杂 migration framework

建议路径：
- `backend/internal/storage/sqlite/schema.sql`

## Artifact Directory Layout

首版约定：

```text
tmp/
  artifacts/
    <job_id>/
      input.json
      normalized_source.json
      provider_debug.json
      screenplay.yaml
```

约束：
- 每个 job 独立目录
- `input.json` 保存原始请求快照
- `normalized_source.json` 便于调试 ingest 结果
- `provider_debug.json` 在 `generation.mode=llm` 时保存 provider 名称、模型、解析模式与原始返回内容
- `screenplay.yaml` 是最终权威产物

## HTTP Status 与 Error Code 映射

固定映射：

| HTTP Status | Error Code | Meaning |
| --- | --- | --- |
| `400` | `invalid_input` | 请求结构不合法、章节不足、字段缺失 |
| `404` | `job_not_found` | job 不存在 |
| `409` | `job_not_ready` | job 尚未完成但请求结果/导出 |
| `500` | `generation_failed` | pipeline 执行失败 |
| `500` | `internal_error` | 非业务预期错误 |

约束：
- 不要把所有错误都返回 `500`
- `job_not_ready` 不应使用 `404`

## Middleware 落地要求

### `RequestID`

- 从 header `X-Request-ID` 读取，若没有则生成
- 响应 header 必须回写 `X-Request-ID`
- `meta.request_id` 与该值保持一致

### `Recoverer`

- 捕获 panic
- 返回 `internal_error`
- 记录 stack trace 到日志

### `Timeout`

- handler 级超时
- 超时后返回统一错误响应

### `AccessLog`

至少记录：
- request_id
- method
- path
- status
- duration_ms
- remote_addr

### `BodyLimit`

- 对 `POST /jobs` 生效
- 超限直接返回 `400 invalid_input`

### `CORS`

- 本地阶段允许配置化 origin
- 不需要实现复杂凭证策略

## Job Runner 约束

执行模型：
- handler 创建 job 记录后立即返回 `202`
- 后台 goroutine / worker pool 拉起执行
- 同时运行数量受 `JOB_MAX_CONCURRENCY` 控制

约束：
- 不允许在 HTTP handler 内直接跑完整 pipeline 再返回
- worker panic 必须被恢复并把 job 标记为 `failed`

## Testing Matrix

首版至少要有以下测试：

### Unit Tests

- ingest 校验：少于 3 章、章节 index 不连续、空文本
- screenplay validator：合法 YAML、非法外键、缺字段
- job state transitions：`queued -> running -> succeeded/failed`

### Integration Tests

- `POST /jobs` 成功创建任务
- `GET /jobs/{id}` 返回阶段状态
- `GET /jobs/{id}/result` 在未完成时返回 `409`
- deterministic 模式可生成合法 YAML
- HTTP 测试优先直接走 `ServeHTTP`，避免依赖本地监听端口

### Fixture Requirements

- 至少一组合法 3 章输入
- 至少一组非法输入
- 至少一组期望 YAML 输出

### Sandbox Verification Note

- agent / CI 场景下执行 Go 校验时，优先设置 `GOCACHE=/tmp/scriptforge-gocache`
- 构建校验建议使用：`go build -o /tmp/scriptforge-api ./cmd/api`

## Logging Key Conventions

日志字段建议固定：
- `request_id`
- `job_id`
- `stage`
- `status`
- `duration_ms`
- `component`

原因：
- 便于后续 agent 统一日志格式
- 便于 demo 时展示链路

## Non-Negotiable Rules

- 不引入 ORM
- 不引入外部 MQ
- 不把 pipeline 直接塞进 handler
- 不把 YAML 仅作为展示字符串，而必须保留结构化对象
- 不新增会显著增加本地运行门槛的重依赖，除非文档先更新
- 在真实供应商未锁定前，允许保留 vendor-neutral 的 `openai_compatible` 适配器，但不得把项目耦合到特定厂商 SDK
