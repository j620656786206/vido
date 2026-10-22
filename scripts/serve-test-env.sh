#!/usr/bin/env bash
# serve-test-env.sh — one-command LOCAL test environment for Vido
# (story: testenv-local-seed, resolves retro-8-TS2).
#
# Builds the production frontend, seeds a deterministic local sqlite fixture DB
# (via apps/api/cmd/seed), and serves BOTH through the Go API — the same
# single-process shape as the Docker image. This is the ONLY sanctioned target
# for TestSprite runs and manual UX verification; never point tests at the NAS.
#
# Usage:
#   scripts/serve-test-env.sh                # build + reseed + serve on :8090
#   scripts/serve-test-env.sh --skip-build   # reuse existing dist/apps/web
#   scripts/serve-test-env.sh --keep-db      # keep the current seeded DB
#   scripts/serve-test-env.sh --failure-fixtures # add homepage failure-state data
#   scripts/serve-test-env.sh --with-encryption-key # let 金鑰設定 save keys
#
# Without --with-encryption-key the server has no ENCRYPTION_KEY, so 金鑰設定
# is read-only (TC132 relies on that). With it, a fixed 32-character test-only
# key is set, so the "only a Gemini key" journeys can save a key. The key is
# a constant on purpose: the seeded DB is rebuilt every run, nothing real is
# ever encrypted with it. Without the flag ENCRYPTION_KEY is unset even if your
# shell exported one, so the default run is always the read-only case.
#
# Env overrides:
#   VIDO_TEST_ENV_DIR  (default: <repo>/.vido-test-env)
#   VIDO_TEST_PORT     (default: 8090 — avoids the :8080 dev backend)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_DIR="${VIDO_TEST_ENV_DIR:-$ROOT/.vido-test-env}"
PORT="${VIDO_TEST_PORT:-8090}"

SKIP_BUILD=0
KEEP_DB=0
FAILURE_FIXTURES=0
WITH_ENCRYPTION_KEY=0
for arg in "$@"; do
  case "$arg" in
    --skip-build) SKIP_BUILD=1 ;;
    --keep-db) KEEP_DB=1 ;;
    --failure-fixtures) FAILURE_FIXTURES=1 ;;
    --with-encryption-key) WITH_ENCRYPTION_KEY=1 ;;
    *)
      echo "unknown option: $arg" >&2
      exit 1
      ;;
  esac
done

# Local-only guard: refuse network/NAS-ish mount points outright.
case "$ENV_DIR" in
  /mnt/* | /Volumes/* | //*)
    echo "refusing ENV_DIR '$ENV_DIR' — the test env is LOCAL-only (never the NAS)" >&2
    exit 1
    ;;
esac

mkdir -p "$ENV_DIR/data" "$ENV_DIR/media"

if [[ "$SKIP_BUILD" -eq 0 ]]; then
  echo "==> Building production frontend (nx build web)"
  (cd "$ROOT" && pnpm nx build web)
else
  if [[ ! -d "$ROOT/dist/apps/web" ]]; then
    echo "--skip-build set but dist/apps/web missing — run without --skip-build first" >&2
    exit 1
  fi
  echo "==> Skipping frontend build (reusing dist/apps/web)"
fi

if [[ "$KEEP_DB" -eq 0 ]]; then
  echo "==> Seeding fixture database"
  SEED_FAILURE_ARG=()
  if [[ "$FAILURE_FIXTURES" -eq 1 ]]; then
    SEED_FAILURE_ARG+=(--failure-fixtures)
  fi
  # ${arr[@]+…} guard: macOS bash 3.2 + `set -u` treats expanding an EMPTY
  # array as an unbound variable and aborts the script.
  (cd "$ROOT/apps/api" && go run ./cmd/seed --data-dir "$ENV_DIR/data" --media-root "$ENV_DIR/media" --reset ${SEED_FAILURE_ARG[@]+"${SEED_FAILURE_ARG[@]}"})
else
  echo "==> Keeping existing database"
fi

echo "==> Serving Vido test env on http://localhost:$PORT (Ctrl-C to stop)"
cd "$ROOT/apps/api"
if [[ "$WITH_ENCRYPTION_KEY" == 1 ]]; then
  echo "==> ENCRYPTION_KEY set (test-only key) — 金鑰設定 can save keys"
  export ENCRYPTION_KEY="vido-local-test-env-key-00000000"
else
  unset ENCRYPTION_KEY
fi
DB_PATH="$ENV_DIR/data/vido.db" \
  VIDO_DATA_DIR="$ENV_DIR/data" \
  VIDO_MEDIA_DIRS="$ENV_DIR/media/movies,$ENV_DIR/media/tv" \
  VIDO_PUBLIC_DIR="$ROOT/dist/apps/web" \
  VIDO_PORT="$PORT" \
  exec go run ./cmd/api
