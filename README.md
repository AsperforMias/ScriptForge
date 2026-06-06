# ScriptForge

AI-powered Novel-to-Screenplay Workspace.

This repository is being initialized for a 72-hour training-camp project. The implementation baseline is document-first: product scope, architecture decisions, progress tracking, PR rules, and handoff context live under [`docs/`](/Users/asperformias/Code/github/ScriptForge/docs/README.md).

Key links:
- [YAML Schema and design rationale](docs/yaml-schema.md)
- [Frontend demo recording guide](docs/demo-recording-guide.md)

Read in this order before making changes:
1. [`docs/final-solution.md`](/Users/asperformias/Code/github/ScriptForge/docs/final-solution.md)
2. [`docs/implementation-progress.md`](/Users/asperformias/Code/github/ScriptForge/docs/implementation-progress.md)
3. [`docs/backend-architecture.md`](/Users/asperformias/Code/github/ScriptForge/docs/backend-architecture.md)
4. [`docs/backend-tech-stack.md`](/Users/asperformias/Code/github/ScriptForge/docs/backend-tech-stack.md)
5. [`docs/api-contract.md`](/Users/asperformias/Code/github/ScriptForge/docs/api-contract.md)
6. [`docs/backend-pipeline.md`](/Users/asperformias/Code/github/ScriptForge/docs/backend-pipeline.md)
7. [`docs/frontend.md`](/Users/asperformias/Code/github/ScriptForge/docs/frontend.md)
8. [`docs/yaml-schema.md`](/Users/asperformias/Code/github/ScriptForge/docs/yaml-schema.md)
9. [`docs/architecture-self-check.md`](/Users/asperformias/Code/github/ScriptForge/docs/architecture-self-check.md)
10. [`docs/collaboration-rules.md`](/Users/asperformias/Code/github/ScriptForge/docs/collaboration-rules.md)

Current state:
- Documentation baseline: ready and executable
- Backend implementation: deterministic pipeline and `llm` provider path are both runnable
- Goal: keep future human/agent sessions aligned to the same scope and judging constraints

Current runnable ability:
- `backend/` exposes `POST /api/v1/jobs`
- background deterministic pipeline persists job status and YAML artifacts
- `GET /api/v1/jobs/:id`, `GET /api/v1/jobs/:id/result`, and `GET /api/v1/jobs/:id/export` are available
- `frontend/` now runs a Vite + React + TypeScript editorial workspace with real manual multi-chapter input, job polling, YAML result loading, structured summary, and export actions
- failed jobs can be regenerated from the current frontend form without adding a separate retry API
- frontend sample presets now cover suspense, workplace, and campus relay source scenarios
- the workspace copy is now author-facing; demo narration and walkthrough notes live in `docs/demo-recording-guide.md` instead of the product page
- the workspace now exposes explicit idle/loading/succeeded/failed copy and remains readable across desktop, tablet, and mobile layouts
- the result workspace now distinguishes backend-original vs local-edited YAML, supports copy/reset/export feedback, and adds screenplay overview cards from backend JSON
- `generation.mode=llm` now supports `mock` and `openai_compatible` providers behind the same job API
- the `openai_compatible` path has been validated against DeepSeek-compatible `/chat/completions` and normalizes loose provider YAML into the canonical project schema
- fixture-backed integration tests cover create, status, result, export, invalid input, not-ready, and llm mock behavior

Backend quick start:
```bash
cd backend
go run ./cmd/api
```

Frontend quick start:
```bash
cd frontend
npm install
npm run dev
```

Frontend smoke-check:
```bash
cd frontend
npm run smoke:workspace
```

Default local demo ports:
- backend API: `http://127.0.0.1:8080`
- frontend dev server: `http://127.0.0.1:5173`
- Vite dev proxy forwards `/api/*` to `http://127.0.0.1:8080` by default

Recommended local startup:
1. Start the backend from the repo root or `backend/`; the default `HTTP_ADDR` is `:8080`.
2. Start the frontend with `npm run dev`; Vite serves the workspace on `:5173`.
3. Open `http://127.0.0.1:5173`; frontend requests to `/api/v1/*` are proxied to the backend automatically.

Demo walkthrough note:
- Use [docs/demo-recording-guide.md](docs/demo-recording-guide.md) for presenter-facing narration, sample order, and recording flow. The product page itself stays focused on author use rather than judge instructions.

Frontend real-chain self-check:
1. Start the backend on `http://127.0.0.1:8080` and the frontend on `http://127.0.0.1:5173`.
2. Open the workspace; the recommended `职场` sample is already loaded for quick start, but you can also click `切换为空白手工输入` and paste your own 3 chapters directly.
3. Keep `generationMode=deterministic`, then click the primary submit action to create a real job through `POST /api/v1/jobs`.
4. Watch the center `Job Status` column until polling moves the job from `queued/running` to `succeeded`.
5. Confirm the right-side result area loads real backend data: YAML text, structured screenplay summary, and export actions.
6. Use the export actions to verify both `下载生成初稿 YAML` and `导出 YAML` paths.
7. Optional failed-path check: switch the form to `generationMode=llm` while the backend runs with `LLM_PROVIDER=disabled`, submit once, confirm the failed stage message appears, then click `重新生成当前内容` to verify the frontend creates a fresh job from the same form state.
8. Narrow the viewport to a tablet or mobile width and confirm the workspace collapses into a readable `Input -> Status -> Result` vertical flow.
9. After a successful result load, modify the YAML once, confirm the toolbar flips from `当前为生成初稿` to `当前为本地编辑稿`, then test `复制当前 YAML` and `恢复生成初稿`.
10. Run one extra non-preset pass: click `切换为空白手工输入`, enter your own 3 chapters, then repeat `create job -> polling -> YAML/result/export` to confirm the main path does not depend on built-in samples.

Scripted frontend smoke-check:
- `npm run smoke:workspace` expects the backend on `:8080`, the frontend dev server on `:5173`, and a local Chrome or Edge executable.
- It verifies two real frontend acceptance paths: a sample preset run and a non-preset manual 3-chapter run, both covering real `POST /api/v1/jobs`, polling, YAML load, structured summary, export, local edit, `复制当前 YAML`, failed-job regenerate, `lastJobId` refresh restore, and mobile `Input -> Status -> Result` panel order.
- Optional overrides:
  - `FRONTEND_SMOKE_UI_URL`
  - `FRONTEND_SMOKE_BACKEND_HEALTH_URL`
  - `FRONTEND_SMOKE_SAMPLE_LABEL`
  - `FRONTEND_SMOKE_CHROME_PATH`
  - `FRONTEND_SMOKE_TIMEOUT_MS`

Frontend API note:
```bash
# bash / zsh: optional when frontend and backend are on different origins
export VITE_API_BASE_URL=http://localhost:8080/api/v1

# bash / zsh: optional when the backend is not on the default local port
export VITE_API_PROXY_TARGET=http://127.0.0.1:8080
```

Local/deployment prerequisite for real provider runs:
```bash
cp .env.local.example .env.local
# edit .env.local and fill your real key
set -a && source .env.local && set +a
```

Backend quick start with a local external provider:
```bash
cd backend
go run ./cmd/api
```

LLM mode options:
```bash
# local verification without external network
export LLM_PROVIDER=mock

# vendor-neutral external provider wiring
export LLM_PROVIDER=openai_compatible
export LLM_BASE_URL=https://your-provider.example/v1
export LLM_MODEL=your-model-name
export LLM_API_KEY=your-api-key
```

Local secret handling:
- keep provider credentials in a repo-root `.env.local`
- `.env.local` is gitignored and must never be committed
- start from `.env.local.example` so other human/agent sessions inherit the same expected variable names
- `openai_compatible` has been validated with DeepSeek using `deepseek-v4-flash` as the current low-cost chain-test model
- DeepSeek official docs currently list the OpenAI-format base URL as `https://api.deepseek.com`, `/chat/completions` as the chat endpoint, and `deepseek-v4-flash` / `deepseek-v4-pro` as the active model IDs
- reference docs:
  - [DeepSeek Create Chat Completion](https://api-docs.deepseek.com/api/create-chat-completion)
  - [DeepSeek Models & Pricing](https://api-docs.deepseek.com/quick_start/pricing)
  - [DeepSeek JSON Output](https://api-docs.deepseek.com/guides/json_mode)

OpenAI-compatible provider note:
- keep `LLM_PROVIDER=openai_compatible`
- swap only `LLM_BASE_URL`, `LLM_MODEL`, and `LLM_API_KEY` when moving from DeepSeek to another compatible provider
- the current backend path remains YAML-first because the project output contract is YAML, even though DeepSeek also documents JSON Output support

Current backend focus:
- extend regression coverage for real-world loose YAML variants returned by `openai_compatible` providers
- continue polishing provider prompt/normalization quality while keeping the current YAML-first output contract
- provider fixture coverage now includes fenced YAML with explanatory preface, `message.content` text-part arrays, and loose YAML that relies on planned metadata/entity fallback
- failure regressions now explicitly cover `job_not_found`, `job_not_ready`, and export-not-ready behavior across service, HTTP, and SQLite store layers

Backend self-check:
```bash
cd backend
GOCACHE=/tmp/scriptforge-gocache go test ./...
GOCACHE=/tmp/scriptforge-gocache go build -o /tmp/scriptforge-api ./cmd/api
```

Backend acceptance note:
- run backend self-checks from `backend/`; the repo root is not a Go module root
- keep YAML fixture regressions, but do not treat fixtures as the only acceptance target
- at least one regression should cover a custom Chinese 3-chapter input through the real job pipeline rather than only comparing canned fixtures

Backend smoke-check:
```bash
# deterministic local path
scripts/run_backend_smoke.sh deterministic

# real provider path (requires .env.local)
scripts/run_backend_smoke.sh llm

# pick a specific demo fixture
scripts/run_backend_smoke.sh deterministic family
scripts/run_backend_smoke.sh deterministic comedy
```

Example fixture inputs:
- [`testdata/novels/night-rain-request.json`](/Users/asperformias/Code/github/ScriptForge/testdata/novels/night-rain-request.json)
- [`testdata/novels/workplace-crisis-request.json`](/Users/asperformias/Code/github/ScriptForge/testdata/novels/workplace-crisis-request.json)
- [`testdata/novels/campus-relay-request.json`](/Users/asperformias/Code/github/ScriptForge/testdata/novels/campus-relay-request.json)
- [`testdata/novels/family-dinner-request.json`](/Users/asperformias/Code/github/ScriptForge/testdata/novels/family-dinner-request.json)
- [`testdata/novels/comedy-live-mixup-request.json`](/Users/asperformias/Code/github/ScriptForge/testdata/novels/comedy-live-mixup-request.json)

Example expected outputs:
- [`testdata/expected/night-rain.screenplay.yaml`](/Users/asperformias/Code/github/ScriptForge/testdata/expected/night-rain.screenplay.yaml)
- [`testdata/expected/workplace-crisis.screenplay.yaml`](/Users/asperformias/Code/github/ScriptForge/testdata/expected/workplace-crisis.screenplay.yaml)
- [`testdata/expected/campus-relay.screenplay.yaml`](/Users/asperformias/Code/github/ScriptForge/testdata/expected/campus-relay.screenplay.yaml)
- [`testdata/expected/family-dinner.screenplay.yaml`](/Users/asperformias/Code/github/ScriptForge/testdata/expected/family-dinner.screenplay.yaml)
- [`testdata/expected/comedy-live-mixup.screenplay.yaml`](/Users/asperformias/Code/github/ScriptForge/testdata/expected/comedy-live-mixup.screenplay.yaml)

Initial repository layout:
```text
docs/
.github/
backend/
frontend/
scripts/
testdata/
deploy/
```

Use [`docs/README.md`](/Users/asperformias/Code/github/ScriptForge/docs/README.md) as the main handoff and intake index.
