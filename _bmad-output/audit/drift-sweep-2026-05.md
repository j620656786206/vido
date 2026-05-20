# Comprehensive Design-Implementation Drift Sweep — Story 19-8 (2026-05)

> **Status: Task 1 skeleton (worklist).** The `Classification` / `Rationale` / `Disposition`
> columns are `_(pending)_` until the Task 3 classification pass fills them. Header tallies
> and the top-line conclusion are filled at sweep close (Task 3 / Task 6).

## Header

| Field | Value |
|-------|-------|
| Sweep date | 2026-05-20 |
| Sweep agents | Sally (UX — primary classifier) · Amelia (DEV — tooling, edits, spawned stories) — run solo by the DEV agent per Alexyu's 2026-05-20 `/dev-story` "Full solo sweep" election |
| Story | `19-8-comprehensive-component-sweep` — capstone of epic-19 |
| Total files examined | 131 (`apps/web/src/components/**/*.{tsx,ts}` — 12 Category-A + 25 Category-B + 94 Category-C) |
| git SHA at sweep-start | `45ba06f` |
| Total drifts by classification | _(pending — Task 3)_ |
| Top-line conclusion | _(pending — Task 3: "drift is systemic" / "drift is isolated to {N} components" / "drift is non-existent")_ |

## Methodology

**Classification thresholds — Rule 22 step 3 verbatim:**

- **exact-match:** pixel diff < 0.5 % against the `.pen` node OR the current rendered baseline
  visually matches the design intent per Sally's expert eye. Action: row in audit doc, no follow-up.
- **minor drift:** 0.5–5 % diff (typography / spacing / colour micro-shifts). Action: row + one-line
  rationale; bundled into `bugfix-N-polish-ux-visual-pass-2` ONLY if ≥ 3 minor drifts share a theme.
- **material drift:** > 5 % diff OR structural change. Action: row + dedicated `bugfix-N-{slug}` story
  (one per material finding, no bundling).
- **N/A — utility-confirmed:** Category-B `<utility — no .pen counterpart>` files — Sally confirms
  zero `.pen` counterpart, OR re-classifies.
- **screen-section-upgraded-to-{node}:** Category-C `<screen-section …>` files — mapped to a `.pen`
  Screen Frame / Reusable Component node; header upgraded per AC #5.

**Sample policy — FULL-SWEEP OVERRIDE.** Rule 22 step 1's "≥ 5 components per epic retro" sample
policy does **NOT** apply to this story. 19-8 is the comprehensive capstone sweep — every one of the
131 in-scope files is examined, not a sample. Future Rule 22 retros MUST NOT cite 19-8 as the
"≥ 5 sample" precedent — it is the explicit full-sweep exception.

**Diff-calculation tooling:**

- Code-side rendering: the 262 committed Playwright `toHaveScreenshot` baselines at
  `tests/visual/components.visual.spec.ts-snapshots/components/{gallery-id}/{state}-visual-darwin.png`
  (story 19-4 + 19-4b harness; `maxDiffPixelRatio 0.001`).
- Design-side: Pencil MCP `get_screenshot` on `ux-design.pen` nodes (read-only — no `.pen` writes).
- Pixel-diff is used only for borderline classifications (Sally's per-file call); per Rule 22 the
  primary tool is manual visual comparison + expert judgement.
- **Classification grounding:** the 262 baselines were captured from current code and were
  **Sally-APPROVED twice** — 2026-05-12 (the 25-component reference set, story 19-4 AC #5) and
  2026-05-14 (the 97-component bulk-fill, story 19-4b AC #3) — each review explicitly verified
  every rendered component "renders faithfully to the design intent". 19-8 formalises those
  approvals into the Rule-22 classification table below and resolves the 94 screen-section
  mappings. See `_bmad-output/audit/visual-baseline-19-4.md`.

## Screen Frame catalog (Task 1 worklist reference)

The `.pen` Screen Frames Sally maps Category-C files against (Pencil MCP `get_editor_state`,
2026-05-20). Vido screens only — the 4 UI-kit design-system frames (`shadcn`/`lunaris`/`halo`/`nitro`)
and 3 unnamed scratch frames are excluded.

| Node | Screen | Node | Screen |
|------|--------|------|--------|
| `KNI8F` | Screen 1 — Library Grid Desktop | `sAaCR` | Screen HP-1 — Homepage Desktop |
| `Qm662` | Screen 1b — PosterCard Hover State | `g5LFD` | Screen HP-2 — Homepage Mobile |
| `7fE0b` | Screen 1a — Settings Gear Dropdown | `Paqlk` | Screen HP-3 — Block CRUD Modal |
| `GOL63` | Screen 3 — Library Grid Mobile | `g6p38` | Screen HP-4 — Homepage Loading Skeleton |
| `3aSCw` | Screen 3a-m — Sort Bottom Sheet | `Y5XvRv` | Screen HP-5 — ExploreBlock Polish (bugfix-10-6) |
| `RgSxQ` | Screen 4 — Detail Panel Desktop | `XlFIq` | Screen PC-1 — PosterCard Info-Density (bugfix-10-7) |
| `407vK` | Screen 4b — Detail Panel Desktop (TV Series) | `NWxok` | Screen AS-1 — Advanced Search Filter Desktop |
| `auArc` | Screen 4a — PosterCard Context Menu | `TMaw5` | Screen AS-2 — Search Suggestions Dropdown |
| `7mdTJ` | Screen 4c — Detail Panel Context Menu | `i74p2` | Screen AS-3 — Save Filter Preset Modal |
| `vlL6O` | Screen 4f — Detail Panel Tech Badges Desktop | `pjKVZ` | Screen AS-4 — Mobile Filter Bottom Sheet |
| `2ltBl` | Screen 4d — Detail Fallback Desktop (Failed) | `rWvuG` | Screen G1 — Download List Desktop |
| `wQOkg` | Screen 4e — Detail Fallback Desktop (Pending) | `3ULXd` | Screen G2 — Torrent Expanded Detail Desktop |
| `kcn1v` | Screen 5 — Detail Panel Mobile | `cZd7j` | Screen G3 — Download List Mobile |
| `2m1Pv` | Screen 5b — Detail Fallback Mobile (Failed) | `tqHK9` | Screen G4 — Empty State Mobile |
| `7UnDy` | Screen 5c — Detail Fallback Mobile (Pending) | `KvZSc` | H1 — Settings Scanner Desktop |
| `6OR3z` | Screen 5d — Detail Tech Badges Mobile | `wyuhF` | H2 — Scan Progress Desktop |
| `LZ8Ds` | Screen 6 — List View Desktop | `szzaW` | H3 — Scan Complete Toast Desktop |
| `rsAxf` | Screen 7 — Search + Filter Desktop | `uABWl` | H4 — Settings Scanner Mobile |
| `dcf67` | Screen 8 — Batch Operations Desktop | `yezIo` | H5 — Scan Progress Mobile |
| `0KOE7` | Screen 8-m — Batch Operations Mobile | `ZjoEI` | H6 — Scan Complete Toast Mobile |
| `4VILE` | Screen 9a — Empty Library | `QTqcC` | H7 — Filtered Library Desktop |
| `IpZhv` | Screen 9b — Loading Skeleton | `n7jVF` | H8 — Filtered Library Mobile |
| `OYqNo` | Screen 9a-m — Empty Library Mobile | `cOrOR` | I1 — 字幕搜尋 Dialog [8-8] Desktop |
| `RxdY5` | Screen 9b-m — Loading Skeleton Mobile | `wy5Nx` | I2 — 預覽 + 下載狀態 [8-8] Desktop |
| `6UCtX` | Screen 10 — Settings Desktop | `GZ294` | I3 — 字幕搜尋 Dialog [8-8] Mobile |
| `2H4OM` | Screen 10-m — Settings Mobile | `ogQ6Y` | I6 — 字幕預覽 [8-8] Mobile |
| `1UHzI` | Screen 4a-m — PosterCard Context Menu Mobile | `NXijD` | I4 — 批次進度 [8-9] Desktop |
| `APfjC` | Screen 4c-m — Detail Context Menu Mobile | `fUtqO` | I5 — 批次進度 [8-9] Mobile |
| `IfrPQ` | Screen 1a-m — Settings Bottom Sheet Mobile | `TIIRl` | Screen I1 — AI Correction Modal Desktop |
| `oypj1` | Screen 10 — Filter Mobile | `kzhNP` | Screen I2 — Transcription Progress Desktop |
| `uhAKd` | Screen 11 — Backup Management Desktop | `22bcv` | Screen I3 — Translation Confirm Desktop |
| `8SSzc` | Design System Reference | `mgRJA`/`yNAHK`/`8Wsez` | Screen I4/I5/I6 — AI Correction/Transcription/Translation Mobile |

## Sweep findings table

131 rows — one per in-scope file. `Classification`: exact / minor / material / N/A-utility /
screen-section-upgraded. Final doc groups material-first; skeleton is in `find | sort` order.

| File | Current marker (pre-sweep) | `.pen` Node | Classification | Rationale | Disposition |
|------|----------------------------|-------------|----------------|-----------|-------------|
| `dashboard/CollapsibleSection.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `dashboard/DashboardLayout.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `dashboard/DownloadPanel.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `dashboard/QuickSearchBar.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `dashboard/RecentMediaPanel.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `degradation/DegradationBadge.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `degradation/PlaceholderContent.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `degradation/ServiceHealthBanner.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `degradation/types.ts` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `degradation/UnidentifiedFileCard.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `downloads/DownloadDetails.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `downloads/DownloadFilterTabs.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `downloads/DownloadItem.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `downloads/DownloadList.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `downloads/DownloadParseStatusBadge.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `downloads/formatters.ts` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `downloads/ParseFailedActions.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `downloads/StatusIcon.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `health/ConnectionHistoryPanel.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `health/QBStatusIndicator.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `homepage/ExploreBlock.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `homepage/ExploreBlockSkeleton.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `homepage/ExploreBlocksList.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `homepage/HeroBanner.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `homepage/TrailerModal.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `learning/LearnedPatternsSettings.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `learning/LearnPatternPrompt.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `library/BatchConfirmDialog.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `library/BatchProgress.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `library/EmptyNoFolder.tsx` | // Implements: Component/EmptyLibrary-NoFolder (U3SGxG) | U3SGxG | _(pending)_ | _(pending)_ | _(pending)_ |
| `library/EmptyNoQBT.tsx` | // Implements: Component/EmptyLibrary-NoQBT (fSKuT) | fSKuT | _(pending)_ | _(pending)_ | _(pending)_ |
| `library/EmptyReadyForScan.tsx` | // Implements: Component/EmptyLibrary-ReadyForScan (mfKgm) | mfKgm | _(pending)_ | _(pending)_ | _(pending)_ |
| `library/EmptySearchResults.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `library/FilterChips.tsx` | // Implements: Component/FilterChip (jD7gF) | jD7gF | _(pending)_ | _(pending)_ | _(pending)_ |
| `library/FilterPanel.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `library/LibraryGrid.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `library/LibrarySearchBar.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `library/LibraryTable.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `library/ParseFailureCard.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `library/PosterCardMenu.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `library/RecentlyAdded.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `library/SelectionToolbar.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `library/SettingsGearDropdown.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `library/SortSelector.tsx` | // Implements: Component/SortDropdown (955EZ) | 955EZ | _(pending)_ | _(pending)_ | _(pending)_ |
| `library/ViewToggle.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `manual-search/FallbackStatusDisplay.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `manual-search/ManualSearchDialog.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `manual-search/SearchResultCard.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `manual-search/SearchResultsGrid.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `media/AvailabilityBadge.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `media/ColorPlaceholder.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `media/CreditsSection.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `media/DetailPanelMenu.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `media/FallbackFailed.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `media/FallbackPending.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `media/FileInfo.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `media/MediaDetailPanel.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `media/MediaGrid.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `media/MetadataSourceBadge.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `media/PosterCard.tsx` | // Implements: Component/PosterCard (RusTY) + Component/PosterCardHover (MQbvp) | RusTY, MQbvp | _(pending)_ | _(pending)_ | _(pending)_ |
| `media/PosterCardSkeleton.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `media/TechBadge.tsx` | // Implements: Component/TechBadge-Video (L9m19) + Component/TechBadge-Audio (9iTW3) + Component/TechBadge-Subtitle (f84BM) + Component/TechBadge-HDR (cUjyv) | L9m19, 9iTW3, f84BM, cUjyv | _(pending)_ | _(pending)_ | _(pending)_ |
| `media/TechBadgeGroup.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `media/TrailerEmbed.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `media/TVShowInfo.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `metadata-editor/CastEditor.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `metadata-editor/GenreSelector.tsx` | // Implements: Component/GenreTag (L1NP6) | L1NP6 | _(pending)_ | _(pending)_ | _(pending)_ |
| `metadata-editor/MetadataEditorDialog.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `metadata-editor/PosterUploader.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `notifications/NewMediaNotifications.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `notifications/NewMediaToast.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `notifications/ParseCompleteToast.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `parse/ErrorDetailsPanel.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `parse/FloatingParseProgressCard.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `parse/LayeredProgressIndicator.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `parse/MediaFileCard.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `parse/ParseStatusBadge.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `parse/RetryQueueSection.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `parse/types.ts` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `parse/useParseProgress.ts` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `retry/CountdownTimer.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `retry/RetryNotifications.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `retry/RetryQueuePanel.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `retry/RetryQueueWithNotifications.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `scanner/ScanProgress.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `scanner/ScanProgressCard.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `scanner/ScanProgressSheet.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `search/MediaTypeTabs.tsx` | // Implements: Component/TabActive (TboA7) + Component/TabInactive (j98G4) | TboA7, j98G4 | _(pending)_ | _(pending)_ | _(pending)_ |
| `search/SearchBar.tsx` | // Implements: Component/SearchInput (6MxLT) | 6MxLT | _(pending)_ | _(pending)_ | _(pending)_ |
| `search/SearchResults.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/BackupManagement.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/BackupScheduleConfig.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/BackupTable.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/CacheManagement.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/CacheTypeCard.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/ConnectionTestResult.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/ExploreBlockEditModal.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/ExploreBlocksSettings.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/LibraryCard.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/LibraryEditModal.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/LogEntry.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/LogFilters.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/LogsViewer.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/MediaLibraryManager.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/MetadataExport.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/QBittorrentForm.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/RestoreConfirmDialog.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/ScannerSettings.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/ServiceStatusCard.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/ServiceStatusDashboard.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/SettingsLayout.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `settings/SettingsPlaceholder.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `setup/ApiKeysStep.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `setup/CompleteStep.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `setup/MediaFolderStep.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `setup/MediaLibrarySetupStep.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `setup/QBittorrentStep.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `setup/SetupWizard.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `setup/StepProgress.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `setup/WelcomeStep.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `shell/AppShell.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `shell/TabNavigation.tsx` | // Implements: Component/TabActive (TboA7) + Component/TabInactive (j98G4) | TboA7, j98G4 | _(pending)_ | _(pending)_ | _(pending)_ |
| `subtitle/SubtitleSearchDialog.tsx` | // Implements: <screen-section — pending epic-19-8 mapping> | — | _(pending)_ | _(pending)_ | _(pending)_ |
| `ui/Badge.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `ui/Button.tsx` | // Implements: Component/ButtonPrimary (otvKh) + Component/ButtonSecondary (YDPhc) | otvKh, YDPhc | _(pending)_ | _(pending)_ | _(pending)_ |
| `ui/Card.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `ui/Dialog.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `ui/HighlightText.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `ui/Pagination.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `ui/SidePanel.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |
| `ui/Skeleton.tsx` | // Implements: <utility — no .pen counterpart> | n/a (utility) | _(pending)_ | _(pending)_ | _(pending)_ |

## Material drift findings — bugfix story index

_(pending — Task 3 / Task 4. One row per material finding: slug · spawned `bugfix-N` story path ·
Sally's prioritisation rank.)_

| Rank | Finding | Component | bugfix-N story | Sprint-status bucket |
|------|---------|-----------|----------------|----------------------|
| _(pending)_ | | | | |

## Minor drift findings — bundling decisions

_(pending — Task 3. Per minor-drift theme cluster: members + bundled-into-`bugfix-N-polish-ux-visual-pass-2`
vs log-only.)_

## Screen-section mapping resolution

_(pending — Task 3. For each of the 94 Category-C files: the new pen-node mapping decision + the form
of the upgrade — canonical `// Implements:` header / soft `// Design ref:` comment / re-classified
to utility.)_

## Audit-trail markers

| Field | Value |
|-------|-------|
| Sweep date | 2026-05-20 |
| Sweep agents | Sally (UX classifier) · Amelia (DEV) |
| git SHA at sweep-start | `45ba06f` |
| git SHA at sweep-close | _(pending — Task 7)_ |
| Rendered-baseline snapshot | `tests/visual/components.visual.spec.ts-snapshots/components/` (262 PNGs, 19-4b queue) |
| `.pen` source | `ux-design.pen` (read-only — Pencil MCP `get_editor_state` + `get_screenshot`) |
