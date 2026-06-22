import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { LibraryFilterRail } from './LibraryFilterRail';
import type { FilterValues } from './FilterPanel';

vi.mock('../../hooks/useLibrary', () => ({
  useLibraryGenres: () => ({ data: ['Action', 'Drama'], refetch: () => {} }),
  useLibraryStats: () => ({ data: { yearMin: 1990, yearMax: 2024 } }),
}));

function renderRail(over: Partial<React.ComponentProps<typeof LibraryFilterRail>> = {}) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const props = {
    filters: { genres: [], yearMin: undefined, yearMax: undefined } as FilterValues,
    mediaType: 'all' as const,
    activeCount: 0,
    onApply: vi.fn(),
    onClear: vi.fn(),
    onTypeChange: vi.fn(),
    onCollapse: vi.fn(),
    ...over,
  };
  render(
    <QueryClientProvider client={qc}>
      <LibraryFilterRail {...props} />
    </QueryClientProvider>
  );
  return props;
}

describe('LibraryFilterRail', () => {
  beforeEach(() => vi.clearAllMocks());

  it('renders the rail with the FilterPanel in instant mode (no 套用/重置)', () => {
    renderRail();
    expect(screen.getByTestId('library-filter-rail')).toBeInTheDocument();
    expect(screen.getByTestId('filter-panel')).toBeInTheDocument();
    expect(screen.queryByTestId('filter-apply')).not.toBeInTheDocument();
  });

  it('shows the active-count badge + pinned clear-all only when filters are active', () => {
    renderRail({ activeCount: 0 });
    expect(screen.queryByTestId('library-rail-active-count')).not.toBeInTheDocument();
    expect(screen.queryByTestId('library-rail-clear-all')).not.toBeInTheDocument();
  });

  it('renders count + clear-all when active, and wires the callbacks', async () => {
    const { onCollapse, onClear } = renderRail({
      activeCount: 2,
      filters: { genres: ['Action'], yearMin: 2020, yearMax: 2029 },
    });
    expect(screen.getByTestId('library-rail-active-count')).toHaveTextContent('2');
    await userEvent.click(screen.getByTestId('library-rail-collapse'));
    expect(onCollapse).toHaveBeenCalledTimes(1);
    await userEvent.click(screen.getByTestId('library-rail-clear-all'));
    expect(onClear).toHaveBeenCalledTimes(1);
  });
});
