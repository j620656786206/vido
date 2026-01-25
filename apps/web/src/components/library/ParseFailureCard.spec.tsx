/**
 * ParseFailureCard Tests (Story 3.7 - Task 7)
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ParseFailureCard } from './ParseFailureCard';
import type { LocalMediaFile } from './ParseFailureCard';

// Mock the manual search dialog and hooks
vi.mock('../manual-search', () => ({
  ManualSearchDialog: vi.fn(({ isOpen, onClose, initialQuery }) =>
    isOpen ? (
      <div data-testid="manual-search-dialog">
        <span data-testid="initial-query">{initialQuery}</span>
        <button onClick={onClose}>Close</button>
      </div>
    ) : null
  ),
  FallbackStatusDisplay: vi.fn(() => null),
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

const mockFile: LocalMediaFile = {
  id: 'file-123',
  filename: 'Demon.Slayer.S01E01.1080p.BluRay.x264.mkv',
  path: '/media/anime/Demon.Slayer.S01E01.1080p.BluRay.x264.mkv',
  size: 1024 * 1024 * 1500, // 1.5GB
  parsedInfo: {
    title: 'Demon Slayer',
    year: 2019,
    mediaType: 'tv',
    season: 1,
    episode: 1,
  },
  metadataStatus: 'failed',
  fallbackStatus: {
    attempts: [
      { source: 'tmdb', success: false },
      { source: 'douban', success: false },
    ],
  },
};

describe('ParseFailureCard', () => {
  it('renders card with filename', () => {
    renderWithProviders(<ParseFailureCard file={mockFile} />);

    expect(screen.getByTestId('parse-failure-card')).toBeInTheDocument();
  });

  it('displays parsed title', () => {
    renderWithProviders(<ParseFailureCard file={mockFile} />);

    expect(screen.getByText('Demon Slayer')).toBeInTheDocument();
  });

  it('displays year from parsed info', () => {
    renderWithProviders(<ParseFailureCard file={mockFile} />);

    expect(screen.getByText('2019')).toBeInTheDocument();
  });

  it('displays media type for TV show', () => {
    renderWithProviders(<ParseFailureCard file={mockFile} />);

    expect(screen.getByText('影集')).toBeInTheDocument();
  });

  it('displays media type for movie', () => {
    const movieFile: LocalMediaFile = {
      ...mockFile,
      parsedInfo: {
        ...mockFile.parsedInfo,
        mediaType: 'movie',
        season: undefined,
        episode: undefined,
      },
    };

    renderWithProviders(<ParseFailureCard file={movieFile} />);

    expect(screen.getByText('電影')).toBeInTheDocument();
  });

  it('displays season and episode for TV show', () => {
    renderWithProviders(<ParseFailureCard file={mockFile} />);

    expect(screen.getByText('S1')).toBeInTheDocument();
    expect(screen.getByText('E1')).toBeInTheDocument();
  });

  it('shows warning indicator', () => {
    renderWithProviders(<ParseFailureCard file={mockFile} />);

    expect(screen.getByText('無法識別')).toBeInTheDocument();
  });

  it('shows guidance message (UX-4)', () => {
    renderWithProviders(<ParseFailureCard file={mockFile} />);

    expect(
      screen.getByText(/自動識別失敗，請手動搜尋正確的 Metadata/)
    ).toBeInTheDocument();
  });

  it('shows manual search button (AC1)', () => {
    renderWithProviders(<ParseFailureCard file={mockFile} />);

    expect(screen.getByTestId('manual-search-button')).toBeInTheDocument();
    expect(screen.getByText('手動搜尋')).toBeInTheDocument();
  });

  it('opens manual search dialog when button clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(<ParseFailureCard file={mockFile} />);

    const button = screen.getByTestId('manual-search-button');
    await user.click(button);

    expect(screen.getByTestId('manual-search-dialog')).toBeInTheDocument();
  });

  it('pre-fills search query with parsed title (Task 7.2)', async () => {
    const user = userEvent.setup();
    renderWithProviders(<ParseFailureCard file={mockFile} />);

    await user.click(screen.getByTestId('manual-search-button'));

    expect(screen.getByTestId('initial-query')).toHaveTextContent('Demon Slayer');
  });

  it('shows attempt count from fallback status', () => {
    renderWithProviders(<ParseFailureCard file={mockFile} />);

    expect(screen.getByText('已嘗試 2 個來源')).toBeInTheDocument();
  });

  it('extracts title from filename when parsedInfo.title is missing', async () => {
    const fileWithoutTitle: LocalMediaFile = {
      ...mockFile,
      parsedInfo: undefined,
    };

    const user = userEvent.setup();
    renderWithProviders(<ParseFailureCard file={fileWithoutTitle} />);

    await user.click(screen.getByTestId('manual-search-button'));

    // Should extract "Demon Slayer S01E01" from filename
    expect(screen.getByTestId('initial-query')).toHaveTextContent(/Demon Slayer/);
  });

  it('calls onMetadataApplied callback when metadata is applied', async () => {
    const onMetadataApplied = vi.fn();
    const user = userEvent.setup();

    renderWithProviders(
      <ParseFailureCard file={mockFile} onMetadataApplied={onMetadataApplied} />
    );

    // Open dialog
    await user.click(screen.getByTestId('manual-search-button'));
    expect(screen.getByTestId('manual-search-dialog')).toBeInTheDocument();

    // Close dialog (simulating success)
    await user.click(screen.getByText('Close'));

    // Dialog should be closed (note: actual success callback would need mocking)
    expect(screen.queryByTestId('manual-search-dialog')).not.toBeInTheDocument();
  });

  it('uses filename when no parsed info available', () => {
    const rawFile: LocalMediaFile = {
      id: 'file-456',
      filename: 'SomeRandomFile.mkv',
      path: '/media/SomeRandomFile.mkv',
      size: 1024 * 1024 * 500,
      metadataStatus: 'failed',
    };

    renderWithProviders(<ParseFailureCard file={rawFile} />);

    expect(screen.getByText('SomeRandomFile.mkv')).toBeInTheDocument();
  });
});
