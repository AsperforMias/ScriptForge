# Implementation Progress

## 当前状态

更新时间：2026-06-06

当前分支对齐目标（`docs/mvp-scope-realignment`）：
- 先修正文档基线，再继续实现
- 明确当前项目的核心偏差不是“前端没打通”，而是“主生成路线被做成了 deterministic 规则改编器”
- 把后续实现重新收束到“比赛导向的 LLM-first YAML 初稿工具”

当前更诚实的项目判断：
- 项目已经具备可运行的任务化 API、前端工作台、YAML 导出和基础 schema 校验能力
- 项目已经具备“对真实自定义三章节输入生成可演示 YAML 初稿”的能力
- 项目尚未具备“对真实自定义三章节输入稳定产出高一致性、可信初稿”的能力
- 主链路方向已基本纠偏回 `llm-first`，当前主要问题不再是路线偏题，而是输出稳定性和内容收敛

当前新增判断：
- 对于比赛剩余时间，继续强化“规则 + 混合提取”不是最高 ROI 路径
- 当前更可行的纠偏方案是：`弱规则约束 + DeepSeek 主判断/主生成`
- 本地规则层应继续减重，避免把 deterministic 的摘要、角色候选、地点候选当成高置信输入重新污染 `llm`

## 已解决或已明显缓解

通过真实自定义输入（例如《雾港回声》这类三章节悬疑文本）已经确认：
- 自定义输入链路本身是通的，前后端并未把用户锁死在 sample preset 或 fixture 上
- 主生成路线已从 `deterministic-first` 拉回 `llm-first`
- 真实 `llm` 主链路已经可运行，并能稳定完成 `screenplay_generation -> validation -> persistence`
- 《雾港回声》真实输入已可回到 canonical parse，不再依赖 deterministic fallback 才能出结果
- 伪角色、整章 `房间` 级地点误判、scene_001 的 `INT/EXT` 回退等早期硬伤已明显缓解
- 当前前端实测已确认“切换为空白手工输入 -> 录入《雾港回声》三章 -> 提交生成 -> 自动载入 YAML 初稿”这条主验收路径可以直接跑通

这些结论说明：
- 主链路与文档方向的纠偏已基本完成
- 项目已经达到“可用于比赛演示”的最低可交付状态

## 仍未解决的核心问题

通过真实自定义输入（例如《雾港回声》这类三章节悬疑文本）已经确认：
- 真实 `llm` 主链路虽然可运行，但输出质量仍会继承上游 grounding 的部分错误先验
- 当 provider 不可用时，`generation.mode=llm` 已会显式回退到 deterministic，并通过 warning 暴露这次不是实际 LLM 结果
- 当前最严重的真实质量问题是：
  - scene 切分数量仍会随 provider 输出波动
  - objective / dialogue / open_questions 仍偶发模板痕迹
  - action beat 有时仍更像“截断摘要”而不是可拍场景片段
  - 低证据场景的 review / validation 诚实度还需要继续收紧
  - 虽然 validation 已更诚实，但生成主链路仍未稳定达到“高一致性、可信的初步可编辑 YAML 初稿”门槛

这些问题说明：
- 当前最需要继续收敛的是真实 provider 输出稳定性、scene 切分一致性和 validation 诚实度
- 继续围绕 deterministic 扩题材规则、扩 fixture、扩 demo 话术，ROI 很低

## 当前方向纠偏

修正后的主方向：
- 方向纠偏已基本完成，当前进入真实样本质量收敛阶段
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
- `openai_compatible` 现已补入 canonical 输出兼容归一化：provider 若返回 `minor` / `extra` 一类角色别名或 `memory` 一类 beat 别名，不再因此被误打回 loose-normalized
- 《雾港回声》真实 provider 复验已确认一条关键纠偏：当前链路已可稳定回到 canonical parse，并持续完成 `screenplay_generation -> validation -> persistence`
- loose-normalized 路径现已显式标低置信度，并在后续 validation 中更诚实地下调为 `failed`，避免“松散 schema 被补齐后看起来也像通过”
- `buildProviderContext` 的地点候选已继续去叙事碎片，减少 `回到老房子`、`送来医院` 这类弱提示重新污染 provider 上下文
- `buildProviderContext` 已进一步减重：移除顶层角色/地点候选和规则 summary，减少 deterministic 摘要与候选列表对 `llm` 主链路的污染
- `openai_compatible` Prompt 已追加“统一双引号字符串 + 缩短 evidence.excerpt”的输出约束；《雾港回声》通过真实 API 再次验证后，当前链路已可在默认本地服务形态下直接成功完成 `screenplay_generation -> validation -> persistence`，不再回退到 deterministic baseline
- `repairGeneratedDocument` 已改为优先保留 provider 给出的更具体 canonical 地点；真实《雾港回声》回归后，scene_001 已不再被本地修复阶段收缩成 `INT/报刊亭`
- 模板化 objective 清理已补入 `建立悬疑基调` 一类新句式；最新真实回归里，scene_001 的模板 objective 已被诚实清空为空值，而不是带着套话通过 validation
- 最新真实回归里，scene_001 已能稳定保持 `EXT`，并且不再因为本地 repair 阶段错误回退为 `INT`
- Chrome 前端实测已确认“切换为空白手工输入 -> 录入《雾港回声》三章 -> 提交生成 -> 页面自动载入 YAML 初稿”这条真实用户路径可以直接跑通

## 尚未完成

### P0：主链路质量纠偏

- 收紧 `openai_compatible` 输入上下文，避免把 deterministic 伪角色、错地点、模板 objective、规则摘要继续灌给 LLM
- 把真实中文三章节样本纳入主回归集，而不是只在 fixture 上看起来稳定
- 保留 deterministic，但仅作为 fallback / smoke baseline

### P1：真实输入可信度

- 收紧真实三章节输入下的人物抽取错误
- 降低模板化 objective / open_questions / dialogue
- 让 beat 更接近可拍动作，而不是截断摘要
- 收敛真实 provider 在 scene 切分上的波动：同一《雾港回声》样本当前仍可能在 3 scene / 6 scene 之间摆动

### P1：validation 诚实度

- 继续提高 validation 对伪角色、模板化 scene 文案、低证据场景的拦截能力
- 在明显语义质量不足时，不再返回 `passed`

## 下一步优先级

优先级 1：
- 继续完成 `openai_compatible` Prompt / context 去污染，减少 LLM 继承 deterministic 错误先验，并观察上游 LLM 耗时波动
- 优先削减 provider context 里的规则摘要和重复候选，尽量让原始章节文本成为唯一高权重输入

优先级 2：
- 把真实《雾港回声》样本继续作为主回归集，围绕角色 / scene 切分 / objective / open_questions 质量收敛

优先级 3：
- 继续提高 validation 对伪角色、模板化 scene 文案、低证据场景的拦截能力，优先覆盖 canonical 输出仍“结构正确但表达偏模板”的场景

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
- 当前 blocker：真实三章节输入已可稳定跑通 canonical parse，scene_001 模板 objective 与 `INT/EXT` 回退问题已明显缓解；但上游 LLM 耗时仍有波动，且当前任务状态在执行期间要到整条 `runner.Run(...)` 结束后才统一写回，前端/轮询观感仍可能长时间停在 `ingest`；另外 scene 切分数量波动仍未稳定收敛

## 协作提醒

- 后续 session 若继续补 deterministic 题材模板，必须先证明这是比赛 ROI 最高的路径；默认视为偏题
- 若实现与本文件冲突，以本文件为准，并应优先修正文档已经指出的偏差
- 下一轮实现前，先确认 `decision-log.md`、`final-solution.md`、`backend-architecture.md`、`backend-pipeline.md` 与本文件保持一致
