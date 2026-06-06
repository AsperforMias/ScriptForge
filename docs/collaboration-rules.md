# Collaboration Rules

## 核心原则

本仓优先展示：
- 工程判断清晰
- 持续交付
- 小步提交
- 前后端协作低摩擦
- 文档驱动，而不是临场猜测

因此后续协作遵循：
- 先更新文档，再做大改
- 一次只交付一件事
- 主分支尽量保持可运行或至少可验证
- 已锁定决策以 `decision-log.md` 为准
- 阶段完成定义以 `milestones.md` 为准

## 目录协作约定

目录职责：
- `backend/`：后端主责区域
- `frontend/`：前端主责区域
- `docs/`：范围、进度、规则、Schema 权威来源
- `scripts/`：工具脚本
- `testdata/`：样例与回归数据

尽量减少冲突：
- 前后端分别在各自目录内完成主要改动
- 跨目录大改动必须先更新文档说明原因
- 共享协议变更时，先改 `docs/` 再改实现

共享协议与架构约束优先参考：
- [`decision-log.md`](decision-log.md)
- [`backend-tech-stack.md`](backend-tech-stack.md)
- [`api-contract.md`](api-contract.md)
- [`backend-pipeline.md`](backend-pipeline.md)

## 分支与 PR 规则

推荐分支前缀：
- `backend/...`
- `frontend/...`
- `docs/...`

分支命名补充：
- 不使用 `codex/` 前缀，避免把工具来源混进仓库分支语义
- 分支名应直接表达改动范围或目标，例如 `backend/llm-hardening`、`frontend/workspace-polish`、`docs/pr-rules-align`

PR 规则：
- 每个 PR 只做一件事
- PR 尽量小，便于评委快速理解
- 大功能拆成多个 PR，例如：
  - 初始化后端骨架
  - 增加 YAML Schema 校验
  - 接入生成接口
  - 增加结果导出

PR 标题模板建议：
- `docs: define screenplay YAML schema`
- `backend: add job creation API skeleton`
- `frontend: add multi-chapter input form`

PR 描述必须包含：
- 本 PR 做了什么
- 为什么现在做
- 如何验证
- 依赖或风险
- 若复用历史代码，写明来源

仓库中已提供：
- [`.github/pull_request_template.md`](../.github/pull_request_template.md)

PR 工作流：
1. 从最新 `main` 切新分支，不直接在 `main` 上开发
2. 在 feature branch 上完成单一目标改动
3. 本地验证通过后先 `push` 分支
4. 基于该分支向 `main` 发起 PR
5. PR 描述按模板补全：变更内容、原因、验证方式、风险
6. 若平台与权限允许，再执行 review / approve
7. 合并后本地切回 `main`
8. 拉取最新 `main`
9. 再从最新 `main` 切下一条 feature branch

执行约束：
- `main` 只接受通过 PR 合并的改动
- 不在同一条 feature branch 上串行堆多个无关功能
- 若一个 PR 已合并，后续开发默认从最新 `main` 新开分支
- 若使用 AI agent 执行该流程，需在最终说明中明确当前 branch、PR 目标和验证结果
- GitHub 平台上 PR 作者不能批准自己的 PR；若仓库没有第二位 reviewer，可记录该限制并基于自检后合并
- 若发现本地或远端已有 `codex/...` 旧分支，后续新工作不要继续沿用该命名

## Commit 规则

commit 目标：
- 体现连续开发过程
- 保持变更粒度清楚
- 避免“大包提交”

建议节奏：
- 每完成一个可独立说明的小步就提交
- 每天都应有持续提交痕迹
- 不要等到最后一天集中导入全部代码

commit 信息建议：
- `docs: summarize competition constraints`
- `chore: initialize repository layout`
- `backend: add screenplay domain models`
- `frontend: add chapter input workflow`

不推荐：
- `update`
- `fix stuff`
- `final`

## 两人协作建议

后端主责：
- API 设计
- 生成管线
- YAML 结构与校验
- 样例与测试

前端主责：
- 输入页面
- 状态流转
- 结果展示与编辑

交界面：
- API 契约
- YAML 显示与导出格式
- 错误状态和阶段状态文案

为降低沟通成本：
- 先固定字段和接口，再分别实现
- 每次接口变更都更新 `docs/frontend.md` 或相关文档

## Agent 参与规则

AI agent 可以用于：
- 代码生成
- 重构
- 文档整理
- 测试样例补全

但必须遵守：
- 不得跳过 `docs/` 直接猜测需求
- 若发现文档缺口，应先补文档
- 所有进度更新优先写入 `implementation-progress.md`
- 不得为了赶工牺牲 PR/commit 可解释性
- 若 `decision-log.md` 将某项标为 `human-needed`，必须停下询问
- 后续 agent 新建分支时，不使用 `codex/...` 命名
- 在沙箱或 CI 场景下做 Go 校验时，优先使用 `GOCACHE=/tmp/scriptforge-gocache`
- 本地构建校验产物优先输出到 `/tmp/scriptforge-api`，不要在仓库内生成 `backend/api` 这类临时二进制
- 本地 provider 凭证统一放在 repo-root `.env.local`，不得提交、不得写进 PR 描述、不得出现在评审可见文档中

## 当前优先级约束

当前阶段的优先级不再固定在初始化顺序，而以 `implementation-progress.md` 为准。

截至 2026-06-05，默认优先级为：
1. demo 视频与演示稿素材收束
2. fixture / smoke-check / provider 兼容回归继续补强
3. 视时间决定是否补可选公网演示，但这不是硬性要求；优先保证 README、本地启动流程与 demo 可复现

若后续阶段变化，应先更新 `implementation-progress.md`，再调整执行顺序。
