# Implementation Progress

## 当前状态

更新时间：2026-06-05

当前仓库处于“文档先行的项目初始化”阶段。
当前仓库已进入“后端 Phase 2: deterministic pipeline + persistence”阶段。

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
- HTTP 集成测试与结果导出验证
- README 后端自检入口
- job 状态持久化一致性（`progress_percent` / `warnings`）
- failed job 结果接口错误码对齐（`generation_failed`）
- LLM provider abstraction 与预留配置
- `generation.mode=llm` 已接入 provider abstraction，并支持 `mock` 本地链路验证

未开始：
- LLM 调用适配层
- 前端应用脚手架
- 更丰富的 fixture 覆盖面
- 更细粒度的存储层测试
- 更完整的 README 部署说明

## 里程碑拆分

阶段 0：初始化与对齐
- 状态：已完成
- 目标：把题目、规则、边界、协作方式固定下来

阶段 1：后端骨架
- 状态：已完成
- 目标：建立 Go 服务入口、配置、路由、领域模型和 YAML 结构

阶段 2：生成管线 MVP
- 状态：进行中
- 目标：实现最小可运行的章节输入 -> 结构化剧本 YAML 输出

阶段 3：前端工作流
- 状态：未开始
- 目标：打通输入、生成、查看、编辑、导出

阶段 4：评审强化
- 状态：未开始
- 目标：补 demo 样例、README、测试说明、部署说明、演示素材

## 下一步优先级

优先级 1：
- 补充基于 SQLite 的状态查询与错误场景测试
- 继续补充失败场景与边界条件的 HTTP 集成测试
- 增加 README 部署说明与演示入口

优先级 2：
- 完善 deterministic 规则质量
- 让输出的 `characters / locations / scenes / beats` 更贴近剧本改编语义
- 扩展更多题材的正式样例小说输入

优先级 3：
- 接入真实模型调用
- 加入阶段状态和错误展示
- 前端接入结果编辑与下载

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

尚未锁定：
- 前端框架选型
- 具体模型供应商
- 是否提供公网演示环境

已补充但尚未落地实现的文档约束：
- 后端技术栈与库选型
- SQLite schema 与 artifact 目录规范
- HTTP status / error code 映射
- 后端测试矩阵

## 协作提醒

后续接手者必须先做两件事：
1. 阅读 `docs/README.md` 的顺序索引
2. 修改任何实现状态前先同步更新本文件

若后续代码实现与本文件状态不一致，以最新代码提交者更新过的本文件为准。
