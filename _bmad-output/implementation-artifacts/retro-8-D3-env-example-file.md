# Story retro-8-D3: Update .env.example with All Config Vars

Status: ready-for-dev

## Story

As a developer deploying Vido to my NAS,
I want a comprehensive `.env.example` file documenting every environment variable with descriptions, defaults, and usage context,
so that I can configure Vido without reading source code.

## Acceptance Criteria

1. `.env.example` includes ALL environment variables from `apps/api/internal/config/config.go` and `apps/api/internal/config/database.go`
2. Every variable has: description comment, default value, and which feature it enables
3. Variables added in Epic 3-8 are present: `CLAUDE_API_KEY`, `AI_PROVIDER`, `ENABLE_DOUBAN`, `ENABLE_WIKIPEDIA`, `ENABLE_CIRCUIT_BREAKER`, `FALLBACK_DELAY_MS`, `CIRCUIT_BREAKER_FAILURE_THRESHOLD`, `CIRCUIT_BREAKER_TIMEOUT_SECONDS`
4. Static serving variable `VIDO_PUBLIC_DIR` is documented (from retro-8-D1)
5. Frontend variable `VITE_API_BASE_URL` is documented with usage context
6. Obsolete Docker section is updated — `WEB_PORT` and `API_PORT` replaced with single `VIDO_PORT` (unified container from retro-8-D1)
7. Sections are logically grouped: Server, Paths, Database, API Keys, TMDb, AI Provider, Metadata Fallback, Docker, Frontend, Testing
8. File passes `prettier --check` formatting

## Tasks / Subtasks

- [ ] Task 1: Audit current `.env.example` against `config.go` and `database.go` (AC: 1)
  - [ ] 1.1 Read `apps/api/internal/config/config.go` — list all `os.Getenv` calls
  - [ ] 1.2 Read `apps/api/internal/config/database.go` — list all DB env vars
  - [ ] 1.3 Compare with current `.env.example` — identify missing vars
- [ ] Task 2: Add missing Epic 3-8 variables (AC: 2, 3, 4)
  - [ ] 2.1 Add AI Provider section: `AI_PROVIDER`, `CLAUDE_API_KEY`
  - [ ] 2.2 Add Metadata Fallback section: `ENABLE_DOUBAN`, `ENABLE_WIKIPEDIA`, `ENABLE_CIRCUIT_BREAKER`, `FALLBACK_DELAY_MS`, `CIRCUIT_BREAKER_FAILURE_THRESHOLD`, `CIRCUIT_BREAKER_TIMEOUT_SECONDS`
  - [ ] 2.3 Add `VIDO_PUBLIC_DIR` to Path Configuration section
- [ ] Task 3: Add frontend variable (AC: 5)
  - [ ] 3.1 Add `VITE_API_BASE_URL` with description noting it's used by `apps/web/` via `import.meta.env`
- [ ] Task 4: Update Docker section for unified container (AC: 6)
  - [ ] 4.1 Remove obsolete `WEB_PORT` and `API_PORT` entries
  - [ ] 4.2 Update Docker section to reference unified single-container deployment
  - [ ] 4.3 Add `GIN_MODE` for production Gin framework mode
- [ ] Task 5: Verify completeness and formatting (AC: 7, 8)
  - [ ] 5.1 Ensure logical section grouping
  - [ ] 5.2 Run `prettier --check .env.example`

## Dev Notes

### Existing State

`.env.example` was created in Epic 1 (2026-01-16) and has NOT been updated since. It's missing ~10 variables added in Epics 3-8.

### Source of Truth for Variables

- **Go config struct:** `apps/api/internal/config/config.go` — main app config
- **DB config struct:** `apps/api/internal/config/database.go` — database config
- **Static serving:** `apps/api/cmd/api/static.go` — `VIDO_PUBLIC_DIR`
- **Frontend:** `apps/web/src/services/*.ts` — `VITE_API_BASE_URL`

### Variables Known Missing (from audit)

| Variable | Section | Default | Added In |
|----------|---------|---------|----------|
| `CLAUDE_API_KEY` | API Keys | empty | Epic 3 |
| `AI_PROVIDER` | AI Provider | `gemini` | Epic 3 |
| `ENABLE_DOUBAN` | Metadata Fallback | `false` | Epic 3 |
| `ENABLE_WIKIPEDIA` | Metadata Fallback | `false` | Epic 3 |
| `ENABLE_CIRCUIT_BREAKER` | Metadata Fallback | `true` | Epic 3 |
| `FALLBACK_DELAY_MS` | Metadata Fallback | `100` | Epic 3 |
| `CIRCUIT_BREAKER_FAILURE_THRESHOLD` | Metadata Fallback | `5` | Epic 3 |
| `CIRCUIT_BREAKER_TIMEOUT_SECONDS` | Metadata Fallback | `30` | Epic 3 |
| `VIDO_PUBLIC_DIR` | Paths | `/app/public` | Retro-8-D1 |
| `VITE_API_BASE_URL` | Frontend | (none) | Epic 2 |
| `GIN_MODE` | Docker | `release` | Retro-8-D1 |

### Docker Section Update

The project consolidated from 2-container (nginx + API) to unified single-container in retro-8-D1. The `.env.example` still references `WEB_PORT` and `API_PORT` from the old architecture. Update to reflect unified `VIDO_PORT` only.

### What NOT to Change

- Do NOT modify `.env` (user's actual config, gitignored)
- Do NOT add CI-only variables (`CI`, `TEST_ENV`, `NODE_ENV`) — these are not user-configurable
- Do NOT add build-time variables (`GO_VERSION`) — these are workflow-internal

### References

- [Source: apps/api/internal/config/config.go] — Config struct definition
- [Source: apps/api/internal/config/database.go] — Database config struct
- [Source: apps/api/cmd/api/static.go] — VIDO_PUBLIC_DIR usage
- [Source: apps/web/src/services/] — VITE_API_BASE_URL usage
- [Source: docker-compose.yml] — Docker env var references
- [Source: Dockerfile] — Container ENV directives
- [Source: epic-8-retro-2026-03-25.md#D3] — Retro action item origin

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List

## Change Log
