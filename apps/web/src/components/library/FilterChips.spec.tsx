import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { FilterChips } from './FilterChips';

describe('FilterChips', () => {
  let onRemoveGenre: ReturnType<typeof vi.fn>;
  let onRemoveYearMin: ReturnType<typeof vi.fn>;
  let onRemoveYearMax: ReturnType<typeof vi.fn>;
  let onClearAll: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    onRemoveGenre = vi.fn();
    onRemoveYearMin = vi.fn();
    onRemoveYearMax = vi.fn();
    onClearAll = vi.fn();
  });

  it('renders nothing when no filters are active', () => {
    const { container } = render(
      <FilterChips
        filters={{ genres: [], yearMin: undefined, yearMax: undefined }}
        onRemoveGenre={onRemoveGenre}
        onRemoveYearMin={onRemoveYearMin}
        onRemoveYearMax={onRemoveYearMax}
        onClearAll={onClearAll}
      />
    );
    expect(container.firstChild).toBeNull();
  });

  it('renders genre chips', () => {
    render(
      <FilterChips
        filters={{ genres: ['Action', '科幻'], yearMin: undefined, yearMax: undefined }}
        onRemoveGenre={onRemoveGenre}
        onRemoveYearMin={onRemoveYearMin}
        onRemoveYearMax={onRemoveYearMax}
        onClearAll={onClearAll}
      />
    );
    expect(screen.getByText('Action')).toBeInTheDocument();
    expect(screen.getByText('科幻')).toBeInTheDocument();
  });

  it('renders year range chips', () => {
    render(
      <FilterChips
        filters={{ genres: [], yearMin: 2000, yearMax: 2020 }}
        onRemoveGenre={onRemoveGenre}
        onRemoveYearMin={onRemoveYearMin}
        onRemoveYearMax={onRemoveYearMax}
        onClearAll={onClearAll}
      />
    );
    expect(screen.getByText('2000 年起')).toBeInTheDocument();
    expect(screen.getByText('至 2020 年')).toBeInTheDocument();
  });

  it('calls onRemoveGenre when clicking X on genre chip', async () => {
    render(
      <FilterChips
        filters={{ genres: ['Action'], yearMin: undefined, yearMax: undefined }}
        onRemoveGenre={onRemoveGenre}
        onRemoveYearMin={onRemoveYearMin}
        onRemoveYearMax={onRemoveYearMax}
        onClearAll={onClearAll}
      />
    );
    await userEvent.click(screen.getByLabelText('移除 Action 篩選'));
    expect(onRemoveGenre).toHaveBeenCalledWith('Action');
  });

  it('calls onRemoveYearMin when clicking X on year min chip', async () => {
    render(
      <FilterChips
        filters={{ genres: [], yearMin: 2000, yearMax: undefined }}
        onRemoveGenre={onRemoveGenre}
        onRemoveYearMin={onRemoveYearMin}
        onRemoveYearMax={onRemoveYearMax}
        onClearAll={onClearAll}
      />
    );
    await userEvent.click(screen.getByLabelText('移除最早年份篩選'));
    expect(onRemoveYearMin).toHaveBeenCalled();
  });

  it('calls onRemoveYearMax when clicking X on year max chip', async () => {
    render(
      <FilterChips
        filters={{ genres: [], yearMin: undefined, yearMax: 2020 }}
        onRemoveGenre={onRemoveGenre}
        onRemoveYearMin={onRemoveYearMin}
        onRemoveYearMax={onRemoveYearMax}
        onClearAll={onClearAll}
      />
    );
    await userEvent.click(screen.getByLabelText('移除最晚年份篩選'));
    expect(onRemoveYearMax).toHaveBeenCalled();
  });

  it('[P2] renders all chips with unified blue color scheme', () => {
    const { container } = render(
      <FilterChips
        filters={{ genres: ['Action'], yearMin: 2000, yearMax: 2020 }}
        onRemoveGenre={onRemoveGenre}
        onRemoveYearMin={onRemoveYearMin}
        onRemoveYearMax={onRemoveYearMax}
        onClearAll={onClearAll}
      />
    );
    // All chip spans should use blue styling (not green)
    const chipSpans = container.querySelectorAll('span.rounded-full');
    expect(chipSpans.length).toBe(3); // 1 genre + yearMin + yearMax
    chipSpans.forEach((span) => {
      expect(span.className).toContain('bg-blue-600/20');
      expect(span.className).toContain('text-blue-300');
      expect(span.className).not.toContain('green');
    });
  });

  it('shows clear all button and calls onClearAll', async () => {
    render(
      <FilterChips
        filters={{ genres: ['Action'], yearMin: 2000, yearMax: 2020 }}
        onRemoveGenre={onRemoveGenre}
        onRemoveYearMin={onRemoveYearMin}
        onRemoveYearMax={onRemoveYearMax}
        onClearAll={onClearAll}
      />
    );
    const clearButton = screen.getByText('清除全部篩選');
    expect(clearButton).toBeInTheDocument();

    await userEvent.click(clearButton);
    expect(onClearAll).toHaveBeenCalled();
  });
});
