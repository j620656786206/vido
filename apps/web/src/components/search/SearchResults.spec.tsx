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
    <a href={`${to.replace('$type', params.type).replace('$id', params.id)}`} {...props}>
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
      originalTitle: 'Demon Slayer: Mugen Train',
      overview: '炭治郎與夥伴登上無限列車...',
      releaseDate: '2020-10-16',
      posterPath: '/poster1.jpg',
      backdropPath: '/backdrop1.jpg',
      voteAverage: 8.5,
      voteCount: 5000,
      genreIds: [16, 28],
    },
    {
      id: 2,
      title: '鬼滅之刃 絆之奇蹟',
      originalTitle: 'Demon Slayer: Kimetsu no Yaiba',
      overview: '電影版劇情...',
      releaseDate: '2019-04-06',
      posterPath: null,
      backdropPath: null,
      voteAverage: 7.5,
      voteCount: 3000,
      genreIds: [16],
    },
  ],
  totalPages: 1,
  totalResults: 2,
};

const mockTVShows: TVShowSearchResponse = {
  page: 1,
  results: [
    {
      id: 3,
      name: '進擊的巨人',
      originalName: 'Attack on Titan',
      overview: '人類與巨人的戰鬥...',
      firstAirDate: '2013-04-07',
      posterPath: '/poster3.jpg',
      backdropPath: '/backdrop3.jpg',
      voteAverage: 9.0,
      voteCount: 10000,
      genreIds: [16, 10759],
    },
  ],
  totalPages: 1,
  totalResults: 1,
};

const mockMoviesWithPagination: MovieSearchResponse = {
  page: 1,
  results: mockMovies.results,
  totalPages: 5,
  totalResults: 100,
};

describe('SearchResults', () => {
  it('should show loading skeleton when isLoading is true', () => {
    render(<SearchResults isLoading={true} type="all" currentPage={1} onPageChange={() => {}} />);

    // Should have skeleton elements
    const skeletons = document.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('should show empty state when no results', () => {
    render(
      <SearchResults
        movies={{ page: 1, results: [], totalPages: 0, totalResults: 0 }}
        tvShows={{ page: 1, results: [], totalPages: 0, totalResults: 0 }}
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
