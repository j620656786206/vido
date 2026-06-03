import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { FilterBottomSheet } from './FilterBottomSheet';
import type { DiscoverFilters } from '../../lib/discoverFilters';

const baseFilters: DiscoverFilters = { genre: [], platform: [], sortBy: 'popularity' };

describe('FilterBottomSheet', () => {
  it('renders nothing when closed (AC #6)', () => {
    render(
      <FilterBottomSheet isOpen={false} onClose={vi.fn()} filters={baseFilters} onApply={vi.fn()} />
    );
    expect(screen.queryByTestId('filter-bottom-sheet')).not.toBeInTheDocument();
  });

  it('renders the sheet with the result-count apply label when open', () => {
    render(
      <FilterBottomSheet
        isOpen
        onClose={vi.fn()}
        filters={baseFilters}
        onApply={vi.fn()}
        resultCount={48}
      />
    );
    expect(screen.getByTestId('filter-bottom-sheet')).toBeInTheDocument();
    expect(screen.getByTestId('filter-sheet-apply')).toHaveTextContent('套用篩選（48 部結果）');
  });

  it('drafts changes locally and only commits on 套用篩選', () => {
    const onApply = vi.fn();
    const onClose = vi.fn();
    render(<FilterBottomSheet isOpen onClose={onClose} filters={baseFilters} onApply={onApply} />);
    // Editing the draft does NOT commit yet.
    fireEvent.click(screen.getByTestId('filter-genre-16'));
    expect(onApply).not.toHaveBeenCalled();

    // Apply commits the draft and closes.
    fireEvent.click(screen.getByTestId('filter-sheet-apply'));
    expect(onApply).toHaveBeenCalledWith(expect.objectContaining({ genre: [16] }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('清除全部 resets the draft without committing', () => {
    const onApply = vi.fn();
    render(
      <FilterBottomSheet
        isOpen
        onClose={vi.fn()}
        filters={{ ...baseFilters, genre: [16], ratingGte: 8 }}
        onApply={onApply}
      />
    );
    fireEvent.click(screen.getByTestId('filter-sheet-clear'));
    expect(onApply).not.toHaveBeenCalled();

    // After clearing the draft, applying commits empty filters (no rating).
    fireEvent.click(screen.getByTestId('filter-sheet-apply'));
    expect(onApply).toHaveBeenCalledWith({ genre: [], platform: [], sortBy: 'popularity' });
  });

  it('clicking the backdrop closes the sheet', () => {
    const onClose = vi.fn();
    render(<FilterBottomSheet isOpen onClose={onClose} filters={baseFilters} onApply={vi.fn()} />);
    fireEvent.click(screen.getByTestId('filter-sheet-backdrop'));
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
