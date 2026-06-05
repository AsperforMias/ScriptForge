import type { JobStatus } from "../../types/api";

interface YamlEditorProps {
  errorMessage?: string;
  hasEditedChanges?: boolean;
  isLoading?: boolean;
  jobId?: string | null;
  jobStatus?: JobStatus | null;
  originalYamlText?: string;
  onChange: (nextValue: string) => void;
  yamlText: string;
}

export function YamlEditor({
  yamlText,
  onChange,
  isLoading,
  hasEditedChanges,
  jobId,
  jobStatus,
  originalYamlText,
  errorMessage,
}: YamlEditorProps) {
  const hasYaml = yamlText.trim().length > 0;
  const lineCount = hasYaml ? yamlText.split(/\r?\n/).length : 0;
  const characterCount = yamlText.length;
  const originalCharacterCount = (originalYamlText ?? "").length;
  const stateCopy = (() => {
    if (errorMessage) {
      return {
        tone: "error",
        title: "YAML 结果载入失败",
        description: errorMessage,
      };
    }

    if (isLoading) {
      return {
        tone: "info",
        title: "正在载入 YAML",
        description: jobId
          ? `任务 ${jobId} 已完成，系统正在获取这次生成的 YAML 初稿与结构摘要。`
          : "生成完成后，结果区会自动载入 YAML 初稿。",
      };
    }

    if (jobStatus === "failed") {
      return {
        tone: "error",
        title: "本次没有生成可编辑初稿",
        description: "请先查看中间的失败原因，再决定是否调整素材或重新生成。",
      };
    }

    if (jobStatus === "queued" || jobStatus === "running") {
      return {
        tone: "info",
        title: "等待生成完成",
        description: "YAML 初稿会在本次生成完成后自动载入，不需要手动刷新页面。",
      };
    }

    return {
      tone: "neutral",
      title: "等待剧本初稿",
      description: "这里会保留结构化 YAML 结果，方便你继续微调、复制和导出。",
    };
  })();

  return (
    <section className="panel-section" aria-labelledby="yaml-editor-heading">
      <div className="section-heading">
        <div>
          <h3 id="yaml-editor-heading">YAML 剧本初稿</h3>
          <p>结果区始终以 YAML 为核心，保留结构化正文，方便继续打磨与版本留存。</p>
        </div>
        <span className="section-tag">Monospace</span>
      </div>
      {hasYaml ? (
        <div className="editor-metadata" aria-label="YAML draft metadata">
          <span className={`editor-metadata__badge ${hasEditedChanges ? "editor-metadata__badge--edited" : "editor-metadata__badge--clean"}`}>
            {hasEditedChanges ? "含本地修改" : "与生成初稿一致"}
          </span>
          <span>行数 {lineCount}</span>
          <span>字符 {characterCount}</span>
          {originalYamlText ? <span>初稿字符 {originalCharacterCount}</span> : null}
        </div>
      ) : null}
      {!hasYaml ? (
        <div className={`editor-state editor-state--${stateCopy.tone}`}>
          <strong>{stateCopy.title}</strong>
          <p>{stateCopy.description}</p>
        </div>
      ) : null}
      <textarea
        className="yaml-editor"
        onChange={(event) => onChange(event.target.value)}
        placeholder={hasYaml ? "在这里继续微调 YAML 剧本初稿。" : stateCopy.title}
        readOnly={!hasYaml}
        value={yamlText}
      />
    </section>
  );
}
