# Story sub-2.2d: F5 code strings + degraded CTA helper + 503 zh-TW envelope

Status: ready-for-dev

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

- [ ] **Task 1 — F5 strings (AC #1):** title/body/header-comment; dialog spec drift-test updates.
- [ ] **Task 2 — Degraded CTA helper (AC #2):** conditional helper + link via `useKeySettings`; open-gated fetch if needed; the three-state tests.
- [ ] **Task 3 — 503 envelopes (AC #3):** both handlers' copy + handler tests.
- [ ] **Task 4 — Gates (AC #4):** full web+api regression, lint, a11y pre-flight.

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

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **③ filed at authoring, RULED same day: `backlog-dialog-helper-verb-drift`** — party mode 2026-08-06 (Alexyu approved Option A): button stays on Route C ASR (the pipeline's real entry is post-scan auto-enqueue; FR12 has zero FE consumers; rewire converges at M2's 抽>ASR>搜尋). Code verb lands in this story's AC #2; design revert via inline-agent; M2 note in spec §8. Sally owned the 1-7a AC #10 over-extension (F2's correct fix stretched to F1).
  - Otherwise: state `N/A — no further out-of-scope work discovered` at implementation.
- Reference: `project-context.md` Rule 24.

### File List
