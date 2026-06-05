interface YamlEditorProps {
  yamlText: string;
}

export function YamlEditor({ yamlText }: YamlEditorProps) {
  return (
    <section className="panel-section" aria-labelledby="yaml-editor-heading">
      <div className="section-heading">
        <div>
          <h3 id="yaml-editor-heading">YAML 初稿</h3>
          <p>首版使用 `textarea` 作为默认编辑器，先把 YAML-first 的阅读与修改体验做稳。</p>
        </div>
        <span className="section-tag">Monospace</span>
      </div>
      <textarea className="yaml-editor" value={yamlText} readOnly />
    </section>
  );
}
