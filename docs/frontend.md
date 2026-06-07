# Frontend Contract

## 目标

前端首版的职责不是做复杂编辑器，而是把“输入小说 -> 触发生成 -> 查看结果 -> 编辑 YAML -> 导出”这条路径打通，并且让评委一眼看懂产品价值。

视觉与版式方向不在本文件重复定义，统一以 [`frontend-visual-direction.md`](frontend-visual-direction.md) 为准。

## 前端必须覆盖的功能

输入区：
- 作品标题
- 作者名或来源备注
- 改编风格 / 额外提示词
- 不少于 3 个章节的文本输入
- 章节标题与顺序管理

当前首版输入范围：
- 以粘贴 / 手工录入为准
- 必须支持用户不依赖 sample preset，直接录入自己的小说章节完成提交
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
- `创作素材台`：录入章节、填写改编要求、提交
- `生成结果`：显示生成进度、错误反馈、YAML、结构化摘要、编辑与导出

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
- `validation.status=passed` 只表示结果通过当前结构 / Schema 校验，不代表人物命名、scene objective、beats、open questions 或整体语义质量已经可靠
- 结果区必须把输出明确表达为“可继续编辑的 YAML 剧本初稿”，而不是“质量已通过的最终剧本”
- `validation.warnings` 必须高可见展示；即使 warnings 为空，前端也不能暗示“内容质量没问题”
- 结果区必须引导用户优先复核：角色名、objective、beats、open questions

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

状态说明（2026-06-07）：
- `frontend/` 已按上述结构落下首版骨架
- 当前已具备可运行的单页工作台、基础样式与真实 API 联调能力
- 多章节表单、创建 job、2s 轮询、YAML 结果载入、结构化摘要与导出动作都已接入真实后端
- Chrome 实测已确认“切换为空白手工输入 -> 手工录入《雾港回声》三章 -> 提交 -> 页面自动载入 YAML 初稿”这条主验收路径可直接跑通
- failed job 现在提供“重新生成当前内容”入口，会基于当前表单重新调用 `POST /api/v1/jobs`
- 首版输入范围已明确收敛为粘贴 / 手工录入，不再承诺尚未实现的上传入口
- 前端示例已扩展为可切换的多题材 preset，至少覆盖悬疑、职场与校园运动
- sample preset 仅作为首次体验与 demo 加速器；“用户手工录入自己的 3 章内容并完成主链路”仍应作为首版主验收路径之一
- 工作台首屏仍默认载入推荐的 `职场` 示例，便于首次体验；同时必须提供显式的“切换为空白手工输入 / 直接覆盖当前字段”入口，避免 UI 暗示只能提交示例内容
- 示例区现已收束为 `2 x 2` 卡片区：悬疑 / 职场 / 校园运动 / 空白手工输入，避免把“空白输入”单独做成旁侧小按钮
- README 已补齐真实前端自检路径，覆盖 sample preset、非 preset 手工输入、job 轮询、YAML/result/export 与 failed-job regenerate 验证步骤
- 工作台已补齐 idle / loading / succeeded / failed 四类真实状态文案，并把结果区空态与失败态对齐到真实 job/result 查询状态
- 响应式断点已细化为桌面双栏、平板双列过渡、移动端纵向堆叠，保持 `Input -> Result` 的阅读顺序，并把生成进度固定收束在结果区顶部
- 当前视觉方向已明显转向“全圆角 + 浅色块层级 + 大留白”；大部分层级不再依赖显式边框，而是通过容器明暗差与间距区分
- `frontend/scripts/smoke-workspace.mjs` 与 `npm run smoke:workspace` 已就位，可自动验证 sample preset 主链路、非 preset 手工 3 章链路、disabled-provider fallback regenerate、`复制当前 YAML`、`lastJobId` 与 workspace draft 刷新恢复，以及移动端 `Input -> Result` 阅读顺序
- 结果区现已统一使用“当前为生成初稿 / 当前为本地编辑稿 / 恢复生成初稿 / 下载生成初稿 YAML / 复制当前 YAML / 导出 YAML”这套文案，并为复制、恢复、导出动作提供真实反馈提示
- `本次结果` 现已从胶囊标签收束为结果区内的普通加粗标题，避免和进度条、信息标签竞争视觉层级
- 结构化摘要现已补充 overview 层，优先展示章节 / 场景 / 角色 / 结构校验状态，再展开角色、地点与 scene 卡片
- 结构化摘要上方的复核提示现已压缩成“短提示 + 动态 warning”形态：固定复核 checklist 已移除，只保留结构状态、提醒数量和最多 3 条动态提醒
- 即使 `validation.warnings` 为空，结果区仍会继续提醒“结构通过 != 内容质量通过”，但不再占用大块固定说明空间
- 页面会在本地保存 `lastJobId` 与 workspace draft，刷新后继续恢复最近一次任务和左侧输入草稿
- 当前已知限制：由于后端现阶段的阶段写回粒度仍偏粗，前端进度条在长耗时 provider 响应期间不会展示细粒度的逐阶段实时推进；这是真实状态写回限制，不是前端伪 loading
- 录屏讲解顺序、默认演示口径与检查点现已迁移到 `docs/demo-recording-guide.md`，与产品页面解耦

推荐自检路径（2026-06-05）：
1. 启动后端 `:8080` 与前端 `:5173`
2. 打开页面后，优先执行一次“切换为空白手工输入 -> 录入自己的 3 章内容”的主链路，不要把默认 preset 当成主要验收证据
3. 以 `generationMode=llm` 作为修正后的主路径提交真实 job；若本地 provider 尚未配置，可临时使用 `deterministic` 做 smoke/debug，但不要再把它当作长期主策略
4. 任务成功后确认 `生成结果` 同时展示顶部生成进度、后端返回的 YAML 文本、结构化摘要与导出动作
5. 若结果区顶部进度在执行期间长时间未出现细粒度阶段变化，按当前实现视为已知限制；应同时确认任务最终能跳到 `已完成` 并载入 YAML，而不是把它误判成前端轮询失效
6. 如需验证 disabled-provider 兜底态，保持后端 `LLM_PROVIDER=disabled`，将表单切到 `generationMode=llm` 提交一次，并确认 job 仍可成功返回、结果区出现明确 fallback warning，且“重新生成当前内容”入口可继续基于当前表单触发新任务
7. 将视口收窄到平板或手机宽度，确认工作区按 `Input -> Result` 纵向阅读，不出现结果区先于输入区的错序
8. 在成功结果上做一次本地 YAML 修改，确认结果工具条会从“当前为生成初稿”切换到“当前为本地编辑稿”，再测试 `复制当前 YAML` 与 `恢复生成初稿`
9. 若本地已启动 Chrome / Edge，可直接运行 `npm run smoke:workspace` 验证 sample preset 与非 preset 手工输入两条真实 job 链路，以及 YAML 载入、结构摘要、导出、本地编辑、复制、disabled-provider fallback regenerate、刷新恢复与移动端阅读顺序
10. 额外执行一次“非 preset 自检”：点击 `切换为空白手工输入`，录入自己的 3 章内容，再走一遍 `create job -> polling -> YAML/result/export`，确认主链路不依赖仓库内置样例

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

首版直接做单页 `WorkspacePage`，桌面端收束为 2 列，生成进度合并进结果区顶部：
- `创作素材台`
  - 作品标题
  - 作者
  - 改编风格
  - 额外说明
  - `2 x 2` 示例 / 空白输入卡片区
  - 章节列表编辑
- `生成结果`
  - 顶部生成进度条
  - `本次结果` 标题与当前 job 基本信息
  - 错误提示 / fallback 提示
  - 重新生成入口
  - YAML 文本编辑区
  - 结构化摘要区
  - 导出按钮

移动端退化成 `Input -> Result` 纵向堆叠，不再保留独立状态列。

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
  - 恢复生成初稿
  - 导出当前文本

结构化摘要视图可直接读取后端返回的 `screenplay` JSON，而不是前端自行解析 YAML。
但前端必须持续提示：当前结果是“可继续编辑的 YAML 剧本初稿”；即使结构校验通过，也仍需人工复核角色名、scene objective、beats 与 open questions。

## 前端 PR 拆分建议

推荐顺序：
1. `frontend: scaffold vite react workspace`
2. `frontend: add multi-chapter input form`
3. `frontend: add job polling and in-result progress strip`
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

详细字段以 [`api-contract.md`](api-contract.md) 为准。

前端不要再假设同步 `POST /api/v1/generate` 版本为最终形态。
如需临时同步接口，只能作为后端内部联调辅助，不应覆盖任务化接口方案。
