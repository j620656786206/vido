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
for arg in "$@"; do
  case "$arg" in
    --skip-build) SKIP_BUILD=1 ;;
    --keep-db) KEEP_DB=1 ;;
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
  (cd "$ROOT/apps/api" && go run ./cmd/seed --data-dir "$ENV_DIR/data" --media-root "$ENV_DIR/media" --reset)
else
  echo "==> Keeping existing database"
fi

echo "==> Serving Vido test env on http://localhost:$PORT (Ctrl-C to stop)"
cd "$ROOT/apps/api"
DB_PATH="$ENV_DIR/data/vido.db" \
  VIDO_DATA_DIR="$ENV_DIR/data" \
  VIDO_MEDIA_DIRS="$ENV_DIR/media/movies,$ENV_DIR/media/tv" \
  VIDO_PUBLIC_DIR="$ROOT/dist/apps/web" \
  VIDO_PORT="$PORT" \
  exec go run ./cmd/api
