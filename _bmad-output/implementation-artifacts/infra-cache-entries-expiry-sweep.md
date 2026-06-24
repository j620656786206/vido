# Story (Infra): `cache_entries` Scheduled Expiry Sweep

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->
<!-- Story key: infra-cache-entries-expiry-sweep (non-epic infra story; whole-app benefit).
     Spawned by tech-spec-ux3-discover-facet-aggregation.md AR-F1 (2026-06-24). -->

## Story

As the **Vido backend (single-NAS, long-running process)**,
I want a **scheduled service that periodically deletes expired rows from the `cache_entries` SQLite table**,
so that **the TMDb response cache does not grow unbounded — especially once the Discover facet-counts fan-out (a write-amplifier) starts populating it — keeping query performance and on-disk row count healthy without manual intervention.**

## Acceptance Criteria

1. **AC1 (sweep runs on a schedule):** A new `CacheSweepScheduler` service, started at app boot, calls `CacheRepository.ClearExpired(ctx)` on a recurring ticker (default interval ~45 min, aligned with the 1h cache TTL). Each successful sweep that deletes ≥1 row logs at INFO with the deleted-row count (the existing `ClearExpired` already logs `"Expired cache entries cleared"`); a sweep that deletes 0 rows is a no-op (no error).
2. **AC2 (settings-configurable interval):** The interval is read from settings key `cache_sweep_interval_minutes` (int) at `Start`. When the setting is unset/unreadable, it defaults to `45`. A value `<= 0` **disables** the sweep (scheduler logs that it is disabled and starts no ticker). A positive value below a safety floor (`5`) is clamped up to `5` (prevents a misconfigured 1-min hammer on the writer lock).
3. **AC3 (clean lifecycle, no goroutine leak):** `Start(ctx)` honors **both** `ctx.Done()` and an explicit `Stop()` signal and returns from its loop on either; `defer ticker.Stop()` releases the ticker. `Stop()` is idempotent (safe to call when never started / called twice) — mirrors `BackupScheduler.Stop()` (`sync.Mutex` + `stopped` guard + `close(stopCh)`).
4. **AC4 (errors swallowed, goroutine never dies):** A `ClearExpired` error is logged at ERROR and the loop **continues** to the next tick (never `return`s on a sweep error, never panics the goroutine). A `defer recover()` in the swept goroutine guards against any unexpected panic so one bad tick can never kill the scheduler.
5. **AC5 (wired into main + graceful shutdown):** The scheduler is constructed in `apps/api/cmd/api/main.go` (alongside the backup/scan schedulers), started in its own goroutine with a dedicated cancellable context, and on shutdown its `cancel()` is invoked **and** `Stop()` is called — matching the exact backup/scan scheduler wiring pattern (construct → `go Start(ctx)` → `cancel()` + `Stop()`).
6. **AC6 (NEVER VACUUM):** The sweep performs **only** the `DELETE` via `ClearExpired`. It MUST NOT run `VACUUM` (or any whole-DB rewrite) on the ticker. (Disk reclamation is explicitly out of scope — see Dev Notes "Side effect 2".)
7. **AC7 (scoped to `cache_entries` only):** The sweep targets the `cache_entries` table exclusively via `CacheRepository.ClearExpired`. It does NOT touch the separate `ai_cache` / offline / image cache stores (those are a separately-tracked backlog item — see Discovery Triage).
8. **AC8 (tests):** Unit tests cover: interval resolution (default / configured / disabled / clamp-floor), `Stop()` idempotency, error-swallow-continues-loop behavior, and that a tick invokes `ClearExpired` exactly once — using a mocked `CacheRepository` + `SettingsRepository` and a short/injected interval, with NO real `time.Sleep`-based flakiness. `go vet` + `staticcheck` clean.

## Tasks / Subtasks

- [ ] **Task 1 — Create `CacheSweepScheduler` service (AC: #1, #2, #3, #4, #6, #7)**
  - [ ] New file `apps/api/internal/services/cache_sweep_scheduler.go`.
  - [ ] Define `CacheSweepSchedulerInterface { Start(ctx context.Context); Stop() }` (mirror `BackupSchedulerInterface` shape, minus the schedule getters/setters which this simpler sweep does not need).
  - [ ] Struct `CacheSweepScheduler{ cacheRepo repository.CacheRepositoryInterface; settingsRepo repository.SettingsRepositoryInterface; mu sync.Mutex; stopCh chan struct{}; stopped bool }`.
  - [ ] `NewCacheSweepScheduler(cacheRepo repository.CacheRepositoryInterface, settingsRepo repository.SettingsRepositoryInterface) *CacheSweepScheduler` — initialize `stopCh: make(chan struct{})`.
  - [ ] Const `settingsKeyCacheSweepInterval = "cache_sweep_interval_minutes"`; const `defaultCacheSweepIntervalMinutes = 45`; const `minCacheSweepIntervalMinutes = 5`.
  - [ ] Helper `resolveInterval(ctx) (time.Duration, bool)` — `settingsRepo.GetInt(ctx, settingsKeyCacheSweepInterval)`; on error use default 45; `<= 0` ⇒ return `(0, false)` (disabled); `> 0 && < 5` ⇒ clamp to 5; return `(time.Duration(mins) * time.Minute, true)`.
  - [ ] `Start(ctx)`: resolve interval; if disabled → `slog.Info("Cache sweep scheduler disabled (interval <= 0)")` and `return`; else `slog.Info("Cache sweep scheduler started", "interval", interval)`, `ticker := time.NewTicker(interval)`, `defer ticker.Stop()`, then `for { select { case <-ctx.Done(): log+return; case <-s.stopCh: log+return; case <-ticker.C: s.sweep(ctx) } }`.
  - [ ] `sweep(ctx)`: `defer func(){ if r := recover(); r != nil { slog.Error("Cache sweep panicked", "recover", r) } }()`; `if _, err := s.cacheRepo.ClearExpired(ctx); err != nil { slog.Error("Cache sweep failed", "error", err) }` — note **no `return` on success or error path that breaks the loop**; the loop owns control flow.
  - [ ] `Stop()`: copy `BackupScheduler.Stop()` verbatim in shape — `s.mu.Lock(); defer s.mu.Unlock(); if !s.stopped { s.stopped = true; close(s.stopCh) }`.
  - [ ] Do **NOT** add any `VACUUM` call anywhere (AC #6).
- [ ] **Task 2 — Wire into `cmd/api/main.go` (AC: #5)**
  - [ ] Construct after the scan scheduler (~`main.go:405`): `cacheSweepScheduler := services.NewCacheSweepScheduler(repos.Cache, repos.Settings)` + `slog.Info("Cache sweep scheduler initialized")`. (Confirm the repos field name for the cache repository — see Dev Notes "main.go wiring", verify `repos.Cache` exists; if the field is named differently, use the actual field.)
  - [ ] Start in goroutine alongside the others (~`main.go:633`): `cacheSweepCtx, cacheSweepCancel := context.WithCancel(context.Background())` + `go cacheSweepScheduler.Start(cacheSweepCtx)` + `slog.Info("Cache sweep scheduler started")`.
  - [ ] Shutdown (~`main.go:667-673`, beside `scanSchedulerCancel()` / `backupScheduler.Stop()`): `cacheSweepCancel()` + `cacheSweepScheduler.Stop()` with a `slog.Info("Stopping cache sweep scheduler...")`.
- [ ] **Task 3 — Unit tests (AC: #8)**
  - [ ] New file `apps/api/internal/services/cache_sweep_scheduler_test.go`.
  - [ ] Add a `MockCacheRepo` implementing `repository.CacheRepositoryInterface` (or reuse an existing testutil mock if one exists — grep `MockCacheRepo` / `CacheRepository` mocks first) with a call-capturing `ClearExpired`; reuse the `MockSchedulerSettingsRepo` pattern from `backup_scheduler_test.go` for `SettingsRepositoryInterface`.
  - [ ] Test `resolveInterval`: unset→45m; `GetInt` returns `120`→120m; returns `0`/`-1`→disabled; returns `2`→clamped 5m.
  - [ ] Test `Stop()` idempotent: `NewCacheSweepScheduler(nil, nil)` then `Stop(); Stop()` does not panic (matches `TestBackupScheduler_Stop`).
  - [ ] Test the sweep tick path WITHOUT real-time flakiness: prefer extracting `sweep(ctx)` and asserting it calls `ClearExpired` once and that a `ClearExpired` error does not propagate/panic. Optionally drive `Start` with a very short configured interval + a context cancelled after the first `ClearExpired` capture (channel signal), then assert `ClearExpired` was called ≥1 and the goroutine exits cleanly on cancel.
  - [ ] Test error-swallow: `ClearExpired` returns an error → `sweep` returns normally (no panic), scheduler still responsive.
  - [ ] Run `go test ./internal/services/ -run CacheSweep -v`, then `pnpm lint:all` (go vet + staticcheck + eslint + prettier) per project-context Rule 12.

## Dev Notes

### Origin & why this is urgent now

- Spawned by **AR-F1 (Critical)** of `tech-spec-ux3-discover-facet-aggregation.md` (2026-06-24). The Discover **facet-counts** fan-out writes one cached `/discover` result **per facet value per probe** into `cache_entries` — a **write-amplifier**. `cache_entries` already had a latent gap: `CacheRepository.ClearExpired` exists but **has no production caller**, so expired rows accumulate forever. The amplifier makes the latent gap urgent → this story is a **hard prerequisite that MUST land before facet-counts ships** (`ux3-discover-facet-aggregation-fe`).
- Sprint-status dependency chain: `infra-cache-entries-expiry-sweep` (this, prereq) → `ux3-discover-facet-aggregation-fe` (consumer). Independent / parallelizable with the BE facet endpoint work, but gates the FE consumer.

### Safe by construction (the core correctness argument — AR-F1)

- `ClearExpired` deletes `WHERE expires_at <= datetime('now')` (`internal/repository/cache_repository.go:123-141`).
- The cache read filter (`CacheRepository.Get`) is `WHERE key = ? AND expires_at > datetime('now')` (`cache_repository.go:30-34`).
- **The sweep predicate is the exact complement of the read filter** — it deletes only rows that reads already treat as misses. It can **never** delete a live-served hit, and there is no timezone divergence (both use `datetime('now')`). Index-backed by `idx_cache_entries_expires_at` (migration `004_create_cache_entries_table.go:47-54`). No new SQL, no new migration — this story only adds the *scheduler that calls the existing method*.

### Side effects to be aware of (both ACCEPTED, do not try to "fix")

- **Side effect 1 — write-lock contention (mainly the FIRST sweep on a bloated table).** SQLite WAL: readers don't block, but a large `DELETE` holds the single writer lock → a concurrent cache `Set` may hit `busy_timeout` → transient cache-write miss (logged warning, **non-fatal**, re-fetched next time). Mitigation is inherent: keep the interval modest; after the first run the table stays small, so steady-state sweeps are tiny.
- **Side effect 2 — disk is NOT reclaimed (AC #6 rationale).** `auto_vacuum` is unset (default `NONE`) → `DELETE` frees pages for **reuse** but does **not** shrink the `.db` file. This sweep keeps **row count** (query perf) healthy, not **file size**. That is the correct trade-off for an ongoing cache (the file stabilizes at the working-set high-water mark). **Never** put `VACUUM` on the ticker — it is a whole-DB rewrite holding an exclusive lock and needing ~DB-size temp space. If disk reclamation is ever genuinely needed, it is a **rare manual/admin action**, not this scheduler's job.

### Implementation guardrails (verbatim from AR-F1 design notes)

1. Mirror `internal/services/backup_scheduler.go`: `Start(ctx)` + `time.NewTicker` + `defer ticker.Stop()` + `select { case <-ctx.Done(): return; case <-s.stopCh: return; case <-ticker.C: ... }` + `Stop()`. No goroutine leak.
2. **Never** `VACUUM` on the ticker (AC #6).
3. Swallow + log `ClearExpired` errors; never panic the goroutine (AC #4).
4. Interval ~30–60 min (aligned with the 1h TTL); settings-configurable, mirroring the backup/scan schedulers (AC #2).
5. (Latent, pre-existing, NOT caused by this sweep) `modernc.org/sqlite` serializes `time.Now()` (local) vs `datetime('now')` (UTC) — works today (cache hits happen, so the comparison is internally consistent); the sweep stays consistent with reads regardless because it reuses the same `datetime('now')` comparison. Do not "fix" this as part of the story.

### Canonical pattern to mirror — `BackupScheduler` (exact anchors)

- **Struct / constructor / Start / Stop:** `apps/api/internal/services/backup_scheduler.go` — struct `:49-58`, `NewBackupScheduler` `:64-75`, `Start(ctx)` `:78-95` (note its `for { select { <-ctx.Done() | <-s.stopCh | <-ticker.C } }` shape and `defer ticker.Stop()`), `Stop()` `:` (the `sync.Mutex` + `stopped` + `close(stopCh)` idempotent form shown above).
- **Error handling convention:** `backup_scheduler.go:184-214` — every error is `slog.Error(...)` then the cycle continues; errors never kill the loop. Copy this discipline.
- **Simpler interval-only precedent (also valid to borrow from):** `apps/api/internal/services/scan_scheduler.go` — loads its interval from settings at `Start` (`loadScheduleFromSettings`), builds a variable-duration ticker. This story is closer to scan's "one interval, fixed ticker" shape than to backup's "1-min tick + run-at-hour" shape, so feel free to lean on scan_scheduler for the interval-loading idiom while keeping backup's clean `stopCh`/`Stop()` lifecycle.

### `main.go` wiring (exact anchors)

- Backup scheduler: construct `apps/api/cmd/api/main.go:177` (`services.NewBackupScheduler(backupService, repos.Settings, repos.Backups)`); start `:628` (`go backupScheduler.Start(schedulerCtx)`); shutdown `:673` (`backupScheduler.Stop()`).
- Scan scheduler: construct `:405`; start `:632-633` (`scanSchedulerCtx, scanSchedulerCancel := context.WithCancel(context.Background()); go scanScheduler.Start(scanSchedulerCtx)`); shutdown `:667-668` (`scanSchedulerCancel(); scanScheduler.Stop()`).
- Place the new cache-sweep wiring **adjacent to these three** (construct near `:405`, start near `:633`, shutdown near `:668`) so the lifecycle reads consistently. **Verify the cache repository field name on `repos`** before writing `repos.Cache` — grep the `repos` struct (the repositories container constructed earlier in `main.go`) for the `Cache` field; use the actual name.

### Settings interface (confirmed — do not invent signatures)

`repository.SettingsRepositoryInterface` (`internal/repository/interfaces.go`) provides exactly: `Set`, `Get`, `GetAll`, `Delete`, `GetString`, `GetInt(ctx, key) (int, error)`, `GetBool`, `SetString`, `SetInt(ctx, key, value) int)`, `SetBool`. Use `GetInt` to read `cache_sweep_interval_minutes`; treat its error return as "unset → default 45". No settings-writer is required for this story (the interval is read-only here; a future settings UI / endpoint to set it is out of scope).

### Layering note (Rule 4)

Existing schedulers (`BackupScheduler`, `ScanScheduler`) inject **repositories directly** (`repository.SettingsRepositoryInterface`, etc.), not service wrappers — a scheduler is an infrastructure driver, not an HTTP handler, so the Handler→Service→Repository rule's "Handler must not skip Service" prohibition does not apply. Follow the established precedent: inject `repository.CacheRepositoryInterface` + `repository.SettingsRepositoryInterface` directly. Do NOT introduce a new service-layer wrapper just to call `ClearExpired`.

### Project Structure Notes

- New service file lives beside its siblings: `apps/api/internal/services/cache_sweep_scheduler.go` (+ co-located `_test.go` per Rule 9). Backend-only (Rule 1 — `/apps/api`, never `/cmd` or root `/internal`).
- No new error codes (Rule 7) — the sweep has no user-facing error surface; failures are internal logged warnings.
- No migration, no schema change, no new repo method — reuses `CacheRepository.ClearExpired` as-is.
- **Cross-stack split check:** 3 backend tasks, 0 frontend tasks → single story, no split required.

### Time-dependent visual coverage

- **N/A — no wall-clock-reading `apps/web/src/components/**/*.{ts,tsx}` touched.** This is a backend-only infra story (Go scheduler); it renders no UI and captures no visual baselines. (Rule 23 applies only to frontend component visual fixtures.)

### References

- [Source: `_bmad-output/implementation-artifacts/tech-spec-ux3-discover-facet-aggregation.md#Architecture Review (applied — AR-F#)`] — AR-F1 finding + the "expiry-sweep design notes" block (safe-by-construction, side effects, the 5 implementation guardrails).
- [Source: `_bmad-output/implementation-artifacts/tech-spec-ux3-discover-facet-aggregation.md#Tasks` → "Task P (PREREQUISITE — separate infra story, AR-F1)"] — the originating task definition.
- [Source: `apps/api/internal/repository/cache_repository.go:122-141`] — `ClearExpired` (the method this story schedules).
- [Source: `apps/api/internal/repository/cache_repository.go:25-54`] — `Get` read filter (`expires_at > datetime('now')`), the complement that proves safety.
- [Source: `apps/api/internal/database/migrations/004_create_cache_entries_table.go`] — `cache_entries` schema + `idx_cache_entries_expires_at`.
- [Source: `apps/api/internal/services/backup_scheduler.go`] — canonical scheduler lifecycle (`Start`/`Stop`/ticker/error-swallow) to mirror.
- [Source: `apps/api/internal/services/scan_scheduler.go`] — interval-from-settings loading idiom.
- [Source: `apps/api/cmd/api/main.go:177,405,628,632-633,667-668,673`] — scheduler construct/start/shutdown wiring sites.
- [Source: `project-context.md` §5 Background Tasks (Worker Pool / goroutines + channels), Rule 1, Rule 4, Rule 9, Rule 12, Rule 13 (error completeness), Rule 14 (Resource Lifecycle — graceful shutdown via `context.Context`)] — governing rules.
- [Source: `project-context.md` Rule 27 (External Integration Standard, Pillar ② cache)] — TTL/cache context for why expired-row sweep belongs to the TMDb cache lifecycle.

## Dev Agent Record

### Agent Model Used

_(to be filled by dev agent)_

### Debug Log References

### Completion Notes List

### Discovery Triage

<!-- Rule 24 (project-context.md). Out-of-scope findings surfaced during story prep MUST be tracked. -->

- **Did this story discover any work outside its current scope?** **YES — two findings (pre-triaged at story-creation time by SM; fact-verified in a Party-Mode review 2026-06-24):**
  - **③ — AI cache + offline cache ALSO lack a scheduled expiry sweep; the sprint-note's claim is doubly wrong.**
    - **Finding (verified via grep of `ClearExpired` callers across `apps/api`):** there is **no scheduled prod caller for ANY repo-method cache store** — not `cache_entries` (`CacheRepository.ClearExpired`, `cache_repository.go:123`), not the AI cache (`ai.Cache.ClearExpired` / `AIService.ClearExpiredCache`, `internal/ai/cache.go:189`, `internal/services/ai_service.go:354`), nor `internal/cache/offline_cache.go:175` (`OfflineCache.ClearExpired`, instantiated at `main.go:124` via `DegradationService`). All three methods exist; **none is invoked periodically.**
    - **The sprint-note "only the AI cache is swept" is DOUBLY FALSE:** (a) the AI cache is **not** swept; (b) the cache that **does** self-sweep is **Douban** — `internal/douban/cache.go:77` `cleanupLoop` + `time.NewTicker(CleanupInterval=1h)` + `DeleteExpired` (`:362`), self-started inside `NewCache` when enabled. **True picture: 3 ORPHANED (cache_entries / ai_cache / offline_cache) + 1 SELF-SWEEP (douban_cache).**
    - **Lane ③ — backlog-with-carry-forward-link.** Real but out-of-scope for THIS story (intentionally scoped to `cache_entries` — the only store the Discover facet fan-out write-amplifies, AC #7) and **non-blocking**. Risk-ranked (TEA): `ai_cache` mid (TTL 30d, written per-scan), `offline_cache` low (written only on degradation) → non-urgent. Filed as sprint-status entry **`infra-ai-offline-cache-expiry-sweep`** (backlog), naming **Douban's cache-owns-its-lifecycle pattern** as the precedent to mirror (vs the orphaned repo-method pattern). Bidirectional link.
    - **Why NOT lane ① (absorb here):** sweeping `ai_cache` + offline cache needs **different repos injected** and those tables are not write-amplified by the fan-out, so absorbing them would broaden the blast radius of an urgent prerequisite for no urgency gain. The new `CacheSweepScheduler` is nonetheless intentionally designed extensible (add a repo + another `ClearExpired` call in `sweep`) so the follow-up folds in cheaply.
  - **③ — process root cause: the sprint-note claim was unverified (Rule-15-class hallucination, sprint-note variant).**
    - **Finding (Analyst):** "_only the AI cache is swept_" was written from memory and conflated "method `ClearExpiredCache` **exists**" with "it is **scheduled/wired**" — the sprint-note analogue of Rule 15's `method-exists ≠ HTTP-route-registered` and Rule 24's prose-only-mention ban.
    - **Lane ③ — backlog-with-carry-forward-link.** Filed as **`retro-cand-sprint-note-claim-verification`** (backlog) — a retro candidate for the next ux3/Epic-11 retro proposing that factual runtime claims in sprint notes ("X is swept/wired/called") be grep-verified at write time or hedged as unverified. ARCH owns the decision (new Rule vs Rule 15/24 extension vs checklist note) at the retro ceremony; **not authored into `project-context.md` now.** Non-blocking, does not gate this story.
- Reference: `project-context.md` Rule 24 (Discovery Triage), Rule 15 (method-exists ≠ wired); origin: this story's prep grep + Party-Mode verification (2026-06-24).

### File List

_(to be filled by dev agent)_

- `apps/api/internal/services/cache_sweep_scheduler.go` (NEW)
- `apps/api/internal/services/cache_sweep_scheduler_test.go` (NEW)
- `apps/api/cmd/api/main.go` (MODIFIED — construct / start / shutdown wiring)
