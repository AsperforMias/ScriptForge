import { useEffect, useMemo, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { useQueryClient } from "@tanstack/react-query";
import { ChapterList } from "../components/input/chapter-list";
import { SourceForm } from "../components/input/source-form";
import { JobStatusPanel } from "../components/jobs/job-status-panel";
import { ExportActions } from "../components/result/export-actions";
import { ScreenplaySummary } from "../components/result/screenplay-summary";
import { YamlEditor } from "../components/result/yaml-editor";
import { mapDraftToCreateJobRequest } from "../features/create-job/mapper";
import {
  cloneWorkspaceFormValues,
  defaultWorkspaceFormValues,
  recommendedWorkspaceSamplePreset,
  sampleWorkspaceFormValues,
  type WorkspaceFormValues,
  type WorkspaceSamplePresetId,
  workspaceSamplePresets,
} from "../features/create-job/form";
import { useCreateJob } from "../features/create-job/use-create-job";
import { useJobPolling } from "../features/job-detail/use-job-polling";
import { useJobResult } from "../features/job-result/use-job-result";
import { downloadTextFile } from "../lib/download";
import { formatDateTime, getErrorMessage } from "../lib/format";
import { requestText } from "../lib/http";
import type { JobStage, JobStatus, PipelineStageName } from "../types/api";

const LAST_JOB_STORAGE_KEY = "scriptforge:lastJobId";
type ResultNoticeTone = "info" | "success" | "error";

const stageOrder: PipelineStageName[] = [
  "ingest",
  "outline",
  "entities",
  "scene_planning",
  "screenplay_generation",
  "validation",
  "persistence",
];

function createIdleStages(): JobStage[] {
  return stageOrder.map((name) => ({
    name,
    status: "queued",
  }));
}

function hasAtLeastThreeCompleteChapters(values: WorkspaceFormValues) {
  const completeChapters = values.chapters.filter(
    (chapter) => chapter.title.trim().length > 0 && chapter.content.trim().length > 0,
  );

  return completeChapters.length >= 3;
}

export function WorkspacePage() {
  const queryClient = useQueryClient();
  const form = useForm<WorkspaceFormValues>({
    defaultValues: cloneWorkspaceFormValues(defaultWorkspaceFormValues),
  });
  const [activeSamplePresetId, setActiveSamplePresetId] = useState<WorkspaceSamplePresetId | null>(
    recommendedWorkspaceSamplePreset.id,
  );
  const [currentJobId, setCurrentJobId] = useState<string | null>(() => {
    return window.localStorage.getItem(LAST_JOB_STORAGE_KEY);
  });
  const [originalYamlText, setOriginalYamlText] = useState("");
  const [editedYamlText, setEditedYamlText] = useState("");
  const [loadedResultJobId, setLoadedResultJobId] = useState<string | null>(null);
  const [formError, setFormError] = useState("");
  const [resultNotice, setResultNotice] = useState<{
    tone: ResultNoticeTone;
    title: string;
    description: string;
  } | null>(null);

  const createJobMutation = useCreateJob();
  const jobDetailsQuery = useJobPolling(currentJobId);
  const jobDetails = jobDetailsQuery.data?.data ?? null;
  const activeJob = jobDetails?.job ?? null;
  const shouldLoadResult = activeJob?.status === "succeeded";
  const jobResultQuery = useJobResult(currentJobId, shouldLoadResult);
  const resultPayload = jobResultQuery.data?.data ?? null;

  useEffect(() => {
    if (currentJobId) {
      window.localStorage.setItem(LAST_JOB_STORAGE_KEY, currentJobId);
      return;
    }

    window.localStorage.removeItem(LAST_JOB_STORAGE_KEY);
  }, [currentJobId]);

  useEffect(() => {
    if (!resultPayload || loadedResultJobId === resultPayload.job_id) {
      return;
    }

    setOriginalYamlText(resultPayload.yaml_text);
    setEditedYamlText(resultPayload.yaml_text);
    setLoadedResultJobId(resultPayload.job_id);
    setResultNotice({
      tone: "success",
      title: "已载入最新生成结果",
      description: "右侧 YAML 编辑区已经同步到这次生成的剧本初稿，你可以继续修改、复制或导出。",
    });
  }, [loadedResultJobId, resultPayload]);

  const stages = jobDetails?.stages?.length ? jobDetails.stages : createIdleStages();
  const resultSummary = resultPayload?.screenplay ?? null;
  const activeJobStatus: JobStatus | null = activeJob?.status ?? null;
  const hasEditedChanges = Boolean(originalYamlText) && editedYamlText !== originalYamlText;
  const resultBadgeLabel = useMemo(() => {
    if (jobResultQuery.isLoading) {
      return "载入中";
    }

    if (activeJobStatus === "succeeded" && resultPayload) {
      return "结果已就绪";
    }

    if (activeJobStatus === "failed") {
      return "等待重新生成";
    }

    if (activeJobStatus === "queued" || activeJobStatus === "running") {
      return "处理中";
    }

    return "可编辑初稿";
  }, [activeJobStatus, jobResultQuery.isLoading, resultPayload]);

  const progressNote = useMemo(() => {
    if (activeJob) {
      return `最近更新：${formatDateTime(activeJob.updated_at)}`;
    }

    if (currentJobId && jobDetailsQuery.isLoading) {
      return "正在恢复最近一次生成记录。";
    }

    return "生成完成后，你可以在右侧继续微调 YAML，并导出为本地文件。";
  }, [activeJob, currentJobId, jobDetailsQuery.isLoading]);

  function startJob(values: WorkspaceFormValues) {
    setFormError("");
    createJobMutation.reset();

    if (!hasAtLeastThreeCompleteChapters(values)) {
      setFormError("至少需要填写完整的 3 个章节后才能开始生成。");
      return;
    }

    const payload = mapDraftToCreateJobRequest(values);

    createJobMutation.mutate(payload, {
      onSuccess: (response) => {
        const nextJobId = response.data?.job.id ?? null;

        setCurrentJobId(nextJobId);
        setOriginalYamlText("");
        setEditedYamlText("");
        setLoadedResultJobId(null);
        setResultNotice({
          tone: "info",
          title: "已开始生成",
          description: "中间会持续更新处理进度；完成后，右侧会自动载入新的剧本初稿。",
        });
        queryClient.removeQueries({ queryKey: ["job-result"] });
      },
      onError: (error) => {
        setFormError(getErrorMessage(error));
      },
    });
  }

  function handleSubmit(values: WorkspaceFormValues) {
    startJob(values);
  }

  function handleRegenerate() {
    startJob(form.getValues());
  }

  async function handleDownloadBackendRaw() {
    if (!currentJobId) {
      return;
    }

    try {
      const yamlText = await requestText(`/jobs/${currentJobId}/export`);
      downloadTextFile(`${currentJobId}.screenplay.yaml`, yamlText);
      setResultNotice({
        tone: "success",
        title: "已下载生成初稿",
        description: "下载内容与本次生成结果保持一致，适合留存原始版本。",
      });
    } catch (error) {
      setFormError(getErrorMessage(error));
      setResultNotice({
        tone: "error",
        title: "下载失败",
        description: getErrorMessage(error),
      });
    }
  }

  function handleLoadSample(presetId: WorkspaceSamplePresetId) {
    const preset = workspaceSamplePresets.find((item) => item.id === presetId);

    form.reset(cloneWorkspaceFormValues(preset?.values ?? sampleWorkspaceFormValues));
    setActiveSamplePresetId(preset?.id ?? recommendedWorkspaceSamplePreset.id);
    setFormError("");
  }

  function handleResetYaml() {
    setEditedYamlText(originalYamlText);
    setResultNotice({
      tone: "info",
      title: "已恢复生成初稿",
      description: "当前编辑区已经放弃本地修改，并重新对齐到本次生成的初始 YAML。",
    });
  }

  function handleExportCurrentYaml() {
    if (!editedYamlText.trim()) {
      return;
    }

    const filename = currentJobId ? `${currentJobId}.edited.screenplay.yaml` : "screenplay.edited.yaml";
    downloadTextFile(filename, editedYamlText);
    setResultNotice({
      tone: "success",
      title: "已导出当前版本",
      description: hasEditedChanges ? "本次导出包含你在页面中的本地修改。" : "本次导出与生成初稿一致。",
    });
  }

  async function handleCopyCurrentYaml() {
    if (!editedYamlText.trim()) {
      return;
    }

    try {
      await navigator.clipboard.writeText(editedYamlText);
      setResultNotice({
        tone: "success",
        title: "已复制 YAML",
        description: hasEditedChanges ? "复制内容包含你当前的本地修改。" : "复制内容与生成初稿一致。",
      });
    } catch (error) {
      setResultNotice({
        tone: "error",
        title: "复制失败",
        description: getErrorMessage(error),
      });
    }
  }

  return (
    <main className="workspace-shell">
      <section className="page-intro">
        <p className="eyebrow">ScriptForge</p>
        <div className="page-intro__heading">
          <div>
            <h1>ScriptForge 剧本改编工作台</h1>
            <p>
              把 3 章以上的小说文本整理成可继续打磨的 YAML 剧本初稿。左侧输入原文与改编要求，中间查看生成进度，右侧继续编辑、摘要浏览与导出。
            </p>
          </div>
          <div className="page-intro__aside">
            <span>支持多章节小说输入与改编要求整理</span>
            <span>内置悬疑、职场、校园运动三组示例</span>
            <span>结果区同时保留 YAML 初稿与结构化摘要</span>
          </div>
        </div>
      </section>

      <section className="workspace-grid" aria-label="ScriptForge workspace">
        <FormProvider {...form}>
          <form className="panel panel--input" onSubmit={form.handleSubmit(handleSubmit)}>
            <div className="panel__header">
              <div>
                <p className="panel__eyebrow">Input Workspace</p>
                <h2>创作素材台</h2>
              </div>
              <span className="panel__badge">至少 3 章</span>
            </div>
            <SourceForm activePresetId={activeSamplePresetId} onLoadSample={handleLoadSample} />
            <ChapterList />
            {formError ? <p className="inline-error">{formError}</p> : null}
            <div className="submit-panel">
              <p className="inline-note">确认章节内容与改编方向后即可开始生成，完成后右侧会自动载入可编辑初稿。</p>
              <button className="primary-button primary-button--full" disabled={createJobMutation.isPending} type="submit">
                {createJobMutation.isPending ? "正在开始生成..." : "生成剧本初稿"}
              </button>
            </div>
          </form>
        </FormProvider>

        <div className="panel panel--status">
          <div className="panel__header">
            <div>
              <p className="panel__eyebrow">Job Status</p>
              <h2>生成进度</h2>
            </div>
            <span className="panel__badge panel__badge--muted">自动更新</span>
          </div>
          <JobStatusPanel
            canRegenerate={activeJob?.status === "failed" && !createJobMutation.isPending}
            createError={getErrorMessage(createJobMutation.error)}
            hasJobId={Boolean(currentJobId)}
            isCreating={createJobMutation.isPending}
            isPolling={jobDetailsQuery.isFetching}
            job={activeJob}
            onRegenerate={handleRegenerate}
            resultError={getErrorMessage(jobResultQuery.error || jobDetailsQuery.error)}
            stages={stages}
          />
          <div className="panel-section">
            <div className="section-heading section-heading--tight">
              <div>
                <h3>继续创作</h3>
                <p>页面会记住最近一次生成记录，刷新后仍可继续查看结果与修改内容。</p>
              </div>
              <span className="section-tag">本地延续</span>
            </div>
            <p className="inline-note">{progressNote}</p>
          </div>
        </div>

        <div className="panel panel--result">
          <div className="panel__header">
            <div>
              <p className="panel__eyebrow">Result Workspace</p>
              <h2>YAML 初稿与结构摘要</h2>
            </div>
            <span className="panel__badge panel__badge--accent">{resultBadgeLabel}</span>
          </div>
          <ExportActions
            canExport={Boolean(editedYamlText.trim())}
            canReset={Boolean(originalYamlText)}
            hasEditedChanges={hasEditedChanges}
            hasJob={Boolean(currentJobId)}
            jobId={currentJobId ?? undefined}
            notice={resultNotice}
            onCopyCurrent={handleCopyCurrentYaml}
            onDownloadBackendRaw={handleDownloadBackendRaw}
            onDownloadCurrent={handleExportCurrentYaml}
            onReset={handleResetYaml}
          />
          <YamlEditor
            errorMessage={getErrorMessage(jobResultQuery.error)}
            hasEditedChanges={hasEditedChanges}
            isLoading={jobResultQuery.isLoading}
            jobId={currentJobId}
            jobStatus={activeJobStatus}
            onChange={setEditedYamlText}
            originalYamlText={originalYamlText}
            yamlText={editedYamlText}
          />
          <ScreenplaySummary
            errorMessage={getErrorMessage(jobResultQuery.error)}
            isLoading={jobResultQuery.isLoading}
            jobId={currentJobId}
            jobStatus={activeJobStatus}
            screenplay={resultSummary}
          />
        </div>
      </section>
    </main>
  );
}
