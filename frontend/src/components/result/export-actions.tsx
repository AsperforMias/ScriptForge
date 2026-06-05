interface ExportActionsProps {
  jobId: string;
}

export function ExportActions({ jobId }: ExportActionsProps) {
  return (
    <section className="result-toolbar" aria-label="Result actions">
      <div className="result-toolbar__job">
        <span className="section-tag">Current Job</span>
        <strong>{jobId}</strong>
      </div>
      <div className="result-toolbar__actions">
        <button className="secondary-button" type="button">
          恢复后端原始结果
        </button>
        <button className="primary-button" type="button">
          导出 YAML
        </button>
      </div>
    </section>
  );
}
