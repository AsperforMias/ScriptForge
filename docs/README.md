# Docs Intake

This directory is the single source of truth for project scope, progress, and collaboration rules. Human contributors and AI agents should read these files before changing code.

Recommended read order:
1. [`final-solution.md`](/Users/asperformias/Code/github/ScriptForge/docs/final-solution.md): final target, scope boundaries, architecture shape, module responsibilities
2. [`implementation-progress.md`](/Users/asperformias/Code/github/ScriptForge/docs/implementation-progress.md): current delivery status, remaining work, next priorities
3. [`decision-log.md`](/Users/asperformias/Code/github/ScriptForge/docs/decision-log.md): locked decisions, deferred decisions, and rationale
4. [`backend-architecture.md`](/Users/asperformias/Code/github/ScriptForge/docs/backend-architecture.md): Go service structure, middleware stack, storage choices, and competitive backend shape
5. [`backend-tech-stack.md`](/Users/asperformias/Code/github/ScriptForge/docs/backend-tech-stack.md): concrete library choices, config, SQLite schema, artifact layout, and testing rules
6. [`api-contract.md`](/Users/asperformias/Code/github/ScriptForge/docs/api-contract.md): required request and response shapes
7. [`backend-pipeline.md`](/Users/asperformias/Code/github/ScriptForge/docs/backend-pipeline.md): stage-by-stage execution contract for the generation pipeline
8. [`frontend.md`](/Users/asperformias/Code/github/ScriptForge/docs/frontend.md): frontend feature contract, user paths, interaction boundaries
9. [`yaml-schema.md`](/Users/asperformias/Code/github/ScriptForge/docs/yaml-schema.md): screenplay YAML schema and design rationale
10. [`milestones.md`](/Users/asperformias/Code/github/ScriptForge/docs/milestones.md): definition of done per phase and suggested PR breakdown
11. [`competition-brief.md`](/Users/asperformias/Code/github/ScriptForge/docs/competition-brief.md): filtered contest requirements and judging implications
12. [`architecture-self-check.md`](/Users/asperformias/Code/github/ScriptForge/docs/architecture-self-check.md): alignment audit against the prompt and judging criteria
13. [`collaboration-rules.md`](/Users/asperformias/Code/github/ScriptForge/docs/collaboration-rules.md): PR, commit, branch, pairing, and agent operating rules

Current runnable ability:
- No application service is intentionally started in this initialization session.
- The repository is in document-first bootstrap state with locked backend execution contracts.
- The next implementation session should begin from `implementation-progress.md`, not from assumptions.

Startup and deployment:
- Local-first delivery is the default target.
- Primary demo target is a locally runnable web app with separate `backend/` and `frontend/` modules.
- Public deployment is optional; reproducible local startup matters more for the first MVP.

Self-check entry before coding:
1. Confirm your planned change is in scope according to `final-solution.md`.
2. Confirm the status in `implementation-progress.md` is still accurate; update it if you change delivery state.
3. Confirm the relevant decision is already locked in `decision-log.md`; if not, add it before broad implementation.
4. Confirm API, pipeline, and tech-stack changes match `api-contract.md`, `backend-pipeline.md`, and `backend-tech-stack.md`.
5. Confirm your change is a single-purpose PR according to `collaboration-rules.md`.
6. If the change affects screenplay output, validate against `yaml-schema.md`.

Agent session rules:
- Treat these docs as higher-priority project context than unstated assumptions.
- If a decision is not covered here, add or update the relevant doc before making a broad architectural change.
- When handing off work, update `implementation-progress.md` first.
- If `decision-log.md` marks an item as requiring human input, stop and ask instead of guessing.

Initial repository structure:
```text
docs/                    Project source of truth
.github/                 PR template and workflow metadata
backend/                 Go-first backend service and pipeline
frontend/                Frontend application owned primarily by the frontend teammate
scripts/                 Utility scripts, fixtures, local automation
testdata/                Sample novel inputs, expected YAML outputs, regression fixtures
deploy/                  Deployment notes or manifests if needed later
```
