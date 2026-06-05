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
import { buildApiUrl, requestText } from "../lib/http";
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

const recommendedDemoFlow = [
  `1. Start from the default ${recommendedWorkspaceSamplePreset.label} sample and introduce the left-side source input first.`,
  "2. Keep generationMode=deterministic, create a real job, and let the center column show 2s polling plus stage transitions.",
  "3. Move to the result workspace to explain YAML, structured summary, and the local edit / reset / export actions.",
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
      title: "已载入后端原始结果",
      description: "当前 YAML 编辑区已同步为这次任务返回的剧本草稿，可直接修改、复制或导出。",
    });
  }, [loadedResultJobId, resultPayload]);

  const stages = jobDetails?.stages?.length ? jobDetails.stages : createIdleStages();
  const resultSummary = resultPayload?.screenplay ?? null;
  const activeJobStatus: JobStatus | null = activeJob?.status ?? null;
  const resultErrorMessage = getErrorMessage(jobResultQuery.error || jobDetailsQuery.error);
  const hasEditedChanges = Boolean(originalYamlText) && editedYamlText !== originalYamlText;
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
  const activeSamplePreset =
    workspaceSamplePresets.find((preset) => preset.id === activeSamplePresetId) ??
    recommendedWorkspaceSamplePreset;

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
        setResultNotice({
          tone: "info",
          title: "任务已创建",
          description: "正在等待真实 job pipeline 完成；成功后结果区会自动载入新的 YAML 草稿。",
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
        title: "已下载后端原始 YAML",
        description: "导出内容保持与后端返回完全一致，未包含当前编辑区中的本地修改。",
      });
    } catch (error) {
      setFormError(getErrorMessage(error));
      setResultNotice({
        tone: "error",
        title: "下载后端原始 YAML 失败",
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
      title: "已恢复后端原始结果",
      description: "当前编辑区已放弃本地修改，重新对齐到本次任务的后端 YAML。",
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
      title: "已导出当前编辑稿",
      description: hasEditedChanges
        ? "本次导出包含你在前端编辑区中的本地修改。"
        : "当前导出内容与后端原始 YAML 一致。",
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
        title: "已复制当前 YAML",
        description: hasEditedChanges ? "复制内容包含当前本地修改。" : "复制内容与后端原始 YAML 一致。",
      });
    } catch (error) {
      setResultNotice({
        tone: "error",
        title: "复制 YAML 失败",
        description: getErrorMessage(error),
      });
    }
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
            <span>默认演示样例：{recommendedWorkspaceSamplePreset.label}</span>
            <span>建议模式：deterministic 首轮演示</span>
            <span>当前 API Base: {buildApiUrl("").replace(/\/$/, "")}</span>
          </div>
        </div>
        <div className="demo-flow-card">
          <div className="section-heading section-heading--tight">
            <div>
              <h3>Demo Flow</h3>
              <p>按这个顺序讲解，评委能最快看出这不是静态演示壳子。</p>
            </div>
            <span className="section-tag">90s walkthrough</span>
          </div>
          <ol className="demo-flow-list">
            {recommendedDemoFlow.map((step) => (
              <li key={step}>{step}</li>
            ))}
          </ol>
          <p className="inline-note">当前样例焦点：{activeSamplePreset.demoFocus}</p>
          <p className="inline-note">
            如果需要切换题材，可在输入区改用悬疑或校园运动 preset，但默认演示路径保持统一。
          </p>
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
            <SourceForm activePresetId={activeSamplePresetId} onLoadSample={handleLoadSample} />
            <ChapterList />
            {formError ? <p className="inline-error">{formError}</p> : null}
            <div className="submit-panel">
              <p className="inline-note">
                推荐演示顺序：先讲左侧输入，再点击生成，通过 `POST /api/v1/jobs` 创建任务并观察中栏 2s 轮询。
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
                <h3>演示检查点</h3>
                <p>这里用来确认当前页面正在走真实 API 链路，而不是静态结果或 fake polling。</p>
              </div>
              <span className="section-tag">Live chain</span>
            </div>
            <p className="inline-note">
              {currentJobId ? `当前任务：${statusNote}` : "当前还没有活动任务。默认样例已就位，可直接点击“生成剧本草稿”开始演示。"}
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
            originalYamlText={originalYamlText}
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
