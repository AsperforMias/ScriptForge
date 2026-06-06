import { existsSync } from "node:fs";
import { spawnSync } from "node:child_process";
import os from "node:os";
import process from "node:process";
import { chromium } from "playwright-core";

const uiUrl = process.env.FRONTEND_SMOKE_UI_URL ?? "http://127.0.0.1:5173";
const backendHealthUrl =
  process.env.FRONTEND_SMOKE_BACKEND_HEALTH_URL ?? "http://127.0.0.1:8080/healthz";
const sampleLabel = process.env.FRONTEND_SMOKE_SAMPLE_LABEL ?? "职场";
const timeoutMs = Number(process.env.FRONTEND_SMOKE_TIMEOUT_MS ?? "60000");
const generateButtonLabel = "生成剧本初稿";
const resetDraftButtonLabel = "恢复生成初稿";
const downloadOriginalButtonLabel = "下载生成初稿 YAML";
const exportCurrentButtonLabel = "导出 YAML";
const blankInputButtonLabel = "切换为空白手工输入";
const localDraftTagLabel = "当前为本地编辑稿";

const manualDraft = {
  title: "潮汐尽头",
  author: "林渡",
  style: "都市情感短剧",
  audience: "女性向",
  notesText: "保留双线情绪推进\n控制主要场景数量",
  chapters: [
    {
      title: "第一章 旧码头来信",
      content:
        "沈知返乡处理母亲留下的仓库合约，却在旧码头收到一封没有署名的来信。信里提醒她不要把仓库卖给航运公司。",
    },
    {
      title: "第二章 茶馆对谈",
      content:
        "她约青梅周屿在老茶馆见面，才知道周屿这些年一直替母亲保管账本。周屿却怀疑沈知这次回城只是为了尽快套现离开。",
    },
    {
      title: "第三章 暴雨夜开仓",
      content:
        "暴雨来临前，沈知独自赶到仓库核对货单，发现母亲当年的货运事故并非意外。她决定连夜把真相和合约一起带到第二天的签约现场。",
    },
  ],
};

function logStep(message) {
  process.stdout.write(`\n[frontend-smoke] ${message}\n`);
}

async function assertReachable(url) {
  const response = await fetch(url);

  if (!response.ok) {
    throw new Error(`${url} returned ${response.status}`);
  }
}

function firstExistingPath(candidates) {
  return candidates.find((candidate) => candidate && existsSync(candidate)) ?? null;
}

function resolveChromeFromCommand(command, args) {
  const result = spawnSync(command, args, {
    encoding: "utf8",
    stdio: ["ignore", "pipe", "ignore"],
  });

  if (result.status !== 0) {
    return null;
  }

  const match = result.stdout
    .split(/\r?\n/)
    .map((line) => line.trim())
    .find(Boolean);

  return match && existsSync(match) ? match : null;
}

function resolveChromeExecutable() {
  if (process.env.FRONTEND_SMOKE_CHROME_PATH && existsSync(process.env.FRONTEND_SMOKE_CHROME_PATH)) {
    return process.env.FRONTEND_SMOKE_CHROME_PATH;
  }

  const platform = os.platform();

  if (platform === "win32") {
    return (
      firstExistingPath([
        "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
        "C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
        `${process.env.LOCALAPPDATA}\\Google\\Chrome\\Application\\chrome.exe`,
        `${process.env.ProgramFiles}\\Microsoft\\Edge\\Application\\msedge.exe`,
        `${process.env["ProgramFiles(x86)"]}\\Microsoft\\Edge\\Application\\msedge.exe`,
      ]) ?? resolveChromeFromCommand("where", ["chrome"])
    );
  }

  if (platform === "darwin") {
    return (
      firstExistingPath([
        "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
        "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
      ]) ?? resolveChromeFromCommand("which", ["google-chrome"])
    );
  }

  return (
    firstExistingPath([
      "/usr/bin/google-chrome",
      "/usr/bin/google-chrome-stable",
      "/usr/bin/chromium",
      "/usr/bin/chromium-browser",
      "/usr/bin/microsoft-edge",
    ]) ?? resolveChromeFromCommand("which", ["google-chrome"])
  );
}

async function clearRememberedJob(page) {
  logStep("opening workspace and clearing remembered job state");
  await page.goto(uiUrl, { waitUntil: "networkidle" });
  await page.evaluate(() => {
    window.localStorage.removeItem("scriptforge:lastJobId");
  });
  await page.reload({ waitUntil: "networkidle" });
}

async function waitForPollingToStart(page) {
  await page.waitForFunction(() => {
    const badge = document.querySelector(".status-badge");
    return badge?.textContent?.includes("处理中") || badge?.textContent?.includes("已完成");
  });
}

async function waitForResultWorkspace(page, expectedMarker) {
  await page.waitForFunction(() => {
    const badge = document.querySelector(".status-badge");
    return badge?.textContent?.includes("已完成");
  });
  await page.waitForFunction((marker) => {
    const area = document.querySelector(".yaml-editor");
    return (
      area instanceof HTMLTextAreaElement &&
      area.value.trim().length > 0 &&
      area.value.includes(marker)
    );
  }, expectedMarker);
  await page.waitForFunction(() => {
    return document.querySelectorAll(".summary-overview__card").length >= 4;
  });
  await page.waitForFunction(() => {
    return document.querySelectorAll(".scene-card").length > 0;
  });
}

async function waitForFreshRun(page) {
  const yamlEditor = page.locator(".yaml-editor");

  if ((await yamlEditor.count()) === 0) {
    return;
  }

  const previousValue = await yamlEditor.inputValue();

  if (!previousValue.trim()) {
    return;
  }

  await page.waitForFunction(() => {
    const area = document.querySelector(".yaml-editor");
    return area instanceof HTMLTextAreaElement && area.value.trim().length === 0;
  });
}

async function verifyDownload(page, buttonLabel, expectedFileSuffix) {
  const [download] = await Promise.all([
    page.waitForEvent("download"),
    page.getByRole("button", { name: buttonLabel }).click(),
  ]);
  const suggestedFilename = await download.suggestedFilename();

  if (!suggestedFilename.endsWith(expectedFileSuffix)) {
    throw new Error(
      `unexpected download filename for ${buttonLabel}: ${suggestedFilename} (expected *${expectedFileSuffix})`,
    );
  }

  return suggestedFilename;
}

async function markLocalEdit(page, suffix) {
  const originalYaml = await page.locator(".yaml-editor").inputValue();
  await page.locator(".yaml-editor").fill(`${originalYaml}\n${suffix}`);
  await page.waitForFunction((label) => {
    const tag = document.querySelector(".result-toolbar__draft-tag");
    return tag?.textContent?.includes(label);
  }, localDraftTagLabel);
}

async function verifyReset(page, editMarker) {
  await page.getByRole("button", { name: resetDraftButtonLabel }).click();
  await page.waitForFunction((marker) => {
    const area = document.querySelector(".yaml-editor");

    return area instanceof HTMLTextAreaElement && !area.value.includes(marker);
  }, editMarker);
}

async function runSamplePresetPath(page) {
  logStep(`loading sample preset: ${sampleLabel}`);
  await page.getByRole("button", { name: sampleLabel }).click();

  logStep("creating a deterministic job from the sample preset path");
  await page.getByRole("button", { name: generateButtonLabel }).click();
  await waitForPollingToStart(page);
  await waitForResultWorkspace(page, "title: 交稿前夜");

  logStep("verifying sample result workspace loads YAML, summary, and original export");
  const originalFilename = await verifyDownload(
    page,
    downloadOriginalButtonLabel,
    ".screenplay.yaml",
  );
  await markLocalEdit(page, "# frontend smoke sample edit");
  await verifyReset(page, "# frontend smoke sample edit");

  return {
    originalFilename,
    sceneCount: await page.locator(".scene-card").count(),
    overviewCount: await page.locator(".summary-overview__card").count(),
  };
}

async function fillManualDraft(page) {
  logStep("switching to blank manual input and filling a custom three-chapter draft");
  await page.getByRole("button", { name: blankInputButtonLabel }).click();

  await page.getByLabel("作品标题").fill(manualDraft.title);
  await page.getByLabel("作者或来源备注").fill(manualDraft.author);
  await page.getByLabel("改编风格").fill(manualDraft.style);
  await page.getByLabel("目标受众").fill(manualDraft.audience);
  await page.getByLabel("补充要求").fill(manualDraft.notesText);

  for (const [index, chapter] of manualDraft.chapters.entries()) {
    await page.getByLabel("章节标题").nth(index).fill(chapter.title);
    await page.getByLabel("章节正文").nth(index).fill(chapter.content);
  }

  const activePresetCount = await page.locator('.sample-preset-card[aria-pressed="true"]').count();
  if (activePresetCount !== 0) {
    throw new Error(`manual draft should not keep a preset selected, found ${activePresetCount}`);
  }
}

async function runManualInputPath(page) {
  await fillManualDraft(page);

  logStep("creating a deterministic job from the non-preset manual input path");
  await page.getByRole("button", { name: generateButtonLabel }).click();
  await waitForFreshRun(page);
  await waitForPollingToStart(page);
  await waitForResultWorkspace(page, `title: ${manualDraft.title}`);

  logStep("verifying manual-input result workspace summary, export, and reset");
  await markLocalEdit(page, "# frontend smoke manual edit");
  const editedFilename = await verifyDownload(
    page,
    exportCurrentButtonLabel,
    ".edited.screenplay.yaml",
  );
  await verifyReset(page, "# frontend smoke manual edit");

  return {
    editedFilename,
    manualSceneCount: await page.locator(".scene-card").count(),
    manualOverviewCount: await page.locator(".summary-overview__card").count(),
  };
}

async function run() {
  logStep(`checking frontend at ${uiUrl}`);
  await assertReachable(uiUrl);

  logStep(`checking backend health at ${backendHealthUrl}`);
  await assertReachable(backendHealthUrl);

  const executablePath = resolveChromeExecutable();
  if (!executablePath) {
    throw new Error(
      "could not find a local Chrome/Edge executable; set FRONTEND_SMOKE_CHROME_PATH to continue",
    );
  }

  logStep(`launching browser with ${executablePath}`);
  const browser = await chromium.launch({
    executablePath,
    headless: true,
  });

  try {
    const page = await browser.newPage();
    page.setDefaultTimeout(timeoutMs);

    await clearRememberedJob(page);

    const sampleResults = await runSamplePresetPath(page);
    const manualResults = await runManualInputPath(page);

    logStep("frontend workspace smoke-check passed");
    process.stdout.write(
      [
        "",
        `sample=${sampleLabel}`,
        `sampleOriginalDownload=${sampleResults.originalFilename}`,
        `sampleSceneCount=${sampleResults.sceneCount}`,
        `sampleOverviewCount=${sampleResults.overviewCount}`,
        `manualTitle=${manualDraft.title}`,
        `manualEditedDownload=${manualResults.editedFilename}`,
        `manualSceneCount=${manualResults.manualSceneCount}`,
        `manualOverviewCount=${manualResults.manualOverviewCount}`,
        "checks=sample preset create job, manual 3-chapter input create job, polling, yaml load, summary, original export, current export, local edit, reset",
        "",
      ].join("\n"),
    );
  } finally {
    await browser.close();
  }
}

run().catch((error) => {
  process.stderr.write(`\n[frontend-smoke] ${error instanceof Error ? error.message : String(error)}\n`);
  process.exitCode = 1;
});
