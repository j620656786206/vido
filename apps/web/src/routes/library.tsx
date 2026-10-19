import { createFileRoute, useMatchRoute, Outlet, redirect } from '@tanstack/react-router';
import type { LibraryMediaType } from '../types/library';
import { LibraryBrowseV2 } from '../components/library/LibraryBrowseV2';

/**
 * No `page` / `pageSize` (disc-2026-10-library-list-triage-b2 AC #7): they were the old
 * LibraryPage's paging, read by nothing since the infinite scroll took over — its page
 * size is `PAGE_SIZE` in hooks/useLibraryInfinite.ts — yet defaulted here to 1 / 20, so
 * the first navigate() wrote `pageSize=20` into a URL that shows 36 rows a page.
 */
interface LibrarySearchParams {
  type?: LibraryMediaType;
  sortBy?: string;
  sortOrder?: string;
  view?: string;
  q?: string;
  genres?: string;
  yearMin?: number;
  yearMax?: number;
  unmatched?: boolean;
  /**
   * Comma-joined backend `chinese_subtitle` groups (has / missing / unknown) —
   * confirmed against [@contract-v1] (Story disc-2026-10-subtitle-filter-disagrees-
   * with-badges AC #2). The batch-subtitle 「查看未找到項目」 link (Story 8-11 AC #6)
   * lands here as `?chineseSubtitle=missing`. The old `?subtitleStatus=` param is no
   * longer read (an old bookmark simply loses its filter). A string guard is Rule-26
   * safe: a group can never be all digits.
   */
  chineseSubtitle?: string;
  /**
   * sub-4-3 F17 deep link: the scan-complete toast's 產生字幕 → link opens the
   * consent flow. Rule 26: the search parser JSON-parses values, so accept the
   * boolean permissively (true / 1 / '1' / 'true') instead of a strict typeof
   * guard — the gallery-route coercion precedent.
   */
  generate?: boolean;
}

export const Route = createFileRoute('/library')({
  validateSearch: (search: Record<string, unknown>): LibrarySearchParams => ({
    // No default: the type is the PATH (/library/movies, /library/tv) — a defaulted
    // 'all' only got written back as `?type=all`, even onto /library/movies (b2).
    type: ['all', 'movie', 'tv'].includes(search.type as string)
      ? (search.type as LibraryMediaType)
      : undefined,
    sortBy: typeof search.sortBy === 'string' ? search.sortBy : undefined,
    sortOrder: ['asc', 'desc'].includes(search.sortOrder as string)
      ? (search.sortOrder as string)
      : undefined,
    view: ['grid', 'list'].includes(search.view as string) ? (search.view as string) : undefined,
    q: typeof search.q === 'string' ? search.q : undefined,
    genres: typeof search.genres === 'string' ? search.genres : undefined,
    yearMin: typeof search.yearMin === 'number' ? search.yearMin : undefined,
    yearMax: typeof search.yearMax === 'number' ? search.yearMax : undefined,
    unmatched: search.unmatched === true ? true : undefined,
    chineseSubtitle:
      typeof search.chineseSubtitle === 'string' ? search.chineseSubtitle : undefined,
    generate:
      search.generate === true ||
      search.generate === 1 ||
      search.generate === '1' ||
      search.generate === 'true'
        ? true
        : undefined,
  }),
  // b2 AC #7: every URL built for /library (and its /movies, /tv children) drops the
  // dead paging params, so an old bookmark that still carries `page=1&pageSize=20`
  // loses them on its first navigate instead of keeping them forever via `...prev`.
  search: {
    middlewares: [
      ({ search, next }) => {
        const {
          page: _page,
          pageSize: _pageSize,
          ...rest
        } = next(search) as LibrarySearchParams & {
          page?: unknown;
          pageSize?: unknown;
        };
        return rest;
      },
    ],
  },
  // ux3-0-5: old ?type= deep links → clean type routes (D2). Route-level redirect
  // (never a component redirect, F1); 'all'/absent stays at /library (merged view).
  beforeLoad: ({ search }) => {
    if (search.type === 'movie' || search.type === 'tv') {
      throw redirect({
        to: search.type === 'movie' ? '/library/movies' : '/library/tv',
        search: { ...search, type: undefined },
      });
    }
  },
  component: LibraryRoute,
});

/**
 * ux3-cutover-3: the legacy LibraryPage is gone — LibraryBrowseV2 is the only
 * render. ux3-0-5 / F5 layout discipline unchanged: the Browse UI is mounted
 * ONCE here in the layout so movies↔tv preserves filter + scroll state; the
 * active type derives from the matched clean child (path markers render null
 * via the Outlet).
 */
function LibraryRoute() {
  const matchRoute = useMatchRoute();
  const type: LibraryMediaType = matchRoute({ to: '/library/movies' })
    ? 'movie'
    : matchRoute({ to: '/library/tv' })
      ? 'tv'
      : 'all';
  return (
    <>
      <LibraryBrowseV2 type={type} />
      <Outlet />
    </>
  );
}
