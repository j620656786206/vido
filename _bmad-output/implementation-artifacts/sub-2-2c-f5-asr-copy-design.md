# Story sub-2.2c: F5 panel purified to ASR semantics + CTA degraded-state copy + j2 badge-table 10th row (design)

Status: done

**Epic:** `epic-subtitle-pipeline-m1-5` (M1.5 follow-up) · **Risk: 🟢 UX/DESIGN-ONLY (Sally ux-designer, NOT dev)** · zero code
**Source:** party-mode ruling 2026-08-05 (Sally 主裁) · promotes `backlog-f5-asr-panel-copy-design` · the 1-7a model: design story first, implementation trails
**Split:** γ of the α/β/γ split. Runs **parallel with α**; β's Task 4 trails this story's ratified copy.
**Depends on:** nothing (Pen.app running).
**Blocks:** sub-2-2b Task 4 (soft — β ships Tasks 1–3 without waiting).

---

## Story

As the NAS owner seeing 尚未設定 states,
I want each panel to name the capability that is ACTUALLY missing and the action that fixes it,
so that an FFmpeg/ASR gap, a translation-key gap, and a finished-but-untranslated result each read as what they are.

---

## 🔎 Findings (verified 2026-08-05, party mode — the premise corrections that motivated this story)

1. **The F5 panel's trigger is ASR, not translation.** The code panel (`generation-not-configured`) fires on **503 `TRANSCRIPTION_DISABLED`** — `TranscriptionService.IsAvailable()` = FFmpeg (audioExtractor) + ASR key (`transcription_service.go:177`), reached via `transcription_handler.go:52`. It does NOT fire on the 409 `AI_NOT_CONFIGURED` (Claude translation key) at `subtitle_pipeline_handler.go:101` — a DIFFERENT endpoint this dialog never calls. **sub-1-7a's ratified F5 copy (「尚未設定翻譯服務金鑰」…) is therefore aimed at the wrong capability and must NOT be pasted into code** — adopting it verbatim would make the panel claim a missing translation key when the actual cause can be a missing FFmpeg binary.
2. **The ruling's core asymmetry:** ASR/FFmpeg missing = **BLOCKING** → stays a warning panel (`$warning-tint` correct). Translation-key missing = **DEGRADED** (English is still producible) → never enters F5; it lives in the CTA helper line (pre-flight) and the completion message + 未翻譯 badge (post-hoc, α/β).
3. **`/settings/keys` exists now** (sub-2-1b, PR #200) with a 雲端 ASR（選配） row — so F5's M1.5 affordance can be a real 前往設定 again (1-7a deleted it in the M1 design because no page existed; that rationale is void).
4. **Keys hot-reload since sub-2-1a** — any copy saying 「…後重啟伺服器」 about the Claude/ASR keys is stale (the backend's own 409 suggestion was already rewritten at 2-1a CR H1). FFmpeg remains a deployment fact: 「FFmpeg／FFprobe 已內建於 Docker 映像檔，無需另行安裝。」 (1-7a's own ruling, keep).

---

## Acceptance Criteria

### AC #1 — F5-D-v2 (`f6ZxY`) purified to ASR-only semantics

**Given** Finding 1, **then** the F5 screen's copy names the ASR capability, not translation. Sally authors the exact strings; constraints:

- Title/body must be truthful for BOTH sub-causes of `TRANSCRIPTION_DISABLED` (FFmpeg absent · ASR key absent) — or explicitly present the two.
- Keep `$warning-tint` (blocking = warning is semantically correct — 1-7a escalation ③ re-affirmed).
- FFmpeg framed as a deployment fact (Finding 4's string), NOT a user setting.
- The M1.5 affordance: 前往設定 → `/settings/keys` is now legitimate (Finding 3) — Sally decides between restoring the button or keeping the 查看部署說明 link + adding the button; the decision and rationale are recorded in Completion Notes.
- No 「重啟伺服器」 phrasing for key-based causes (Finding 4).
- The dev-facing handoff is the **ratified string table in Completion Notes** (spec PNGs export ≤400px — `backlog-pen-spec-screen-readable-export`), same as 1-7a.

### AC #2 — F1-D-v2 (`r1EY9`) CTA degraded-state helper

**Given** the ruling's pre-flight half, **then** F1 gains the conditional helper-line state under 生成字幕: translation key unconfigured ⇒ 「僅能產生英文字幕——尚未設定翻譯金鑰」-class copy + a settings affordance; the CTA itself stays visually ENABLED (degraded ≠ blocked). Sally ratifies the final string + affordance shape; β Task 4 implements it verbatim.

- Also ratify the en-only **completion** line (β AC #3, working copy 「已生成英文字幕；尚未翻譯」) so dialog copy ships from one ruling.

### AC #3 — `flow-j-specs/j2-d` badge table gains the 10th row

**Given** 1-7a's spec screen owns the badge vocabulary, **then** the table adds: **未翻譯** · neutral tint (`--bg-tertiary`/`--text-muted` class pair, NO new colour token) · generation-artifacts-only (never inferred from embedded tracks) · recovery = 設定金鑰後重跑（僅執行翻譯段） · distinct-by-recovery note vs 缺字幕/無字幕源.

### AC #4 — Export + staging discipline

1. `python3 scripts/export-pen-screenshots.py` after the `.pen` edits (Pen.app running; **verify the .pen is SAVED before export** — MCP reads app memory, not disk).
2. Stage ONLY the changed PNGs — expected: `f5-d-v2.png`, `f1-d-v2.png` (+ `f1-m-v2.png` if the mobile frame carries the helper line), `j2-d.png`. A surviving 5th PNG means re-render noise was staged (full regen is non-deterministic).
3. No `SCREENS` dict changes (all three nodes already mapped).
4. Commit `.pen` + PNGs together: `feat: update UX design — F5 ASR purify + F1 degraded CTA + j2 未翻譯 row`.

### AC #5 — Scope fence

- ❌ Zero code files — β implements; α is independent.
- ❌ No new screens/frames — three existing nodes only (a genuinely-new-screen need = lane ② stop-and-file).
- ❌ No badge colour-token invention (1-7a constraint stands).
- ❌ No F2/F10 or other subtitle-flow screens.

---

## Tasks / Subtasks

- [x] **Task 1 — F5 purify (AC #1):** author strings, decide affordance, edit `f6ZxY`, record ratified table + rationale in Completion Notes.
- [x] **Task 2 — F1 CTA state (AC #2):** degraded helper + completion line, edit `r1EY9` (+ mobile if applicable), ratify strings.
- [x] **Task 3 — j2 10th row (AC #3):** extend the badge table.
- [x] **Task 4 — Export + commit (AC #4):** save-check, export, stage exactly the changed PNGs, commit.

---

## Dev Notes

- **Pencil MCP only** for `.pen` access (never Read/Grep); `get_app_state` with all four flags first if schema not in context.
- **1-7a is the template** for this story's shape: standalone design story, ratified-table handoff, exact-PNG staging, UX gate with Alexyu before close.
- **Copy constraints inherited:** labels 3–4 CJK chars for badges; F5 warning semantics; byte-alignment with backend strings is a NON-goal here (the backend 503 message is English today — flagging THAT is allowed as a lane ③ discovery, not fixed here).
- **Feedback rules:** verify `.pen` saved before commit (`feedback_verify_pen_saved_before_commit`); label overlap check (`feedback_pencil_label_overlap`); spec annotations get their own screen only if genuinely new (none expected).

### Dev-flagged inputs for AC #3's j2-d row (from sub-2-2b CR, 2026-08-05)

1. **Semantics note (CR L2):** `untranslated` + an embedded 繁中 track renders 未翻譯 — the verdict outranks track inference, consistent with `no_text_source`/`skipped`. But in this (rare, force-regenerated) case the recovery hint is moot: a 繁中 track already exists and the user needs to do nothing. Sally decides whether the j2-d row's recovery wording acknowledges it or the consistency stance stands as-is. No code shipped either way by β.
2. **EpisodeList glyph ratification (CR M1):** β added the `untranslated` EpisodeList entry ahead of 9R-10a (so a settled verdict never falls back to 尚未搜尋字幕's bare `Minus`). Per the J2-D icon grammar it uses CIRCLED + muted with label 「已生成英文字幕，尚未翻譯」; the **`CircleDashed` glyph is PROVISIONAL** — Sally ratifies or replaces it in the j2-d row (a one-line map edit either way; the label is the contract).

### Time-dependent visual coverage

**N/A — design-only.** Rule 23 does not apply.

### References

- [party-mode ruling: sprint-status `backlog-f5-not-configured-panel-copy-ruling` (superseded entry, full record)]
- [`apps/api/internal/services/transcription_service.go`:177 (`IsAvailable` = FFmpeg+ASR) · `apps/api/internal/handlers/transcription_handler.go`:52 (503 gate) · `apps/api/internal/handlers/subtitle_pipeline_handler.go`:101 (the OTHER 409 — not this dialog's trigger)]
- [`apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx`:356-379 (the panel as coded) · :425 (CTA helper line)]
- [`sub-1-7a-subtitle-status-badge-design.md` (template + the superseded F5 copy + FFmpeg framing) · `sub-2-2b-untranslated-badge-frontend.md`#AC #4]

---

## Dev Agent Record

### Agent Model Used

Sally (UX Designer) — Claude Opus 5 (1M context). Executed 2026-08-06 via the **inline-agent collaboration model** (Alexyu ruling this session): Sally authored the ratified strings + three node-anchored prompts → Alexyu ran them through Pencil's Inline AI Agent → Sally reviewed via Pencil MCP (structure + verbatim copy + bounds), Alexyu handled save-verify (AppleScript ⌘S after catching the disk file stale at 8/4 — the `feedback_verify_pen_saved_before_commit` trap), export (147/147), exact-PNG staging (3 only, noise reverted), and the signed commit `8207310`.

### Debug Log References

- Copy premise verified in code BEFORE authoring: `TranscribeMovie` gates on 503 `TRANSCRIPTION_DISABLED` = `IsAvailable()` = FFmpeg + ASR key (`transcription_service.go:177`) — NOT the translation-key 409. AND the **WhisperClient is boot-built** (`main.go:505` `if cfg.HasOpenAIKey()`), so a UI-saved ASR key needs a RESTART — the F5 body says so; 「立即生效」 would have been a lie (→ Discovery Triage #2).
- MCP review evidence: F5 panel text verbatim + `btn-goto-settings` present + design-notes order M1.5→FFmpeg→查看部署說明 + zero `ctx.problems`; F1 `design-notes-conditional` block sits between `sec-generate` and `row-glossary` with all 6 lines verbatim; J2 `row-untranslated` five columns x/w EXACTLY match `row-skipped` (0/200/410/660/850), pill 未翻譯, `circle-dashed` icon, constraint-box +2 lines verbatim.

### Completion Notes List

**🎨 THE RATIFIED STRING TABLE (the dev handoff — spec PNGs export ≤400px, this table is authoritative):**

| Surface | Ratified |
|---|---|
| F5 title | 「語音辨識尚未設定」 |
| F5 body | 「生成字幕需要雲端語音辨識（ASR）金鑰。請至金鑰設定儲存，並重啟伺服器後再試。」 |
| F5 primary | 前往設定 button → `/settings/keys` (RESTORED — 2-1b shipped the page incl. the 雲端 ASR row; 1-7a's deletion rationale is void) |
| F5 secondary | 查看部署說明 → moved beside the FFmpeg deployment-fact line |
| F1 helper (default) | 「抽取內嵌字幕＋AI 翻譯，約需數分鐘」 (unchanged) |
| F1 helper (degraded) | 「僅能產生英文字幕——尚未設定翻譯金鑰」 + 前往設定 text link → `/settings/keys`; **CTA stays enabled** (degraded ≠ blocked); signal = GET `/settings/keys` `claude.configured` |
| Completion ×2 | 「字幕已生成完成」 / 「已生成英文字幕；尚未翻譯」 (β's shipped copy ratified as-is) |
| j2-d 10th row | 未翻譯 · 底 `#2e3b56`/字 `#a0aabe` · EpisodeList **CircleDashed** `--text-muted` |

**Rulings on the three dev-flagged inputs (all closed):**

1. **CircleDashed glyph: RATIFIED.** Fits the J2-2 icon grammar exactly — circled = settled, dashed = incomplete (英文字幕已生成、翻譯未執行). β's provisional pick stands; the KGtqa grammar line now names it.
2. **untranslated + embedded-繁中 semantics: consistency stands.** Verdict-outranks-tracks is the family rule; the rare case is ACKNOWLEDGED in the constraint box as a deliberate tradeoff, not a defect.
3. **F5 trigger purity: the panel now names ASR**, never the translation key — the 1-7a copy (wrong capability, wrong endpoint) is fully superseded.

**Inline-agent deviation, reviewed and APPROVED:** the KGtqa addendum was inserted mid-sentence (after CircleSlash, before the closing 「—— 管線對這個檔案已經有答案。」) rather than literally at the end — Alexyu's grammar call, and better than my instruction: the closing clause now covers all four circled glyphs.

**AC #4 export discipline (executed by Alexyu):** disk-stale trap caught (⌘S via AppleScript, verified `M ux-design.pen` + grep before export); 147/147 export; EXACTLY three PNGs staged (`f5-d-v2.png`, `f1-d-v2.png`, `j2-d.png` — F1-M carries no helper-line annotation, desktop annotation notes mobile parity), re-render noise reverted; `SCREENS` dict untouched.

### Discovery Triage

- **Did this story discover any work outside its current scope?** Two lane-③ items, both filed at discovery time (bidirectional):
  - **③ `backlog-f5-cta-code-strings`** — the CODE half this design story deliberately does not ship: F5 panel strings in `ManageSubtitleDialogV2.tsx` still carry the pre-γ copy; the AC #2 degraded CTA helper (β Task 4's explicit deferral) and the F5 前往設定-button copy alignment now have ratified strings and nothing tracking implementation. One small frontend story/entry.
  - **③ `backlog-asr-runtime-key-resolution`** — WhisperClient is boot-built from env (`main.go:505`), the exact class 2-1a fixed for Claude and `backlog-tmdb-runtime-key-resolution` tracks for TMDb; OpenAI/ASR had NO tracking entry. A UI-saved ASR key silently does nothing until restart (F5/keys-page copy now says so honestly, but the fix-when-picked-up is a holder or resolver re-point).
  - Pre-flagged candidate CONFIRMED still open: backend 503 message is English (Rule 3 gap) — folded into `backlog-f5-cta-code-strings` (same handler surface).
- Reference: `project-context.md` Rule 24.

### File List

- `ux-design.pen` (F5-D-v2 `f6ZxY` · F1-D-v2 `r1EY9` design-notes block · J2-D `FdlJ2` row + grammar + constraints)
- `_bmad-output/screenshots/flow-f-subtitle-v2/f5-d-v2.png` · `f1-d-v2.png` · `flow-j-specs/j2-d.png`
- `_bmad-output/implementation-artifacts/sub-2-2c-f5-asr-copy-design.md` (this file)
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
|---|---|
| 2026-08-06 | Tasks 1–4 via the inline-agent model (Sally prompts → Alexyu executes → Sally MCP-reviews). F5 purified to ASR semantics with the verified restart truth + 前往設定 restored; F1 conditional-copy spec block (helper ×2, completion ×2, mobile parity note); J2-D 10th row + CircleDashed ratified + 2 constraint notes. KGtqa mid-sentence placement deviation approved. 3 PNGs staged exactly; commit `8207310`. Two lane-③ discoveries filed: `backlog-f5-cta-code-strings`, `backlog-asr-runtime-key-resolution`. Story → done. |

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - If **NO**: state `N/A — no out-of-scope work discovered`.
  - Pre-flagged candidate (lane ③ if confirmed): the backend 503 `TRANSCRIPTION_DISABLED` message is English (`"Transcription is not available"`) — a Rule 3 zh-TW envelope gap, backend-owned, NOT fixed by a design story.
- Reference: `project-context.md` Rule 24.

### File List
