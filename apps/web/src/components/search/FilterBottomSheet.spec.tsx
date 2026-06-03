import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { FilterBottomSheet } from './FilterBottomSheet';
import type { DiscoverFilters } from '../../lib/discoverFilters';
import { useDiscoverResults } from '../../hooks/useDiscoverResults';

vi.mock('../../hooks/useDiscoverResults', () => ({
  useDiscoverResults: vi.fn(),
}));

const mockUseDiscoverResults = vi.mocked(useDiscoverResults);

const baseFilters: DiscoverFilters = { genre: [], platform: [], sortBy: 'popularity' };

// Count depends on the (drafted) filters + the open gate so the tests can prove
// the apply-button count is LIVE for the draft, not the committed filters.
function liveCount(filters: DiscoverFilters, _mediaType: unknown, _page: unknown, enabled = true) {
  return {
    moviesQuery: {} as never,
    tvQuery: {} as never,
    isLoading: false,
    totalResults: !enabled ? 0 : filters.genre.length ? 5 : 48,
  };
}

beforeEach(() => {
  mockUseDiscoverResults.mockReset();
  mockUseDiscoverResults.mockImplementation(liveCount as never);
});

describe('FilterBottomSheet', () => {
  it('renders nothing when closed (AC #6)', () => {
    render(
      <FilterBottomSheet
        isOpen={false}
        onClose={vi.fn()}
        filters={baseFilters}
        onApply={vi.fn()}
        mediaType="all"
      />
    );
    expect(screen.queryByTestId('filter-bottom-sheet')).not.toBeInTheDocument();
  });

  it('shows a LIVE result count that reflects the drafted filters (M1)', () => {
    // Open with no filters → unfiltered count.
    render(
      <FilterBottomSheet
        isOpen
        onClose={vi.fn()}
        filters={baseFilters}
        onApply={vi.fn()}
        mediaType="all"
      />
    );
    expect(screen.getByTestId('filter-sheet-apply')).toHaveTextContent('套用篩選（48 部結果）');

    // Drafting a genre re-counts immediately — proves the label is NOT the
    // committed-filter count but the live draft count.
    fireEvent.click(screen.getByTestId('filter-genre-16'));
    expect(screen.getByTestId('filter-sheet-apply')).toHaveTextContent('套用篩選（5 部結果）');
  });

  it('drafts changes locally and only commits on 套用篩選', () => {
    const onApply = vi.fn();
    const onClose = vi.fn();
    render(
      <FilterBottomSheet
        isOpen
        onClose={onClose}
        filters={baseFilters}
        onApply={onApply}
        mediaType="all"
      />
    );
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
        mediaType="all"
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
    render(
      <FilterBottomSheet
        isOpen
        onClose={onClose}
        filters={baseFilters}
        onApply={vi.fn()}
        mediaType="all"
      />
    );
    fireEvent.click(screen.getByTestId('filter-sheet-backdrop'));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('closes on Escape for keyboard a11y (M2)', () => {
    const onClose = vi.fn();
    render(
      <FilterBottomSheet
        isOpen
        onClose={onClose}
        filters={baseFilters}
        onApply={vi.fn()}
        mediaType="all"
      />
    );
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
