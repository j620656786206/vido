/**
 * ManualSearchDialog Tests (Story 3.7 - AC1, AC2, AC4)
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ManualSearchDialog } from './ManualSearchDialog';
import type { FallbackStatus } from './ManualSearchDialog';

// Mock the hooks
vi.mock('../../hooks/useManualSearch', () => ({
  useManualSearch: vi.fn(() => ({
    data: null,
    isLoading: false,
    error: null,
  })),
  useApplyMetadata: vi.fn(() => ({
    mutateAsync: vi.fn(),
    isPending: false,
    error: null,
  })),
}));

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
}

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  );
}

describe('ManualSearchDialog', () => {
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    initialQuery: '',
    mediaId: 'test-media-id',
    onSuccess: vi.fn(),
  };

  it('renders dialog when isOpen is true', () => {
    renderWithProviders(<ManualSearchDialog {...defaultProps} />);
    expect(screen.getByTestId('manual-search-dialog')).toBeInTheDocument();
  });

  it('does not render when isOpen is false', () => {
    renderWithProviders(<ManualSearchDialog {...defaultProps} isOpen={false} />);
    expect(screen.queryByTestId('manual-search-dialog')).not.toBeInTheDocument();
  });

  it('renders with initial query pre-filled', () => {
    renderWithProviders(
      <ManualSearchDialog {...defaultProps} initialQuery="Demon Slayer" />
    );
    const input = screen.getByPlaceholderText(/輸入電影或影集名稱/);
    expect(input).toHaveValue('Demon Slayer');
  });

  it('calls onClose when close button is clicked', async () => {
    const onClose = vi.fn();
    renderWithProviders(
      <ManualSearchDialog {...defaultProps} onClose={onClose} />
    );

    const closeButton = screen.getByLabelText('關閉');
    await userEvent.click(closeButton);

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('calls onClose when Escape key is pressed', async () => {
    const onClose = vi.fn();
    renderWithProviders(
      <ManualSearchDialog {...defaultProps} onClose={onClose} />
    );

    fireEvent.keyDown(document, { key: 'Escape' });

    await waitFor(() => {
      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });

  it('renders media type toggle buttons', () => {
    renderWithProviders(<ManualSearchDialog {...defaultProps} />);

    expect(screen.getByRole('button', { name: '電影' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '影集' })).toBeInTheDocument();
  });

  it('renders source selector', () => {
    renderWithProviders(<ManualSearchDialog {...defaultProps} />);

    const sourceSelect = screen.getByRole('combobox');
    expect(sourceSelect).toBeInTheDocument();
    expect(sourceSelect).toHaveValue('all');
  });

  it('renders fallback status when provided', () => {
    const fallbackStatus: FallbackStatus = {
      attempts: [
        { source: 'tmdb', success: false },
        { source: 'douban', success: false },
      ],
    };

    renderWithProviders(
      <ManualSearchDialog {...defaultProps} fallbackStatus={fallbackStatus} />
    );

    expect(screen.getByText(/已嘗試的來源/)).toBeInTheDocument();
    // Use getAllByText and check that FallbackStatusDisplay rendered the badges
    const tmdbElements = screen.getAllByText('TMDb');
    expect(tmdbElements.length).toBeGreaterThanOrEqual(1);
    // The badge with bg-red-500/20 class is from FallbackStatusDisplay
    const tmdbBadge = tmdbElements.find(el => el.closest('span')?.classList.contains('bg-red-500/20'));
    expect(tmdbBadge).toBeDefined();
  });

  it('shows initial state message when no search performed', () => {
    renderWithProviders(<ManualSearchDialog {...defaultProps} />);

    expect(screen.getByText(/輸入至少 2 個字元開始搜尋/)).toBeInTheDocument();
  });
});
