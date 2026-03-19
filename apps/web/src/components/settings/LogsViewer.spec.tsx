import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { LogsViewer } from './LogsViewer';

// Mock the logService
vi.mock('../../services/logService', () => ({
  logService: {
    getLogs: vi.fn(),
    clearLogs: vi.fn(),
  },
}));

import { logService } from '../../services/logService';

const mockLogs = {
  logs: [
    {
      id: 1,
      level: 'ERROR' as const,
      message: 'Failed to fetch metadata from TMDb',
      source: 'tmdb',
      context: { error_code: 'TMDB_TIMEOUT', movie_id: '123' },
      hint: '檢查網路連線，或稍後重試。TMDb API 可能暫時不可用。',
      createdAt: '2026-03-18T10:00:00Z',
    },
    {
      id: 2,
      level: 'WARN' as const,
      message: 'Cache miss for movie poster',
      source: 'cache',
      createdAt: '2026-03-18T09:55:00Z',
    },
    {
      id: 3,
      level: 'INFO' as const,
      message: 'Server started successfully',
      createdAt: '2026-03-18T09:50:00Z',
    },
  ],
  total: 3,
  page: 1,
  perPage: 50,
};

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

describe('LogsViewer', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows loading state initially', () => {
    vi.mocked(logService.getLogs).mockReturnValue(new Promise(() => {}));
    render(<LogsViewer />, { wrapper: createWrapper() });
    expect(screen.getByTestId('logs-loading')).toBeInTheDocument();
  });

  it('renders log entries', async () => {
    vi.mocked(logService.getLogs).mockResolvedValue(mockLogs);

    render(<LogsViewer />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByTestId('logs-viewer')).toBeInTheDocument();
    });

    const entries = screen.getAllByTestId('log-entry');
    expect(entries).toHaveLength(3);

    // Check header
    expect(screen.getByText('系統日誌')).toBeInTheDocument();
    expect(screen.getByText(/共 3 筆記錄/)).toBeInTheDocument();
  });

  it('shows error state', async () => {
    vi.mocked(logService.getLogs).mockRejectedValue(new Error('Network error'));

    render(<LogsViewer />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByTestId('logs-error')).toBeInTheDocument();
    });
    expect(screen.getByText('無法載入系統日誌')).toBeInTheDocument();
  });

  it('shows empty state when no logs', async () => {
    vi.mocked(logService.getLogs).mockResolvedValue({
      logs: [],
      total: 0,
      page: 1,
      perPage: 50,
    });

    render(<LogsViewer />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByTestId('logs-empty')).toBeInTheDocument();
    });
    expect(screen.getByText('沒有符合條件的日誌記錄')).toBeInTheDocument();
  });

  it('filters by level when chip clicked', async () => {
    const user = userEvent.setup();
    vi.mocked(logService.getLogs).mockResolvedValue(mockLogs);

    render(<LogsViewer />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByTestId('logs-viewer')).toBeInTheDocument();
    });

    // Click ERROR filter
    await user.click(screen.getByTestId('log-filter-error'));

    await waitFor(() => {
      expect(logService.getLogs).toHaveBeenCalledWith(expect.objectContaining({ level: 'ERROR' }));
    });
  });

  it('searches by keyword on Enter', async () => {
    const user = userEvent.setup();
    vi.mocked(logService.getLogs).mockResolvedValue(mockLogs);

    render(<LogsViewer />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByTestId('logs-viewer')).toBeInTheDocument();
    });

    const input = screen.getByTestId('log-keyword-input');
    await user.type(input, 'tmdb{Enter}');

    await waitFor(() => {
      expect(logService.getLogs).toHaveBeenCalledWith(expect.objectContaining({ keyword: 'tmdb' }));
    });
  });

  it('clears old logs', async () => {
    const user = userEvent.setup();
    vi.mocked(logService.getLogs).mockResolvedValue(mockLogs);
    vi.mocked(logService.clearLogs).mockResolvedValue({
      entriesRemoved: 10,
      days: 30,
    });

    render(<LogsViewer />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByTestId('logs-viewer')).toBeInTheDocument();
    });

    await user.click(screen.getByTestId('clear-old-logs-btn'));

    await waitFor(() => {
      expect(screen.getByTestId('logs-clear-result')).toBeInTheDocument();
    });
    expect(screen.getByText(/已清除 10 筆日誌記錄/)).toBeInTheDocument();
  });
});
