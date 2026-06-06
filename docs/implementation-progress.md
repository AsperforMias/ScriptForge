# Implementation Progress

## 当前状态

更新时间：2026-06-06

当前仓库已完成 docs-first 初始化、deterministic 主链路、任务化 API、SQLite/artifact 持久化、`llm` mode 抽象与 vendor-neutral `openai_compatible` 适配器，以及前端工作台首版真实联调落地。
当前后端处于“Phase 5: LLM enhancement and demo hardening”阶段；前端主链路已完成并进入“Phase 4: review hardening / product-facing polish”，但 2026-06-06 的 `main` 自检表明当前重点应先回到“自定义输入验收链路修复 + smoke-check / README / 文案同步 + deterministic 泛化补强”，再继续录屏与 demo 收束。

2026-06-06 自检补充结论：
- 当前产品链路并未把用户锁死在 fixture/sample preset 上；前端表单与后端 `POST /api/v1/jobs` 均支持用户直接粘贴 / 手工录入自己的 3 章以上小说文本
- 当前真正的问题不是“不能输自己的内容”，而是“验证路径过度围绕样例组织”：README、frontend smoke script 和部分结果区文案仍与当前 UI 实现存在漂移
- `npm run build` 可通过，前后端真实联调也可跑通，但 `go test ./...` 与 `npm run smoke:workspace` 在当前 `main` 上都不能作为稳定验收依据
- deterministic 链路不是空壳，但对非 fixture 的真实用户输入仍有明显泛化风险：当前规则更偏单主角、单地点、单冲突模板，scene objective / open question 也容易在同题材内重复
- 因此，下一阶段文档与实现都必须把“用户自定义章节输入”提升为主验收路径，而不是把 sample preset 当作默认成功条件

已完成：
- 题目与赛事要求的精简总结
- 最终方案、范围边界、模块职责文档
- Go 后端架构、API 契约、pipeline 合同文档
- 后端技术栈、库选型与 SQLite schema 文档
- 前端协作边界文档
- YAML Schema 设计文档
- 里程碑与决策记录文档
- 架构自检文档
- PR / commit / 协作规则文档
- 仓库初始目录骨架
- `.gitignore`
- PR 模板
- Go 后端模块初始化
- HTTP 路由与中间件骨架
- `screenplay` 领域模型、YAML 序列化与校验
- SQLite job store 与 artifact store
- deterministic pipeline runner
- 基础后端测试与 pipeline 端到端测试
- 正式样例输入 fixture
- 悬疑 / 职场 / 校园运动三类 deterministic 样例输入输出 fixture
- 家庭情感 / 都市轻喜剧两类 deterministic 样例输入输出 fixture
- HTTP 集成测试与结果导出验证
- README 后端自检入口
- job 状态持久化一致性（`progress_percent` / `warnings`）
- failed job 结果接口错误码对齐（`generation_failed`）
- LLM provider abstraction 与预留配置
- `generation.mode=llm` 已接入 provider abstraction，并支持 `mock` 本地链路验证
- vendor-neutral `openai_compatible` adapter 已就位，并已完成 DeepSeek-compatible `/chat/completions` 真实外部调用验证
- 真实 provider 的 loose YAML 已可归一化为项目 canonical screenplay schema，并通过 `/result` 与 `/export` 链路返回
- 当真实 provider 省略地点、时间或对话信息时，后端会回退到既有 scene plan，补全 slugline 与关键 dialogue beat
- `generation.mode=llm` 现在会为每个 job 额外落盘 `provider_debug.json`，保存 provider、model、parse mode 与原始返回内容，便于 demo 前排查兼容问题
- repo-root `.env.local.example` 已补齐，可作为人类与 agent session 的统一 provider 配置模板
- `openai_compatible` 已补入 fixture 驱动的回归集，覆盖 canonical、fenced、loose schema、缺字段回填与无 scenes 失败场景
- `scripts/run_backend_smoke.sh` 已补齐，提供 deterministic 与 real-provider 的统一后端烟测入口
- `scripts/run_backend_smoke.sh` 已分别通过 deterministic 与 DeepSeek real-provider 路径验证
- `scripts/run_backend_smoke.sh` 现在支持按题材切换 demo fixture，可直接验证 suspense / workplace / campus / family / comedy 路径
- `openai_compatible` 的 provider 失败语义已补回归，覆盖 HTTP 429、error payload 与 empty choices
- `openai_compatible` 已进一步补入“前置说明 + fenced YAML”、“message.content 文本分片数组”与“缺 metadata / characters 的 loose YAML”三类兼容变体，增强真实 provider 输出覆盖
- HTTP / service / SQLite 层已进一步补入 `job_not_found`、`job_not_ready` 与 export 未就绪等失败路径回归
- 前端默认落地架构已补入 `docs/frontend.md`，达到后续 Codex session 可直接脚手架实现的程度
- 前端视觉方向已补入 `docs/frontend-visual-direction.md`，足以支撑后续 session 直接落地 UI 并继续细调
- `frontend/` 已按锁定方案落地 `Vite + React + TypeScript + TanStack Query + React Hook Form` 骨架
- 单页 `WorkspacePage` 已落地三栏工作台信息架构：`Input Workspace` / `Job Status` / `Result Workspace`
- 前端目录骨架、API 类型、React Query hooks scaffold 与 editorial 基础样式已就位
- README 已补齐前端本地启动说明与跨源 API 环境变量约定，并统一为 `backend@8080 + frontend@5173` 本地启动契约
- 前端工作台已接入真实 `POST /api/v1/jobs`、`GET /api/v1/jobs/{id}`、`GET /api/v1/jobs/{id}/result` 与 `GET /api/v1/jobs/{id}/export`
- 多章节输入、`react-hook-form` 校验、`lastJobId` 持久化恢复、2s 轮询与失败/警告展示已接入真实后端数据
- failed job 现在支持基于当前表单重新创建 job，补齐文档要求的失败后“重新生成”入口
- MVP 文档中的小说输入范围已收敛为粘贴 / 手工录入，移除未实现的上传承诺
- 前端 sample preset 已扩展到悬疑、职场、校园运动三类题材，便于演示多场景链路
- 前端首屏仍默认载入推荐的 `职场` 示例，便于首次体验；但页面文案已经收敛为面向作者的产品语言，演示顺序与录屏提示已迁移到 `docs/demo-recording-guide.md`
- README 与 `docs/frontend.md` 已补齐真实前端自检路径，可直接按 sample -> create job -> polling -> YAML/result/export -> failed regenerate 的顺序验收
- 前端状态文案已补齐到 idle / loading / succeeded / failed 四类真实链路，不再把空态、失败态和结果载入态混成同一套提示
- 响应式布局已细化为桌面三栏、平板双列过渡、移动端 `Input -> Status -> Result` 纵向堆叠，便于现场演示和手机查看
- `frontend/scripts/smoke-workspace.mjs` 与 `npm run smoke:workspace` 已补齐，可用本机 Chrome/Edge 自动验证 sample -> create job -> polling -> YAML load -> local edit -> reset
- 结果区现已区分“后端原稿”与“本地编辑稿”，并为复制、恢复、导出动作补齐可见反馈，不再只有静态按钮
- 结构化摘要现已补充 overview 层与 validation warning 展示，继续保持只读取后端 `screenplay` JSON 而不在前端解析 YAML
- 结果区已以 YAML 文本为核心，支持恢复后端原始结果、下载后端原始 YAML、导出当前编辑文本
- 结构化摘要区已切换为直接读取后端返回的 `screenplay` JSON，不再依赖静态 demo 数据
- 本地 `backend@8080 + frontend@5173` 已完成 deterministic 与 `llm(openai_compatible)` 两条真实 UI 链路联调
- SQLite store 已补充串行连接、`busy_timeout` 与 `WAL` 配置，解决轮询联调下 job 完成态偶发 `database is locked` 导致的假卡住问题
- deterministic workflow 规则已补强为中文目标、对话、开放问题生成
- deterministic workflow 已补充家庭情感与都市轻喜剧两类题材规则
- deterministic workflow 单测与 fixture 回归测试

尚未完成：
- 自定义输入验收链路 hardening
  说明：需要把“用户手工录入自己的小说章节”补成明确自检项，并同步修复前端 smoke script、README 操作步骤与 UI 文案漂移，避免当前只对 sample preset 讲得清、验得通
- deterministic 对非 fixture 输入的泛化补强
  说明：当前能生成合法 YAML，但对真实用户输入仍偏模板化，需要补强角色抽取、地点/冲突推断与 scene 级差异化表达，避免 demo 以外的文本看起来过度理想化
- 前后端验收口径重新对齐
  说明：前端需要明确“preset 只是辅助，不是唯一入口”，后端需要明确“fixture 只是回归基线，不代表真实用户输入已经被充分覆盖”
- 更丰富的 fixture 覆盖面
  说明：当前已具备多题材 deterministic 样例和多类 provider fixture，但仍可继续扩展更多真实 provider 返回变体与 demo 专用样例
- demo 视频与演示稿素材
- 公网部署选项

## 里程碑拆分

阶段 0：初始化与对齐
- 状态：已完成
- 目标：把题目、规则、边界、协作方式固定下来

阶段 1：后端骨架
- 状态：已完成
- 目标：建立 Go 服务入口、配置、路由、领域模型和 YAML 结构

阶段 2：生成管线 MVP
- 状态：已完成
- 目标：实现最小可运行的章节输入 -> 结构化剧本 YAML 输出

阶段 3：前端工作流
- 状态：已完成
- 目标：打通输入、生成、查看、编辑、导出

阶段 4：评审强化
- 状态：进行中
- 目标：继续补 smoke-check 覆盖、演示素材、README 收束和必要的 demo 参数说明

阶段 5：LLM 增强
- 状态：进行中
- 目标：保持 deterministic 基线不退化，并继续增强真实外部 provider 的兼容回归与演示稳定性

## 下一步优先级

优先级 1：
- 修复“自定义输入优先”的验收链路：同步更新 README、frontend smoke-check、结果区文案与相关 docs，使 `main` 分支重新具备稳定自检能力
- 补一条非 preset 的真实用户输入自检路径，至少覆盖“清空默认样例 -> 手工录入 3 章 -> create job -> polling -> YAML/result/export”

优先级 2：
- 补强 deterministic 对非 fixture / 非样例输入的泛化能力，降低单主角、单模板输出在真实用户输入下的违和感
- 扩展 deterministic 与 llm 的 fixture 覆盖面
- 继续扩展真实 provider 返回变体回归
- 视演示需要补充更多题材样例输入输出

优先级 3：
- 录制 demo 视频与演示稿素材，沿用当前默认 `职场` 样例和已固化的讲解顺序
- 若演示时间允许，可继续增强 smoke-check 对结果区 polish 的覆盖面
- 视时间决定是否提供公网演示环境

## 最近已完成的 PR 计划

前端近期已完成并落回 `main`：
1. `feat/frontend-phase6-responsive-and-empty-states`
2. `feat/frontend-phase7-result-editing-polish`
3. `feat/frontend-phase8-smoke-check`
4. `feat/frontend-phase9-demo-copy-and-flow`
5. `feat/frontend-phase10-product-facing-copy`

后端近期已完成并落回 `main`：
1. `feat/backend-phase18-provider-fixture-expansion`
2. `feat/backend-phase19-add-more-genre-fixtures`
3. `feat/backend-phase20-http-and-storage-failure-regressions`
4. `feat/backend-phase21-demo-hardening`

## 后续可继续拆分的 PR 方向

前端建议继续按以下小 PR 推进：
1. `feat/frontend-phase11-smoke-coverage-expansion`
- 目标：扩展 smoke-check 对失败态、移动端和复制/恢复反馈的覆盖
- 验收：README 与脚本都能稳定验证更多关键交互，而不是只覆盖 happy path

2. `feat/frontend-phase12-demo-asset-polish`
- 目标：继续收敛录屏时的默认视口、默认文案和状态提示细节
- 验收：录制 demo 时不需要再临时解释 UI 文案或页面行为

后端建议继续按以下小 PR 推进：
1. `feat/backend-phase22-provider-variant-regressions`
- 目标：继续扩展 `openai_compatible` 的真实返回变体与容错 fixture
- 验收：新增变体后 `go test ./...` 仍通过，provider 解析行为更稳定

2. `feat/backend-phase23-demo-asset-support`
- 目标：补更多可直接录屏使用的 fixture、说明文案或 smoke 参数收束
- 验收：后端 demo 路径、fixture 选择与 README 说明进一步统一

## 已锁定决策

已确定：
- 单仓结构
- 前后端分目录
- Go-first 后端
- Go 单体 HTTP API + 进程内异步 job runner
- SQLite + 本地文件持久化
- 显式中间件栈
- YAML 作为核心输出格式
- 文档优先的协作方式
- 后端 pipeline 作为核心展示点
- `openai_compatible` 作为真实外部 LLM 接入协议
- DeepSeek `deepseek-v4-flash` 作为当前低成本功能链路验证模型

尚未锁定：
- 最终 demo 是否沿用当前验证模型，还是切换到更强但更贵的兼容模型
- 是否提供公网演示环境

已补充但尚未落地实现的文档约束：
- 外部 provider 的实际账号与部署参数

## 协作提醒

后续接手者必须先做两件事：
1. 阅读 `docs/README.md` 的顺序索引
2. 修改任何实现状态前先同步更新本文件

当前人工输入依赖：
- 若要继续真实 provider 调试，本地在仓库根目录维护 `.env.local`
- `.env.local` 必须保持 gitignored，且不能在 commit、PR 描述或 README 示例中写入真实 key

若后续代码实现与本文件状态不一致，以最新代码提交者更新过的本文件为准。
