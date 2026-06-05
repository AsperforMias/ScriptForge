import type { JobStatus } from "../../types/api";

interface YamlEditorProps {
  errorMessage?: string;
  isLoading?: boolean;
  jobId?: string | null;
  jobStatus?: JobStatus | null;
  onChange: (nextValue: string) => void;
  yamlText: string;
}

export function YamlEditor({
  yamlText,
  onChange,
  isLoading,
  jobId,
  jobStatus,
  errorMessage,
}: YamlEditorProps) {
  const hasYaml = yamlText.trim().length > 0;
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
          ? `任务 ${jobId} 已完成，正在拉取后端返回的 YAML 与结构化摘要。`
          : "任务成功后，结果区会自动载入后端返回的 YAML。",
      };
    }

    if (jobStatus === "failed") {
      return {
        tone: "error",
        title: "本次任务未生成 YAML",
        description: "请在中栏查看失败阶段和错误信息，再直接重新生成当前表单。",
      };
    }

    if (jobStatus === "queued" || jobStatus === "running") {
      return {
        tone: "info",
        title: "等待任务完成",
        description: "YAML 结果会在任务成功后自动载入，当前不需要手动刷新。",
      };
    }

    return {
      tone: "neutral",
      title: "等待剧本草稿",
      description: "右侧始终以 YAML 为核心结果区；当你在左侧创建任务后，这里会显示可编辑的剧本初稿。",
    };
  })();

  return (
    <section className="panel-section" aria-labelledby="yaml-editor-heading">
      <div className="section-heading">
        <div>
          <h3 id="yaml-editor-heading">YAML 初稿</h3>
          <p>编辑区以 YAML 为核心，支持直接修改、恢复后端原始结果和导出当前文本。</p>
        </div>
        <span className="section-tag">Monospace</span>
      </div>
      {!hasYaml ? (
        <div className={`editor-state editor-state--${stateCopy.tone}`}>
          <strong>{stateCopy.title}</strong>
          <p>{stateCopy.description}</p>
        </div>
      ) : null}
      <textarea
        className="yaml-editor"
        onChange={(event) => onChange(event.target.value)}
        placeholder={hasYaml ? "在这里继续微调 YAML 剧本草稿。" : stateCopy.title}
        readOnly={!hasYaml}
        value={yamlText}
      />
    </section>
  );
}
