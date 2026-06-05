import { formatDateTime, formatJobStatus, formatStageName } from "../../lib/format";
import type { JobStage } from "../../types/api";

interface StageTimelineProps {
  stages: JobStage[];
}

export function StageTimeline({ stages }: StageTimelineProps) {
  return (
    <ol className="timeline" aria-label="Pipeline stages">
      {stages.map((stage) => {
        const timelineDescription =
          stage.error_message ||
          (stage.status === "succeeded"
            ? "该阶段已完成，继续向后推进到下一环节。"
            : stage.status === "running"
              ? "当前停留阶段，会持续刷新直至成功或失败。"
              : "按锁定阶段顺序排队，尚未进入执行。");

        const timelineMeta = stage.finished_at
          ? `完成：${formatDateTime(stage.finished_at)}`
          : stage.started_at
            ? `开始：${formatDateTime(stage.started_at)}`
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
