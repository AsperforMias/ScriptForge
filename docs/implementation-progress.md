# Implementation Progress

## 当前状态

更新时间：2026-06-05

当前仓库已完成 docs-first 初始化、deterministic 主链路、任务化 API、SQLite/artifact 持久化、`llm` mode 抽象与 vendor-neutral `openai_compatible` 适配器，以及前端工作台首版真实联调落地。
当前后端处于“Phase 5: LLM enhancement and demo hardening”阶段；前端处于“Phase 3: frontend workflow”主链路打通阶段，已完成单页工作台骨架、editorial 三栏布局、多章节表单、真实 job 创建/轮询、YAML 结果区、结构化摘要与导出动作接入。

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
- `openai_compatible` 已进一步补入“前置说明 + fenced YAML”与“缺 metadata / characters 的 loose YAML”两类 fixture，增强真实 provider 输出变体覆盖
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
- README 与 `docs/frontend.md` 已补齐真实前端自检路径，可直接按 sample -> create job -> polling -> YAML/result/export -> failed regenerate 的顺序验收
- 前端状态文案已补齐到 idle / loading / succeeded / failed 四类真实链路，不再把空态、失败态和结果载入态混成同一套提示
- 响应式布局已细化为桌面三栏、平板双列过渡、移动端 `Input -> Status -> Result` 纵向堆叠，便于现场演示和手机查看
- 结果区现已区分“后端原稿”与“本地编辑稿”，并为复制、恢复、导出动作补齐可见反馈，不再只有静态按钮
- 结构化摘要现已补充 overview 层与 validation warning 展示，继续保持只读取后端 `screenplay` JSON 而不在前端解析 YAML
- 结果区已以 YAML 文本为核心，支持恢复后端原始结果、下载后端原始 YAML、导出当前编辑文本
- 结构化摘要区已切换为直接读取后端返回的 `screenplay` JSON，不再依赖静态 demo 数据
- 本地 `backend@8080 + frontend@5173` 已完成 deterministic 与 `llm(openai_compatible)` 两条真实 UI 链路联调
- SQLite store 已补充串行连接、`busy_timeout` 与 `WAL` 配置，解决轮询联调下 job 完成态偶发 `database is locked` 导致的假卡住问题
- deterministic workflow 规则已补强为中文目标、对话、开放问题生成
- deterministic workflow 已补充家庭情感与都市轻喜剧两类题材规则
- deterministic workflow 单测与 fixture 回归测试

未开始：
- 更丰富的 fixture 覆盖面
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
- 状态：进行中
- 目标：打通输入、生成、查看、编辑、导出

阶段 4：评审强化
- 状态：进行中
- 目标：补 demo 样例、README、测试说明、部署说明、演示素材

阶段 5：LLM 增强
- 状态：进行中
- 目标：保持 deterministic 基线不退化，并把真实外部 provider 接入收缩到配置层

## 下一步优先级

优先级 1：
- 若演示时间允许，补前端 smoke/check 脚本，把 README 自检路径进一步固化为可执行入口
- 固化最小前端 smoke-check，减少评委和队友手动点完全链路的成本

优先级 2：
- 扩展 deterministic 与 llm 的 fixture 覆盖面
- 补充更多题材样例输入输出
- 增补存储层与 HTTP 失败场景回归

优先级 3：
- 前端 demo copy、引导文案与默认演示顺序收敛
- 视时间决定是否提供公网演示环境

## 建议 PR 计划

前端建议按以下小 PR 顺序推进：
1. `feat/frontend-phase6-responsive-and-empty-states`
- 目标：补移动端可读性、空态、加载态、失败态与状态文案细化
- 验收：桌面三栏保持稳定，移动端退化为纵向堆叠，create/polling/result/failed 各状态都有清晰展示

2. `feat/frontend-phase7-result-editing-polish`
- 目标：优化 YAML 编辑区、结果摘要信息层次、导出反馈与编辑恢复体验
- 验收：YAML 编辑、恢复、导出路径更顺滑，不改变后端契约

3. `feat/frontend-phase8-smoke-check`
- 目标：补最小前端自检入口，可为 README 脚本化检查或轻量 e2e
- 验收：队友、评委或 agent 能按固定步骤快速验证真实前端链路

4. `feat/frontend-phase9-demo-copy-and-flow`
- 目标：固化演示时默认 sample、页面文案、引导信息与 demo 操作顺序
- 验收：首屏信息、按钮文案、提示文本有统一口径，便于录视频和现场讲解

后端建议按以下小 PR 顺序推进：
1. `feat/backend-phase18-provider-fixture-expansion`
- 目标：继续扩展 `openai_compatible` provider 的 loose-output fixture 与解析回归
- 验收：新增 fixture 后 `go test ./...` 仍通过，provider 容错覆盖面扩大

2. `feat/backend-phase19-add-more-genre-fixtures`
- 目标：增加 1 到 2 类新题材 deterministic/llm 样例输入输出
- 验收：新增题材至少覆盖样例请求、期望 YAML 或可验证结果，README 或 docs 可引用

3. `feat/backend-phase20-http-and-storage-failure-regressions`
- 目标：补 HTTP 错误、存储异常、provider 异常等失败场景回归
- 验收：失败路径错误码、返回体和状态持久化行为稳定

4. `feat/backend-phase21-demo-hardening`
- 目标：收敛 demo 使用模型、provider 调试信息、最终演示参数与自检路径
- 验收：本地 demo 路径、真实 provider 路径和排障入口稳定，录制视频前无需再改主链路

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
