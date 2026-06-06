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
- 真实 `llm` 主链路已经可运行，但在当前 Prompt / Pipeline 设计下，输出质量仍会继承上游 grounding 的错误先验
- 当 provider 不可用时，`generation.mode=llm` 已会显式回退到 deterministic，并通过 warning 暴露这次不是实际 LLM 结果
- 当前最严重的真实质量问题仍是：
  - 人物抽取会把叙述碎片误当角色名
  - 地点抽取会被局部词触发，出现整章地点误判
  - scene objective / dialogue / open_questions 仍有强模板痕迹
  - action beat 更像“截断摘要”而不是可拍场景片段
  - 虽然 validation 已更诚实，但生成主链路仍未稳定达到“可信的初步可编辑 YAML 初稿”门槛

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
- `generation.mode` 默认值已切到 `llm`
- backend 现已自动读取 repo-root `.env.local`，`cd backend && go run ./cmd/api` 可直接启用本地 LLM 配置
- 前端现已恢复 `lastJobId` 与 workspace draft，刷新后左侧输入与右侧结果保持一致

## 尚未完成

### P0：主链路质量纠偏

- 收紧 `openai_compatible` 输入上下文，避免把 deterministic 伪角色、错地点、模板 objective 继续灌给 LLM
- 把真实中文三章节样本纳入主回归集，而不是只在 fixture 上看起来稳定
- 保留 deterministic，但仅作为 fallback / smoke baseline

### P1：真实输入可信度

- 收紧真实三章节输入下的人物抽取错误
- 收紧地点 / slugline 误判
- 降低模板化 objective / open_questions / dialogue
- 让 beat 更接近可拍动作，而不是截断摘要

### P1：validation 诚实度

- 继续提高 validation 对人物碎片、地点误判、模板化 scene 文案的拦截能力
- 在明显语义质量不足时，不再返回 `passed`

## 下一步优先级

优先级 1：
- 完成 `openai_compatible` Prompt / context 去污染，减少 LLM 继承 deterministic 错误先验

优先级 2：
- 把真实《雾港回声》样本接成主回归集，围绕角色 / 地点 / objective / open_questions 质量收敛

优先级 3：
- 继续提高 validation 对伪角色、地点误判、模板化 scene 文案的拦截能力

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
- 当前 blocker：真实三章节输入下，LLM 主链路仍被错误 grounding 先验污染

## 协作提醒

- 后续 session 若继续补 deterministic 题材模板，必须先证明这是比赛 ROI 最高的路径；默认视为偏题
- 若实现与本文件冲突，以本文件为准，并应优先修正文档已经指出的偏差
- 下一轮实现前，先确认 `decision-log.md`、`final-solution.md`、`backend-architecture.md`、`backend-pipeline.md` 与本文件保持一致
