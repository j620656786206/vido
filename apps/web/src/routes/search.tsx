import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { SearchBar } from '../components/search/SearchBar';
import { SearchResults } from '../components/search/SearchResults';
import { MediaTypeTabs, type MediaTypeFilter } from '../components/search/MediaTypeTabs';
import { useSearchMovies, useSearchTVShows } from '../hooks/useSearchMedia';

interface SearchParams {
  q?: string;
  page?: number;
  type?: 'all' | 'movie' | 'tv';
}

export const Route = createFileRoute('/search')({
  validateSearch: (search: Record<string, unknown>): SearchParams => ({
    q: typeof search.q === 'string' ? search.q : '',
    page: typeof search.page === 'number' ? search.page : 1,
    type: ['all', 'movie', 'tv'].includes(search.type as string)
      ? (search.type as SearchParams['type'])
      : 'all',
  }),
  component: SearchPage,
});

function SearchPage() {
  const { q, page, type } = Route.useSearch();
  const navigate = useNavigate({ from: Route.fullPath });

  const query = q || '';
  const currentPage = page || 1;
  const currentType = type || 'all';

  const moviesQuery = useSearchMovies(query, currentPage);
  const tvQuery = useSearchTVShows(query, currentPage);

  const isLoading = moviesQuery.isLoading || tvQuery.isLoading;

  // dsr-8 AC #5: a failed search is not an empty one. Only the queries the current tab
  // needs count, and only when they have nothing to show (a failed background refetch
  // keeps its cached data — dsr-2 CR #2).
  const wantMovies = currentType !== 'tv';
  const wantTV = currentType !== 'movie';
  const moviesErr = wantMovies && moviesQuery.isError && !moviesQuery.data;
  const tvErr = wantTV && tvQuery.isError && !tvQuery.data;
  const allErr = (moviesErr || tvErr) && (!wantMovies || moviesErr) && (!wantTV || tvErr);
  const failedSide: 'movie' | 'tv' | undefined = allErr
    ? undefined
    : moviesErr
      ? 'movie'
      : tvErr
        ? 'tv'
        : undefined;
  const errorCode = (
    (moviesErr ? moviesQuery.error : tvErr ? tvQuery.error : null) as { code?: string } | null
  )?.code;
  const retryFailed = () => {
    if (moviesErr) moviesQuery.refetch();
    if (tvErr) tvQuery.refetch();
  };
  // Only a failed query's refetch is a retry — a healthy side refreshing is not.
  const retrying = (moviesErr && moviesQuery.isFetching) || (tvErr && tvQuery.isFetching);
  // Tab counts ignore the current tab: both queries always run here, and 全部 sums both.
  const countUnavailable =
    (moviesQuery.isError && !moviesQuery.data) || (tvQuery.isError && !tvQuery.data);

  const handleSearch = (newQuery: string) => {
    navigate({ search: { q: newQuery, page: 1, type: currentType } });
  };

  const handlePageChange = (newPage: number) => {
    navigate({ search: { q: query, page: newPage, type: currentType } });
  };

  const handleTypeChange = (newType: MediaTypeFilter) => {
    navigate({ search: { q: query, page: 1, type: newType } });
  };

  return (
    <div>
      <div className="container mx-auto px-4 py-8">
        {/* Page-ground heading — ink, not paper: it sits on --bg-primary. */}
        <h1 className="mb-4 text-2xl font-bold text-[var(--text-primary)]">搜尋媒體</h1>
        <div className="mb-6">
          <SearchBar onSearch={handleSearch} initialQuery={query} />
        </div>

        {/* Minimum character message */}
        {query && query.length > 0 && query.length < 2 && (
          <div className="text-[var(--text-secondary)]">請輸入至少 2 個字元進行搜尋</div>
        )}

        {/* Media type tabs and search results */}
        {query.length >= 2 && (
          <>
            <MediaTypeTabs
              activeType={currentType}
              onTypeChange={handleTypeChange}
              // No counts while a side is down: 全部 would sum only the half that answered.
              movieCount={countUnavailable ? undefined : moviesQuery.data?.totalResults}
              tvCount={countUnavailable ? undefined : tvQuery.data?.totalResults}
              className="mb-6"
            />
            <SearchResults
              movies={moviesQuery.data}
              tvShows={tvQuery.data}
              isLoading={isLoading}
              type={currentType}
              currentPage={currentPage}
              onPageChange={handlePageChange}
              isError={allErr}
              failedSide={failedSide}
              errorCode={errorCode}
              onRetry={retryFailed}
              retrying={retrying}
            />
          </>
        )}
      </div>
    </div>
  );
}
