# Milestones

## 用途

本文件定义每一阶段“做到什么才算完成”。后续 human/agent session 应按此判断是否可以切换到下一阶段。

## Phase 1: Backend Skeleton

完成定义：
- `backend/` 有 Go module
- 服务能启动 HTTP 端口
- 路由存在：`POST /api/v1/jobs`、`GET /api/v1/jobs/{id}`、`GET /api/v1/jobs/{id}/result`、`GET /api/v1/jobs/{id}/export`
- 中间件栈完整接入
- 返回结构符合 `api-contract.md`

建议 PR 拆分：
- `backend: initialize go module and service entrypoint`
- `backend: add middleware chain and request envelope`
- `backend: add job routes and handler skeleton`

## Phase 2: Domain and Validation

完成定义：
- YAML 领域模型已实现
- YAML 可序列化为稳定文本
- Schema 校验器可运行
- 至少有一组合法样例和一组非法样例

建议 PR 拆分：
- `backend: add screenplay domain models`
- `backend: implement yaml serializer and validator`
- `testdata: add valid and invalid screenplay fixtures`

## Phase 3: Deterministic Pipeline MVP

完成定义：
- 3 章以上输入可创建 job
- job 能走完整 pipeline
- 可返回合法 YAML 结果
- 结果可导出
- SQLite 和本地产物存储可用

建议 PR 拆分：
- `backend: add sqlite job store`
- `backend: implement deterministic pipeline runner`
- `backend: persist yaml artifacts and export endpoint`

## Phase 4: Frontend Integration

完成定义：
- 能录入 3 章以上文本
- 能查询任务状态
- 能查看 YAML 文本
- 能编辑并导出 YAML

建议 PR 拆分：
- `frontend: add multi-chapter input flow`
- `frontend: add job polling and status ui`
- `frontend: add yaml result workspace`

## Phase 5: LLM Enhancement and Demo Hardening

完成定义：
- `llm` 模式可选接入
- README 具备启动方式、自检方式、依赖说明、demo 链接位
- demo 数据和讲解路径固定
- 至少有最小回归测试

建议 PR 拆分：
- `backend: add llm adapter abstraction`
- `backend: support llm generation mode`
- `docs: finalize readme demo and verification guide`

## Stop Rules

出现以下情况时，不应盲目前进：
- 上一阶段完成定义未满足
- 文档与实现冲突
- 关键接口被临时改动但未回写文档
