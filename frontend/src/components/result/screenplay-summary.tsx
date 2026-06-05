import type { ScreenplayDocument } from "../../types/screenplay";

interface ScreenplaySummaryProps {
  screenplay: ScreenplayDocument;
}

export function ScreenplaySummary({ screenplay }: ScreenplaySummaryProps) {
  return (
    <section className="panel-section" aria-labelledby="screenplay-summary-heading">
      <div className="section-heading">
        <div>
          <h3 id="screenplay-summary-heading">结构化摘要</h3>
          <p>摘要区直接以后端返回的 `screenplay` JSON 为主，不把 YAML 解析职责挪到前端。</p>
        </div>
        <span className="section-tag">JSON-backed</span>
      </div>

      <div className="summary-grid">
        <article className="summary-card">
          <h4>角色</h4>
          <ul className="summary-list">
            {screenplay.characters.map((character) => (
              <li key={character.id}>
                <strong>{character.name}</strong>
                <span>{character.role}</span>
              </li>
            ))}
          </ul>
        </article>

        <article className="summary-card">
          <h4>地点</h4>
          <ul className="summary-list">
            {screenplay.locations.map((location) => (
              <li key={location.id}>
                <strong>{location.name}</strong>
                <span>{location.description || "待补充地点说明"}</span>
              </li>
            ))}
          </ul>
        </article>
      </div>

      <div className="scene-stack">
        {screenplay.scenes.map((scene) => (
          <article className="scene-card" key={scene.id}>
            <div className="scene-card__header">
              <div>
                <p className="scene-card__kicker">{scene.id}</p>
                <h4>{scene.title}</h4>
              </div>
              <span className="chip">来源章节 {scene.source_chapters.join(", ")}</span>
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
    </section>
  );
}
