# Bugfix: `AutoGenerator` exclusion set never expires — bind it to the file's mtime and a free-lane epoch

Status: done

**Origin:** Rule 24 ③ from 9R-10b 補審 **M2** (2026-08-21). `excludedMediaIDs` parks an item for good once it is deferred (paid) or has failed `autoFailureAttemptLimit` times. Correct on the main line (a file that needs ASR does not become free by itself), wrong in two cases: **(a)** the user **replaces the file** (new remux now carries an embedded Chinese track → free, but never picked); **(b)** an **upgrade** makes the free lane support a format it previously failed on. Third sibling of #263 / #264; backend only.

---

## Story

As the Vido NAS operator,
I want an item that was parked out of the free lane to become a candidate again when its media file has changed or the free lane itself has improved,
so that replacing a file — or upgrading Vido — is enough for auto-generation to try it again, without a manual run.

## Root cause (verified)

- `excludedMediaIDs` (`auto_generation.go:540-600`) returns a set of ids; nothing about the *file* or the *code version* enters the decision. `SubtitleRun` has no file fingerprint column (`models/subtitle_run.go:103-120`); `MetadataHash` hashes show metadata, not the file.
- What IS available without a schema change: `SubtitleRun.StartedAt` (every row) and the media file's mtime (`Movie.FilePath` / `Episode.FilePath` are in the enumeration's full-column selects — `movie_repository.go:908`, `episode_repository.go:161`).

## Design

| # | Decision | Chosen | Why |
|---|----------|--------|-----|
| D1 | Case (a) judgment | **A parked verdict holds only for runs that STARTED AFTER the file's current change time = max(mtime, ctime)** (review H1: `rsync -a` / `cp -p` / *arr imports preserve mtime; ctime is kernel-set on every replace). Deferred: the latest skipped run's `StartedAt` must be after mtime. Failed: only rows with `StartedAt` after mtime count toward the limit. | mtime is the cheapest "is this still the same file" signal; a replacement always moves it forward. No migration. |
| D2 | Case (b) judgment | **`FreeLaneEpoch` package constant** (a `time.Time`): rows that started before it are ignored entirely. Bumped by hand in the story that widens the free lane. | The "upgrade made it free" fact lives in code, not data; a date constant is the honest, grep-able record of it. |
| D3 | When to stat | **Only for items that are both parked and otherwise eligible** — the set is built with `StartedAt`s, the stat happens in `collect` at the moment an enumerated item would be skipped. | Bounds the stat count to the parked∩enumerated intersection; never one per run row, never for items that would be skipped anyway. |
| D4 | Stat fails (file gone, share asleep) | **Stay parked** (fail closed) | A missing file would fail anyway and write another row; re-probing it every scan is the H1 starvation again. Logged at Debug. |
| D5 | Port shape | `modTime func(path string) (time.Time, error)` field, default `fileChangedAt` (os.Stat + platform ctime via build-tagged `inodeChangeTime`); `WithAutoFileModTime(fn)` for tests | Rule 11: narrow; no new interface type needed. |
| D6 | Stat time bound (review M2) | `boundedModTime`: the lookup runs in a goroutine, raced against the round ctx and `autoStatTimeout = 10s`; either expiring = stat error → stays parked | A D-state mount must not hang `collect` outside the per-item deadline and hold `Stop()` past the grace period. The goroutine may leak on a truly hung stat; the round does not. |

## Acceptance Criteria

1. **AC #1 — Replaced file un-parks a deferred item.** Given an item whose latest skipped run is `deferred-paid` with `StartedAt = T`, when the file's mtime is after `T`, then the item is enumerated; when mtime is before `T`, it is still excluded.
2. **AC #2 — Replaced file resets the failure count.** Given an item with `autoFailureAttemptLimit` failed rows, when the file's mtime is after all of them, then it is enumerated; when the mtime is after only some of them so that fewer than the limit remain, it is enumerated; when the limit remains after the mtime, it stays excluded.
3. **AC #3 — Epoch.** Rows with `StartedAt` before `FreeLaneEpoch` never count (deferred or failed); rows after it do.
4. **AC #4 — Stat failure keeps the park.** When the mod-time lookup errors, the item stays excluded and a Debug line is logged.
5. **AC #5 — Stat is bounded.** The mod-time port is called only for items that are parked AND enumerated AND eligible — never for un-parked items, never per run row.
6. **AC #6 — Cancelled rows still exempt; #263/#264 behaviour untouched.** All existing `TestAutoGenerator_*` stay green (with fixtures now carrying realistic `StartedAt`s), `cost_consent_test.go` diff 0, `-race` green.

## Tasks / Subtasks

- [x] **Task 1 — Record, not set** (AC #1–#3) — `auto_generation.go`
  - [x] 1.1 `FreeLaneEpoch` constant (`time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC)`) with a doc comment: *bump when a story widens the free lane; rows before it are stale verdicts.*
  - [x] 1.2 Replace `map[string]struct{}` with `map[string]parkedRecord{deferredAt time.Time; failedAt []time.Time}`; build it from rows with `StartedAt.After(FreeLaneEpoch)` only; keep the latest-skipped-decides and cancelled-exempt rules. Keep only records that are parked against a zero mtime (deferredAt set, or `len(failedAt) >= limit`) so the map stays small.
  - [x] 1.3 `parkedRecord.stillParked(mtime time.Time) bool`: `deferredAt.After(mtime) || countAfter(failedAt, mtime) >= autoFailureAttemptLimit`.
- [x] **Task 2 — Stat at the skip point** (AC #4, #5) — `auto_generation.go`
  - [x] 2.1 `modTime func(string) (time.Time, error)` field, default `osModTime` (`os.Stat(...).ModTime()`); `WithAutoFileModTime(fn)`.
  - [x] 2.2 In `collect`, for movies and episodes: replace `if _, skip := excluded[id]; skip { continue }` with `if rec, parked := excluded[id]; parked && g.stillParked(rec, filePath) { continue }` — and keep this check **after** the status/library filters so only eligible items are stat-ed (AC #5). `stillParked` stats; on error → Debug log, return true.
- [x] **Task 3 — Tests** — `auto_generation_test.go`
  - [x] 3.1 Builders gain `StartedAt: autoNow` (a fixed `time.Time` after the epoch); harness gets a `modTimes map[string]time.Time` + `statErr` fake wired via `WithAutoFileModTime`, default mtime = `autoNow.Add(-time.Hour)` (older than every row ⇒ parked, so existing tests keep their meaning). Track stat calls.
  - [x] 3.2 AC #1 (two cases), AC #2 (three cases), AC #3 (deferred + failed before epoch), AC #4 (stat error), AC #5 (stat count: parked+eligible only; un-parked and ineligible items never stat-ed).
  - [x] 3.3 `-race -count=5`.
- [x] **Task 4 — Gates + record.**

### Cross-stack split check — Backend 4 / Frontend 0 ⇒ no split.

## Dev Notes

- `ListByStatus` returns rows newest-first; the latest-skipped rule depends on it (補審 M3). Do not sort.
- Movies: `m.FilePath` is `models.NullString`; episodes likewise. An empty path cannot be stat-ed ⇒ treat as stat error (stay parked).
- Do **not** add a schema column; D1/D2 were chosen precisely to avoid migration 032 for a judgment `StartedAt` + mtime already give.
- #263's `CancelledRunPrefix` rows stay out of `failedAt` entirely.
- Stat cost note for the PR: ≤ one `os.Stat` per parked-and-eligible item per round, on the enumerated order, stopping at `maxPerRun` like everything else in `collect`.

### Time-dependent visual coverage — N/A (backend only).

### References

- [Source: `apps/api/internal/subtitle/auto_generation.go:540-600` — current `excludedMediaIDs`; `collect` skip points]
- [Source: `apps/api/internal/models/subtitle_run.go:103-120` — no fingerprint column; `StartedAt`]
- [Source: `_bmad-output/implementation-artifacts/bugfix-autogenerator-no-timeout-or-shutdown.md` — Known limits "head-of-line re-selection" (this story does not address it; cancelled rows remain exempt)]

## Dev Agent Record

### Agent Model Used

Claude Fable 5 (claude-fable-5) — 2026-08-23.

### Debug Log References

- RED: package failed to compile on `FreeLaneEpoch` / `WithAutoFileModTime`; after GREEN, `TestDeferredMarker_WriterAndReaderAgree` went red because its hand-built row had a zero `StartedAt` (before the epoch) — fixture now copies the writer's `StartedAt` and asserts the harness clock is past the epoch.

### Completion Notes List

- `parkedRecord{deferredAt, failedAt}` replaces the id set; `stillParked(changedAt)` = deferral after changedAt OR ≥ limit failures after changedAt. Rows before `freeLaneEpoch` (2026-08-23, unexported var + `FreeLaneEpoch()` getter) are ignored. Prefilter keeps only records parked against a zero time so the per-item stat is bounded to parked ∩ eligible (movies after status+library filter; episodes after the series→library filter — test-pinned for both).
- Production lookup `fileChangedAt` = max(mtime, ctime); ctime via `inodeChangeTime` in `auto_generation_ctime_{linux,darwin,other}.go` (`GOOS=linux go vet` clean). `boundedModTime` races the stat against ctx + 10 s.
- Tests (12 new): AC #1 ×2, AC #2 table ×3, AC #3 ×3, AC #4 ×2 (stat error, empty path), AC #5 (movies + episodes), AC #6 cancelled-exempt, `TestFileChangedAt_PrefersInodeChangeTimeOverAPreservedMtime` (real temp file, `os.Chtimes` back 48 h), hung-stat-under-ctx. `-race -count=20` green.
- Gates: `pnpm nx test api` zero FAIL · `pnpm nx test web` green · lint api · format:check · 0 orphaned workers · `cost_consent_test.go` diff 0.
- Rollout note: the epoch is today, so every parked row on the NAS is stale on deploy — the first rounds re-probe up to `maxPerRun` previously-parked items per scan until fresh verdicts land. Intended.
- 🔗 AC Drift: NONE (9R-10b CR H1 "exclude deferred items" and 補審 M1 "park after 3 failures" still hold; they are now scoped to the current file/epoch). 📎 Contract Stamps: NONE. 🎭 A11y: N/A (backend). 🎨 UX: SKIPPED.
- Code review (fresh-context agent, same model): 1 H / 1 M / 4 L — all addressed (H1 mtime-preserved replacement → ctime; M2 stat bound; L epoch exported var → getter; L un-park log Info→Debug; L episode AC #5 coverage; L `nilIfEmpty` removed). Not changed: unbounded `ListByStatus(…, 0)` (pre-existing; a `WHERE started_at > ?` push-down is a repo change — noted, not filed: cost is two reads per round on NAS-scale tables, same as before).

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - `N/A — no out-of-scope work discovered`.

### File List

- `apps/api/internal/subtitle/auto_generation.go`
- `apps/api/internal/subtitle/auto_generation_ctime_linux.go` (new)
- `apps/api/internal/subtitle/auto_generation_ctime_darwin.go` (new)
- `apps/api/internal/subtitle/auto_generation_ctime_other.go` (new)
- `apps/api/internal/subtitle/auto_generation_test.go`
- `apps/api/internal/subtitle/cost_consent_free_lane_test.go` (fixture: `StartedAt` copied from the writer)
- `_bmad-output/implementation-artifacts/bugfix-auto-exclusion-never-expires.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
|------|--------|
| 2026-08-23 | dev + code-review (1H/1M/4L addressed) + gates green. D1 widened to max(mtime, ctime); D6 added. Status → done. |
| 2026-08-23 | Story created (create-story, yolo). |
