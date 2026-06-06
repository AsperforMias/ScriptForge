import {
  formatJobStatus,
  formatStageName,
  formatUserFacingError,
} from "../../lib/format";
import type { JobStage, JobSummary, PipelineStageName } from "../../types/api";

interface GenerationProgressStripProps {
  canRegenerate?: boolean;
  createError?: string;
  hasJobId?: boolean;
  isCreating?: boolean;
  isPolling?: boolean;
  job: JobSummary | null;
  onRegenerate?: () => void;
  resultError?: string;
  stages: JobStage[];
}

const compactStageLabels: Record<PipelineStageName, string> = {
  ingest: "素材接收",
  outline: "章节梳理",
  entities: "角色地点",
  scene_planning: "场景规划",
  screenplay_generation: "剧本生成",
  validation: "结构校验",
  persistence: "结果保存",
};

function formatCompactStageName(stageName: PipelineStageName) {
  return compactStageLabels[stageName] ?? formatStageName(stageName);
}

function buildStageRows(stages: JobStage[]) {
  return [stages.slice(0, 4), stages.slice(4)];
}

function isConnectorActive(current: JobStage, next?: JobStage) {
  if (current.status === "succeeded") {
    return true;
  }

  if (!next) {
    return false;
  }

  return next.status === "running" || next.status === "succeeded" || next.status === "failed";
}

export function GenerationProgressStrip({
  canRegenerate,
  createError,
  hasJobId,
  isCreating,
  isPolling,
  job,
  onRegenerate,
  resultError,
  stages,
}: GenerationProgressStripProps) {
  const failedMessage =
    job?.status === "failed"
      ? formatUserFacingError(job?.error_message || createError || resultError || "")
      : "";
  const badgeStatus = job?.status ?? (isCreating ? "running" : "queued");
  const railStages = stages.map((stage, index) => {
    if (
      index === 0 &&
      !job &&
      (isCreating || (hasJobId && isPolling)) &&
      stage.status === "queued"
    ) {
      return { ...stage, status: "running" as const };
    }

    return stage;
  });
  const stageRows = buildStageRows(railStages);

  return (
    <section className="panel-section progress-strip" aria-labelledby="generation-progress-heading">
      <div className="progress-strip__header">
        <div>
          <h3 id="generation-progress-heading">生成进度</h3>
        </div>
        <span className={`status-badge status-badge--${badgeStatus}`}>
          {job ? formatJobStatus(job.status) : "未开始"}
        </span>
      </div>

      <div className="progress-strip__rail" aria-label="处理阶段" role="list">
        {stageRows.map((row, rowIndex) => (
          <div className="progress-strip__row" key={`row-${rowIndex}`} role="list">
            {row.map((stage, index) => {
              const nextStage = row[index + 1];

              return (
                <div className="progress-strip__segment" key={stage.name} role="listitem">
                  <div className={`progress-step progress-step--${stage.status}`}>
                    <span className="progress-step__pill">
                      <span className="progress-step__dot" aria-hidden="true" />
                      <span className="progress-step__label">{formatCompactStageName(stage.name)}</span>
                    </span>
                  </div>
                  {nextStage ? (
                    <div className="progress-link" aria-hidden="true">
                      <span className={`progress-link__fill ${isConnectorActive(stage, nextStage) ? "progress-link__fill--active" : ""}`} />
                    </div>
                  ) : null}
                </div>
              );
            })}
          </div>
        ))}
      </div>

      {job?.status === "failed" ? (
        <div className="progress-strip__failure">
          {failedMessage ? <p className="inline-note progress-strip__failure-text">{failedMessage}</p> : null}
          <button className="secondary-button" disabled={!canRegenerate} onClick={onRegenerate} type="button">
            {isCreating ? "正在重新生成..." : "重新生成当前内容"}
          </button>
        </div>
      ) : null}
    </section>
  );
}
