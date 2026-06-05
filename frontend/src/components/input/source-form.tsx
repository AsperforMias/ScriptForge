export function SourceForm() {
  return (
    <section className="panel-section" aria-labelledby="source-form-heading">
      <div className="section-heading">
        <div>
          <h3 id="source-form-heading">作品与改编设定</h3>
          <p>首版保留清晰表单结构，为后续 `react-hook-form` 接入预留位置。</p>
        </div>
        <span className="section-tag">Metadata</span>
      </div>

      <div className="field-grid">
        <label className="field">
          <span>作品标题</span>
          <input className="text-input" type="text" placeholder="例如：夜雨疑云" />
        </label>
        <label className="field">
          <span>作者或来源备注</span>
          <input className="text-input" type="text" placeholder="例如：示例作者 / 连载草稿" />
        </label>
      </div>

      <div className="field-grid">
        <label className="field">
          <span>改编风格</span>
          <input className="text-input" type="text" placeholder="例如：悬疑网剧 / 青春群像" />
        </label>
        <label className="field">
          <span>受众定位</span>
          <input className="text-input" type="text" placeholder="例如：大众向 / 女性向" />
        </label>
      </div>

      <label className="field">
        <span>补充要求</span>
        <textarea
          className="text-area text-area--compact"
          placeholder="例如：强化外部冲突、保留心理悬念、控制场景数量。"
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
