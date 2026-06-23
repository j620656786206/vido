# ux3-3-1 — Discover v2 REDRAW prompt (popover → persistent instant rail)

> Paste the block below into Pencil.app's in-app AI agent. It REVISES the existing
> `flow-i-discover-v2` frames so the DESKTOP filter is a persistent instant LEFT RAIL
> (matching the shipped `/discover` + the sibling 媒體庫 Library rail), NOT a batch popover.
> Decision basis: adversarial UX panel — instant rail won 9/9/9 across three lenses and
> survived the refuter; the shipped `/discover` is ALREADY an instant rail with region/rating/
> streaming LIVE, so the popover was a regression. After you finish, I'll read it via Pencil MCP,
> re-export screenshots, review, and update the story.

---

You are REVISING the **探索 / Discover** v2 screens in the current `ux-design.pen` (flow group
`flow-i-discover-v2`). The DESKTOP filter must change from a toolbar-button + floating popover
(batch 套用) to a **PERSISTENT INSTANT LEFT RAIL** — visually and behaviorally the SAME pattern
as the already-shipped 媒體庫 Library filter rail. Do NOT create a new flow group; EDIT the
existing frames in place (do not delete-and-recreate). Keep all v2 token/type/a11y rules.

## Use the Library rail as your structural template (converge, don't invent)

The shipped Library filter rail already exists in this file — frames **`i5-d` (`vpDLh`)** =
rail persistent, and **`i6-d` (`VwTvy`)** = rail collapsed. **Model Discover's rail on those**:
same 264px persistent 2nd-level rail chrome (`$bg-primary`, distinct from the `$bg-secondary`
nav), same collapse affordance, same instant-apply feel. Discover only swaps in its OWN
dimension set + per-facet counts + Discover-only furniture. The two rails should read as
siblings.

## The new DESKTOP layout pattern (apply to every desktop frame)

Three columns, left→right:
1. **v2 nav sidebar** — `Component/HomeSidebar-v2` (`BDeUS`) instanced, active item = **探索**
   (already set in these frames — keep it).
2. **Persistent filter rail (264px, `$bg-primary`, right `$border-subtle` 1px)** — the EDITOR:
   - Header row: `篩選` (Noto 16/600) + a **collapse chevron** (◀) toggling to the collapsed
     state. (Model on Library i5-d's rail header.)
   - **Instant apply — there is NO 套用 / 重設 button anywhere.** Toggling a chip reshapes the
     grid immediately (this is the whole point; do not draw an apply button).
   - Dimension sections, each a label + chip group, with active chips = `$accent-subtle` +
     `$accent-text` and **a per-facet result count in JetBrains Mono next to each value**
     (e.g. `動作 340`, `2020s 128`, `7+ 412`). Sections:
     - **類型 (genre)** — chips 動作/驚悚/喜劇/劇情/科幻/恐怖/愛情/動畫 (some active)
     - **年份 (year)** — chips 2020s/2010s/2000s/90s (or a min–max range pair)
     - **評分 (rating)** — chips 7+/8+/9+ (some active)
     - **地區 (region)** — **NOW LIVE** chips 美國/日本/韓國/台灣/香港 (NOT "即將推出")
     - **串流平台 (streaming)** — **NOW LIVE** chips Netflix/Disney+/Apple TV+/HBO/Prime
       (NOT "即將推出")
   - ⚠️ **Remove the「即將推出」disabled treatment from 地區 and 串流平台** — these are
     backend-backed and live today. Draw them as real, toggleable chips like 類型/年份/評分.
3. **Results column (flex, fills remaining)** — Discover-only furniture ABOVE the grid (these do
   NOT go in the rail):
   - **Preset chip row** — saved presets 週末動作片/經典科幻 + `+ 儲存目前篩選` (Epic 11 E-6).
   - **Active-filter summary chip bar** — the active filters as a removable READ/REMOVE summary
     + `清除全部`. (This is a SUMMARY, not a second editor — the rail is the editor; keep it
     visually lighter than the rail's chips so they don't read as two competing controls.)
   - **Thin toolbar** — **sort control** (評分排序 / 新增日期 — sort lives in the toolbar, matching
     the Library rail convention) + the **想要清單 · 即將推出** inert Epic-13 entry (KEEP this,
     opacity 0.5, `$text-disabled`). **Remove the old `篩選` (sliders) popover-trigger button** —
     the rail replaces it; the only filter toggle now is the rail's collapse chevron.
   - **Results grid** — `Component/PosterCard-v2` (`hD7Tw`), rating/評分 visible on every card
     (keep as-is).

## Per-frame changes

- **`I1-D-v2` (`fxCVk`, main-content `eiTCo`) — DEFAULT (hero):** restructure to the 3-column
  pattern above. Rail EXPANDED, all 5 dims live with per-facet counts. Move the dimension chips
  OUT of the old popover and INTO the rail. Keep search? — the big search input belongs in the
  results column top (it drives the instant-search suggestions, #I3); keep it above the preset
  row. Remove the toolbar `篩選` popover button.
- **`I4-D-v2` (`m4fY7c`, main-content `nU8Pn`) — REPURPOSE popover → COLLAPSED RAIL state:**
  delete the floating filter-panel popover; instead show the rail **collapsed** to a `篩選(n)`
  button (n = active filter count, e.g. `篩選(3)`) and the **grid reflowed WIDER** (more poster
  columns) to demonstrate the width reclaim. Model exactly on Library `i6-d` (`VwTvy`).
  Rename the frame to `I4-D-v2 · 篩選 rail（收合）`.
- **`I6-D-v2` (`YYEBd`) — loading skeleton:** add the rail column (skeleton-shaped rail sections)
  + grid skeleton; match the new 3-column layout.
- **`I7-D-v2` (`S3qke`, content `nZVwI`) — no-result:** add the persistent rail (expanded) on the
  left; keep the no-result state (找不到相符的結果 + active-filter echo + 清除篩選/調整搜尋) in
  the results column.
- **`I8-D-v2` (`KdnVw`, content `E2CZU7`) — per-section fail-soft:** add the persistent rail on
  the left; keep the fail-soft sections (媒體庫結果 renders / TMDB inline error+重試 / 串流平台
  panel) in the results column. NOTE: since 串流平台 is now a LIVE rail dimension, the old
  「串流平台可用性 · 即將推出」results-section can stay as a separate availability concept OR be
  dropped — your call, but the rail's 串流平台 FILTER must be live.
- **`I3-D-v2` (`m0Zew`) — suggestions:** keep the sectioned (媒體庫/TMDB) suggestions popover over
  the search input, but update the BACKGROUND to the new 3-column rail layout for consistency.
- **`I5-D-v2` (`nLrzc`) — save-preset dialog:** unchanged.

## MOBILE — DO NOT CHANGE (this is correct as-is)

- **`I2-M-v2` (`hi6WD`)** and **`I4-M-v2` (`kzzjc`, the bottom sheet)** stay exactly as they are.
  Mobile correctly uses a `篩選` button → bottom SHEET with BATCH apply (`套用篩選（N 部結果）`),
  because a transient sheet = batch is the right mobile pattern and matches the shipped
  `FilterBottomSheet`. Leave 探索-via-More + the bottom-4 tab bar untouched.

## Design language v2 (unchanged rules)

- Color = tokens only (`$bg-primary/secondary/tertiary`, `$accent-subtle/text/tint`,
  `$success*/$warning*/$error*`, `$border-subtle`, `$text-primary/secondary/muted/disabled`,
  `$radius-*`, `$gap-*`). No hex literals.
- ALL CJK = Noto Sans TC; per-facet counts / years / ratings = JetBrains Mono.
- Touch ≥ 44px; visible focus; rail collapse respects the same affordance as Library.
- `placeholder: true` on any frame while you edit it; remove when that frame is done.

## Done checklist

- Every DESKTOP frame (I1/I4/I6/I7/I8 + I3 background) shows the **persistent left rail**; the
  old toolbar `篩選` popover-trigger is gone; NO 套用/重設 button on desktop.
- The Discover rail reads as a sibling of the Library rail (i5-d/i6-d) — same chrome, Discover's
  dimensions.
- **地區 + 串流平台 are LIVE toggleable chips** (no「即將推出」on the FILTER dimensions).
- Per-facet result counts (Mono) next to facet values.
- Preset chips + active-filter summary chip bar + sort live in the RESULTS column (not the rail);
  想要清單·即將推出 inert Epic-13 entry kept in the toolbar.
- I4-D-v2 = collapsed-rail state (篩選(n) + wider grid), not a popover.
- MOBILE (I2-M / I4-M) untouched (batch sheet stays).
- Tokens only; Noto Sans TC CJK; Mono numerics; 44px; placeholders removed.

---

### After you finish (I'll do this via Pencil MCP — token-light)

1. Read each revised frame, screenshot the changed desktop frames, verify against this checklist.
2. Re-run `python3 scripts/export-pen-screenshots.py`; revert re-render noise; stage only the
   genuinely-changed PNGs.
3. Update `ux3-3-1-discover-design.md`: correct the Rule-24 section (region/streaming are LIVE,
   not reserved), record the persistent-instant-rail decision + the adversarial-panel verdict,
   and add the **ux3-3-2 FE acceptance criteria** for the required refinements (debounce numeric
   year/score; `replace:true` on intermediate toggles; coalesce `type='all'` double-fire;
   collapsible rail; reuse the enabled-gated draft-count infra for per-facet counts; demote the
   chip bar to a read/remove summary so it doesn't compete with the rail). Correct the review
   close-out (the earlier "0 corrections" missed the shipped-code divergence).
