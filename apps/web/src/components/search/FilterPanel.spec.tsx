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

  // ux3-discover-facet-aggregation-fe — per-chip contextual counts (AC1/#2/#7)
  describe('facet counts', () => {
    it('renders a per-chip contextual count in JetBrains Mono, keyed by id/code/value (AC1)', () => {
      render(
        <FilterPanel
          filters={baseFilters}
          onChange={vi.fn()}
          facetCounts={{
            genre: { '16': 340 },
            region: { JP: 1280 },
            rating: { '8': 56 },
            platform: { '8': 540 },
          }}
        />
      );
      const genreCount = screen.getByTestId('facet-count-genre-16');
      expect(genreCount).toHaveTextContent('340');
      expect(genreCount).toHaveClass('font-mono', 'tabular-nums');
      expect(screen.getByTestId('facet-count-region-JP')).toHaveTextContent('1,280');
      expect(screen.getByTestId('facet-count-rating-8')).toHaveTextContent('56');
      expect(screen.getByTestId('facet-count-platform-8')).toHaveTextContent('540');
    });

    it('dims a 0-result chip but keeps it clickable — NOT disabled (AC2)', () => {
      const onChange = vi.fn();
      render(
        <FilterPanel
          filters={baseFilters}
          onChange={onChange}
          facetCounts={{ genre: { '16': 0 } }}
        />
      );
      const chip = screen.getByTestId('filter-genre-16');
      // Dimmed (opacity-70) — assert the class, not toBeVisible (Rule 16 / memory).
      expect(chip).toHaveClass('opacity-70');
      expect(chip).not.toBeDisabled();
      // Still fires onChange — the user can switch to a dead-end facet.
      fireEvent.click(chip);
      expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ genre: [16] }));
      expect(screen.getByTestId('facet-count-genre-16')).toHaveTextContent('0');
    });

    it('shows the computing "–" placeholder for an unresolved (partial) facet, no spinner (AC4/AC5)', () => {
      render(
        <FilterPanel
          filters={baseFilters}
          onChange={vi.fn()}
          // genre resolved; platform key 8 omitted (partial fill)
          facetCounts={{ genre: { '16': 12 } }}
        />
      );
      expect(screen.getByTestId('facet-count-genre-16')).toHaveTextContent('12');
      const computing = screen.getByTestId('facet-count-platform-8');
      expect(computing).toHaveTextContent('–');
      // No per-chip spinner element.
      expect(computing.querySelector('svg')).toBeNull();
    });

    it('colors the count by state: active=accent, resting=muted, dead-end/computing=disabled (AC1)', () => {
      render(
        <FilterPanel
          filters={{ ...baseFilters, genre: [16] }} // 動畫 active
          onChange={vi.fn()}
          facetCounts={{ genre: { '16': 88, '28': 120, '35': 0 } }} // 16 active, 28 resting, 35 dead-end
        />
      );
      // active chip → accent (the count signals "worth picking" the selected facet)
      expect(screen.getByTestId('facet-count-genre-16')).toHaveClass('text-[var(--accent-text)]');
      // resting chip with a positive count → muted
      expect(screen.getByTestId('facet-count-genre-28')).toHaveClass('text-[var(--text-muted)]');
      // dead-end (0) → disabled token (also dims the chip via opacity-70)
      expect(screen.getByTestId('facet-count-genre-35')).toHaveClass('text-[var(--text-disabled)]');
      // counts are sighted-only — must not leak into the chip's accessible name (AC2 a11y)
      expect(screen.getByTestId('facet-count-genre-16')).toHaveAttribute('aria-hidden', 'true');
    });

    it('renders NO counts when facetCounts is omitted — mobile sheet / fallback stays count-less (AC7/AC8)', () => {
      render(<FilterPanel filters={baseFilters} onChange={vi.fn()} />);
      expect(screen.queryByTestId('facet-count-genre-16')).not.toBeInTheDocument();
      expect(screen.queryByTestId('facet-count-platform-8')).not.toBeInTheDocument();
      // The chip itself is unchanged (not dimmed).
      expect(screen.getByTestId('filter-genre-16')).not.toHaveClass('opacity-70');
    });
  });
});
