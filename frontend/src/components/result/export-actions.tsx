interface ExportActionsProps {
  canExport: boolean;
  canReset: boolean;
  hasEditedChanges: boolean;
  hasJob: boolean;
  jobId?: string;
  notice?: {
    tone: "info" | "success" | "error";
    title: string;
    description: string;
  } | null;
  onCopyCurrent: () => void;
  onDownloadCurrent: () => void;
  onDownloadBackendRaw: () => void;
  onReset: () => void;
}

export function ExportActions({
  canExport,
  canReset,
  hasEditedChanges,
  hasJob,
  jobId,
  notice,
  onCopyCurrent,
  onDownloadCurrent,
  onDownloadBackendRaw,
  onReset,
}: ExportActionsProps) {
  return (
    <section className="result-toolbar" aria-label="Result actions">
      <div className="result-toolbar__job">
        <span className="section-tag">本次结果</span>
        <strong>{jobId || "尚未生成"}</strong>
        <span
          className={`result-toolbar__draft-tag ${
            hasEditedChanges ? "result-toolbar__draft-tag--edited" : "result-toolbar__draft-tag--clean"
          }`}
        >
          {hasEditedChanges ? "当前为本地修改稿" : "当前为生成初稿"}
        </span>
      </div>
      <div className="result-toolbar__actions">
        <button className="secondary-button" disabled={!canReset} onClick={onReset} type="button">
          恢复生成初稿
        </button>
        <button className="secondary-button" disabled={!hasJob} onClick={onDownloadBackendRaw} type="button">
          下载初始 YAML
        </button>
        <button className="secondary-button" disabled={!canExport} onClick={onCopyCurrent} type="button">
          复制当前 YAML
        </button>
        <button className="primary-button" disabled={!canExport} onClick={onDownloadCurrent} type="button">
          导出 YAML
        </button>
      </div>
      {notice ? (
        <div className={`result-toolbar__notice status-notice status-notice--${notice.tone}`}>
          <strong>{notice.title}</strong>
          <p>{notice.description}</p>
        </div>
      ) : null}
    </section>
  );
}
