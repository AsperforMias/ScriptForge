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

    logStep("opening workspace and clearing remembered job state");
    await page.goto(uiUrl, { waitUntil: "networkidle" });
    await page.evaluate(() => {
      window.localStorage.removeItem("scriptforge:lastJobId");
    });
    await page.reload({ waitUntil: "networkidle" });

    logStep(`loading sample preset: ${sampleLabel}`);
    await page.getByRole("button", { name: sampleLabel }).click();

    logStep("creating a real deterministic job from the frontend");
    await page.getByRole("button", { name: "生成剧本草稿" }).click();

    logStep("waiting for result workspace to load real YAML and summary data");
    await page.waitForFunction(() => {
      const area = document.querySelector(".yaml-editor");
      return area instanceof HTMLTextAreaElement && area.value.trim().length > 0;
    });
    await page.waitForFunction(() => {
      const sceneCards = document.querySelectorAll(".scene-card");
      return sceneCards.length > 0;
    });

    const originalYaml = await page.locator(".yaml-editor").inputValue();
    await page.locator(".yaml-editor").fill(`${originalYaml}\n# frontend smoke local edit`);

    const hasDraftTag = (await page.locator(".result-toolbar__draft-tag").count()) > 0;
    if (hasDraftTag) {
      await page.waitForFunction(() => {
        const tag = document.querySelector(".result-toolbar__draft-tag");
        return tag?.textContent?.includes("本地编辑稿");
      });
    }

    logStep("verifying reset restores the backend-original YAML");
    await page.getByRole("button", { name: "恢复后端原始结果" }).click();
    await page.waitForFunction(() => {
      const area = document.querySelector(".yaml-editor");

      return (
        area instanceof HTMLTextAreaElement &&
        !area.value.includes("# frontend smoke local edit")
      );
    });

    const sceneCount = await page.locator(".scene-card").count();
    const overviewCount = await page.locator(".summary-overview__card").count();
    const hasCopyButton = (await page.getByRole("button", { name: "复制当前 YAML" }).count()) > 0;

    if (sceneCount < 1) {
      throw new Error(
        `unexpected summary shape: sceneCount=${sceneCount}, overviewCount=${overviewCount}`,
      );
    }

    logStep("frontend workspace smoke-check passed");
    process.stdout.write(
      [
        "",
        `sample=${sampleLabel}`,
        `sceneCount=${sceneCount}`,
        `overviewCount=${overviewCount}`,
        `hasDraftTag=${hasDraftTag}`,
        `hasCopyButton=${hasCopyButton}`,
        "checks=frontend load, create job, polling, yaml load, local edit, reset, scene summary",
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
