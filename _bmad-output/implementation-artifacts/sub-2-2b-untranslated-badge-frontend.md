# Story sub-2.2b: 「未翻譯」 badge + derivation-ladder fix + honest completion message

Status: ready-for-dev

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

- [ ] **Task 1 — Ladder fix (AC #1):** `deriveSubtitleStatus` step-1 fallback no longer inherits 缺字幕 from empty tracks.
- [ ] **Task 2 — Badge (AC #2):** `untranslated` case in the step-2 switch + doc comment; scope-fence comment.
- [ ] **Task 3 — Completion message (AC #3):** conditional on `zhSrtPath`.
- [ ] **Task 4 — CTA helper (AC #4, trails γ):** conditional helper text + `/settings/keys` affordance via `useKeySettings`.
- [ ] **Task 5 — Tests + gates (AC #5):** Murat's four + fence + dialog + CTA groups; foreground vitest; `pnpm lint:all`.

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

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - If **NO**: state `N/A — no out-of-scope work discovered`.
- Reference: `project-context.md` Rule 24.

### File List
