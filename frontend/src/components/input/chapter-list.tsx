import { useFieldArray, useFormContext } from "react-hook-form";
import { createEmptyChapterDraft, type WorkspaceFormValues } from "../../features/create-job/form";

export function ChapterList() {
  const {
    control,
    register,
    formState: { errors, isSubmitting },
  } = useFormContext<WorkspaceFormValues>();
  const { fields, append, remove, move } = useFieldArray({
    control,
    name: "chapters",
  });

  return (
    <section className="panel-section" aria-labelledby="chapter-list-heading">
      <div className="section-heading">
        <div>
          <h3 id="chapter-list-heading">章节列表</h3>
          <p>前端即时校验会拦截少于 3 章的提交，但后端仍是最终裁定方。</p>
        </div>
        <span className="section-tag">Chapters</span>
      </div>

      <div className="chapter-list">
        {fields.map((field, index) => (
          <article className="chapter-card" key={field.id}>
            <div className="chapter-card__header">
              <span className="chapter-index">Chapter {index + 1}</span>
              <div className="chapter-card__actions">
                <button
                  className="ghost-button ghost-button--compact"
                  disabled={index === 0 || isSubmitting}
                  onClick={() => move(index, index - 1)}
                  type="button"
                >
                  上移
                </button>
                <button
                  className="ghost-button ghost-button--compact"
                  disabled={index === fields.length - 1 || isSubmitting}
                  onClick={() => move(index, index + 1)}
                  type="button"
                >
                  下移
                </button>
                <button
                  className="ghost-button ghost-button--compact"
                  disabled={fields.length <= 3 || isSubmitting}
                  onClick={() => remove(index)}
                  type="button"
                >
                  删除
                </button>
              </div>
            </div>
            <label className="field">
              <span>章节标题</span>
              <input
                className="text-input"
                type="text"
                {...register(`chapters.${index}.title`, { required: "章节标题不能为空" })}
              />
              {errors.chapters?.[index]?.title ? (
                <small className="field-error">{errors.chapters[index]?.title?.message}</small>
              ) : null}
            </label>
            <label className="field">
              <span>章节正文</span>
              <textarea
                className="text-area"
                {...register(`chapters.${index}.content`, {
                  required: "章节正文不能为空",
                  minLength: {
                    value: 10,
                    message: "章节正文至少输入 10 个字符",
                  },
                })}
              />
              {errors.chapters?.[index]?.content ? (
                <small className="field-error">{errors.chapters[index]?.content?.message}</small>
              ) : null}
            </label>
          </article>
        ))}
      </div>

      <div className="action-row">
        <button
          className="secondary-button"
          disabled={isSubmitting}
          onClick={() => append(createEmptyChapterDraft(fields.length + 1))}
          type="button"
        >
          添加章节
        </button>
        <p className="inline-note">章节顺序会直接映射到后端的 `source.chapters[].index`。</p>
      </div>
    </section>
  );
}
