import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { LibraryGridSkeletonV2, LibraryNoResultV2, LibraryErrorV2 } from './LibraryStatesV2';
import { LIBRARY_GRID_COLS } from './libraryGridCols';

describe('LibraryStatesV2', () => {
  it('skeleton renders the requested number of placeholder cards and is busy', () => {
    render(<LibraryGridSkeletonV2 count={6} />);
    const skeleton = screen.getByTestId('library-grid-skeleton');
    expect(skeleton).toHaveAttribute('aria-busy', 'true');
    expect(skeleton.querySelectorAll('.aspect-\\[2\\/3\\]')).toHaveLength(6);
  });

  it('no-result calls onClearFilters and is distinct from empty', () => {
    const onClear = vi.fn();
    render(<LibraryNoResultV2 onClearFilters={onClear} />);
    expect(screen.getByTestId('library-no-result')).toHaveTextContent('找不到符合的結果');
    fireEvent.click(screen.getByTestId('clear-all-filters'));
    expect(onClear).toHaveBeenCalledTimes(1);
  });

  // A7p-D: the screen names the media type AND the filters that are actually on.
  // 「試著調整或清除篩選」 with no subject makes you go hunting for what excluded
  // everything; the filters are right there, so say them.
  it('no-result names the type and the live filters when there are any', () => {
    render(
      <LibraryNoResultV2
        onClearFilters={vi.fn()}
        mediaType="movie"
        activeFilters={['動畫', '2010s']}
      />
    );
    expect(screen.getByTestId('library-no-result')).toHaveTextContent(
      '沒有電影符合目前的篩選條件（動畫、2010s）。試著調整或清除篩選。'
    );
  });

  it('no-result falls back to the generic line when nothing is filtering', () => {
    render(<LibraryNoResultV2 onClearFilters={vi.fn()} mediaType="all" activeFilters={[]} />);
    // Naming a filter that is not on would be a lie; the generic line is correct here.
    expect(screen.getByTestId('library-no-result')).toHaveTextContent(
      '試著調整或清除目前的篩選條件。'
    );
  });

  it('error shows the code, offers retry, and is an alert (fail-soft)', () => {
    const onRetry = vi.fn();
    render(<LibraryErrorV2 code="DB_QUERY_FAILED" onRetry={onRetry} />);
    const err = screen.getByTestId('library-error');
    expect(err).toHaveAttribute('role', 'alert');
    expect(err).toHaveTextContent('DB_QUERY_FAILED');
    // A8p-D — the first thought when a media library fails to load is 「我的檔案還
    // 在嗎」. 「請稍後再試」 does not answer it.
    expect(err).toHaveTextContent('媒體庫資料查詢失敗，你的檔案沒有受影響。');
    fireEvent.click(screen.getByTestId('library-error-retry'));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  // dsr-1b-c AC #2: the skeleton and the real grid share ONE column table, so the page
  // cannot reflow between "loading" and "loaded" (it did: lg 4→3, xl 6→4).
  describe('skeleton ↔ grid column parity (dsr-1b-c)', () => {
    const tokens = (el: Element) => (el.getAttribute('class') ?? '').split(/\s+/).filter(Boolean);
    const colTokens = (cls: string) => cls.split(/\s+/).filter((t) => t.includes('grid-cols'));

    it("[P0] rail-open skeleton carries exactly the grid's rail-open column classes", () => {
      render(<LibraryGridSkeletonV2 />);
      const got = tokens(screen.getByTestId('library-grid-skeleton')).filter((t) =>
        t.includes('grid-cols')
      );
      expect(got).toEqual(colTokens(LIBRARY_GRID_COLS.railOpen));
    });

    it("[P0] railCollapsed switches to the grid's rail-collapsed column classes", () => {
      render(<LibraryGridSkeletonV2 railCollapsed />);
      const got = tokens(screen.getByTestId('library-grid-skeleton')).filter((t) =>
        t.includes('grid-cols')
      );
      expect(got).toEqual(colTokens(LIBRARY_GRID_COLS.railCollapsed));
      expect(LIBRARY_GRID_COLS.railCollapsed).not.toBe(LIBRARY_GRID_COLS.railOpen);
    });

    it('[P0] the skeleton gap matches the grid gap (phone 16, sm 12, md+ 16)', () => {
      render(<LibraryGridSkeletonV2 />);
      expect(tokens(screen.getByTestId('library-grid-skeleton'))).toEqual(
        expect.arrayContaining(LIBRARY_GRID_COLS.gap.split(/\s+/))
      );
    });

    it('[P0] the phone gap is 16px (A3p-M `wW2oF` gap $Space/lg), sm keeps the old 12', () => {
      const g = LIBRARY_GRID_COLS.gap.split(/\s+/);
      expect(g[0]).toBe('gap-4');
      expect(g).toContain('sm:gap-3');
      expect(g).toContain('md:gap-4');
    });

    it('[P1] each tile mirrors PosterCardV2: poster, then a text block with the 2.75em title reserve and an 11px meta line', () => {
      render(<LibraryGridSkeletonV2 count={2} />);
      const tiles = screen.getByTestId('library-grid-skeleton').children;
      expect(tiles).toHaveLength(2);
      const tile = tiles[0];
      // Same box model as the card: [poster, textBlock] — not three loose bars.
      expect(tile.children).toHaveLength(2);
      expect(tokens(tile.children[0])).toContain('aspect-[2/3]');
      const text = tile.children[1];
      expect(text.children).toHaveLength(2);
      expect(tokens(text.children[0])).toEqual(
        expect.arrayContaining(['min-h-[2.75em]', 'text-sm', 'leading-snug'])
      );
      expect(tokens(text.children[1])).toEqual(
        expect.arrayContaining(['mt-0.5', 'font-mono', 'text-[11px]'])
      );
      expect(text.children[1].firstElementChild!.getAttribute('class')).toContain('w-[60px]');
    });
  });
});
