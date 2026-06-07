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

## CI wiring (LIVE since story 19-5, 2026-05-18)

**✅ DONE — see `.github/workflows/visual-regression.yml` + story
`_bmad-output/implementation-artifacts/19-5-github-actions-visual-regression-pr.md`.**

19-5 landed the `Visual Regression` GitHub Actions workflow:
- **PR job** (`Visual Regression / PR`) runs `pnpm run test:visual` against the committed
  baselines on every PR touching `apps/web/src/{components,routes,styles}/**` /
  `tailwind.config.js` / `index.css` / `main.tsx` / `tests/visual/**` /
  `playwright.config.ts` / `package.json` / the workflow file itself. Fail → ❌ check on
  the PR; combined with the branch-protection "Required check" rule the owner enables
  out-of-band, blocks merge. Diff artifacts (`actual.png`, `diff.png`, traces) uploaded
  with 14-day retention as `visual-regression-diffs-pr-${{ github.run_id }}`.
- **Main job** (`Visual Regression / Main`) runs on every push to `main`/`develop`
  without a `paths:` filter (drift can enter via dependency bumps, transitive Tailwind
  changes, or runner-image security patches that ship between deliberate-rebless PRs).
- **Runner** pinned to `ubuntu-24.04` (NOT `ubuntu-latest`). Image bumps follow the
  Baseline-update discipline below: deliberate-rebless PR labeled `requires-manual-review`,
  own commit, audit-doc line append `Linux baselines re-blessed {YYYY-MM-DD} via runner
  {new-image-label} (ImageVersion: {value}, previous: {old-image-label})` (CR 2026-05-19
  H1 — adds ImageVersion capture in lieu of Docker digest pin; see "Image pinning policy"
  section below).
- **First-run bootstrap** (implemented per the decision tree below): on the first
  main-push after the workflow lands, the main job detects no `-linux.png` baselines,
  runs `pnpm run test:visual:update`, appends a `Linux baselines bootstrapped {YYYY-MM-DD}
  via runner ubuntu-24.04 (ImageVersion: {value})` line to this doc (revised 2026-05-19
  per CR H1 — captures the `actions/runner-images` release version since GitHub-Hosted
  Runners don't expose a SHA digest), and opens a `chore(visual): bootstrap Linux baselines`
  PR labeled `requires-manual-review` via `peter-evans/create-pull-request@v6`. Once
  that PR merges, the bootstrap path is dead code forever (idempotent: the `find` count
  of `-linux.png` files is the single source of truth).

Operational follow-up the owner does post-merge (a single web-UI click): GitHub →
Settings → Branches → Branch protection rule for `main`/`develop` → "Require status
checks to pass before merging" → tick `Visual Regression / PR`. The workflow MAKES the
check appear in PR UI; making it required is the policy click the workflow can't perform
itself (story 19-5 AC #2 + Completion Notes follow-up).

---

**Platform suffix — DECIDED 19-4b Task 5 (2026-05-14): Linux baselines are bootstrapped by 19-5's
CI on first run (Option B).** Committed set is currently `-darwin` (dev machine). 19-5's CI
workflow (`.github/workflows/visual-regression.yml`, Linux runner) will, on its first execution,
detect the absent `-linux.png` files, run `pnpm run test:visual:update`, and open a one-off PR
committing the `-linux` set. That PR MUST carry the `requires-manual-review` label so Sally can
scan Linux-vs-darwin rendering before merge (content drift not expected — Sally already approved
darwin content; only rendering drift from font / emoji / sub-pixel differences anticipated). After
merge, CI runs in verify-only mode (`pnpm run test:visual`) on every PR thereafter.

The one-off PR's body MUST append a bootstrap-marker line to this audit doc in the form
`Linux baselines bootstrapped {YYYY-MM-DD} via runner {image-label} (ImageVersion: {value})` so
any future mass-rebless can be correlated to the runner image revision that produced the original
`-linux` set. (Revised 2026-05-19 per story 19-5 CR finding H1 — see "Image pinning policy" below
for why this format replaced the original `via CI image {full-digest-sha256:…}` form.)

**Image pinning policy (revised 2026-05-19, story 19-5 CR H1):** Story 19-5 ships with
`runs-on: ubuntu-24.04` (GitHub-Hosted Runner version-tag) rather than the originally-specified
`mcr.microsoft.com/playwright:vX.Y.Z-jammy@sha256:…` Docker image digest pin. The pivot was
forced by two implementation realities: (i) GitHub does not expose a SHA digest for hosted-runner
images, so the original "pin the digest at workflow-config time" mandate is not satisfiable for
hosted runners; (ii) switching to `container:` execution conflicts with the workflow's reliance
on `nx serve web` to reach the `/test/gallery` dev-gated route. **Mitigations in place:** (1)
each bootstrap audit line captures the `ImageVersion` env var (e.g. `20260512.1.0`, published by
[`actions/runner-images`](https://github.com/actions/runner-images)) so future investigation can
correlate baseline drift to a specific image revision; (2) the main-push job runs the full
visual suite without a `paths:` filter, so a runner-image roll that silently shifts glyphs
surfaces as a failing `Visual Regression / Main` check — the originally-feared mass-rebless still
gets a CI signal, just on `main` rather than on the PR that triggered the roll. **Image-label
upgrades (e.g. `24.04` → `26.04`) MUST be a deliberate-rebless PR** with the same discipline as
any reviewed-design rebless: `pnpm run test:visual:update` + `requires-manual-review` label + own
commit + audit-doc line append `Linux baselines re-blessed {YYYY-MM-DD} via runner {new-label}
(ImageVersion: {value}, previous: {old-label})`. The Docker-image-digest pin path is recorded
below as the rejected alternative; re-considerable if a future incident shows mutable-tag drift
slipping past the main-push safety net.

**Rejected alternative — Playwright Docker image with digest pin.** Originally specified in
this audit doc's earlier draft and in `tests/visual/README.md` §Platform-suffix (`mcr.microsoft.com/playwright:vX.Y.Z-jammy@sha256:…`).
Rejected during 19-5 implementation because (a) `container:` execution conflicts with `nx serve web`
(the only way to reach `/test/gallery`, which is gated behind `!import.meta.env.PROD`), (b) the
`ImageVersion`-based audit trail provides sufficient post-hoc traceability, (c) the main-push
job's no-filter full-suite run catches silent runner-image drift as a CI failure rather than
as silent baseline corruption.

**Rejected alternative (Option A) — local `scripts/visual-baseline.sh` Docker helper.** A thin
wrapper that `docker run`s the Playwright image with the repo mounted, runs `:update` inside the
container, producing `-linux` PNGs on the dev machine. Rejected by Party Mode 2026-05-14
(Murat + Winston + Bob; Alexyu ratified) because: (i) creates a second authoritative baseline
source (dev local + CI Linux) which violates "one authoritative environment" and risks
"I produced locally vs CI produced" drift (Murat — test reliability); (ii) the `scripts/`-pinned
image version bit-rots vs whatever 19-5's CI actually uses, especially under digest-pinning
(Winston — pin tag not enough, must pin digest); (iii) 19-4b's bounded context is fixture +
baseline coverage, not CI tooling — Option A crosses into 19-5's scope and duplicates the same
"one story too big" failure mode that originally caused 19-4 → 19-4b to be split (Bob — story
scope discipline). Re-considerable if 19-5 surfaces a concrete need for local Linux preview that
the CI-regen path cannot serve.

See `tests/visual/README.md` "Baseline-update discipline" → Platform suffix for the canonical
CI first-run decision tree (the implementer-facing form of this decision).

---

## Delivered baselines (122 unique components / 123 fixture entries / 262 PNGs)

> **Count nuance**: 122 distinct component source files; `media/AvailabilityBadge.tsx` is rendered
> twice (the `owned` and `requested` variants are visually distinct enough to baseline separately),
> giving 123 fixture entries in `apps/web/src/routes/test/-gallery.fixtures.tsx`. State distribution:
> 65 fixtures snapshot default/hover/focus (3-state), 3 fixtures snapshot default/hover/focus/open
> (`library/SortSelector`, `library/SettingsGearDropdown`, `media/DetailPanelMenu` — the
> `openTrigger` opt-ins), 55 fixtures are `statesOnly: ['default']` — total 262 PNGs (= 65·3 + 3·4 + 55·1).
> Delta vs 19-4 closeout: **+97 fixture entries / +216 PNGs** (19-4 shipped 26 entries / 46 PNGs;
> 19-4b Task 0 added the `library-sort-selector/open` state ⇒ +1 PNG and re-blessed 4 focus baselines;
> 19-4b Task 4 bulk-fill added 97 entries / 215 PNGs).

### Story-19-4 reference set (25 components / 26 fixture entries / 47 PNGs after 19-4b Task 0 re-bless)

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
| `library-sort-selector` | `library/SortSelector.tsx` | `955EZ` | default, hover, focus, **open** | Component/SortDropdown — `open` state added 19-4b Task 0 (reference fixture for the `openTrigger` mechanism; `data-testid="sort-selector-button"`) |
| `library-empty-no-qbt` | `library/EmptyNoQBT.tsx` | `fSKuT` | default | Component/EmptyLibrary-NoQBT |
| `library-empty-no-folder` | `library/EmptyNoFolder.tsx` | `U3SGxG` | default | Component/EmptyLibrary-NoFolder |
| `library-empty-ready-for-scan` | `library/EmptyReadyForScan.tsx` | `mfKgm` | default | Component/EmptyLibrary-ReadyForScan |
| `library-empty-search-results` | `library/EmptySearchResults.tsx` | `screen-section` | default | |
| `metadata-editor-genre-selector` | `metadata-editor/GenreSelector.tsx` | `L1NP6` | default, hover, focus | Component/GenreTag — 2 selected |
| `search-search-bar` | `search/SearchBar.tsx` | `6MxLT` | default, hover, focus | Component/SearchInput — `focus` re-blessed 19-4b Task 0 (sentinel+Tab → `:focus-visible` paints) |
| `search-media-type-tabs` | `search/MediaTypeTabs.tsx` | `TboA7` (+`j98G4`) | default, hover, focus | TabActive/TabInactive — `movie` active, counts |
| `shell-tab-navigation` | `shell/TabNavigation.tsx` | `TboA7` (+`j98G4`) | default, hover, focus | TabActive/TabInactive — all 3 baselines re-blessed 19-4b Task 0 (nested memory `RouterProvider` with `routePath: '/library'` ⇒ active tab now paints); darwin re-blessed 2026-06-04 (disc-nav-entry-discover-route — added 探索 `/discover` tab, 2nd position: 媒體庫·探索·下載中·待解析·設定). `-linux` re-bless pending CI. |
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

**19-4b Task 0 close-out (2026-05-13):** all three follow-ups landed (sentinel+Tab keyboard focus
spec, nested memory `RouterProvider` for `useRouterState`-driven fixtures via the optional
`routePath?: StubRoutePath` field, `openTrigger?: string` interactive-open mechanism). The 26
reference fixtures are unchanged except the four notes above: `library-sort-selector` gained an
`open` state (+1 PNG), `search-search-bar/focus` was re-blessed for `:focus-visible`, all three
`shell-tab-navigation/{default,hover,focus}` baselines were re-blessed with `/library` active.

### Story-19-4b bulk-fill set (97 fixture entries / 215 PNGs, delivered 2026-05-14)

> **Bucketing**: 63 Presentational (Task 2) + 34 Query-driven (Task 3 — `seedQueries`) — 0 Store-driven
> (Task 1 inventory found no `apps/web/src/stores/` consumers under `components/`; the `seedStore?`
> infrastructure stays for forward compatibility). 3 fixtures opt into the `open` state via
> `openTrigger` (added by 19-4b Task 4: `library/SettingsGearDropdown`, `media/DetailPanelMenu`,
> plus the Task 0 reference `library/SortSelector` already listed above). All Category-A `.pen`-node
> mappings were already covered by 19-4; every new bulk-fill entry lands at `screen-section`
> (Category C, 89 entries) or `utility` (Category B, 8 entries).

#### `ui/` (3 — all utility)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `ui-dialog` | `ui/Dialog.tsx` | `utility` | default | Radix `DialogContent` portals to `document.body` — captured via Task 4 viewport-fallback for zero-bbox state divs |
| `ui-highlight-text` | `ui/HighlightText.tsx` | `utility` | default | width 240 |
| `ui-side-panel` | `ui/SidePanel.tsx` | `utility` | default | fixed-position viewport overlay — viewport-fallback capture |

#### `media/` (8 new — 1 utility + 7 screen-section)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `media-credits-section` | `media/CreditsSection.tsx` | `screen-section` | default | width 480 |
| `media-detail-panel-menu` | `media/DetailPanelMenu.tsx` | `screen-section` | default, hover, focus, **open** | `openTrigger: '[data-testid="detail-menu-trigger"]'` (inline absolutely-positioned dropdown, not portaled) |
| `media-fallback-failed` | `media/FallbackFailed.tsx` | `screen-section` | default | uses TanStack Link; app-shell router resolves |
| `media-fallback-pending` | `media/FallbackPending.tsx` | `screen-section` | default | width 480 |
| `media-file-info` | `media/FileInfo.tsx` | `utility` | default | width 360 |
| `media-media-grid` | `media/MediaGrid.tsx` | `screen-section` | default | PosterCard children with `id:0` keep `useMovieDetails`/`useTVShowDetails` disabled — same defensive pattern as `media-poster-card` |
| `media-trailer-embed` | `media/TrailerEmbed.tsx` | `screen-section` | default, hover, focus | only the "▶ 觀看預告片" trigger; iframe state explicitly excluded |
| `media-tv-show-info` | `media/TVShowInfo.tsx` | `screen-section` | default | width 480 |

#### `degradation/` (3 new — 1 utility + 2 screen-section)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `degradation-placeholder-content` | `degradation/PlaceholderContent.tsx` | `utility` | default | width 200 |
| `degradation-service-health-banner` | `degradation/ServiceHealthBanner.tsx` | `screen-section` | default | width 640 |
| `degradation-unidentified-file-card` | `degradation/UnidentifiedFileCard.tsx` | `screen-section` | default | width 480 |

#### `dashboard/` (4 — 1 utility + 3 screen-section)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `dashboard-collapsible-section` | `dashboard/CollapsibleSection.tsx` | `utility` | default, hover, focus | width 480; `useNavigate`-only, no data hooks |
| `dashboard-download-panel` | `dashboard/DownloadPanel.tsx` | `screen-section` | default, hover, focus | Q-bucket — `seedQueries` for downloads list; `routePath` to /downloads |
| `dashboard-quick-search-bar` | `dashboard/QuickSearchBar.tsx` | `screen-section` | default, hover, focus | width 480 |
| `dashboard-recent-media-panel` | `dashboard/RecentMediaPanel.tsx` | `screen-section` | default, hover, focus | Q-bucket — seeded recent-media query; `routePath` to /library |

#### `downloads/` (6 — 1 utility + 5 screen-section)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `downloads-download-details` | `downloads/DownloadDetails.tsx` | `screen-section` | default, hover, focus | Q-bucket — seeded torrent details |
| `downloads-download-filter-tabs` | `downloads/DownloadFilterTabs.tsx` | `screen-section` | default, hover, focus | width 720 |
| `downloads-download-item` | `downloads/DownloadItem.tsx` | `screen-section` | default, hover, focus | width 720 |
| `downloads-download-list` | `downloads/DownloadList.tsx` | `screen-section` | default, hover, focus | width 720; `expandedHash=null` keeps `useDownloadDetails` dormant |
| `downloads-download-parse-status-badge` | `downloads/DownloadParseStatusBadge.tsx` | `screen-section` | default | width 160 |
| `downloads-parse-failed-actions` | `downloads/ParseFailedActions.tsx` | `screen-section` | default, hover, focus | width 320 |
| `downloads-status-icon` | `downloads/StatusIcon.tsx` | `utility` | default | width 120 |

#### `health/` (2 — Q-bucket)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `health-connection-history-panel` | `health/ConnectionHistoryPanel.tsx` | `screen-section` | default | Q-bucket; wraps in `SidePanel` (fixed-position, viewport-fallback) |
| `health-qb-status-indicator` | `health/QBStatusIndicator.tsx` | `screen-section` | default, hover, focus | Q-bucket — seeded qBT health |

#### `homepage/` (4 new — all screen-section, all Q-bucket except TrailerModal)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `homepage-explore-block` | `homepage/ExploreBlock.tsx` | `screen-section` | default, hover, focus | Q-bucket — seeded explore items; `image.tmdb.org/**` aborted in spec ⇒ deterministic fallback paint |
| `homepage-explore-blocks-list` | `homepage/ExploreBlocksList.tsx` | `screen-section` | default, hover, focus | Q-bucket — seeded explore blocks list |
| `homepage-hero-banner` | `homepage/HeroBanner.tsx` | `screen-section` | default, hover, focus | Q-bucket — seeded hero; TMDb image abort applies |
| `homepage-trailer-modal` | `homepage/TrailerModal.tsx` | `screen-section` | default | P-bucket via defensive `tmdbId: 0` (`useQuery` disabled) — empty-state render; viewport-fallback for `fixed inset-0` overlay |

#### `learning/` (2 — 1 P + 1 Q, both screen-section)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `learning-learn-pattern-prompt` | `learning/LearnPatternPrompt.tsx` | `screen-section` | default, hover, focus | |
| `learning-learned-patterns-settings` | `learning/LearnedPatternsSettings.tsx` | `screen-section` | default, hover, focus | Q-bucket — seeded learned-patterns |

#### `library/` (10 new — all screen-section)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `library-batch-confirm-dialog` | `library/BatchConfirmDialog.tsx` | `screen-section` | default | plain `fixed inset-0`, viewport-fallback |
| `library-batch-progress` | `library/BatchProgress.tsx` | `screen-section` | default | viewport-fallback overlay |
| `library-filter-panel` | `library/FilterPanel.tsx` | `screen-section` | default, hover, focus | Q-bucket — seeded filter facets |
| `library-library-grid` | `library/LibraryGrid.tsx` | `screen-section` | default, hover, focus | |
| `library-library-search-bar` | `library/LibrarySearchBar.tsx` | `screen-section` | default, hover, focus | |
| `library-library-table` | `library/LibraryTable.tsx` | `screen-section` | default, hover, focus | |
| `library-parse-failure-card` | `library/ParseFailureCard.tsx` | `screen-section` | default, hover, focus | defensive `parsedInfo.title: ''` + `filename: 'a.mkv'` keeps inner `useManualSearch` disabled (`params.query.length < 2`) |
| `library-poster-card-menu` | `library/PosterCardMenu.tsx` | `screen-section` | default | |
| `library-recently-added` | `library/RecentlyAdded.tsx` | `screen-section` | default, hover, focus | Q-bucket — seeded recently-added |
| `library-selection-toolbar` | `library/SelectionToolbar.tsx` | `screen-section` | default, hover, focus | |
| `library-settings-gear-dropdown` | `library/SettingsGearDropdown.tsx` | `screen-section` | default, hover, focus, **open** | `openTrigger` opt-in (added by Task 4 to give the gear menu an open baseline) |

#### `manual-search/` (4 — 1 Q + 3 P, all screen-section)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `manual-search-fallback-status-display` | `manual-search/FallbackStatusDisplay.tsx` | `screen-section` | default | |
| `manual-search-manual-search-dialog` | `manual-search/ManualSearchDialog.tsx` | `screen-section` | default | Q-bucket; custom `fixed inset-0` (not Radix portal) — viewport-fallback |
| `manual-search-search-result-card` | `manual-search/SearchResultCard.tsx` | `screen-section` | default, hover, focus | |
| `manual-search-search-results-grid` | `manual-search/SearchResultsGrid.tsx` | `screen-section` | default, hover, focus | |

#### `metadata-editor/` (3 new — all screen-section)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `metadata-editor-cast-editor` | `metadata-editor/CastEditor.tsx` | `screen-section` | default, hover, focus | |
| `metadata-editor-metadata-editor-dialog` | `metadata-editor/MetadataEditorDialog.tsx` | `screen-section` | default | Q-bucket; custom `fixed inset-0` overlay — viewport-fallback |
| `metadata-editor-poster-uploader` | `metadata-editor/PosterUploader.tsx` | `screen-section` | default, hover, focus | |

#### `media/` Q-bucket addition (1 — screen-section)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `media-media-detail-panel` | `media/MediaDetailPanel.tsx` | `screen-section` | default, hover, focus | Q-bucket — seeded movie/TV details |

#### `notifications/` (3 — all screen-section, default-only)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `notifications-new-media-notifications` | `notifications/NewMediaNotifications.tsx` | `screen-section` | default | |
| `notifications-new-media-toast` | `notifications/NewMediaToast.tsx` | `screen-section` | default | |
| `notifications-parse-complete-toast` | `notifications/ParseCompleteToast.tsx` | `screen-section` | default | |

#### `parse/` (6 — all screen-section)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `parse-error-details-panel` | `parse/ErrorDetailsPanel.tsx` | `screen-section` | default, hover, focus | |
| `parse-floating-parse-progress-card` | `parse/FloatingParseProgressCard.tsx` | `screen-section` | default | Q-bucket — `taskId={null}` ⇒ idle state |
| `parse-layered-progress-indicator` | `parse/LayeredProgressIndicator.tsx` | `screen-section` | default | |
| `parse-media-file-card` | `parse/MediaFileCard.tsx` | `screen-section` | default, hover, focus | |
| `parse-parse-status-badge` | `parse/ParseStatusBadge.tsx` | `screen-section` | default | |
| `parse-retry-queue-section` | `parse/RetryQueueSection.tsx` | `screen-section` | default, hover, focus | Q-bucket; CountdownTimer fixtures use `nextAttemptAt: '2020-...'` for "即將重試" stability (burn-in fix) |

#### `retry/` (4 — all screen-section)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `retry-countdown-timer` | `retry/CountdownTimer.tsx` | `screen-section` | default | `secondsRemaining=0` ⇒ stable "即將重試" literal |
| `retry-retry-notifications` | `retry/RetryNotifications.tsx` | `screen-section` | default | Q-bucket |
| `retry-retry-queue-panel` | `retry/RetryQueuePanel.tsx` | `screen-section` | default, hover, focus | Q-bucket; `nextAttemptAt: '2020-...'` stability fix |
| `retry-retry-queue-with-notifications` | `retry/RetryQueueWithNotifications.tsx` | `screen-section` | default, hover, focus | Q-bucket; same CountdownTimer stabilization |

#### `scanner/` (2 — all screen-section, default-only)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `scanner-scan-progress-card` | `scanner/ScanProgressCard.tsx` | `screen-section` | default | |
| `scanner-scan-progress-sheet` | `scanner/ScanProgressSheet.tsx` | `screen-section` | default | |

#### `search/` (1 new — screen-section, default-only)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `search-search-results` | `search/SearchResults.tsx` | `screen-section` | default | |

#### `settings/` (21 — 1 utility + 20 screen-section)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `settings-backup-management` | `settings/BackupManagement.tsx` | `screen-section` | default, hover, focus | Q-bucket |
| `settings-backup-schedule-config` | `settings/BackupScheduleConfig.tsx` | `screen-section` | default, hover, focus | Q-bucket |
| `settings-backup-table` | `settings/BackupTable.tsx` | `screen-section` | default | |
| `settings-cache-management` | `settings/CacheManagement.tsx` | `screen-section` | default, hover, focus | Q-bucket |
| `settings-cache-type-card` | `settings/CacheTypeCard.tsx` | `screen-section` | default, hover, focus | |
| `settings-connection-test-result` | `settings/ConnectionTestResult.tsx` | `screen-section` | default | |
| `settings-explore-block-edit-modal` | `settings/ExploreBlockEditModal.tsx` | `screen-section` | default, hover, focus | inline-fixed overlay (not Radix portal) — viewport-fallback |
| `settings-explore-blocks-settings` | `settings/ExploreBlocksSettings.tsx` | `screen-section` | default, hover, focus | Q-bucket |
| `settings-library-card` | `settings/LibraryCard.tsx` | `screen-section` | default, hover, focus | |
| `settings-library-edit-modal` | `settings/LibraryEditModal.tsx` | `screen-section` | default, hover, focus | Q-bucket; inline-fixed overlay — viewport-fallback |
| `settings-log-entry` | `settings/LogEntry.tsx` | `screen-section` | default, hover, focus | |
| `settings-log-filters` | `settings/LogFilters.tsx` | `screen-section` | default, hover, focus | |
| `settings-logs-viewer` | `settings/LogsViewer.tsx` | `screen-section` | default, hover, focus | Q-bucket |
| `settings-media-library-manager` | `settings/MediaLibraryManager.tsx` | `screen-section` | default, hover, focus | Q-bucket |
| `settings-metadata-export` | `settings/MetadataExport.tsx` | `screen-section` | default, hover, focus | |
| `settings-qbittorrent-form` | `settings/QBittorrentForm.tsx` | `screen-section` | default, hover, focus | Q-bucket |
| `settings-restore-confirm-dialog` | `settings/RestoreConfirmDialog.tsx` | `screen-section` | default | `fixed inset-0` — viewport-fallback |
| `settings-scanner-settings` | `settings/ScannerSettings.tsx` | `screen-section` | default, hover, focus | Q-bucket |
| `settings-service-status-card` | `settings/ServiceStatusCard.tsx` | `screen-section` | default, hover, focus | |
| `settings-service-status-dashboard` | `settings/ServiceStatusDashboard.tsx` | `screen-section` | default, hover, focus | Q-bucket |
| `settings-settings-placeholder` | `settings/SettingsPlaceholder.tsx` | `utility` | default | |

#### `setup/` (7 — 1 utility + 6 screen-section)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `setup-api-keys-step` | `setup/ApiKeysStep.tsx` | `screen-section` | default, hover, focus | |
| `setup-complete-step` | `setup/CompleteStep.tsx` | `screen-section` | default, hover, focus | |
| `setup-media-folder-step` | `setup/MediaFolderStep.tsx` | `screen-section` | default, hover, focus | |
| `setup-media-library-setup-step` | `setup/MediaLibrarySetupStep.tsx` | `screen-section` | default, hover, focus | |
| `setup-qbittorrent-step` | `setup/QBittorrentStep.tsx` | `screen-section` | default, hover, focus | |
| `setup-step-progress` | `setup/StepProgress.tsx` | `utility` | default | |
| `setup-welcome-step` | `setup/WelcomeStep.tsx` | `screen-section` | default, hover, focus | |

#### `subtitle/` (1 — screen-section, default-only)

| `data-gallery-id` | Component | `data-pen-node` | States | Notes |
|---|---|---|---|---|
| `subtitle-subtitle-search-dialog` | `subtitle/SubtitleSearchDialog.tsx` | `screen-section` | default | Q-bucket; custom `fixed inset-0` overlay — viewport-fallback |

**Coverage:** all 102 in-scope `apps/web/src/components/**/*.tsx` files identified by Task 1 inventory
either ship a fixture (97 new + 5 already covered by 19-4) or are documented as deliberate skips
(5 — see below). The 4 type/util `.ts` files (`parse/types.ts`, `degradation/types.ts`,
`downloads/formatters.ts`, `parse/useParseProgress.ts`) remain "no baseline ever" per Task 1
sub-bullet #3.

**Architectural fix landed during Task 4 (per Alex 2026-05-14 — Plan-D):** 12 `fixed inset-0` /
Radix-portal overlay fixtures intercepted pointer events globally and made any neighbour hover/focus
impossible. The gallery route gained `?fixture=<id>` + `?manifest=1` search params; the visual spec
navigates per-fixture for isolation. Zero-bbox state-divs (Radix portals + fixed-positioned children)
fall back to a viewport screenshot so the overlay paint is still captured. The single-fixture-per-page
architecture is the new harness default.

**UX (Sally) review — ✅ APPROVED (2026-05-14, parallel session).** Live `/test/gallery` probe + direct
PNG read across all 12 overlay viewport-captures + 3 CountdownTimer fixtures + 11 image-fallback
fixtures + broad P/Q-bucket spot-checks. **3 non-blocking observations** recorded (autofocus on
TrailerModal, HeroBanner fallback inconsistency, Max-update-depth warnings) — flagged for 19-8
design-drift sweep / follow-up bugfix, **NOT AC #3 blockers**. Verdict: AC #3 satisfied. See the
19-4b story file's "🎨 UX Verification — Task 4 Sally close gate" Completion Note for the full record.

**Burn-in:** `pnpm run test:visual` ×5 → 5/5 PASS (cold 2.3m / warm 1.0–1.1m). Two flake classes
fixed during Task 4 burn-in iterations: (1) `image.tmdb.org/**` aborted in `beforeEach` to force
deterministic error-fallback paint on poster-loaders (homepage-explore-block, dashboard-recent-media-panel,
homepage-hero-banner); (2) `nextAttemptAt: '2020-01-01...'` in 5 retry/parse fixtures to stabilize
CountdownTimer text (mirroring the standalone `retry-countdown-timer` pattern with
`secondsRemaining=0` ⇒ "即將重試" literal). `page.goto` per-fixture timeout raised to 60s with
`waitUntil: 'domcontentloaded'` to tolerate Vite's slow chunk-compilation under many sequential
navigations.

---

## Worklist closure — 19-4b bulk-fill (2026-05-14)

The "19-4b worklist — ~99 components" originally tracked here is **closed**. 19-4b shipped 97 new
fixture entries (63 P-bucket via Task 2 + 34 Q-bucket via Task 3) plus 215 new PNGs (Task 4 burn-in
green, Sally APPROVED 2026-05-14). The remaining 5 in-scope files are documented deliberate skips
(see below).

Final totals: **122 unique components / 123 fixture entries / 262 PNGs** (covered by the table
above). All 12 Category-A `.pen`-node mappings remain at their `// Implements:` header values
(unchanged from 19-4); every bulk-fill entry lands at `screen-section` (89 entries, Category C
placeholder pending 19-8 mapping) or `utility` (8 entries, Category B). Architectural exception:
12 `fixed inset-0` / Radix-portal overlays use the Task 4 single-fixture-per-page viewport-fallback
capture path.

### Deliberate skips (recorded per AC #4 — not renderable in isolation, no baseline ever)

**Type / util modules (4 — no JSX):**

| File | Reason |
|---|---|
| `apps/web/src/components/parse/types.ts` | TypeScript types only — no JSX |
| `apps/web/src/components/degradation/types.ts` | TypeScript types only — no JSX |
| `apps/web/src/components/downloads/formatters.ts` | Pure formatter utilities — no JSX |
| `apps/web/src/components/parse/useParseProgress.ts` | Misfiled hook under `components/` (Category B per drift-19-3); exempt per Rule 21 hooks clause |

**Layout shells (4 L-bucket — no isolated visual surface):**

| File | Lines | Reason |
|---|---|---|
| `shell/AppShell.tsx` | 111 | Wraps `TabNavigation` + `{children}`; TabNavigation has its own fixture, AppShell standalone screenshots the same thing |
| `dashboard/DashboardLayout.tsx` | 17 | Transparent `<div className=…>{children}</div>` — no isolated visual surface |
| `settings/SettingsLayout.tsx` | 182 | Sidebar nav driven by `useRouterState()`; cost/benefit weak — individual settings components ARE baselined (21 entries above) |
| `setup/SetupWizard.tsx` | 157 | Stateful multi-step controller (`useState` step machine + `useQueryClient`); no single static snapshot is meaningful; individual `*Step` components ARE baselined (7 entries above) |

**Other deliberate skip (1 — Task 3 documented):**

| File | Reason |
|---|---|
| `scanner/ScanProgress.tsx` | SSE-gated; idle-state would duplicate `scanner-scan-progress-card`/`scanner-scan-progress-sheet` (which ARE baselined). If Sally wants an in-progress baseline later, mock the progress state via a per-fixture override |

**Math check:** 102 in-scope (Task 1) − 97 new fixtures − 5 deliberate skips = 0 ✓.
Plus the 4 type/util `.ts` files already-excluded by `find -name '*.tsx'` = 5 + 4 = 9 documented
exemptions total.

**This is NOT a list of design-drift findings.** No `bugfix-N` stories are filed from this doc.
Per-component pixel-classification against `.pen` (Rule 22 exact/minor/material) is **19-8's** job,
using this harness; material drift there → tracked `bugfix-N` stories.

---

## Material drift findings (Rule 22)

**None this story.** 19-4 builds the diff *tool*; it does not run the component-vs-`.pen` *diff*
(that's 19-8). The delivered baselines were spot-checked and render correctly; the `screen-section`/
`utility` `data-pen-node` values are carried straight from the 19-3 `// Implements:` headers.

<!-- Historical record — original workflow format (CR H1 revised this format on 2026-05-19;
     this line predates the revision so ImageVersion is unknown from this record.
     `actions/runner-images` release notes for 2026-05-18 can recover the version retroactively. -->
Linux baselines bootstrapped 2026-05-18 via ubuntu-24.04

---

## 19-8 sweep classification (2026-05-20)

Story 19-8 (comprehensive design-implementation drift sweep) ran the Rule 22 classification
against the 262 baselines catalogued above. **Per-component classification is recorded in
`_bmad-output/audit/drift-sweep-2026-05.md` `## Sweep findings table`** (the authoritative
131-row table) — not duplicated here to avoid a second source of truth.

Post-sweep tally across all 131 in-scope `components/` files:

| Classification | Count | Notes |
|----------------|-------|-------|
| material drift | **0** | no `bugfix-N` story spawned |
| minor drift | **2** | `homepage/HeroBanner` (image-fallback background), `homepage/TrailerModal` (close-button autofocus) — both carried from this doc's 19-4b Sally review "3 non-blocking observations"; log-only (< 3, no shared theme) |
| exact-match | **97** | 12 Category-A (9 directly design-node-vs-baseline compared) + 85 Category-C |
| N/A — utility-confirmed | **25** | Category-B; 0 re-classifications |
| N/A — design-coverage gap | **7** | 6 `setup/*` wizard steps + `learning/LearnPatternPrompt` — no `.pen` screen frame |

**Conclusion:** the Sally baseline approvals recorded above (19-4 2026-05-12 + 19-4b 2026-05-14
— "every component renders faithfully to the design intent") are confirmed by the formal Rule 22
sweep. The bugfix-10-4 `PosterCardHover` drift was isolated, not systemic. The 3rd 19-4b
observation ("Max-update-depth warnings") is tracked separately by
`bugfix-19-4b-1-gallery-max-update-depth-warnings.md` (a console-warning bug, not visual drift).

Linux baselines incrementally bootstrapped 2026-06-05 via runner ubuntu-24.04 (ImageVersion: 20260525.161.1) — 3 fixtures: tests/visual/components.visual.spec.ts-snapshots/components/shell-tab-navigation/default-visual-linux.png tests/visual/components.visual.spec.ts-snapshots/components/shell-tab-navigation/hover-visual-linux.png tests/visual/components.visual.spec.ts-snapshots/components/shell-tab-navigation/focus-visual-linux.png

Linux baselines incrementally bootstrapped 2026-06-07 via runner ubuntu-24.04 (ImageVersion: 20260525.161.1) — 3 fixtures: tests/visual/components.visual.spec.ts-snapshots/components/media-media-detail-panel/default-visual-linux.png tests/visual/components.visual.spec.ts-snapshots/components/media-media-detail-panel/hover-visual-linux.png tests/visual/components.visual.spec.ts-snapshots/components/media-media-detail-panel/focus-visual-linux.png
