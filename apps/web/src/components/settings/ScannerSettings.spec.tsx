import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ScannerSettings } from './ScannerSettings';
import { useScanStatus } from '../../hooks/useScanner';
import type { LastScan, ScanStatus } from '../../services/scannerService';

// Local wall-clock 14:30 whatever the runner's time zone — the page shows
// local time, and this keeps the E1-D string assertable everywhere.
const LAST_SCAN: LastScan = {
  completedAt: new Date(2026, 2, 22, 14, 30).toISOString(),
  filesFound: 1247,
  durationMs: 192_000,
};

// The real /scanner/status shape after fetchApi's snake→camel transform.
function statusWith(lastScan: LastScan | null): ScanStatus {
  return {
    isActive: false,
    filesFound: 0,
    filesCreated: 0,
    filesUpdated: 0,
    filesSkipped: 0,
    filesRemoved: 0,
    errorCount: 0,
    currentFile: '',
    percentDone: 0,
    lastScan,
  };
}

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

// bugfix-scan-instant-completion-no-feedback: controls when "the progress
// stream is connected" so the order (connect, THEN POST) is observable.
const mockRequestScanTracking = vi.fn(() => Promise.resolve());
vi.mock('../../hooks/useScanProgress', () => ({
  requestScanTracking: () => mockRequestScanTracking(),
}));

vi.mock('../../hooks/useScanner', () => ({
  useScanStatus: vi.fn(() => ({ data: statusWith(LAST_SCAN), isLoading: false })),
  useTriggerScan: vi.fn(() => ({
    mutateAsync: mockTriggerScan,
    isPending: false,
  })),
  useScanSchedule: vi.fn(() => ({
    data: { interval: 'hourly' },
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
    mockRequestScanTracking.mockClear();
    mockUpdateSchedule.mockReset();
    vi.mocked(useScanStatus).mockReturnValue({
      data: statusWith(LAST_SCAN),
      isLoading: false,
    } as unknown as ReturnType<typeof useScanStatus>);
  });

  it('renders scanner settings section', () => {
    renderWithProviders();
    // Title lives at the route level now (one header contract).
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

  // bugfix-last-scan-never-shown AC #3: E1-D「2026-03-22 14:30 · 1,247 檔案 · 耗時 3 分 12 秒」.
  it('displays last scan info in the E1-D format', () => {
    renderWithProviders();
    expect(screen.getByTestId('last-scan-info')).toHaveTextContent(
      '2026-03-22 14:30 · 1,247 檔案 · 耗時 3 分 12 秒'
    );
  });

  it.each([
    [400, '不到 1 秒'],
    [45_000, '45 秒'],
    [120_000, '2 分'],
    [192_000, '3 分 12 秒'],
    [3_725_000, '1 小時 2 分'],
  ])('duration %i ms reads 「%s」', async (durationMs, text) => {
    const { useScanStatus } = await import('../../hooks/useScanner');
    vi.mocked(useScanStatus).mockReturnValue({
      data: statusWith({ ...LAST_SCAN, durationMs }),
      isLoading: false,
    } as unknown as ReturnType<typeof useScanStatus>);
    renderWithProviders();
    expect(screen.getByTestId('last-scan-info')).toHaveTextContent(`耗時 ${text}`);
  });

  it('never scanned → 尚未執行過掃描', async () => {
    const { useScanStatus } = await import('../../hooks/useScanner');
    vi.mocked(useScanStatus).mockReturnValue({
      data: statusWith(null),
      isLoading: false,
    } as unknown as ReturnType<typeof useScanStatus>);
    renderWithProviders();
    expect(screen.getByTestId('last-scan-info')).toHaveTextContent('尚未執行過掃描');
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

  it('opens the progress stream and waits for it BEFORE starting the scan', async () => {
    let connect!: () => void;
    mockRequestScanTracking.mockImplementationOnce(
      () => new Promise<void>((resolve) => (connect = resolve))
    );
    mockTriggerScan.mockResolvedValue({});
    renderWithProviders();

    fireEvent.click(screen.getByTestId('scan-trigger-button'));
    await waitFor(() => expect(mockRequestScanTracking).toHaveBeenCalledTimes(1));
    expect(mockTriggerScan).not.toHaveBeenCalled();
    expect(screen.getByTestId('scan-trigger-button')).toBeDisabled(); // no double start

    connect();
    await waitFor(() => expect(mockTriggerScan).toHaveBeenCalledTimes(1));
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

  // disc-2026-09-scan-trigger-error-english: no English from the server or the
  // network reaches the notification.
  it('any other failure reads in zh-TW', async () => {
    mockTriggerScan.mockRejectedValue({ code: 'INTERNAL_ERROR', message: 'Failed to fetch' });
    renderWithProviders();
    fireEvent.click(screen.getByTestId('scan-trigger-button'));
    const note = await screen.findByTestId('scanner-notification');
    expect(note).toHaveTextContent('掃描沒有開始，請再試一次。');
    expect(note).not.toHaveTextContent('Failed');
  });

  it('a zh-TW server reason is shown as is', async () => {
    mockTriggerScan.mockRejectedValue({ code: 'SCANNER_BUSY', message: '掃描伺服器忙線中' });
    renderWithProviders();
    fireEvent.click(screen.getByTestId('scan-trigger-button'));
    expect(await screen.findByTestId('scanner-notification')).toHaveTextContent('掃描伺服器忙線中');
  });

  it('calls updateSchedule on schedule change', async () => {
    mockUpdateSchedule.mockResolvedValue({ interval: 'daily' });
    renderWithProviders();

    const select = screen.getByTestId('schedule-select');
    fireEvent.change(select, { target: { value: 'daily' } });

    await waitFor(() => {
      expect(mockUpdateSchedule).toHaveBeenCalledWith('daily');
    });
  });

  // bugfix-scan-schedule-field-mismatch
  it('shows the saved schedule, read from the API interval field', () => {
    renderWithProviders();
    expect(screen.getByTestId('schedule-select')).toHaveValue('hourly');
  });

  it('a failed save says so in Chinese, not with the backend English message', async () => {
    mockUpdateSchedule.mockRejectedValueOnce(
      Object.assign(new Error("Request body must contain an 'interval' field"), {
        code: 'SCANNER_SCHEDULE_INVALID',
      })
    );
    renderWithProviders();
    fireEvent.change(screen.getByTestId('schedule-select'), { target: { value: 'daily' } });
    await waitFor(() => {
      expect(screen.getByTestId('scanner-notification')).toHaveTextContent(
        '排程沒有存成功，請再試一次。'
      );
    });
    expect(screen.queryByText(/interval/)).not.toBeInTheDocument();
  });

  it('shows scanning state on button when scanning', async () => {
    const { useScanStatus } = await import('../../hooks/useScanner');
    // The backend's live flag is `is_active` (never `isScanning`).
    (useScanStatus as ReturnType<typeof vi.fn>).mockReturnValue({
      data: { ...statusWith(null), isActive: true, filesFound: 500, percentDone: 40 },
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
