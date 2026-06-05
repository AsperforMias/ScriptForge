import { useFormContext } from "react-hook-form";
import type { WorkspaceFormValues } from "../../features/create-job/form";

interface SourceFormProps {
  onLoadSample: () => void;
}

export function SourceForm({ onLoadSample }: SourceFormProps) {
  const {
    register,
    formState: { errors, isSubmitting },
  } = useFormContext<WorkspaceFormValues>();

  return (
    <section className="panel-section" aria-labelledby="source-form-heading">
      <div className="section-heading">
        <div>
          <h3 id="source-form-heading">作品与改编设定</h3>
          <p>这里直接接入真实表单状态，创建 job 时会按文档契约映射为后端请求。</p>
        </div>
        <button className="ghost-button" onClick={onLoadSample} type="button">
          载入示例
        </button>
      </div>

      <div className="field-grid">
        <label className="field">
          <span>作品标题</span>
          <input
            className="text-input"
            type="text"
            placeholder="例如：夜雨疑云"
            {...register("title", { required: "作品标题不能为空" })}
          />
          {errors.title ? <small className="field-error">{errors.title.message}</small> : null}
        </label>
        <label className="field">
          <span>作者或来源备注</span>
          <input
            className="text-input"
            type="text"
            placeholder="例如：示例作者 / 连载草稿"
            {...register("author")}
          />
        </label>
      </div>

      <div className="field-grid">
        <label className="field">
          <span>改编风格</span>
          <input
            className="text-input"
            type="text"
            placeholder="例如：悬疑网剧 / 青春群像"
            {...register("style", { required: "改编风格不能为空" })}
          />
          {errors.style ? <small className="field-error">{errors.style.message}</small> : null}
        </label>
        <label className="field">
          <span>受众定位</span>
          <input
            className="text-input"
            type="text"
            placeholder="例如：大众向 / 女性向"
            {...register("audience")}
          />
        </label>
      </div>

      <div className="field-grid">
        <label className="field">
          <span>生成模式</span>
          <select className="text-input" disabled={isSubmitting} {...register("generationMode")}>
            <option value="deterministic">deterministic</option>
            <option value="llm">llm</option>
          </select>
        </label>
        <div className="field field--hint">
          <span>模式说明</span>
          <p className="inline-note">建议先用 deterministic 跑通链路，再切换 llm 验证真实 provider。</p>
        </div>
      </div>

      <label className="field">
        <span>补充要求</span>
        <textarea
          className="text-area text-area--compact"
          placeholder="一行一条，例如：强化外部冲突、保留心理悬念、控制场景数量。"
          {...register("notesText")}
        />
      </label>

      <div className="chip-row" aria-label="adaptation note examples">
        <span className="chip">强化冲突</span>
        <span className="chip">保留第一人称压迫感</span>
        <span className="chip">优先可拍摄动作</span>
      </div>
    </section>
  );
}
