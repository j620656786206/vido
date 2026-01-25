/**
 * SearchResultCard Tests (Story 3.7 - AC2, AC4)
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SearchResultCard } from './SearchResultCard';
import type { ManualSearchResultItem } from '../../services/metadata';

const mockItem: ManualSearchResultItem = {
  id: 'tmdb-85937',
  source: 'tmdb',
  title: 'Demon Slayer: Kimetsu no Yaiba',
  titleZhTW: '鬼滅之刃',
  year: 2019,
  mediaType: 'tv',
  overview: 'It is the Taisho Period in Japan...',
  posterUrl: 'https://image.tmdb.org/t/p/w500/test.jpg',
  rating: 8.7,
};

describe('SearchResultCard', () => {
  const defaultProps = {
    item: mockItem,
    isSelected: false,
    onSelect: vi.fn(),
  };

  it('renders card with title', () => {
    render(<SearchResultCard {...defaultProps} />);

    // Should show Traditional Chinese title first
    expect(screen.getByText('鬼滅之刃')).toBeInTheDocument();
    // Original title as secondary
    expect(screen.getByText('Demon Slayer: Kimetsu no Yaiba')).toBeInTheDocument();
  });

  it('renders year', () => {
    render(<SearchResultCard {...defaultProps} />);

    expect(screen.getByText('2019')).toBeInTheDocument();
  });

  it('renders source badge (AC4)', () => {
    render(<SearchResultCard {...defaultProps} />);

    expect(screen.getByText('TMDB')).toBeInTheDocument();
  });

  it('renders rating', () => {
    render(<SearchResultCard {...defaultProps} />);

    expect(screen.getByText('⭐ 8.7')).toBeInTheDocument();
  });

  it('renders media type badge', () => {
    render(<SearchResultCard {...defaultProps} />);

    expect(screen.getByText('影集')).toBeInTheDocument();
  });

  it('calls onSelect when clicked', async () => {
    const onSelect = vi.fn();
    render(<SearchResultCard {...defaultProps} onSelect={onSelect} />);

    const card = screen.getByTestId('search-result-card');
    await userEvent.click(card);

    expect(onSelect).toHaveBeenCalledTimes(1);
  });

  it('shows selected indicator when isSelected is true', () => {
    render(<SearchResultCard {...defaultProps} isSelected={true} />);

    // Card should have ring styling
    const card = screen.getByTestId('search-result-card');
    expect(card).toHaveClass('ring-2');
    expect(card).toHaveClass('ring-blue-500');
  });

  it('shows overview on hover', async () => {
    render(<SearchResultCard {...defaultProps} />);

    const card = screen.getByTestId('search-result-card');
    fireEvent.mouseEnter(card);

    expect(screen.getByText(/It is the Taisho Period/)).toBeInTheDocument();
  });

  it('renders fallback when no poster', () => {
    const itemWithoutPoster = { ...mockItem, posterUrl: undefined };
    render(<SearchResultCard {...defaultProps} item={itemWithoutPoster} />);

    expect(screen.getByText('🎬')).toBeInTheDocument();
  });

  it('uses title when titleZhTW is not available', () => {
    const itemWithoutZhTW = { ...mockItem, titleZhTW: undefined };
    render(<SearchResultCard {...defaultProps} item={itemWithoutZhTW} />);

    expect(screen.getByText('Demon Slayer: Kimetsu no Yaiba')).toBeInTheDocument();
  });
});
