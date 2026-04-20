# Story: Rename `QB_` Error-Code Prefix to `QBITTORRENT_`

Status: backlog

**Origin:** Winston (Architect) architectural review of retro-10-AI3 Rule 7 expansion — 2026-04-20.
**Priority:** LOW (cosmetic consistency; wire contract still functions as-is).
**Scope estimate:** 4 constants + ~16 call-site files (tests, handlers, services, Swagger annotations, E2E specs, frontend e2e).

## Problem

Across Rule 7's 13 registered prefixes, the convention is **`SOURCE = uppercase(package name)`**:

| Package          | Prefix       |
|------------------|--------------|
| `tmdb`           | `TMDB_`      |
| `douban`         | `DOUBAN_`    |
| `wikipedia`      | `WIKIPEDIA_` |
| `metadata`       | `METADATA_`  |
| `library`        | `LIBRARY_`   |
| `scanner`        | `SCANNER_`   |
| `sse`            | `SSE_`       |
| `subtitle`       | `SUBTITLE_`  |
| `plugins`        | `PLUGIN_`    |
| `ai`             | `AI_`        |
| **`qbittorrent`** | **`QB_`** ← outlier |

`QB_` is the only prefix that breaks this rule. The shortening is parochial to qBittorrent community jargon and inconsistent with how future downloader integrations would be named (`TRANSMISSION_`, `DELUGE_`, `NZBGET_` — each would follow the package-name-uppercased convention per the existing TMDB_/DOUBAN_/... precedent).

## Acceptance Criteria

1. Given `apps/api/internal/qbittorrent/types.go:42-47`, when the file is read after the change, then the 4 constants are:
   ```go
   ErrCodeConnectionFailed = "QBITTORRENT_CONNECTION_FAILED"
   ErrCodeAuthFailed       = "QBITTORRENT_AUTH_FAILED"
   ErrCodeTimeout          = "QBITTORRENT_TIMEOUT"
   ErrCodeNotConfigured    = "QBITTORRENT_NOT_CONFIGURED"
   ```
   plus any additional `QB_*` codes discovered in `torrent.go` (grep audit in Task 1).
2. Given `rg -n '"QB_'` is run against the repo after the change, when it completes, then **zero hits remain** outside this story file itself.
3. Given `go test ./...` + `pnpm nx test api` + `pnpm nx test web` run after the change, when they complete, then all tests pass (test assertions that match on the string literal have been updated in the same commit).
4. Given Swagger annotations on qBittorrent handlers (e.g., `apps/api/internal/handlers/qbittorrent_handler.go` `@Failure` lines), when read, then any response-body examples referencing `"QB_*"` are updated to `"QBITTORRENT_*"`.
5. Given E2E specs under `tests/e2e/*qbittorrent*.spec.ts` and `tests/e2e/downloads*.spec.ts`, when read, then any hard-coded `"QB_"` response-body matchers are updated.
6. Given `project-context.md` Rule 7 after the change, when read, then the prefix list entry **`QB_,`** is replaced by **`QBITTORRENT_,`** in line 300's authoritative set. The "Last Updated" header is bumped with a note citing this story.
7. Given `_bmad/bmm/workflows/4-implementation/code-review/instructions.xml` after the change, when read, then (a) line 112 `QB_,` → `QBITTORRENT_,` in the inline prefix list and (b) line 146 auto-fix map entry `qbittorrent/** → QB_` → `qbittorrent/** → QBITTORRENT_`. Sync date (line 97, line 111) bumped.
8. Given `apps/web/src/` and `libs/shared-types/` error-handling code, when audited via `rg -n '"QB_'`, then any frontend references are updated (likely none per retro-10-AI3's L3 call-site enumeration, but defensive check required).

## Task Sketch (for SM /create-story to flesh out)

- **Task 1** — Audit all `QB_*` call sites with `rg -n '"QB_[A-Z]'` before starting. Expected touch-list (from retro-10-AI3 discovery): 16 files across `apps/api/internal/qbittorrent/{torrent.go,types.go,types_test.go,torrent_test.go}`, `apps/api/internal/handlers/{qbittorrent,download}_handler{,_test}.go`, `apps/api/internal/services/{qbittorrent,download}_service{,_test}.go`, `tests/e2e/*.spec.ts`.
- **Task 2** — Rename constants + strings in a single commit (atomic wire-contract change). Verify via `rg '"QB_'` after = 0 hits.
- **Task 3** — Update Rule 7 (project-context.md) + CR workflow (instructions.xml) sync points in same commit.
- **Task 4** — Run `pnpm lint:all` + `pnpm nx test api` + `pnpm nx test web` + E2E smoke (if deploying). Confirm zero regressions.

## Out of Scope

- Adding other downloader integrations (Transmission, Deluge, NZBGet) — that's a separate epic and would naturally use their own package-name-uppercased prefixes.
- Changing any qBittorrent API behavior or error categories — this is a **pure rename**; wire-contract surface is byte-for-byte updated but structurally identical.

## References

- Winston (Architect) verdict — retro-10-AI3 Winston-prompt file, Item 3 ruling (2026-04-20)
- Rule 7 source: `project-context.md#rule-7-error-codes-system`
- Current outlier files: `apps/api/internal/qbittorrent/{types.go:42-47, torrent.go}`
- 16-file call-site enumeration: from retro-10-AI3 auto-fix Task 1 grep output
