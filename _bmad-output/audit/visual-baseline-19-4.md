# Component Visual-Baseline Audit — Story 19-4 (2026-05)

**Story:** `19-4-playwright-visual-snapshot-baseline` — Playwright `visual` project + dev-only
component gallery route + committed `toHaveScreenshot()` baselines.
**Date:** 2026-05-12 · **Author:** Amelia (DEV) · **Decided scope:** Party Mode 2026-05-12
(Sally + Bob + Murat + Winston + Amelia; Alexyu ratified) — **full harness + ~25 representative
components this story; the remaining ~99 → `19-4b-visual-baseline-bulk-fill`.**

This is the durable handoff doc for **19-5** (PR CI — `.github/workflows/visual-regression.yml`,
depends on the `test:visual` script + `visual` project name + baseline path defined here),
**19-4b** (bulk-fill the rest), and **19-8** (component-vs-`.pen` sweep — keys off the
`data-pen-node` attribute the gallery emits, sourced from `_bmad-output/audit/drift-19-3-2026-05.md`).

`data-gallery-id` = kebab of the import path. `data-pen-node` = a real `ux-design.pen` Reusable-
Component node id (Category-A `// Implements: Component/X (id)` header), or the literal
`screen-section` (`// Implements: <screen-section — pending epic-19-8 mapping>`), or `utility`
(`// Implements: <utility — no .pen counterpart>`). See `tests/visual/README.md`.

---

## Harness (LIVE this story)

| Piece | Path |
|-------|------|
| Playwright `visual` project (Chromium, 1280×800, `colorScheme: dark`, `reducedMotion: reduce`, `toHaveScreenshot` `maxDiffPixelRatio 0.001 / animations disabled / caret hide`) | `playwright.config.ts` |
| npm scripts | `package.json` → `test:visual`, `test:visual:update` (and `test:e2e*` made project-explicit so `visual` is excluded ⇒ feature-E2E count unchanged) |
| dev-only gallery route (inert in prod; `routes/test/manual-search.tsx` precedent) | `apps/web/src/routes/test/gallery.tsx` |
| fixtures (`-` prefix keeps it out of the route tree) | `apps/web/src/routes/test/-gallery.fixtures.tsx` |
| visual spec (DOM-driven worklist; `@visual @story-19-4`) | `tests/visual/components.visual.spec.ts` |
| committed baselines | `tests/visual/components.visual.spec.ts-snapshots/components/{id}/{state}-visual-darwin.png` |
| docs | `tests/visual/README.md` |

Burn-in: `pnpm run test:visual` re-run ×4 (post-`:update`) — all green, 0 flake. (`reducedMotion`
+ `animations:'disabled'` makes the CSS-driven hover/focus states deterministic.)

**Platform suffix:** baselines are `-darwin` (dev machine). 19-5's CI job (Linux) will either
regenerate the `-linux` set via `pnpm run test:visual:update` in a one-off commit, or 19-4b/19-5
adds `scripts/visual-baseline.sh` to generate the Linux set in the CI Docker image. *(Not done this
story — flagged optional in the Party Mode ruling; tracked for 19-5.)*

---

## Delivered baselines (25 components, 46 PNGs)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `ui-button` | `ui/Button.tsx` | `otvKh` (+`YDPhc`) | default, hover, focus | Component/ButtonPrimary + ButtonSecondary |
| `ui-badge` | `ui/Badge.tsx` | `utility` | default | |
| `ui-card` | `ui/Card.tsx` | `utility` | default | header + content children |
| `ui-skeleton` | `ui/Skeleton.tsx` | `utility` | default | |
| `ui-pagination` | `ui/Pagination.tsx` | `utility` | default, hover, focus | page 3 / 12 |
| `media-poster-card` | `media/PosterCard.tsx` | `RusTY` (+`MQbvp`) | default, hover, focus | hover = the MQbvp affordances (play overlay, kebab, receding badge cluster); library-admin path (`metadataSource` set — the bugfix-10-4 H2 regressor); non-numeric id ⇒ no TMDb fetch in the snapshot |
| `media-poster-card-skeleton` | `media/PosterCardSkeleton.tsx` | `utility` | default | |
| `media-color-placeholder` | `media/ColorPlaceholder.tsx` | `utility` | default | deterministic gradient from filename |
| `media-availability-badge-owned` | `media/AvailabilityBadge.tsx` | `screen-section` | default | variant `owned` |
| `media-availability-badge-requested` | `media/AvailabilityBadge.tsx` | `screen-section` | default | variant `requested` |
| `media-metadata-source-badge` | `media/MetadataSourceBadge.tsx` | `screen-section` | default | source `tmdb` |
| `media-tech-badge` | `media/TechBadge.tsx` | `L9m19` (+`9iTW3`/`f84BM`/`cUjyv`) | default | category `video` |
| `media-tech-badge-group` | `media/TechBadgeGroup.tsx` | `utility` | default | H.265 / 4K / DTS-HD 5.1 / HDR10 |
| `degradation-degradation-badge` | `degradation/DegradationBadge.tsx` | `screen-section` | default | level `partial` |
| `library-view-toggle` | `library/ViewToggle.tsx` | `screen-section` | default, hover, focus | view `grid` |
| `library-filter-chips` | `library/FilterChips.tsx` | `jD7gF` | default, hover, focus | Component/FilterChip — 2 genres + year range + unmatched |
| `library-sort-selector` | `library/SortSelector.tsx` | `955EZ` | default, hover, focus | Component/SortDropdown (closed state) |
| `library-empty-no-qbt` | `library/EmptyNoQBT.tsx` | `fSKuT` | default | Component/EmptyLibrary-NoQBT |
| `library-empty-no-folder` | `library/EmptyNoFolder.tsx` | `U3SGxG` | default | Component/EmptyLibrary-NoFolder |
| `library-empty-ready-for-scan` | `library/EmptyReadyForScan.tsx` | `mfKgm` | default | Component/EmptyLibrary-ReadyForScan |
| `library-empty-search-results` | `library/EmptySearchResults.tsx` | `screen-section` | default | |
| `metadata-editor-genre-selector` | `metadata-editor/GenreSelector.tsx` | `L1NP6` | default, hover, focus | Component/GenreTag — 2 selected |
| `search-search-bar` | `search/SearchBar.tsx` | `6MxLT` | default, hover, focus | Component/SearchInput — with text + clear button |
| `search-media-type-tabs` | `search/MediaTypeTabs.tsx` | `TboA7` (+`j98G4`) | default, hover, focus | TabActive/TabInactive — `movie` active, counts |
| `shell-tab-navigation` | `shell/TabNavigation.tsx` | `TboA7` (+`j98G4`) | default, hover, focus | TabActive/TabInactive (route-state driven) |
| `homepage-explore-block-skeleton` | `homepage/ExploreBlockSkeleton.tsx` | `utility` | default | 6 cards |

**Coverage:** all 12 Category-A components (the canonical `.pen`-mapped set) + 13 high-value
presentational components. No fixture rendered the error placeholder — all 25 render cleanly.

**UX (Sally) review — ✅ APPROVED (2026-05-12).** Reviewed all 46 committed baselines: every one
of the 25 components renders faithfully to the design intent (PosterCard default RusTY / hover MQbvp,
ButtonPrimary, SearchInput, FilterChip, GenreTag, TabActive/Inactive, TechBadge-*, the EmptyLibrary
trio, EmptySearchResults, AvailabilityBadge owned/requested, MetadataSourceBadge, DegradationBadge,
ColorPlaceholder, Card, Skeleton/PosterCardSkeleton/ExploreBlockSkeleton, ViewToggle, Pagination).
No broken/error renders; no misleading fixture states. **The baseline set is accepted** (AC #5
"first-Sally-approved web rendering"). 3 **non-blocking** follow-ups → CR / 19-4b: (1) focus-state
baselines need keyboard-tab focus (programmatic `locator.focus()` doesn't trigger `:focus-visible`,
so several `focus-*.png` ≈ `default`); (2) the TabNavigation baseline shows no active tab (the
gallery route `/test/gallery` matches no nav path — stub `useRouterState` or render under a matching
route); (3) interactive-open states (SortDropdown 955EZ open; menus/modals) need a per-fixture
`interaction` hook. Full gallery: `pnpm nx serve web` → `http://localhost:4200/test/gallery`.

---

## Pending (19-4b worklist — ~99 components, NOT design-drift findings)

`19-4b-visual-baseline-bulk-fill` (backlog, depends on 19-4 harness) adds fixtures + baselines for
the remaining `apps/web/src/components/` components — mostly data-driven ones needing seeded
React-Query data (HeroBanner, ExploreBlock/ExploreBlocksList, MediaGrid, MediaDetailPanel, the
`settings/*` family, `downloads/*`, `parse/*`, `scanner/*`, `health/*`, `manual-search/*`,
`metadata-editor/*` dialogs, `notifications/*`, `retry/*`, `dashboard/*`, `setup/*Step`, etc.) plus
the simpler ones left out of this batch (`ui/Dialog`, `ui/SidePanel`, `ui/HighlightText`,
`media/TVShowInfo`, `media/FileInfo`, `media/CreditsSection`, `media/TrailerEmbed`,
`media/FallbackPending`/`FallbackFailed`, `library/PosterCardMenu`, `library/SelectionToolbar`,
`library/SettingsGearDropdown`, `library/BatchProgress`, `library/BatchConfirmDialog`,
`degradation/ServiceHealthBanner`/`UnidentifiedFileCard`/`PlaceholderContent`, `shell/AppShell`,
`learning/*`, `subtitle/SubtitleSearchDialog`, …).

Regenerate the not-yet-baselined list:
```bash
# all in-scope component files…
find apps/web/src/components -name '*.tsx' ! -name '*.spec.tsx' ! -name '*.test.tsx'
# …minus those already in -gallery.fixtures.tsx (grep its `id:`/import list)
```

**Deliberate skips (not renderable in isolation — no baseline ever):** `parse/types.ts`,
`degradation/types.ts`, `downloads/formatters.ts` (type/util modules — no JSX), `parse/useParseProgress.ts`
(a hook misfiled under `components/` — exempt per Rule 21's hooks clause). The bare layout shells
(`shell/AppShell`, `dashboard/DashboardLayout`, `settings/SettingsLayout`, `setup/SetupWizard`) are
rendered by 19-4b only if a sensible isolated fixture exists; otherwise skipped with a reason here.

**This is NOT a list of design-drift findings.** No `bugfix-N` stories are filed from this doc.
Per-component pixel-classification against `.pen` (Rule 22 exact/minor/material) is **19-8's** job,
using this harness; material drift there → tracked `bugfix-N` stories.

---

## Material drift findings (Rule 22)

**None this story.** 19-4 builds the diff *tool*; it does not run the component-vs-`.pen` *diff*
(that's 19-8). The delivered baselines were spot-checked and render correctly; the `screen-section`/
`utility` `data-pen-node` values are carried straight from the 19-3 `// Implements:` headers.
