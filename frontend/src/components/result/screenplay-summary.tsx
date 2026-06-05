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
          title: "结构摘要载入失败",
          description: errorMessage,
        };
      }

      if (isLoading) {
        return {
          tone: "info",
          title: "正在整理结构摘要",
          description: jobId ? `任务 ${jobId} 已完成，系统正在载入这次生成的角色、场景与结构信息。`
          : "生成完成后，这里会自动载入结构摘要。",
        };
      }

      if (jobStatus === "failed") {
        return {
          tone: "error",
          title: "本次没有可用的结构摘要",
          description: "请先查看中间的失败原因，再决定是否调整素材或重新生成。",
        };
      }

      if (jobStatus === "queued" || jobStatus === "running") {
        return {
          tone: "info",
          title: "等待生成完成",
          description: "角色、地点与场景摘要会在生成成功后一起出现。",
        };
      }

      return {
        tone: "neutral",
        title: "暂未生成结构摘要",
        description: "生成成功后，这里会展示角色、地点与场景整理结果，方便快速浏览整份剧本。",
      };
    })();

    return (
      <section className="panel-section" aria-labelledby="screenplay-summary-heading">
        <div className="section-heading">
          <div>
            <h3 id="screenplay-summary-heading">结构摘要</h3>
            <p>先看整体规模，再顺着角色、地点和场景卡片快速浏览这次改编结果。</p>
          </div>
          <span className="section-tag">Summary</span>
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
      hint: "已整理可拍摄场景",
    },
    {
      label: "角色",
      value: String(screenplay.characters.length),
      hint: "用于快速回看人物关系",
    },
    {
      label: "校验",
      value: screenplay.validation.status === "passed" ? "通过" : "需检查",
      hint: validationWarnings.length ? `${validationWarnings.length} 条提醒` : "当前无提醒",
    },
  ];

  return (
    <section className="panel-section" aria-labelledby="screenplay-summary-heading">
      <div className="section-heading">
        <div>
          <h3 id="screenplay-summary-heading">结构摘要</h3>
          <p>这部分帮助你先把握角色、地点和场景结构，再回到 YAML 继续精修正文。</p>
        </div>
        <span className="section-tag">Summary</span>
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
          <strong>结构提醒</strong>
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
            <span className="section-tag">{screenplay.characters.length} 位</span>
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
            <p className="summary-empty">当前结果还没有单独整理角色信息。</p>
          )}
        </article>

        <article className="summary-card">
          <div className="summary-card__heading">
            <h4>地点</h4>
            <span className="section-tag">{screenplay.locations.length} 处</span>
          </div>
          {screenplay.locations.length ? (
            <ul className="summary-list">
              {screenplay.locations.map((location) => (
                <li key={location.id}>
                  <strong>{location.name}</strong>
                  <span>{location.description || "等待补充地点说明"}</span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="summary-empty">当前结果还没有单独整理地点信息。</p>
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
                  <span className="chip">节拍 {scene.beats.length}</span>
                </div>
              </div>
              <p className="scene-card__slugline">
                {scene.slugline.interior_exterior}. {scene.slugline.location_id} / {scene.slugline.time}
              </p>
              <p>{scene.summary}</p>
              <dl className="scene-card__meta">
                <div>
                  <dt>场景目标</dt>
                  <dd>{scene.objective || "待补充"}</dd>
                </div>
                <div>
                  <dt>开放问题</dt>
                  <dd>{scene.notes?.open_questions?.[0] || "暂无"}</dd>
                </div>
              </dl>
            </article>
          ))}
        </div>
      ) : (
        <div className="empty-card empty-card--neutral">
          <strong>当前结果还没有场景卡片</strong>
          <p>生成成功后，这里会展示按场景整理的摘要内容，帮助你快速回看整份剧本。</p>
        </div>
      )}
    </section>
  );
}
