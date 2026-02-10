import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { DownloadDetails } from './DownloadDetails';

// Mock the useDownloads hook
const mockUseDownloadDetails = vi.fn();
vi.mock('../../hooks/useDownloads', () => ({
  useDownloadDetails: (...args: unknown[]) => mockUseDownloadDetails(...args),
}));

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

describe('DownloadDetails', () => {
  it('[P1] renders loading state', () => {
    // GIVEN: Details are loading
    mockUseDownloadDetails.mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    });

    // WHEN: DownloadDetails is rendered
    renderWithProviders(<DownloadDetails hash="abc123" />);

    // THEN: Loading text is displayed
    expect(screen.getByText('載入詳細資料中...')).toBeTruthy();
  });

  it('[P1] renders error state', () => {
    // GIVEN: Details request failed
    mockUseDownloadDetails.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('Connection failed'),
    });

    // WHEN: DownloadDetails is rendered
    renderWithProviders(<DownloadDetails hash="abc123" />);

    // THEN: Error message is displayed
    expect(screen.getByText(/無法載入詳細資料/)).toBeTruthy();
    expect(screen.getByText(/Connection failed/)).toBeTruthy();
  });

  it('[P1] renders detail fields when data is loaded (AC4)', () => {
    // GIVEN: Details data is available
    mockUseDownloadDetails.mockReturnValue({
      data: {
        hash: 'abc123',
        name: 'Test Movie [1080p]',
        size: 4294967296,
        progress: 0.85,
        downloadSpeed: 10485760,
        uploadSpeed: 524288,
        eta: 600,
        status: 'downloading',
        addedOn: '2026-01-15T10:00:00Z',
        seeds: 10,
        peers: 5,
        downloaded: 3650722201,
        uploaded: 104857600,
        ratio: 0.03,
        savePath: '/downloads/movies',
        pieceSize: 4194304,
        comment: 'Test comment',
        createdBy: 'qBittorrent v4.5.2',
        creationDate: '2026-01-10T08:00:00Z',
        totalWasted: 1024,
        timeElapsed: 3600,
        seedingTime: 0,
        avgDownSpeed: 8388608,
        avgUpSpeed: 262144,
      },
      isLoading: false,
      error: null,
    });

    // WHEN: DownloadDetails is rendered
    renderWithProviders(<DownloadDetails hash="abc123" />);

    // THEN: Detail fields are displayed
    const detailsContainer = screen.getByTestId('download-details');
    expect(detailsContainer).toBeTruthy();

    // AC4: Total size / Downloaded size
    expect(screen.getByText('總大小')).toBeTruthy();
    expect(screen.getByText('已下載')).toBeTruthy();

    // AC4: Seeds / Peers count
    expect(screen.getByText('做種數')).toBeTruthy();
    expect(screen.getByText('10')).toBeTruthy();
    expect(screen.getByText('節點數')).toBeTruthy();
    expect(screen.getByText('5')).toBeTruthy();

    // AC4: Save path
    expect(screen.getByText('儲存路徑')).toBeTruthy();
    expect(screen.getByText('/downloads/movies')).toBeTruthy();

    // AC4: Added date
    expect(screen.getByText('新增時間')).toBeTruthy();

    // Average speeds
    expect(screen.getByText('平均下載速度')).toBeTruthy();
    expect(screen.getByText('平均上傳速度')).toBeTruthy();

    // Piece size
    expect(screen.getByText('分塊大小')).toBeTruthy();

    // Time elapsed
    expect(screen.getByText('已耗時間')).toBeTruthy();

    // Comment
    expect(screen.getByText('備註')).toBeTruthy();
    expect(screen.getByText('Test comment')).toBeTruthy();
  });

  it('[P1] renders completion date when torrent is completed (AC4)', () => {
    // GIVEN: Completed torrent with completedOn date
    mockUseDownloadDetails.mockReturnValue({
      data: {
        hash: 'xyz789',
        name: 'Completed Movie',
        size: 8589934592,
        progress: 1,
        downloadSpeed: 0,
        uploadSpeed: 262144,
        eta: 8640000,
        status: 'completed',
        addedOn: '2026-01-14T10:00:00Z',
        completedOn: '2026-01-15T12:00:00Z',
        seeds: 20,
        peers: 3,
        downloaded: 8589934592,
        uploaded: 1073741824,
        ratio: 0.125,
        savePath: '/downloads/series',
        pieceSize: 4194304,
        creationDate: '2026-01-10T08:00:00Z',
        totalWasted: 0,
        timeElapsed: 7200,
        seedingTime: 3600,
        avgDownSpeed: 5242880,
        avgUpSpeed: 131072,
      },
      isLoading: false,
      error: null,
    });

    // WHEN: DownloadDetails is rendered
    renderWithProviders(<DownloadDetails hash="xyz789" />);

    // THEN: Completion date is shown
    expect(screen.getByText('完成時間')).toBeTruthy();
  });

  it('[P2] does not render completion date for in-progress torrent', () => {
    // GIVEN: Downloading torrent without completedOn
    mockUseDownloadDetails.mockReturnValue({
      data: {
        hash: 'inprogress',
        name: 'In Progress Movie',
        size: 4294967296,
        progress: 0.5,
        downloadSpeed: 5242880,
        uploadSpeed: 0,
        eta: 1200,
        status: 'downloading',
        addedOn: '2026-01-15T10:00:00Z',
        seeds: 5,
        peers: 3,
        downloaded: 2147483648,
        uploaded: 0,
        ratio: 0,
        savePath: '/downloads/movies',
        pieceSize: 4194304,
        creationDate: '2026-01-10T08:00:00Z',
        totalWasted: 0,
        timeElapsed: 1800,
        seedingTime: 0,
        avgDownSpeed: 5242880,
        avgUpSpeed: 0,
      },
      isLoading: false,
      error: null,
    });

    // WHEN: DownloadDetails is rendered
    renderWithProviders(<DownloadDetails hash="inprogress" />);

    // THEN: Completion date is NOT shown
    expect(screen.queryByText('完成時間')).toBeNull();
  });
});
