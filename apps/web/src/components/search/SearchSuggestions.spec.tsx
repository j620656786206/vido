import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { SearchSuggestions, buildNavigableItems } from './SearchSuggestions';
import type { UnifiedSearchResult } from '../../types/tmdb';

const sample: UnifiedSearchResult = {
  query: 'name',
  page: 1,
  localMovies: [],
  localTv: [],
  movies: [
    {
      id: 1,
      title: '你的名字',
      originalTitle: 'Your Name',
      overview: '',
      releaseDate: '2016-08-26',
      posterPath: '/poster1.jpg',
      backdropPath: null,
      voteAverage: 8.4,
      voteCount: 100,
      genreIds: [],
    },
  ],
  tvShows: [
    {
      id: 2,
      name: '你的名字的故事',
      originalName: 'The Story of Your Name',
      overview: '',
      firstAirDate: '2023-01-01',
      posterPath: null,
      backdropPath: null,
      voteAverage: 6.8,
      voteCount: 10,
      genreIds: [],
    },
  ],
  people: [
    {
      id: 5655,
      name: '新海誠',
      originalName: 'Makoto Shinkai',
      profilePath: null,
      knownForDepartment: 'Directing',
      popularity: 12,
      gender: 2,
    },
  ],
};

function renderSuggestions(
  overrides: Partial<React.ComponentProps<typeof SearchSuggestions>> = {}
) {
  const onSelect = vi.fn();
  const onSubmitAll = vi.fn();
  const onActiveIndexChange = vi.fn();
  render(
    <SearchSuggestions
      result={sample}
      isLoading={false}
      query="name"
      activeIndex={-1}
      onSelect={onSelect}
      onSubmitAll={onSubmitAll}
      onActiveIndexChange={onActiveIndexChange}
      {...overrides}
    />
  );
  return { onSelect, onSubmitAll, onActiveIndexChange };
}

describe('buildNavigableItems', () => {
  it('flattens movies then TV in render order; excludes people', () => {
    expect(buildNavigableItems(sample)).toEqual([
      { type: 'movie', id: 1 },
      { type: 'tv', id: 2 },
    ]);
  });

  it('returns empty array for undefined result', () => {
    expect(buildNavigableItems(undefined)).toEqual([]);
  });
});

describe('SearchSuggestions', () => {
  it('renders the three category sections 電影 / 影集 / 人物', () => {
    renderSuggestions();
    expect(screen.getByText('電影')).toBeInTheDocument();
    expect(screen.getByText('影集')).toBeInTheDocument();
    expect(screen.getByText('人物')).toBeInTheDocument();
  });

  it('renders movie title, year and rating', () => {
    renderSuggestions();
    expect(screen.getByText('你的名字')).toBeInTheDocument();
    expect(screen.getByText(/Your Name \(2016\)/)).toBeInTheDocument();
    expect(screen.getByText(/8\.4/)).toBeInTheDocument();
  });

  it('renders a person with localized department label', () => {
    renderSuggestions();
    expect(screen.getByText('新海誠')).toBeInTheDocument();
    // Directing → 導演, then original name
    expect(screen.getByText(/導演 · Makoto Shinkai/)).toBeInTheDocument();
  });

  it('calls onSelect with the media identity when a suggestion is clicked (AC #4)', () => {
    const { onSelect } = renderSuggestions();
    fireEvent.click(screen.getByText('你的名字'));
    expect(onSelect).toHaveBeenCalledWith({ type: 'movie', id: 1 });
  });

  it('calls onSubmitAll when the footer is clicked', () => {
    const { onSubmitAll } = renderSuggestions();
    fireEvent.click(screen.getByTestId('search-suggestions-submit-all'));
    expect(onSubmitAll).toHaveBeenCalled();
  });

  it('marks the active row with aria-selected', () => {
    renderSuggestions({ activeIndex: 1 }); // second navigable = the TV show
    const options = screen.getAllByRole('option');
    expect(options[0]).toHaveAttribute('aria-selected', 'false'); // movie
    expect(options[1]).toHaveAttribute('aria-selected', 'true'); // tv
  });

  it('gives each option the searchOptionId id and keeps options inside the listbox', () => {
    renderSuggestions();
    const options = screen.getAllByRole('option'); // movies + TV only (people are non-options)
    expect(options).toHaveLength(2);
    expect(options[0]).toHaveAttribute('id', 'search-option-0'); // movie
    expect(options[1]).toHaveAttribute('id', 'search-option-1'); // tv
    // Every option is a descendant of the listbox so aria-activedescendant resolves.
    const listbox = screen.getByRole('listbox');
    options.forEach((opt) => expect(listbox).toContainElement(opt));
    // The person row is rendered but is NOT an option inside the listbox.
    const person = screen.getByTestId('search-suggestion-person');
    expect(listbox).not.toContainElement(person);
  });

  it('shows the loading state', () => {
    renderSuggestions({ isLoading: true, result: undefined });
    expect(screen.getByTestId('search-suggestions-loading')).toBeInTheDocument();
  });

  it('shows the empty state when there are no results', () => {
    renderSuggestions({
      result: {
        query: 'zzz',
        page: 1,
        localMovies: [],
        localTv: [],
        movies: [],
        tvShows: [],
        people: [],
      },
      query: 'zzz',
    });
    expect(screen.getByTestId('search-suggestions-empty')).toHaveTextContent('找不到「zzz」的結果');
  });
});

// testsprite-round1 TC092 — the owned 媒體庫 section: local hits must render
// (with their LOCAL string id) even when every TMDb category is empty, so the
// search journey survives a dead/unconfigured TMDb.
describe('SearchSuggestions — owned 媒體庫 section', () => {
  const localOnly: UnifiedSearchResult = {
    query: '駭客',
    page: 1,
    localMovies: [
      {
        id: 'seed-mv-003',
        mediaType: 'movie',
        title: '駭客任務',
        originalTitle: 'The Matrix',
        releaseDate: '1999-03-31',
        posterPath: '/matrix.jpg',
      },
    ],
    localTv: [{ id: 'seed-sr-002', mediaType: 'tv', title: '怪奇物語' }],
    movies: [],
    tvShows: [],
    people: [],
  };

  it('renders the 媒體庫 section with the 已擁有 badge instead of the empty state', () => {
    renderSuggestions({ result: localOnly, query: '駭客' });
    expect(screen.getByText('媒體庫')).toBeInTheDocument();
    expect(screen.getByText('駭客任務')).toBeInTheDocument();
    expect(screen.getAllByTestId('search-suggestion-owned-badge')).toHaveLength(2);
    expect(screen.queryByTestId('search-suggestions-empty')).not.toBeInTheDocument();
  });

  it('selecting an owned row emits its LOCAL id (routes to the local detail view)', () => {
    const { onSelect } = renderSuggestions({ result: localOnly, query: '駭客' });
    fireEvent.click(screen.getByText('駭客任務'));
    expect(onSelect).toHaveBeenCalledWith({ type: 'movie', id: 'seed-mv-003' });
  });

  it('buildNavigableItems puts owned items first, before TMDb rows', () => {
    const mixed: UnifiedSearchResult = { ...sample, localMovies: localOnly.localMovies };
    expect(buildNavigableItems(mixed)).toEqual([
      { type: 'movie', id: 'seed-mv-003' },
      { type: 'movie', id: 1 },
      { type: 'tv', id: 2 },
    ]);
  });
});
