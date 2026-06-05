interface YamlEditorProps {
  isLoading?: boolean;
  onChange: (nextValue: string) => void;
  yamlText: string;
}

export function YamlEditor({ yamlText, onChange, isLoading }: YamlEditorProps) {
  return (
    <section className="panel-section" aria-labelledby="yaml-editor-heading">
      <div className="section-heading">
        <div>
          <h3 id="yaml-editor-heading">YAML 初稿</h3>
          <p>编辑区以 YAML 为核心，支持直接修改、恢复后端原始结果和导出当前文本。</p>
        </div>
        <span className="section-tag">Monospace</span>
      </div>
      <textarea
        className="yaml-editor"
        onChange={(event) => onChange(event.target.value)}
        placeholder={isLoading ? "结果加载中..." : "任务成功后，后端返回的 YAML 会显示在这里。"}
        value={yamlText}
      />
    </section>
  );
}
