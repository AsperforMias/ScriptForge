# Frontend Contract

## 目标

前端首版的职责不是做复杂编辑器，而是把“输入小说 -> 触发生成 -> 查看结果 -> 编辑 YAML -> 导出”这条路径打通，并且让评委一眼看懂产品价值。

视觉与版式方向不在本文件重复定义，统一以 [`frontend-visual-direction.md`](/Users/asperformias/Code/github/ScriptForge/docs/frontend-visual-direction.md) 为准。

## 前端必须覆盖的功能

输入区：
- 作品标题
- 作者名或来源备注
- 改编风格 / 额外提示词
- 不少于 3 个章节的文本输入
- 章节标题与顺序管理

当前首版输入范围：
- 以粘贴 / 手工录入为准
- 不包含独立文件上传或导入解析链路

任务区：
- 提交生成
- 展示任务状态
- 展示错误信息和重试入口

结果区：
- YAML 文本查看
- 结构化摘要查看
- YAML 手动编辑
- 导出 YAML 文件

## 推荐页面或区域划分

可以是单页，也可以是两页，但必须包含：
- `Input Workspace`：录入章节、填写改编要求、提交
- `Generation View`：显示生成中、分阶段状态、错误反馈
- `Result Workspace`：显示 YAML、结构化摘要、编辑与导出

## 用户路径

标准路径：
1. 用户输入至少 3 个章节
2. 用户点击生成
3. 前端调用后端任务接口
4. 用户看到阶段状态或处理中提示
5. 任务完成后加载 YAML
6. 用户修改 YAML
7. 用户导出文件

异常路径：
- 章节数不足：前端即时拦截
- YAML 解析失败：展示错误并保留文本
- 生成失败：展示失败阶段和重试入口

## 前后端边界

前端负责：
- 表单收集
- 章节管理交互
- 状态展示
- YAML 文本编辑体验
- 下载和基础展示

后端负责：
- 章节合法性校验
- 文本清洗与切分
- 人物、地点、场景、剧情结构提取
- 剧本 YAML 生成
- YAML Schema 校验
- 任务状态和结果持久化

边界约束：
- 前端不复制后端业务规则
- “至少 3 章”可以前后端都校验，但后端是最终裁定方
- YAML 是否合法以后端 Schema 校验结果为准

## 对前端实现的建议

评审导向上，前端应优先做到：
- 结构清楚
- 输入体验顺滑
- 结果展示稳定
- 不因 UI 复杂度拖慢全链路交付

不建议首版投入过多时间在：
- 重型富文本编辑器
- 复杂拖拽交互
- 过度动画
- 非核心视觉特效

## 默认落地方案

若没有前端 teammate 的明确反对，后续 human/agent session 默认按以下方案开工：
- 构建方式：`Vite`
- 框架：`React + TypeScript`
- 路由：单页应用，保留 `react-router` 的最小接入能力
- 异步数据：`@tanstack/react-query`
- 表单：`react-hook-form`
- 样式：模块化 CSS 或轻量 utility class，首版不引入复杂设计系统
- YAML 编辑：首版默认 `textarea`，若时间充足再替换为 `Monaco`

这样选择的原因：
- 启动快，适合 72h 项目
- API 轮询与状态缓存简单
- 对 Codex session 足够明确，不需要再猜前端脚手架

## 默认目录结构

建议直接落成：

```text
frontend/
  package.json
  index.html
  src/
    main.tsx
    app/
      router.tsx
      query-client.ts
    pages/
      workspace-page.tsx
    components/
      input/
        source-form.tsx
        chapter-list.tsx
      jobs/
        job-status-panel.tsx
        stage-timeline.tsx
      result/
        yaml-editor.tsx
        screenplay-summary.tsx
        export-actions.tsx
    features/
      create-job/
        api.ts
        mapper.ts
        use-create-job.ts
      job-detail/
        api.ts
        use-job-polling.ts
      job-result/
        api.ts
        use-job-result.ts
    lib/
      http.ts
      download.ts
      format.ts
    types/
      api.ts
      screenplay.ts
    styles/
      globals.css
```

状态说明（2026-06-05）：
- `frontend/` 已按上述结构落下首版骨架
- 当前已具备可运行的单页工作台、基础样式与真实 API 联调能力
- 多章节表单、创建 job、2s 轮询、YAML 结果载入、结构化摘要与导出动作都已接入真实后端
- failed job 现在提供“重新生成当前表单”入口，会基于当前表单重新调用 `POST /api/v1/jobs`
- 首版输入范围已明确收敛为粘贴 / 手工录入，不再承诺尚未实现的上传入口
- 前端示例已扩展为可切换的多题材 preset，至少覆盖悬疑、职场与校园运动
- README 已补齐真实前端自检路径，覆盖 sample preset、job 轮询、YAML/result/export 与 failed-job regenerate 验证步骤
- 工作台已补齐 idle / loading / succeeded / failed 四类真实状态文案，并把结果区空态与失败态对齐到真实 job/result 查询状态
- 响应式断点已细化为桌面三栏、平板双列过渡、移动端纵向堆叠，保持 `Input -> Status -> Result` 的阅读顺序
- 页面会在本地保存 `lastJobId`，刷新后继续查询最近一次任务
- 后续 PR 继续在现有结构上细化结果编辑体验、导出反馈与 demo copy

推荐自检路径（2026-06-05）：
1. 启动后端 `:8080` 与前端 `:5173`
2. 在 `Input Workspace` 选择 `悬疑` / `职场` / `校园运动` 任一 preset
3. 以 `generationMode=deterministic` 提交真实 job，观察 `Job Status` 区的 2s 轮询与阶段变化
4. 任务成功后确认 `Result Workspace` 同时展示后端返回的 YAML 文本、结构化摘要与导出动作
5. 如需验证失败态，保持后端 `LLM_PROVIDER=disabled`，将表单切到 `generationMode=llm` 提交一次，并确认失败信息与“重新生成当前表单”入口可用
6. 将视口收窄到平板或手机宽度，确认三工作区按 `Input -> Status -> Result` 纵向阅读，不出现结果区先于状态区的错序

本地启动契约（2026-06-05）：
- 后端默认监听 `:8080`
- 前端 Vite dev server 默认监听 `:5173`
- 前端本地开发默认通过 Vite proxy 把 `/api/*` 转发到 `http://127.0.0.1:8080`
- 若前后端分离部署或不走 proxy，前端通过 `VITE_API_BASE_URL` 显式指向 `/api/v1`

约束：
- `features/` 只放和后端接口直接耦合的逻辑
- `components/` 只放 UI
- `types/api.ts` 必须对齐 `docs/api-contract.md`
- `types/screenplay.ts` 必须对齐 `docs/yaml-schema.md`

## 页面方案

首版直接做单页 `WorkspacePage`，分成 3 列或 3 个纵向区块：
- `Input Workspace`
  - 作品标题
  - 作者
  - 改编风格
  - 额外说明
  - 章节列表编辑
- `Job Status`
  - 当前 job 基本信息
  - 阶段时间线
  - 错误提示
  - 重新生成入口
- `Result Workspace`
  - YAML 文本编辑区
  - 结构化摘要区
  - 导出按钮

移动端可以退化成分段折叠，不要求桌面三列完全保留。

## 状态流与接口调用

默认状态流：
1. 输入区本地维护表单状态
2. 点击生成后调用 `POST /api/v1/jobs`
3. 拿到 `job.id` 后进入轮询
4. 轮询 `GET /api/v1/jobs/{job_id}`，直到 `succeeded` 或 `failed`
5. 若成功，调用 `GET /api/v1/jobs/{job_id}/result`
6. 点击导出时，直接请求 `GET /api/v1/jobs/{job_id}/export`

默认轮询规则：
- 轮询间隔：`2s`
- `queued/running` 时继续轮询
- `succeeded/failed` 时停止轮询
- 页面刷新后，如本地仍保存 `lastJobId`，允许继续查询

## 前端数据模型

建议在 `frontend/src/types/api.ts` 定义：
- `CreateJobRequest`
- `JobSummary`
- `JobStage`
- `JobDetailsResponse`
- `JobResultResponse`
- `ApiEnvelope<T>`
- `ApiError`

建议在 `frontend/src/types/screenplay.ts` 定义：
- `ScreenplayDocument`
- `ScreenplayScene`
- `ScreenplayBeat`

约束：
- 先从后端 contract 生成或手写最小类型
- 不要在组件内写匿名大对象类型

## YAML 编辑策略

首版默认策略：
- 后端返回的 `yaml_text` 直接放入编辑区
- 前端不负责 YAML Schema 校验
- 编辑区只提供：
  - 文本修改
  - 重置为后端原始结果
  - 导出当前文本

结构化摘要视图可直接读取后端返回的 `screenplay` JSON，而不是前端自行解析 YAML。

## 前端 PR 拆分建议

推荐顺序：
1. `frontend: scaffold vite react workspace`
2. `frontend: add multi-chapter input form`
3. `frontend: add job polling and stage status panel`
4. `frontend: add yaml result workspace and export actions`
5. `frontend: refine responsive layout and error states`

每个 PR 都必须保证：
- 能本地启动
- 不破坏现有 API contract
- README 或相关 docs 同步更新

## API 协作预期

首版建议后端提供的最小接口集合：
- `POST /api/v1/jobs`
- `GET /api/v1/jobs/:id`
- `GET /api/v1/jobs/:id/result`
- `GET /api/v1/jobs/:id/export`

详细字段以 [`api-contract.md`](/Users/asperformias/Code/github/ScriptForge/docs/api-contract.md) 为准。

前端不要再假设同步 `POST /api/v1/generate` 版本为最终形态。
如需临时同步接口，只能作为后端内部联调辅助，不应覆盖任务化接口方案。
