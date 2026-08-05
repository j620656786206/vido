# Story sub-2.2a: `untranslated` terminal status + en-only writeback + translate-only resume

Status: done

**Epic:** `epic-subtitle-pipeline-m1-5` (M1.5 follow-up) · **Risk: 🔴 HIGH-VALUE/LOW-RISK** · **BACKEND-ONLY**
**Source:** party-mode ruling 2026-08-05 (Sally 主裁, Alexyu confirmed ×2 — recorded in sprint-status `backlog-f5-not-configured-panel-copy-ruling`) · promotes `backlog-subtitle-untranslated-status-backend`
**Split:** α of the α/β/γ three-way split. Backend tasks 4 / frontend tasks **0** → single story.
**Depends on:** nothing. **Do this FIRST — it alone stops the bleeding.**
**Blocks:** sub-2-2b (consumes the new enum value).

---

## Story

As a NAS owner who ran 生成字幕 before setting a Claude key,
I want the system to REMEMBER that an English SRT was produced and translation was skipped,
so that the library stops telling me 缺字幕 over a paid-for subtitle, and a second click costs a translate-only run instead of a full ASR re-run.

---

## 🔎 Findings (verified 2026-08-05, party mode)

1. **THE BUG — en-only completion writes nothing back.** `transcription_service.go:448`: the 9R-16 AC 12 writeback fires ONLY on a zh-Hant place (`if zhSRTPath != ""`); its own comment says "en-only runs write nothing". Chain: English SRT is already on disk (persisted before `translateAndPersist`) → `subtitle_status` stays `not_searched` → the badge derives from EMBEDDED tracks (which never include the just-written external SRT) → a file with no embedded tracks badges **缺字幕** → the user reasonably clicks 生成字幕 again → **the ENTIRE ASR re-runs at full Whisper cost**. A money-burning silent-degradation loop.
2. **The silent skip.** `transcription_service.go:416`: `if translate && s.translationService != nil && s.translationService.IsConfigured()` — key missing ⇒ `IsConfigured()` false ⇒ the translate branch is skipped with no error, no SSE signal, no record. Job completes 202/`transcription_complete`. (Non-fatal translate FAILURE lands in the same place: zh path stays empty, deliberate swallow per 9R-16 AC 6c.)
3. **Zero migration needed.** `018_add_subtitle_fields.go:18` — `subtitle_status TEXT DEFAULT 'not_searched'`, **no CHECK constraint**. The 10th value is Go-constants + consumers only.
4. **The enum is a stamped wire contract.** `models/movie.go:53` carries `[@contract-v1] (story sub-1-2 AC #2)`; its doc comment says "Adding or renaming a value is a Rule 20 bump plus a downstream stale-mark". All four v1 consumers (sub-1-4/1-5/1-6/1-7b) are **done** → frozen, no stale-marks owed; sub-2-2b is drafted acking **v2** directly.
5. **Alexyu ruling (A+續跑):** resume goes INTO this story; B (one-click translate endpoint) stays a deferred backlog (`backlog-translate-existing-subtitle-endpoint`).

---

## Acceptance Criteria

### AC #1 — `[@contract-v1→v2]` `SubtitleStatus` gains the 10th terminal value `untranslated`

**Given** `models/movie.go`, **then** `SubtitleStatusUntranslated SubtitleStatus = "untranslated"` is added to the pipeline-flavoured block with a doc comment stating the semantics: **TERMINAL. A generated subtitle exists but the expected translation step did not run** (key unconfigured, or a non-fatal translate failure) — written ONLY by the generation pipeline, never inferred from embedded tracks. Recoverable by configuring a key and re-running (which resumes translate-only, AC #3).

- `AllSubtitleStatuses()` gains the entry (the existing sync-guard test enforces this).
- `IsTerminal()` returns **true** for it.
- The `[@contract-v1]` stamp on the type doc BUMPS to `[@contract-v2]` with a Change Log row `[@contract-v1→v2] AC #2 (sub-1-2): 9→10 values, +untranslated; downstream badge renderers must handle it or items park badge-less`.
- **Rule 20 bump-side obligation (MANDATORY):** run the downstream grep (`confirmed against \`?\[@contract-v1\]` × sub-1-2 AC #2) — expected result: all hits are done stories (frozen); record `no not-done consumers` in the Change Log row if so. sub-2-2b acks v2 by construction.

### AC #2 — en-only completion writes `untranslated` back

**Given** a run where **translate was requested** (`cfg.translate == true`) but `zhSRTPath == ""` (key unconfigured — the silent skip — OR a non-fatal translate failure), **then** `translateAndPersist` writes back via the SAME `SubtitleStatusWriter`:

```go
UpdateSubtitleStatus(ctx, mediaID, models.SubtitleStatusUntranslated, srtPath /* the EN SRT */, "en", 0)
```

- The zh-Hant success path (`found`/`zh-Hant`) is **untouched**.
- A writeback failure propagates (Rule 13), exactly like the existing `found` writeback — reporting success while the row still lies would recreate the bug one layer up.
- `translate == false` runs keep today's behaviour (no writeback): no translation was **expected**, so `untranslated` would be a lie. (Today the dialog always sends `translate=true`; this branch is theoretical but must not regress.)
- `ai.ErrBudgetExceeded` mid-translate still propagates (9R-16 AC 6c). **Amended by CR M1 (2026-08-05):** `untranslated` IS recorded (best-effort, log-only on failure) before the sentinel propagates — the en SRT exists and translation is missing, which is factually true mid-pause, and it makes the post-pause resume translate-only instead of re-paying the whole ASR (the story's headline concern). The original "not even untranslated" derivation was dev-authored, not party-ruled.

### AC #3 — A+續跑: resume skips ASR when the English SRT survives

**Given** `StartTranscription` on a movie whose row reads `subtitle_status == untranslated` AND whose recorded `subtitle_path` (the EN SRT from AC #2) exists on disk AND `translate` is requested, **then** the pipeline **skips extract + Whisper entirely**: read the SRT from disk → enter the translate phase directly → zh-Hant place → `found` writeback. SSE starts at the `translating` phase.

- ⚠️ **The resume condition MUST be bound to `subtitle_status == untranslated`** — NEVER bare "an `.srt` exists on disk". A user-placed subtitle must not be mistaken for a resume point (Winston guard, party-mode ruling).
- File missing/moved/unreadable → fall back to a full run (fail-soft; log at Info).
- The service needs a READ of the media row it currently cannot do: extend the narrow Rule 11 interface (e.g. a `SubtitleStatusReader` alongside `SubtitleStatusWriter`, both satisfied by `*repository.MovieRepository`) — do NOT widen the service to a full repository dependency.
- Movies-only, matching the transcription surface (series CTA is disabled — capability honor; series resume arrives with 9R-10a's surface, not here).
- `ai.ErrBudgetExceeded` from the resume path propagates identically to AC #2.

### AC #4 — Tests (Rule 9/16)

1. Enum: `untranslated` in `AllSubtitleStatuses()`, `IsValid()` true, `IsTerminal()` true (the existing sync-guard test extended).
2. Writeback: translate-requested + key-unconfigured run ⇒ `UpdateSubtitleStatus` called with (`untranslated`, en path, `"en"`); zh-success run unchanged (`found`/`zh-Hant`); translate-not-requested run ⇒ NO writeback; writeback error propagates.
3. **Idempotency (Murat, MANDATORY):** second `StartTranscription` on an `untranslated` row with the SRT present ⇒ ASR/extract are NOT invoked (assert via the audio-extractor/ASR mocks), translate IS ⇒ `found`. **The money-burning path must never regress.**
4. Resume fail-soft: `untranslated` row + SRT deleted from disk ⇒ full pipeline runs.
5. Resume guard: `not_searched` row + a stray on-disk `.srt` ⇒ NO resume (full run) — the Winston-guard test.
6. Budget sentinel: `ErrBudgetExceeded` from the resume-path translate propagates; row does NOT flip to `found` (per CR M1 it re-records `untranslated`, idempotently).
7. `pnpm nx test api` green (full regression gate) + `go vet` + `staticcheck` + `gofmt` clean on touched files.

### AC #5 — Scope fence

- ❌ Zero `apps/web/**` files — badge/dialog rendering is sub-2-2b's.
- ❌ No new endpoint — B (`backlog-translate-existing-subtitle-endpoint`) stays deferred by ruling.
- ❌ No SSE **stage-enum** change — D6 (`subtitle/engine.go` PipelineStage) and D2 (this enum) share vocabulary but are DISTINCT stamped contracts; do NOT "deduplicate" (project-context.md sub-1-3 entry). The transcription SSE events (`transcription_*`) are a third, unstamped surface — reuse its existing phases.
- ❌ No series transcription, no batch semantics changes beyond the sentinel behaviour already ruled in 9R-16.
- ❌ No migration (Finding 3).

---

## Tasks / Subtasks

- [x] **Task 1 — Enum + bump (AC #1):** constant, `AllSubtitleStatuses()`, `IsTerminal()`, doc comment, `[@contract-v1→v2]` stamp + Change Log row + downstream grep record.
- [x] **Task 2 — en-only writeback (AC #2):** `translateAndPersist` writes `untranslated`+en-path+`"en"` when translate expected but zh empty; error propagation; translate=false untouched.
- [x] **Task 3 — Resume (AC #3):** `SubtitleStatusReader` narrow interface + main.go wiring; status-bound skip of extract+ASR; fail-soft on missing file; SSE from `translating`.
- [x] **Task 4 — Tests + gates (AC #4):** the seven test groups; foreground; full `go test ./...`.

---

## Dev Notes

- **Rule 20** — this story BUMPS sub-1-2 AC #2 (`[@contract-v1→v2]`). Producer-side stale-mark grep is MANDATORY at implementation (AC #1). It consumes no stamped upstream AC itself.
- **Rule 11** — `SubtitleStatusWriter` (`transcription_service.go:23-29`) is deliberately narrow; mirror it for the reader. `*repository.MovieRepository` satisfies both (`movie_repository.go:831` for the writer).
- **Rule 13** — the AC #2 writeback failure propagates; the AC #3 file-missing fallback is a logged degrade, not an error.
- **Sally's semantics fence (for the doc comment):** `untranslated` names the MISSING STEP, not the artifact — it must never be inferred from embedded tracks (a foreign film with an embedded English track was never owed a translation).
- **Precedents to mirror:** 9R-16 AC 12 (the `found` writeback shape + Rule 13 stance), 9R-16 AC 6c (budget sentinel), sub-1-2 (enum extension mechanics + sync-guard test).
- **Feedback rules in play:** foreground tests only; `pnpm run format:check` before commit; no background vitest.

### Time-dependent visual coverage

**N/A — backend-only.** No wall-clock-reading components touched; Rule 23 does not apply.

### References

- [party-mode ruling: sprint-status `backlog-f5-not-configured-panel-copy-ruling` (superseded entry, full record)]
- [`apps/api/internal/services/transcription_service.go`:416 (silent skip) · :448 (found-only writeback) · :23-29 (writer interface)]
- [`apps/api/internal/models/movie.go`:51-96 (stamped enum) · `apps/api/internal/database/migrations/018_add_subtitle_fields.go`:18 (no CHECK)]
- [`sub-1-2-pipeline-state-model.md`#AC #2 · `project-context.md`#Rule 11/13/20]

---

## Dev Agent Record

### Agent Model Used

Amelia (Developer Agent) — Claude Opus 5 (1M context), effort high. Implemented 2026-08-05.

### Debug Log References

- Red-green honoured per task: Task 1's guard-test expectation (10 values) failed compilation before the constant existed; Task 2/3's new tests failed `go vet` on the old `translateAndPersist` arity before the implementation landed.
- `staticcheck` invoked via `go run honnef.co/go/tools/cmd/staticcheck@2025.1` (not installed locally).

### Completion Notes List

**🔗 AC Drift: FOUND — see below** (checked: `grep -n "en-only|write nothing|zh-Hant place|AC 12" _bmad-output/implementation-artifacts/9R-16*.md` + the stamped enum grep — one genuine DRIFT beyond the one this story planned):

1. **sub-1-2 AC #2 `[@contract-v1→v2]`** (planned, AC #1): the `SubtitleStatus` enum 9→10. Stamp bumped at its declaration (`models/movie.go` doc comment). Downstream grep (`confirmed against \`?\[@contract-v1\]` × sub-1-2): sub-1-5b / sub-1-6 / sub-1-7b — **all done → frozen, no stale-marks owed**; sub-2-2b was drafted acking v2 by construction.
2. **9R-16 AC #12 `[@contract-v1→v2]`** (DISCOVERED at Step 2): AC 12's v1 text pins "en-only runs … do NOT write; failures write nothing" and had a test enforcing it (`TestTranslateAndPersist_EnOnlyNoWrite` / `_TranslateFailureNoWrite`). AC #2 of this story changes exactly that observable contract → bump recorded here and at the drifted test (`_TranslateFailureWritesUntranslated` carries the `[@contract-v2]` comment). The `translate` ABSENT no-write clause is **kept** (no expectation → no lie). Downstream grep for 9R-16 AC 12 ackers: 9R-18 / ux3-subtitle-v2(+batch) / ux3-ai-1 / ux3-ai-2 — **all done → frozen**.

**📎 Contract Stamps: FOUND (2 stamped upstream ACs, both BUMPED v1→v2 by this story — see AC Drift; this story stamps nothing new).** Change Log rows carry the mandatory {what changed, what breaks downstream} bodies.

**🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched).**

**🎨 UX Verification: SKIPPED — no UI changes in this story.**

**Pre-existing fix (Epic 9c Retro AI-2 option 1):** `cmd/api/main.go` carried a duplicate `internal/database/migrations` import (named at :19 + blank at :40) — pre-existing **ST1019**, verified present on clean main via `git stash`. Blank import removed (the named import already triggers `init()` registration); gofmt applied. staticcheck now clean on all touched packages.

**What shipped**

- **AC #1** — `SubtitleStatusUntranslated = "untranslated"`: 10th value, TERMINAL, doc comment carries Sally's semantics fence (names the missing step; written only by the generation pipeline; never inferred from embedded tracks). `AllSubtitleStatuses()` + `IsTerminal()` extended; the source-regex guard test enforces sync. Zero migration (018 has no CHECK).
- **AC #2** — `translateAndPersist` gained `srtPath` and writes the VERDICT: zh success → `found`/`zh-Hant` (untouched); translate expected + zh empty (key unconfigured OR non-fatal translate failure) → `untranslated`/en-path/`"en"`; `translate=false` → nothing. Writeback failures propagate (Rule 13); `ErrBudgetExceeded` propagates BEFORE any writeback — a paused batch item is not terminal.
- **AC #3** — `tryTranslateOnlyResume` at the top of `runPipeline`: bound to `subtitle_status == untranslated` + recorded `SubtitlePath` readable on disk + translate requested → skip extract+ASR, reuse the on-disk EN SRT, SSE starts at `translating` (the broadcast already inside `translateAndPersist`). Any failure mode degrades to a full run (logged, never an error). New narrow `SubtitleStateReader` (Rule 11, mirrors `SubtitleStatusWriter`; `*MovieRepository.FindByID` satisfies it); wired in `main.go` beside the writer. Movies-only, matching the transcription surface.
- **AC #4** — 10 new/updated tests: enum sync/valid/terminal; key-unconfigured writes untranslated; translate-failure writes untranslated (the drifted test, re-commented); en-only still writes nothing; budget-exceeded writes nothing (not even untranslated); untranslated-writeback failure propagates; **idempotency** (resume completes → extract/ASR provably never invoked — they would error on the nonexistent media file — and the row flips `found`); Winston guard (stray on-disk `.srt` + `not_searched` row → full run attempted, never laundered to `found`); fail-soft (SRT deleted → full run attempted); resume-path budget sentinel; nil-reader safety.
- **Gates** — `go build` ✓, `go vet` ✓, `staticcheck` clean on touched packages, `gofmt` clean on touched files, **full `go test ./...` 34 packages green**, `pnpm nx test web` 228 files / 2528 tests green (regression gate), no orphaned test processes.

**Test-evidence note (the idempotency proxy):** the resume test constructs a service whose extractor probes a NONEXISTENT media file and whose Whisper key is bogus — any invocation of extract/ASR fails the pipeline, so a successful completion PROVES non-invocation. Recorded because the extractor is a concrete `*AudioExtractorService` (not an interface), making call-count mocks unavailable without a refactor this story does not own.

### Senior Developer Review (AI) — 2026-08-05

**Outcome: Approve (all findings fixed same session).** Adversarial CR (Amelia-as-reviewer, Fable 5): **0 High / 2 Medium / 2 Low — all 4 fixed.** Git-vs-File-List: 0 discrepancies. Rule 7: PASS (touched Go files carry no error-code constants; zero new codes). Rule 20: PASS (both v1→v2 bumps re-verified — every downstream acker `done`/frozen; sub-2-2b acks v2). Rule 25: N/A. Enumeration interplay adversarially checked and CLEAN: batch missing-scope (`subtitle_language != 'zh-Hant'`) keeps `untranslated` rows in scope so batch re-runs become translate-only; the online-search batch (`not_searched|not_found`) correctly excludes them; the D2 pipeline enqueues only via explicit FR12 requests.

- ✅ **[M1] Budget pause still burned the paid ASR** — full run places the en SRT, translate hits the budget, sentinel propagated BEFORE any writeback → row stayed `not_searched` → post-pause resume re-ran the whole ASR: the story's headline bug with a different trigger. The AC's "not even untranslated" clause was a dev derivation, not a party ruling. Fix: record `untranslated` (best-effort, log-only on failure — the sentinel must survive) before propagating. AC #2/#4 amended with the CR annotation; 2 tests updated + 1 resume-path variant added.
- ✅ **[M2] Resume over-gated by `IsAvailable()`** — handler + both service entries required FFmpeg+ASR even for a translate-only resume that needs neither (operator removed the ASR key after generating → 503 despite zero ASR work). Fix: `resumeSource` split out (stat-only eligibility), new `CanResumeTranslateOnly(ctx, id)` on the service + handler interface, all three gates relaxed to `IsAvailable() || resume-eligible`, and a defensive `IsAvailable` guard added at the top of the full-run branch so a gate-passed run whose SRT vanishes mid-flight refuses cleanly instead of dereferencing a nil extractor. +2 tests (ASR-less resume succeeds; ASR-less non-resumable still 503s at entry).
- ✅ **[L1] Resume completion SSE claimed "Transcription complete"** — a resumed run transcribed nothing; message now "Translation complete (resumed from existing English subtitle)".
- ✅ **[L2] Async entry resume had no direct coverage** — `TestStartTranscription_ResumeSmoke` added (goroutine completes translate-only, row flips `found`); `fakeSubtitleWriter` gained a mutex + `snapshot()` for cross-goroutine assertions.

Gates re-run post-fix: `go build` ✓, `go vet` ✓, staticcheck clean (services+handlers), gofmt clean on touched files (`transcription_translation_test.go` drift verified PRE-existing via git stash — left per the in-scope-only rule), **full `go test ./...` 34 packages green**.

### Discovery Triage

- **Did this story discover any work outside its current scope?** One item, plus a quick fix absorbed:
  - **① absorbed in place:** the ST1019 duplicate-import quick fix (recorded above under Pre-existing fix; behaviour-neutral).
  - **① clarified in place:** the 9R-16 AC #12 bump was not pre-planned by this story's AC #1 but is the same change viewed from the consumer side — absorbed into the existing Rule 20 task rather than spawning an entry (both bumps verified frozen-downstream).
  - Otherwise: `N/A — no further out-of-scope work discovered`.
- Reference: `project-context.md` Rule 24.

### File List

**Modified**

- `apps/api/internal/models/movie.go` (10th enum value + `[@contract-v2]` stamp + `AllSubtitleStatuses` + `IsTerminal`)
- `apps/api/internal/models/movie_test.go` (guard/valid/terminal tests → 10 values)
- `apps/api/internal/services/transcription_service.go` (`SubtitleStateReader` + setter; `tryTranslateOnlyResume`; `translateAndPersist` srtPath + verdict writeback; runPipeline resume branch)
- `apps/api/internal/services/transcription_generation_test.go` (10 new/updated tests + `fakeStateReader`/`resumeService` harness)
- `apps/api/cmd/api/main.go` (reader wiring + pre-existing ST1019 duplicate-import fix)
- `apps/api/internal/handlers/transcription_handler.go` (CR M2 — resume-aware availability gate + interface method)
- `apps/api/internal/handlers/transcription_handler_test.go` (CR M2 — mock method)
- `_bmad-output/implementation-artifacts/9R-16-batch-generation-endpoint.md` (AC #12 stamp bump annotation — AC drift reference, see Completion Notes)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (status transitions)
- `_bmad-output/implementation-artifacts/sub-2-2a-untranslated-status-backend.md` (this file)

## Change Log

| Date | Change |
|---|---|
| 2026-08-05 | Task 1 — `untranslated` 10th `SubtitleStatus` value (TERMINAL) + guard/valid/terminal tests. `[@contract-v1→v2]` sub-1-2 AC #2: 9→10 values, +untranslated; downstream badge renderers must handle the new value or items park badge-less. Downstream grep: sub-1-5b/sub-1-6/sub-1-7b all done (frozen); no not-done v1 consumers. |
| 2026-08-05 | Task 2 — en-only writeback: translate-expected + zh empty ⇒ `untranslated`/en-path/`"en"`; zh path untouched; `translate=false` writes nothing; budget sentinel propagates pre-writeback. `[@contract-v1→v2]` 9R-16 AC #12: "en-only/failed runs write nothing" → "translation-expected runs always record a verdict (found or untranslated)"; downstream consumers relying on no-write-on-failure would now see `untranslated` rows — grep: 9R-18/ux3-subtitle-v2(+batch)/ux3-ai-1/ux3-ai-2 all done (frozen); no not-done v1 consumers. |
| 2026-08-05 | Task 3 — `tryTranslateOnlyResume`: status-bound (Winston guard) translate-only resume skipping extract+ASR, fail-soft to full run; `SubtitleStateReader` narrow interface wired in main.go. |
| 2026-08-05 | Adversarial CR: 2M/2L found, all fixed — untranslated recorded before the budget sentinel propagates (M1, AC #2/#4 amended with annotation), resume-aware availability gates via CanResumeTranslateOnly on service + handler (M2, +nil-extractor fallback guard), resume-aware completion message (L1), async resume smoke test + mutexed fake writer (L2). Full go test ./... 34 pkgs green. Story → done. |
| 2026-08-05 | Task 4 — 10 tests incl. idempotency (non-invocation proxy), stray-`.srt` guard, resume-path budget sentinel, nil-reader. Full `go test ./...` 34 pkgs green; web 228/2528 green; staticcheck/gofmt/vet clean. Pre-existing ST1019 (duplicate migrations import in main.go) fixed in place. |
