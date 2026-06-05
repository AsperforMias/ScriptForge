import { formatDateTime, formatJobStatus } from "../../lib/format";
import type { JobStage, JobSummary } from "../../types/api";
import { StageTimeline } from "./stage-timeline";

interface JobStatusPanelProps {
  job: JobSummary;
  stages: JobStage[];
}

export function JobStatusPanel({ job, stages }: JobStatusPanelProps) {
  return (
    <section className="status-stack" aria-labelledby="job-status-heading">
      <article className="status-card">
        <div className="section-heading section-heading--tight">
          <div>
            <h3 id="job-status-heading">当前任务</h3>
            <p>脚手架阶段先用静态示例把状态区信息架构固定下来。</p>
          </div>
          <span className={`status-badge status-badge--${job.status}`}>{formatJobStatus(job.status)}</span>
        </div>

        <dl className="status-metadata">
          <div>
            <dt>Job ID</dt>
            <dd>{job.id}</dd>
          </div>
          <div>
            <dt>作品</dt>
            <dd>{job.source_title}</dd>
          </div>
          <div>
            <dt>模式</dt>
            <dd>{job.generation_mode}</dd>
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
      </article>

      <article className="status-card">
        <div className="section-heading section-heading--tight">
          <div>
            <h3>阶段时间线</h3>
            <p>后续接入真实轮询时，这里直接映射 `/jobs/{'{id}'}` 返回值。</p>
          </div>
          <span className="section-tag">Timeline</span>
        </div>
        <StageTimeline stages={stages} />
      </article>

      <article className="status-card">
        <div className="section-heading section-heading--tight">
          <div>
            <h3>最近一次更新</h3>
            <p>状态区保留 warnings / error / 更新时间等调试入口。</p>
          </div>
          <span className="section-tag">Notes</span>
        </div>
        <p className="inline-note">
          更新时间：{formatDateTime(job.updated_at)}。如任务失败，此区会直接展示失败阶段与错误信息。
        </p>
        <button className="primary-button primary-button--full" type="button">
          生成剧本草稿
        </button>
      </article>
    </section>
  );
}
