interface ExportActionsProps {
  canExport: boolean;
  canReset: boolean;
  hasJob: boolean;
  jobId?: string;
  onDownloadCurrent: () => void;
  onDownloadBackendRaw: () => void;
  onReset: () => void;
}

export function ExportActions({
  canExport,
  canReset,
  hasJob,
  jobId,
  onDownloadCurrent,
  onDownloadBackendRaw,
  onReset,
}: ExportActionsProps) {
  return (
    <section className="result-toolbar" aria-label="Result actions">
      <div className="result-toolbar__job">
        <span className="section-tag">Current Job</span>
        <strong>{jobId || "尚未生成"}</strong>
      </div>
      <div className="result-toolbar__actions">
        <button className="secondary-button" disabled={!canReset} onClick={onReset} type="button">
          恢复后端原始结果
        </button>
        <button className="secondary-button" disabled={!hasJob} onClick={onDownloadBackendRaw} type="button">
          下载后端原始 YAML
        </button>
        <button className="primary-button" disabled={!canExport} onClick={onDownloadCurrent} type="button">
          导出 YAML
        </button>
      </div>
    </section>
  );
}
