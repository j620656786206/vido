# Story ux3-4-1 — Downloads v2 design (`.pen` flow-d-downloads-v2)

**Epic:** ux3-downloads-v2 (UX Redesign Phase 3, Epic 4) · **Status:** done (design merged, PR #107)
**Owner:** ux-designer (Pencil MCP) · **Type:** design · **FRs:** PH3-M3 · re-chassis of Epic 14 (P3-010 done, P3-012 SSE) — "the downloads deep page, seen and controlled"

## Story

As the design system,
I want the **下載 Downloads** retained **deep operations page** redrawn to v2 — with the
**card actions** it has never had and **SSE-fed live progress** replacing the polling storm
(per-flow recipe step 1),
So that dev builds Downloads v2 against a spec — migrating the existing qBittorrent download
dashboard to the v2 shell + tokens, adding per-download **pause / resume / remove** + **batch
ops** + **pagination**, holding the **D4-1 boundary** (downloads keep a **dedicated deep page**,
NOT crammed into the Activity hub), and entered from the Activity summary row already shipped
(ux3-2-x).

## Context — migrate-existing destination, the RETAINED deep page (not a hub row)

Downloads is **destination #5** in the ADR's 7-destination IA (`01-nav-ia-decision-adr.md` §1,
D4-1). Its job is the **deep operations surface** for the acquisition queue (qBittorrent
monitoring today): list / filter / inspect / **act on** in-flight + completed downloads. Route
`/downloads` is unchanged (`apps/web/src/routes/downloads.tsx` exists); this story migrates its
*surface* to v2 and specs the missing controls.

**Why a deep page at all (ADR D4-1, the central IA decision):** downloads **KEEP a dedicated
deep page** — "Epic 14 v2's **pagination / batch-ops** needs don't fit a hub." The Activity hub
(ux3-activity-hub, done) carries only a **downloads summary row that links into this page**; it
does NOT host the queue. So this design is the full page the summary row points at — do not
duplicate the hub's summary view here.

**Three things this story must keep straight:**

1. **Stale v3 design — `flow-d-downloads` frames `d1-d` (`rWvuG`), `d1-m` (`cZd7j`), `d2-d`
   (`3ULXd`), `d3-m` (`tqHK9`)** are the **pre-v2** download mockups. They predate Design
   Language v2. **Do NOT copy / mutate them** — redraw to v2 from the Design Language
   (`01-design-language-v2.md`) and the shipped v2 shell, in a **new** folder
   `flow-d-downloads-v2` (memory: _design-must-conform-to-current-design-system_; mirrors how
   home/activity/discover each got a `-v2` folder).
2. **The documented pain this design must fix** (`00-redesign-brief.md` Part 4 D1 / row 7):
   **"Console bursts, polling storms, no card actions in design."** So the v2 design MUST (a)
   show **live progress fed by real-time push (SSE), not polling** (P3-012 / Epic 14 H-1), and
   (b) introduce the **card actions** the design has never had. Drawing another polling-shaped,
   action-less page would re-ship the pain.
3. **D4-1 boundary — deep page ≠ Activity hub.** Activity is the unified *task* hub (parse /
   subtitle-batch / scan / AI) that **links** here; Downloads is the high-volume *acquisition*
   queue with its own pagination + batch ops. Keep them distinct; the summary-row entry
   (ux3-2-x) is the only overlap and it already exists.

## Design scope — what to draw (in `ux-design.pen`, new flow folder `flow-d-downloads-v2`)

Draw to v2 via Pencil MCP, reusing the shipped v2 shell (`HomeSidebar-v2` instanced with the
active item flipped to **下載**; the existing `MobileTabItem` set with **下載 active in the
bottom-4**) and the v2 atoms per `01-design-language-v2.md` §5.1. Token-only color, Noto Sans TC
for all CJK, **JetBrains Mono for ALL numerics** (%, speed ↓↑, ETA, size, counts), 44px touch
floor, Base UI primitives for menu/confirm (focus-trap / Escape correct by default).

Frames to land (codes are folder-scoped — they do NOT collide with `flow-d-downloads`):

- **`D1-D-v2`** — desktop default. Nav sidebar → content column. Top: a **status-filter
  toolbar** exposing the **six live filter values** the backend actually supports —
  `全部 / 下載中 / 已暫停 / 已完成 / 做種 / 錯誤` (verified against `download_handler.go` `filter`
  param; do NOT invent statuses) + a count per filter (Mono). Body: a **list of download cards**.
  Each card: title (Noto Sans TC, 2-line CJK clamp), a **source indicator** (qBittorrent;
  NZBGet **reserved-inert** per decision #4), a **live progress bar + `xx.x%`** (Mono), **speed
  ↓ / ↑** (Mono), **ETA** (Mono), **size** (Mono), a **status token** (DL-v2 §2.5 status→token),
  and the **card actions** (decision #2). Bottom: a **pagination control** (the deep-page
  reason-to-exist).
- **`D2-D-v2`** — **batch-select mode**: per-card checkboxes + a **batch action bar**
  (`全選` · `批次暫停` · `批次移除` · `已選 N 項`). The other deep-page reason-to-exist.
- **`D3-D-v2`** — **card-action affordance** (the headline "no card actions" gap): the per-card
  action set — `暫停` / `繼續` / `移除（保留檔案）` / `移除（連同檔案刪除）` — as an inline icon
  cluster and/or an overflow menu. The destructive `連同檔案刪除` gets a **confirm** step
  (Base UI dialog). Pause vs Resume is state-dependent (one shows per card state).
- **`D1-M-v2`** — **mobile**. Single-column condensed card list. **Bottom tab bar = the bottom-4
  with 下載 ACTIVE** (`首頁 · 媒體庫 · 活動 · 下載 · 更多`) — Downloads **IS** a primary mobile tab
  (contrast Discover, which lives in More). Card actions reachable on mobile (swipe action or
  overflow menu), 44px targets. Top app bar: 下載 title + the status filter as a condensed
  control.

Four-state standard (N4 — `01-design-language-v2.md` §7; design ALL of them or it doesn't ship):

- **`D4-D-v2`** — **loading skeleton** (card-shaped skeleton rows + toolbar skeleton;
  `prefers-reduced-motion` respected).
- **`D5-D-v2`** — **empty: no downloads**. **Distinct from a load failure** — a calm
  `目前沒有下載任務` with a quiet affordance (e.g. `前往探索` to find something), never a bare
  blank.
- **`D6-D-v2`** — **connection / per-section fail-soft** (ADR F3 / B1): **qBittorrent
  unreachable** → the queue area degrades to an inline `無法連線到 qBittorrent` + `重試` +
  `前往設定`; the **shell + nav still render** (page never hard-fails). This state matters
  more here than elsewhere — the entire page depends on the downloader connection.

A reusable component (`Component/DownloadCard-v2`) should be added — the card's anatomy
(title + source + progress + speed/ETA/size + status token + action cluster + select checkbox)
is rich and repeated; spec it once as a component with the action/select/state variants.

## Key design decisions (resolved with rationale — flag the two capability checks)

1. **Retained deep page, not a hub/popover (ADR D4-1).** Pagination + batch-ops are the
   reason-to-exist; both MUST be in the design. Route `/downloads` unchanged; the Activity
   summary-row entry (ux3-2-x, shipped) is the canonical way in — keep the visual hand-off
   consistent (no second, competing entry point).
2. **⚠️ Card actions — Rule 24 capability-honor + Rule 15 (THE critical flag #1).** **Verified
   2026-06-30:** `download_handler.go` `RegisterRoutes` registers **only three GET routes** —
   `GET /downloads`, `/downloads/counts`, `/downloads/:hash`. **There are NO action endpoints**
   (no pause / resume / remove). **Update 2026-06-30 (ux3-4-2 capability audit):** the deeper
   check found the qBittorrent **client is fully read-only too** — pause/resume/delete exist at
   **NO layer** (client / service / handler), so this is a **full vertical build**, not merely an
   unwired route (a stronger form of the Rule 15 "method-exists ≠ wired" gap — here the method
   doesn't exist at all). So: **draw the card actions as the target v2 state** (design-ahead is
   correct here), but the **FE story is HARD-BLOCKED** on the **BE story `ux3-4-2-downloads-actions-be`**
   that builds `POST /api/v1/downloads/:hash/pause`,
   `.../resume`, and `DELETE /api/v1/downloads/:hash` (with a `deleteFiles` option). Recorded in
   Discovery Triage as the cross-stack BE half. Do **not** draw any action as if it is already
   live — they are all net-new backend.
3. **⚠️ SSE-fed live progress, NOT polling — P3-012 / Epic 14 H-1 (critical flag #2).**
   **Verified 2026-06-30:** the SSE hub broadcasts `scan_*` / `subtitle_*` / `notification`
   events but **no download events** (project-context §8). The page **polls** today — the
   "polling storms" pain. The v2 design shows **real-time** progress / speed / ETA, which
   requires **new backend download SSE events**. The spec must state "real-time push, not a
   poll loop" so dev builds SSE (Epic 14 H-1), and the lazy-SSE-connection rule (project-context
   §8 frontend pattern) applies on the FE side. Recorded in Discovery Triage.
4. **qBittorrent only; NZBGet (H-2) reserved-inert.** Epic 14 H-4 imagines a unified qBT + NZBGet
   view with source indicators, but **NZBGet is not built** (H-2 deferred). Rule 24: draw the
   **qBT** source live; a **source-indicator slot may be reserved** (single-source / inert) to
   host NZBGet later — never draw NZBGet as live. Mirrors the reserved-slot pattern (Epic 13
   Requests in ux3-3-1; continue-watching in ux3-1-3).
5. **Lifecycle consistency (N1).** Download status tokens (`下載中 xx%` / `已暫停` / `已完成` /
   `做種` / `錯誤`) use the **same status→token mapping** (DL-v2 §2.5) as the poster lifecycle
   badge (ux3-0-1/0-2) and the unified lifecycle moat (`想要 → 下載中 x% → 整理中 → 已入庫`). No
   bespoke download palette.
6. **Shell reuse, no fork.** `HomeSidebar-v2` instanced (active → 下載); mobile bottom bar keeps
   the **bottom-4 with 下載 active**. Atoms migrate to v2 per DL-v2 §5.1 (Noto Sans TC labels,
   status tokens, 44px, `focus-ring`) — no new palette, no new shell.

## Acceptance Criteria

1. **Given** the v2 Design Language + shipped v2 shell, **when** Downloads is drawn, **then** all
   frames land in a **new** `flow-d-downloads-v2` folder and the stale `d1-d / d1-m / d2-d / d3-m`
   (in `flow-d-downloads`) are **left untouched** (no reuse, no mutation).
2. **Given** the deep-page feature set, **then** the design covers: the **six live status
   filters** (全部/下載中/已暫停/已完成/做種/錯誤), download **cards** (title / source / live
   progress % / speed ↓↑ / ETA / size / status token), **card actions** (暫停/繼續/移除（保留檔案）/
   移除（連同檔案刪除）+ destructive-confirm), **batch-select + batch-ops** bar, and **pagination**.
3. **Given** N4, **then** all four states are drawn: default (`D1`), loading skeleton (`D4`),
   **empty-no-downloads distinct-from-failure** (`D5`), and **qBittorrent-unreachable per-section
   fail-soft** (`D6`).
4. **Given** the D4-1 boundary, **then** the design is the **retained deep page** with pagination
   + batch-ops, consistent with the **Activity summary-row** entry (ux3-2-x) and **not**
   duplicated into the hub.
5. **Given** Rule 24 / Rule 15 capability-honor, **then** the design **records both backend
   gaps** — (a) **card actions need net-new endpoints** (only GETs exist today) and (b) **live
   progress needs net-new download SSE events** (hub has none) — as the **cross-stack BE half**;
   no action is drawn as already-live; **NZBGet** is reserved-inert, never live.
6. **Given** mobile IA, **then** Downloads is the **mobile bottom-4 ACTIVE tab**
   (首頁·媒體庫·活動·下載, NOT More), card actions are reachable on mobile (swipe / overflow), and
   all touch targets ≥ 44px.
7. **Given** v2 enforcement, **then** color is token-only (no hex literals), all CJK is Noto Sans
   TC (TY-1), **all numerics** (%, speed, ETA, size, counts) are JetBrains Mono, colored body
   text uses `*-text` AA variants (TC-2), and `text-disabled` carries no load-bearing text (TC-1).
8. **Given** the UX screenshots workflow (CLAUDE.md), **then** `scripts/export-pen-screenshots.py`
   `SCREENS` dict is extended with every new `flow-d-downloads-v2` node ID → code, screenshots
   are regenerated, and **only genuinely-changed PNGs** are committed alongside the `.pen` (regen
   is non-deterministic).

## Tasks / Subtasks (designer)

- [ ] (AC #1) Pencil MCP: `get_editor_state(include_schema:true)`; confirm v2 shell components
      (`HomeSidebar-v2`, `MobileTabItem`) + v2 atoms; create the `flow-d-downloads-v2` frames.
- [ ] (AC #2, #5) **Rule-24/15 capability audit** of the download backend — confirm the live
      read shape (list/counts/:hash) and that **no** action/SSE surface exists yet; draw actions
      + live progress as target state and record the BE gap (do NOT draw them as live-wired).
- [ ] (AC #2) Draw `D1-D-v2` (filter toolbar → card list → pagination) + `D2-D-v2` (batch-select +
      batch bar) + `D3-D-v2` (card-action cluster/menu + destructive-confirm). Spec
      `Component/DownloadCard-v2` with its variants.
- [ ] (AC #3) Draw `D4-D-v2` skeleton, `D5-D-v2` empty-no-downloads, `D6-D-v2` qBT-unreachable
      fail-soft.
- [ ] (AC #4) Verify the deep-page ↔ Activity summary-row hand-off reads consistently; no second
      entry point.
- [ ] (AC #6) Draw `D1-M-v2` (mobile, **下載 active in bottom-4**, card actions reachable, 44px).
- [ ] (AC #7) Token-lint pass: no literals, Noto Sans TC CJK, **Mono numerics**, AA color rules.
- [ ] (AC #8) Update `SCREENS` dict; `python3 scripts/export-pen-screenshots.py`; commit `.pen` +
      only-changed PNGs together (`feat(ux3-4-1): Downloads v2 design (.pen flow-d-downloads-v2)`).

## Dev Notes

- **Design Language v2:** `_bmad-output/planning-artifacts/ux-redesign/01-design-language-v2.md`
  — tokens §2 (status→token §2.5 — reuse for download status), type §3 (TY-1/TY-2 + **Mono
  numerics**), atoms §5.1, shell §6, four-state §7, a11y §8.
- **Nav/IA ADR (D4-1 deep page):** `…/01-nav-ia-decision-adr.md` — downloads KEEP a dedicated
  deep page (pagination/batch-ops don't fit the hub); Activity summary row links in.
- **Phase-3 map:** `…/03-phase3-destination-epic-map.md` §1 row 5 (Downloads v2), §3 (flow D →
  下載; Epic 14 v2), §6 (per-flow recipe step 1 = design-first, design-ahead allowed).
- **Redesign brief pain:** `…/00-redesign-brief.md` Part 4 D1 / row 7 — console bursts, polling
  storms, no card actions.
- **Epic 4 skeleton + Epic 14 source:** `epics.md` §"Epic 4: ux3-downloads-v2" (L223) +
  `epics/epic-14-download-management-v2.md` (H-1 SSE / H-2 NZBGet / H-3 notifications / H-4
  unified dashboard; P3-010 done from Epic 4).
- **Backend capability anchors (audit done 2026-06-30):** `apps/api/internal/handlers/download_handler.go`
  — only `GET /downloads` (`:269`), `/downloads/counts` (`:270`), `/downloads/:hash` (`:271`)
  registered; **no action routes**; `filter` values `all/downloading/paused/completed/seeding/error`
  (`:73`). SSE hub (`apps/api/internal/sse/`) has **no** download events (project-context §8).
- **Reserved-slot precedents:** `ux3-3-1-discover-design.md` (Epic 13 Requests inert entry);
  `ux3-1-3-continue-watching-slot.md` (inert "later" affordance) — apply to NZBGet source slot.
- **Activity entry precedent:** `ux3-2-1-activity-design.md` / ux3-2-3 — the downloads summary
  row that links here (keep the hand-off consistent).
- **Memory:** _design-must-conform-to-current-design-system_ (don't copy stale `flow-d` v3);
  _pencil-mcp-edits-need-manual-save_ (Cmd+S the `.pen` before screenshot regen).

### Project Structure Notes

- Design-only story: edits `ux-design.pen` + `_bmad-output/screenshots/flow-d-downloads-v2/` +
  `scripts/export-pen-screenshots.py` (`SCREENS`). No app code; no cross-stack split (design story).
- Route `/downloads` already exists (`apps/web/src/routes/downloads.tsx`); this story does not touch it.
- **Cascade ordering:** Downloads is **PH3-M3** — sequenced after `ux3-home-v2` (M1) + `ux3-discover-v2`
  (M2) epics close. This **design** story is authored **design-ahead** (per-flow recipe step 1
  explicitly allows it — precedent: ux3-3-1 drawn ahead of home-v2 epic close). The downstream BE/FE
  build stories should not start until the M1/M2 epics close and the BE gap (below) is sequenced.

### Time-dependent visual coverage

- N/A — design-only story; adds/modifies no `apps/web/src/components/**/*.{ts,tsx}`. (Rule 23
  applies to code stories; the downstream FE story re-evaluates if any Downloads component reads
  wall-clock time — ETA is server-supplied, not a client clock read.)

### Discovery Triage

- **YES — out-of-scope (backend) work surfaced, all triaged. These are the cross-stack BE half
  that gates the FE build (NOT this design story, which is unblocked):**
  - **③ backlog-with-carry-forward-link — download CARD-ACTION endpoints do not exist.** Verified:
    `download_handler.go` registers only 3 GET routes; pause/resume/remove have **no HTTP route**
    (Rule 15 "method-exists ≠ wired"). File/confirm a **BE story** (`ux3-4-2-downloads-actions-be`,
    suggested) to add `POST /downloads/:hash/pause|resume` + `DELETE /downloads/:hash` (deleteFiles
    option), wiring the existing qBittorrent client methods. Blocks the FE build (`ux3-4-3`), not
    this design.
  - **③ — download SSE events do not exist (P3-012 / Epic 14 H-1).** Verified: the SSE hub has no
    download events; the page polls. File/confirm the **download SSE** work (same BE story or a
    sibling `ux3-4-2…-sse`) so the v2 page is real-time, not a poll loop. Blocks the FE build.
  - **Deferred Epic 14 scope (NOT prerequisites — do not pull in):** H-2 (NZBGet source), H-3
    (completion notifications/webhook), H-4 (unified multi-source dashboard) are **additive**
    features beyond the v2 restyle; they stay backlog and must not gate Downloads v2.
- Reference: `project-context.md` Rule 24 (Discovery Triage) + Rule 15 (method-exists ≠ wired);
  origin: this story's capability audit (2026-06-30).

### References

- [Source: _bmad-output/planning-artifacts/ux-redesign/01-design-language-v2.md#§2-§8]
- [Source: _bmad-output/planning-artifacts/ux-redesign/01-nav-ia-decision-adr.md#D4-1]
- [Source: _bmad-output/planning-artifacts/ux-redesign/03-phase3-destination-epic-map.md#§1-row5,§3,§6]
- [Source: _bmad-output/planning-artifacts/ux-redesign/00-redesign-brief.md#Part-4-D1]
- [Source: _bmad-output/planning-artifacts/epics.md#Epic-4-ux3-downloads-v2]
- [Source: _bmad-output/planning-artifacts/epics/epic-14-download-management-v2.md#H-1..H-4]
- [Source: apps/api/internal/handlers/download_handler.go#RegisterRoutes] — capability audit (GETs only)
- [Source: project-context.md#§8-SSE-Hub] — hub event types (no download events today)

## Change Log

| Date       | Change                                                                                                                                          |
| ---------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-06-30 | Design story authored (SM create-story), design-ahead per per-flow recipe step 1. Capability audit baked in: card actions + download SSE are net-new BE (cross-stack BE half flagged in Discovery Triage). Status → ready-for-dev (design not started). |
| 2026-07-02 | Design DONE, merged PR #107. Built by Pencil inline-agent, Sally adversarial review ×2, v2.1 rework ratified by Alexyu: desktop single-column List default + D7 sortable Table view + list toolbar (in-place batch-bar morph); mobile sheet-first actions (D8) + long-name split (2-line title + TechBadge row) + D9 detail sheet + D10 sort sheet; AA count/hint fixes; SSE + net-new-BE spec notes on canvas. 11 frames + Component/DownloadCard-v2; SCREENS +11; AC1-8 met; full CI green. Deferred: sort-sheet 狀態/名稱 options, Table-view skeleton. Status → done. |
