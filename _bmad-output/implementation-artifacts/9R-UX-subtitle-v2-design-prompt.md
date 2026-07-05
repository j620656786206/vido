# 9R-UX — Subtitle management v2 design prompt (for the Pencil In-App AI agent)

> Paste the block below into Pencil.app's in-app AI agent to draw the generation-centric
> subtitle-management v2 screens. Then ux-designer reviews the result via Pencil MCP
> (read/screenshot/targeted-fix) — token-light vs. drawing the whole flow over MCP. Same path
> as ux3-1-1 / ux3-3-1 / ux3-4-1. Story spec: `9R-UX-subtitle-v2-design.md`.

---

You are redesigning the **字幕管理 / Subtitle management** flow of the Vido NAS media app to the
**v2 design language** — and re-centering it from *fetch* to **generation (Route C)**. Work in
the current `ux-design.pen`. Create NEW frames in a **new flow group** `flow-f-subtitle-v2`.

**The one-sentence brief:** online subtitle fetching for 繁中 is DEAD (sources unreachable);
subtitles are now **generated** — 提取音訊 → 轉錄 (Whisper) → LLM 翻譯 → 簡轉繁 → 完成 — with a
**per-show glossary (名詞對照表)** keeping proper nouns consistent. `生成字幕` is the primary
action everywhere; fetch survives only as one quiet, dormant secondary affordance.

Name the new frames (caption text above each frame, Noto Sans TC 14/600 `#888888`, ~45px above
the frame so it doesn't collide with Pencil's frame-name chrome):

`F1-D-v2 · 管理字幕 — 已有字幕（桌面）` · `F1-M-v2 · 管理字幕（手機）` ·
`F2-D-v2 · 管理字幕 — 缺字幕` · `F3-D-v2 · 生成進度` · `F3-M-v2 · 生成進度（手機）` ·
`F4-D-v2 · 生成失敗` · `F5-D-v2 · 尚未設定 fail-soft` · `F6-D-v2 · 名詞對照表` ·
`F6-M-v2 · 名詞對照表（手機）` · `F7-D-v2 · 名詞對照表 — 空狀態` · `F8-D-v2 · 批次生成` ·
`F8-M-v2 · 批次生成（手機）` · `F9-D-v2 · 批次生成 — 預算上限` · `F10-D-v2 · 載入骨架`

Start with a block title (`DM Sans 24/700` `#222222`: `F v2 字幕管理（生成中心） — Subtitle v2
(generation-centric)`) + description line (Noto Sans TC 14 `#666666`). Desktop row above, mobile
row below, same step column-aligned (merged-block convention). Use `FindEmptySpace` to anchor
the whole block **clear of every existing frame** (the old F block sits at x≈17040 y≈13120, the
old G block at x≈17040 y≈15720 — do NOT overlap or touch them). Set `clip: true` and
`placeholder: true` on each new screen while building; remove `placeholder` per finished screen.

## ⚠️ Do NOT reuse / mutate these (superseded surfaces — leave untouched)

- `F1-D` `cOrOR` / `F1-M` `GZ294` / `F2-D` `wy5Nx` / `F2-M` `ogQ6Y` / `F3-D` `NXijD` /
  `F3-M` `fUtqO` — the **dead fetch UI** (Assrt/Zimuku/OpenSub source chips, source-scored
  rows, batch-fetch). Do not copy their content; the sources no longer work.
- `G1-D` `TIIRl` / `G1-M` `mgRJA` / `G2-D` `kzhNP` / `G2-M` `yNAHK` / `G3-D` `22bcv` /
  `G3-M` `8Wsez` — pre-v2 AI-correction mockups; their generation ideas move into F3 here.
- **NEVER show Zimuku anywhere.** No fetch-source chips, no source-score breakdowns, no
  multi-source picker — that world is retired.
- After the new block is done, add ONE standalone note frame near each old block (do not touch
  the frames): amber note style (`fill #F59E0B15`, `stroke #F59E0B40`, radius 6 — same idiom as
  the existing `design-note` in `cOrOR`), text:
  `⚠️ SUPERSEDED 2026-06-16（ADR Route C）— 新版見 flow-f-subtitle-v2`.

## The surfaces (composition)

**F1/F2 — detail `管理字幕` dialog (desktop) / bottom sheet (mobile).** Opened from the v2
detail page's 管理字幕 CTA (reference backdrop: dim the v2 detail frame `B3p-D` `uRGu2` behind a
`$bg-secondary` dialog, `$radius-lg`, `$border-subtle`, `$overlay-scrim` — same overlay idiom as
old `cOrOR` but ALL-new v2 content). Dialog content, top → bottom:

1. **現有字幕 section** — list of subtitle tracks: language pill (`繁中` / `簡中` / `英文`),
   source label (`已生成` / `線上下載` / `本地檔案`), filename (Noto Sans TC, mid-truncate).
   F1 shows a populated list; F2 shows the empty variant (`尚無字幕`).
2. **Primary action:** `生成字幕`（`Component/ButtonPrimary` restyled v2）— THE hero action,
   especially prominent in F2 (缺字幕). Subtext: `轉錄＋AI 翻譯，約需數分鐘`.
3. **簡轉繁 affordance** on 簡中 tracks (`轉為繁中` secondary button). If the title is 中國大陸
   content, show a quiet policy line instead: `陸劇保留簡體字幕（對白一致）` + an override
   affordance — the simplified track on CN content is CORRECT, not an error.
4. **名詞對照表 entry** — a quiet row: `名詞對照表（12 條）` + chevron → F6.
5. **Dormant fetch secondary** — ONE quiet text-level affordance at the bottom:
   `搜尋線上字幕（成功率低）` styled `$text-secondary`, NO chips, NO sources named. Honest and
   small; never competes with 生成字幕.

**F3 — generation progress.** The pipeline as a stage stepper — build it as a NEW reusable
component **`Component/GenerationProgress-v2`** first, then instance it here and in F8:
stages `提取音訊 → 轉錄中 → 翻譯中 → 簡轉繁 → 完成` (done = `$success` check, active = spinner +
`$accent-text` label + Mono %, pending = `$text-muted`). Below the stepper: current-stage detail
line + a cost line `本次用量：$0.42 / 上限 $5.00`（JetBrains Mono numerics, split number and
unit into separate text nodes）. Add a small live-push note chip: `即時更新（SSE）`. Cancel =
`ButtonSecondary` `取消`.

**F4 — failed-at-stage.** Same stepper with one stage in `$error` state, an `$error-tint` panel
(`翻譯失敗：AI 服務逾時` in `$error-text`) + `重試` primary + `稍後再試` secondary. The dialog
still renders everything else (fail-soft, never a dead end).

**F5 — not configured.** `TRANSCRIPTION_DISABLED` state: calm `$warning-tint` panel —
`字幕生成尚未設定` + explainer `需要 FFmpeg 與 AI API Key` + `前往設定` button. The 現有字幕
list still renders above it; only generation degrades.

**F6/F7 — 名詞對照表 (glossary).** Per-show term table INSIDE the dialog context (title bar
shows the show name). Build rows as a NEW reusable component **`Component/GlossaryRow-v2`**:
`term_src`（原文, Noto Sans TC/JetBrains Mono for Latin terms）↔ `term_zh`（繁中）+ a **source
badge** (`字幕` `$info-tint` / `中繼資料` `$accent-tint` / `手動` `$bg-tertiary`) + a
**confirmed state**: unconfirmed rows carry a `未確認` `$warning-tint` chip + `確認`/`編輯` row
actions; confirmed rows are quiet. Header: `新增詞彙` secondary button + a one-line explainer
`生成字幕時會依此表固定譯名`. F7 = empty state: `尚無詞彙 — 生成字幕時自動累積` + `新增詞彙`.

**F8/F9 — 批次生成 (library-wide).** Entered from the 活動 Activity hub. Desktop = dialog over
the Activity v2 page (backdrop reference: `A1-D` `kMeWS`; reuse `Component/ActivityRow-v2`
`fF8nX` idiom for rows — do NOT invent a second task-row look). Content: scope line
(`範圍：缺字幕的項目（38 部）`), a per-item progress list (poster thumb + title +
`GenerationProgress-v2` compact instance + per-item stage label), overall header
`已完成 12 / 38`（Mono）+ overall bar, `全部取消` secondary. F9 = budget-ceiling state: a
`$warning-tint` banner `已達本次預算上限（$5.00）— 已完成 12 部，剩餘 26 部下次繼續`，list shows
completed ✓ + remaining paused; partial completion reads as a NORMAL outcome, not an error.

**F10 — loading skeleton.** The F1 dialog shape with `$bg-secondary`/`$bg-tertiary` skeleton
blocks (track rows + button strip); note `prefers-reduced-motion` respected.

## ⚠️ Capability-honor spec notes (draw as TARGET state, mark the gaps)

Everything generation-shaped is **net-new backend** — draw it, and add ONE amber design-note
frame (same idiom as above) inside the new block, listing:
`BE gaps（設計為目標狀態，尚未上線）：① 生成管線＋階段 SSE = 9R-10 ② glossary HTTP API = 9R-15
③ 批次生成 = 9R-10+11 ④ 影集觸發 = 9R-10（現況僅電影）`. Nothing in the frames may claim to be
live; no fake data promises (e.g. don't label the SSE chip as "connected").

## Design language v2 (apply everywhere)

- **Type (zero-tolerance):** ALL CJK = **Noto Sans TC**. DM Sans ONLY for the `vido` logo /
  pure-English display. **JetBrains Mono for ALL numerics** (%, counts, N/M, cost figures,
  timestamps). Mixed strings like `12 部` / `$5.00 上限` MUST be split into a Mono number node +
  a Noto unit node (gap 3-4) — the cnt/cntUnit precedent. This applies to EVERY new frame, even
  content visually copied from elsewhere.
- **Color = tokens only, no hex literals** (the two amber note frames are the sole exception —
  they are canvas annotations, not UI). Variables: `$bg-primary`, `$bg-secondary`,
  `$bg-tertiary`, `$text-primary`, `$text-secondary`, `$text-muted`, `$text-disabled` (never
  load-bearing), `$accent-primary`, `$accent-text`, `$accent-subtle`, `$accent-tint`,
  `$success`/`$success-tint`, `$warning`/`$warning-tint`, `$error`/`$error-text`/`$error-tint`,
  `$info`/`$info-tint`, `$border-subtle`, `$overlay-scrim`, `$radius-*`, `$shadow-*`, `$gap-*`.
  Reference the `Design Language v2` frame `V2Kez` (esp. `sec-states` for the status→token map —
  generation states join the SAME system as the lifecycle badges, no bespoke palette).
- **Colored body text uses the `*-text` AA variants** (`$accent-text`/`$error-text`), never the
  base fill hues.
- **Touch targets ≥ 44×44px** on every interactive element; dialog buttons, row actions, chips.
- **Spacing:** section rhythm 24–48; intra-component 8–12. Flat chrome (`$border-subtle`).
- Base-UI-style dialog anatomy: title bar + close ✕ (44px), footer action row right-aligned.

## Mobile frames (390px)

`F1-M` / `F3-M` / `F6-M` / `F8-M` are **bottom sheets** (`$radius-xl` top corners,
`$overlay-scrim` backdrop, drag handle) over their page context. Keep the SAME content order as
desktop, condensed; row actions become full-width 44px rows; the stepper stacks vertically if
horizontal doesn't fit. Bottom tab bar where the page context shows one: bottom-4
`首頁 · 媒體庫 · 活動 · 下載` + More (use `Component/MobileTabItem` `S86VM`) — 字幕 is NOT a
tab; it's a feature surface reached from 詳情 or 活動.

## Components to REUSE (don't reinvent)

- `Component/ActivityRow-v2` (`fF8nX`) — batch rows idiom in F8.
- `Component/ButtonPrimary` (`otvKh`) / `ButtonSecondary` (`YDPhc`) — restyle instances to v2
  tokens; don't fork new button components.
- `Component/MobileTabItem` (`S86VM`) — mobile tab bar.
- `Component/HomeSidebar-v2` (`BDeUS`) + `Navigation Shell v2` frame (`CLo58`) — only if a page
  context needs the shell (dialog backdrops may reuse existing v2 page frames dimmed).
- `Component/TechBadge-Subtitle` (`f84BM`) — visual anchor for how subtitle info renders today.
- NEW reusables this task creates: `Component/GenerationProgress-v2`, `Component/GlossaryRow-v2`
  (create each as a top-level reusable component FIRST, in its own step, then instance).
- **Register both new components in the Component Library frame `sJzat`**: add a cell in the
  matching v2 section (`content-cards-v2` for GlossaryRow; `filter-controls-v2` or a new
  `progress-v2` row for GenerationProgress) — cell = vertical frame gap 8, ref instance
  (narrowed if needed) + Noto Sans TC 12 `$text-muted` caption below.

## Done checklist

- New frames only, in the new `flow-f-subtitle-v2` block; old `F*`/`G*` frames UNTOUCHED, each
  old block gets one standalone superseded note.
- `生成字幕` is unmistakably the primary action; fetch = one quiet dormant affordance, no
  source chips, Zimuku nowhere.
- Pipeline stepper states complete (pending/active/done/failed); SSE note present; cost line
  Mono + split number/unit.
- Glossary rows show source badge + confirmed/unconfirmed + row actions; empty state drawn.
- Batch surface reuses ActivityRow-v2 idiom; budget-ceiling partial completion reads as normal.
- Four states covered: default (F1/F6/F8) / loading (F10) / empty (F2/F7) / fail-soft (F4/F5/F9).
- CN-content 簡繁 policy line present on F1.
- The amber BE-gaps note frame present (9R-10 / 9R-15 / 9R-10+11 / series trigger).
- All CJK Noto Sans TC; all numerics JetBrains Mono (mixed strings split); tokens only; AA
  `*-text` variants; 44px targets; nothing clipped; captions don't overlap frame chrome.
- Both new components registered in `sJzat`; `placeholder` removed on every finished screen.

---

### After the in-app agent finishes (ux-designer, via MCP — token-light)

1. `get_screenshot` each new `F*-v2` frame; check against this checklist + DL-v2 (`V2Kez`) +
   story ACs (`9R-UX-subtitle-v2-design.md` AC #1–#9).
2. Targeted `batch_design` fixes only (don't redraw): hex literals → tokens, font slips (CJK in
   DM Sans / unsplit number+unit strings), missing states, 44px floors, Zimuku/source-chip
   leakage, the amber BE-gaps + superseded notes, `sJzat` registration cells.
3. Confirm old `f1-f3` (`cOrOR` `GZ294` `wy5Nx` `ogQ6Y` `NXijD` `fUtqO`) + `g1-g3` (`TIIRl`
   `mgRJA` `kzhNP` `yNAHK` `22bcv` `8Wsez`) are byte-untouched (no accidental edits).
4. **Cmd+S in Pencil.app** (MCP/inline edits need a manual save), then update
   `scripts/export-pen-screenshots.py` `SCREENS` (new node IDs → `("flow-f-subtitle-v2", "f1-d-v2")`
   etc.) → `python3 scripts/export-pen-screenshots.py` → commit `.pen` + **only
   genuinely-changed PNGs** (regen is non-deterministic) on a new branch off main as
   `feat(9R-UX): subtitle management v2 design — generation-centric (.pen flow-f-subtitle-v2)`.
5. Fill the story's Dev Agent Record (node IDs, deviations) → status `review`; after
   code-review/merge set `9R-UX-subtitle-v2-design` → `done` in sprint-status (epic-9R block).
