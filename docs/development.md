# Vido 開發指南

> 本專案的開發者文件（Nx monorepo 架構、本地開發、建置、測試）。
> 產品說明見專案根目錄的 [README.md](../README.md)。

Vido is an Nx monorepo with a React frontend and a Go backend, shipped as a single Docker container.

| Project        | Path                | Stack                                               |
| -------------- | ------------------- | --------------------------------------------------- |
| `web`          | `apps/web`          | React 19, Vite 7, TanStack Router/Query, Tailwind 4 |
| `api`          | `apps/api`          | Go 1.25, Gin, SQLite (WAL + FTS5)                   |
| `shared-types` | `libs/shared-types` | TypeScript types shared across the workspace        |

## Prerequisites

- **Node.js** — `lts/iron` (>= 20). The repo has an `.nvmrc`, so `nvm use` picks the right one.
- **pnpm** — v9. This is the workspace package manager; CI runs `pnpm install --frozen-lockfile` and `pnpm-lock.yaml` is the authoritative lockfile.
- **Go** — >= 1.25 (`apps/api/go.mod` targets 1.25.0; CI pins 1.25).
- **ffmpeg / ffprobe** — required for the audio-extraction and subtitle-track features. The Docker image installs them; for local development install them yourself (`brew install ffmpeg`).

## Setup

```bash
nvm use
pnpm install

cp .env.example .env   # configure media paths and API keys
```

> `init.sh` at the repo root predates the pnpm migration and still runs `npm install`. Use `pnpm install` instead until that script is updated.

## Project Structure

```
vido/
├── apps/
│   ├── web/                    # React frontend
│   │   ├── src/
│   │   │   ├── routes/         # TanStack Router file-based routes
│   │   │   ├── components/
│   │   │   └── lib/
│   │   ├── project.json
│   │   └── vite.config.mts
│   │
│   └── api/                    # Go backend
│       ├── cmd/api/            # Entry point (main.go — wiring + route registration)
│       ├── internal/
│       │   ├── handlers/       # HTTP handlers, each with RegisterRoutes()
│       │   ├── services/       # Business logic
│       │   ├── repository/     # DB access
│       │   ├── database/       # Schema + migrations
│       │   ├── models/
│       │   ├── ai/             # LLM + Whisper clients, throttle, budget
│       │   ├── subtitle/       # Providers (ASSRT/OpenSubtitles), scorer, converter
│       │   ├── tmdb/ douban/ wikipedia/   # Metadata sources
│       │   ├── qbittorrent/    # Download client integration
│       │   └── sse/ events/    # Server-sent events
│       ├── go.mod
│       └── project.json
│
├── libs/shared-types/          # Shared TypeScript types (@vido/shared-types)
├── tests/e2e/                  # Playwright specs
├── .env.example                # All supported environment variables
├── docker-compose.yml
├── nx.json
└── project-context.md          # Conventions AI agents and contributors follow
```

## Commands

Run everything through Nx so caching and task dependencies apply.

### Development

```bash
pnpm nx serve api      # Go API on http://localhost:8080 (GIN_MODE=debug)
pnpm nx serve web      # React app on http://localhost:4200
```

The Vite dev server proxies `/api` to `http://localhost:8080`, so run both when working on the frontend.

### Building

```bash
pnpm nx build api      # -> dist/apps/api/main
pnpm nx build web
pnpm nx run-many -t build
```

### Testing

```bash
pnpm nx test api       # go test ./... -v
pnpm nx test web       # Vitest
pnpm nx run-many -t test

pnpm test:e2e          # Playwright, all browser projects
pnpm test:ci           # Playwright, @ci-tagged subset only
```

Run test suites in the foreground — backgrounding them leaves orphaned Vitest workers.

Visual-regression baselines are platform-specific. `-linux` baselines cannot be generated on macOS; when CI reports missing ones it opens a bootstrap PR with the correct baselines, which is the intended way to fix that failure.

### Linting and formatting

```bash
pnpm run lint          # eslint
pnpm run lint:fix
pnpm run format        # prettier --write .
pnpm run format:check  # what CI runs — a formatting miss fails the build
pnpm run lint:all      # nx lint (incl. go vet + staticcheck) + eslint + format:check
```

`pnpm nx lint api` runs `go vet` plus `staticcheck`, installing the pinned staticcheck version on first use.

## Backend Conventions

### Adding an endpoint

Handlers live in `apps/api/internal/handlers/`, each exposing a `RegisterRoutes(group *gin.RouterGroup)`. Register the handler in `apps/api/cmd/api/main.go`, where the `/api/v1` group is assembled. Route order matters in a few places — literal paths must be registered before conflicting `:param` routes, and the existing call sites carry comments where that applies.

### Response envelope

All handlers use the shared helpers in `apps/api/internal/handlers/response.go`:

```json
// Success
{ "success": true, "data": {} }

// Error
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid request",
    "suggestion": "optional hint for the UI"
  }
}
```

Error codes are namespaced by source (e.g. `TMDB_`, `TRANSCRIPTION_`, `AI_`). `project-context.md` holds the registry — add new codes there when you introduce them.

There is no swaggo/OpenAPI generation in `apps/api`; the `api/openapi.json` at the repo root is a leftover from the original scaffold and is not regenerated.

## Configuration

All environment variables are documented in [`.env.example`](../.env.example) at the repo root. The ones you are most likely to need locally:

| Variable                                            | Default      | Purpose                                             |
| --------------------------------------------------- | ------------ | --------------------------------------------------- |
| `VIDO_PORT`                                         | `8080`       | API port                                            |
| `VIDO_MEDIA_DIRS`                                   | `/media`     | Comma-separated media library paths                 |
| `VIDO_DATA_DIR`                                     | `/vido-data` | Database and cache location                         |
| `TMDB_API_KEY`                                      | —            | Metadata lookups                                    |
| `AI_PROVIDER` + `GEMINI_API_KEY` / `CLAUDE_API_KEY` | `gemini`     | Filename parsing, subtitle translation              |
| `OPENAI_API_KEY`                                    | —            | Whisper transcription                               |
| `ASR_BASE_URL` / `ASR_MODEL`                        | —            | Point at a self-hosted OpenAI-compatible ASR engine |
| `AI_RUN_BUDGET_USD`                                 | `5`          | Spend ceiling per run                               |
| `ENABLE_DOUBAN` / `ENABLE_WIKIPEDIA`                | `false`      | Metadata fallback providers (opt-in)                |

AI features degrade gracefully: without keys the rest of the app works and the AI paths report as disabled.

## Docker

```bash
docker compose up -d --build
```

The image is a multi-stage build (pnpm build for the web bundle, `go build ./cmd/api` for the API) producing one container that serves both. `docker-compose.prod.yml` layers on resource limits.

## Troubleshooting

**Node version mismatch** — `nvm use` (reads `.nvmrc`).

**Go dependency issues** — `cd apps/api && go clean -modcache && go mod download`.

**Nx cache issues** — `pnpm nx reset`.

**Port already in use**

```bash
lsof -ti:4200 | xargs kill -9   # web
lsof -ti:8080 | xargs kill -9   # api
```

**Formatting failures in CI** — run `pnpm run format` before committing; `format:check` is a hard gate.

## Scaffold Leftovers

A few paths date from the project's initial scaffold and are not part of the build. Don't edit them expecting an effect:

- `cmd/api/main.go` at the repo root — the real entry point is `apps/api/cmd/api/main.go`
- `api/openapi.json` — no longer generated
- `.air.toml` at the repo root — targets the legacy root entry point; `pnpm nx serve api` is the supported way to run the API

## Contributing

1. Branch off `main` — never commit to `main` directly.
2. Conventional commits with a scope: `feat(subtitle): ...`, `fix(media-detail): ...`.
3. Run `pnpm run lint:all` and the relevant tests before pushing.
4. Open a PR; CI runs lint, unit tests, E2E shards, and a Docker build.

## 授權

授權條款尚未確定，原因見 [README](../README.md#授權)。
