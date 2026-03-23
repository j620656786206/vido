import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ScannerSettings } from './ScannerSettings';

const mockTriggerScan = vi.fn();
const mockUpdateSchedule = vi.fn();

vi.mock('../../hooks/useScanner', () => ({
  useScanStatus: vi.fn(() => ({
    data: {
      is_scanning: false,
      files_found: 0,
      files_processed: 0,
      current_file: '',
      percent_done: 0,
      error_count: 0,
      estimated_time: '',
      last_scan_at: '2026-03-22T14:30:00Z',
      last_scan_files: 1247,
      last_scan_duration: '3 分 12 秒',
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

  it('displays media folder info', () => {
    renderWithProviders();
    expect(screen.getByTestId('folder-list')).toBeInTheDocument();
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
    mockTriggerScan.mockRejectedValue({ code: 'SCANNER_ALREADY_RUNNING', message: '掃描已在進行中' });
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
        is_scanning: true,
        files_found: 500,
        files_processed: 200,
        current_file: 'test.mkv',
        percent_done: 40,
        error_count: 0,
        estimated_time: '2 分',
        last_scan_at: '',
        last_scan_files: 0,
        last_scan_duration: '',
      },
      isLoading: false,
    });

    renderWithProviders();
    const btn = screen.getByTestId('scan-trigger-button');
    expect(btn.textContent).toContain('掃描進行中...');
    expect(btn).toBeDisabled();
  });
});
