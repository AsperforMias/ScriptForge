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
- `openai_compatible` 的 provider 失败语义已补回归，覆盖 HTTP 429、error payload 与 empty choices
- 前端默认落地架构已补入 `docs/frontend.md`，达到后续 Codex session 可直接脚手架实现的程度
- 前端视觉方向已补入 `docs/frontend-visual-direction.md`，足以支撑后续 session 直接落地 UI 并继续细调
- `frontend/` 已按锁定方案落地 `Vite + React + TypeScript + TanStack Query + React Hook Form` 骨架
- 单页 `WorkspacePage` 已落地三栏工作台信息架构：`Input Workspace` / `Job Status` / `Result Workspace`
- 前端目录骨架、API 类型、React Query hooks scaffold 与 editorial 基础样式已就位
- README 已补齐前端本地启动说明与跨源 API 环境变量约定，并统一为 `backend@8080 + frontend@5173` 本地启动契约
- 前端工作台已接入真实 `POST /api/v1/jobs`、`GET /api/v1/jobs/{id}`、`GET /api/v1/jobs/{id}/result` 与 `GET /api/v1/jobs/{id}/export`
- 多章节输入、`react-hook-form` 校验、`lastJobId` 持久化恢复、2s 轮询与失败/警告展示已接入真实后端数据
- 结果区已以 YAML 文本为核心，支持恢复后端原始结果、下载后端原始 YAML、导出当前编辑文本
- 结构化摘要区已切换为直接读取后端返回的 `screenplay` JSON，不再依赖静态 demo 数据
- 本地 `backend@8080 + frontend@5173` 已完成 deterministic 与 `llm(openai_compatible)` 两条真实 UI 链路联调
- SQLite store 已补充串行连接、`busy_timeout` 与 `WAL` 配置，解决轮询联调下 job 完成态偶发 `database is locked` 导致的假卡住问题
- deterministic workflow 规则已补强为中文目标、对话、开放问题生成
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
- 固化 demo 演示路径、README 运行说明和评审入口
- 收敛前端 README、dev 配置与实现中的本地端口契约，避免多套说法
- 前端细化错误态、空态与移动端可读性，准备演示口径

优先级 2：
- 扩展 deterministic 与 llm 的 fixture 覆盖面
- 补充更多题材样例输入输出
- 增补存储层与 HTTP 失败场景回归

优先级 3：
- 前端结果编辑体验、响应式细化与错误状态优化
- 视时间决定是否提供公网演示环境

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
