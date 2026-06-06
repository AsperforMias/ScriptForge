import { formatDateTime, formatGenerationMode, formatJobStatus, formatStageName } from "../../lib/format";
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
        title: "正在建立本次生成",
        description: "素材已经提交，系统正在准备本次剧本生成任务。",
      };
    }

    if (!job && hasJobId && isPolling) {
      return {
        tone: "info",
        title: "正在恢复最近一次记录",
        description: "页面刷新后，工作台会继续找回最近一次生成进度，不需要重新填写内容。",
      };
    }

    if (!job) {
      return {
        tone: "neutral",
        title: "等待开始生成",
        description: "完成左侧素材填写后点击生成，这里会开始显示本次改编进度。",
      };
    }

    if (job.status === "failed") {
      return {
        tone: "error",
        title: "本次生成未完成",
        description: "你可以先查看停留阶段与错误原因，再基于当前表单重新生成。",
      };
    }

    if (job.status === "succeeded") {
      return {
        tone: "success",
        title: "剧本初稿已生成",
        description:
          "右侧会载入可继续编辑的 YAML 初稿与结构化摘要；请优先复核角色名、场景目标、beats 与开放问题。",
      };
    }

    return {
      tone: "info",
      title: "系统正在处理中",
      description: "工作台会自动刷新每个阶段的状态，直到本次生成完成或中断。",
    };
  })();

  return (
    <section className="status-stack" aria-labelledby="job-status-heading">
      <article className="status-card">
        <div className="section-heading section-heading--tight">
          <div>
            <h3 id="job-status-heading">当前生成</h3>
            <p>查看本次生成状态、更新时间与异常信息。</p>
          </div>
          <span className={`status-badge status-badge--${badgeStatus}`}>
            {job ? formatJobStatus(job.status) : "未开始"}
          </span>
        </div>

        <div
          className={`status-notice ${
            summaryCopy.tone === "neutral" ? "status-notice--neutral" : `status-notice--${summaryCopy.tone}`
          }`}
        >
          <strong>{summaryCopy.title}</strong>
          <p>{summaryCopy.description}</p>
        </div>

        {job ? (
          <>
            <dl className="status-metadata">
              <div>
                <dt>生成编号</dt>
                <dd>{job.id}</dd>
              </div>
              <div>
                <dt>作品标题</dt>
                <dd>{job.source_title || "-"}</dd>
              </div>
              <div>
                <dt>生成方式</dt>
                <dd>{job.generation_mode ? formatGenerationMode(job.generation_mode) : "-"}</dd>
              </div>
              <div>
                <dt>当前阶段</dt>
                <dd>{formatStageName(job.current_stage)}</dd>
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

            <p className="inline-note">最近更新：{formatDateTime(job.updated_at)}</p>
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
            <p className="inline-note">重新生成会沿用左侧当前填写的内容，不会清空你已经整理好的素材。</p>
            <button className="secondary-button" disabled={!canRegenerate} onClick={onRegenerate} type="button">
              {isCreating ? "正在重新生成..." : "重新生成当前内容"}
            </button>
          </div>
        ) : null}

        {job?.warnings?.length ? (
          <div className="status-notice status-notice--warning">
            <strong>结果提醒</strong>
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
            <h3>处理阶段</h3>
            <p>系统会依次整理素材、梳理角色与场景，并产出结构化剧本初稿。</p>
          </div>
          <span className="section-tag">自动更新</span>
        </div>
        <StageTimeline stages={stages} />
      </article>

      <article className="status-card">
        <div className="section-heading section-heading--tight">
          <div>
            <h3>使用提示</h3>
            <p>生成过程会自动刷新状态，完成后右侧会同步载入可编辑结果。</p>
          </div>
          <span className="section-tag">进度说明</span>
        </div>
        <p className="inline-note">
          {isCreating
            ? "系统正在接收这次生成请求，稍后会自动切换到进度更新。"
            : hasJobId && isPolling && !job
              ? "正在找回最近一次生成记录，成功后会同步更新中间和右侧工作区。"
              : "如果生成中断，这里会保留停留阶段和错误信息，方便你直接重新生成。"}
        </p>
      </article>
    </section>
  );
}
