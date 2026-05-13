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

// ===== 19-4b Task 2 P-bucket additions (63 components, 17 subfolders) =====
import { Database } from 'lucide-react';

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '../../components/ui/Dialog';
import { HighlightText } from '../../components/ui/HighlightText';
import { SidePanel } from '../../components/ui/SidePanel';
import { CreditsSection } from '../../components/media/CreditsSection';
import { DetailPanelMenu } from '../../components/media/DetailPanelMenu';
import { FallbackFailed } from '../../components/media/FallbackFailed';
import { FallbackPending } from '../../components/media/FallbackPending';
import { FileInfo } from '../../components/media/FileInfo';
import { MediaGrid } from '../../components/media/MediaGrid';
import { TrailerEmbed } from '../../components/media/TrailerEmbed';
import { TVShowInfo } from '../../components/media/TVShowInfo';
import { PlaceholderContent } from '../../components/degradation/PlaceholderContent';
import { ServiceHealthBanner } from '../../components/degradation/ServiceHealthBanner';
import { UnidentifiedFileCard } from '../../components/degradation/UnidentifiedFileCard';
import { CollapsibleSection } from '../../components/dashboard/CollapsibleSection';
import { QuickSearchBar } from '../../components/dashboard/QuickSearchBar';
import { DownloadFilterTabs } from '../../components/downloads/DownloadFilterTabs';
import { DownloadItem } from '../../components/downloads/DownloadItem';
import { DownloadList } from '../../components/downloads/DownloadList';
import { DownloadParseStatusBadge } from '../../components/downloads/DownloadParseStatusBadge';
import { ParseFailedActions } from '../../components/downloads/ParseFailedActions';
import { StatusIcon } from '../../components/downloads/StatusIcon';
import { BatchConfirmDialog } from '../../components/library/BatchConfirmDialog';
import { BatchProgress } from '../../components/library/BatchProgress';
import { LibrarySearchBar } from '../../components/library/LibrarySearchBar';
import { LibraryTable } from '../../components/library/LibraryTable';
import { ParseFailureCard } from '../../components/library/ParseFailureCard';
import { PosterCardMenu } from '../../components/library/PosterCardMenu';
import { SelectionToolbar } from '../../components/library/SelectionToolbar';
import { SettingsGearDropdown } from '../../components/library/SettingsGearDropdown';
import { TrailerModal } from '../../components/homepage/TrailerModal';
import { LearnPatternPrompt } from '../../components/learning/LearnPatternPrompt';
import { FallbackStatusDisplay } from '../../components/manual-search/FallbackStatusDisplay';
import { SearchResultCard } from '../../components/manual-search/SearchResultCard';
import { SearchResultsGrid } from '../../components/manual-search/SearchResultsGrid';
import { CastEditor } from '../../components/metadata-editor/CastEditor';
import { PosterUploader } from '../../components/metadata-editor/PosterUploader';
import { NewMediaNotifications } from '../../components/notifications/NewMediaNotifications';
import { NewMediaToast } from '../../components/notifications/NewMediaToast';
import { ParseCompleteToast } from '../../components/notifications/ParseCompleteToast';
import { ErrorDetailsPanel } from '../../components/parse/ErrorDetailsPanel';
import { LayeredProgressIndicator } from '../../components/parse/LayeredProgressIndicator';
import { MediaFileCard } from '../../components/parse/MediaFileCard';
import { ParseStatusBadge } from '../../components/parse/ParseStatusBadge';
import { CountdownTimer } from '../../components/retry/CountdownTimer';
import { ScanProgressCard } from '../../components/scanner/ScanProgressCard';
import { ScanProgressSheet } from '../../components/scanner/ScanProgressSheet';
import { SearchResults } from '../../components/search/SearchResults';
import { BackupTable } from '../../components/settings/BackupTable';
import { CacheTypeCard } from '../../components/settings/CacheTypeCard';
import { ConnectionTestResult } from '../../components/settings/ConnectionTestResult';
import { LogEntry } from '../../components/settings/LogEntry';
import { LogFilters } from '../../components/settings/LogFilters';
import { RestoreConfirmDialog } from '../../components/settings/RestoreConfirmDialog';
import { ServiceStatusCard } from '../../components/settings/ServiceStatusCard';
import { SettingsPlaceholder } from '../../components/settings/SettingsPlaceholder';
import { ApiKeysStep } from '../../components/setup/ApiKeysStep';
import { CompleteStep } from '../../components/setup/CompleteStep';
import { MediaFolderStep } from '../../components/setup/MediaFolderStep';
import { MediaLibrarySetupStep } from '../../components/setup/MediaLibrarySetupStep';
import { QBittorrentStep } from '../../components/setup/QBittorrentStep';
import { StepProgress } from '../../components/setup/StepProgress';
import { WelcomeStep } from '../../components/setup/WelcomeStep';

import type { CastMember, CrewMember, TVShowDetails } from '../../types/tmdb';
import type { ServicesHealth } from '../../components/degradation/types';
import type { ParseStep } from '../../components/parse/types';
import type { ScanProgressState } from '../../hooks/useScanProgress';

const noop = () => {};

// ----- Shared mock-data consts for 19-4b Task 2 (parse/* and scanner/* fixtures) -----
const PARSE_STEPS_FAILED: ParseStep[] = [
  { name: 'filename_extract', label: '解析檔名', status: 'success' },
  { name: 'tmdb_search', label: '搜尋 TMDb', status: 'failed', error: 'API timeout' },
  { name: 'douban_search', label: '搜尋豆瓣', status: 'failed', error: '無法連線' },
  { name: 'wikipedia_search', label: '搜尋 Wikipedia', status: 'skipped' },
  { name: 'ai_retry', label: 'AI 重試', status: 'pending' },
  { name: 'download_poster', label: '下載海報', status: 'pending' },
];

const PARSE_STEPS_IN_PROGRESS: ParseStep[] = [
  { name: 'filename_extract', label: '解析檔名', status: 'success' },
  { name: 'tmdb_search', label: '搜尋 TMDb', status: 'success' },
  { name: 'douban_search', label: '搜尋豆瓣', status: 'in_progress' },
  { name: 'wikipedia_search', label: '搜尋 Wikipedia', status: 'pending' },
  { name: 'ai_retry', label: 'AI 重試', status: 'pending' },
  { name: 'download_poster', label: '下載海報', status: 'pending' },
];

const SCAN_STATE_ACTIVE: ScanProgressState = {
  isScanning: true,
  percentDone: 62,
  currentFile: '[Leopard-Raws] Demon Slayer S03E01.mkv',
  filesFound: 847,
  filesProcessed: 524,
  errorCount: 3,
  estimatedTime: '1 分 42 秒',
  isComplete: false,
  isCancelled: false,
  isMinimized: false,
  isDismissed: false,
  connectionStatus: 'sse',
};

export type GalleryState = 'default' | 'hover' | 'focus' | 'open';

/**
 * Memory-router pathnames a fixture can pin via `routePath` (19-4b Task 0 Fix B).
 * Mirrors `apps/web/src/components/shell/TabNavigation.tsx` `TABS.matchPaths`.
 */
export type StubRoutePath = '/library' | '/downloads' | '/pending' | '/settings';

export interface GalleryFixture {
  /** Stable kebab id derived from the component's import path (e.g. `media/PosterCard` → `media-poster-card`). */
  id: string;
  /** Human label shown above the card in the gallery (not screenshotted — outside the state divs). */
  label: string;
  component: ComponentType<Record<string, unknown>>;
  /** Props for every state. The same props are used for default/hover/focus/open (the state is applied by Playwright). */
  props?: Record<string, unknown>;
  /** `.pen` node id, or `'screen-section'` / `'utility'`. */
  penNode: string;
  /** If set, only these states are rendered & snapshotted. Default: `['default', 'hover', 'focus']`. */
  statesOnly?: GalleryState[];
  /** Wrap the component in a fixed-width box so badges/inline elements don't collapse to 0-width. */
  width?: number;
  /**
   * CSS selector (relative to the component's render — searched inside the state div) for
   * the element that, when clicked, opens an interactive sub-UI (dropdown / menu / modal).
   * If set, the gallery emits a `<div data-gallery-state="open">` and the visual spec clicks
   * this selector before screenshotting that state — captures e.g. `library/SortSelector`'s
   * open `SortDropdown 955EZ` panel. Combine with `statesOnly` to opt in. Added 19-4b Task 0 Fix C.
   */
  openTrigger?: string;
  /**
   * If set, the fixture renders inside a nested memory `RouterProvider` pinned to this path
   * (`/library` etc.). Used by components whose render depends on `useRouterState()` —
   * notably `shell/TabNavigation` for its active-tab state. The gallery route `/test/gallery`
   * matches none of `TabNavigation`'s `TABS.matchPaths`, so without this stub the
   * active-tab state never paints. Added 19-4b Task 0 Fix B.
   */
  routePath?: StubRoutePath;
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
    // 19-4b Task 0 Fix C: the dropdown panel `955EZ` itself is only visible when
    // the trigger button is clicked open. The visual spec clicks `openTrigger`
    // for the `open` state and captures the opened panel.
    statesOnly: ['default', 'hover', 'focus', 'open'],
    openTrigger: '[data-testid="sort-selector-button"]',
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
    // 19-4b Task 0 Fix B: TabNavigation reads `useRouterState()` to decide the active tab.
    // The gallery route `/test/gallery` matches none of `TABS.matchPaths`, so without a
    // stub the active-tab state never paints. `routePath` wraps this fixture in a nested
    // memory `RouterProvider` pinned to `/library` → the `TabActive (TboA7)` state paints.
    penNode: 'TboA7', // + j98G4 (TabActive / TabInactive)
    width: 480,
    routePath: '/library',
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

  // ========================================================================
  // 19-4b Task 2 — Presentational bucket bulk fill (63 components)
  // ========================================================================

  // ----- ui/ (P-bucket additions) -----
  {
    id: 'ui-dialog',
    label: 'ui/Dialog',
    // Radix `Root` re-export — the DialogContent is portaled to document.body.
    // The state-div screenshot may be empty; the portaled content paints under
    // the app shell providers but outside the per-fixture crop. Default-only.
    component: Dialog as ComponentType<Record<string, unknown>>,
    props: {
      open: true,
      onOpenChange: noop,
      children: (
        <DialogContent>
          <DialogHeader>
            <DialogTitle>確認操作</DialogTitle>
            <DialogDescription>請確認是否要繼續此操作。</DialogDescription>
          </DialogHeader>
          <p className="text-sm text-[var(--text-secondary)]">範例對話框內容。</p>
        </DialogContent>
      ),
    },
    penNode: 'utility',
    statesOnly: ['default'],
    width: 480,
  },
  {
    id: 'ui-highlight-text',
    label: 'ui/HighlightText',
    component: HighlightText as ComponentType<Record<string, unknown>>,
    props: { text: '銀翼殺手 2049', query: '銀翼' },
    penNode: 'utility',
    statesOnly: ['default'],
    width: 240,
  },
  {
    id: 'ui-side-panel',
    label: 'ui/SidePanel',
    // `position: fixed inset-0` overlay — covers the viewport when isOpen=true.
    // Snapshot default only; hover/focus on a viewport-wide overlay is not useful.
    component: SidePanel as ComponentType<Record<string, unknown>>,
    props: {
      isOpen: true,
      onClose: noop,
      title: '詳細資訊',
      children: <div className="p-4 text-sm text-[var(--text-secondary)]">面板內容範例</div>,
    },
    penNode: 'utility',
    statesOnly: ['default'],
  },

  // ----- media/ (P-bucket additions) -----
  {
    id: 'media-credits-section',
    label: 'media/CreditsSection',
    component: CreditsSection as ComponentType<Record<string, unknown>>,
    props: {
      director: {
        id: 1,
        name: '導演名',
        job: 'Director',
        department: 'Directing',
        profilePath: '/director.jpg',
      } satisfies CrewMember,
      cast: [
        { id: 1, name: '演員一', character: '角色一', profilePath: '/actor1.jpg', order: 0 },
        { id: 2, name: '演員二', character: '角色二', profilePath: '/actor2.jpg', order: 1 },
        { id: 3, name: '演員三', character: '角色三', profilePath: null, order: 2 },
        { id: 4, name: '演員四', character: '角色四', profilePath: '/actor4.jpg', order: 3 },
        { id: 5, name: '演員五', character: '角色五', profilePath: '/actor5.jpg', order: 4 },
        { id: 6, name: '演員六', character: '角色六', profilePath: '/actor6.jpg', order: 5 },
      ] satisfies CastMember[],
    },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 480,
  },
  {
    id: 'media-detail-panel-menu',
    label: 'media/DetailPanelMenu',
    // Internal open/close state via useState — opt into `open` via openTrigger
    // (same pattern as library-sort-selector). Dropdown is absolutely-positioned
    // inline (not Radix portal), so it stays inside the state div.
    component: DetailPanelMenu as ComponentType<Record<string, unknown>>,
    props: { onReparse: noop, onExport: noop, onDelete: noop },
    penNode: 'screen-section',
    statesOnly: ['default', 'hover', 'focus', 'open'],
    openTrigger: '[data-testid="detail-menu-trigger"]',
    width: 240,
  },
  {
    id: 'media-fallback-failed',
    label: 'media/FallbackFailed',
    // Uses TanStack `Link` — gallery route shares the app's RouterProvider.
    component: FallbackFailed as ComponentType<Record<string, unknown>>,
    props: {
      title: '[Leopard-Raws] Kimi no Na wa (BD)',
      mediaType: 'movie',
      filePath: '/volume1/Movies/Anime/[Leopard-Raws] Kimi no Na wa (BD).mkv',
      fileSize: 4509715660,
      createdAt: '2026-03-28T14:32:00Z',
      parseStatus: 'failed',
      onEditClick: noop,
    },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 480,
  },
  {
    id: 'media-fallback-pending',
    label: 'media/FallbackPending',
    component: FallbackPending as ComponentType<Record<string, unknown>>,
    props: { filename: '[Leopard-Raws] Kimi no Na wa (BD).mkv' },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 480,
  },
  {
    id: 'media-file-info',
    label: 'media/FileInfo',
    component: FileInfo as ComponentType<Record<string, unknown>>,
    props: {
      filePath: '/volume1/Movies/銀翼殺手 2049 (2017) 2160p.HDR.mkv',
      fileSize: 12_884_901_888,
    },
    penNode: 'utility',
    statesOnly: ['default'],
    width: 360,
  },
  {
    id: 'media-media-grid',
    label: 'media/MediaGrid',
    // Children id:0 keeps PosterCard's useMovieDetails/useTVShowDetails disabled
    // (no network on hover) — same pattern as the existing `media-poster-card` fixture.
    component: MediaGrid as ComponentType<Record<string, unknown>>,
    props: {
      items: [
        {
          mediaType: 'movie',
          item: {
            id: 0,
            title: '銀翼殺手 2049',
            originalTitle: 'Blade Runner 2049',
            overview: '',
            releaseDate: '2017-10-06',
            posterPath: null,
            backdropPath: null,
            voteAverage: 8.0,
            voteCount: 1000,
            genreIds: [878],
          },
        },
        {
          mediaType: 'tv',
          item: {
            id: 0,
            name: '咒術迴戰',
            originalName: 'Jujutsu Kaisen',
            overview: '',
            firstAirDate: '2020-10-03',
            posterPath: null,
            backdropPath: null,
            voteAverage: 8.6,
            voteCount: 500,
            genreIds: [16],
          },
        },
      ],
    },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 640,
  },
  {
    id: 'media-trailer-embed',
    label: 'media/TrailerEmbed',
    // Renders the "▶ 觀看預告片" button until clicked; iframe state (which would
    // load YouTube) is explicitly NOT captured.
    component: TrailerEmbed as ComponentType<Record<string, unknown>>,
    props: { videoKey: 'dQw4w9WgXcQ', title: '銀翼殺手 2049' },
    penNode: 'screen-section',
    width: 360,
  },
  {
    id: 'media-tv-show-info',
    label: 'media/TVShowInfo',
    component: TVShowInfo as ComponentType<Record<string, unknown>>,
    props: {
      show: {
        id: 0,
        name: '測試影集',
        originalName: 'Test TV Show',
        overview: '這是一部測試影集',
        firstAirDate: '2023-06-15',
        lastAirDate: '2024-01-20',
        posterPath: null,
        backdropPath: null,
        voteAverage: 8.5,
        voteCount: 2000,
        episodeRunTime: [45, 50],
        numberOfSeasons: 3,
        numberOfEpisodes: 30,
        status: 'Returning Series',
        type: 'Scripted',
        tagline: '',
        genres: [{ id: 1, name: '劇情' }],
        createdBy: [],
        networks: [
          { id: 1, name: 'Netflix', logoPath: null },
          { id: 2, name: 'HBO', logoPath: null },
        ],
        inProduction: true,
        seasons: [],
      } satisfies TVShowDetails,
    },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 480,
  },

  // ----- degradation/ (P-bucket additions) -----
  {
    id: 'degradation-placeholder-content',
    label: 'degradation/PlaceholderContent',
    component: PlaceholderContent as ComponentType<Record<string, unknown>>,
    props: { field: 'overview' },
    penNode: 'utility',
    statesOnly: ['default'],
    width: 200,
  },
  {
    id: 'degradation-service-health-banner',
    label: 'degradation/ServiceHealthBanner',
    component: ServiceHealthBanner as ComponentType<Record<string, unknown>>,
    props: {
      level: 'partial',
      services: {
        tmdb: {
          name: 'tmdb',
          displayName: 'TMDb',
          status: 'degraded',
          lastCheck: '2026-05-13T10:00:00Z',
          lastSuccess: '2026-05-13T09:00:00Z',
          errorCount: 3,
        },
        douban: {
          name: 'douban',
          displayName: '豆瓣',
          status: 'down',
          lastCheck: '2026-05-13T10:00:00Z',
          lastSuccess: '2026-05-12T15:00:00Z',
          errorCount: 12,
        },
        wikipedia: {
          name: 'wikipedia',
          displayName: 'Wikipedia',
          status: 'healthy',
          lastCheck: '2026-05-13T10:00:00Z',
          lastSuccess: '2026-05-13T10:00:00Z',
          errorCount: 0,
        },
        ai: {
          name: 'ai',
          displayName: 'AI',
          status: 'healthy',
          lastCheck: '2026-05-13T10:00:00Z',
          lastSuccess: '2026-05-13T10:00:00Z',
          errorCount: 0,
        },
      } satisfies ServicesHealth,
      onDismiss: noop,
    },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 640,
  },
  {
    id: 'degradation-unidentified-file-card',
    label: 'degradation/UnidentifiedFileCard',
    component: UnidentifiedFileCard as ComponentType<Record<string, unknown>>,
    props: {
      filename: 'Unknown.Movie.2024.1080p.WEB-DL.x264.mkv',
      attemptedSources: ['tmdb', 'douban', 'wikipedia'],
      onManualSearch: noop,
      onEditFilename: noop,
      onSkip: noop,
    },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 480,
  },

  // ----- dashboard/ (P-bucket additions) -----
  {
    id: 'dashboard-collapsible-section',
    label: 'dashboard/CollapsibleSection',
    component: CollapsibleSection as ComponentType<Record<string, unknown>>,
    props: {
      title: '最近新增',
      defaultExpanded: true,
      children: '段落內容範例',
      testId: 'collapsible-section-demo',
    },
    penNode: 'utility',
    width: 480,
  },
  {
    id: 'dashboard-quick-search-bar',
    label: 'dashboard/QuickSearchBar',
    // useNavigate from @tanstack/react-router — provided by the app shell.
    component: QuickSearchBar as ComponentType<Record<string, unknown>>,
    props: {},
    penNode: 'screen-section',
    width: 480,
  },

  // ----- downloads/ (P-bucket additions) -----
  {
    id: 'downloads-download-filter-tabs',
    label: 'downloads/DownloadFilterTabs',
    component: DownloadFilterTabs as ComponentType<Record<string, unknown>>,
    props: {
      activeFilter: 'all',
      counts: { all: 10, downloading: 3, paused: 2, completed: 4, seeding: 1, error: 0 },
      onFilterChange: noop,
    },
    penNode: 'screen-section',
    width: 720,
  },
  {
    id: 'downloads-download-item',
    label: 'downloads/DownloadItem',
    component: DownloadItem as ComponentType<Record<string, unknown>>,
    props: {
      download: {
        hash: 'abc123def456',
        name: '[SubGroup] Movie Name (2024) [1080p]',
        size: 4294967296,
        progress: 0.85,
        downloadSpeed: 10485760,
        uploadSpeed: 524288,
        eta: 600,
        status: 'downloading',
        addedOn: '2026-01-15T10:00:00Z',
        seeds: 10,
        peers: 5,
        downloaded: 3650722201,
        uploaded: 104857600,
        ratio: 0.03,
        savePath: '/downloads/movies',
      },
      expanded: false,
      onToggleExpand: noop,
    },
    penNode: 'screen-section',
    width: 720,
  },
  {
    id: 'downloads-download-list',
    label: 'downloads/DownloadList',
    // DownloadDetails (which calls useDownloadDetails) only mounts on row-expand —
    // default expandedHash is null, so no network hooks fire on mount.
    component: DownloadList as ComponentType<Record<string, unknown>>,
    props: {
      downloads: [
        {
          hash: 'abc123',
          name: 'Movie A [1080p]',
          size: 4294967296,
          progress: 0.85,
          downloadSpeed: 10485760,
          uploadSpeed: 0,
          eta: 600,
          status: 'downloading',
          addedOn: '2026-01-15T10:00:00Z',
          seeds: 10,
          peers: 5,
          downloaded: 3650722201,
          uploaded: 0,
          ratio: 0,
          savePath: '/downloads/movies',
        },
        {
          hash: 'xyz789',
          name: 'Series B S01',
          size: 8589934592,
          progress: 1,
          downloadSpeed: 0,
          uploadSpeed: 262144,
          eta: 8640000,
          status: 'completed',
          addedOn: '2026-01-14T10:00:00Z',
          seeds: 20,
          peers: 3,
          downloaded: 8589934592,
          uploaded: 1073741824,
          ratio: 0.125,
          savePath: '/downloads/series',
        },
      ],
      sortField: 'added_on',
      sortOrder: 'desc',
      onSortChange: noop,
      onOrderChange: noop,
    },
    penNode: 'screen-section',
    width: 720,
  },
  {
    id: 'downloads-download-parse-status-badge',
    label: 'downloads/DownloadParseStatusBadge',
    component: DownloadParseStatusBadge as ComponentType<Record<string, unknown>>,
    props: { parseStatus: { status: 'completed', mediaId: 'media-123' } },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 160,
  },
  {
    id: 'downloads-parse-failed-actions',
    label: 'downloads/ParseFailedActions',
    component: ParseFailedActions as ComponentType<Record<string, unknown>>,
    props: {
      torrentHash: 'abc123',
      errorMessage: '無法解析檔名',
      onRetry: noop,
      onManualSearch: noop,
    },
    penNode: 'screen-section',
    width: 320,
  },
  {
    id: 'downloads-status-icon',
    label: 'downloads/StatusIcon',
    component: StatusIcon as ComponentType<Record<string, unknown>>,
    props: { status: 'downloading' },
    penNode: 'utility',
    statesOnly: ['default'],
    width: 120,
  },

  // ----- library/ (P-bucket additions) -----
  {
    id: 'library-batch-confirm-dialog',
    label: 'library/BatchConfirmDialog',
    // Plain fixed-overlay dialog (not Radix portal); renders inline when isOpen=true.
    component: BatchConfirmDialog as ComponentType<Record<string, unknown>>,
    props: {
      isOpen: true,
      itemCount: 5,
      action: 'delete',
      onConfirm: noop,
      onCancel: noop,
    },
    penNode: 'screen-section',
    statesOnly: ['default'],
  },
  {
    id: 'library-batch-progress',
    label: 'library/BatchProgress',
    component: BatchProgress as ComponentType<Record<string, unknown>>,
    props: {
      isOpen: true,
      current: 5,
      total: 20,
      action: '刪除中...',
      isComplete: false,
      onClose: noop,
      onCancel: noop,
    },
    penNode: 'screen-section',
    statesOnly: ['default'],
  },
  {
    id: 'library-library-search-bar',
    label: 'library/LibrarySearchBar',
    component: LibrarySearchBar as ComponentType<Record<string, unknown>>,
    props: { onSearch: noop, initialQuery: '鬼滅之刃', resultCount: 15 },
    penNode: 'screen-section',
    width: 480,
  },
  {
    id: 'library-library-table',
    label: 'library/LibraryTable',
    // Renders TanStack `Link`s — router context provided by app shell.
    component: LibraryTable as ComponentType<Record<string, unknown>>,
    props: {
      items: [
        {
          type: 'movie',
          movie: {
            id: 'movie-1',
            title: '測試電影',
            originalTitle: 'Test Movie',
            releaseDate: '2023-06-15',
            genres: ['動作', '冒險', '科幻'],
            voteAverage: 8.5,
            posterPath: null,
            tmdbId: 123,
            parseStatus: 'success',
            createdAt: '2024-01-15T00:00:00Z',
            updatedAt: '2024-01-15T00:00:00Z',
          },
        },
        {
          type: 'series',
          series: {
            id: 'series-1',
            title: '測試影集',
            originalTitle: 'Test Series',
            firstAirDate: '2022-03-10',
            genres: ['劇情'],
            voteAverage: 9.1,
            posterPath: null,
            tmdbId: 456,
            parseStatus: 'success',
            createdAt: '2024-02-01T00:00:00Z',
            updatedAt: '2024-02-01T00:00:00Z',
          },
        },
      ],
      isLoading: false,
      sortBy: 'title',
      sortOrder: 'asc',
      onSort: noop,
    },
    penNode: 'screen-section',
    width: 960,
  },
  {
    id: 'library-parse-failure-card',
    label: 'library/ParseFailureCard',
    // ParseFailureCard renders ManualSearchDialog (closed). The dialog mounts with
    // useManualSearch enabled when initial query length >= 2 (derived from parsedInfo.title
    // OR filename). Defense: empty parsedInfo.title + short filename gives derived query
    // length < 2 → useQuery stays disabled, no network. Adjusted from sub-agent draft.
    component: ParseFailureCard as ComponentType<Record<string, unknown>>,
    props: {
      file: {
        id: 'file-123',
        filename: 'a.mkv',
        path: '/media/anime/a.mkv',
        size: 1572864000,
        parsedInfo: {
          title: '',
          year: undefined,
          mediaType: undefined,
          season: undefined,
          episode: undefined,
        },
        metadataStatus: 'failed',
        fallbackStatus: {
          attempts: [
            { source: 'tmdb', success: false },
            { source: 'douban', success: false },
          ],
        },
      },
      onMetadataApplied: noop,
    },
    penNode: 'screen-section',
    width: 280,
  },
  {
    id: 'library-poster-card-menu',
    label: 'library/PosterCardMenu',
    // No internal trigger button — rendered open via isOpen=true. Positioned absolute,
    // anchored by a width-constrained wrapper.
    component: PosterCardMenu as ComponentType<Record<string, unknown>>,
    props: {
      onViewDetails: noop,
      onReparse: noop,
      onExport: noop,
      onDelete: noop,
      isOpen: true,
      onClose: noop,
      isMobile: false,
    },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 240,
  },
  {
    id: 'library-selection-toolbar',
    label: 'library/SelectionToolbar',
    component: SelectionToolbar as ComponentType<Record<string, unknown>>,
    props: {
      selectedCount: 3,
      onDelete: noop,
      onReparse: noop,
      onExport: noop,
      onCancel: noop,
      isProcessing: false,
    },
    penNode: 'screen-section',
    width: 720,
  },
  {
    id: 'library-settings-gear-dropdown',
    label: 'library/SettingsGearDropdown',
    // Trigger button [data-testid="settings-gear-button"] toggles internal isOpen.
    // Opt into 'open' state via openTrigger to capture the dropdown panel.
    component: SettingsGearDropdown as ComponentType<Record<string, unknown>>,
    props: {
      preferences: { density: 'medium', defaultSort: 'created_at', titleLanguage: 'zh-tw' },
      onPreferencesChange: noop,
    },
    penNode: 'screen-section',
    statesOnly: ['default', 'hover', 'focus', 'open'],
    openTrigger: '[data-testid="settings-gear-button"]',
    width: 320,
  },

  // ----- homepage/ (P-bucket additions) -----
  {
    id: 'homepage-trailer-modal',
    label: 'homepage/TrailerModal',
    // FLAG (Task 1 inventory correction): TrailerModal calls useQuery internally —
    // should ideally be in Q-bucket. Defensive: tmdbId=0 ⇒ enabled=false ⇒ no network.
    // Renders the "找不到預告片" empty state deterministically. Re-bucket to Q in Task 3
    // if seeded video-key payload baseline is desired.
    component: TrailerModal as ComponentType<Record<string, unknown>>,
    props: {
      open: true,
      onClose: noop,
      mediaType: 'movie',
      tmdbId: 0,
      title: '鬥陣俱樂部',
    },
    penNode: 'screen-section',
    statesOnly: ['default'],
  },

  // ----- learning/ (P-bucket additions) -----
  {
    id: 'learning-learn-pattern-prompt',
    label: 'learning/LearnPatternPrompt',
    component: LearnPatternPrompt as ComponentType<Record<string, unknown>>,
    props: {
      filename: '[SubsPlease] Frieren - 01 (1080p) [A1B2C3D4].mkv',
      extractedPattern: {
        fansubGroup: 'SubsPlease',
        titlePattern: 'Frieren',
        patternType: 'fansub',
      },
      metadataId: 'meta-001',
      metadataType: 'series',
      tmdbId: 209867,
      onConfirm: noop,
      onSkip: noop,
      onError: noop,
    },
    penNode: 'screen-section',
    width: 560,
  },

  // ----- manual-search/ (P-bucket additions) -----
  {
    id: 'manual-search-fallback-status-display',
    label: 'manual-search/FallbackStatusDisplay',
    component: FallbackStatusDisplay as ComponentType<Record<string, unknown>>,
    props: {
      status: {
        attempts: [
          { source: 'tmdb', success: false },
          { source: 'douban', success: true },
          { source: 'wikipedia', success: false, skipped: true },
        ],
        totalDuration: 1850,
      },
    },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 640,
  },
  {
    id: 'manual-search-search-result-card',
    label: 'manual-search/SearchResultCard',
    component: SearchResultCard as ComponentType<Record<string, unknown>>,
    props: {
      item: {
        id: 'tmdb-550',
        source: 'tmdb',
        title: 'Fight Club',
        titleZhTW: '鬥陣俱樂部',
        year: 1999,
        mediaType: 'movie',
        overview: '一個失眠的上班族與一個肥皂商人成立了一個地下搏擊俱樂部……',
        rating: 8.4,
      },
      isSelected: false,
      onSelect: noop,
    },
    penNode: 'screen-section',
    width: 200,
  },
  {
    id: 'manual-search-search-results-grid',
    label: 'manual-search/SearchResultsGrid',
    component: SearchResultsGrid as ComponentType<Record<string, unknown>>,
    props: {
      results: [
        {
          id: 'tmdb-550',
          source: 'tmdb',
          title: 'Fight Club',
          titleZhTW: '鬥陣俱樂部',
          year: 1999,
          mediaType: 'movie',
          rating: 8.4,
        },
        {
          id: 'douban-1291546',
          source: 'douban',
          title: '霸王別姬',
          titleZhTW: '霸王別姬',
          year: 1993,
          mediaType: 'movie',
          rating: 9.6,
        },
        {
          id: 'tmdb-1396',
          source: 'tmdb',
          title: 'Breaking Bad',
          titleZhTW: '絕命毒師',
          year: 2008,
          mediaType: 'tv',
          rating: 8.9,
        },
        {
          id: 'wikipedia-frieren',
          source: 'wikipedia',
          title: 'Frieren: Beyond Journey’s End',
          titleZhTW: '葬送的芙莉蓮',
          year: 2023,
          mediaType: 'tv',
          rating: 9.1,
        },
      ],
      selectedId: 'tmdb-550',
      onSelect: noop,
      isLoading: false,
      searchedSources: ['tmdb', 'douban', 'wikipedia'],
    },
    penNode: 'screen-section',
    width: 880,
  },

  // ----- metadata-editor/ (P-bucket additions) -----
  {
    id: 'metadata-editor-cast-editor',
    label: 'metadata-editor/CastEditor',
    component: CastEditor as ComponentType<Record<string, unknown>>,
    props: {
      cast: ['布萊德彼特', '愛德華諾頓', '海倫娜寶漢卡特'],
      onAdd: noop,
      onRemove: noop,
    },
    penNode: 'screen-section',
    width: 480,
  },
  {
    id: 'metadata-editor-poster-uploader',
    label: 'metadata-editor/PosterUploader',
    component: PosterUploader as ComponentType<Record<string, unknown>>,
    props: {
      mediaId: 'media-001',
      onUpload: noop,
      onUrlSubmit: noop,
      isUploading: false,
      error: null,
    },
    penNode: 'screen-section',
    width: 520,
  },

  // ----- notifications/ (P-bucket additions) -----
  {
    id: 'notifications-new-media-notifications',
    label: 'notifications/NewMediaNotifications',
    // Fixed-position bottom-right container; toast items animate in (10ms setTimeout)
    // and auto-dismiss at 5s — Playwright disables animations, so the static rendered
    // frame is deterministic. Default-only — hover/focus on toasts is not meaningful.
    component: NewMediaNotifications as ComponentType<Record<string, unknown>>,
    props: {
      notifications: [
        {
          id: 'notif-1',
          media: {
            id: 'media-1',
            title: '鬥陣俱樂部',
            year: 1999,
            mediaType: 'movie',
            justAdded: true,
            addedAt: '2026-05-13T12:00:00Z',
          },
          timestamp: 1747130400000,
        },
        {
          id: 'notif-2',
          media: {
            id: 'media-2',
            title: '葬送的芙莉蓮',
            year: 2023,
            mediaType: 'tv',
            justAdded: true,
            addedAt: '2026-05-13T12:01:00Z',
          },
          timestamp: 1747130460000,
        },
      ],
      onDismiss: noop,
    },
    penNode: 'screen-section',
    statesOnly: ['default'],
  },
  {
    id: 'notifications-new-media-toast',
    label: 'notifications/NewMediaToast',
    component: NewMediaToast as ComponentType<Record<string, unknown>>,
    props: { title: '鬥陣俱樂部', mediaType: 'movie' },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 360,
  },
  {
    id: 'notifications-parse-complete-toast',
    label: 'notifications/ParseCompleteToast',
    component: ParseCompleteToast as ComponentType<Record<string, unknown>>,
    props: { title: '葬送的芙莉蓮 S01E01', mediaType: 'tv', status: 'success' },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 360,
  },

  // ----- parse/ (P-bucket additions) -----
  {
    id: 'parse-error-details-panel',
    label: 'parse/ErrorDetailsPanel',
    component: ErrorDetailsPanel as ComponentType<Record<string, unknown>>,
    props: {
      steps: PARSE_STEPS_FAILED,
      filename: 'Demon.Slayer.S03E01.1080p.WEB-DL.x265.mkv',
      onManualSearch: noop,
      onEditFilename: noop,
      onSkip: noop,
    },
    penNode: 'screen-section',
    width: 560,
  },
  {
    id: 'parse-layered-progress-indicator',
    label: 'parse/LayeredProgressIndicator',
    // in_progress step renders animate-pulse "搜尋中..." — Playwright disables CSS
    // animations during screenshot, so the static frame is deterministic.
    component: LayeredProgressIndicator as ComponentType<Record<string, unknown>>,
    props: { steps: PARSE_STEPS_IN_PROGRESS, currentStep: 2, compact: false },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 480,
  },
  {
    id: 'parse-media-file-card',
    label: 'parse/MediaFileCard',
    component: MediaFileCard as ComponentType<Record<string, unknown>>,
    props: {
      file: {
        id: 'gallery-mfc-uuid-0001',
        filename: 'Blade.Runner.2049.2017.1080p.BluRay.x265.mkv',
        path: '/mnt/media/movies/Blade.Runner.2049.2017.1080p.BluRay.x265.mkv',
        size: 8_589_934_592,
        mediaType: 'movie',
        parseStatus: 'success',
        parsedInfo: { title: '銀翼殺手 2049', year: 2017 },
        posterPath: null,
      },
      isParsing: false,
      onClick: noop,
    },
    penNode: 'screen-section',
    width: 240,
  },
  {
    id: 'parse-parse-status-badge',
    label: 'parse/ParseStatusBadge',
    // Pin to `success` (parsing would animate-spin even with Playwright anims disabled).
    component: ParseStatusBadge as ComponentType<Record<string, unknown>>,
    props: { status: 'success', size: 'md', showLabel: true },
    penNode: 'screen-section',
    statesOnly: ['default'],
  },

  // ----- retry/ (P-bucket additions) -----
  {
    id: 'retry-countdown-timer',
    label: 'retry/CountdownTimer',
    // CountdownTimer ticks every 1s via setInterval — inherently flaky. Pin targetTime
    // to a PAST ISO: initial secondsRemaining = 0, formatTimeRemaining(0) → '即將重試'
    // stable literal, every tick recomputes to the same 0. onComplete fires once into noop.
    component: CountdownTimer as ComponentType<Record<string, unknown>>,
    props: { targetTime: '2020-01-01T00:00:00.000Z', onComplete: noop, showIcon: true },
    penNode: 'screen-section',
    statesOnly: ['default'],
  },

  // ----- scanner/ (P-bucket additions) -----
  {
    id: 'scanner-scan-progress-card',
    label: 'scanner/ScanProgressCard',
    // useNavigate via app-shell RouterProvider; percentDone=62 pins the bar fill.
    component: ScanProgressCard as ComponentType<Record<string, unknown>>,
    props: {
      state: SCAN_STATE_ACTIVE,
      onCancel: noop,
      onToggleMinimize: noop,
      onDismiss: noop,
      isCancelling: false,
    },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 400,
  },
  {
    id: 'scanner-scan-progress-sheet',
    label: 'scanner/ScanProgressSheet',
    // Default expanded=false → captures the 64px collapsed mobile pill.
    component: ScanProgressSheet as ComponentType<Record<string, unknown>>,
    props: {
      state: SCAN_STATE_ACTIVE,
      onCancel: noop,
      onDismiss: noop,
      isCancelling: false,
    },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 400,
  },

  // ----- search/ (P-bucket additions) -----
  {
    id: 'search-search-results',
    label: 'search/SearchResults',
    // isLoading=true renders the deterministic skeleton grid (PosterCardSkeleton ×N).
    // Real results would trigger useMovieDetails / useTVShowDetails on PosterCard mount
    // → network → flake. Skeleton state is the safe baseline.
    component: SearchResults as ComponentType<Record<string, unknown>>,
    props: { isLoading: true, type: 'all', currentPage: 1, onPageChange: noop },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 960,
  },

  // ----- settings/ (P-bucket additions) -----
  {
    id: 'settings-backup-table',
    label: 'settings/BackupTable',
    // Backup shape mirrors services/backupService — see BackupTable.spec.tsx.
    component: BackupTable as ComponentType<Record<string, unknown>>,
    props: {
      backups: [
        {
          id: 'b1',
          filename: 'vido-backup-20260320-140000-v17.tar.gz',
          sizeBytes: 52428800,
          schemaVersion: 17,
          checksum: 'abc123',
          status: 'completed',
          createdAt: '2026-03-20T14:00:00Z',
        },
        {
          id: 'b2',
          filename: 'vido-backup-20260319-030000-v17.tar.gz',
          sizeBytes: 0,
          schemaVersion: 17,
          checksum: '',
          status: 'failed',
          errorMessage: 'disk full',
          createdAt: '2026-03-19T03:00:00Z',
        },
        {
          id: 'b3',
          filename: 'vido-backup-20260320-150000-v17.tar.gz',
          sizeBytes: 0,
          schemaVersion: 17,
          checksum: '',
          status: 'running',
          createdAt: '2026-03-20T15:00:00Z',
        },
      ],
      onDelete: noop,
      onVerify: noop,
      onRestore: noop,
      isDeleting: false,
      isVerifying: false,
      isRestoring: false,
    },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 760,
  },
  {
    id: 'settings-cache-type-card',
    label: 'settings/CacheTypeCard',
    component: CacheTypeCard as ComponentType<Record<string, unknown>>,
    props: {
      cacheType: { type: 'ai', label: 'AI 解析快取', sizeBytes: 52428800, entryCount: 120 },
      onClear: noop,
    },
    penNode: 'screen-section',
    width: 480,
  },
  {
    id: 'settings-connection-test-result',
    label: 'settings/ConnectionTestResult',
    component: ConnectionTestResult as ComponentType<Record<string, unknown>>,
    props: { success: true, message: '連線成功!', version: 'v4.5.2', apiVersion: '2.9.3' },
    penNode: 'screen-section',
    statesOnly: ['default'],
    width: 480,
  },
  {
    id: 'settings-log-entry',
    label: 'settings/LogEntry',
    // SystemLog from services/logService — ERROR level + source + context + hint
    // exercises the full collapsed render (badge / message / source / ts).
    component: LogEntry as ComponentType<Record<string, unknown>>,
    props: {
      log: {
        id: 1,
        level: 'ERROR',
        message: '無法連線至 TMDb API:請求逾時',
        source: 'tmdb',
        createdAt: '2026-03-18T10:30:00Z',
        context: { error_code: 'TMDB_TIMEOUT', retries: 3 },
        hint: '檢查網路連線或 TMDb 服務狀態',
      },
    },
    penNode: 'screen-section',
    width: 720,
  },
  {
    id: 'settings-log-filters',
    label: 'settings/LogFilters',
    component: LogFilters as ComponentType<Record<string, unknown>>,
    props: { level: 'ERROR', keyword: '', onLevelChange: noop, onKeywordChange: noop },
    penNode: 'screen-section',
    width: 640,
  },
  {
    id: 'settings-restore-confirm-dialog',
    label: 'settings/RestoreConfirmDialog',
    // NOT Radix-portal — plain fixed-position overlay, always renders when mounted.
    component: RestoreConfirmDialog as ComponentType<Record<string, unknown>>,
    props: {
      backup: {
        id: 'b1',
        filename: 'vido-backup-20260320-140000-v17.tar.gz',
        sizeBytes: 52428800,
        schemaVersion: 17,
        checksum: 'abc123',
        status: 'completed',
        createdAt: '2026-03-20T14:00:00Z',
      },
      isRestoring: false,
      onConfirm: noop,
      onCancel: noop,
    },
    penNode: 'screen-section',
    statesOnly: ['default'],
  },
  {
    id: 'settings-service-status-card',
    label: 'settings/ServiceStatusCard',
    component: ServiceStatusCard as ComponentType<Record<string, unknown>>,
    props: {
      service: {
        name: 'tmdb',
        displayName: 'TMDb API',
        status: 'connected',
        message: '已連線',
        lastSuccessAt: '2026-02-10T14:30:00Z',
        lastCheckAt: '2026-02-10T14:30:00Z',
        responseTimeMs: 45,
      },
      onTest: noop,
      isTesting: false,
    },
    penNode: 'screen-section',
    width: 520,
  },
  {
    id: 'settings-settings-placeholder',
    label: 'settings/SettingsPlaceholder',
    component: SettingsPlaceholder as ComponentType<Record<string, unknown>>,
    props: { icon: Database, title: '快取管理', description: '管理快取資料,釋放儲存空間' },
    penNode: 'utility',
    statesOnly: ['default'],
    width: 480,
  },

  // ----- setup/ (P-bucket additions) -----
  // All 7 setup steps receive StepProps (data/onUpdate/onNext/onBack/onSkip/isFirst/
  // isLast/isSubmitting) — pure presentational, no data hooks.
  {
    id: 'setup-api-keys-step',
    label: 'setup/ApiKeysStep',
    component: ApiKeysStep as ComponentType<Record<string, unknown>>,
    props: {
      data: { tmdbApiKey: '', aiProvider: '', aiApiKey: '' },
      onUpdate: noop,
      onNext: noop,
      onBack: noop,
      onSkip: noop,
      isFirst: false,
      isLast: false,
      isSubmitting: false,
    },
    penNode: 'screen-section',
  },
  {
    id: 'setup-complete-step',
    label: 'setup/CompleteStep',
    component: CompleteStep as ComponentType<Record<string, unknown>>,
    props: {
      data: {
        language: 'zh-TW',
        qbtUrl: 'http://localhost:8080',
        mediaFolderPath: '/media/videos',
        tmdbApiKey: 'set',
        aiProvider: 'gemini',
      },
      onUpdate: noop,
      onNext: noop,
      onBack: noop,
      isFirst: false,
      isLast: true,
      isSubmitting: false,
    },
    penNode: 'screen-section',
  },
  {
    id: 'setup-media-folder-step',
    label: 'setup/MediaFolderStep',
    component: MediaFolderStep as ComponentType<Record<string, unknown>>,
    props: {
      data: { mediaFolderPath: '/media/videos' },
      onUpdate: noop,
      onNext: noop,
      onBack: noop,
      isFirst: false,
      isLast: false,
      isSubmitting: false,
    },
    penNode: 'screen-section',
  },
  {
    id: 'setup-media-library-setup-step',
    label: 'setup/MediaLibrarySetupStep',
    // Mount useEffect calls onUpdate({ libraries: [...] }) when libraries is undefined.
    // Pre-seed with stable ids to short-circuit the effect (avoids onUpdate trigger).
    component: MediaLibrarySetupStep as ComponentType<Record<string, unknown>>,
    props: {
      data: {
        libraries: [
          { id: 'fixture-lib-1', path: '/media/movies', contentType: 'movie' },
          { id: 'fixture-lib-2', path: '/media/tv', contentType: 'series' },
        ],
      },
      onUpdate: noop,
      onNext: noop,
      onBack: noop,
      isFirst: false,
      isLast: false,
      isSubmitting: false,
    },
    penNode: 'screen-section',
  },
  {
    id: 'setup-qbittorrent-step',
    label: 'setup/QBittorrentStep',
    component: QBittorrentStep as ComponentType<Record<string, unknown>>,
    props: {
      data: { qbtUrl: 'http://localhost:8080', qbtUsername: 'admin', qbtPassword: '' },
      onUpdate: noop,
      onNext: noop,
      onBack: noop,
      onSkip: noop,
      isFirst: false,
      isLast: false,
      isSubmitting: false,
    },
    penNode: 'screen-section',
  },
  {
    id: 'setup-step-progress',
    label: 'setup/StepProgress',
    component: StepProgress as ComponentType<Record<string, unknown>>,
    props: {
      steps: [
        { id: 'welcome', title: '歡迎' },
        { id: 'qbittorrent', title: 'qBittorrent' },
        { id: 'media-folder', title: '媒體庫' },
        { id: 'api-keys', title: 'API 金鑰' },
        { id: 'complete', title: '完成' },
      ],
      currentStep: 2,
    },
    penNode: 'utility',
    statesOnly: ['default'],
    width: 320,
  },
  {
    id: 'setup-welcome-step',
    label: 'setup/WelcomeStep',
    component: WelcomeStep as ComponentType<Record<string, unknown>>,
    props: {
      data: { language: 'zh-TW' },
      onUpdate: noop,
      onNext: noop,
      onBack: noop,
      isFirst: true,
      isLast: false,
      isSubmitting: false,
    },
    penNode: 'screen-section',
  },
];
