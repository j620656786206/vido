import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { FilterChipBar } from './FilterChipBar';
import type { DiscoverFilters } from '../../lib/discoverFilters';

const baseFilters: DiscoverFilters = { genre: [], platform: [], sortBy: 'popularity' };

describe('FilterChipBar', () => {
  it('renders nothing when there are no active filters', () => {
    const { container } = render(
      <FilterChipBar filters={baseFilters} onChange={vi.fn()} onClearAll={vi.fn()} />
    );
    expect(container.firstChild).toBeNull();
  });

  it('renders a labelled chip per active filter (AC #1)', () => {
    render(
      <FilterChipBar
        filters={{ ...baseFilters, genre: [16], region: 'JP' }}
        onChange={vi.fn()}
        onClearAll={vi.fn()}
      />
    );
    expect(screen.getByText('類型: 動畫')).toBeInTheDocument();
    expect(screen.getByText('地區: 日本')).toBeInTheDocument();
  });

  it('removing a chip emits the next filter state (AC #2)', () => {
    const onChange = vi.fn();
    render(
      <FilterChipBar
        filters={{ ...baseFilters, genre: [16, 28] }}
        onChange={onChange}
        onClearAll={vi.fn()}
      />
    );
    fireEvent.click(screen.getByLabelText('移除類型: 動畫篩選'));
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ genre: [28] }));
  });

  it('shows 清除全部 only when ≥2 filters are active (AC #3, Task 2.5)', () => {
    const { rerender } = render(
      <FilterChipBar
        filters={{ ...baseFilters, genre: [16] }}
        onChange={vi.fn()}
        onClearAll={vi.fn()}
      />
    );
    expect(screen.queryByTestId('clear-all-filters')).not.toBeInTheDocument();

    rerender(
      <FilterChipBar
        filters={{ ...baseFilters, genre: [16], region: 'JP' }}
        onChange={vi.fn()}
        onClearAll={vi.fn()}
      />
    );
    expect(screen.getByTestId('clear-all-filters')).toBeInTheDocument();
  });

  it('清除全部 calls onClearAll', () => {
    const onClearAll = vi.fn();
    render(
      <FilterChipBar
        filters={{ ...baseFilters, genre: [16], ratingGte: 8 }}
        onChange={vi.fn()}
        onClearAll={onClearAll}
      />
    );
    fireEvent.click(screen.getByTestId('clear-all-filters'));
    expect(onClearAll).toHaveBeenCalledTimes(1);
  });

  it('shows 儲存篩選 only when onSavePreset is provided and ≥1 filter active (Story 11-4 AC #1)', () => {
    // No callback → no save button.
    const { rerender } = render(
      <FilterChipBar
        filters={{ ...baseFilters, genre: [16] }}
        onChange={vi.fn()}
        onClearAll={vi.fn()}
      />
    );
    expect(screen.queryByTestId('save-preset-button')).not.toBeInTheDocument();

    // Callback provided + active filter → save button visible.
    rerender(
      <FilterChipBar
        filters={{ ...baseFilters, genre: [16] }}
        onChange={vi.fn()}
        onClearAll={vi.fn()}
        onSavePreset={vi.fn()}
      />
    );
    expect(screen.getByTestId('save-preset-button')).toBeInTheDocument();
  });

  it('儲存篩選 is hidden when no filters are active even if onSavePreset is set', () => {
    const { container } = render(
      <FilterChipBar
        filters={baseFilters}
        onChange={vi.fn()}
        onClearAll={vi.fn()}
        onSavePreset={vi.fn()}
      />
    );
    // The bar renders nothing at all when there are no chips.
    expect(container.firstChild).toBeNull();
  });

  it('儲存篩選 calls onSavePreset', () => {
    const onSavePreset = vi.fn();
    render(
      <FilterChipBar
        filters={{ ...baseFilters, genre: [16] }}
        onChange={vi.fn()}
        onClearAll={vi.fn()}
        onSavePreset={onSavePreset}
      />
    );
    fireEvent.click(screen.getByTestId('save-preset-button'));
    expect(onSavePreset).toHaveBeenCalledTimes(1);
  });
});
