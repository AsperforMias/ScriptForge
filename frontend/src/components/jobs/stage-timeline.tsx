import { formatDateTime, formatJobStatus, formatStageName } from "../../lib/format";
import type { JobStage } from "../../types/api";

interface StageTimelineProps {
  stages: JobStage[];
}

export function StageTimeline({ stages }: StageTimelineProps) {
  return (
    <ol className="timeline" aria-label="处理阶段">
      {stages.map((stage) => {
        const timelineDescription =
          stage.error_message ||
          (stage.status === "succeeded"
            ? "这一阶段已经完成，系统会继续推进后续整理。"
            : stage.status === "running"
              ? "系统正在处理这一阶段，状态会自动刷新。"
              : "按照既定顺序等待执行。");

        const timelineMeta = stage.finished_at
          ? `完成于：${formatDateTime(stage.finished_at)}`
          : stage.started_at
            ? `开始于：${formatDateTime(stage.started_at)}`
            : "等待执行";

        return (
          <li className="timeline-item" data-status={stage.status} key={stage.name}>
            <div className="timeline-item__marker" aria-hidden="true" />
            <div className="timeline-item__content">
              <div className="timeline-item__heading">
                <strong>{formatStageName(stage.name)}</strong>
                <span>{formatJobStatus(stage.status)}</span>
              </div>
              <p>{timelineDescription}</p>
              <small>{timelineMeta}</small>
            </div>
          </li>
        );
      })}
    </ol>
  );
}
