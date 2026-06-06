# Implementation Progress

## 当前状态

更新时间：2026-06-06

当前分支对齐目标（`docs/mvp-scope-realignment`）：
- 先修正文档基线，再继续实现
- 明确当前项目的核心偏差不是“前端没打通”，而是“主生成路线被做成了 deterministic 规则改编器”
- 把后续实现重新收束到“比赛导向的 LLM-first YAML 初稿工具”

当前更诚实的项目判断：
- 项目已经具备可运行的任务化 API、前端工作台、YAML 导出和基础 schema 校验能力
- 项目尚未具备“对真实自定义三章节输入稳定产出可信初稿”的能力
- 当前最大问题不是功能缺失，而是主链路质量方向跑偏

## 已确认的核心问题

通过真实自定义输入（例如《雾港回声》这类三章节悬疑文本）已经确认：
- 自定义输入链路本身是通的，前后端并未把用户锁死在 sample preset 或 fixture 上
- 但默认 `generation.mode=deterministic` 产物仍存在明显错误：
  - 人物抽取会把叙述碎片误当角色名
  - 地点抽取会被局部词触发，出现整章地点误判
  - scene objective / dialogue / open_questions 仍有强模板痕迹
  - action beat 更像“截断摘要”而不是可拍场景片段
  - validation 仍可能在明显语义不足时返回 `passed`

这些问题说明：
- 当前最需要纠正的是产品主链路与文档方向
- 继续围绕 deterministic 扩题材规则、扩 fixture、扩 demo 话术，ROI 很低

## 当前方向纠偏

修正后的主方向：
- `llm` 是主生成链路
- `deterministic` 仅保留为 fallback / mock / smoke baseline
- 规则层职责是 grounding、normalize、validate、fallback
- 结果目标是“稀疏但可信的 YAML 初稿”，不是“字段填满的伪完整剧本”

不再鼓励的方向：
- 继续把 `backend/internal/workflow/deterministic.go` 扩成跨题材模板库
- 为了让 deterministic 更像成品而持续追加题材 hardcode
- 用越来越多 fixture 掩盖真实自定义输入下的质量问题

## 已完成

- docs-first 项目骨架与比赛范围文档
- Go 后端任务化 API、SQLite/artifact 持久化与阶段状态
- YAML 领域模型、序列化与基础结构校验
- 前端工作台：输入、轮询、结果查看、编辑、导出
- `generation.mode=llm` provider abstraction
- vendor-neutral `openai_compatible` 适配器
- `openai_compatible` loose YAML 归一化
- scene-level `evidence` / `review` schema hardening
- validation 最小内容审计补入，减少“结构通过掩盖明显模板化结果”

## 尚未完成

### P0：主链路纠偏

- 把默认生成模式从 `deterministic` 切到 `llm`
- 把 README、demo、自检、smoke 的主叙事改成 `llm-first`
- 保留 deterministic，但降级为 fallback / smoke baseline

### P1：真实输入可信度

- 收紧真实三章节输入下的人物抽取错误
- 收紧地点 / slugline 误判
- 降低模板化 objective / open_questions / dialogue
- 让 beat 更接近可拍动作，而不是截断摘要

### P1：validation 诚实度

- 继续提高 validation 对人物碎片、地点误判、模板化 scene 文案的拦截能力
- 在明显语义质量不足时，不再返回 `passed`

### P1：前端真实输入连续性

- 当前页面会恢复 `lastJobId`，但输入表单仍可能回落到默认 sample preset
- 这会干扰调试、复盘和演示，必须补全“结果恢复时同步恢复输入草稿”的文档与实现约束

## 下一步优先级

优先级 1：
- 完成文档纠偏，统一所有核心文档对主链路的定义
- 把 `GENERATION_MODE_DEFAULT`、README 自检步骤、demo 叙事改到 `llm-first`

优先级 2：
- 调整实现，使 `llm` 成为默认主链路
- deterministic 只保留最小合法结构与 fallback 责任

优先级 3：
- 用真实中文三章节输入建立新的主验收集
- fixture 继续保留，但不再作为“主成功证据”

## 里程碑状态

阶段 0：初始化与对齐
- 状态：已完成

阶段 1：后端骨架
- 状态：已完成

阶段 2：任务化链路与 YAML 产物
- 状态：已完成

阶段 3：前端工作台
- 状态：已完成

阶段 4：方向纠偏与可信度收敛
- 状态：进行中
- 当前 blocker：主生成路线与文档叙事不一致

## 协作提醒

- 后续 session 若继续补 deterministic 题材模板，必须先证明这是比赛 ROI 最高的路径；默认视为偏题
- 若实现与本文件冲突，以本文件为准，并应优先修正文档已经指出的偏差
- 下一轮实现前，先确认 `decision-log.md`、`final-solution.md`、`backend-architecture.md`、`backend-pipeline.md` 与本文件保持一致
