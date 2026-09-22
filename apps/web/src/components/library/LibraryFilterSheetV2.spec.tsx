import { describe, it, expect, vi, beforeEach } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { FilterValues } from './FilterPanel';
import { SORT_OPTIONS } from './SortSelector';

const g = vi.hoisted(() => ({
  genres: { data: ['動畫', '科幻'], refetch: () => {} } as Record<string, unknown>,
  list: { data: undefined as unknown, isPending: false, isError: false } as Record<string, unknown>,
  listArgs: [] as unknown[],
}));
vi.mock('../../hooks/useLibrary', () => ({
  useLibraryGenres: () => g.genres,
  useLibraryList: (params: unknown, options: unknown) => {
    g.listArgs.push([params, options]);
    return g.list;
  },
}));

import { LibraryFilterSheetV2, sheetCountParams } from './LibraryFilterSheetV2';

const EMPTY: FilterValues = { genres: [] };

function renderSheet(over: Partial<React.ComponentProps<typeof LibraryFilterSheetV2>> = {}) {
  const onApply = vi.fn();
  const onSortChange = vi.fn();
  const onOpenChange = vi.fn();
  const onClear = vi.fn();
  const view = render(
    <LibraryFilterSheetV2
      open
      onOpenChange={onOpenChange}
      sortBy="created_at"
      sortOrder="desc"
      onSortChange={onSortChange}
      filters={EMPTY}
      mediaType="movie"
      onApply={onApply}
      onClear={onClear}
      {...over}
    />
  );
  return { onApply, onSortChange, onOpenChange, onClear, view };
}

describe('LibraryFilterSheetV2 (dsr-1b-b A6p-M)', () => {
  beforeEach(() => {
    g.genres = { data: ['動畫', '科幻'], refetch: () => {} };
    g.list = { data: undefined, isPending: true, isError: false };
    g.listArgs = [];
  });

  it('[P0] sort is a radiogroup of the four SORT_OPTIONS, in their words, one checked', () => {
    renderSheet();
    const group = screen.getByRole('radiogroup', { name: '排序方式' });
    const radios = within(group).getAllByRole('radio');
    expect(radios.map((r) => r.textContent)).toEqual(SORT_OPTIONS.map((o) => o.label));
    expect(radios.filter((r) => r.getAttribute('aria-checked') === 'true')).toHaveLength(1);
    expect(screen.getByTestId('library-sort-created_at')).toHaveAttribute('data-order', 'desc');
  });

  it('[P0] tapping an unchecked sort chip picks it with its default order; tapping again flips', async () => {
    const user = userEvent.setup();
    renderSheet();
    await user.click(screen.getByTestId('library-sort-title'));
    expect(screen.getByTestId('library-sort-title')).toHaveAttribute('aria-checked', 'true');
    expect(screen.getByTestId('library-sort-title')).toHaveAttribute('data-order', 'asc');
    await user.click(screen.getByTestId('library-sort-title'));
    expect(screen.getByTestId('library-sort-title')).toHaveAttribute('data-order', 'desc');
  });

  it('[P0] the 字幕 chips are there, 全部/電影/影集 are not', () => {
    renderSheet();
    expect(screen.getByTestId('filter-subtitle-not_found')).toHaveTextContent('缺字幕');
    expect(screen.queryByTestId('filter-type-all')).not.toBeInTheDocument();
  });

  it('[P0] the footer asks the list for the draft (pageSize 1) and shows 套用篩選 · N 部', async () => {
    const user = userEvent.setup();
    g.list = { data: { totalItems: 12 }, isPending: false, isError: false };
    renderSheet();
    await user.click(screen.getByTestId('filter-subtitle-not_found'));
    // The count query is debounced 200ms behind the draft.
    await waitFor(() =>
      expect((g.listArgs.at(-1) as [Record<string, unknown>])[0].subtitleStatus).toBe('not_found')
    );
    const [params, options] = g.listArgs.at(-1) as [
      Record<string, unknown>,
      { enabled: boolean; keepPrevious?: boolean },
    ];
    expect(params).toEqual(
      sheetCountParams({ genres: [], subtitleStatus: ['not_found'] }, 'movie', 'created_at', 'desc')
    );
    expect(params.pageSize).toBe(1);
    expect(params.subtitleStatus).toBe('not_found');
    expect(options.enabled).toBe(true);
    expect(options.keepPrevious).toBe(true);
    expect(screen.getByTestId('library-filter-apply')).toHaveTextContent('套用篩選 · 12 部');
  });

  it('[P0] while the count is pending the button just says 套用篩選', () => {
    renderSheet();
    expect(screen.getByTestId('library-filter-apply')).toHaveTextContent(/^套用篩選$/);
  });

  it('[P0] 重設 empties the draft and resets sort without closing', async () => {
    const user = userEvent.setup();
    const { onOpenChange } = renderSheet({
      filters: { genres: ['動畫'], subtitleStatus: ['found'] },
      sortBy: 'title',
      sortOrder: 'asc',
    });
    expect(screen.getByTestId('filter-subtitle-found')).toHaveAttribute('aria-pressed', 'true');
    await user.click(screen.getByTestId('library-filter-reset'));
    expect(screen.getByTestId('filter-subtitle-found')).toHaveAttribute('aria-pressed', 'false');
    expect(screen.getByTestId('library-sort-created_at')).toHaveAttribute('aria-checked', 'true');
    expect(onOpenChange).not.toHaveBeenCalled();
    expect(screen.getByTestId('library-filter-reset').className).toContain('size-11');
  });

  it('[P0] 套用 commits sort + filters in one go and closes', async () => {
    const user = userEvent.setup();
    const { onApply, onSortChange, onOpenChange } = renderSheet();
    await user.click(screen.getByTestId('library-sort-title'));
    await user.click(screen.getByTestId('filter-subtitle-not_found'));
    await user.click(screen.getByTestId('library-filter-apply'));
    expect(onSortChange).toHaveBeenCalledWith('title', 'asc');
    expect(onApply).toHaveBeenCalledWith(
      expect.objectContaining({ subtitleStatus: ['not_found'] })
    );
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('[P1] 套用 with an unchanged sort does not fire onSortChange', async () => {
    const user = userEvent.setup();
    const { onSortChange, onApply } = renderSheet();
    await user.click(screen.getByTestId('library-filter-apply'));
    expect(onSortChange).not.toHaveBeenCalled();
    expect(onApply).toHaveBeenCalledTimes(1);
  });

  it('[P1] reopening re-seeds the draft from the page (a discarded draft does not linger)', async () => {
    const user = userEvent.setup();
    const { view } = renderSheet();
    await user.click(screen.getByTestId('filter-subtitle-not_found'));
    expect(screen.getByTestId('filter-subtitle-not_found')).toHaveAttribute('aria-pressed', 'true');
    view.rerender(
      <LibraryFilterSheetV2
        open={false}
        onOpenChange={vi.fn()}
        sortBy="created_at"
        sortOrder="desc"
        onSortChange={vi.fn()}
        filters={EMPTY}
        mediaType="movie"
        onApply={vi.fn()}
        onClear={vi.fn()}
      />
    );
    view.rerender(
      <LibraryFilterSheetV2
        open
        onOpenChange={vi.fn()}
        sortBy="created_at"
        sortOrder="desc"
        onSortChange={vi.fn()}
        filters={EMPTY}
        mediaType="movie"
        onApply={vi.fn()}
        onClear={vi.fn()}
      />
    );
    expect(screen.getByTestId('filter-subtitle-not_found')).toHaveAttribute(
      'aria-pressed',
      'false'
    );
  });

  it('[P1] the footer sits outside the scrolling band', () => {
    renderSheet();
    const apply = screen.getByTestId('library-filter-apply');
    const scroller = screen.getByTestId('filter-panel').closest('.overflow-y-auto');
    expect(scroller).not.toBeNull();
    expect(scroller!.contains(apply)).toBe(false);
  });
});

describe('LibraryFilterSheetV2 — CR follow-ups (dsr-1b-b)', () => {
  beforeEach(() => {
    g.genres = { data: ['動畫', '科幻'], refetch: () => {} };
    g.list = { data: { totalItems: 7 }, isPending: false, isError: false };
    g.listArgs = [];
  });

  it('[P0] the sort group is one Tab stop: only the checked chip is tabbable, arrows wrap', () => {
    renderSheet();
    const radios = screen.getAllByRole('radio');
    expect(radios.map((r) => r.tabIndex)).toEqual([0, -1, -1, -1]);
    fireEvent.keyDown(radios[0], { key: 'ArrowRight' });
    expect(screen.getByTestId('library-sort-title')).toHaveFocus();
    fireEvent.keyDown(screen.getByTestId('library-sort-title'), { key: 'ArrowLeft' });
    fireEvent.keyDown(screen.getByTestId('library-sort-created_at'), { key: 'ArrowLeft' });
    expect(screen.getByTestId('library-sort-rating')).toHaveFocus();
    fireEvent.keyDown(screen.getByTestId('library-sort-rating'), { key: 'ArrowDown' });
    expect(screen.getByTestId('library-sort-created_at')).toHaveFocus();
  });

  it('[P0] the count query is debounced: three quick taps → one new draft reaches the hook, never the intermediate ones', async () => {
    const user = userEvent.setup();
    renderSheet();
    await user.click(screen.getByTestId('filter-subtitle-found'));
    await user.click(screen.getByTestId('filter-subtitle-not_found'));
    await user.click(screen.getByTestId('filter-subtitle-not_searched'));
    await waitFor(() =>
      expect((g.listArgs.at(-1) as [Record<string, unknown>])[0].subtitleStatus).toBe(
        'found,not_found,not_searched'
      )
    );
    const seen = g.listArgs.map((a) => (a as [Record<string, unknown>])[0].subtitleStatus);
    expect(seen).not.toContain('found');
    expect(seen).not.toContain('found,not_found');
  });

  it('[P1] while the debounced params lag the draft, the number is kept and marked stale', async () => {
    const user = userEvent.setup();
    renderSheet();
    expect(screen.getByTestId('library-filter-apply-count')).not.toHaveAttribute('data-stale');
    await user.click(screen.getByTestId('filter-subtitle-found'));
    expect(screen.getByTestId('library-filter-apply-count')).toHaveAttribute('data-stale', 'true');
    expect(screen.getByTestId('library-filter-apply')).toHaveTextContent('套用篩選 · 7 部');
    await waitFor(() =>
      expect(screen.getByTestId('library-filter-apply-count')).not.toHaveAttribute('data-stale')
    );
  });
});
