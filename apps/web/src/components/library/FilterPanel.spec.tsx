import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { FilterPanel } from './FilterPanel';
import type { FilterValues } from './FilterPanel';

// Mock the hooks
vi.mock('../../hooks/useLibrary', () => ({
  useLibraryGenres: () => ({
    data: ['Action', 'Drama', 'Comedy', '科幻'],
  }),
  useLibraryStats: () => ({
    data: { yearMin: 1990, yearMax: 2024, movieCount: 50, tvCount: 30, totalCount: 80 },
  }),
}));

function renderWithProvider(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

describe('FilterPanel', () => {
  const emptyFilters: FilterValues = { genres: [], yearMin: undefined, yearMax: undefined };
  let onApply: ReturnType<typeof vi.fn>;
  let onClear: ReturnType<typeof vi.fn>;
  let onTypeChange: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    onApply = vi.fn();
    onClear = vi.fn();
    onTypeChange = vi.fn();
  });

  it('renders panel heading', () => {
    renderWithProvider(
      <FilterPanel
        filters={emptyFilters}
        mediaType="all"
        onApply={onApply}
        onClear={onClear}
        onTypeChange={onTypeChange}
      />
    );
    expect(screen.getByText('篩選條件')).toBeInTheDocument();
  });

  it('renders type section with chip toggles', () => {
    renderWithProvider(
      <FilterPanel
        filters={emptyFilters}
        mediaType="all"
        onApply={onApply}
        onClear={onClear}
        onTypeChange={onTypeChange}
      />
    );
    expect(screen.getByText('全部')).toBeInTheDocument();
    expect(screen.getByText('電影')).toBeInTheDocument();
    expect(screen.getByText('影集')).toBeInTheDocument();
  });

  it('renders genre chip toggles from API data', () => {
    renderWithProvider(
      <FilterPanel
        filters={emptyFilters}
        mediaType="all"
        onApply={onApply}
        onClear={onClear}
        onTypeChange={onTypeChange}
      />
    );
    expect(screen.getByText('Action')).toBeInTheDocument();
    expect(screen.getByText('Drama')).toBeInTheDocument();
    expect(screen.getByText('Comedy')).toBeInTheDocument();
    expect(screen.getByText('科幻')).toBeInTheDocument();
  });

  it('renders decade chip toggles instead of number inputs', () => {
    renderWithProvider(
      <FilterPanel
        filters={emptyFilters}
        mediaType="all"
        onApply={onApply}
        onClear={onClear}
        onTypeChange={onTypeChange}
      />
    );
    expect(screen.getByText('2020s')).toBeInTheDocument();
    expect(screen.getByText('2010s')).toBeInTheDocument();
    expect(screen.getByText('2000s')).toBeInTheDocument();
    expect(screen.getByText('1990s')).toBeInTheDocument();
    expect(screen.getByText('更早')).toBeInTheDocument();
    // No number inputs should exist
    expect(screen.queryAllByRole('spinbutton')).toHaveLength(0);
  });

  it('shows apply and reset buttons with correct labels', () => {
    renderWithProvider(
      <FilterPanel
        filters={emptyFilters}
        mediaType="all"
        onApply={onApply}
        onClear={onClear}
        onTypeChange={onTypeChange}
      />
    );
    expect(screen.getByText('套用')).toBeInTheDocument();
    expect(screen.getByText('重置')).toBeInTheDocument();
  });

  it('calls onApply with selected genres', async () => {
    renderWithProvider(
      <FilterPanel
        filters={emptyFilters}
        mediaType="all"
        onApply={onApply}
        onClear={onClear}
        onTypeChange={onTypeChange}
      />
    );

    // Click genre chip toggle
    await userEvent.click(screen.getByTestId('filter-genre-Action'));
    await userEvent.click(screen.getByTestId('filter-apply'));

    expect(onApply).toHaveBeenCalledWith(expect.objectContaining({ genres: ['Action'] }));
  });

  it('calls onApply with decade-based year range', async () => {
    renderWithProvider(
      <FilterPanel
        filters={emptyFilters}
        mediaType="all"
        onApply={onApply}
        onClear={onClear}
        onTypeChange={onTypeChange}
      />
    );

    await userEvent.click(screen.getByTestId('filter-decade-2020s'));
    await userEvent.click(screen.getByTestId('filter-apply'));

    expect(onApply).toHaveBeenCalledWith(expect.objectContaining({ yearMin: 2020, yearMax: 2029 }));
  });

  it('calls onClear and resets form', async () => {
    renderWithProvider(
      <FilterPanel
        filters={{ genres: ['Action'], yearMin: 2000, yearMax: 2020 }}
        mediaType="all"
        onApply={onApply}
        onClear={onClear}
        onTypeChange={onTypeChange}
      />
    );
    await userEvent.click(screen.getByTestId('filter-reset'));

    expect(onClear).toHaveBeenCalled();
  });

  it('calls onTypeChange when type chip is clicked', async () => {
    renderWithProvider(
      <FilterPanel
        filters={emptyFilters}
        mediaType="all"
        onApply={onApply}
        onClear={onClear}
        onTypeChange={onTypeChange}
      />
    );

    await userEvent.click(screen.getByTestId('filter-type-movie'));
    expect(onTypeChange).toHaveBeenCalledWith('movie');
  });

  it('shows year section label as 年份', () => {
    renderWithProvider(
      <FilterPanel
        filters={emptyFilters}
        mediaType="all"
        onApply={onApply}
        onClear={onClear}
        onTypeChange={onTypeChange}
      />
    );
    expect(screen.getByText('年份')).toBeInTheDocument();
  });
});
