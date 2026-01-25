/**
 * SearchResultsGrid Tests (Story 3.7 - AC2)
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { SearchResultsGrid } from './SearchResultsGrid';
import type { ManualSearchResultItem } from '../../services/metadata';

const mockResults: ManualSearchResultItem[] = [
  {
    id: 'tmdb-85937',
    source: 'tmdb',
    title: 'Demon Slayer: Kimetsu no Yaiba',
    titleZhTW: '鬼滅之刃',
    year: 2019,
    mediaType: 'tv',
    overview: 'It is the Taisho Period in Japan...',
    posterUrl: 'https://image.tmdb.org/t/p/w500/test.jpg',
    rating: 8.7,
  },
  {
    id: 'douban-30277296',
    source: 'douban',
    title: '鬼灭之刃',
    titleZhTW: '鬼滅之刃',
    year: 2019,
    mediaType: 'tv',
    overview: '大正時期，少年炭治郎...',
    posterUrl: 'https://img.doubanio.com/test.jpg',
    rating: 8.4,
  },
];

describe('SearchResultsGrid', () => {
  const defaultProps = {
    results: [],
    selectedId: null,
    onSelect: vi.fn(),
    isLoading: false,
    searchedSources: [],
  };

  it('renders loading skeleton when isLoading is true', () => {
    render(<SearchResultsGrid {...defaultProps} isLoading={true} />);

    // Should show loading state
    const skeletons = document.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('renders initial state when no search performed', () => {
    render(<SearchResultsGrid {...defaultProps} />);

    expect(screen.getByText(/輸入至少 2 個字元開始搜尋/)).toBeInTheDocument();
  });

  it('renders empty state when no results found', () => {
    render(
      <SearchResultsGrid
        {...defaultProps}
        results={[]}
        searchedSources={['tmdb', 'douban']}
      />
    );

    expect(screen.getByText('找不到結果')).toBeInTheDocument();
    expect(screen.getByText(/試試其他關鍵字/)).toBeInTheDocument();
  });

  it('renders results grid with correct count', () => {
    render(
      <SearchResultsGrid
        {...defaultProps}
        results={mockResults}
        searchedSources={['tmdb', 'douban']}
      />
    );

    expect(screen.getByText('找到 2 個結果')).toBeInTheDocument();
  });

  it('renders search result cards for each result', () => {
    render(
      <SearchResultsGrid
        {...defaultProps}
        results={mockResults}
        searchedSources={['tmdb', 'douban']}
      />
    );

    const cards = screen.getAllByTestId('search-result-card');
    expect(cards).toHaveLength(2);
  });

  it('shows searched sources', () => {
    render(
      <SearchResultsGrid
        {...defaultProps}
        results={mockResults}
        searchedSources={['tmdb', 'douban']}
      />
    );

    expect(screen.getByText(/來源：tmdb, douban/)).toBeInTheDocument();
  });
});
