# Codex 5.3 Fog Harbor Hardening Prompt

你当前在 `ScriptForge` 仓库中继续执行“比赛导向的 llm-first 小说转 YAML 剧本 MVP”纠偏。

## 先读这些文件

1. `/Users/asperformias/Code/github/ScriptForge/docs/final-solution.md`
2. `/Users/asperformias/Code/github/ScriptForge/docs/backend-pipeline.md`
3. `/Users/asperformias/Code/github/ScriptForge/docs/implementation-progress.md`
4. `/Users/asperformias/Code/github/ScriptForge/backend/internal/llm/openai_compatible.go`
5. `/Users/asperformias/Code/github/ScriptForge/backend/internal/llm/openai_compatible_test.go`
6. `/Users/asperformias/Code/github/ScriptForge/backend/internal/testutil/fog_harbor_echo_input.go`

## 当前已知事实

- 项目路线已经纠偏到 `llm-first`
- `deterministic` 只保留为 fallback / smoke baseline
- backend 已自动读取 repo-root `.env.local`
- 《雾港回声》已经被接成主回归样本
- 当前主问题不是“没接上 LLM”，而是“LLM 仍会继承错误 grounding 先验”

## 当前主要风险

1. `openai_compatible` 的上下文仍可能过重，导致真实 provider 超时或继续抄写低质量 hints
2. 《雾港回声》真实 provider 路径下，仍需继续收敛：
   - 角色误识别
   - 地点误判
   - 模板化 objective / open_questions
   - beat 仍偏摘要
3. `validation` 目前更诚实了，但还不够像真正的质量闸门

## 这轮必须完成的目标

### 目标 1：把《雾港回声》作为真实主验收样本继续收紧

- 不要再围绕 sample fixture 讲“看起来通过”
- 优先使用《雾港回声》验证真实链路
- 若真实 provider 输出仍差，优先修 Prompt / context / postprocess / validation，而不是继续扩 deterministic 题材模板

### 目标 2：继续瘦身 `openai_compatible` 输入上下文

- 检查 `buildProviderContext`
- 目标是“raw chapter text + source-grounded hints”
- 不要重新把 `req.Entities` / `req.Plan` 整包塞回 prompt
- 如果 context 太重导致超时，优先减无效字段，而不是重新依赖 deterministic 主生成

### 目标 3：提高《雾港回声》的最小通过标准

至少满足：
- 不再出现 `像是`、`没有立刻` 这类伪角色
- 章节地点不再整体退化为 `房间`
- 明显模板 objective / open_questions 能被清空或降级，而不是留在结果里
- `validation` 能诚实暴露低质量结果

## 执行顺序

1. 先跑：
   - `cd /Users/asperformias/Code/github/ScriptForge/backend && go test ./internal/llm ./internal/pipeline`
2. 用真实 provider 跑一次《雾港回声》
3. 如果超时：
   - 先减 prompt/context
   - 再评估是否要调高 `LLM_REQUEST_TIMEOUT`
4. 如果结果仍有伪角色 / 错地点 / 模板 scene 文案：
   - 优先修 `openai_compatible.go`
   - 其次修 `screenplay/quality.go`
5. 每次改动后都回归：
   - `go test ./...`
   - 再跑一次《雾港回声》真实链路

## 严格约束

- 不要把项目重新带回 “deterministic 规则改编器”
- 不要靠新增 fixture 掩盖真实样本质量问题
- 不要把 `validation passed` 当成质量成功
- 不要默认字段填满优于字段留空

## 交付要求

完成后必须给出：

1. 《雾港回声》真实链路最新结果摘要
2. 仍存在的问题
3. 根因判断
4. 下一轮最高 ROI 的 1-3 项修复
