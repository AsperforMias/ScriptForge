#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="${1:-deterministic}"
PORT="${PORT:-18080}"
JOB_TIMEOUT_SECONDS="${JOB_TIMEOUT_SECONDS:-60}"
BACKEND_DIR="$ROOT_DIR/backend"
TMP_DIR="$(mktemp -d /tmp/scriptforge-smoke.XXXXXX)"
LOG_PATH="$TMP_DIR/backend.log"
REQUEST_PATH="$TMP_DIR/request.json"
EXPORT_PATH="$TMP_DIR/export.yaml"

SERVER_PID=""

cleanup() {
  if [[ -n "$SERVER_PID" ]] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
}

trap cleanup EXIT

usage() {
  cat <<'EOF'
Usage:
  scripts/run_backend_smoke.sh [deterministic|llm]

Environment overrides:
  PORT=<port>                   Temporary backend listen port. Default: 18080
  JOB_TIMEOUT_SECONDS=<seconds> Poll timeout. Default: 60

Notes:
  - llm mode expects repo-root .env.local to exist and contain valid provider vars.
  - deterministic mode does not require any external provider setup.
EOF
}

if [[ "$MODE" != "deterministic" && "$MODE" != "llm" ]]; then
  usage
  exit 1
fi

if [[ "$MODE" == "llm" ]]; then
  if [[ ! -f "$ROOT_DIR/.env.local" ]]; then
    echo "missing $ROOT_DIR/.env.local"
    echo "copy .env.local.example to .env.local and fill your provider key first"
    exit 1
  fi

  set -a
  # shellcheck disable=SC1091
  source "$ROOT_DIR/.env.local"
  set +a
fi

cp "$ROOT_DIR/testdata/novels/night-rain-request.json" "$REQUEST_PATH"
if [[ "$MODE" == "llm" ]]; then
  perl -0pi -e 's/"mode": "deterministic"/"mode": "llm"/' "$REQUEST_PATH"
fi

(
  cd "$BACKEND_DIR"
  GOCACHE=/tmp/scriptforge-gocache HTTP_ADDR=":$PORT" go run ./cmd/api >"$LOG_PATH" 2>&1
) &
SERVER_PID="$!"

for _ in $(seq 1 30); do
  if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    echo "backend exited before becoming healthy"
    cat "$LOG_PATH"
    exit 1
  fi
  if curl -fsS "http://127.0.0.1:$PORT/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if ! curl -fsS "http://127.0.0.1:$PORT/healthz" >/dev/null 2>&1; then
  echo "backend did not become healthy on port $PORT"
  cat "$LOG_PATH"
  exit 1
fi

create_response="$(curl -fsS -X POST "http://127.0.0.1:$PORT/api/v1/jobs" \
  -H 'Content-Type: application/json' \
  --data-binary @"$REQUEST_PATH")"
job_id="$(printf '%s' "$create_response" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')"

if [[ -z "$job_id" ]]; then
  echo "failed to parse job id from create response"
  echo "$create_response"
  exit 1
fi

deadline=$((SECONDS + JOB_TIMEOUT_SECONDS))
status_response=""
while (( SECONDS < deadline )); do
  status_response="$(curl -fsS "http://127.0.0.1:$PORT/api/v1/jobs/$job_id")"
  if printf '%s' "$status_response" | grep -q '"status":"succeeded"'; then
    break
  fi
  if printf '%s' "$status_response" | grep -q '"status":"failed"'; then
    echo "job failed"
    echo "$status_response"
    cat "$LOG_PATH"
    exit 1
  fi
  sleep 2
done

if ! printf '%s' "$status_response" | grep -q '"status":"succeeded"'; then
  echo "job did not finish within ${JOB_TIMEOUT_SECONDS}s"
  echo "$status_response"
  cat "$LOG_PATH"
  exit 1
fi

result_response="$(curl -fsS "http://127.0.0.1:$PORT/api/v1/jobs/$job_id/result")"
curl -fsS "http://127.0.0.1:$PORT/api/v1/jobs/$job_id/export" >"$EXPORT_PATH"

echo "smoke mode: $MODE"
echo "job id: $job_id"
echo "export path: $EXPORT_PATH"
echo "backend log: $LOG_PATH"

provider_debug_path="$BACKEND_DIR/tmp/artifacts/$job_id/provider_debug.json"
if [[ -f "$provider_debug_path" ]]; then
  echo "provider debug: $provider_debug_path"
fi

printf '%s\n' "$result_response" | sed -n '1,20p'
