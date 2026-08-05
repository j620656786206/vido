# Story sub-2.2a: `untranslated` terminal status + en-only writeback + translate-only resume

Status: ready-for-dev

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
- `ai.ErrBudgetExceeded` mid-translate still propagates BEFORE any writeback (9R-16 AC 6c — a paused batch item is not terminal).

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
6. Budget sentinel: `ErrBudgetExceeded` from the resume-path translate propagates; row does NOT flip to `found`.
7. `pnpm nx test api` green (full regression gate) + `go vet` + `staticcheck` + `gofmt` clean on touched files.

### AC #5 — Scope fence

- ❌ Zero `apps/web/**` files — badge/dialog rendering is sub-2-2b's.
- ❌ No new endpoint — B (`backlog-translate-existing-subtitle-endpoint`) stays deferred by ruling.
- ❌ No SSE **stage-enum** change — D6 (`subtitle/engine.go` PipelineStage) and D2 (this enum) share vocabulary but are DISTINCT stamped contracts; do NOT "deduplicate" (project-context.md sub-1-3 entry). The transcription SSE events (`transcription_*`) are a third, unstamped surface — reuse its existing phases.
- ❌ No series transcription, no batch semantics changes beyond the sentinel behaviour already ruled in 9R-16.
- ❌ No migration (Finding 3).

---

## Tasks / Subtasks

- [ ] **Task 1 — Enum + bump (AC #1):** constant, `AllSubtitleStatuses()`, `IsTerminal()`, doc comment, `[@contract-v1→v2]` stamp + Change Log row + downstream grep record.
- [ ] **Task 2 — en-only writeback (AC #2):** `translateAndPersist` writes `untranslated`+en-path+`"en"` when translate expected but zh empty; error propagation; translate=false untouched.
- [ ] **Task 3 — Resume (AC #3):** `SubtitleStatusReader` narrow interface + main.go wiring; status-bound skip of extract+ASR; fail-soft on missing file; SSE from `translating`.
- [ ] **Task 4 — Tests + gates (AC #4):** the seven test groups; foreground; full `go test ./...`.

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

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - If **NO**: state `N/A — no out-of-scope work discovered`.
- Reference: `project-context.md` Rule 24.

### File List
