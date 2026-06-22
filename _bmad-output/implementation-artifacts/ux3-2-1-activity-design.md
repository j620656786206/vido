# Story ux3-2-1 — Activity hub v2 design (`.pen` flow-k-activity-v2)

**Epic:** ux3-activity-hub (UX Redesign Phase 3) · **Status:** done (design landed)
**Owner:** ux-designer (Pencil MCP) · **Type:** design · **FRs:** PH3-F4 · delivers Epic 18 slices (P4-020/021/022)

## Story

As the design system,
I want the net-new 活動 (Activity) hub drawn to v2 (per-flow recipe step 1),
So that dev builds the Activity hub against a spec — validating D4-1 (HYBRID activity hub:
unify parse / subtitle-batch / scan / AI-jobs with explain-why rows; downloads keep a deep
page, surfaced as a summary row that links out).

## Context — net-new destination

Activity is **destination #4** in the ADR's 7-destination IA and is **net-new** (no legacy
route to strangle; `/activity` and `/api/v1/activity` do not exist yet). It aggregates the
old A–J flows that fragment across destinations: 待解析 (pending parse) + **E** scan + **F**
subtitle-batch + **G** AI-jobs + a downloads-summary row. It unblocks the mobile bottom-4
3rd slot (首頁 · 媒體庫 · 活動 · 下載) and gives P8's previously-invisible journeys a home.

## What landed (in `ux-design.pen`, flow `flow-k-activity-v2`)

Drawn via Pencil MCP, reusing the shipped v2 shell (`HomeSidebar-v2` instanced with the
active item flipped 首頁→活動) + the existing `MobileTabItem` for the mobile bottom bar. New
frames + one new reusable component:

- **`Component/ActivityRow-v2`** (`fF8nX`) — the explain-why row: `[icon-chip · title +
  why-subtitle · right-slot]` over an optional progress track. Right-slot carries a Mono
  percent / count, a Noto status word, or an accent CTA; the progress track is disabled for
  non-progress rows. Token-only, Noto Sans TC + JetBrains Mono numerics.
- **`A1-D-v2`** (`kMeWS`) — desktop default. Main column top→bottom: **進行中** (掃描 / 批次
  字幕 / AI 校正 — each an explain-why row with a live progress bar + 進行中·N chip) →
  **待處理** (待解析項目 → `前往處理 →`, deep-links the library unmatched filter) → **下載**
  (summary row → `開啟下載頁 →`, links the existing `/downloads` deep page, D4-1) → **活動
  記錄** (recently-completed log — 完成/失敗 with success/error tint, P4-022).
- **`A2-M-v2`** (`QIwY1`) — mobile: top app bar (活動 title + search) + condensed rows +
  bottom tab bar with **活動 active** (the bottom-4 slot going live, ADR D1-b). 44px touch
  targets (search button, tab items).
- **`A4-D-v2`** (`suCiI`) — loading skeleton (row-shaped skeleton blocks, §7).
- **`A5-D-v2`** (`DZnSv`) — empty state (calm centered card: 「目前沒有進行中的活動」 + a
  next-step `掃描媒體庫` CTA, N "always show the next step").
- **`A6-D-v2`** (`M6ra92`) — per-section fail-soft: the 進行中 section degrades to an inline
  「無法載入，請稍後再試」 + 重試 banner; the rest of the page still renders (F3, page never
  hard-fails).

## Design decisions / review checklist — PASS

- **D4-1 HYBRID** verified: one hub unifies scan/subtitle/AI/parse as explain-why rows;
  downloads is a **summary row that links out** to the dedicated deep page (not absorbed).
- **N1 one truthful state machine:** lifecycle vocabulary reused — 進行中 (accent), 完成
  (success tint), 失敗 (error tint); the 進行中·N chip mirrors the home recently-added chip.
- **N4 four states present:** default / loading skeleton / empty / per-section fail-soft.
- **Shell reuse (no fork):** `HomeSidebar-v2` instanced with active flipped to 活動;
  `MobileTabItem` reused for the bottom bar. token-only colors; Noto Sans TC (CJK) +
  JetBrains Mono (numerics); 44px touch floor on mobile.

## Close-out

- Screenshots regenerated via `scripts/export-pen-screenshots.py` (`SCREENS` extended with
  the 5 `flow-k-activity-v2` frames); re-render was byte-stable this run (0 noise) — only the
  5 new PNGs committed alongside the `.pen`.
- **Next (ux3-activity-hub build):** this is a **cross-stack** epic, split per the rule —
  (1) BE: `GET /api/v1/activity` composition endpoint (fail-soft per-section, B1/F3),
  (2) FE: `/activity` route + components from this design (reuse `ActivityRow-v2`), wire the
  sidebar/mobile-tab 活動 destination live. Needs Foundation F1 + F2 (both shipped). E2E must
  reuse `tests/support/helpers/seed-helpers.ts` real seeding (no data-dependent self-skips).
