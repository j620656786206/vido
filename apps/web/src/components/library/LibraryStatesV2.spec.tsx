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
    expect(screen.getByTestId('library-no-result')).toHaveTextContent('沒有符合條件的項目');
    fireEvent.click(screen.getByTestId('clear-all-filters'));
    expect(onClear).toHaveBeenCalledTimes(1);
  });

  it('error shows the code, offers retry, and is an alert (fail-soft)', () => {
    const onRetry = vi.fn();
    render(<LibraryErrorV2 code="DB_QUERY_FAILED" onRetry={onRetry} />);
    const err = screen.getByTestId('library-error');
    expect(err).toHaveAttribute('role', 'alert');
    expect(err).toHaveTextContent('DB_QUERY_FAILED');
    fireEvent.click(screen.getByTestId('library-error-retry'));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });
});
