const chapterDrafts = [
  {
    index: 1,
    title: "第一章",
    excerpt: "主角深夜回家，在门前发现锁芯上残留新划痕。",
  },
  {
    index: 2,
    title: "第二章",
    excerpt: "楼道录音与监控片段交错出现，新的嫌疑人被带入视线。",
  },
  {
    index: 3,
    title: "第三章",
    excerpt: "旧公寓与楼顶平台的对峙把冲突正式拉到台前。",
  },
];

export function ChapterList() {
  return (
    <section className="panel-section" aria-labelledby="chapter-list-heading">
      <div className="section-heading">
        <div>
          <h3 id="chapter-list-heading">章节列表</h3>
          <p>首版固定展示多章节录入形态，下一条 PR 再接入真实表单状态和增删交互。</p>
        </div>
        <span className="section-tag">Chapters</span>
      </div>

      <div className="chapter-list">
        {chapterDrafts.map((chapter) => (
          <article className="chapter-card" key={chapter.index}>
            <div className="chapter-card__header">
              <span className="chapter-index">Chapter {chapter.index}</span>
              <button className="ghost-button" type="button">
                调整顺序
              </button>
            </div>
            <label className="field">
              <span>章节标题</span>
              <input className="text-input" type="text" defaultValue={chapter.title} />
            </label>
            <label className="field">
              <span>章节正文</span>
              <textarea className="text-area" defaultValue={chapter.excerpt} />
            </label>
          </article>
        ))}
      </div>

      <div className="action-row">
        <button className="secondary-button" type="button">
          添加章节
        </button>
        <p className="inline-note">前端即时校验会拦截少于 3 章的提交，后端仍是最终裁定方。</p>
      </div>
    </section>
  );
}
