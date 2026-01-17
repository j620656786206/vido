import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { SearchResults } from './SearchResults';
import type { MovieSearchResponse, TVShowSearchResponse } from '../../types/tmdb';

// Mock TanStack Router
vi.mock('@tanstack/react-router', () => ({
  Link: ({
    children,
    to,
    params,
    ...props
  }: {
    children: React.ReactNode;
    to: string;
    params: Record<string, string>;
  }) => (
    <a
      href={`${to.replace('$type', params.type).replace('$id', params.id)}`}
      {...props}
    >
      {children}
    </a>
  ),
}));

const mockMovies: MovieSearchResponse = {
  page: 1,
  results: [
    {
      id: 1,
      title: '鬼滅之刃 無限列車篇',
      original_title: 'Demon Slayer: Mugen Train',
      overview: '炭治郎與夥伴登上無限列車...',
      release_date: '2020-10-16',
      poster_path: '/poster1.jpg',
      backdrop_path: '/backdrop1.jpg',
      vote_average: 8.5,
      vote_count: 5000,
      genre_ids: [16, 28],
    },
    {
      id: 2,
      title: '鬼滅之刃 絆之奇蹟',
      original_title: 'Demon Slayer: Kimetsu no Yaiba',
      overview: '電影版劇情...',
      release_date: '2019-04-06',
      poster_path: null,
      backdrop_path: null,
      vote_average: 7.5,
      vote_count: 3000,
      genre_ids: [16],
    },
  ],
  total_pages: 1,
  total_results: 2,
};

const mockTVShows: TVShowSearchResponse = {
  page: 1,
  results: [
    {
      id: 3,
      name: '進擊的巨人',
      original_name: 'Attack on Titan',
      overview: '人類與巨人的戰鬥...',
      first_air_date: '2013-04-07',
      poster_path: '/poster3.jpg',
      backdrop_path: '/backdrop3.jpg',
      vote_average: 9.0,
      vote_count: 10000,
      genre_ids: [16, 10759],
    },
  ],
  total_pages: 1,
  total_results: 1,
};

const mockMoviesWithPagination: MovieSearchResponse = {
  page: 1,
  results: mockMovies.results,
  total_pages: 5,
  total_results: 100,
};

describe('SearchResults', () => {
  it('should show loading skeleton when isLoading is true', () => {
    render(
      <SearchResults
        isLoading={true}
        type="all"
        currentPage={1}
        onPageChange={() => {}}
      />
    );

    // Should have skeleton elements
    const skeletons = document.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('should show empty state when no results', () => {
    render(
      <SearchResults
        movies={{ page: 1, results: [], total_pages: 0, total_results: 0 }}
        tvShows={{ page: 1, results: [], total_pages: 0, total_results: 0 }}
        isLoading={false}
        type="all"
        currentPage={1}
        onPageChange={() => {}}
      />
    );

    // MediaGrid shows combined empty message
    expect(screen.getByText(/找不到符合的結果/)).toBeInTheDocument();
  });

  it('should display movie results with Traditional Chinese title prominently', () => {
    render(
      <SearchResults
        movies={mockMovies}
        isLoading={false}
        type="movie"
        currentPage={1}
        onPageChange={() => {}}
      />
    );

    expect(screen.getByText('鬼滅之刃 無限列車篇')).toBeInTheDocument();
  });

  it('should display TV show results', () => {
    render(
      <SearchResults
        tvShows={mockTVShows}
        isLoading={false}
        type="tv"
        currentPage={1}
        onPageChange={() => {}}
      />
    );

    expect(screen.getByText('進擊的巨人')).toBeInTheDocument();
  });

  it('should display combined results for type "all"', () => {
    render(
      <SearchResults
        movies={mockMovies}
        tvShows={mockTVShows}
        isLoading={false}
        type="all"
        currentPage={1}
        onPageChange={() => {}}
      />
    );

    // Should show movies
    expect(screen.getByText('鬼滅之刃 無限列車篇')).toBeInTheDocument();
    // Should show TV shows
    expect(screen.getByText('進擊的巨人')).toBeInTheDocument();
  });

  it('should show media type badges', () => {
    render(
      <SearchResults
        movies={mockMovies}
        tvShows={mockTVShows}
        isLoading={false}
        type="all"
        currentPage={1}
        onPageChange={() => {}}
      />
    );

    expect(screen.getAllByText('電影').length).toBeGreaterThan(0);
    expect(screen.getAllByText('影集').length).toBeGreaterThan(0);
  });

  it('should show year from release date', () => {
    render(
      <SearchResults
        movies={mockMovies}
        isLoading={false}
        type="movie"
        currentPage={1}
        onPageChange={() => {}}
      />
    );

    expect(screen.getByText('2020')).toBeInTheDocument();
    expect(screen.getByText('2019')).toBeInTheDocument();
  });

  it('should show total results count', () => {
    render(
      <SearchResults
        movies={mockMovies}
        tvShows={mockTVShows}
        isLoading={false}
        type="all"
        currentPage={1}
        onPageChange={() => {}}
      />
    );

    expect(screen.getByText('找到 3 個結果')).toBeInTheDocument();
  });

  it('should handle missing poster gracefully', () => {
    render(
      <SearchResults
        movies={mockMovies}
        isLoading={false}
        type="movie"
        currentPage={1}
        onPageChange={() => {}}
      />
    );

    // Movie with null poster shows fallback placeholder with emoji
    expect(screen.getByTestId('poster-fallback')).toBeInTheDocument();
  });

  it('should filter results by type', () => {
    render(
      <SearchResults
        movies={mockMovies}
        tvShows={mockTVShows}
        isLoading={false}
        type="movie"
        currentPage={1}
        onPageChange={() => {}}
      />
    );

    // Should only show movies
    expect(screen.getByText('鬼滅之刃 無限列車篇')).toBeInTheDocument();
    expect(screen.queryByText('進擊的巨人')).not.toBeInTheDocument();
  });

  it('should show pagination when total pages > 1', () => {
    render(
      <SearchResults
        movies={mockMoviesWithPagination}
        isLoading={false}
        type="movie"
        currentPage={1}
        onPageChange={() => {}}
      />
    );

    expect(screen.getByLabelText('分頁導航')).toBeInTheDocument();
  });

  it('should not show pagination when total pages is 1', () => {
    render(
      <SearchResults
        movies={mockMovies}
        isLoading={false}
        type="movie"
        currentPage={1}
        onPageChange={() => {}}
      />
    );

    expect(screen.queryByLabelText('分頁導航')).not.toBeInTheDocument();
  });

  it('should call onPageChange when clicking pagination', () => {
    const onPageChange = vi.fn();
    render(
      <SearchResults
        movies={mockMoviesWithPagination}
        isLoading={false}
        type="movie"
        currentPage={1}
        onPageChange={onPageChange}
      />
    );

    fireEvent.click(screen.getByLabelText('第 2 頁'));
    expect(onPageChange).toHaveBeenCalledWith(2);
  });
});
