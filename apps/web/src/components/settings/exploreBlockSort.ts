// Implements: <utility — no .pen counterpart>
import type { ExploreBlockContentType } from '../../services/exploreBlockService';

/**
 * TMDb discover `sort_by` values an explore block may use, with the words the
 * user sees. ONE list for both the edit modal's dropdown and the settings
 * row's description (dsr-3d), so a row never names a sort the modal calls
 * something else.
 */
export interface SortOption {
  value: string;
  label: string;
}

const SHARED_SORT_OPTIONS: SortOption[] = [
  { value: 'popularity.desc', label: '熱門度（高→低）' },
  { value: 'vote_average.desc', label: '評分（高→低）' },
];

export const MOVIE_SORT_OPTIONS: SortOption[] = [
  ...SHARED_SORT_OPTIONS,
  { value: 'primary_release_date.desc', label: '發行日期（新→舊）' },
  { value: 'revenue.desc', label: '票房（高→低）' },
];

export const TV_SORT_OPTIONS: SortOption[] = [
  ...SHARED_SORT_OPTIONS,
  { value: 'first_air_date.desc', label: '首播日期（新→舊）' },
];

export function getSortOptions(ct: ExploreBlockContentType): SortOption[] {
  return ct === 'tv' ? TV_SORT_OPTIONS : MOVIE_SORT_OPTIONS;
}

/** The label for a stored sort_by, looked up in the block's own list; an unknown value is shown as-is. */
export function sortLabel(
  value: string | undefined,
  ct: ExploreBlockContentType = 'movie'
): string | undefined {
  if (!value) return undefined;
  return getSortOptions(ct).find((o) => o.value === value)?.label ?? value;
}
