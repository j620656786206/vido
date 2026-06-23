# Story ux3-3-1 — Discover v2 design (`.pen` flow-i-discover-v2)

**Epic:** ux3-discover-v2 (UX Redesign Phase 3) · **Status:** ready-for-dev (design not started)
**Owner:** ux-designer (Pencil MCP) · **Type:** design · **FRs:** PH3-M2, PH3-R2 · re-chassis of Epic 11 (P2-010..015); reserves Epic 13 (P3-001..005)
**Status update:** design **complete** — drawn as a persistent instant filter rail, MCP-reviewed (PASS + gap polish) 2026-06-23; `.pen` saved + screenshots staged; **pending combined commit** (then sprint-status → done). See Close-out.

## Story

As the design system,
I want the 探索 Discover **active power-filter tool** redrawn to v2 (per-flow recipe step 1),
So that dev builds Discover v2 against a spec — migrating Epic 11's chips / saved-presets /
instant-search to the v2 shell + tokens, holding the **D3 home/Discover boundary** (Discover
grows NO dashboard), and reserving the **Epic 13 Requests** landing slot — all visually
distinct from the Library own-collection filter rail (ux3-0-6/0-7).

## Context — migrate-existing destination, NOT a redo of the Library rail

Discover is **destination #3** in the ADR's 7-destination IA (`01-nav-ia-decision-adr.md` §1,
D3). Its job is the **active power-filter tool**: discover titles across **local library +
TMDB** by multi-dimension filters, saved presets, and instant search. Its route `/discover`
is unchanged (ADR §"route table": "stays the active power-filter tool"); this story migrates
its *surface* to v2.

**Three things this story must keep straight (all live in `flow-i` today — do NOT reuse them):**

1. **Stale source design** — `flow-i-advanced-search` frames `i1-d` (`NWxok`), `i2` (`TMaw5`),
   `i3` (`i74p2`), `i4-m` (`pjKVZ`) are the **pre-v2 advanced-search** mockups. They predate
   Design Language v2 and `i1-d` invented dimensions (地區 / 最低評分) with no settled backend
   contract. **Do NOT copy them** — redraw to v2 from the Design Language (`01-design-language-v2.md`)
   and the shipped v2 shell (memory: *design-must-conform-to-current-design-system*).
2. **Library filter rail (different feature)** — `flow-i-advanced-search` frames `i5-d`
   (`vpDLh`), `i6-d` (`VwTvy`), `i7-d` (`SgncH`) are the **媒體庫 own-collection filter rail**
   (ux3-0-6, PR #89). That rail filters *what you already have*; Discover power-filters *across
   TMDB + library to find things*. They are **different surfaces** — Discover v2 is its own
   draw, in its own folder, and must read as distinct from the Library rail (ADR §1 row 2:
   "distinct from Discover's power-filter").
3. **D3 boundary guardrail #2** — Discover **grows no dashboard** (no DownloadPanel /
   QBStatusIndicator / ConnectionHistory). It is the discovery side of the D3 split; Home is
   curation, Activity is tasks. (`01-nav-ia-decision-adr.md` D3 guardrail #2; `discover.tsx`
   row: "grows NO dashboard".)

## Design scope — what to draw (in `ux-design.pen`, new flow folder `flow-i-discover-v2`)

> **⚠️ AS-BUILT supersedes this section.** This was the original forward spec (toolbar chips + a filter
> panel/popover). The desktop filter was **revised to a persistent instant left rail** after an adversarial
> UX panel + a shipped-code audit — see **Close-out** for the as-built frames, decision, and ux3-3-2 ACs.

Draw to v2 via Pencil MCP, reusing the shipped v2 shell (`HomeSidebar-v2` instanced with the
active item flipped 首頁 → **探索**; the existing `MobileTabItem` set) and the v2 atoms
migrated per `01-design-language-v2.md` §5.1 (`FilterChip`, `SearchInput`, `SortDropdown`,
`PosterCardV2`). Token-only color, Noto Sans TC for all CJK, JetBrains Mono for numerics, 44px
touch floor, Base UI primitives for popover/sheet (focus-trap/Escape correct by default).

Frames to land (codes are folder-scoped — they do NOT collide with `flow-i-advanced-search`):

- **`I1-D-v2`** — desktop default. Top→bottom: a **search input** (instant, debounced) →
  a **persistent FilterChip row** (Epic 11 E-2; active filters as removable chips + a
  `清除全部` affordance) → a **saved-preset chip row** (Epic 11 E-6; named presets as chips,
  applied = `accent-subtle` active, + a `儲存目前篩選` affordance) → a **sort control**
  (DL-v2: sort lives in the toolbar) → the **results grid** of `PosterCardV2` (rating /
  `vote_average` visible on the card — closes the v2-followups gap that `/search` omitted it).
  A reserved **Requests entry affordance** (see §"Epic 13 reservation") sits in the toolbar,
  inert this epic.
- **`I2-M-v2`** — mobile. **Discover is NOT in the mobile bottom-4** — it is reached via the
  **More sheet** (ADR D1-b / DL-v2 §6.3: "探索 is a desktop power-filter idiom; lives in More").
  Draw: top app bar (探索 title + omnisearch entry) → condensed chip row → a **`篩選` trigger**
  opening the filter sheet → results grid. Bottom tab bar shows the **bottom-4 (首頁·媒體庫·活動·
  下載) with 探索 in More** (no 探索 tab). 44px touch targets.
- **`I3-D-v2`** — **instant-search suggestions dropdown** (Epic 11 E-4): debounced suggestions
  in a `shadow-lg` Base UI popover, **sectioned** 媒體庫 (local) / TMDB, zh-TW results boosted
  first (E-5 zh-TW priority). Each row Noto Sans TC; counts/years Mono.
- **`I4-D-v2`** (+ `I4-M-v2` if the mobile sheet differs materially) — the **multi-dimension
  filter panel/sheet** (Epic 11 E-1): the dimension set Discover exposes (see §"Dimension set"
  decision). Desktop = inline panel or popover; mobile = bottom sheet (`radius-xl` top,
  `overlay-scrim`, reuse the Epic-11 bottom-sheet component — don't rebuild).
- **`I5-D-v2`** — **save-preset** affordance state (name + confirm; Epic 11 E-6) — the
  `儲存目前篩選` interaction.

Four-state standard (N4 — `01-design-language-v2.md` §7; design ALL of them or it doesn't ship):

- **`I6-D-v2`** — **loading skeleton** (grid-shaped skeleton blocks + chip-row skeleton;
  `prefers-reduced-motion` respected).
- **`I7-D-v2`** — **no-result** (filter/search returned nothing). **Distinct from empty** —
  acknowledges the active filter, offers `清除篩選` / `調整搜尋`; never a bare blank.
- **`I8-D-v2`** — **per-section fail-soft** (ADR F3 / B1): TMDB suggestions or availability
  source unavailable → that section degrades to an inline `無法載入，請稍後再試` + `重試`;
  the rest of the page (local results, chips) still renders. Page never hard-fails.

A reusable component (`Component/PresetChip-v2` or a `FilterChip` variant) may be added if the
preset chip's anatomy (name + applied-state + remove) diverges from the plain filter chip;
prefer a `FilterChip` instance override before forking a new component.

## Key design decisions (resolved with rationale — flag the one capability check)

1. **Dimension set — Discover ≠ Library rail (the central distinction).** The Library rail
   (ux3-0-6) deliberately exposes only own-collection dimensions (類型 / 類別 / 年份 / 未匹配)
   and **dropped `i1-d`'s invented 地區 / 最低評分**. Discover is the *power-filter*, so it
   surfaces the **fuller Epic 11 dimension set** (E-1: genre, year, **region 地區**, **rating
   評分**, **streaming platform 串流平台**). **Rule 24 capability-honor — VERIFY each dimension
   against the shipped Epic 11 filter engine before drawing it as active**: only draw a
   dimension the backend can actually filter on; if 地區 / 評分 / 串流平台 are not backed by a
   real query parameter today, either omit them or draw them explicitly disabled with a
   reserved note (do NOT re-introduce `i1-d`'s unbacked invented fields as if live). The FE
   story (ux3-3-2) inherits the verified set.
   **[RESOLVED 2026-06-23 — capability audit].** The shipped `/discover` route already power-filters
   on **all five** dimensions (類型 · 年份 · 評分 · 地區 · 串流平台) — region / rating / streaming are
   backend-backed and **LIVE today**, not reserved. The first Discover-v2 draft (popover) demoted
   地區 / 串流平台 to `text-disabled「即將推出」`, which was a **regression vs. shipped code**; the
   rail revision restores all five as **live, toggleable chips with per-facet Mono counts**. ux3-3-2
   must still confirm the exact `/discover` query-param names and that 評分 maps to a real rating-min
   param (carry-forward — if any is unbacked, demote only that one).
2. **`vote_average` on result cards (v2-followups gap).** Discover result cards use
   `PosterCardV2` with the rating visible — Discover is where the rating-sort/rating-filter
   lives, so the value must be on the card. (Memory: *v2-followups-filter-rail-and-search-rating*
   — `/search` List query omitted `vote_average`; the FE/BE story must confirm the Discover
   list query returns it.)
3. **Epic 13 Requests reservation (PH3-R2).** Discover **reserves** the Requests landing entry
   but does **not** build the flow (full Requests = Epic 13, backlog). Draw a quiet, inert
   entry affordance (mirrors Home's reserved continue-watching slot pattern, ux3-1-3) — never a
   broken/empty surface; becomes live when Epic 13 lands. (ADR: "host Epic 13 requests later".)
4. **Shell reuse, no fork.** `HomeSidebar-v2` instanced (active → 探索); mobile bottom bar keeps
   the bottom-4 with 探索 in More (no new tab). Reuse Epic-11 bottom-sheet for the mobile filter
   sheet. Atoms migrate to v2 per DL-v2 §5.1 (Noto Sans TC labels, `accent-subtle` active chip,
   44px, `focus-ring`) — no new palette, no new shell.

## Acceptance Criteria

1. **Given** the v2 Design Language + shipped v2 shell, **when** Discover is drawn, **then** all
   frames land in a **new** `flow-i-discover-v2` folder and the stale `i1-d/i2/i3/i4-m` and the
   Library-rail `i5-d/i6-d/i7-d` (in `flow-i-advanced-search`) are **left untouched** (no reuse,
   no mutation).
2. **Given** Epic 11's feature set, **then** the design covers: instant-search + sectioned
   suggestions (E-4, zh-TW boosted E-5), persistent removable filter chips + clear (E-2),
   multi-dimension filter panel/sheet (E-1, dimension set per decision #1), saved-preset chips +
   save affordance (E-6), and toolbar sort.
3. **Given** N4, **then** all four states are drawn: default (`I1`), loading skeleton (`I6`),
   no-result distinct-from-empty (`I7`), per-section fail-soft (`I8`).
4. **Given** the D3 boundary, **then** Discover shows **no dashboard elements** (no
   DownloadPanel / QBStatus / ConnectionHistory) — guardrail #2 visibly held.
5. **Given** PH3-R2, **then** a reserved, inert Epic 13 Requests entry affordance is present and
   reads as "coming later", never as a broken control.
6. **Given** mobile IA (ADR D1-b), **then** the mobile frame reaches Discover via **More**
   (bottom-4 = 首頁·媒體庫·活動·下載, no 探索 tab) and the filter sheet reuses the Epic-11
   bottom sheet; all touch targets ≥ 44px.
7. **Given** v2 enforcement, **then** color is token-only (no hex literals), all CJK is Noto
   Sans TC (TY-1), numerics are JetBrains Mono, colored body text uses `*-text` AA variants
   (TC-2), and `text-disabled` carries no load-bearing text (TC-1).
8. **Given** the UX screenshots workflow (CLAUDE.md), **then** `scripts/export-pen-screenshots.py`
   `SCREENS` dict is extended with every new `flow-i-discover-v2` node ID → code, screenshots are
   regenerated, and **only genuinely-changed PNGs** are committed alongside the `.pen` (regen is
   non-deterministic).

## Tasks / Subtasks (designer)

- [ ] (AC #1) Pencil MCP: `get_editor_state(include_schema:true)`; confirm v2 shell components
      (`HomeSidebar-v2`, `MobileTabItem`) + v2 atoms; create `flow-i-discover-v2` frames.
- [ ] (AC #1, decision #1) **Rule-24 capability audit** of the Epic 11 filter engine — list the
      dimensions actually queryable today; mark 地區 / 評分 / 串流平台 active vs reserved-disabled.
- [ ] (AC #2) Draw `I1-D-v2` (default: search → chips → presets → sort → grid) + `I3-D-v2`
      (suggestions) + `I4-D-v2` (+ `I4-M-v2`) filter panel/sheet + `I5-D-v2` save-preset.
- [ ] (AC #3) Draw `I6-D-v2` skeleton, `I7-D-v2` no-result, `I8-D-v2` per-section fail-soft.
- [ ] (AC #4, #5) Verify no dashboard elements; place the inert Epic 13 Requests entry.
- [ ] (AC #6) Draw `I2-M-v2` (Discover-via-More, bottom-4 unchanged, 44px, Epic-11 sheet reuse).
- [ ] (AC #7) Token-lint pass: no literals, Noto Sans TC CJK, Mono numerics, AA color rules.
- [ ] (AC #8) Update `SCREENS` dict; `python3 scripts/export-pen-screenshots.py`; commit `.pen` +
      only-changed PNGs together (`feat(ux3-3-1): Discover v2 design (.pen flow-i-discover-v2)`).

## Dev Notes

- **Design Language v2:** `_bmad-output/planning-artifacts/ux-redesign/01-design-language-v2.md`
  — tokens §2 (incl. status→token §2.5), type §3 (TY-1/TY-2), atoms §5.1, shell §6, four-state §7, a11y §8.
- **Nav/IA ADR (D3 + route table + mobile):**
  `_bmad-output/planning-artifacts/ux-redesign/01-nav-ia-decision-adr.md` — Discover = active
  power-filter, grows no dashboard (D3 guardrail #2), hosts Epic 13 later; 探索 in More on mobile.
- **Phase-3 map:** `_bmad-output/planning-artifacts/ux-redesign/03-phase3-destination-epic-map.md`
  §1 row 3 (Discover v2), §3 (flow I → 探索), §6 (per-flow recipe).
- **Epic 3 skeleton + Epic 11 source:** `_bmad-output/planning-artifacts/epics.md` §"Epic 3:
  ux3-discover-v2" + `epics/epic-11-advanced-search-filter.md` (E-1..E-6).
- **Distinct-from precedent (Library rail):** `ux3-0-6-library-filter-rail-design.md` — what was
  deliberately scoped OUT of the rail (地區/最低評分) and why; Discover re-includes per backend capability.
- **Reserved-slot precedent:** `ux3-1-3-continue-watching-slot.md` — how to draw an inert
  "later" affordance without a broken/empty tile (apply to the Epic 13 Requests entry).
- **Memory:** *design-must-conform-to-current-design-system* (don't copy stale `flow-i i1-d`);
  *v2-followups-filter-rail-and-search-rating* (`vote_average` must reach the card).

### Project Structure Notes

- Design-only story: edits `ux-design.pen` + `_bmad-output/screenshots/flow-i-discover-v2/` +
  `scripts/export-pen-screenshots.py` (`SCREENS`). No app code, no cross-stack split (design story).
- Route `/discover` already exists (`apps/web/src/routes/discover.tsx`); this story does not touch it.

### Time-dependent visual coverage

- N/A — design-only story; adds/modifies no `apps/web/src/components/**/*.{ts,tsx}`. (Rule 23
  applies to code stories; the downstream FE story ux3-3-2 re-evaluates if any Discover component
  reads wall-clock time.)

### Discovery Triage

- **YES — out-of-scope work surfaced, all triaged:**
  - **③ backlog-with-carry-forward-link** — Epic 11 filter-engine **capability audit** for
    地區/評分/串流平台 (decision #1): if any dimension is unbacked, file/confirm a backend
    sprint-status entry at audit time rather than drawing an unbacked field live (Rule 24).
  - **③** — `vote_average` on the Discover **list** query (v2-followups gap): tracked for the FE
    story ux3-3-2 / its backend confirm; this design only ensures the card *slot* shows rating.
  - Neither blocks the design deliverable; both carry into ux3-3-2 (frontend).

### References

- [Source: _bmad-output/planning-artifacts/ux-redesign/01-design-language-v2.md#§2-§8]
- [Source: _bmad-output/planning-artifacts/ux-redesign/01-nav-ia-decision-adr.md#D3]
- [Source: _bmad-output/planning-artifacts/ux-redesign/03-phase3-destination-epic-map.md#§1,§3,§6]
- [Source: _bmad-output/planning-artifacts/epics.md#Epic-3-ux3-discover-v2]
- [Source: _bmad-output/planning-artifacts/epics/epic-11-advanced-search-filter.md#E-1..E-6]

## Close-out (drawn + rail-revised + reviewed 2026-06-23 — `.pen` saved; pending combined commit → then Status: done)

### Design revision: popover → persistent instant LEFT RAIL (desktop)

The first v2 draft put the multi-dimension filter in a toolbar-button + floating popover with a batch
`套用篩選` button, and demoted 地區 / 串流平台 to `text-disabled「即將推出」`. An adversarial UX panel
(three lenses) scored a **persistent instant left rail 9 / 9 / 9** and it survived the refuter;
decisively, the **shipped `/discover` is ALREADY an instant rail with region / rating / streaming LIVE**,
so the popover draft was a **regression vs. shipped code**. The desktop frames were revised **in place**
(no new folder, no delete-and-recreate) to a **3-column shell**: v2 nav sidebar → persistent 264px filter
rail (`$bg-primary`, right `$border-subtle` — a sibling of the 媒體庫 Library rail i5-d/i6-d) → results
column. **Instant apply — no `套用` / `重設` button on desktop.**

### Frames landed in `flow-i-discover-v2` (folder-scoped codes; do NOT collide with `flow-i-advanced-search`)

- `fxCVk` **I1-D-v2** (default): rail EXPANDED, all 5 dims LIVE with per-facet Mono counts — 類型(動作✓ 340…)
  · 年份(2020s✓ 128…) · 評分(7+✓ 412…) · **地區(日本✓ 410…)** · **串流平台(Netflix✓ 540…)**; rail header
  篩選 + badge `5` + collapse chevron; `清除全部篩選` footer. Results column: search → active-filter summary
  chips + 清除全部 → preset chips 週末動作片/經典科幻 + 儲存目前篩選 → toolbar (評分排序 sort +
  **想要清單·即將推出** inert Epic-13 entry) → PosterCard-v2 grid with ★ + rating on every card.
- `m4fY7c` **I4-D-v2 · 篩選 rail（收合）**: rail COLLAPSED to a `篩選` + badge `3` toolbar button; grid
  reflowed WIDER (5 columns) to demonstrate the width reclaim. Old popover + 套用 button deleted. (Models
  Library i6-d.)
- `m0Zew` **I3-D-v2**: instant-search suggestions popover (sectioned 媒體庫 / TMDB) over the 3-column rail
  background.
- `S3qke` **I7-D-v2** (no-result, distinct-from-empty): persistent rail (動作 + 2020s active) + search-x +
  找不到相符的結果 + active-filter echo (動作 · 2020s) + 清除篩選 / 調整搜尋.
- `KdnVw` **I8-D-v2** (per-section fail-soft): persistent rail + 媒體庫結果 renders 4 cards + TMDB section
  inline `$error-tint`「TMDB 服務暫時無法連線，媒體庫結果不受影響」+ 重試. (串流平台 is now a live rail
  dimension, so the old availability-section was dropped.)
- `YYEBd` **I6-D-v2**: loading skeleton — rail-shaped skeleton sections + grid skeleton (3-column).
- `nLrzc` **I5-D-v2**: save-preset dialog (name + preview + actions) — unchanged.
- `hi6WD` **I2-M-v2** + `kzzjc` **I4-M-v2**: mobile UNCHANGED — top app bar 探索 + search → condensed chip
  row → `篩選` button → bottom SHEET with BATCH apply (`套用篩選（N 部結果）`, radius-xl + overlay-scrim);
  bottom tab bar = **首頁·媒體庫·活動·下載·更多, NO 探索 tab** (探索 via More, ADR D1-b). A transient sheet
  = batch is the correct mobile pattern; the rail revision is desktop-only.

### MCP review of the rail revision (ux-designer, 2026-06-23) — PASS, 1 polish correction

The earlier "PASS, 0 corrections" note was for the popover draft and **missed the shipped-code divergence**
(region / streaming should have been live; desktop should have been a rail). The in-app **rail redraw was
re-reviewed against the shipped code**, and the substantive ACs were **already satisfied by the redraw** — no
regression fixes needed: all 5 desktop frames (I1 expanded · I4 collapsed · I6 skeleton · I7 · I8) + the I3
background carry the persistent rail; I1's toolbar was already clean (sort + inert 想要清單 only — no stale
`篩選` popover-trigger); I7 already had its rail; 地區 / 串流平台 are live toggleable chips with per-facet
Mono counts; tokens-only; Noto Sans TC CJK + Mono numerics; `vote_average` on every card; Epic-13 entry inert;
mobile untouched; no `套用` / `重設` on desktop.

**One polish correction applied via `batch_design`:** unified the I1 rail `gap` **20 → 16** — 20 was an
off-scale outlier (token spacing scale = 4 / 8 / 12 / **16** / 24 / 32) and the other rails (I6/I7/I8) were
already `gap-lg 16`. No other change.

### Design-system updates (this revision — Alexyu updated alongside the redraw)

Three reference frames were updated to document the new pattern (verified via MCP screenshot, no issues):

- **Design Language v2** (`V2Kez`) — new §7「篩選 Rail 型錄 / FilterRail Pattern」: 264px persistent instant
  rail, same source as the 媒體庫 Library rail, instant on chip toggle, **no 套用/重設 button**, Mono facet
  counts, 地區/串流平台 = live dimensions, mobile = bottom sheet (batch).
- **Component Library** (`sJzat`) — new `filter-controls-v2` group (SearchInput / SortDropdown / FilterChip
  default+active / **FacetCountChip** 動作 340) + `content-cards-v2`.
- **Design System Reference** (`8SSzc`) — new §7「v2 元件補遺」with cross-pointers to the two catalogs above.

### Screenshots / commit

- `SCREENS` dict maps all 9 `flow-i-discover-v2` frames (stable IDs — redraw edited in place); re-exported
  after the gap fix.
- ✅ `ux-design.pen` **saved to disk** (`git` shows it modified). `export-pen-screenshots.py` run; full-regen
  re-render noise reverted via `git checkout`, keeping **only genuinely-changed PNGs**: the **9
  `flow-i-discover-v2/*`** (new) + the **3 changed `design-system/{design-language-v2, component-library,
  design-system-reference}.png`**. (`flow-k-activity-v2/a1-d` transiently skipped on export — "no image data",
  unrelated; its committed PNG untouched.)
- **Pending combined commit** (per Alexyu — bundle everything): `.pen` + 9 discover PNGs + 3 design-system
  PNGs + `export-pen-screenshots.py` + this doc + the two prompt docs + `sprint-status.yaml` →
  `feat(ux3-3-1): Discover v2 design — persistent instant filter rail (.pen flow-i-discover-v2)`.

### ux3-3-2 (frontend) acceptance criteria — required refinements from the rail decision

1. **Persistent instant rail**, not a batch popover; **collapsible** (chevron → `篩選(n)` collapsed state,
   grid reclaims width) — converge with the 媒體庫 Library rail chrome.
2. **All 5 dimensions live**: 類型 · 年份 · 評分 · **地區** · **串流平台** (confirm exact `/discover`
   query-param names; confirm 評分 maps to a real rating-min param — if unbacked, demote only 評分).
3. **Per-facet result counts** next to each facet value — reuse the existing **enabled-gated draft-count**
   infra (do not over-fetch).
4. **Debounce** numeric year/score inputs; on intermediate chip toggles use **`replace:true`** (don't stack
   history); **coalesce the `type='all'` double-fire**.
5. **Demote the active-filter chip bar to a read/remove summary** (lighter than the rail's chips) so it does
   not read as a second competing editor.
6. **`vote_average` on result cards** — confirm the Discover list query returns it (v2-followups gap;
   `/search` List omitted it).
7. Reuse Epic 11 filter-engine / presets / suggestions backend (no new BE expected — confirm). E2E reuses
   `tests/support/helpers/seed-helpers.ts` real seeding (no data-dependent self-skips).
