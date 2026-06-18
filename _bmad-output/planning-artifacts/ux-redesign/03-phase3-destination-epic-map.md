---
status: 'in-progress'
phase: 'UX Redesign Phase 3 — Per-flow strangler cascade (destination→epic map + sequencing)'
author: 'Phase-3 kickoff session with Alexyu'
date: '2026-06-14'
pairs_with:
  - 01-nav-ia-decision-adr.md (D1–D4 + 7-destination IA)
  - 01-design-language-v2.md (shell spec §6, tokens, state standard)
  - 02-pilot-validation.md (§4 refuted assumptions = Phase-3 backend inputs; §5 untested)
inputDocuments:
  - _bmad-output/planning-artifacts/ux-redesign/00-redesign-brief.md
  - _bmad-output/planning-artifacts/ux-redesign/01-nav-ia-decision-adr.md
  - _bmad-output/planning-artifacts/ux-redesign/01-design-language-v2.md
  - _bmad-output/planning-artifacts/ux-redesign/02-pilot-validation.md
  - _bmad-output/implementation-artifacts/sprint-status.yaml
  - apps/web/src/routes/ (ground truth, verified 2026-06-14)
---

# Phase 3 — Destination → Epic Map & Cascade Sequencing

> **What this is.** The reconciliation step the Phase-3 plan demands *before*
> mechanically expanding the A–J flow folders. The ADR's **7 navigation
> destinations ≠ the old A–J flows** (C/E/F fragment across destinations), so this
> document maps every destination to its source flow(s), its migration status, and
> the Phase-3 epic that owns it — then sequences the cascade. This is the input
> contract for `sprint-planning` + `create-epics-and-stories`.

## 0. Confirmed decisions (Alexyu, 2026-06-14)

1. **Activity hub (D4-1) — BUILD this phase.** Net-new destination (no legacy route
   to strangle; purely additive). Unblocks the mobile bottom-4 3rd slot (首頁·媒體庫·
   活動·下載), delivers N1 ambient visibility, and gives P8's invisible journeys
   (AI subtitle, batch subtitle, scan) a home. Depends on Foundation #1 (N1
   item-level fields) + a new `/api/v1/activity` aggregate endpoint.
2. **Cascade start — 首頁 Home (flow H) FIRST**, immediately after the 3 Foundation
   backend items. Highest-risk-first: directly validates D3 ("does own-above-external
   hold under real content?") — the ADR's headline watch-item, untested in the pilot
   (§5). Foundation #1 (N1 fields) feeds the Home own-content blocks.
3. **Epic numbering reconciliation (corrected, see §4):** `sprint-status.yaml` /
   `epic-list.md` (PRD v4) is authoritative over the brief/ADR's stale labels.
   **繼續觀看 / continue-watching IS Epic 17's feature** (P4-011 "Home page Continue
   Watching based on **Plex/Jellyfin watch progress**") — and since the pilot
   confirmed **Vido has no playback path of its own** (§4.1), continue-watching data
   can ONLY come from Epic 17 (Media Server Integration, **backlog**). So Home v2
   **reserves the continue-watching slot** (designs the block + own-above-external
   ordering) but its **live data is deferred to Epic 17**; the block renders
   fail-soft-empty (or is hidden) until a media server is connected.

## 1. The 7 destinations → status / source flow / Phase-3 epic

Verb tags: ✅ done (pilot) · 🆕 net-new · ✏️ migrate-existing · ➡️ move.

| # | Destination | Status | Source A–J flow(s) | Phase-3 epic / work |
|---|---|---|---|---|
| 1 | **首頁 Home** | not migrated | **H** homepage | 🆕 **Home v2** — highest risk; validates D3 own-above-external; own-content blocks (recently-added / task-status, **+ continue-watching slot reserved but data deferred to Epic 17**) ABOVE Hero/ExploreBlocks; dashboard remnants (DownloadPanel/QBStatus/ConnectionHistory) move OFF to Activity/status. |
| 2 | **媒體庫 Library ▸ 電影/影集** | ✅ **pilot done (ux2-2/2-3) — do NOT redo** | A browse + B detail | Cleanup only: clean-route `/library/{movies,tv}` split (Foundation #3); **desktop filter rail** (PH3-F6 — replaces the mobile bottom-sheet misused on `lg`+; design flow-i `I5/I6/I7`, PR #89; own-collection filter, **distinct from Discover's power-filter in row 3**); pinnable saved views (D2 general mechanism); recs tile → `PosterCardV2`; **Epic 12 detail-block design backfill** (§3). |
| 3 | **探索 Discover** | not migrated | **I** advanced-search | 🆕 **Discover v2** — active power-filter tool (chips/presets/suggestions/filter-sheet) to v2; D3 boundary (grows NO dashboard); hosts Epic 13 Requests landing later. |
| 4 | **活動 Activity** | 🆕 **net-new (no route exists)** | aggregates 待解析 + **F** subtitle-batch + **E** scan + **G** AI-jobs + downloads-summary | 🆕 **Activity hub** (BUILD — decision §0.1). Composition endpoint + explain-why rows; downloads keep deep page (summary row links out). |
| 5 | **下載 Downloads** | not migrated | **D** downloads | 🆕 **Downloads v2** (Epic 14 v2 scope) — retained deep page, v2 restyle, card actions, pagination/batch-ops; linked from Activity. |
| 6 | **系統 System** | 🆕 **net-new route** | **C/E** settings children | 🆕 **System split** — move `settings/{status,logs,cache,performance,backup,export}` → `/system/*` (route-level redirects, 1 release); landing for Epic 16 (Stats) / Epic 18 (Health). |
| 7 | **設定 Settings** | not migrated | **C/E** settings | ✏️ **Settings v2** — preferences only (連線/掃描/首頁/qBT) to v2; pairs with the System split. |

**Cross-cutting (not destinations, but shell-level):**
- **Omnisearch (header, D4-5)** — retires `/search`; extend `search_service` (local + TMDB sectioned). Sources flow C search-half + flow I suggestions.
- **Ambient status strip (sidebar footer, D4-2)** — needs Foundation #2 (`/status/summary`). Shipped pilot-degraded; goes live with Foundation #2.
- **J 規格 (specs)** — not a destination; `.pen` spec frames maintained as each flow migrates.
- **F 字幕 / G AI 字幕** — NOT top-level destinations. They surface inside Detail (`管理字幕` CTA) + Activity. **G AI subtitle is P8 greenfield** (backend done, frontend UI never built) → a build, not a restyle.

## 2. Foundation backend (FIRST — cross-flow, not buried in a flow; pilot §4)

These three are blocking foundations, sequenced ahead of the flow cascade:

| # | Work | Why first | Touches |
|---|---|---|---|
| **F1** | **N1 item-level subtitle/lifecycle field** | Poster badge converges to **durable** states (整理中/已入庫/失敗/繁中/缺字幕); transient 簡轉繁·AI校正中 → Activity live SSE; 下載中% → Epic 13/14 (pilot §4.2 — not derivable on a steady grid). Feeds Home + Library + Activity. | Backend field + migration; poster-badge wiring (frontend). |
| **F2** | **`GET /api/v1/status/summary`** | Status strip real data (disk / active-scan / queue). Strip shipped pilot-degraded (§4.3). | New/extended aggregate handler (fail-soft per-section, B1/F3). |
| **F3** | **D2 clean-route `/library/{movies,tv}` split** | Low-risk cleanup deferred from pilot (§4.4); `/library?type=` redirects preserved. | Route-file split + `beforeLoad` redirects. |

## 3. A–J flow → destination crosswalk (why mechanical A–J expansion is wrong)

| A–J flow | Maps to destination(s) | Note |
|---|---|---|
| A Browse | 媒體庫 | ✅ pilot done |
| B Detail-interaction | 媒體庫 | ✅ pilot done (+ Epic 12 backfill) |
| C Search+Settings | **設定 + 系統 + omnisearch** | fragments 3 ways |
| D Downloads | 下載 | Epic 14 v2 |
| E Scanner | **設定/系統 (settings) + 活動 (scan progress)** | scan-progress overlay ownership moves into v2 shell |
| F Subtitle | **Detail 管理字幕 + 活動 (batch)** | feature surface, not a destination |
| G AI Subtitle | **活動 (AI jobs) + Detail** | **greenfield FE build (P8)** |
| H Homepage | 首頁 | highest risk (D3) |
| I Advanced-search | 探索 | |
| J Specs | cross-cutting `.pen` | not a destination |

## 4. Epic-numbering reconciliation (must settle before opening epics)

Conflict found between the brief/ADR (written 2026-06-12) and `sprint-status.yaml`
(PRD v4, the implementation tracker):

| Epic | brief/ADR says | PRD v4 (epic-list.md) says | Resolution (corrected) |
|---|---|---|---|
| 15 | "scope undefined, no IA slot" | **Indexer Integration** (P3-020/021) | Use PRD v4. NOT a UX-redesign destination — acquisition backend (Prowlarr / built-in tracker); no Phase-3 flow. |
| 17 | "Continue-watching → home block" | **Media Server Integration** (P4-010/011/012) | Use PRD v4. Continue-watching (P4-011) **IS** Epic 17 — sourced from Plex/Jellyfin watch progress; Vido has no playback (§4.1). Home v2 reserves the slot; **live data deferred to Epic 17 (backlog)**. |

Epics 13 (Requests, P3-001–005) / 14 (Downloads v2, P3-010–014) / 16 (Stats,
P4-001–004) / 18 (Health, P4-020–022) agree across docs and map as in §1.
**`epic-list.md` + `sprint-status.yaml` are authoritative.** Cross-reference worth
noting: the status strip (D4-2) + Activity hub (D4-1) deliver slices of **Epic 18**
(P4-020 service status, P4-021 disk warning, P4-022 activity log) — Foundation #2
(`/status/summary`) partially advances Epic 18.

## 5. Phase-3 epic list + cascade sequence (dependency-aware)

Proposed ordering for `sprint-planning`. High-freq / high-pain / high-risk first,
respecting dependencies. Naming follows the Phase-2 `ux2-*` convention → `ux3-*`.

```
0. ux3-foundation        (BLOCKING, first)
   F1 N1 item-level lifecycle field + poster-badge wiring
   F2 GET /api/v1/status/summary + status-strip go-live
   F3 /library/{movies,tv} clean-route split + redirects
   (independent units per ADR — can parallelize; F1 gates Home + Activity)

1. ux3-home-v2           (FIRST flow — decision §0.2; validates D3)
   own-content blocks (recently-added / task-status) ABOVE Hero+ExploreBlocks;
   continue-watching slot reserved but live data deferred to Epic 17 (§4); move
   dashboard remnants OFF home; D3 own-above-external acceptance is the headline
   gate. Needs F1.

2. ux3-activity-hub      (BUILD — decision §0.1; net-new)
   /activity route + /api/v1/activity composition; parse/subtitle-batch/scan/AI
   summary rows + downloads-summary row; unblocks mobile bottom-4. Needs F1 + F2.

3. ux3-discover-v2       (flow I → 探索)  + Epic 13 Requests landing slot
4. ux3-downloads-v2      (flow D → 下載; Epic 14 v2)
5. ux3-system-settings   (系統 split + 設定 v2 + omnisearch retire /search)
6. ux3-subtitle-v2       (flow F → Detail 管理字幕 + Activity batch)
7. ux3-ai-subtitle       (flow G → Activity AI jobs + Detail; P8 GREENFIELD FE)
8. ux3-detail-backfill   (Epic 12 blocks 豆瓣短評/預告/串流/推薦 → v2 .pen + verify)

9. MILESTONE — flag retirement
   core flows migrated → new_shell_enabled default-ON → batch-delete legacy shell
   (TabNavigation, AppShell, LegacyContentContainer) + remove tab-{label} testids
   (ADR D1-c: removed in the same story that deletes legacy shell, not aliased).
```

Ordering is a sprint-planning starting point, refinable. Hard dependencies: F1→Home,
F1+F2→Activity. 設定/系統 (5) is low-risk and can float earlier if a lower-risk win
is wanted between high-risk flows.

## 6. Per-flow workflow recipe (every epic in §5)

Each flow epic is a vertical slice (design → FE → BE → test), shipped then legacy
retired (strangler), staying green throughout:

1. **ux-designer** redraws the flow's `.pen` to v2 (tokens §2, type §3, four-state §7,
   shell §6) → run `python3 scripts/export-pen-screenshots.py`, update `SCREENS`
   dict if new screens, **commit only genuinely-changed PNGs** (regen is
   non-deterministic).
2. **sm** `create-story` → **dev** `dev-story` → **tea (Murat)** owns visual baselines
   + E2E.
3. **E2E MUST reuse `tests/support/helpers/seed-helpers.ts`** real seeding —
   **no "no data → `test.skip`" self-skips** (Epic 20 lesson). For surfaces needing
   data the create endpoints can't seed (e.g. seasons table), follow story-20-4's
   path (test-only seed endpoint or POST extension) rather than skipping.
4. Ship the flow, retire its legacy version.

**Conventions:** `-linux` visual baselines are NOT generated locally (darwin machine);
the CI `Visual Regression` workflow auto-opens a `chore(visual): bootstrap` PR — merge
it. PR/merge with gh `j620656786206` (verify `gh api user --jq .login`; switch in the
same shell command). Docs English, conversation zh-TW. Convention: open a fresh
session + Fable 5 per phase (this planning turn ran on Opus 4.8 1M for the
reconciliation).

## 7. Definition of Done (Phase 3)

- All 7 destinations + cross-cutting surfaces migrated to v2; Activity hub built.
- The 3 Foundation backend items (F1/F2/F3) shipped.
- `new_shell_enabled` flipped default-ON; legacy shell + components retired.
- Visual baselines + E2E green throughout; no data-dependent self-skips.
- Epic 12 detail-block design coverage backfilled to v2.
