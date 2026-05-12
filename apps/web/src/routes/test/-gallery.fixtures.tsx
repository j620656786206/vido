/**
 * Component-gallery fixtures (story 19-4 — Playwright per-component visual baselines).
 *
 * One entry per in-scope `apps/web/src/components/` component. The gallery route
 * (`/test/gallery`) renders each entry inside a `<section data-gallery-id data-pen-node>`
 * wrapper with up to three `<div data-gallery-state>` blocks (default/hover/focus); the
 * Playwright `visual` project (`tests/visual/components.visual.spec.ts`) screenshots each.
 *
 * `penNode`:
 *   - a real `.pen` Reusable-Component node id (`RusTY`, `otvKh`, …) for Category-A files
 *     (the `// Implements: Component/X (id)` header — see `_bmad-output/audit/drift-19-3-2026-05.md`),
 *   - the literal `'screen-section'` for files carrying `// Implements: <screen-section …>`,
 *   - `'utility'` for in-scope Category-B files (`// Implements: <utility — no .pen counterpart>`).
 *   epic-19-8 keys its component-vs-`.pen` sweep off this attribute.
 *
 * `statesOnly`: restrict which states are rendered/snapshotted (e.g. skeletons & badges have
 * no meaningful hover/focus — default-only). Omit for the full default/hover/focus set.
 *
 * SCOPE NOTE (story 19-4, Party Mode 2026-05-12 ruling): this file currently covers the 12
 * Category-A components + ~13 high-value presentational components (≈25 of 124). The remaining
 * ~99 — most data-driven (HeroBanner, ExploreBlock, MediaDetailPanel, the settings/* family, …)
 * needing seeded React-Query/store fixtures — are tracked in `19-4b-visual-baseline-bulk-fill`
 * and the worklist table in `_bmad-output/audit/visual-baseline-19-4.md`.
 *
 * Typing is intentionally loose (`props: Record<string, unknown>` cast at render) — this is a
 * test fixture aggregator, not production code.
 */
import type { ComponentType } from 'react';

import { Button } from '../../components/ui/Button';
import { Badge } from '../../components/ui/Badge';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from '../../components/ui/Card';
import { Skeleton } from '../../components/ui/Skeleton';
import { Pagination } from '../../components/ui/Pagination';
import { PosterCard } from '../../components/media/PosterCard';
import { PosterCardSkeleton } from '../../components/media/PosterCardSkeleton';
import { ColorPlaceholder } from '../../components/media/ColorPlaceholder';
import { AvailabilityBadge } from '../../components/media/AvailabilityBadge';
import { MetadataSourceBadge } from '../../components/media/MetadataSourceBadge';
import { TechBadge } from '../../components/media/TechBadge';
import { TechBadgeGroup } from '../../components/media/TechBadgeGroup';
import { DegradationBadge } from '../../components/degradation/DegradationBadge';
import { ViewToggle } from '../../components/library/ViewToggle';
import { FilterChips } from '../../components/library/FilterChips';
import { SortSelector } from '../../components/library/SortSelector';
import { EmptyNoQBT } from '../../components/library/EmptyNoQBT';
import { EmptyNoFolder } from '../../components/library/EmptyNoFolder';
import { EmptyReadyForScan } from '../../components/library/EmptyReadyForScan';
import { EmptySearchResults } from '../../components/library/EmptySearchResults';
import { GenreSelector } from '../../components/metadata-editor/GenreSelector';
import { SearchBar } from '../../components/search/SearchBar';
import { MediaTypeTabs } from '../../components/search/MediaTypeTabs';
import { TabNavigation } from '../../components/shell/TabNavigation';
import { ExploreBlockSkeleton } from '../../components/homepage/ExploreBlockSkeleton';

const noop = () => {};

export type GalleryState = 'default' | 'hover' | 'focus';

export interface GalleryFixture {
  /** Stable kebab id derived from the component's import path (e.g. `media/PosterCard` → `media-poster-card`). */
  id: string;
  /** Human label shown above the card in the gallery (not screenshotted — outside the state divs). */
  label: string;
  component: ComponentType<Record<string, unknown>>;
  /** Props for every state. The same props are used for default/hover/focus (the state is applied by Playwright). */
  props?: Record<string, unknown>;
  /** `.pen` node id, or `'screen-section'` / `'utility'`. */
  penNode: string;
  /** If set, only these states are rendered & snapshotted. Default: all three. */
  statesOnly?: GalleryState[];
  /** Wrap the component in a fixed-width box so badges/inline elements don't collapse to 0-width. */
  width?: number;
}

export const GALLERY_FIXTURES: GalleryFixture[] = [
  // ----- ui/ -----
  {
    id: 'ui-button',
    label: 'ui/Button',
    component: Button as ComponentType<Record<string, unknown>>,
    props: { children: '主要按鈕' },
    penNode: 'otvKh', // + YDPhc (ButtonSecondary) — see drift-19-3-2026-05.md
  },
  {
    id: 'ui-badge',
    label: 'ui/Badge',
    component: Badge as ComponentType<Record<string, unknown>>,
    props: { children: '標籤' },
    penNode: 'utility',
    statesOnly: ['default'],
  },
  {
    id: 'ui-card',
    label: 'ui/Card',
    component: Card as ComponentType<Record<string, unknown>>,
    props: {
      children: (
        <>
          <CardHeader>
            <CardTitle>卡片標題</CardTitle>
            <CardDescription>卡片描述文字</CardDescription>
          </CardHeader>
          <CardContent>卡片內容區塊</CardContent>
        </>
      ),
    },
    penNode: 'utility',
    statesOnly: ['default'],
    width: 320,
  },
  {
    id: 'ui-skeleton',
    label: 'ui/Skeleton',
    component: Skeleton as ComponentType<Record<string, unknown>>,
    props: { className: 'h-6 w-48' },
    penNode: 'utility',
    statesOnly: ['default'],
  },
  {
    id: 'ui-pagination',
    label: 'ui/Pagination',
    component: Pagination as ComponentType<Record<string, unknown>>,
    props: { currentPage: 3, totalPages: 12, onPageChange: noop },
    penNode: 'utility',
  },

  // ----- media/ -----
  {
    id: 'media-poster-card',
    label: 'media/PosterCard',
    component: PosterCard as ComponentType<Record<string, unknown>>,
    // Non-numeric id ⇒ tmdbId 0 ⇒ useMovieDetails/useTVShowDetails stay disabled even on hover
    // (no network in the snapshot). Library-admin path: metadataSource set (bugfix-10-4 H2 regressor).
    props: {
      id: 'gallery-pc-uuid-0001',
      type: 'movie',
      title: '銀翼殺手 2049',
      posterPath: null,
      releaseDate: '2017-10-06',
      voteAverage: 8.0,
      metadataSource: 'TMDb',
      isNew: true,
      highlightQuery: '銀翼',
      onMenuClick: noop,
    },
    penNode: 'RusTY', // + MQbvp (PosterCardHover) — hover state captures the MQbvp affordances
    width: 200,
  },
  {
    id: 'media-poster-card-skeleton',
    label: 'media/PosterCardSkeleton',
    component: PosterCardSkeleton as ComponentType<Record<string, unknown>>,
    penNode: 'utility',
    statesOnly: ['default'],
    width: 200,
  },
  {
    id: 'media-color-placeholder',
    label: 'media/ColorPlaceholder',
    component: ColorPlaceholder as ComponentType<Record<string, unknown>>,
    props: { filename: '銀翼殺手 2049.mkv', height: 240 },
    penNode: 'utility',
    statesOnly: ['default'],
  },
  {
    id: 'media-availability-badge-owned',
    label: 'media/AvailabilityBadge (owned)',
    component: AvailabilityBadge as ComponentType<Record<string, unknown>>,
    props: { variant: 'owned' },
    penNode: 'screen-section',
    statesOnly: ['default'],
  },
  {
    id: 'media-availability-badge-requested',
    label: 'media/AvailabilityBadge (requested)',
    component: AvailabilityBadge as ComponentType<Record<string, unknown>>,
    props: { variant: 'requested' },
    penNode: 'screen-section',
    statesOnly: ['default'],
  },
  {
    id: 'media-metadata-source-badge',
    label: 'media/MetadataSourceBadge',
    component: MetadataSourceBadge as ComponentType<Record<string, unknown>>,
    props: { source: 'tmdb', fetchDate: '2026-01-15' },
    penNode: 'screen-section',
    statesOnly: ['default'],
  },
  {
    id: 'media-tech-badge',
    label: 'media/TechBadge',
    component: TechBadge as ComponentType<Record<string, unknown>>,
    props: { label: 'H.265', category: 'video' },
    penNode: 'L9m19', // + 9iTW3/f84BM/cUjyv (TechBadge-Audio/Subtitle/HDR)
    statesOnly: ['default'],
  },
  {
    id: 'media-tech-badge-group',
    label: 'media/TechBadgeGroup',
    component: TechBadgeGroup as ComponentType<Record<string, unknown>>,
    props: {
      videoCodec: 'H.265',
      videoResolution: '3840x2160',
      audioCodec: 'DTS-HD',
      audioChannels: 6,
      hdrFormat: 'HDR10',
    },
    penNode: 'utility',
    statesOnly: ['default'],
    width: 360,
  },

  // ----- degradation/ -----
  {
    id: 'degradation-degradation-badge',
    label: 'degradation/DegradationBadge',
    component: DegradationBadge as ComponentType<Record<string, unknown>>,
    props: { level: 'partial' },
    penNode: 'screen-section',
    statesOnly: ['default'],
  },

  // ----- library/ -----
  {
    id: 'library-view-toggle',
    label: 'library/ViewToggle',
    component: ViewToggle as ComponentType<Record<string, unknown>>,
    props: { view: 'grid', onViewChange: noop },
    penNode: 'screen-section',
  },
  {
    id: 'library-filter-chips',
    label: 'library/FilterChips',
    component: FilterChips as ComponentType<Record<string, unknown>>,
    props: {
      filters: { genres: ['動作', '科幻'], yearMin: 2010, yearMax: 2023, unmatched: true },
      onRemoveGenre: noop,
      onRemoveYearMin: noop,
      onRemoveYearMax: noop,
      onRemoveUnmatched: noop,
      onClearAll: noop,
    },
    penNode: 'jD7gF', // Component/FilterChip
    width: 640,
  },
  {
    id: 'library-sort-selector',
    label: 'library/SortSelector',
    component: SortSelector as ComponentType<Record<string, unknown>>,
    props: { sortBy: 'created_at', sortOrder: 'desc', onSortChange: noop },
    penNode: '955EZ', // Component/SortDropdown
  },
  {
    id: 'library-empty-no-qbt',
    label: 'library/EmptyNoQBT',
    component: EmptyNoQBT as ComponentType<Record<string, unknown>>,
    penNode: 'fSKuT', // Component/EmptyLibrary-NoQBT
    statesOnly: ['default'],
    width: 640,
  },
  {
    id: 'library-empty-no-folder',
    label: 'library/EmptyNoFolder',
    component: EmptyNoFolder as ComponentType<Record<string, unknown>>,
    penNode: 'U3SGxG', // Component/EmptyLibrary-NoFolder
    statesOnly: ['default'],
    width: 640,
  },
  {
    id: 'library-empty-ready-for-scan',
    label: 'library/EmptyReadyForScan',
    component: EmptyReadyForScan as ComponentType<Record<string, unknown>>,
    penNode: 'mfKgm', // Component/EmptyLibrary-ReadyForScan
    statesOnly: ['default'],
    width: 640,
  },
  {
    id: 'library-empty-search-results',
    label: 'library/EmptySearchResults',
    component: EmptySearchResults as ComponentType<Record<string, unknown>>,
    props: { query: '不存在的電影', onClear: noop },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 480,
  },

  // ----- metadata-editor/ -----
  {
    id: 'metadata-editor-genre-selector',
    label: 'metadata-editor/GenreSelector',
    component: GenreSelector as ComponentType<Record<string, unknown>>,
    props: { selectedGenres: ['action', 'sci-fi'], onToggle: noop },
    penNode: 'L1NP6', // Component/GenreTag
    width: 560,
  },

  // ----- search/ -----
  {
    id: 'search-search-bar',
    label: 'search/SearchBar',
    component: SearchBar as ComponentType<Record<string, unknown>>,
    props: { onSearch: noop, initialQuery: '銀翼殺手' },
    penNode: '6MxLT', // Component/SearchInput
    width: 480,
  },
  {
    id: 'search-media-type-tabs',
    label: 'search/MediaTypeTabs',
    component: MediaTypeTabs as ComponentType<Record<string, unknown>>,
    props: { activeType: 'movie', onTypeChange: noop, movieCount: 128, tvCount: 64 },
    penNode: 'TboA7', // + j98G4 (TabActive / TabInactive)
    width: 400,
  },

  // ----- shell/ -----
  {
    id: 'shell-tab-navigation',
    label: 'shell/TabNavigation',
    component: TabNavigation as ComponentType<Record<string, unknown>>,
    penNode: 'TboA7', // + j98G4 (TabActive / TabInactive)
    width: 480,
  },

  // ----- homepage/ -----
  {
    id: 'homepage-explore-block-skeleton',
    label: 'homepage/ExploreBlockSkeleton',
    component: ExploreBlockSkeleton as ComponentType<Record<string, unknown>>,
    props: { count: 6 },
    penNode: 'utility',
    statesOnly: ['default'],
    width: 900,
  },
];
