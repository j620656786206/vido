# Story sub-1.6: Wire triggering, gating, and progress

Status: ready-for-dev

**Epic:** `epic-subtitle-pipeline-m1` (M1) · **Risk: 🟡 MEDIUM (wiring + one HTTP surface)** · **BACKEND-ONLY**
**Source:** `epics-subtitle-pipeline.md` § Story 1.6 · architecture **D5** + **V2/V3/V6** + **P8** · IR rulings **F4** (G2/G4 bars live here) + **F9** (deployment-doc half)
**Depends on (merged):** **sub-1-5b** (`ProcessItem` — acked below) · sub-1-3 (stages) transitively. This is the story where the pipeline becomes **user-reachable**.
**Blocks:** nothing in M1 — this closes the backend chain. (sub-1-7b and Epic 2 are parallel tracks.)
**Cross-stack split check:** backend tasks = 6, frontend tasks = **0** → single story (F2/F3/F5 surfaces already render `message`/`stage` fail-soft — verified in 1-3).

---

## Story

As a NAS owner,
I want the pipeline triggered automatically on media-add and on demand, gated when unconfigured, with live progress,
so that subtitles appear without manual steps and failures are always visible.

---

## 🔎 Codebase findings (verified 2026-07-27)

1. **The scan-complete hook already exists — and is already occupied.** `ScannerService.SetOnScanComplete(fn func())` (`scanner_service.go:82`, fired at `:314` when files were created/updated) is wired at **`main.go:422`**. FR13 therefore needs **zero `scanner_service.go` edits** — main.go composes the existing callback (`old(); pipeline.EnqueueMissing(ctx)`). The architecture's V3 "adds `scanner_service.go ✏️`" is unnecessary; AC #7 corrects the delta tree (the third such correction this epic).
2. **The flag seam is exactly one call:** `bp.engine.Process(...)` at `batch.go:~244`. D5's `legacy | pipeline` conditional wraps that single call site — nowhere else.
3. **Movie enumeration exists:** `FindMissingZhHantSubtitle` (`movie_repository.go:892`, Route C heritage — "movies with a media file but no zh-Hant subtitle"). Episode-grain: locate the 9R-16 generation-batch enumeration at implementation (Task 2 decision tree).
4. **`docs/deployment.zh-TW.md` does not exist** — every other user-facing doc has a zh-TW twin (sse-event-types, testsprite-local-dev, unraid-installation-guide); deployment.md never got one. Pre-existing Rule 17 debt, discovered now → **lane ③** entry `backlog-deployment-doc-zh-tw-twin` (filed with this story); F9's AC here edits EN and the twin inherits the section when it materializes. *(Alexyu may overrule: translate the full twin inside this story — see Open Questions.)*

---

## Acceptance Criteria

### AC #1 — D5: the feature flag gates exactly one seam

**Given** `VIDO_SUBTITLE_PIPELINE_MODE` = `legacy` (default) | `pipeline` (config.go `loadString` pattern + validation; env-var, not a settings row — the migration-029 cleanup of `new_shell_enabled` is the cautionary precedent for table-backed flags), **when** set to `pipeline`, **then** the `bp.engine.Process(...)` call at `batch.go:~244` becomes `pipeline.ProcessItem(ctx, MediaRef{...}, ProcessOptions{})`; `legacy` behaves byte-identically to today. **One conditional, one call site** — the flag appears nowhere inside pipeline stages, handlers, or the scanner path (D5's ban). Flag read once at startup into the wiring, not per item.

### AC #2 — FR13: auto-enqueue on scan complete (zero scanner edits)

**Given** a completed scan with created/updated files, **then** main.go's composed `onScanComplete` callback calls `pipeline.EnqueueMissing(ctx)`, which enumerates eligible items (movies via `FindMissingZhHantSubtitle`; episodes via the located 9R-16 path) and enqueues them. Eligibility re-checks are cheap here — `ProcessItem`'s P5 pre-flight (1.5b) is the authoritative gate, so over-enumeration is safe.

- Enqueue is honoured **only in `pipeline` mode** — in `legacy` mode the callback composition still runs the pre-existing `:422` behaviour untouched.
- The existing `:422` callback body is preserved exactly (wrap, don't replace).

### AC #3 — Worker pool: fixed concurrency 2, graceful lifecycle

**Given** AD #5 + NFR-P3 (M1), **then** the pipeline owns a pool: **2 workers** (const `PipelineConcurrencyM1 = 2`), buffered channel (cap 1024), **non-blocking enqueue with drop-and-`slog.Warn`** on overflow (fail-soft — the next scan re-enqueues; mirrors the SSE hub's drop discipline), **in-flight dedup** by `MediaRef` key (bounded map, Rule 14 — an item already queued/running is not double-enqueued), `Start(ctx)`/`Stop()` lifecycle in main.go's goroutine zone + graceful-shutdown block (the `retry/scheduler.go` pattern, per the 13-4a precedent).

### AC #4 — FR12: `POST /api/v1/subtitles/pipeline/run`

**Given** F2-D-v2's 生成字幕 button (V2), **then** a **new** `handlers/subtitle_pipeline_handler.go` (the existing `subtitle_handler.go` stays 🔒 — manual search path unchanged, D3):

- Request: `{media_id, media_type, force?}` (`media_type` ∈ movie|series|episode — sub-1-2 AC #1 vocabulary).
- Behaviour: capability gate (AC #5) → dedup-aware enqueue → **`202 Accepted`** with `{success, data: {status: "queued"|"already_queued", media_id}}`. Never synchronous — a translate run is minutes; progress flows over SSE (AC #6).
- Rule 3 envelope · Rule 10 versioning · **Swagger annotations + `swag init`** (Rule 15 — this story adds the epic's only new HTTP surface) · `RegisterRoutes` called from main.go and **verified** (Rule 15's route↔client check; the 10-2 precedent).
- 400 on unknown `media_type`; 404 when the item doesn't exist (repo lookup); gate failure per AC #5.

### AC #5 — FR23: one capability gate, three entry points (V6)

**Given** no configured translation key, **then** a single check owned by the `Pipeline` (`configured func() bool` field, wired from `cfg.HasClaudeKey` — V6's "top of pipeline.go") governs **all three** entry paths (endpoint, batch seam, scanner enqueue):

- Endpoint → `409` with **`AI_NOT_CONFIGURED`** (reuse — `ErrAINotConfigured` exists; **zero new Rule 7 codes**, zero registry edits) + zh-TW message: `尚未設定翻譯服務金鑰` + suggestion `請設定 CLAUDE_API_KEY 環境變數後重啟（設定頁面將於 M1.5 提供）` — matching F5-D-v2's 尚未設定 framing and J3's env-var reality.
- Scanner/batch paths → gate short-circuits **before enqueue** with one `slog.Info` (not per-item spam) — no silent failure, no queued work that can only fail.
- **Ruling (scope-honest):** the gate closes the **whole** pipeline entry in M1, including keyless extract-only routes — matches F5's wholesale-gated UX and V6's single-gate design. Keyless zh-extraction is a noted P2 candidate, not built.

### AC #6 — FR33/P8: SSE progress wiring

**Given** 1.5b's nil-safe progress hook, **then** main.go connects it to the existing SSE hub: `subtitle_progress` events (**event type unchanged** — `sse/hub.go` stays 🔒) carrying `{media_id, media_type, stage, message}` with 1-3's stage values (`probing`/`extracting`/`translating`/`skipped` now reach the wire for the first time). Cadence is already P8-correct (once per chunk + stage transitions — the hook's call sites, 1.5b). Messages are zh-TW user-facing strings composed at the wiring layer (e.g. `抽取內嵌字幕中…`, `翻譯中（第 N/M 段）`). Frontend consumes fail-soft today (`useSubtitleSearch.ts:21` — verified in 1-3); richer stage UI is F3-D-v2's existing surface reading `message`.

### AC #7 — [F4 ruling] The G2/G4 measurable bars — **PROPOSED numbers, confirm before dev starts**

**Given** M1's purpose is validating trust on real hardware, **then** these bars are citable ACs of M1 (measured at the pilot on the DS920+; per-item timing comes free from `subtitle_runs.started_at/completed_at`):

| # | Bar | Proposed threshold |
|---|---|---|
| (a) | **NFR-P1 resource bound** during one item | pipeline-attributable sustained CPU ≤ **1 core** (≈25% of J4125) and incremental RSS ≤ **256 MB**; concurrent playback (Video Station/Plex) stays functional — verified via `docker stats` per the pilot procedure noted in the run log |
| (b) | **time-to-`.zh-Hant.srt`** | translate path: ~600-cue episode ≤ **5 min**, ~1,200-cue movie ≤ **10 min** · direct/convert path (no LLM) ≤ **60 s** |
| (c) | **trust bar** | ≥ **90%** of a **20-item** pilot sample accepted **without hand-editing** (protocol: skim + spot-play 3 random cues per item; recorded in pilot notes) |

**⚠️ Discovery under (b):** the PRD's *"tens of seconds … one translation call"* assumed a single call; 1.5a ships sequential chunk=10 → a 600-cue episode is ~60 chunks = **minutes, not tens of seconds**. The bar above is the honest number; the PRD's prose estimate is flagged (F2-class, optional edit — Open Questions).

### AC #8 — [F9 ruling] Deployment docs: EN section now, zh-TW twin as filed debt

**Given** `Dockerfile:47` (ffmpeg bundled) and `docker.yml:80` (amd64+arm64) are shipped infrastructure, **then** `docs/deployment.md` gains a short section (under § Prerequisites): the image **bundles ffmpeg/ffprobe** (no host install; required by the subtitle pipeline — absence degrades silently, the 2026-06 audit), the image is **multi-arch**, and the `VIDO_SUBTITLE_PIPELINE_MODE` + `CLAUDE_API_KEY` env vars are documented under § Configuration. **`docs/deployment.zh-TW.md` does not exist** (Finding 4) → the twin is `backlog-deployment-doc-zh-tw-twin` (lane ③, filed with this story); it inherits this section. NFR-S3's HTTPS half stays with Story 2.1.

### AC #9 — Tests + scope fence

**Tests (Rule 9/16):** flag seam (both modes, byte-identical legacy path via a spy engine); enqueue dedup + overflow-drop + mode-gating; pool start/stop with ctx cancel (no goroutine leak — `goleak`-style or WaitGroup assertion); handler table (202 queued / already_queued / 409 gate / 400 / 404) + envelope shape; gate short-circuit before enqueue (spy: zero enqueues when unconfigured); SSE wiring emits stage+message per hook call (fake hub). `go test ./...` + `pnpm lint:all` green.

**Fence:** ❌ no frontend (F2's button already POSTs? — no: the FE call-site wiring to the new endpoint is **sub-1-7b-adjacent but NOT here**; M1's button wiring rides the existing generation dialog surface — if a FE edit turns out to be required to point 生成字幕 at the new endpoint, that is a **lane ② discovery**, stop and file it) · ❌ no batch-scope UI (FR34 P2) · ❌ no cost estimate (FR14 P2) · ❌ no key-config UI (2-1) · ❌ no new Rule 7 codes · ❌ no `scanner_service.go` / `sse/hub.go` / `subtitle_handler.go` edits.

### AC #10 — Architecture micro-sync

Delta tree: `scanner_service.go ✏️` → `🔒 (hook existed; composed in main.go — corrected at 1.6 drafting)` · add `handlers/subtitle_pipeline_handler.go 🆕` · main.go already ✏️.

---

## Tasks / Subtasks

- [ ] **Task 1 — Flag + config (AC #1):** `VIDO_SUBTITLE_PIPELINE_MODE` in config.go (+validation+test); the one-conditional seam at `batch.go:~244` with a spy-engine byte-identity test for legacy.
- [ ] **Task 2 — Pool + enqueue (AC #2, #3):** pool with lifecycle/dedup/overflow; `EnqueueMissing` (movies via `FindMissingZhHantSubtitle`; **decision tree:** locate the 9R-16 episode enumeration — found ⇒ reuse; absent ⇒ expand scope with a new AC per Rule 24 lane ①, do not silently build a new query); main.go callback composition preserving `:422`.
- [ ] **Task 3 — Endpoint (AC #4):** new handler + routes + Swagger + `swag init` + main.go wiring + Rule 15 route verification.
- [ ] **Task 4 — Gate (AC #5):** `configured` wiring, three-entry-point coverage, zh-TW messages, tests.
- [ ] **Task 5 — SSE + docs (AC #6, #8):** progress-hook→hub adapter + zh-TW stage messages; deployment.md EN section; file `backlog-deployment-doc-zh-tw-twin` in sprint-status (verify not already filed).
- [ ] **Task 6 — Sync + gates (AC #7, #9, #10):** record the pilot-bar numbers as confirmed (or adjusted) by Alexyu; delta-tree corrections; full test + lint gates; Rule 20 ack recorded.

---

## Dev Notes

- **Rule 20 acks (record verbatim):** `confirmed against [@contract-v1] sub-1-5b AC #1` (`ProcessItem`/`MediaRef`/`ProcessOptions`) · stage constants + sentinels via sub-1-3 (registry codes, no stamp) · `MediaRef.MediaType` vocabulary per sub-1-2 AC #1.
- **This story stamps nothing new** — it is pure composition; the HTTP request/response shape is v0 until Epic 2/FE work needs it stamped.
- **main.go touch points:** flag read → pipeline construction (deps: repos, cache, placer, converter, translation service, `modelID` from `cfg.GetClaudeModel()`/default, `configured` from `HasClaudeKey`) → pool Start + shutdown hook → scan-callback composition → handler registration → SSE adapter. Rule 15's main.go-wiring check applies to every one.
- **Why 202-async:** a translate item is minutes (AC #7b); holding HTTP open invites proxy timeouts and duplicate submits. SSE is the progress channel (J2), the run row is the record.
- **Rule 12/13/14/17** all in play; zh-TW strings live at the wiring/handler layer only (Rule 3's `message`/`suggestion` discipline).

### Time-dependent visual coverage

**N/A — no wall-clock-reading components touched.** Backend-only Go. Rule 23 does not apply.

### References

- [architecture #D5 (:255-257) · #V2/V3/V6 (:546-557) · #P8 (:380-382) · § M1 acceptance (spec §8)]
- [`apps/api/internal/subtitle/batch.go`:238-246 · `services/scanner_service.go`:78-83,312-315 · `cmd/api/main.go`:422 · `repository/movie_repository.go`:889-918 · `internal/config/config.go`:105-131]
- [IR report 2026-07-27 — F4/F9 rulings + the Dockerfile:47/docker.yml:80 verification]
- [`project-context.md`#Rule 3/10/13/14/15/17/24 · AD #5 · NFR-P3]

---

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Pre-recorded at authoring:**
  - **① scanner hook already exists + occupied** → absorbed as AC #2's zero-edit composition ruling + AC #10's delta-tree correction.
  - **③ `docs/deployment.zh-TW.md` missing entirely** (pre-existing Rule 17 debt) → `backlog-deployment-doc-zh-tw-twin` filed with this story (bidirectional); AC #8 edits EN only.
  - **① PRD "tens of seconds" vs chunked reality** → absorbed as AC #7(b)'s honest bar + flagged as an optional PRD edit (Open Questions).
  - Episode-enumeration decision tree (Task 2) may add a lane-① AC at implementation.
- Reference: `project-context.md` Rule 24.

### File List

---

## Open Questions for Alexyu (AC #7 numbers need your confirmation — the rest proceed on stated rulings)

1. **G2/G4 bars (AC #7):** (a) ≤1 core / ≤256 MB · (b) episode ≤5 min, movie ≤10 min, direct ≤60 s · (c) trust ≥ **90%** over 20 items. Confirm or adjust — **(c) especially: X=90 is my proposal, the AC says X is yours.**
2. **PRD prose:** "time-to-`.zh-Hant.srt` on the order of tens of seconds (… one translation call)" is inconsistent with chunk=10 sequential reality. Edit the PRD line, or leave it and let the pilot report supersede?
3. **Deployment zh-TW twin:** filed as backlog (lane ③). If you'd rather 1.6 create the full translation now, say so and Task 5 expands.
