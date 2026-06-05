import { formatDateTime, formatJobStatus } from "../../lib/format";
import type { JobStage, JobSummary } from "../../types/api";
import { StageTimeline } from "./stage-timeline";

interface JobStatusPanelProps {
  canRegenerate?: boolean;
  hasJobId?: boolean;
  job: JobSummary | null;
  stages: JobStage[];
  createError?: string;
  resultError?: string;
  isCreating?: boolean;
  isPolling?: boolean;
  onRegenerate?: () => void;
}

export function JobStatusPanel({
  canRegenerate,
  hasJobId,
  job,
  stages,
  createError,
  resultError,
  isCreating,
  isPolling,
  onRegenerate,
}: JobStatusPanelProps) {
  const activeError = job?.error_message || createError || resultError || "";
  const badgeStatus = job?.status ?? (isCreating ? "running" : "queued");
  const summaryCopy = (() => {
    if (isCreating && !job) {
      return {
        tone: "info",
        title: "正在创建任务",
        description: "前端已提交真实 create job 请求，成功后这里会自动切换到 2s 轮询。",
      };
    }

    if (!job && hasJobId && isPolling) {
      return {
        tone: "info",
        title: "正在恢复最近任务",
        description: "页面刷新后会继续查询最近一次任务，不会退回静态演示状态。",
      };
    }

    if (!job) {
      return {
        tone: "neutral",
        title: "等待创建任务",
        description: "填写左侧输入区或载入示例后点击“生成剧本草稿”，这里会开始展示真实 pipeline 状态。",
      };
    }

    if (job.status === "failed") {
      return {
        tone: "error",
        title: "任务生成失败",
        description: "失败阶段、错误信息和最近更新时间会保留在这里，便于直接重新生成当前表单。",
      };
    }

    if (job.status === "succeeded") {
      return {
        tone: "success",
        title: "结果已完成",
        description: "右侧结果区会继续加载 YAML 与结构化摘要，导出动作也会同步可用。",
      };
    }

    return {
      tone: "info",
      title: "任务处理中",
      description: "中栏会继续轮询 queued/running 状态，并把停留阶段明确显示在时间线上。",
    };
  })();

  return (
    <section className="status-stack" aria-labelledby="job-status-heading">
      <article className="status-card">
        <div className="section-heading section-heading--tight">
          <div>
            <h3 id="job-status-heading">当前任务</h3>
            <p>这里展示 create job、轮询状态、失败阶段与结果载入情况。</p>
          </div>
          <span className={`status-badge status-badge--${badgeStatus}`}>
            {job ? formatJobStatus(job.status) : "未开始"}
          </span>
        </div>

        <div className={`status-notice ${summaryCopy.tone === "neutral" ? "status-notice--neutral" : `status-notice--${summaryCopy.tone}`}`}>
          <strong>{summaryCopy.title}</strong>
          <p>{summaryCopy.description}</p>
        </div>

        {job ? (
          <>
            <dl className="status-metadata">
              <div>
                <dt>Job ID</dt>
                <dd>{job.id}</dd>
              </div>
              <div>
                <dt>作品</dt>
                <dd>{job.source_title || "-"}</dd>
              </div>
              <div>
                <dt>模式</dt>
                <dd>{job.generation_mode || "-"}</dd>
              </div>
              <div>
                <dt>当前阶段</dt>
                <dd>{job.current_stage}</dd>
              </div>
            </dl>

            <div className="progress-block">
              <div className="progress-block__meta">
                <span>整体进度</span>
                <strong>{job.progress_percent}%</strong>
              </div>
              <div className="progress-bar" aria-hidden="true">
                <div className="progress-bar__fill" style={{ width: `${job.progress_percent}%` }} />
              </div>
            </div>

            <p className="inline-note">最近更新时间：{formatDateTime(job.updated_at)}</p>
          </>
        ) : null}

        {activeError ? (
          <div className="status-notice status-notice--error">
            <strong>异常信息</strong>
            <p>{activeError}</p>
          </div>
        ) : null}
        {job?.status === "failed" ? (
          <div className="action-row action-row--stacked">
            <p className="inline-note">可直接基于左侧当前表单再次创建 job，不依赖额外 retry API。</p>
            <button className="secondary-button" disabled={!canRegenerate} onClick={onRegenerate} type="button">
              {isCreating ? "正在重新生成..." : "重新生成当前表单"}
            </button>
          </div>
        ) : null}
        {job?.warnings?.length ? (
          <div className="status-notice status-notice--warning">
            <strong>Warnings</strong>
            <ul className="notice-list">
              {job.warnings.map((warning) => (
                <li key={warning}>{warning}</li>
              ))}
            </ul>
          </div>
        ) : null}
      </article>

      <article className="status-card">
        <div className="section-heading section-heading--tight">
          <div>
            <h3>阶段时间线</h3>
            <p>严格按文档锁定的 pipeline 阶段顺序展示。</p>
          </div>
          <span className="section-tag">Timeline</span>
        </div>
        <StageTimeline stages={stages} />
      </article>

      <article className="status-card">
        <div className="section-heading section-heading--tight">
          <div>
            <h3>状态说明</h3>
            <p>左侧提交后这里会自动轮询；`queued/running` 持续刷新，`succeeded/failed` 自动停止。</p>
          </div>
          <span className="section-tag">Notes</span>
        </div>
        <p className="inline-note">
          {isCreating
            ? "正在创建 job，成功后会自动进入轮询。"
            : hasJobId && isPolling && !job
              ? "页面正在恢复最近一次任务状态，查询成功后会自动刷新中栏和右侧结果区。"
              : "如果任务失败，这里会直接保留失败阶段、错误信息与最后一次更新时间。"}
        </p>
      </article>
    </section>
  );
}
