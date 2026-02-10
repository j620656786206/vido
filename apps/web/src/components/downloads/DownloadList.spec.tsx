import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { DownloadList } from './DownloadList';
import type { Download, SortField, SortOrder } from '../../services/downloadService';

// Mock the useDownloadDetails hook used by DownloadDetails
vi.mock('../../hooks/useDownloads', () => ({
  useDownloadDetails: () => ({
    data: null,
    isLoading: true,
    error: null,
  }),
}));

const mockDownloads: Download[] = [
  {
    hash: 'abc123',
    name: 'Movie A [1080p]',
    size: 4294967296,
    progress: 0.85,
    downloadSpeed: 10485760,
    uploadSpeed: 0,
    eta: 600,
    status: 'downloading',
    addedOn: '2026-01-15T10:00:00Z',
    seeds: 10,
    peers: 5,
    downloaded: 3650722201,
    uploaded: 0,
    ratio: 0,
    savePath: '/downloads/movies',
  },
  {
    hash: 'xyz789',
    name: 'Series B S01',
    size: 8589934592,
    progress: 1,
    downloadSpeed: 0,
    uploadSpeed: 262144,
    eta: 8640000,
    status: 'completed',
    addedOn: '2026-01-14T10:00:00Z',
    seeds: 20,
    peers: 3,
    downloaded: 8589934592,
    uploaded: 1073741824,
    ratio: 0.125,
    savePath: '/downloads/series',
  },
];

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

describe('DownloadList', () => {
  let onSortChange: ReturnType<typeof vi.fn>;
  let onOrderChange: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    onSortChange = vi.fn();
    onOrderChange = vi.fn();
  });

  it('renders download items', () => {
    renderWithProviders(
      <DownloadList
        downloads={mockDownloads}
        sortField="added_on"
        sortOrder="desc"
        onSortChange={onSortChange}
        onOrderChange={onOrderChange}
      />
    );

    expect(screen.getByText('Movie A [1080p]')).toBeTruthy();
    expect(screen.getByText('Series B S01')).toBeTruthy();
  });

  it('shows empty state when no downloads', () => {
    renderWithProviders(
      <DownloadList
        downloads={[]}
        sortField="added_on"
        sortOrder="desc"
        onSortChange={onSortChange}
        onOrderChange={onOrderChange}
      />
    );

    expect(screen.getByText('目前沒有下載任務')).toBeTruthy();
  });

  it('shows item count', () => {
    renderWithProviders(
      <DownloadList
        downloads={mockDownloads}
        sortField="added_on"
        sortOrder="desc"
        onSortChange={onSortChange}
        onOrderChange={onOrderChange}
      />
    );

    expect(screen.getByText('2 個項目')).toBeTruthy();
  });

  it('renders sort dropdown', () => {
    renderWithProviders(
      <DownloadList
        downloads={mockDownloads}
        sortField="added_on"
        sortOrder="desc"
        onSortChange={onSortChange}
        onOrderChange={onOrderChange}
      />
    );

    const select = screen.getByLabelText('排序：');
    expect(select).toBeTruthy();
  });

  it('calls onSortChange when sort is changed', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <DownloadList
        downloads={mockDownloads}
        sortField="added_on"
        sortOrder="desc"
        onSortChange={onSortChange}
        onOrderChange={onOrderChange}
      />
    );

    const select = screen.getByLabelText('排序：');
    await user.selectOptions(select, 'status');
    expect(onSortChange).toHaveBeenCalledWith('status');
  });

  it('calls onOrderChange when order button is clicked', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <DownloadList
        downloads={mockDownloads}
        sortField="added_on"
        sortOrder="desc"
        onSortChange={onSortChange}
        onOrderChange={onOrderChange}
      />
    );

    const orderButton = screen.getByTitle('降冪');
    await user.click(orderButton);
    expect(onOrderChange).toHaveBeenCalledWith('asc');
  });

  it('expands download details on click', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <DownloadList
        downloads={mockDownloads}
        sortField="added_on"
        sortOrder="desc"
        onSortChange={onSortChange}
        onOrderChange={onOrderChange}
      />
    );

    const buttons = screen.getAllByRole('button');
    // Click the first download item (not the sort order button)
    const downloadButton = buttons.find((btn) => btn.textContent?.includes('Movie A'));
    expect(downloadButton).toBeTruthy();
    if (downloadButton) {
      await user.click(downloadButton);
      // After click, expanded details should be rendered
      expect(screen.getByText('載入詳細資料中...')).toBeTruthy();
    }
  });
});
