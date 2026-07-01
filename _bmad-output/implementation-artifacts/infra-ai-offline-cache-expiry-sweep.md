# Story (Infra): `ai_cache` + `offline_cache` Scheduled Expiry Sweep

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->
<!-- Story key: infra-ai-offline-cache-expiry-sweep (non-epic infra story; whole-app benefit).
     Rule-24 ③ follow-up from infra-cache-entries-expiry-sweep (#98, 2026-06-24 Party-Mode).
     Pattern decision RATIFIED 2026-06-30 (Alexyu): Option A — extend the centralized
     CacheSweepScheduler — over Option B (per-cache self-sweep à la Douban). See Dev Notes
     "Pattern decision". -->

## Story

As the **Vido backend (single-NAS, long-running process)**,
I want the **two remaining orphaned DB-table caches — `ai_cache` and `offline_cache` — swept of expired rows on the same scheduled ticker that already sweeps `cache_entries`**,
so that **every persistent SQLite cache store has a scheduled expiry caller (closing the "3 orphaned caches" gap found during #98), and no cache table grows unbounded on row-count without manual intervention.**

## Pattern decision (RATIFIED — read FIRST)

This story had a genuine architecture fork, decided by Alexyu on 2026-06-30:

- **✅ CHOSEN — Option A: extend the centralized `CacheSweepScheduler` (#98).** Generalize the existing scheduler to sweep N DB-table caches in its single goroutine/ticker. `ai_cache` and `offline_cache` are **DB-table caches with the identical `ClearExpired(ctx) (int64, error)` shape** as `cache_entries`, and #98's own Discovery Triage already declared it "designed extensible (add a repo + another `ClearExpired` call in `sweep`) so the follow-up folds in cheaply." Option A reuses all of #98's CR-hardening (panic-recover, interval clamp, graceful stop, cold-start sweep, one settings key) and keeps a **single** goroutine so the three sweeps never contend the one SQLite writer lock concurrently.
- **❌ REJECTED — Option B: per-cache self-sweep à la Douban** (`internal/douban/cache.go:77` `cleanupLoop`). Would add two more tickers (3 independent goroutines on one writer lock), re-derive the loop/shutdown logic twice, and skip #98's hardening (Douban's loop has no panic-recover, no clamp, no settings-config). The sprint-note originally suggested mirroring Douban, but that was written **before** verifying these two are DB-table caches structurally identical to `cache_entries`; with #98 now shipped, converging is the lower-risk, DRY choice.

## Acceptance Criteria

1. **AC1 (multi-target sweep on the existing schedule):** `CacheSweepScheduler` sweeps `cache_entries` **plus** `ai_cache` (`ai.Cache.ClearExpired` via `AIService.ClearExpiredCache`) **plus** `offline_cache` (`OfflineCache.ClearExpired`) on the **same** recurring ticker, using the **same** existing settings key `cache_sweep_interval_minutes` and the same interval resolution (default 45 / disable ≤0 / clamp floor 5 / ceiling 7d). No new settings key, no second ticker, no second goroutine.
2. **AC2 (per-target error + panic isolation):** On a given tick the scheduler sweeps each target independently. A `ClearExpired` **error** from one target is logged at ERROR with the target name and does **NOT** prevent the remaining targets from being swept on that same tick. A **panic** in one target's sweep is recovered per-target (logged with the target name) and likewise does not abort the others or kill the goroutine. (Stronger than #98's single-target recover, which would have aborted the whole tick.)
3. **AC3 (nil / absent targets are safe):** A target whose underlying cache is absent — specifically the AI cache when **no AI provider is configured (`aiService == nil` at `main.go`)** — is **skipped at construction** and never produces a nil-func call/panic. `cache_entries` is still swept even when zero extra targets are supplied (backward-compatible with #98).
4. **AC4 (backward compatibility — #98 behavior preserved):** All existing #98 behavior is unchanged: `cache_entries` is still swept, interval resolution / disable / clamp / cold-start (boot) sweep / graceful `Stop()` + `ctx.Done()` lifecycle all identical. The constructor stays **source-compatible** (extra targets are variadic) so **every existing `cache_sweep_scheduler_test.go` test compiles and passes unchanged**.
5. **AC5 (main.go wiring):** The `ai_cache` + `offline_cache` targets are wired in `apps/api/cmd/api/main.go` where the scheduler is constructed (`:409`). `offline_cache` (always constructed at `:124`) is always added; the `ai_cache` target is added **only when `aiService != nil`** (`:214`). Start (`:642`) / Stop (`:682`) wiring is otherwise unchanged.
6. **AC6 (NEVER VACUUM; sequential sweep):** The sweep performs **only** the per-target `DELETE` via each cache's existing `ClearExpired`. It MUST NOT run `VACUUM` (inherited from #98 AC6). The three targets are swept **sequentially** (no errgroup / no concurrency) so they never hold the single SQLite writer lock simultaneously — this sequential property is the whole point of Option A.
7. **AC7 (preserve each cache's existing `ClearExpired` predicate — no SQL change):** Each target is swept by calling its **existing** `ClearExpired` unchanged. In particular `OfflineCache.ClearExpired` deletes only `WHERE expires_at < CURRENT_TIMESTAMP AND is_stale = 1` — the `is_stale = 1` guard is **intentional** (it keeps not-yet-stale offline fallback data available for offline mode) and MUST NOT be "fixed" to be more aggressive. `ai_cache` deletes `WHERE expires_at < ?`. No migration, no new SQL, no new repo/cache method.
8. **AC8 (tests + lint):** New unit tests cover: (a) a tick sweeps **all** registered targets exactly once; (b) one target erroring still sweeps the others (error isolation); (c) one target panicking still sweeps the others and does not crash (panic isolation); (d) a `SweepFunc`/`SweepTarget` with a nil cache/func is skipped at construction (no panic). All **pre-existing** #98 tests stay green. `go vet` + `staticcheck` clean (Rule 12), `-race` clean.

## Tasks / Subtasks

- [x] **Task 1 — Generalize `CacheSweepScheduler` to N targets (AC: #1, #2, #3, #4, #6, #7)**
  - [x] File: `apps/api/internal/services/cache_sweep_scheduler.go`.
  - [x] Add a small exported abstraction (place near the top of the file):
    - `type ExpirableCache interface { ClearExpired(ctx context.Context) (int64, error) }`
    - `type CacheSweepTarget struct { name string; sweep func(context.Context) (int64, error) }` (fields unexported — constructed only via the helpers below; exporting the **type** while keeping fields private avoids the "exported func returns unexported type" lint).
    - `func SweepTarget(name string, c ExpirableCache) CacheSweepTarget { return CacheSweepTarget{name: name, sweep: c.ClearExpired} }`
    - `func SweepFunc(name string, fn func(context.Context) (int64, error)) CacheSweepTarget { return CacheSweepTarget{name: name, sweep: fn} }` — needed for `AIService.ClearExpiredCache`, whose method **name differs** from `ClearExpired` so it does not satisfy `ExpirableCache` directly.
  - [x] Struct change: replace the `cacheRepo repository.CacheRepositoryInterface` field (`:40`) with `targets []CacheSweepTarget`. Keep `settingsRepo`, `mu`, `stopCh`, `stopped`.
  - [x] Constructor (`:51`) becomes variadic, source-compatible:
    ```go
    func NewCacheSweepScheduler(
        cacheRepo repository.CacheRepositoryInterface,
        settingsRepo repository.SettingsRepositoryInterface,
        extraTargets ...CacheSweepTarget,
    ) *CacheSweepScheduler {
        var targets []CacheSweepTarget
        if cacheRepo != nil { // keeps NewCacheSweepScheduler(nil, settingsRepo) tests valid
            targets = append(targets, SweepTarget("cache_entries", cacheRepo))
        }
        for _, t := range extraTargets {
            if t.sweep != nil { // skip a nil/absent cache (AC3)
                targets = append(targets, t)
            }
        }
        return &CacheSweepScheduler{targets: targets, settingsRepo: settingsRepo, stopCh: make(chan struct{})}
    }
    ```
  - [x] Refactor `sweep` (`:155`) into a loop with **per-target** isolation (move the recover + context-cancel handling into `sweepOne`):
    ```go
    func (s *CacheSweepScheduler) sweep(ctx context.Context) {
        for _, t := range s.targets {
            s.sweepOne(ctx, t)
        }
    }
    func (s *CacheSweepScheduler) sweepOne(ctx context.Context, t CacheSweepTarget) {
        defer func() {
            if r := recover(); r != nil {
                slog.Error("Cache sweep panicked", "target", t.name, "recover", r)
            }
        }()
        if _, err := t.sweep(ctx); err != nil {
            if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
                slog.Debug("Cache sweep interrupted by context", "target", t.name, "error", err)
                return
            }
            slog.Error("Cache sweep failed", "target", t.name, "error", err)
        }
    }
    ```
  - [x] Update the struct doc-comment (`:35-38`, "targets cache_entries ONLY") to reflect N targets; add `"targets", len(s.targets)` to the `slog.Info("Cache sweep scheduler started", ...)` line (`:106`) for observability. Do **NOT** add any `VACUUM` (AC6). Keep `run()`/cold-start/`Start`/`Stop`/`resolveInterval` otherwise **byte-identical** — the cold-start `s.sweep(ctx)` now drains all targets at boot for free.
- [x] **Task 2 — Wire `ai_cache` + `offline_cache` targets into `main.go` (AC: #5, #3)**
  - [x] File: `apps/api/cmd/api/main.go`, at the scheduler construction site (`:409`). Replace the single-arg construct with:
    ```go
    // Extra DB-cache sweep targets (infra-ai-offline-cache-expiry-sweep). offline_cache is always
    // constructed (:124); the AI cache only exists when an AI provider is configured (:214).
    cacheSweepExtra := []services.CacheSweepTarget{
        services.SweepTarget("offline_cache", offlineCache),
    }
    if aiService != nil {
        cacheSweepExtra = append(cacheSweepExtra, services.SweepFunc("ai_cache", aiService.ClearExpiredCache))
    }
    cacheSweepScheduler := services.NewCacheSweepScheduler(repos.Cache, repos.Settings, cacheSweepExtra...)
    ```
  - [x] Confirm in-scope at `:409`: `offlineCache` (`cache.NewOfflineCache`, `:124`) and `aiService` (`services.NewAIService`, `:214`, **nilable**). No change to the start (`go cacheSweepScheduler.Start(cacheSweepCtx)`, `:642`) or stop (`cacheSweepScheduler.Stop()`, `:682`) lines.
  - [x] Update the adjacent `slog.Info("Cache sweep scheduler initialized")` comment (`:408`) from "cache_entries expiry sweep" to note the 3 targets.
- [x] **Task 3 — Unit tests (AC: #8, #2, #3, #4)**
  - [x] File: `apps/api/internal/services/cache_sweep_scheduler_test.go` (extend, do not rewrite — all existing tests MUST keep passing per AC4).
  - [x] Add a tiny local `fakeTarget` helper or reuse `mockCacheRepo`: e.g. a struct with a call counter + configurable error/panic, adapted via `SweepFunc("name", fake.clear)`.
  - [x] **Multi-target (AC1):** construct with `cacheRepo` + two extra `SweepFunc` targets; call `s.sweep(ctx)`; assert all three call-counters == 1.
  - [x] **Error isolation (AC2):** middle target returns `assert.AnError`; assert the other two still incremented (== 1) and `assert.NotPanics`.
  - [x] **Panic isolation (AC2):** one target's func `panic("boom")`; assert the others still swept and the scheduler does not crash (`assert.NotPanics`).
  - [x] **Nil-skip (AC3):** `NewCacheSweepScheduler(cacheRepo, nil, SweepFunc("x", nil))` (and/or `SweepTarget` with a nil interface) → constructing + `sweep` does not panic; only the real targets run.
  - [x] **Backward-compat (AC4):** keep `mockCacheRepo` + the existing `NewCacheSweepScheduler(cacheRepo, nil)` tests verbatim; confirm `s.sweep(ctx)` still yields `cacheRepo.callCount() == 1`.
  - [x] Run `go test ./internal/services/ -run CacheSweep -v` (+ `-race`); then `pnpm lint:all` (Rule 12).

## Dev Notes

### Origin & the "3 orphaned caches" finding

- Spawned by the **Discovery Triage** of `infra-cache-entries-expiry-sweep` (#98), fact-verified in a Party-Mode review on 2026-06-24. Grepping `ClearExpired` callers across `apps/api` confirmed: **no scheduled production caller existed for any repo/cache `ClearExpired`** — `cache_entries` (fixed by #98), `ai_cache`, and `offline_cache` were all orphaned; only **Douban** self-sweeps. #98 fixed `cache_entries` (the one store the Discover facet fan-out write-amplifies); this story closes the remaining two.
- The sibling's sprint-note "only the AI cache is swept" was **doubly false** (the AI cache is NOT swept; the one that self-sweeps is Douban). This story's existence corrects that record. The process root-cause (unverified sprint-note claim, a Rule-15-class "method-exists ≠ wired" hallucination) is separately tracked as the retro candidate `retro-cand-sprint-note-claim-verification`.

### What kind of caches these are (verified 2026-06-30)

- `ai.Cache` — **DB-table** (`ai_cache`, `*sql.DB`), TTL 30 days (`internal/ai/cache.go:19`), written **per scan** that hits the AI parser. `ClearExpired(ctx) (int64, error)` = `DELETE FROM ai_cache WHERE expires_at < ?` (`internal/ai/cache.go:189`). Reached from `main.go` via `AIService.ClearExpiredCache` (`internal/services/ai_service.go:354`, which just calls `s.cache.ClearExpired`). **Risk: medium** (real, recurring growth) — the main motivation.
- `OfflineCache` — **DB-table** (`offline_cache`, `*sql.DB`), written **only when an external service degrades** (`internal/cache/offline_cache.go`). `ClearExpired(ctx) (int64, error)` = `DELETE FROM offline_cache WHERE expires_at < CURRENT_TIMESTAMP AND is_stale = 1` (`:177`). **Risk: low** (rarely written). Note the conservative `is_stale = 1` predicate — see AC7.
- Both share the **same SQLite DB** as `cache_entries` (and Douban), which is exactly why one sequential sweep goroutine (Option A) is preferable to three independent tickers.

### The generalization design (Option A — exact shape)

- The scheduler moves from one hard-wired `cacheRepo` to a `[]CacheSweepTarget`, each a `{name, sweep func}`. `cache_entries` stays the **first** target (added from the existing `cacheRepo` param), so #98's tests and behavior are untouched; the AI + offline targets are **appended** via the variadic `extraTargets`.
- `SweepTarget` adapts anything satisfying `ExpirableCache` (i.e. `repos.Cache`, `offlineCache`). `SweepFunc` adapts a bare `func(ctx)(int64,error)` — required because `AIService.ClearExpiredCache` has a **different method name** and so does not implement `ExpirableCache`. (Do NOT add a `ClearExpired` alias to `AIService` just to fit the interface — `SweepFunc` exists precisely to avoid polluting that API.)
- **Per-target isolation** is the one deliberate behavior upgrade over #98: `sweepOne` wraps each target in its own `recover` + error log keyed by `target` name, so one bad cache can never starve the others on a tick. #98's single-target `sweep` had the recover at the whole-tick level; with multiple targets that would let one panic skip the rest.

### Anchors (verified against the as-merged tree, 2026-06-30)

- `apps/api/internal/services/cache_sweep_scheduler.go`: const block `:13-27` (settings key + default/floor/ceiling — **reuse all of it, add nothing**); `CacheSweepSchedulerInterface` `:30-33` (unchanged); struct `:39-45` (swap `cacheRepo`→`targets`); `NewCacheSweepScheduler` `:51-60` (make variadic); `resolveInterval` `:68-94` (unchanged); `Start` `:99-108` (add `targets` to the log line); `run`/cold-start/ticker `:117-148` (unchanged — cold-start now drains all targets); `sweep` `:155-172` (refactor into `sweep`+`sweepOne`); `Stop` `:176-184` (unchanged).
- `apps/api/internal/services/cache_sweep_scheduler_test.go`: existing `mockCacheRepo` `:16-54`, `resolveInterval`/`Stop`/`Sweep`/`Run`/`Start` tests `:56-302` — **all must stay green** (AC4); add the new multi-target/isolation/nil-skip cases alongside.
- `apps/api/internal/ai/cache.go:189` — `Cache.ClearExpired`; `apps/api/internal/services/ai_service.go:353-355` — `AIService.ClearExpiredCache` wrapper (also in `AIServiceInterface` `:41-42`).
- `apps/api/internal/cache/offline_cache.go:177` — `OfflineCache.ClearExpired` (conservative `is_stale=1` predicate — AC7).
- `apps/api/cmd/api/main.go`: `offlineCache` `:124`; `aiService` (nilable) `:214`; scheduler construct `:409`; start `:642`; stop `:682`. Existing `repos.Cache` is `CacheRepositoryInterface` and already satisfies `ExpirableCache`.
- Pattern-to-reject reference (do NOT copy): `apps/api/internal/douban/cache.go:77` `cleanupLoop` (the self-sweep we are NOT mirroring).

### Inherited guardrails (from #98 — still apply)

- **NEVER `VACUUM` on the ticker** (AC6). `auto_vacuum=NONE` → `DELETE` frees pages for reuse but doesn't shrink the `.db`; disk reclamation is a rare manual/admin action, never this scheduler's job. This story keeps **row count** healthy, not file size.
- **Sequential, not concurrent.** Three small `DELETE`s in series at a 45-min cadence is trivial and avoids three concurrent SQLite writers — the explicit reason Option A beats Option B.
- **Safe by construction.** Each `ClearExpired` deletes only rows reads already treat as expired (`ai_cache`: `expires_at < now`; `offline_cache`: expired **and** stale). The sweep can never delete a live-served hit. Reuses each cache's existing predicate verbatim (AC7) — no new SQL, no migration.

### Layering note (Rule 4)

A scheduler is an infrastructure driver, not an HTTP handler, so it may take repositories/caches directly (mirrors `BackupScheduler`/`ScanScheduler`/#98). `cache_entries` uses `repository.CacheRepositoryInterface`; `ai_cache` is reached through the existing `AIService.ClearExpiredCache` (a service method, already in scope in `main.go`); `offline_cache` is the `*cache.OfflineCache` value already constructed in `main.go`. No new service-layer wrapper is introduced.

### Project Structure Notes

- All edits in `apps/api/internal/services/cache_sweep_scheduler.go` (+ co-located `_test.go`, Rule 9) and `apps/api/cmd/api/main.go`. Backend-only (Rule 1).
- No new error codes (Rule 7 — internal logged failures only), no migration, no schema change, no new repo/cache method, no new settings key (reuses `cache_sweep_interval_minutes`).
- **Cross-stack split check:** 3 backend tasks, 0 frontend tasks → single story, no split required.

### Time-dependent visual coverage

- **N/A — no wall-clock-reading `apps/web/src/components/**/*.{ts,tsx}` touched.** Backend-only infra story (Go scheduler); renders no UI, captures no visual baselines. (Rule 23 is frontend-only.)

### References

- [Source: `_bmad-output/implementation-artifacts/infra-cache-entries-expiry-sweep.md`] — #98, the canonical scheduler this story extends; its Discovery Triage (the "3 orphaned + 1 self-sweep" finding) that filed this story and declared the scheduler "designed extensible."
- [Source: `apps/api/internal/services/cache_sweep_scheduler.go`] — the scheduler being generalized (anchors above).
- [Source: `apps/api/internal/ai/cache.go:189`, `apps/api/internal/services/ai_service.go:353-355`] — `ai_cache` expiry path.
- [Source: `apps/api/internal/cache/offline_cache.go:177`] — `offline_cache` expiry path (conservative `is_stale=1` predicate, AC7).
- [Source: `apps/api/internal/douban/cache.go:77`] — the self-sweep precedent considered for Option B and rejected.
- [Source: `apps/api/cmd/api/main.go:124,214,409,642,682`] — offline cache / AI service / scheduler construct / start / stop sites.
- [Source: `project-context.md` §5 (Background Tasks — goroutines + graceful shutdown), Rule 1, Rule 4, Rule 9, Rule 12, Rule 13 (error completeness), Rule 14 (Resource Lifecycle)] — governing rules.

## Dev Agent Record

### Agent Model Used

Opus 4.8 (1M context) — `claude-opus-4-8[1m]`, via BMAD dev-story workflow (agent "Amelia").

### Debug Log References

- `go test ./internal/services/ -run CacheSweep -v -race` → PASS. New `TestCacheSweepScheduler_MultiTarget` cases (AC1/AC2/AC3/AC4) green; logs confirm per-target isolation with the target name (`ERROR Cache sweep failed target=mid …`, `ERROR Cache sweep panicked target=boom recover=boom`). All pre-existing #98 tests unchanged and green.
- `go test ./...` (full api regression) → every package `ok`, incl. `github.com/vido/api/cmd/api` (proves the main.go wiring compiles).
- `go vet ./...` exit 0; `pnpm nx lint api` → `go vet` + `staticcheck@2026.1` clean; `gofmt -l` empty on all 3 changed Go files.
- `pnpm nx test web` → 202 files / 2251 tests passed; `test:cleanup` → "No test processes found" (no orphaned workers).

### Completion Notes List

- 🔗 **AC Drift: NONE** — grep `CacheSweepScheduler|cache_sweep_interval|NewCacheSweepScheduler` across `_bmad-output/implementation-artifacts/*.md` → 3 hits: this story; #98 `infra-cache-entries-expiry-sweep` (parent — extended **additively**, AC4 preserves all #98 behavior and the constructor is kept source-compatible via a variadic tail → **REUSE not DRIFT**); `ux3-4-2b-downloads-sse-be.md` (cites the scheduler only as a lifecycle *precedent to copy*, sets no contract on it → REUSE). No prior AC contract is contradicted.
- 📎 **Contract Stamps: NONE** — no `[@contract-v*]` stamps in this story; upstream #98 carries none either (pre-Rule-20, implicit v0), so no `confirmed against` ack line is required.
- 🎭 **A11y Pre-Flight: N/A** (100% backend — no `apps/web/` files touched).
- 🎨 **UX Verification: SKIPPED** — no UI changes in this story (backend-only Go scheduler).
- **Per-target isolation** is the one deliberate upgrade over #98: `sweep` now loops all targets and delegates each to `sweepOne`, which owns its own `recover` + error log keyed by `target` name. One target erroring or panicking neither aborts the remaining targets on the tick nor kills the goroutine (AC2) — asserted by the error-isolation and panic-isolation tests.
- **Defensive nil-guard** added to `SweepTarget` (`if c == nil`) alongside the constructor's `t.sweep != nil` filter, so AC3 holds for BOTH a nil `ExpirableCache` (via `SweepTarget`) and a nil func (via `SweepFunc`) — both dropped at construction (asserted via `len(s.targets)`).
- **Backward-compat (AC4):** all pre-existing #98 tests kept verbatim and green; `NewCacheSweepScheduler(cacheRepo, nil)` still yields exactly the `cache_entries` target and one sweep call.
- **AC6/AC7 preserved:** sweep is `DELETE`-only via each target's existing `ClearExpired`, sequential (no errgroup/concurrency), never `VACUUM`. No new SQL/migration; `offline_cache`'s conservative `expires_at < now AND is_stale = 1` predicate is used unchanged.
- **Pre-existing failures: NONE** detected (full api + web suites green).

### Discovery Triage

<!-- Rule 24 (project-context.md). Out-of-scope findings surfaced during this story MUST be tracked. -->

- **Did this story discover any work outside its current scope?** **YES — one item, pre-noted:**
  - **③ — disk reclamation / `.db` file shrink is still not addressed (by design).** Like #98, this sweep keeps row-count healthy but never `VACUUM`s, so the on-disk `.db` stabilizes at its working-set high-water mark rather than shrinking. This is the correct trade-off for ongoing caches (AC6). If file-size reclamation is ever genuinely needed it is a separate rare manual/admin action — file a `backlog` entry at that point (not now — YAGNI, no current consumer).
- Reference: `project-context.md` Rule 24; the sibling retro candidate `retro-cand-sprint-note-claim-verification` (already filed) tracks the process root cause that surfaced this whole cache-sweep family.

### File List

- `apps/api/internal/services/cache_sweep_scheduler.go` (MODIFIED — `ExpirableCache`/`CacheSweepTarget`/`SweepTarget`/`SweepFunc`; struct `cacheRepo`→`targets`; variadic constructor; `sweep`→`sweep`+`sweepOne` per-target isolation)
- `apps/api/internal/services/cache_sweep_scheduler_test.go` (MODIFIED — multi-target / error-isolation / panic-isolation / nil-skip tests; existing tests unchanged)
- `apps/api/cmd/api/main.go` (MODIFIED — build `ai_cache` (nil-guarded) + `offline_cache` sweep targets, pass to `NewCacheSweepScheduler`)

## Change Log

| Date       | Change                                                                                                                   |
| ---------- | ------------------------------------------------------------------------------------------------------------------------ |
| 2026-06-30 | Story created (SM create-story). Pattern fork resolved by Alexyu → Option A (extend `CacheSweepScheduler`). Anchors verified against as-merged tree; #98 backward-compat preserved via variadic constructor. Status → ready-for-dev. |
| 2026-07-01 | Implemented (dev-story). Generalized `CacheSweepScheduler` to N **sequential** targets (`ExpirableCache` + exported `CacheSweepTarget` + `SweepTarget`/`SweepFunc`; struct `cacheRepo`→`targets`; variadic source-compatible constructor; `sweep`→`sweep`+`sweepOne` per-target `recover`/error isolation keyed by target name; `Start` log gains `targets`). Wired `offline_cache` (always) + `ai_cache` (nil-guarded on `aiService`) targets at `main.go:409`. Added multi-target / error-isolation / panic-isolation / nil-skip tests; all #98 tests unchanged & green. `go vet` + `staticcheck@2026.1` + `gofmt` clean, `-race` clean, full api + web regression green (2251 web tests). Status → review. |
