import type { JobStatus } from "../../types/api";
import type { ScreenplayDocument } from "../../types/screenplay";
import {
  formatCharacterDescription,
  formatCharacterRole,
  formatInteriorExterior,
  formatLocationDescription,
  formatSceneTime,
  formatUserFacingError,
  formatUserFacingWarning,
} from "../../lib/format";

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
          description: formatUserFacingError(errorMessage),
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
          description: "请先查看上方失败原因，再决定是否调整素材或重新生成。",
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
            <h3 id="screenplay-summary-heading">角色、地点与场景</h3>
            <p>先扫一眼规模，再顺着角色、地点和场景卡片快速回看这次改编结果。</p>
          </div>
          <span className="section-tag">概览</span>
        </div>
        <div className={`empty-card empty-card--${stateCopy.tone}`}>
          <strong>{stateCopy.title}</strong>
          <p>{stateCopy.description}</p>
        </div>
      </section>
    );
  }

  const validationWarnings = screenplay.validation.warnings ?? [];
  const validationLabel = screenplay.validation.status === "passed" ? "已通过" : "待修订";
  const primarySummary =
    screenplay.validation.status === "passed"
      ? "结构已通过，接下来重点看人物命名、场景目标和片段是否可信。"
      : "结构仍有待修订，建议先处理提醒，再继续判断内容质量。";
  const warningSummary = validationWarnings.length
    ? `本次有 ${validationWarnings.length} 条内容提醒。`
    : "当前没有额外结构提醒。";
  const locationsById = new Map(screenplay.locations.map((location) => [location.id, location]));
  const seenCharacterDescriptions = new Set<string>();
  const summarizedCharacters = screenplay.characters.map((character) => {
    const description = formatCharacterDescription(character.description);
    if (!description || seenCharacterDescriptions.has(description)) {
      return { ...character, summaryDescription: "" };
    }

    seenCharacterDescriptions.add(description);
    return { ...character, summaryDescription: description };
  });
  const locationGroups = new Map<string, { id: string; name: string; description: string; count: number }>();
  for (const location of screenplay.locations) {
    const key = location.name.trim() || location.id;
    const current = locationGroups.get(key);
    const description = formatLocationDescription(location.description);
    if (!current) {
      locationGroups.set(key, {
        id: location.id,
        name: location.name,
        description,
        count: 1,
      });
      continue;
    }

    current.count += 1;
    if (
      (!current.description || current.description === "等待补充地点说明") &&
      description &&
      description !== "等待补充地点说明"
    ) {
      current.description = description;
    }
  }
  const summarizedLocations = [...locationGroups.values()].map((location) => {
    const genericDescription =
      location.description === "当前章节里最主要的发生地点。" ||
      location.description === "根据章节线索整理出的主要发生地点。";

    if (location.count > 1 && genericDescription) {
      return {
        ...location,
        summaryDescription: "",
      };
    }

    if (location.count > 1) {
      return {
        ...location,
        summaryDescription: location.description,
      };
    }

    return {
      ...location,
      summaryDescription: genericDescription ? "" : location.description,
    };
  });
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
      label: "结构校验",
      value: validationLabel,
      hint: validationWarnings.length
        ? `${validationWarnings.length} 条提醒，仍需人工复核内容质量`
        : "仅表示结构完整，仍需人工复核内容质量",
    },
  ];

  return (
    <section className="panel-section" aria-labelledby="screenplay-summary-heading">
      <div className="section-heading">
        <div>
          <h3 id="screenplay-summary-heading">角色、地点与场景</h3>
          <p>先看整体规模，再回看角色、地点和场景卡片，最后回到 YAML 继续精修。</p>
        </div>
        <span className="section-tag">概览</span>
      </div>

      <div className="quality-guard quality-guard--compact" role="note" aria-label="结果复核提示">
        <div className="quality-guard__header">
          <strong>可编辑初稿，仍需人工复核</strong>
          <span className={`quality-guard__badge quality-guard__badge--${screenplay.validation.status}`}>
            {validationLabel}
          </span>
        </div>
        <p>{primarySummary}</p>
        <p className="quality-guard__summary">{warningSummary}</p>
        {validationWarnings.length ? (
          <div className="quality-guard__warnings status-notice status-notice--warning">
            <ul className="notice-list">
              {validationWarnings.slice(0, 3).map((warning) => (
                <li key={warning}>{formatUserFacingWarning(warning)}</li>
              ))}
            </ul>
          </div>
        ) : null}
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

      <div className="summary-grid">
        <article className="summary-card">
          <div className="summary-card__heading">
            <h4>角色</h4>
            <span className="section-tag">{screenplay.characters.length} 位</span>
          </div>
          {screenplay.characters.length ? (
            <div className="summary-card__scroll">
              <ul className="summary-list">
              {summarizedCharacters.map((character) => (
                <li key={character.id}>
                  <div className="summary-list__primary">
                    <strong>{character.name}</strong>
                    <span className="summary-inline-tag">{formatCharacterRole(character.role)}</span>
                  </div>
                  {character.summaryDescription ? <small>{character.summaryDescription}</small> : null}
                </li>
              ))}
              </ul>
            </div>
          ) : (
            <p className="summary-empty">当前结果还没有单独整理角色信息。</p>
          )}
        </article>

        <article className="summary-card">
          <div className="summary-card__heading">
            <h4>地点</h4>
            <span className="section-tag">{summarizedLocations.length} 处</span>
          </div>
          {screenplay.locations.length ? (
            <div className="summary-card__scroll">
              <ul className="summary-list">
              {summarizedLocations.map((location) => (
                <li key={location.id}>
                  <div className="summary-list__primary">
                    <strong>{location.name}</strong>
                    <span className="summary-inline-tag">{location.count} 场</span>
                  </div>
                  {location.summaryDescription ? <small>{location.summaryDescription}</small> : null}
                </li>
              ))}
              </ul>
            </div>
          ) : (
            <p className="summary-empty">当前结果还没有单独整理地点信息。</p>
          )}
        </article>
      </div>

      {screenplay.scenes.length ? (
        <div className="scene-stack">
          {screenplay.scenes.map((scene, index) => (
            <article className="scene-card" key={scene.id}>
              <div className="scene-card__header">
                <div>
                  <p className="scene-card__kicker">场景 {index + 1}</p>
                  <h4>{scene.title}</h4>
                </div>
                <div className="scene-card__chips">
                  <span className="chip">来源章节 {scene.source_chapters.join(", ")}</span>
                  <span className="chip">片段 {scene.beats.length}</span>
                </div>
              </div>
              <p className="scene-card__slugline">
                {formatInteriorExterior(scene.slugline.interior_exterior)} ·{" "}
                {locationsById.get(scene.slugline.location_id)?.name || "地点待补充"} ·{" "}
                {formatSceneTime(scene.slugline.time)}
              </p>
              <p>{scene.summary}</p>
              <dl className="scene-card__meta">
                <div>
                  <dt>场景目标</dt>
                  <dd>{scene.objective || "待补充"}</dd>
                </div>
                <div>
                  <dt>后续悬念</dt>
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
