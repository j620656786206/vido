import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ScannerSettings } from './ScannerSettings';

const mockTriggerScan = vi.fn();
const mockUpdateSchedule = vi.fn();

const mockDeleteLibrary = vi.fn();
const mockCreateLibrary = vi.fn();
const mockUpdateLibrary = vi.fn();
const mockAddPath = vi.fn();
const mockRemovePath = vi.fn();
const mockRefreshPaths = vi.fn();

vi.mock('../../hooks/useMediaLibrary', () => ({
  useMediaLibraries: vi.fn(() => ({
    data: {
      libraries: [
        {
          id: '1',
          name: '電影庫',
          contentType: 'movie',
          paths: [{ path: '/media/movies' }],
          mediaCount: 42,
        },
      ],
    },
    isLoading: false,
    error: null,
  })),
  useMediaLibrary: vi.fn(() => ({ data: null, isLoading: false })),
  useCreateLibrary: vi.fn(() => ({ mutateAsync: mockCreateLibrary, isPending: false })),
  useUpdateLibrary: vi.fn(() => ({ mutateAsync: mockUpdateLibrary, isPending: false })),
  useDeleteLibrary: vi.fn(() => ({ mutateAsync: mockDeleteLibrary, isPending: false })),
  useAddLibraryPath: vi.fn(() => ({ mutateAsync: mockAddPath, isPending: false })),
  useRemoveLibraryPath: vi.fn(() => ({ mutateAsync: mockRemovePath, isPending: false })),
  useRefreshLibraryPaths: vi.fn(() => ({ mutateAsync: mockRefreshPaths, isPending: false })),
  libraryKeys: { all: ['libraries'], detail: (id: string) => ['libraries', id] },
}));

vi.mock('../../hooks/useScanner', () => ({
  useScanStatus: vi.fn(() => ({
    data: {
      isScanning: false,
      filesFound: 0,
      filesProcessed: 0,
      currentFile: '',
      percentDone: 0,
      errorCount: 0,
      estimatedTime: '',
      lastScanAt: '2026-03-22T14:30:00Z',
      lastScanFiles: 1247,
      lastScanDuration: '3 分 12 秒',
    },
    isLoading: false,
  })),
  useTriggerScan: vi.fn(() => ({
    mutateAsync: mockTriggerScan,
    isPending: false,
  })),
  useScanSchedule: vi.fn(() => ({
    data: { frequency: 'hourly' },
    isLoading: false,
  })),
  useUpdateScanSchedule: vi.fn(() => ({
    mutateAsync: mockUpdateSchedule,
    isPending: false,
  })),
}));

function renderWithProviders() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <ScannerSettings />
    </QueryClientProvider>
  );
}

describe('ScannerSettings', () => {
  beforeEach(() => {
    mockTriggerScan.mockReset();
    mockUpdateSchedule.mockReset();
  });

  it('renders scanner settings section', () => {
    renderWithProviders();
    expect(screen.getByText('媒體庫掃描')).toBeInTheDocument();
    expect(screen.getByTestId('scanner-settings')).toBeInTheDocument();
  });

  it('displays media library manager', () => {
    renderWithProviders();
    expect(screen.getByTestId('media-library-manager')).toBeInTheDocument();
  });

  it('renders schedule selector with current value', () => {
    renderWithProviders();
    const select = screen.getByTestId('schedule-select') as HTMLSelectElement;
    expect(select.value).toBe('hourly');
  });

  it('displays last scan info', () => {
    renderWithProviders();
    const lastScan = screen.getByTestId('last-scan-info');
    expect(lastScan.textContent).toContain('1,247');
    expect(lastScan.textContent).toContain('3 分 12 秒');
  });

  it('renders scan trigger button', () => {
    renderWithProviders();
    const btn = screen.getByTestId('scan-trigger-button');
    expect(btn).toBeInTheDocument();
    expect(btn.textContent).toContain('掃描媒體庫');
  });

  it('calls triggerScan on button click', async () => {
    mockTriggerScan.mockResolvedValue({});
    renderWithProviders();

    const btn = screen.getByTestId('scan-trigger-button');
    fireEvent.click(btn);

    await waitFor(() => {
      expect(mockTriggerScan).toHaveBeenCalledTimes(1);
    });
  });

  it('shows warning notification when scan already running', async () => {
    mockTriggerScan.mockRejectedValue({
      code: 'SCANNER_ALREADY_RUNNING',
      message: '掃描已在進行中',
    });
    renderWithProviders();

    const btn = screen.getByTestId('scan-trigger-button');
    fireEvent.click(btn);

    await waitFor(() => {
      expect(screen.getByTestId('scanner-notification')).toBeInTheDocument();
      expect(screen.getByText('掃描已在進行中')).toBeInTheDocument();
    });
  });

  it('calls updateSchedule on schedule change', async () => {
    mockUpdateSchedule.mockResolvedValue({ frequency: 'daily' });
    renderWithProviders();

    const select = screen.getByTestId('schedule-select');
    fireEvent.change(select, { target: { value: 'daily' } });

    await waitFor(() => {
      expect(mockUpdateSchedule).toHaveBeenCalledWith('daily');
    });
  });

  it('shows scanning state on button when scanning', async () => {
    const { useScanStatus } = await import('../../hooks/useScanner');
    (useScanStatus as ReturnType<typeof vi.fn>).mockReturnValue({
      data: {
        isScanning: true,
        filesFound: 500,
        filesProcessed: 200,
        currentFile: 'test.mkv',
        percentDone: 40,
        errorCount: 0,
        estimatedTime: '2 分',
        lastScanAt: '',
        lastScanFiles: 0,
        lastScanDuration: '',
      },
      isLoading: false,
    });

    renderWithProviders();
    const btn = screen.getByTestId('scan-trigger-button');
    expect(btn.textContent).toContain('掃描進行中...');
    expect(btn).toBeDisabled();
  });

  it('[P0] renders empty state when no media libraries configured (bugfix-7)', async () => {
    const { useMediaLibraries } = await import('../../hooks/useMediaLibrary');
    vi.mocked(useMediaLibraries).mockReturnValue({
      data: { libraries: [] },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useMediaLibraries>);

    renderWithProviders();
    expect(screen.getByTestId('media-library-manager')).toBeInTheDocument();
    expect(screen.getByText('尚未設定任何媒體庫。請新增媒體庫以開始掃描。')).toBeInTheDocument();
  });
});
