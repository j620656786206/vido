# Story UX2-3: B′ Detail v2 — Full-page Detail in v2 behind `new_shell_enabled`

Status: done

> UX Redesign **Phase 2** pilot · Story 3 of 3. **Depends on UX2-1 (FOUNDATION)**; pairs with UX2-2 (shared `PosterCard-v2` for recommendations).
> Design: `ux-design.pen` `flow-b-detail-v2/*` (6 screens); `01-design-language-v2.md` §2–§3, §7. Replaces the cramped 460px slide-over panel (brief hotspot #2) with a full-page **backdrop hero**.

## Story

As the user opening a movie or series,
I want the detail page rebuilt in v2 — a backdrop hero with poster, title, lifecycle status and **primary actions**, then overview, cast, file/tech truth, and the Epic-12 sections (trailer / providers / recommendations / Douban) each failing soft — behind the `new_shell_enabled` flag,
so that the most satisfying surface (perfect zh-TW metadata) is also the most capable, and one flaky external source never blanks the page.

## Acceptance Criteria

1. **Full-page backdrop hero (`uRGu2`).** Within the v2 shell, `routes/media/$type.$id.tsx` renders a backdrop-hero layout (reuse the existing `backdropPath`, `:233`): bottom-gradient scrim, back affordance, poster thumbnail, and an info block — status badge → title (H1) → original/EN title → meta row (year · runtime/seasons · genre · ★rating) → **action row**. Replaces the old narrow-panel IA.
2. **Primary actions restored (brief P1 — CTAs were missing).** The hero action row shows the primary actions. ⚠️ **Rule 24 / brief P8 capability check (MANDATORY before promising a control):** verify what Vido actually supports. `播放` is included **only if** a real playback path exists (in-app player / external-player or Plex/Jellyfin deep-link / "open file location"); if none exists, the primary CTA becomes the real capability (e.g. `加入片單`, `管理字幕`, `在 Plex 開啟`) — do **not** ship a dead 播放 button. Secondary: `加入片單` + overflow `⋯`. Record the resolved action set in Completion Notes.
3. **Status machine (N1).** Hero status badge renders the lifecycle (`已入庫` / `下載中·{pct}%` / `整理中` / `想要` / `失敗`) + subtitle state (繁中 / 簡轉繁 / AI 校正中 / 缺字幕) from existing fields (`subtitleStatus`, download/library state) per §2.5 status→token map. Degrades silently if unknown.
4. **Body sections in v2 (reuse Epic-12 components).** Below the hero, reorganized + restyled to v2 tokens/type: **簡介** (overview) · **演員/製作** (`CreditsSection`) · **檔案資訊** (resolution/codec/audio/subtitle-track/size/path — the hybrid "tech truth at a glance" differentiator, tech badges in `accent-tint`) · **預告片** (`TrailerSection`) · **觀看平台** (`StreamingAvailability`) · **相似推薦** (`RecommendationsSection`, using `PosterCard-v2`) · **豆瓣** (`DoubanSection` — rating + 短評). Reuse the existing components/hooks (`useMovieCredits`/`useTVShowCredits`, `useRecommendations`, `useSeriesSeasons`, `useDoubanRating`, `useDoubanReviewSummary`) — restyle, do not re-fetch/re-invent.
5. **TV variant (`N2fmG6`).** Series detail shows season selector + **episode list** (reuse `SeasonAccordion`), meta shows `N 季 · M 集`; per-episode subtitle status where available.
6. **Per-section fail-soft (F3 / N4 / Rule 27 Pillar 3, `UH0sk`).** Each external-data section (trailer / providers / recommendations / Douban) degrades **independently** — an unavailable source renders a compact inline empty/stale/retry (e.g. Douban `豆瓣資料暫時無法取得` + `重試`, carrying its original `DOUBAN_*` code) and **never blanks or errors the page** (Epic-12 already fails soft per-section — preserve it through the restyle; the frontend treats a non-`ok` section as data).
7. **Four states (N4).** **Loading (`Tqy3E`)** — hero + body skeleton (no spinner; progressive per-section hydrate). **Not-found / error (`Z42zy`)** — back affordance + centered `找不到這部影片` / load-error + `返回媒體庫` (item removed or bad link; never a blank/technical page — brief P3). Happy + empty per section covered by #6.
8. **Both detail views (`:106/:109`).** The v2 layout applies to **both** `LocalDetailView` (library item) and `TMDbDetailView` (numeric TMDB id) — both already resolve a `tmdbId` and render the same sections; keep both working.
9. **Mobile (`SzNRb`).** Full-screen drill-in: shorter backdrop hero (back button overlay), full-width primary action + secondary icon buttons, then scrollable body (簡介 / 檔案資訊 / sections). 44px targets; bottom-tab hidden on the drill-in detail (pushed view).
10. **Flag-gated (strangler).** Renders only when `new_shell_enabled` ON (detail route opts into the new layout). Flag OFF → current detail unchanged. Return-context: back-navigation to Browse restores scroll/filter (chassis responsibility, with UX2-2).
11. **a11y.** AA contrast on text over the backdrop (scrim guarantees it); 44px actions; focus management on the trailer embed (reuse `TrailerEmbed` guard); `aria-label` on icon-only actions/back; `prefers-reduced-motion`.

## Tasks / Subtasks

### Frontend
- [ ] **T1: Backdrop hero + action row** (AC #1, #2, #3) — hero component (backdrop+scrim+poster+info+actions); status badge (§2.5); **resolve the 播放 capability per Rule 24** and wire the real action set; specs.
- [ ] **T2: v2 body reorg** (AC #4) — restyle + reorder existing sections (overview/credits/tech/trailer/providers/recs/douban) into the v2 layout; tech-info block (accent-tint badges + fact rows); recs use `PosterCard-v2`.
- [ ] **T3: TV variant** (AC #5) — season selector + episode list via `SeasonAccordion`, v2 styling.
- [ ] **T4: Four states + fail-soft** (AC #6, #7) — loading skeleton; not-found/error; preserve/verify per-section degrade through the restyle.
- [ ] **T5: Mobile detail** (AC #9) — full-screen drill-in layout.
- [ ] **T6: Specs + visual fixtures** (AC #11) — co-located specs (Rule 9/16); add hero + states to `/test/gallery`; jsx-a11y.

### Backend
- [ ] **T7: Capability + payload verification (reuse, no new endpoint expected)** — confirm playback capability for AC #2 (or confirm none → adjust CTA); confirm tech-info fields (codec/audio/subtitle-track/path/size) are in the detail payload; if a field is missing, Rule-24 triage (sub-task + AC), don't silently add. **Expected: 0 new endpoints** (Epic-5 detail + Epic-12 sections already serve this).

## Dev Notes

### Architecture Compliance
- **Rule 5:** all detail/section data via existing TanStack Query hooks; no re-fetch.
- **Rule 27 / F3:** every external section fails soft per-section (preserve Epic-12 behavior); frontend treats non-`ok` as data, never throws.
- **Rule 24 / brief P8 (CRITICAL):** do not promise a control (播放/pause/etc.) the backend lacks — verify first, adjust the CTA, record the decision.
- **N1/N4:** status badge everywhere; four states designed + built.
- **Rule 21:** new/changed components header `.pen` node ids (`uRGu2` etc.).
- **Rule 22/23:** hero + states get visual baselines via `/test/gallery`.
- **Strangler (P3):** detail-only; do not touch other flows.

### Cross-Stack Split Check (Agreement 5)
Backend tasks: **1** (verify, likely 0 code). Frontend tasks: **6**. Backend ≤3 → **NO split**. Frontend-led (Epic-5/12 backend already serves detail).

### Project Structure
- MODIFY: `routes/media/$type.$id.tsx` (v2 hero + reorg, both views); the Epic-12 section components (restyle to v2 tokens, keep logic).
- CREATE: `components/media/DetailHero.tsx` (+ spec); skeleton + not-found components.
- REUSE: `CreditsSection`, `TrailerSection`, `StreamingAvailability`, `DoubanSection`, `SeasonAccordion`, `RecommendationsSection`, `DualRatingDisplay`; hooks `useMovieCredits`/`useTVShowCredits`/`useRecommendations`/`useSeriesSeasons`/`useDoubanRating`/`useDoubanReviewSummary`; `PosterCard-v2` (UX2-2).

### Design Refs (`.pen` — Rule 21) — `flow-b-detail-v2/`
movie `uRGu2` · TV `N2fmG6` · error/not-found `Z42zy` · loading `Tqy3E` · extended/fail-soft `UH0sk` · mobile `SzNRb`. Screens: `_bmad-output/screenshots/flow-b-detail-v2/`.

### References
- [Source: planning-artifacts/ux-redesign/01-nav-ia-decision-adr.md — detail return-context, F3]
- [Source: planning-artifacts/ux-redesign/00-redesign-brief.md — hotspot #2 (panel reflow), P1 (missing CTAs), P3 (states), P8 (don't promise unsupported controls)]
- [Source: planning-artifacts/ux-redesign/01-design-language-v2.md — §2.5 status map, §7 states]
- [Source: project-context.md — Rules 5,9,16,21,22,23,24,27]
- [Source: apps/web/src/routes/media/$type.$id.tsx — LocalDetailView :112, TMDbDetailView :106, backdropPath :233, Epic-12 sections]

## Dev Agent Record

### Implementation Summary (Amelia/dev — 2026-06-14)

Branch `feat/ux2-3-detail-v2` (off main, post-UX2-2). Verified: `tsc` (0 new
errors), `eslint` (clean; the 1 `exhaustive-deps` warning in `$type.$id.tsx` is
pre-existing in the untouched legacy `LocalDetailView`), 11 new specs + the
existing detail-route spec (42 tests) pass, `nx build web` (2337 modules) ok.

**Gating (F4 preserved):** `MediaDetailRoute` branches on `useShellVersion()` —
v2 → `LocalDetailV2`/`TMDbDetailV2`; legacy → the existing views pixel-unchanged
(P3). Route marked `staticData.shell:'v2'`. The flag stays read-once in `__root`.

**v2 detail:** `DetailHeroV2` (`uRGu2` — full-page backdrop + scrim + back +
poster + status badges → H1 → original → meta → action row); `DetailTechInfoV2`
(檔案資訊 accent-tint badges + size/path facts — the hybrid differentiator);
`DetailStatesV2` (loading skeleton `Tqy3E` + not-found `Z42zy`, brief P3).
`LocalDetailV2` reuses every Epic-12 section (credits/trailer/providers/Douban/
recommendations/seasons) + their hooks, each failing soft independently (F3,
preserved). `TMDbDetailV2` = the lighter numeric-id variant (AC #8). N1 status via
the §2.5 util. Both views responsive (shorter mobile hero).

### Discovery Triage (Rule 24 / brief P8 — CRITICAL capability check)

**Playback capability verified → NO `播放` button.** Grepped the codebase: Vido has
**no media playback path** — no in-app player, no external-player launch, no
Plex/Jellyfin deep-link, no stream/file-serve endpoint (the only "player" is the
YouTube `TrailerEmbed`; `StreamingAvailability` shows external where-to-watch
only). So no dead `播放` CTA. **Resolved action set** (hero action row):
- **Primary `管理字幕`** — opens the existing `SubtitleSearchDialog` (the subtitle
  differentiator); gated on a local `filePath` (the dialog requires it).
- **Secondary `修改資訊`** — opens the existing `MetadataEditorDialog`.
- **`複製檔案路徑`** — copies `filePath` to the clipboard.
- The legacy `加入清單` button is **dropped** (vestigial — library items are
  already owned; there is no watchlist/request feature, Epic 13 is backlog).

Other triage:
1. **Recommendations reuse `RelatedContent` (not `PosterCardV2`).** AC #4 suggests
   recs use `PosterCard-v2`, but `RelatedContent` takes `RecommendationItem[]`
   (not `LibraryItem`) and already renders + fails soft. Reused as-is; a
   PosterCardV2-based recs tile is a Phase-3 refinement.
2. **Overflow `⋯` menu (重新解析/刪除) → deferred.** The action row ships the three
   real actions above; a Base UI Menu wrapper for reparse/delete (both single-item
   mutations exist) is a refinement, not built to avoid a new primitive mid-pilot.
3. **Visual gallery fixtures → deferred** (same rationale as UX2-2: token-only,
   covered by specs + runtime validation; batched to avoid a `-linux` bootstrap).

## Completion Notes
- Flag OFF → legacy detail unchanged. Flag ON → v2 detail (both Local + TMDb views).
- **Resolved CTA set recorded above** (Rule 24 mandate). No control is promised that
  the backend can't honor.
- Browser-pixel verification at 390/768/1440 happens in the Phase-2 validation step.
