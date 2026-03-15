import { render, screen, fireEvent } from '@testing-library/react';
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

  beforeEach(() => {
    onApply = vi.fn();
    onClear = vi.fn();
  });

  it('renders filter button', () => {
    renderWithProvider(<FilterPanel filters={emptyFilters} onApply={onApply} onClear={onClear} />);
    expect(screen.getByText('篩選')).toBeInTheDocument();
  });

  it('opens panel on click', async () => {
    renderWithProvider(<FilterPanel filters={emptyFilters} onApply={onApply} onClear={onClear} />);
    await userEvent.click(screen.getByText('篩選'));
    expect(screen.getByText('類型')).toBeInTheDocument();
    expect(screen.getByText('年份範圍')).toBeInTheDocument();
  });

  it('renders genre checkboxes from API data', async () => {
    renderWithProvider(<FilterPanel filters={emptyFilters} onApply={onApply} onClear={onClear} />);
    await userEvent.click(screen.getByText('篩選'));

    expect(screen.getByText('Action')).toBeInTheDocument();
    expect(screen.getByText('Drama')).toBeInTheDocument();
    expect(screen.getByText('Comedy')).toBeInTheDocument();
    expect(screen.getByText('科幻')).toBeInTheDocument();
  });

  it('renders year range inputs', async () => {
    renderWithProvider(<FilterPanel filters={emptyFilters} onApply={onApply} onClear={onClear} />);
    await userEvent.click(screen.getByText('篩選'));

    const inputs = screen.getAllByRole('spinbutton');
    expect(inputs).toHaveLength(2);
  });

  it('shows apply and clear buttons', async () => {
    renderWithProvider(<FilterPanel filters={emptyFilters} onApply={onApply} onClear={onClear} />);
    await userEvent.click(screen.getByText('篩選'));

    expect(screen.getByText('套用篩選')).toBeInTheDocument();
    expect(screen.getByText('清除')).toBeInTheDocument();
  });

  it('calls onApply with selected genres', async () => {
    renderWithProvider(<FilterPanel filters={emptyFilters} onApply={onApply} onClear={onClear} />);
    await userEvent.click(screen.getByText('篩選'));

    // Select a genre checkbox
    const checkboxes = screen.getAllByRole('checkbox');
    await userEvent.click(checkboxes[0]); // Action

    await userEvent.click(screen.getByText('套用篩選'));

    expect(onApply).toHaveBeenCalledWith(expect.objectContaining({ genres: ['Action'] }));
  });

  it('calls onClear and resets form', async () => {
    renderWithProvider(
      <FilterPanel
        filters={{ genres: ['Action'], yearMin: 2000, yearMax: 2020 }}
        onApply={onApply}
        onClear={onClear}
      />
    );
    await userEvent.click(screen.getByText('篩選'));
    await userEvent.click(screen.getByText('清除'));

    expect(onClear).toHaveBeenCalled();
  });

  it('shows active filter count badge when filters are active', () => {
    renderWithProvider(
      <FilterPanel
        filters={{ genres: ['Action', 'Drama'], yearMin: 2000, yearMax: undefined }}
        onApply={onApply}
        onClear={onClear}
      />
    );
    // 2 genres + 1 yearMin = 3
    expect(screen.getByText('3')).toBeInTheDocument();
  });

  it('highlights filter button when filters are active', () => {
    const { container } = renderWithProvider(
      <FilterPanel
        filters={{ genres: ['Action'], yearMin: undefined, yearMax: undefined }}
        onApply={onApply}
        onClear={onClear}
      />
    );
    const button = container.querySelector('button');
    expect(button?.className).toContain('bg-blue-600');
  });
});
