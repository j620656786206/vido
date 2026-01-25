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
    <div className="min-h-screen bg-slate-900">
      <div className="container mx-auto px-4 py-8">
        <h1 className="text-2xl font-bold text-white mb-6">搜尋媒體</h1>

        <div className="mb-6">
          <SearchBar onSearch={handleSearch} initialQuery={query} />
        </div>

        {/* Minimum character message */}
        {query && query.length > 0 && query.length < 2 && (
          <div className="text-slate-400">請輸入至少 2 個字元進行搜尋</div>
        )}

        {/* Media type tabs and search results */}
        {query.length >= 2 && (
          <>
            <MediaTypeTabs
              activeType={currentType}
              onTypeChange={handleTypeChange}
              movieCount={moviesQuery.data?.total_results}
              tvCount={tvQuery.data?.total_results}
              className="mb-6"
            />
            <SearchResults
              movies={moviesQuery.data}
              tvShows={tvQuery.data}
              isLoading={isLoading}
              type={currentType}
              currentPage={currentPage}
              onPageChange={handlePageChange}
            />
          </>
        )}
      </div>
    </div>
  );
}
