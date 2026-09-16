import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { LibraryGridSkeletonV2, LibraryNoResultV2, LibraryErrorV2 } from './LibraryStatesV2';

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
});
