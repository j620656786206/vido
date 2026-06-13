---
status: 'complete'
phase: 'UX Redesign Phase 2 — Browse + Detail pilot validation (go/no-go gate)'
author: 'Amelia (dev) + Alexyu'
date: '2026-06-14'
inputDocuments:
  - _bmad-output/planning-artifacts/ux-redesign/00-redesign-brief.md
  - _bmad-output/planning-artifacts/ux-redesign/01-nav-ia-decision-adr.md
  - _bmad-output/planning-artifacts/ux-redesign/01-design-language-v2.md
  - _bmad-output/implementation-artifacts/ux2-1-foundation-shell.md
  - _bmad-output/implementation-artifacts/ux2-2-browse-v2.md
  - _bmad-output/implementation-artifacts/ux2-3-detail-v2.md
shipped:
  - 'PR #67 — ux2-1 FOUNDATION shell (merged)'
  - 'PR #68 — ux2-2 A′ Browse v2 (merged)'
  - 'PR #69 — ux2-3 B′ Detail v2'
---

# Phase 2 Pilot Validation — Browse + Detail (go/no-go gate)

## 1. What shipped

The full Browse + Detail vertical slice landed behind `new_shell_enabled`
(default **OFF**), as three sequential PRs each green on CI (lint / unit / build /
docker / **E2E 4 shards** / visual):

| Story | What | PR |
|---|---|---|
| UX2-1 | v2 nav shell (collapsible sidebar + 64px rail + mobile tab/More sheet), Base UI wrappers + F2 ban, R3 token sweep + `--text-muted` AA fix, `LegacyContentContainer`, flag chokepoint | #67 (merged) |
| UX2-2 | A′ Browse v2 — `LibraryBrowseV2` (shared grid + state across `?type=`), `PosterCardV2` + N1 badge, integrated toolbar, four states, continuous scroll, merged mobile sheet | #68 (merged) |
| UX2-3 | B′ Detail v2 — `DetailHeroV2` backdrop hero + resolved CTAs, `DetailTechInfoV2`, reused Epic-12 sections (fail-soft), `TMDbDetailV2` variant, four states | #69 |

**Verification done in-pilot:** ~57 new unit specs across the three stories; the
existing E2E suite (legacy path, flag OFF) stayed green on every PR; `tsc` 0 new
errors; `nx build web` green. The **flag-OFF legacy path is provably unchanged**
(E2E + the 42-test detail-route spec + the legacy library/grid specs all pass with
no v2 provider in scope).

## 2. Method

This validation is **implementation-evidence first**: the act of building A+B
end-to-end against the real codebase tested the Phase-0/1 assumptions far more
sharply than a mockup walkthrough could — several were **empirically refuted at
the code boundary** (§4). The **live browser-pixel pass at 390 / 768 / 1440 with
real library data** (brief P10) is the one part that must run against the NAS
deployment with the flag ON — it is Alexyu's final gate (§6).

## 3. Assumptions that HELD

- **The strangler mechanism works (D1-c / F4).** One flag read in `__root.tsx`
  selects the shell; a shell-version context lets migrated routes render v2 vs
  legacy content **without a second flag read**. Flag OFF = legacy pixel-unchanged
  (E2E green ×3 PRs); flag ON = v2. Instant rollback (flip one setting) is real —
  the kill-clause is genuine, not aspirational. **This is the single most important
  thing the pilot proves: the per-flow migration model is safe and reversible.**
- **`LegacyContentContainer` fidelity (Murat acceptance).** Untouched routes render
  unchanged inside the v2 shell; their existing baselines/E2E pass under flag-ON.
- **N4 four-states is buildable as a standard, not an afterthought.** Browse ships
  all four (empty via the reused 3-state classifier / loading skeleton / no-result
  / fail-soft error); Detail ships loading + not-found + per-section fail-soft.
- **F3 per-section fail-soft survives a restyle.** Every Epic-12 detail section was
  reused with its `isLoading/isError/onRetry` intact — one flaky external source
  (Douban/TMDB/trailer) degrades its own section, never the page.
- **Base UI outsources the a11y that hand-rolling kept breaking (D1-d intent).**
  Tooltip/Sheet(Dialog) give focus-trap, Escape, scrim, scroll-lock by
  construction — the P4 failure class is structurally avoided for new overlays.
- **Shared grid + shared filter/sort/scroll state across type switches (F5).** One
  `LibraryBrowseV2` serves all/movie/tv, so movies↔tv preserves context — no
  forked grid logic (the P7 black-hole guard holds).
- **Backdrop hero replaces the cramped 460px panel (hotspot #2).** The full-page
  hero with poster + status + actions is a clear structural win over the slide-over.
- **N2 zh-TW typography as a material.** Noto for CJK, JetBrains Mono for
  numeric/tech values, 2-line CJK title grid — applied throughout A+B.

## 4. Assumptions REFUTED or ADJUSTED (the high-value findings)

These are the pilot's real payload — design assumptions that only break when you
build against the actual system. Each becomes a concrete Phase-3 input.

1. **Detail does NOT need a `播放` button — and assuming it did was the wrong
   frame (brief P1 vs P8).** Capability check (Rule 24): Vido has **no playback
   path** (no in-app player / external-player / Plex-Jellyfin deep-link /
   file-serve). The brief's P1 ("CTAs were missing") implicitly anchored on a
   media-player CTA; the real, owned capability is **subtitle management** (the
   differentiator). Resolved CTAs: `管理字幕` (primary) / `修改資訊` / `複製檔案路徑`.
   **This validates P8 directly** and is the clearest example of the pilot catching
   a design-implementation contract gap before it shipped a dead control.
2. **N1 "one truthful state machine" is only partially truthful at list scope.**
   Movies/series carry `parseStatus` + `subtitleTracks` but **no item-level
   `subtitleStatus`/download-progress** field (that's episode-only). So the badge
   truthfully shows 已入庫/整理中/失敗 + 繁中/簡中/缺字幕, but the richer process
   states the brief promised (`下載中·% / 簡轉繁 / AI 校正中`) are **not derivable
   on a poster today**. → **Phase-3 backend: an item-level subtitle/lifecycle field**
   so N1 is fully truthful on every poster.
3. **The ambient status strip (D4-2) has no data source yet.** Only service-health
   is cheaply available; disk headroom / active-scan / queue need an aggregate
   endpoint. The strip shipped **pilot-degraded** (health dots live, the rest
   fail-soft empty). → **Phase-3 backend: `GET /api/v1/status/summary`** (already
   anticipated by the ADR as Phase-3).
4. **The clean-route scheme (D2: `/library/movies`,`/library/tv`) fights the
   strangler flag.** Route-file splitting needs flag-conditional `beforeLoad`
   redirects for marginal go/no-go value (the URL is not what's validated). The
   pilot kept `/library?type=` (deep links preserved, legacy untouched) and
   delivers the identical v2 experience. **D2's intent holds; the route-file
   migration is a low-risk Phase-3 cleanup** once v2 is validated.
5. **Base UI's maturity premise in the ADR was stale.** D1-d cited "Base UI v1.0
   shipped" but the package was renamed (`@base-ui-components/react` → frozen rc.0)
   and is now `@base-ui/react@1.5.0` stable. Confirmed with Alexyu → followed the
   ADR with the corrected package. (Resolved; ADR/project-context text to be
   corrected when next touched.)
6. **The token sweep did NOT force a visual re-shoot.** AC #8 expected the
   `--text-muted` change to diff the committed `-linux` baselines; CI Visual
   Regression **passed** on UX2-1 — the feared bootstrap cycle never materialized.
7. **`解析度` as a filter chip is unsupported** (no backend list param) — deferred.

## 5. Not exercised by this pilot (scope honesty)

- **D3 hot-homepage principle-tension** — the ADR's *primary* Phase-2 watch-item
  ("does own-above-external hold under real content?") was **not tested**: the
  pilot migrated Browse + Detail only, not the homepage. This remains open and
  must be the headline watch-item when the homepage migrates in Phase 3.
- **Mobile bottom-4 frequency (活動 vs 下載).** `/activity` does not exist yet, so
  the pilot substituted `探索` for the 4th slot. The empirical frequency question
  is unanswerable until the Activity hub ships.
- **Collapsed-rail density at full scale.** The rail shows the 5 pilot
  destinations cleanly; the 7-destination + pinned-views density is untested.
- **Recommendations tile** reuses the Epic-12 `RelatedContent` (not `PosterCardV2`)
  — a Phase-3 refinement.

## 6. The remaining gate (Alexyu, on the NAS, flag ON)

Turn the flag on (`POST /api/v1/settings {key:"new_shell_enabled", value:true,
type:"bool"}` or set the DB row), then walk the slice at **390 / 768 / 1440** with
real library data:

- [ ] **1440 desktop:** sidebar (expanded ↔ 64px rail, persisted), library counts,
      active-state on `電影`/`影集`, grid density + 2-line CJK titles, status badges,
      list view, the four states (force a filter-miss + a backend error), detail
      backdrop hero + `管理字幕`/`修改資訊` actually working, tech-info, sections
      fail-soft, TV season accordion.
- [ ] **768 tablet:** grid reflow, toolbar wrap, hero scale.
- [ ] **390 mobile:** bottom tab bar + More sheet, merged sort+filter sheet,
      drill-in detail hero, 44px targets, no clipped CJK (R2).
- [ ] **Both shells:** flip the flag off → confirm the legacy library + detail are
      byte-for-byte the old experience (rollback proof).

## 7. Go / No-Go

**Recommendation: GO to Phase 3 — conditional on Alexyu's §6 browser-pixel pass.**

Rationale: the pilot **proved the thing Phase 2 existed to prove** — the
flag-gated, shell-version, per-flow strangler migration is coherent, safe, and
instantly reversible, and the v2 Browse + Detail are a clear improvement on the
two highest-traffic surfaces (the brief's #1 and #2 hotspots). The refuted
assumptions (§4) did not break the slice; they **sharpened the Phase-3 backlog**
with concrete, evidence-backed backend work (item-level subtitle/lifecycle field;
`/status/summary` aggregate; clean-route migration) instead of vague intentions.
The one genuine open risk is the **live feel at the three breakpoints with real
data** (§6) — the brief's own P10 ("tests green ≠ feature works") insists this
is a human gate, so the final GO is Alexyu's after that walk-through. The D3
homepage tension (§5) is **not** a Phase-2 blocker — it is the headline watch-item
for the *next* flow to migrate, not this one.
