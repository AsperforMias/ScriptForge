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
- [`decision-log.md`](/Users/asperformias/Code/github/ScriptForge/docs/decision-log.md)
- [`api-contract.md`](/Users/asperformias/Code/github/ScriptForge/docs/api-contract.md)
- [`backend-pipeline.md`](/Users/asperformias/Code/github/ScriptForge/docs/backend-pipeline.md)

## 分支与 PR 规则

推荐分支前缀：
- `codex/backend/...`
- `codex/frontend/...`
- `codex/docs/...`

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
- [`.github/pull_request_template.md`](/Users/asperformias/Code/github/ScriptForge/.github/pull_request_template.md)

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

## 当前优先级约束

在首个实现阶段，优先级固定为：
1. 后端骨架
2. YAML Schema 程序化实现
3. 最小可运行生成链路
4. 前端接入
5. 演示和部署补强

除非文档更新明确改动，否则不要颠倒这一顺序。
