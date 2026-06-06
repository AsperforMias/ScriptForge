# Docs Intake

This directory is the single source of truth for project scope, progress, and collaboration rules. Human contributors and AI agents should read these files before changing code.

Recommended read order:
1. [`final-solution.md`](final-solution.md): final target, scope boundaries, architecture shape, module responsibilities
2. [`implementation-progress.md`](implementation-progress.md): current delivery status, remaining work, next priorities
3. [`decision-log.md`](decision-log.md): locked decisions, deferred decisions, and rationale
4. [`backend-architecture.md`](backend-architecture.md): Go service structure, middleware stack, storage choices, and competitive backend shape
5. [`backend-tech-stack.md`](backend-tech-stack.md): concrete library choices, config, SQLite schema, artifact layout, and testing rules
6. [`api-contract.md`](api-contract.md): required request and response shapes
7. [`backend-pipeline.md`](backend-pipeline.md): stage-by-stage execution contract for the generation pipeline
8. [`frontend.md`](frontend.md): frontend feature contract, user paths, interaction boundaries
9. [`frontend-visual-direction.md`](frontend-visual-direction.md): locked visual language, layout priority, and UI constraints for the frontend workspace
10. [`yaml-schema.md`](yaml-schema.md): screenplay YAML schema and design rationale
11. [`demo-recording-guide.md`](demo-recording-guide.md): presenter-facing demo order, narration cues, and recording checks that should stay out of the product page
12. [`milestones.md`](milestones.md): definition of done per phase and suggested PR breakdown
13. [`competition-brief.md`](competition-brief.md): filtered contest requirements and judging implications
14. [`architecture-self-check.md`](architecture-self-check.md): alignment audit against the prompt and judging criteria
15. [`collaboration-rules.md`](collaboration-rules.md): PR, commit, branch, pairing, and agent operating rules

Current runnable ability:
- The backend job API and YAML result pipeline are runnable locally.
- `generation.mode=llm` supports both `mock` and `openai_compatible` provider paths.
- The `openai_compatible` path has been validated against a DeepSeek-compatible endpoint and now normalizes loose provider YAML into the canonical screenplay schema.
- Corrected project direction: `llm` is the intended main generation path; `deterministic` should be treated as fallback / smoke baseline instead of the long-term primary generator.

Startup and deployment:
- Local-first delivery is the default target.
- Primary demo target is a locally runnable web app with separate `backend/` and `frontend/` modules.
- Public deployment is optional; reproducible local startup matters more for the first MVP.
- For any real-provider run, start from repo-root `.env.local.example`, copy it to `.env.local`, fill credentials, then `set -a && source .env.local && set +a`.
- Preferred smoke-check entry is `scripts/run_backend_smoke.sh`, which can verify both `deterministic` and `llm` job paths without manual curl steps.

Self-check entry before coding:
1. Confirm your planned change is in scope according to `final-solution.md`.
2. Confirm the status in `implementation-progress.md` is still accurate; update it if you change delivery state.
3. Confirm the relevant decision is already locked in `decision-log.md`; if not, add it before broad implementation.
4. Confirm API, pipeline, and tech-stack changes match `api-contract.md`, `backend-pipeline.md`, and `backend-tech-stack.md`.
5. Confirm your change is a single-purpose PR according to `collaboration-rules.md`.
6. If the change affects screenplay output, validate against `yaml-schema.md`.
7. If your change keeps expanding deterministic template logic, stop and verify that it still matches the corrected `llm-first` product direction.

Agent session rules:
- Treat these docs as higher-priority project context than unstated assumptions.
- If a decision is not covered here, add or update the relevant doc before making a broad architectural change.
- When handing off work, update `implementation-progress.md` first.
- If `decision-log.md` marks an item as requiring human input, stop and ask instead of guessing.
- Keep local provider credentials in repo-root `.env.local`; it is gitignored and must not appear in commits, PR text, or docs.
- Prefer repo-root `.env.local.example` as the handoff template so new agent sessions inherit the same provider variable contract.

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
