# ScriptForge

AI-powered Novel-to-Screenplay Workspace.

This repository is being initialized for a 72-hour training-camp project. The implementation baseline is document-first: product scope, architecture decisions, progress tracking, PR rules, and handoff context live under [`docs/`](/Users/asperformias/Code/github/ScriptForge/docs/README.md).

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
- Backend implementation: phase 2 MVP is in place
- Goal: keep future human/agent sessions aligned to the same scope and judging constraints

Current runnable ability:
- `backend/` exposes `POST /api/v1/jobs`
- background deterministic pipeline persists job status and YAML artifacts
- `GET /api/v1/jobs/:id`, `GET /api/v1/jobs/:id/result`, and `GET /api/v1/jobs/:id/export` are available
- fixture-backed integration tests cover create, status, result, export, invalid input, and not-ready behavior

Backend quick start:
```bash
cd backend
go run ./cmd/api
```

Backend self-check:
```bash
cd backend
go test ./...
go build -o /tmp/scriptforge-api ./cmd/api
```

Example fixture inputs:
- [`testdata/novels/night-rain-request.json`](/Users/asperformias/Code/github/ScriptForge/testdata/novels/night-rain-request.json)
- [`testdata/expected/night-rain.screenplay.yaml`](/Users/asperformias/Code/github/ScriptForge/testdata/expected/night-rain.screenplay.yaml)

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
