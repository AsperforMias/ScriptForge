import type { JobStatus } from "../../types/api";
import type { ScreenplayDocument } from "../../types/screenplay";

interface ScreenplaySummaryProps {
  errorMessage?: string;
  isLoading?: boolean;
  jobId?: string | null;
  jobStatus?: JobStatus | null;
  screenplay: ScreenplayDocument | null;
}

export function ScreenplaySummary({
  screenplay,
  isLoading,
  jobId,
  jobStatus,
  errorMessage,
}: ScreenplaySummaryProps) {
  if (!screenplay) {
    const stateCopy = (() => {
      if (errorMessage) {
        return {
          tone: "error",
          title: "结构化摘要载入失败",
          description: errorMessage,
        };
      }

      if (isLoading) {
        return {
          tone: "info",
          title: "正在整理结构化摘要",
          description: jobId
            ? `任务 ${jobId} 的 screenplay JSON 正在载入，摘要区不会自行解析 YAML。`
            : "任务成功后，这里会直接读取后端返回的 screenplay JSON。",
        };
      }

      if (jobStatus === "failed") {
        return {
          tone: "error",
          title: "本次任务没有结构化摘要",
          description: "生成失败时不会伪造场景卡片，请先处理失败原因再重新生成。",
        };
      }

      if (jobStatus === "queued" || jobStatus === "running") {
        return {
          tone: "info",
          title: "等待任务完成",
          description: "角色、地点和 scene 摘要会在任务成功后一起出现，保持与后端结果一致。",
        };
      }

      return {
        tone: "neutral",
        title: "暂无结构化结果",
        description: "任务成功后，这里会展示角色、地点和 scene 摘要。",
      };
    })();

    return (
      <section className="panel-section" aria-labelledby="screenplay-summary-heading">
        <div className="section-heading">
          <div>
            <h3 id="screenplay-summary-heading">结构化摘要</h3>
            <p>摘要区直接使用后端返回的 `screenplay` JSON，不在前端自行解析 YAML。</p>
          </div>
          <span className="section-tag">JSON-backed</span>
        </div>
        <div className={`empty-card empty-card--${stateCopy.tone}`}>
          <strong>{stateCopy.title}</strong>
          <p>{stateCopy.description}</p>
        </div>
      </section>
    );
  }

  const validationWarnings = screenplay.validation.warnings ?? [];
  const summaryStats = [
    {
      label: "章节",
      value: String(screenplay.source.chapter_count),
      hint: screenplay.source.title,
    },
    {
      label: "场景",
      value: String(screenplay.scenes.length),
      hint: "来自后端 screenplay JSON",
    },
    {
      label: "角色",
      value: String(screenplay.characters.length),
      hint: "用于辅助 scene 阅读",
    },
    {
      label: "校验",
      value: screenplay.validation.status === "passed" ? "通过" : "未通过",
      hint: validationWarnings.length ? `${validationWarnings.length} 条 warning` : "当前无 warning",
    },
  ];

  return (
    <section className="panel-section" aria-labelledby="screenplay-summary-heading">
      <div className="section-heading">
        <div>
          <h3 id="screenplay-summary-heading">结构化摘要</h3>
          <p>摘要区直接使用后端返回的 `screenplay` JSON，不把 YAML 解析职责挪到前端。</p>
        </div>
        <span className="section-tag">JSON-backed</span>
      </div>

      <div className="summary-overview">
        {summaryStats.map((item) => (
          <article className="summary-overview__card" key={item.label}>
            <span className="summary-overview__label">{item.label}</span>
            <strong>{item.value}</strong>
            <small>{item.hint}</small>
          </article>
        ))}
      </div>

      {validationWarnings.length ? (
        <div className="status-notice status-notice--warning">
          <strong>Validation Warnings</strong>
          <ul className="notice-list">
            {validationWarnings.map((warning) => (
              <li key={warning}>{warning}</li>
            ))}
          </ul>
        </div>
      ) : null}

      <div className="summary-grid">
        <article className="summary-card">
          <div className="summary-card__heading">
            <h4>角色</h4>
            <span className="section-tag">{screenplay.characters.length} 个</span>
          </div>
          {screenplay.characters.length ? (
            <ul className="summary-list">
              {screenplay.characters.map((character) => (
                <li key={character.id}>
                  <strong>{character.name}</strong>
                  <span>{character.role}</span>
                  {character.description ? <small>{character.description}</small> : null}
                </li>
              ))}
            </ul>
          ) : (
            <p className="summary-empty">当前结果没有单独列出角色。</p>
          )}
        </article>

        <article className="summary-card">
          <div className="summary-card__heading">
            <h4>地点</h4>
            <span className="section-tag">{screenplay.locations.length} 个</span>
          </div>
          {screenplay.locations.length ? (
            <ul className="summary-list">
              {screenplay.locations.map((location) => (
                <li key={location.id}>
                  <strong>{location.name}</strong>
                  <span>{location.description || "待补充地点说明"}</span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="summary-empty">当前结果没有单独列出地点。</p>
          )}
        </article>
      </div>

      {screenplay.scenes.length ? (
        <div className="scene-stack">
          {screenplay.scenes.map((scene) => (
            <article className="scene-card" key={scene.id}>
              <div className="scene-card__header">
                <div>
                  <p className="scene-card__kicker">{scene.id}</p>
                  <h4>{scene.title}</h4>
                </div>
                <div className="scene-card__chips">
                  <span className="chip">来源章节 {scene.source_chapters.join(", ")}</span>
                  <span className="chip">beats {scene.beats.length}</span>
                </div>
              </div>
              <p className="scene-card__slugline">
                {scene.slugline.interior_exterior}. {scene.slugline.location_id} / {scene.slugline.time}
              </p>
              <p>{scene.summary}</p>
              <dl className="scene-card__meta">
                <div>
                  <dt>Objective</dt>
                  <dd>{scene.objective || "待补充"}</dd>
                </div>
                <div>
                  <dt>Open Question</dt>
                  <dd>{scene.notes?.open_questions?.[0] || "暂无"}</dd>
                </div>
              </dl>
            </article>
          ))}
        </div>
      ) : (
        <div className="empty-card empty-card--neutral">
          <strong>当前结果没有场景卡片</strong>
          <p>摘要区保持忠于后端 `screenplay` JSON，不会在前端补造 scene 数据。</p>
        </div>
      )}
    </section>
  );
}
