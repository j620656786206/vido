# Story sub-2.2d: F5 code strings + degraded CTA helper + 503 zh-TW envelope

Status: done

**Epic:** `epic-subtitle-pipeline-m1-5` (M1.5 follow-up) · **Risk: 🟠 CROSS-STACK-SMALL (frontend 2 / backend 1 — under the >3-per-side split threshold)**
**Source:** promotes `backlog-f5-cta-code-strings` (filed by sub-2-2c Discovery Triage, 2026-08-06) · implements γ's **ratified string table** (`sub-2-2c-f5-asr-copy-design.md` → Completion Notes — the authoritative handoff; spec PNGs export ≤400px)
**Split check:** frontend tasks 2 / backend tasks 1 → single story.
**Depends on:** **sub-2-2c merged** (#204 — the ratified strings) and sub-2-1b merged (the `/settings/keys` page + `useKeySettings`).
**Blocks:** nothing — this closes the α/β/γ value chain's last gap.

---

## Story

As a NAS owner hitting the key-missing states,
I want the LIVE dialog and API to speak γ's ratified copy — the right capability, the right knob, the right timing truth —
so that the design and the running product say the same sentences.

---

## 🔎 Findings (verified 2026-08-06)

1. **The F5 panel code is two revisions behind.** `ManageSubtitleDialogV2.tsx:369` 「字幕生成尚未設定」 / `:372` 「需要 FFmpeg 與 AI API Key，設定完成後即可轉錄＋翻譯生成字幕」 — pre-γ (and the body still frames FFmpeg as a user setting, which 1-7a already overruled). The button already navigates to `/settings/keys` (sub-2-2b AC #4) — only the words lag.
2. **The 503s are split-brained.** `transcription_handler.go` answers `TRANSCRIPTION_DISABLED` in **English** ("Transcription is not available" / "Ensure FFmpeg is installed and OPENAI_API_KEY is configured.") — a Rule 3 zh-TW envelope gap. `generation_batch_handler.go:71-74` IS zh-TW but keeps the pre-2-1b env framing (「…設定 OPENAI_API_KEY」) with no settings-page mention and no restart truth.
3. **The degraded-CTA signal is already built.** `useKeySettings()` (sub-2-1b) returns the GET `/settings/keys` state; `keys.find(k => k.name === 'claude')?.configured` is γ's ratified pre-flight signal. Rule 5 — no new fetch layer.
4. **The restart truth holds today**: WhisperClient is boot-built (`main.go:505`); γ's F5 body says 「…並重啟伺服器後再試」. If `backlog-asr-runtime-key-resolution` later ships a holder, THAT story owns updating this clause (coupling noted, not owed here).

---

## Acceptance Criteria

### AC #1 — F5 panel speaks the ratified copy

**Given** the 503 `TRANSCRIPTION_DISABLED` panel (`generation-not-configured`), **then**:

- Title (`:369`) → 「**語音辨識尚未設定**」
- Body (`:372`) → 「**生成字幕需要雲端語音辨識（ASR）金鑰。請至金鑰設定儲存，並重啟伺服器後再試。**」
- The 前往設定 button and `data-testid="go-to-settings"` are untouched (already correct).
- The file-header comment's F5 line (`:15`) is synced.

### AC #2 — Degraded CTA helper (β Task 4's deferred implementation)

**Given** the dialog open with `mediaType === 'movie'` and the translation key unconfigured (`useKeySettings` → claude row `configured === false`), **then** the helper line (`:433`) renders 「**僅能產生英文字幕——尚未設定翻譯金鑰**」 plus a 前往設定 text-link affordance → `/settings/keys`; the 生成字幕 CTA stays **enabled** (degraded ≠ blocked — the party-mode asymmetry).

- Key configured → the default line, **verb-corrected per the 2026-08-06 party-mode ruling**: 「**語音辨識＋AI 翻譯，約需數分鐘**」 (was 轉錄＋AI 翻譯 — same vocabulary as F5's 語音辨識尚未設定, honest for the transcribe endpoint this button actually calls). Query loading/error → the same default line (**fail-soft; never flash the degraded warning on an unresolved query**).
- Series branch (影集字幕生成即將推出) untouched.
- Rule 5: `useKeySettings` only; gate the fetch on dialog `open` if a mount-time fetch would fire for every card (mirror `useGlossaryTerms(mediaId, open)`).
- Completion lines: **no change** — γ ratified β's shipped pair as-is.

### AC #3 — 503 envelopes speak zh-TW with the ratified framing

1. `transcription_handler.go` `TRANSCRIPTION_DISABLED` → message 「語音辨識尚未設定」, suggestion 「生成字幕需要雲端語音辨識（ASR）金鑰。請至金鑰設定（`/settings/keys`）儲存雲端 ASR 金鑰，並重啟伺服器。」 (Rule 7 code unchanged — copy only.)
2. `generation_batch_handler.go:71-74` suggestion aligned to the same framing (settings-page first, restart truth; drop the bare env-var instruction). Message 「字幕生成功能未啟用」 may stay — it is batch-scoped and zh-TW already.

### AC #4 — Tests (Rule 9/16)

1. Dialog spec: the existing F5 assertions updated to the ratified strings (they MUST fail before the code change — the drift test).
2. CTA helper: unconfigured ⇒ degraded line + link (CTA `toBeEnabled()`); configured ⇒ default line; loading ⇒ default line (the no-flash fence).
3. Handler tests: both 503 bodies assert the zh-TW message/suggestion.
4. `pnpm nx test web` + `pnpm nx test api` full green (**foreground**), `pnpm lint:all` green, a11y pre-flight on the touched dialog.

### AC #5 — Scope fence

- ✅ **The default-helper verb is RULED (party mode 2026-08-06, Alexyu approved Option A):** code adopts 「語音辨識＋AI 翻譯」 (AC #2); the F1 design line is reverted 抽取→語音辨識 via the inline-agent model (Sally's prompt, outside this story's code scope); F2's 抽取 stays (it IS the pipeline screen); the M2 rewire note lives in spec §8. `backlog-dialog-helper-verb-drift` is superseded by the ruling — do not re-litigate it here.
- ❌ No ASR holder / hot-reload work (`backlog-asr-runtime-key-resolution` owns it, and the restart-clause update when it lands).
- ❌ No `.pen` edits, no screenshot regen (γ shipped the design; this is the code half).
- ❌ No new Rule 7 codes, no endpoint changes, no visual baselines.

---

## Tasks / Subtasks

- [x] **Task 1 — F5 strings (AC #1):** title/body/header-comment; dialog spec drift-test updates.
- [x] **Task 2 — Degraded CTA helper (AC #2):** conditional helper + link via `useKeySettings`; open-gated fetch if needed; the three-state tests.
- [x] **Task 3 — 503 envelopes (AC #3):** both handlers' copy + handler tests.
- [x] **Task 4 — Gates (AC #4):** full web+api regression, lint, a11y pre-flight.

---

## Dev Notes

- **The ratified string table in `sub-2-2c-f5-asr-copy-design.md` → Completion Notes is the copy authority.** Byte-match it; do not re-author.
- **Rule 20** — consumes sub-2-1a `[@contract-v1]` AC #3 via `useKeySettings` (record the reuse ack — 2-1b's original ack covers the shape; this story adds a consumer, no bump). Stamps nothing.
- **2-1a CR L1 reminder** — branch on GET's `configured`, never on an error code.
- **Rule 3** — the 503 bodies follow the standard envelope; only strings move.
- **Feedback rules:** foreground tests; `format:check` before commit; design verification = byte-compare against the ratified table (no screenshots owed — the design already shipped).

### Time-dependent visual coverage

**N/A — no wall-clock-reading components touched.** Rule 23 does not apply.

### References

- [`sub-2-2c-f5-asr-copy-design.md`#Completion Notes (ratified table) · #Discovery Triage (this entry's origin)]
- [`apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx`:15,369,372,433 · `apps/web/src/hooks/useKeySettings.ts`]
- [`apps/api/internal/handlers/transcription_handler.go` (English 503) · `generation_batch_handler.go`:71-74 (env-framed 503)]
- [`project-context.md`#Rule 3/5/20]

---

## Dev Agent Record

### Agent Model Used

Amelia (Developer Agent) — Claude Opus 5 (1M context). Implemented 2026-08-06.

### Debug Log References

- Red-green honoured: 5 FE spec failures + 2 BE handler-test failures confirmed before any implementation (the F5 drift assertions, the three helper states, the two 503 bodies).

### Completion Notes List

**🔗 AC Drift: FOUND — sanctioned.** (checked: `grep -rn "字幕生成尚未設定" _bmad-output/implementation-artifacts/*.md` → ux3-subtitle-v2 / 9R-UX design docs pinned the OLD F5 copy, and the dialog spec asserted it. No `[@contract-v*]` stamp exists on any of those copy surfaces (stamp-grep empty). The change is the explicit purpose of this story, sanctioned by γ's ratified table + the two party-mode rulings — recorded as sanctioned drift, not silent.)

**📎 Contract Stamps: NONE bumped; 1 reuse ack.** confirmed against [@contract-v1] sub-2-1a AC #3 (reuse — the GET `/settings/keys` shape via `useKeySettings`; originally acked by sub-2-1b, this story adds the dialog as a consumer, no bump). This story stamps nothing.

**🎭 A11y Pre-Flight: PASS** (2 components/hooks checked — dialog + hook; 0 jsx-a11y warnings on touched files, 0 introduced. The helper link is a native `<button>` with visible text inside the existing `<p>`; no new modal/image/live-region surface.)

**🎨 UX Verification: PASS — byte-compare against the ratified table** (per Dev Notes: no screenshots owed, the design shipped in #204/#205). All five strings byte-match: F5 title/body, degraded helper, default helper (the ruled verb), both 503 envelopes follow the ratified framing.

**What shipped**

- **AC #1** — F5 panel: 「語音辨識尚未設定」 + ASR body with the restart truth; header comment synced and extended with the degradation-vs-block distinction.
- **AC #2** — degraded CTA helper: `useKeySettings({ enabled: open })` (hook gained the optional `enabled` param — backward compatible, gating test added), `claude.configured === false` ⇒ 「僅能產生英文字幕——尚未設定翻譯金鑰」 + 前往設定 (`helper-goto-settings` → `/settings/keys`), CTA stays enabled; loading/error ⇒ default line (strict `=== false` makes the no-flash fence structural); default line carries the ruled verb 「語音辨識＋AI 翻譯，約需數分鐘」; series branch untouched; completion pair untouched.
- **AC #3** — `transcription_handler` 503 now zh-TW (was English); `generation_batch_handler` suggestion re-framed settings-page-first + restart truth + FFmpeg-as-deployment-fact (the bare env-var instruction asserted ABSENT in its test).
- **AC #4** — +7 tests (4 helper states incl. link navigation, 1 hook gating, 2 handler-body): web **228 files / 2546 tests green** (+5 net), api 34 packages green, `lint:all` 0 errors, `format:check` green, gofmt clean, a11y PASS, no orphaned workers.

### Senior Developer Review (AI) — 2026-08-06

**Outcome: Approve (all findings fixed same session).** Adversarial CR: **0 High / 1 Medium / 2 Low — all fixed.** Git-vs-File-List: 0 discrepancies. Rule 7 PASS (zero new codes), Rule 20 N/A (reuse ack only), Rule 25 N/A. Global greps clean: no English 503 remnants anywhere; the batch FE dialog carries no hardcoded copy (surfaces server messages — auto-benefits).

- ✅ **[M1] Open-gating untested at the dialog seam** — the spec's `useKeySettings` mock swallowed its argument, so dropping `{ enabled: open }` would stay green while every mounted-closed dialog fired a fetch. Fix: mock captures options + a two-phase test (closed ⇒ `{enabled:false}`, open ⇒ `{enabled:true}`). The first version of that test itself had a leftover-capture bug (async router render + no beforeEach reset) — fixed with per-phase reset + waitFor.
- ✅ **[L1] `/settings/keys` path unpinned in the per-item 503 test** — the ratified framing's core; Contains added.
- ✅ **[L2] No symmetric English-remnant guard on the per-item test** — `NotContains` for both old English strings added (mirrors the batch test's env-var absence pin).

Gates re-run: web **228 files / 2547 tests green** (+1), api 34 packages green (incl. the tightened handler assertions), lint 0 errors, format green.

### Discovery Triage

- **Did this story discover any work outside its current scope?** `N/A — no further out-of-scope work discovered` (the verb-drift entry was filed at authoring and ruled before implementation; nothing new surfaced).
- Reference: `project-context.md` Rule 24.

### File List

**Modified**

- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx` (F5 strings + header comment + degraded/default helper)
- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.spec.tsx` (drift assertions + 4 helper tests + useKeySettings mock)
- `apps/web/src/hooks/useKeySettings.ts` (optional `enabled`)
- `apps/web/src/hooks/useKeySettings.spec.tsx` (gating test)
- `apps/api/internal/handlers/transcription_handler.go` (zh-TW 503)
- `apps/api/internal/handlers/transcription_handler_test.go` (body assertions)
- `apps/api/internal/handlers/generation_batch_handler.go` (re-framed 503 suggestion)
- `apps/api/internal/handlers/generation_batch_handler_test.go` (body assertions)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (status transitions)
- `_bmad-output/implementation-artifacts/sub-2-2d-f5-cta-code-strings.md` (this file)

## Change Log

| Date | Change |
|---|---|
| 2026-08-06 | Adversarial CR: 1M/2L, all fixed — dialog-seam open-gating test (M1, incl. fixing the test's own async-capture bug), /settings/keys path pin (L1), symmetric English-remnant guards (L2). Web 228/2547, api 34. Story → done. |
| 2026-08-06 | Tasks 1–4, RED-first (5 FE + 2 BE failures pre-impl). F5 speaks γ's ratified ASR copy; degraded CTA helper lands (β Task 4 closed) with the ruled verb on the default line and a structural no-flash fence; both 503s carry the ratified zh-TW framing. +7 tests; web 228/2546, api 34, lint 0 errors, a11y PASS. The α/β/γ/2.2d chain is code-complete. Story → review. |

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **③ filed at authoring, RULED same day: `backlog-dialog-helper-verb-drift`** — party mode 2026-08-06 (Alexyu approved Option A): button stays on Route C ASR (the pipeline's real entry is post-scan auto-enqueue; FR12 has zero FE consumers; rewire converges at M2's 抽>ASR>搜尋). Code verb lands in this story's AC #2; design revert via inline-agent; M2 note in spec §8. Sally owned the 1-7a AC #10 over-extension (F2's correct fix stretched to F1).
  - Otherwise: state `N/A — no further out-of-scope work discovered` at implementation.
- Reference: `project-context.md` Rule 24.

### File List
