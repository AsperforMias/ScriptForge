import { useFormContext } from "react-hook-form";
import { workspaceSamplePresets, type WorkspaceFormValues, type WorkspaceSamplePresetId } from "../../features/create-job/form";

interface SourceFormProps {
  activePresetId: WorkspaceSamplePresetId | null;
  onLoadSample: (presetId: WorkspaceSamplePresetId) => void;
  onResetToBlank: () => void;
}

export function SourceForm({ activePresetId, onLoadSample, onResetToBlank }: SourceFormProps) {
  const {
    register,
    formState: { errors, isSubmitting },
  } = useFormContext<WorkspaceFormValues>();
  const activePresetLabel =
    workspaceSamplePresets.find((preset) => preset.id === activePresetId)?.label ?? null;

  return (
    <section className="panel-section" aria-labelledby="source-form-heading">
      <div className="section-heading">
        <div>
          <h3 id="source-form-heading">作品与改编设定</h3>
          <p>默认会载入一组推荐示例帮助你快速起稿，但你可以随时覆盖字段，或切到空白草稿直接录入自己的章节内容。</p>
        </div>
        <span className="section-tag">快速起稿</span>
      </div>

      <div className="sample-preset-list" aria-label="sample presets">
        {workspaceSamplePresets.map((preset) => (
          <button
            aria-pressed={activePresetId === preset.id}
            className={`sample-preset-card${activePresetId === preset.id ? " sample-preset-card--active" : ""}`}
            key={preset.id}
            onClick={() => onLoadSample(preset.id)}
            title={`${preset.label} - ${preset.description}`}
            type="button"
          >
            <span className="sample-preset-card__header">
              <strong>{preset.label}</strong>
            </span>
            <span className="sample-preset-card__description">{preset.description}</span>
          </button>
        ))}
        <button
          className="sample-preset-card sample-preset-card--action"
          disabled={isSubmitting}
          onClick={onResetToBlank}
          type="button"
        >
          <span className="sample-preset-card__header">
            <strong>切换为空白手工输入</strong>
          </span>
          <span className="sample-preset-card__description">直接清空当前示例内容，开始录入你自己的三章小说文本。</span>
        </button>
      </div>

      <p className="inline-note action-row__note action-row__note--full">
        {activePresetLabel
          ? `当前已载入「${activePresetLabel}」示例。你可以直接覆盖字段，或先切换为空白手工输入再录入自己的 3 章内容。`
          : "当前是自定义草稿，可直接粘贴自己的 3 章内容并提交生成。"}
      </p>

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
          <input className="text-input" type="text" placeholder="例如：你的笔名 / 连载草稿" {...register("author")} />
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
          <span>目标受众</span>
          <input className="text-input" type="text" placeholder="例如：大众向 / 女性向" {...register("audience")} />
        </label>
      </div>

      <div className="field-grid">
        <label className="field">
          <span>生成方式</span>
          <select className="text-input" disabled={isSubmitting} {...register("generationMode")}>
            <option value="llm">AI 增强</option>
            <option value="deterministic">标准草稿（兜底）</option>
          </select>
        </label>
        <div className="field field--hint">
          <span>方式说明</span>
          <p className="inline-note">默认优先使用 AI 增强生成可编辑 YAML 初稿；若你只想跑结构基线或作为兜底，可切换到标准草稿。</p>
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

      <div className="chip-row chip-row--static" aria-label="adaptation note examples">
        <span className="chip">强化冲突</span>
        <span className="chip">保留第一人称压迫感</span>
        <span className="chip">优先可拍摄动作</span>
      </div>
    </section>
  );
}
