import React from 'react';
import { fireEvent, render, screen, within } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router';
import { LibraryListV2 } from './LibraryListV2';
import type { LibraryListRowData } from './LibraryListRowV2';
import type { LibraryMovie } from '../../types/library';

const row = (id: string, title: string): LibraryListRowData => ({
  id,
  type: 'movie',
  title,
  posterPath: null,
  meta: '2025 · 動作',
  voteAverage: 7.3,
  media: {
    id,
    title,
    releaseDate: '2025-01-01',
    genres: ['動作'],
    parseStatus: 'success',
    chineseSubtitle: 'zh_hant',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
  } satisfies LibraryMovie,
});

const ROWS = [row('a', '阿凡達：火與燼'), row('b', '捍衛任務：復仇芭蕾'), row('c', 'F1電影')];

function renderList(props: Partial<React.ComponentProps<typeof LibraryListV2>> = {}) {
  const rootRoute = createRootRoute({
    component: () => <LibraryListV2 rows={ROWS} {...props} />,
  });
  const detail = createRoute({
    getParentRoute: () => rootRoute,
    path: '/media/$type/$id',
    component: () => null,
  });
  const router = createRouter({
    routeTree: rootRoute.addChildren([detail]),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  });
  render(<RouterProvider router={router} />);
}

describe('LibraryListV2', () => {
  it('is one framed list with a row per item, inside a size container (AC #1)', async () => {
    renderList();
    const frame = await screen.findByTestId('library-list-v2');
    expect(frame.className).toContain('border-[var(--border-subtle)]');
    expect(frame.className).toContain('overflow-hidden');
    // The container query wraps the frame, so the breakpoints read the list's full width.
    expect(frame.parentElement?.className).toContain('@container');
    const list = within(frame).getByRole('list');
    expect(within(list).getAllByRole('listitem')).toHaveLength(3);
    expect(list.className).toContain('divide-y');
    // Corner rows round with the frame so the inset focus ring is not cut (measured in E2E).
    expect(list.className).toContain('[&>li:last-child>a]:rounded-b-');
    expect(screen.getByTestId('list-row-v2-b')).toHaveAttribute('href', '/media/movie/b');
  });

  it('has a header row with the four column names, hidden from assistive tech (AC #1, #12)', async () => {
    renderList();
    const header = await screen.findByTestId('library-list-header');
    expect(header).toHaveAttribute('aria-hidden', 'true');
    expect(header).toHaveTextContent('片名字幕評分規格');
    // Below 576 the header is not drawn; 規格 only from 672.
    expect(header.className).toContain('hidden');
    expect(header.className).toContain('@xl:flex');
    expect(screen.getByTestId('library-list-header-spec').className).toContain('@2xl:block');
  });

  // Smoke only: both read LIBRARY_LIST_COLS. The real alignment proof is the E2E's x checks.
  it('header and rows are wired to the same column template (smoke)', async () => {
    renderList();
    const header = await screen.findByTestId('library-list-header');
    const headGrid = header.querySelector('.grid');
    const rowGrid = screen.getByTestId('list-row-v2-a').querySelector('.grid');
    const cols = (el: Element | null) =>
      (el?.className ?? '').split(' ').filter((c) => c.includes('grid-cols-'));
    expect(cols(headGrid).length).toBe(3);
    expect(cols(headGrid)).toEqual(cols(rowGrid));
  });

  it('selection mode: an indicator per row, a blank header slot, aria-pressed and onSelect(id) (AC #10)', async () => {
    const onSelect = vi.fn();
    renderList({ selectable: true, selectedIds: new Set(['b']), onSelect });
    expect(await screen.findByTestId('row-select-indicator-a')).toBeInTheDocument();
    expect(screen.getByTestId('row-select-indicator-c')).toBeInTheDocument();
    expect(screen.getByTestId('library-list-header').firstElementChild?.className).toContain('w-5');
    expect(screen.getByTestId('list-row-v2-a')).toHaveAttribute('aria-pressed', 'false');
    expect(screen.getByTestId('list-row-v2-b')).toHaveAttribute('aria-pressed', 'true');
    fireEvent.click(screen.getByTestId('list-row-v2-c'));
    expect(onSelect).toHaveBeenCalledWith('c', expect.anything());
  });

  it('outside selection mode there are no indicators and no header slot', async () => {
    renderList();
    await screen.findByTestId('list-row-v2-a');
    expect(screen.queryByTestId('row-select-indicator-a')).not.toBeInTheDocument();
    expect(screen.getByTestId('library-list-header').firstElementChild?.className).not.toContain(
      'w-5'
    );
  });
});
