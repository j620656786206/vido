import { render, screen, fireEvent, act } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { FilterPanel } from './FilterPanel';
import type { DiscoverFilters } from '../../lib/discoverFilters';

const baseFilters: DiscoverFilters = { genre: [], platform: [], sortBy: 'popularity' };

describe('FilterPanel', () => {
  it('renders all filter sections', () => {
    render(<FilterPanel filters={baseFilters} onChange={vi.fn()} />);
    expect(screen.getByText('類型')).toBeInTheDocument();
    expect(screen.getByText('地區')).toBeInTheDocument();
    expect(screen.getByText('年份範圍')).toBeInTheDocument();
    expect(screen.getByText('最低評分')).toBeInTheDocument();
    expect(screen.getByText('平台')).toBeInTheDocument();
    expect(screen.getByText('排序方式')).toBeInTheDocument();
  });

  it('toggling a genre adds its id (AC #5)', () => {
    const onChange = vi.fn();
    render(<FilterPanel filters={baseFilters} onChange={onChange} />);
    fireEvent.click(screen.getByTestId('filter-genre-16'));
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ genre: [16] }));
  });

  it('toggling an active genre removes it', () => {
    const onChange = vi.fn();
    render(<FilterPanel filters={{ ...baseFilters, genre: [16] }} onChange={onChange} />);
    fireEvent.click(screen.getByTestId('filter-genre-16'));
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ genre: [] }));
  });

  it('selecting a region sets the region code; reselecting clears it', () => {
    const onChange = vi.fn();
    const { rerender } = render(<FilterPanel filters={baseFilters} onChange={onChange} />);
    fireEvent.click(screen.getByTestId('filter-region-JP'));
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ region: 'JP' }));

    rerender(<FilterPanel filters={{ ...baseFilters, region: 'JP' }} onChange={onChange} />);
    fireEvent.click(screen.getByTestId('filter-region-JP'));
    expect(onChange).toHaveBeenLastCalledWith(expect.objectContaining({ region: undefined }));
  });

  it('year inputs set yearGte/yearLte', () => {
    const onChange = vi.fn();
    render(<FilterPanel filters={baseFilters} onChange={onChange} />);
    fireEvent.change(screen.getByTestId('filter-year-gte'), { target: { value: '2020' } });
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ yearGte: 2020 }));
    fireEvent.change(screen.getByTestId('filter-year-lte'), { target: { value: '2024' } });
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ yearLte: 2024 }));
  });

  it('clearing a year input resets it to undefined', () => {
    const onChange = vi.fn();
    render(<FilterPanel filters={{ ...baseFilters, yearGte: 2020 }} onChange={onChange} />);
    fireEvent.change(screen.getByTestId('filter-year-gte'), { target: { value: '' } });
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ yearGte: undefined }));
  });

  it('selecting a min rating sets ratingGte', () => {
    const onChange = vi.fn();
    render(<FilterPanel filters={baseFilters} onChange={onChange} />);
    fireEvent.click(screen.getByTestId('filter-rating-7'));
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ ratingGte: 7 }));
  });

  it('toggling a platform adds its provider id', () => {
    const onChange = vi.fn();
    render(<FilterPanel filters={baseFilters} onChange={onChange} />);
    fireEvent.click(screen.getByTestId('filter-platform-8'));
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ platform: [8] }));
  });

  it('changing the sort dropdown updates sortBy', () => {
    const onChange = vi.fn();
    render(<FilterPanel filters={baseFilters} onChange={onChange} />);
    fireEvent.change(screen.getByTestId('filter-sort'), { target: { value: 'rating' } });
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ sortBy: 'rating' }));
  });

  it('debounces the numeric year input so typing fires onChange once (AC #4)', () => {
    vi.useFakeTimers();
    try {
      const onChange = vi.fn();
      render(<FilterPanel filters={baseFilters} onChange={onChange} debounceMs={350} />);
      const input = screen.getByTestId('filter-year-gte');
      for (const value of ['1', '19', '199', '1995']) {
        fireEvent.change(input, { target: { value } });
      }
      // Displayed value tracks typing immediately…
      expect(input).toHaveValue(1995);
      // …but the commit is debounced.
      expect(onChange).not.toHaveBeenCalled();
      act(() => vi.advanceTimersByTime(350));
      expect(onChange).toHaveBeenCalledTimes(1);
      expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ yearGte: 1995 }));
    } finally {
      vi.useRealTimers();
    }
  });

  it('keeps categorical chips instant even when debounceMs is set (AC #4)', () => {
    const onChange = vi.fn();
    render(<FilterPanel filters={baseFilters} onChange={onChange} debounceMs={350} />);
    fireEvent.click(screen.getByTestId('filter-genre-16'));
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ genre: [16] }));
  });
});
