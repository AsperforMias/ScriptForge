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
  defaultWorkspaceFormValues,
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
import { buildApiUrl, requestText } from "../lib/http";
import type { JobStage, JobStatus, PipelineStageName } from "../types/api";

const LAST_JOB_STORAGE_KEY = "scriptforge:lastJobId";

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
    defaultValues: defaultWorkspaceFormValues,
  });
  const [currentJobId, setCurrentJobId] = useState<string | null>(() => {
    return window.localStorage.getItem(LAST_JOB_STORAGE_KEY);
  });
  const [originalYamlText, setOriginalYamlText] = useState("");
  const [editedYamlText, setEditedYamlText] = useState("");
  const [loadedResultJobId, setLoadedResultJobId] = useState<string | null>(null);
  const [formError, setFormError] = useState("");

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
  }, [loadedResultJobId, resultPayload]);

  const stages = jobDetails?.stages?.length ? jobDetails.stages : createIdleStages();
  const resultSummary = resultPayload?.screenplay ?? null;
  const activeJobStatus: JobStatus | null = activeJob?.status ?? null;
  const resultErrorMessage = getErrorMessage(jobResultQuery.error || jobDetailsQuery.error);
  const resultBadgeLabel = useMemo(() => {
    if (jobResultQuery.isLoading) {
      return "结果载入中";
    }

    if (activeJobStatus === "succeeded" && resultPayload) {
      return "结果已就绪";
    }

    if (activeJobStatus === "failed") {
      return "等待重新生成";
    }

    if (activeJobStatus === "queued" || activeJobStatus === "running") {
      return "等待 YAML";
    }

    return "YAML 核心结果";
  }, [activeJobStatus, jobResultQuery.isLoading, resultPayload]);

  const statusNote = useMemo(() => {
    if (!activeJob) {
      if (currentJobId && jobDetailsQuery.isLoading) {
        return `正在恢复最近任务：${currentJobId}`;
      }

      return "尚未创建任务";
    }

    return `${activeJob.id} / ${formatDateTime(activeJob.updated_at)}`;
  }, [activeJob, currentJobId, jobDetailsQuery.isLoading]);

  function startJob(values: WorkspaceFormValues) {
    setFormError("");
    createJobMutation.reset();

    if (!hasAtLeastThreeCompleteChapters(values)) {
      setFormError("至少需要 3 个填写完整的章节后才能提交。");
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
    } catch (error) {
      setFormError(getErrorMessage(error));
    }
  }

  function handleLoadSample(presetId: WorkspaceSamplePresetId) {
    const preset = workspaceSamplePresets.find((item) => item.id === presetId);

    form.reset(preset?.values ?? sampleWorkspaceFormValues);
    setFormError("");
  }

  function handleResetYaml() {
    setEditedYamlText(originalYamlText);
  }

  function handleExportCurrentYaml() {
    if (!editedYamlText.trim()) {
      return;
    }

    const filename = currentJobId ? `${currentJobId}.edited.screenplay.yaml` : "screenplay.edited.yaml";
    downloadTextFile(filename, editedYamlText);
  }

  return (
    <main className="workspace-shell">
      <section className="page-intro">
        <p className="eyebrow">ScriptForge Editorial Studio</p>
        <div className="page-intro__heading">
          <div>
            <h1>小说转剧本工作台</h1>
            <p>
              真实联调版本：左侧输入多章节与改编要求，中间观察真实 job pipeline，右侧查看并编辑后端返回的
              YAML 结果。
            </p>
          </div>
          <div className="page-intro__aside">
            <span>Vite + React + TypeScript</span>
            <span>TanStack Query + React Hook Form</span>
            <span>当前 API Base: {buildApiUrl("").replace(/\/$/, "")}</span>
          </div>
        </div>
      </section>

      <section className="workspace-grid" aria-label="ScriptForge workspace">
        <FormProvider {...form}>
          <form className="panel panel--input" onSubmit={form.handleSubmit(handleSubmit)}>
            <div className="panel__header">
              <div>
                <p className="panel__eyebrow">Input Workspace</p>
                <h2>创作素材板</h2>
              </div>
              <span className="panel__badge">至少 3 章</span>
            </div>
            <SourceForm onLoadSample={handleLoadSample} />
            <ChapterList />
            {formError ? <p className="inline-error">{formError}</p> : null}
            <div className="submit-panel">
              <p className="inline-note">
                当前会调用 `POST /api/v1/jobs`，成功后自动轮询 `GET /api/v1/jobs/{'{id}'}`。
              </p>
              <button className="primary-button primary-button--full" disabled={createJobMutation.isPending} type="submit">
                {createJobMutation.isPending ? "正在创建任务..." : "生成剧本草稿"}
              </button>
            </div>
          </form>
        </FormProvider>

        <div className="panel panel--status">
          <div className="panel__header">
            <div>
              <p className="panel__eyebrow">Job Status</p>
              <h2>任务进度带</h2>
            </div>
            <span className="panel__badge panel__badge--muted">2s polling</span>
          </div>
          <JobStatusPanel
            canRegenerate={activeJob?.status === "failed" && !createJobMutation.isPending}
            createError={getErrorMessage(createJobMutation.error)}
            isCreating={createJobMutation.isPending}
            isPolling={jobDetailsQuery.isFetching}
            hasJobId={Boolean(currentJobId)}
            job={activeJob}
            onRegenerate={handleRegenerate}
            resultError={resultErrorMessage}
            stages={stages}
          />
          <div className="panel-section">
            <div className="section-heading section-heading--tight">
              <div>
                <h3>联调状态</h3>
                <p>这里用于确认当前 UI 是否已经脱离静态 mock。</p>
              </div>
              <span className="section-tag">Debug</span>
            </div>
            <p className="inline-note">
              {currentJobId ? `当前任务：${statusNote}` : "当前还没有活动任务。"}
            </p>
          </div>
        </div>

        <div className="panel panel--result">
          <div className="panel__header">
            <div>
              <p className="panel__eyebrow">Result Workspace</p>
              <h2>剧本初稿与结构摘要</h2>
            </div>
            <span className="panel__badge panel__badge--accent">{resultBadgeLabel}</span>
          </div>
          <ExportActions
            canExport={Boolean(editedYamlText.trim())}
            canReset={Boolean(originalYamlText)}
            hasJob={Boolean(currentJobId)}
            jobId={currentJobId ?? undefined}
            onDownloadBackendRaw={handleDownloadBackendRaw}
            onDownloadCurrent={handleExportCurrentYaml}
            onReset={handleResetYaml}
          />
          <YamlEditor
            errorMessage={getErrorMessage(jobResultQuery.error)}
            isLoading={jobResultQuery.isLoading}
            jobId={currentJobId}
            jobStatus={activeJobStatus}
            onChange={setEditedYamlText}
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
