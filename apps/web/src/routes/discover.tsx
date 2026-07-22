// Design ref: ux-design.pen Screen AS-1 Advanced Filter Chips Desktop (rsAxf)
import { createFileRoute } from '@tanstack/react-router';
import type { MediaTypeFilter } from '../components/search/MediaTypeTabs';
import { DiscoverBrowseV2 } from '../components/search/DiscoverBrowseV2';
import type { SortKey } from '../lib/discoverFilters';

interface DiscoverSearchParams {
  genre?: string;
  year_gte?: number;
  year_lte?: number;
  region?: string;
  rating_gte?: number;
  platform?: string;
  sort_by?: SortKey;
  type?: MediaTypeFilter;
  page?: number;
  /** Story 13-1b — the Discover-hosted 想要清單 (nav-ADR:630, PH3-R2 lit). */
  view?: 'requests';
}

const SORT_KEYS: SortKey[] = ['popularity', 'date', 'rating'];

function toOptionalNumber(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined;
}

// genre/platform are CSV strings, but the default search parser JSON-parses a
// single numeric value (e.g. `genre=16`) into a number — coerce it back so a
// one-element deep link is not silently dropped.
function toCsvString(value: unknown): string | undefined {
  if (typeof value === 'string') return value || undefined;
  if (typeof value === 'number' && Number.isFinite(value)) return String(value);
  return undefined;
}

export const Route = createFileRoute('/discover')({
  // ux3-cutover-3: legacy branch removed — DiscoverBrowseV2 is the only render.
  validateSearch: (search: Record<string, unknown>): DiscoverSearchParams => ({
    genre: toCsvString(search.genre),
    year_gte: toOptionalNumber(search.year_gte),
    year_lte: toOptionalNumber(search.year_lte),
    region: typeof search.region === 'string' ? search.region : undefined,
    rating_gte: toOptionalNumber(search.rating_gte),
    platform: toCsvString(search.platform),
    sort_by: SORT_KEYS.includes(search.sort_by as SortKey)
      ? (search.sort_by as SortKey)
      : undefined,
    type: ['all', 'movie', 'tv'].includes(search.type as string)
      ? (search.type as MediaTypeFilter)
      : undefined,
    page: toOptionalNumber(search.page),
    // Rule 26: a string-enum guard is safe here — 'requests' can never arrive
    // as a lone JSON-parsed number (never all-digits).
    view: search.view === 'requests' ? 'requests' : undefined,
  }),
  component: DiscoverBrowseV2,
});
