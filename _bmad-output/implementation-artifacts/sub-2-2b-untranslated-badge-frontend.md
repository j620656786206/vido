# Story sub-2.2b: 「未翻譯」 badge + derivation-ladder fix + honest completion message

Status: done

**Epic:** `epic-subtitle-pipeline-m1-5` (M1.5 follow-up) · **Risk: 🟡 MEDIUM** · **FRONTEND-ONLY**
**Source:** party-mode ruling 2026-08-05 (Sally 主裁) · promotes `backlog-subtitle-untranslated-badge-frontend`
**Split:** β of the α/β/γ split. Frontend tasks 4 / backend tasks **0** → single story.
**Depends on:** **sub-2-2a merged** (the `untranslated` enum value — hard). **sub-2-2c** (γ's ratified CTA copy — soft: Task 4 trails γ if its copy is not ratified yet; Tasks 1–3 do not wait).
**Blocks:** nothing.

---

## Story

As a NAS owner whose generation produced an English-only subtitle,
I want the library badge and the dialog to SAY so — 「未翻譯」, not 缺字幕 or a false 完成 —
so that I know exactly what I have and what step (setting the translation key) gets me the rest.

---

## 🔎 Findings (verified 2026-08-05, party mode)

1. **The derivation-ladder hole (pre-existing, zero coverage).** `libraryStatus.ts:115` — step 1 (`subtitleStatus === 'found'`) ends with `deriveFromTracks(media) ?? { label: '有字幕' }`, but `deriveFromTracks` returns `{label: '缺字幕'}` — **not `null`** — on an empty track array (`:95`). So `found` + non-zh `subtitleLanguage` + no embedded tracks renders **缺字幕 over an authoritative engine verdict**. Once α writes `found`/`untranslated` with en paths onto files with no embedded tracks, this hole fires constantly. The authoritative source must never lose to a naive track count.
2. **The dialog lies on en-only completion.** `ManageSubtitleDialogV2.tsx:303` renders 「字幕已生成完成」 unconditionally on `phase === 'complete'` — but `useGenerationProgress.ts:214` already carries `zhSrtPath` (null on en-only runs; backend omits `zh_srt_path`, `transcription_service.go:400`). The honest message is one conditional away.
3. **Sally's badge ruling:** label 「未翻譯」 (3 CJK chars, within the 1-7a 3–4 limit) · **neutral tint** (`TINT.neutral` — accent reserved for in-progress per Sally gate 2026-07-05; this is not an error) · names the MISSING STEP (the user's next action), the deliberate mirror of 1-7a's "distinct by recovery, not by meaning" (無字幕源 vs 缺字幕) · pairs naturally with in-flight 翻譯中.
4. **Scope fence (Sally):** `untranslated` is written ONLY by the generation pipeline — **an embedded English track on a foreign film must NOT badge 未翻譯** (painting a normal state as a defect). The badge derives from `subtitleStatus` exclusively, never from tracks.

---

## Acceptance Criteria

### AC #1 — Ladder fix: authoritative verdict never loses to track inference

**Given** `deriveSubtitleStatus` step 1 (`found`), **then** a non-zh `subtitleLanguage` with **no embedded tracks** renders 「有字幕」 (neutral), never 缺字幕. Track-derived zh classification (繁中/簡中) may still refine a `found` verdict; the 缺字幕-from-empty-tracks fallback may not override it.

### AC #2 — `untranslated` → 「未翻譯」 badge

**Given** `subtitleStatus === 'untranslated'`, **then** the badge renders `{ label: '未翻譯', className: TINT.neutral }` — an exception (non-steady) state, so `pickPosterBadge` shows it on the grid.

- Placed in the step-2 pipeline-verdict switch (above track inference, same reasoning as `no_text_source`/`skipped` — the file HAS a track/SRT the verdict already accounts for).
- The doc comment records the recovery: set the key, re-run 生成字幕 — α's resume makes that translate-only.
- **Never inferred from tracks** (Finding 4).

### AC #3 — Honest completion message

**Given** `phase === 'complete'` with `zhSrtPath === null`, **then** `generation-complete-note` renders 「已生成英文字幕；尚未翻譯」 (with the settings affordance per γ's ratified copy if available); with `zhSrtPath` non-null it stays 「字幕已生成完成」.

### AC #4 — Pre-flight CTA helper text (trails γ)

**Given** γ's ratified copy, **then** the 生成字幕 helper line (`ManageSubtitleDialogV2.tsx:425` 「轉錄＋AI 翻譯，約需數分鐘」) becomes conditional: translation key unconfigured ⇒ 「僅能產生英文字幕——尚未設定翻譯金鑰」 + a `/settings/keys` affordance. Signal = `useKeySettings()` GET `claude.configured` (Winston: good-enough proxy for M1.5, no new capability endpoint). The user can still click — degraded ≠ blocked.

- **If γ's copy is not ratified when Tasks 1–3 are done:** ship Tasks 1–3, mark this task deferred-to-γ explicitly in Completion Notes (three-state, not silence), and file nothing new — γ already tracks it.

### AC #5 — Tests (Rule 9/16 + Murat's mandate)

1. **The four empty-tracks + authoritative-verdict tests (Murat, MANDATORY — this class has ZERO coverage today):** empty `subtitleTracks` array with each of `found`+`subtitleLanguage:'en'` ⇒ 有字幕 · `untranslated` ⇒ 未翻譯 · `no_text_source` ⇒ 無字幕源 · `skipped` ⇒ 已略過. None may render 缺字幕.
2. Scope fence: embedded-English-track file with `subtitleStatus` absent/`not_searched` ⇒ 有字幕 (track inference), NEVER 未翻譯.
3. `untranslated` is non-steady ⇒ `pickPosterBadge` surfaces it; `found`+zh-Hant steady behaviour unchanged.
4. Dialog: `zhSrtPath === null` complete ⇒ 「已生成英文字幕；尚未翻譯」; non-null ⇒ 「字幕已生成完成」; existing dialog tests stay green.
5. (With AC #4) key-unconfigured ⇒ degraded helper text + affordance; configured ⇒ today's line; the CTA stays enabled in both.
6. `pnpm nx test web` green (**foreground**) + `pnpm lint:all` green + a11y pre-flight on touched components.

### AC #6 — Scope fence

- ❌ Zero `apps/api/**` files. The enum and writeback are α's.
- ❌ No `.pen` edits, no screenshot regeneration — γ owns the design surface (j2-d 10th row, F1 CTA states, F5 purify).
- ❌ No library filter for `untranslated` (`disc-2026-06-library-subtitle-status-filter` stays its own backlog).
- ❌ No visual baselines (settings/* and badge fixtures — same Rule 22 boundary as 1-7b AC #5 unless gallery fixtures already cover the badge component; extend the EXISTING badge fixture only if 1-7b created one).

---

## Tasks / Subtasks

- [x] **Task 1 — Ladder fix (AC #1):** `deriveSubtitleStatus` step-1 fallback no longer inherits 缺字幕 from empty tracks.
- [x] **Task 2 — Badge (AC #2):** `untranslated` case in the step-2 switch + doc comment; scope-fence comment.
- [x] **Task 3 — Completion message (AC #3):** conditional on `zhSrtPath`.
- [x] **Task 4 — CTA helper (AC #4, trails γ):** executed via AC #4's explicit DEFERRAL branch — γ (sub-2-2c) has not ratified the copy, so Tasks 1–3 ship and this task's outcome is the recorded deferral (no code, nothing new filed — γ already tracks the CTA copy + affordance). See Completion Notes.
- [x] **Task 5 — Tests + gates (AC #5):** Murat's four + fence + dialog + CTA groups; foreground vitest; `pnpm lint:all`.

---

## Dev Notes

- **Rule 20** — consumes sub-1-2 AC #2 at **`[@contract-v2]`** (the version α lands — record `confirmed against [@contract-v2] sub-1-2 AC #2 (as bumped by sub-2-2a)` at implementation) and sub-2-1a `[@contract-v1]` AC #3 indirectly via `useKeySettings` (already acked by sub-2-1b; record reuse, no new ack surface). Stamps nothing.
- **Rule 5** — `useKeySettings` is the existing TanStack Query hook (sub-2-1b); do not hand-roll a fetch for AC #4.
- **`HANT`/`HANS` stay the single source** (`libraryStatus.ts:73-74`) — the ladder fix must not fork classification.
- **1-7a's spec screen (`flow-j-specs/j2-d`) is the badge table's home** — γ adds the 10th row there; this story links it in the component doc comment once γ lands (soft reference, not a blocker).
- **Feedback rules in play:** foreground tests; `format:check` before commit; design verification against γ's surface for Task 4.

### Time-dependent visual coverage

**N/A — no wall-clock-reading components touched.** `libraryStatus.ts` and `ManageSubtitleDialogV2.tsx` read no `Date.now()`/`new Date()`; Rule 23 does not apply.

### References

- [party-mode ruling: sprint-status `backlog-f5-not-configured-panel-copy-ruling` (superseded entry)]
- [`apps/web/src/utils/libraryStatus.ts`:95 (缺字幕-on-empty) · :115 (the hole) · :133-145 (the step-2 switch to extend)]
- [`apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx`:298-305 (completion note) · :425 (CTA helper line)]
- [`apps/web/src/hooks/useGenerationProgress.ts`:214 (`zhSrtPath`) · `apps/web/src/hooks/useKeySettings.ts` (pre-flight signal)]
- [`sub-2-2a-untranslated-status-backend.md`#AC #1 · `sub-2-2c-f5-asr-copy-design.md` · `project-context.md`#Rule 5/20/22]

---

## Dev Agent Record

### Agent Model Used

Amelia (Developer Agent) — Claude Opus 5 (1M context), effort high. Implemented 2026-08-05.

### Debug Log References

- Red-green honoured: the 11 new spec cases were written first and failed 5 (the ladder hole + all `untranslated` renderings) before the implementation landed.
- UX verification screenshots (git-ignored `tmp/badge-untranslated.png`, not committed): live `/library` grid against `nx serve web` with the list API stubbed via Playwright `route.fulfill` — five cards covering 未翻譯 / 有字幕(ladder-hole item) / 無字幕源 / 已略過 / 缺字幕. Dev server stopped afterwards.

### Completion Notes List

**🔗 AC Drift: NONE** (checked: `grep -n "found|有字幕|缺字幕" apps/web/src/utils/libraryStatus.spec.ts` + sub-1-7b story ACs — no prior test or AC pins the `found`+non-zh+empty-tracks path (that zero coverage IS the finding this story fixes); sub-1-7b's pinned behaviours — terminal verdicts above track inference, transients → null, ordering-regression pair — are all preserved and still green. The dialog completion note had no prior text assertion.)

**📎 Contract Stamps: FOUND (1 upstream ack, no bumps by this story):** confirmed against [@contract-v2] sub-1-2 AC #2 (as bumped by sub-2-2a — the 10-value `SubtitleStatus` enum; this story renders the 10th value). The `useKeySettings`/sub-2-1a AC #3 consumption did NOT materialize (Task 4 deferred), so no new ack surface. This story stamps nothing.

**🎭 A11y Pre-Flight: PASS** (2 components/files checked — `ManageSubtitleDialogV2.tsx`, `libraryStatus.ts`; 0 jsx-a11y warnings on touched files, 0 introduced. The change is a text-conditional and a pure derivation function — no new interactive surface, no modal/image/live-region changes. Lazy-load contract N/A.)

**⏸ Task 4 (AC #4 CTA helper): DEFERRED-TO-γ — the AC's explicit branch, not a silent skip.** γ (sub-2-2c, ready-for-dev) has not run, so no ratified copy exists. Per AC #4: Tasks 1–3 ship, the deferral is recorded here, and NOTHING new is filed — sub-2-2c already tracks the CTA copy + affordance, and its implementation lands with/after γ (a one-conditional change against `useKeySettings().data.keys` claude row).

**What shipped**

- **AC #1 (the ladder hole)** — `deriveSubtitleStatus` step 1: embedded tracks may REFINE a `found` verdict (zh upgrade preserved, asserted) but may never CONTRADICT it — the `deriveFromTracks ?? 有字幕` fallback no longer lets 缺字幕-on-empty-array override the authoritative engine result. Critical since sub-2-2a: external SRTs are never in `subtitleTracks`, so every generation-found row with no embedded tracks hit this hole.
- **AC #2** — `untranslated` → 「未翻譯」 `TINT.neutral`, placed in the step-2 pipeline-verdict switch (above track inference, same reasoning as its siblings), non-steady so `pickPosterBadge` surfaces it on the grid; doc comment carries the distinct-by-recovery semantics + the generation-only fence.
- **AC #3** — completion note conditional on `zhSrtPath`: 「已生成英文字幕；尚未翻譯」 when null, 「字幕已生成完成」 otherwise.
- **AC #5** — 13 new tests: Murat's four empty-tracks×authoritative-verdict cases (`found`+en / `untranslated` / `no_text_source` / `skipped`), zh-refinement survival, neutral-tint + accent-reserved assertions, untranslated-beats-en-track, the SCOPE FENCE (embedded en track without the verdict → 有字幕, never 未翻譯), `pickPosterBadge` surfacing, and the two dialog completion variants. Type-safety fix en route: the dialog spec's hoisted `srtPath`/`zhSrtPath` were inferred as `null` — annotated `string | null`.
- **AC #6 fence honoured** — 0 `apps/api/**` files, no `.pen` edits, no screenshot regen, no library filter, no visual baselines.

**Gates** — `pnpm nx test web` **228 files / 2539 tests green** (+11), `pnpm nx test api` 34 packages green (regression gate), `pnpm lint:all` 0 errors, `format:check` green, a11y pre-flight PASS, no orphaned workers.

### 🎨 UX Verification: PASS (live-render check; j2-d spec row pending γ by design)

Verified against the party-mode ruled badge table (Sally 主裁 2026-08-05 — label 未翻譯 / neutral tint / generation-only / recovery=set-key-then-rerun) plus a live `/library` render with five mocked cards:

| Area | Ruled spec | Implementation | Match? |
|---|---|---|---|
| Label | 未翻譯 (3 CJK, names the missing step) | 未翻譯 | ✅ |
| Tint | neutral (`--bg-tertiary`/`--text-muted`), NOT error, accent reserved | `TINT.neutral`, asserted in tests + visually identical pill to 無字幕源/已略過 | ✅ |
| Grid visibility | exception → shows on poster | non-steady, `pickPosterBadge` surfaces it (screenshot) | ✅ |
| Scope | generation artifacts only | fence test: embedded en track alone → 有字幕 | ✅ |
| Ladder | authoritative verdict never loses to empty tracks | screenshot card 2: `found`+en+`[]` → 有字幕 | ✅ |
| j2-d spec table row | 10th row | **pending γ (sub-2-2c AC #3)** — by the split's design, not a drift | ⏸ γ |

### Senior Developer Review (AI) — 2026-08-05

**Outcome: Approve (all findings fixed same session).** Adversarial CR (Amelia-as-reviewer, Fable 5): **0 High / 1 Medium / 2 Low — all addressed.** Git-vs-File-List: 0 discrepancies. Rule 7: N/A (pure frontend). Rule 20: N/A (acks only, verified verbatim). Rule 25: N/A. All ACs re-verified, incl. AC #4's deferral branch executed as written.

- ✅ **[M1] `EpisodeList` lacked the 10th value** — `untranslated` fell back to `not_searched` (`Minus`/「尚未搜尋字幕」): a SETTLED verdict wearing a not-started accessible name, the exact skipped-vs-not_searched class Sally ruled on 2026-08-04. Unreachable today (α is movies-only) but 9R-10a (ready-for-dev) makes it live, and that story would not know to add it — α's own bump note mandates renderers handle the value. Fix: map entry per the ratified J2-D icon grammar (CIRCLED settled + muted — nothing broke; poster ruling says neutral-not-error) with label 「已生成英文字幕，尚未翻譯」; **`CircleDashed` glyph marked PROVISIONAL pending γ** and flagged in sub-2-2c's story file. +2 tests (long-form name; glyph ≠ not_searched).
- ✅ **[L1] Ladder guard filtered by label string** — `fromTracks.label !== '缺字幕'` coupled the story's core fix to display copy. Refactored to positive intent: `trackLangs()` extracted (single parse), the `found` branch refines via HANT/HANS classification directly; `deriveFromTracks` rebuilt on the same helper. All 38 ladder tests green unchanged.
- ✅ **[L2] `untranslated` + embedded 繁中 track renders 未翻譯** — verdict-outranks-tracks is consistent with its siblings, but the recovery hint is moot when a 繁中 track exists. No code change (consistency stands); recorded as a Sally decision input in sub-2-2c's "Dev-flagged inputs" block alongside the M1 glyph ratification.

Gates re-run post-fix: `pnpm nx test web` **228 files / 2541 tests green** (+2), `pnpm lint:all` 0 errors, `format:check` green.

### Discovery Triage

- **Did this story discover any work outside its current scope?** `N/A — no out-of-scope work discovered` (the Task 4 deferral is AC #4's own branch and is tracked by sub-2-2c; nothing new filed).
- Reference: `project-context.md` Rule 24.

### File List

**Modified**

- `apps/web/src/utils/libraryStatus.ts` (ladder fix + `untranslated` case + header doc)
- `apps/web/src/utils/libraryStatus.spec.ts` (+11 tests)
- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx` (AC #3 conditional completion note)
- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.spec.tsx` (+2 completion tests; hoisted-state type annotations)
- `apps/web/src/components/media/EpisodeList.tsx` (CR M1 — 10th-value entry, provisional CircleDashed)
- `apps/web/src/components/media/EpisodeList.spec.tsx` (CR M1 — +2 tests)
- `_bmad-output/implementation-artifacts/sub-2-2c-f5-asr-copy-design.md` (CR M1/L2 — Sally decision inputs for the j2-d row)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (status transitions)
- `_bmad-output/implementation-artifacts/sub-2-2b-untranslated-badge-frontend.md` (this file)

## Change Log

| Date | Change |
|---|---|
| 2026-08-05 | Task 1 — ladder fix: embedded tracks refine but never contradict an authoritative `found` (缺字幕-on-empty no longer overrides). Task 2 — `untranslated` → 未翻譯 neutral badge in the step-2 switch, non-steady, generation-only fence. Task 3 — completion note conditional on `zhSrtPath`. RED-first (5 failures before implementation). |
| 2026-08-05 | Adversarial CR: 1M/2L, all addressed — EpisodeList 10th-value entry ahead of 9R-10a (M1, CircleDashed provisional pending γ), trackLangs()-based positive refinement replacing the label-string guard (L1), untranslated+embedded-繁中 semantics routed to γ as a Sally input (L2). Web 228/2541 green. Story → done. |
| 2026-08-05 | Task 4 — executed as AC #4's deferral branch (γ unratified; sub-2-2c tracks the CTA copy). Task 5 — 13 new tests incl. Murat's four zero-coverage cases + scope fence + dialog variants. Gates: web 228/2539 green, api 34 green, lint 0 errors. Rule 20: acks sub-1-2 [@contract-v2]. UX verification via live /library render (five-card screenshot). Story → review. |
