import { formatDateTime, formatStageName } from "../../lib/format";
import type { JobStage } from "../../types/api";

interface StageTimelineProps {
  stages: JobStage[];
}

export function StageTimeline({ stages }: StageTimelineProps) {
  return (
    <ol className="timeline" aria-label="Pipeline stages">
      {stages.map((stage) => (
        <li className="timeline-item" data-status={stage.status} key={stage.name}>
          <div className="timeline-item__marker" aria-hidden="true" />
          <div className="timeline-item__content">
            <div className="timeline-item__heading">
              <strong>{formatStageName(stage.name)}</strong>
              <span>{stage.status}</span>
            </div>
            <p>{stage.error_message || "按锁定阶段顺序展示生成过程与停留位置。"}</p>
            <small>
              {stage.started_at ? `开始：${formatDateTime(stage.started_at)}` : "等待执行"}
            </small>
          </div>
        </li>
      ))}
    </ol>
  );
}
