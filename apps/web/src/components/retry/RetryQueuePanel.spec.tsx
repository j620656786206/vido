/**
 * RetryQueuePanel Component Tests (Story 3.11 - Task 8.1)
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { RetryQueuePanel } from './RetryQueuePanel';
import { retryService } from '../../services/retry';

// Mock the retry service
vi.mock('../../services/retry', () => ({
  retryService: {
    getPending: vi.fn(),
    triggerImmediate: vi.fn(),
    cancel: vi.fn(),
  },
}));

const mockRetryService = vi.mocked(retryService);

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
}

function renderWithQueryClient(ui: React.ReactElement) {
  const queryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      {ui}
    </QueryClientProvider>
  );
}

const mockRetryItems = [
  {
    id: 'retry-1',
    taskId: 'parse-task-123',
    taskType: 'parse' as const,
    attemptCount: 1,
    maxAttempts: 4,
    lastError: 'TMDb timeout',
    nextAttemptAt: new Date(Date.now() + 5000).toISOString(),
    timeUntilRetry: '5s',
  },
  {
    id: 'retry-2',
    taskId: 'metadata-task-456',
    taskType: 'metadata_fetch' as const,
    attemptCount: 2,
    maxAttempts: 4,
    lastError: 'Rate limited',
    nextAttemptAt: new Date(Date.now() + 30000).toISOString(),
    timeUntilRetry: '30s',
  },
];

describe('RetryQueuePanel', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it('shows loading state initially', () => {
    mockRetryService.getPending.mockImplementation(() => new Promise(() => {}));

    renderWithQueryClient(<RetryQueuePanel />);

    expect(screen.getByTestId('retry-queue-loading')).toBeInTheDocument();
    expect(screen.getByText('載入中...')).toBeInTheDocument();
  });

  it('shows error state when fetch fails', async () => {
    mockRetryService.getPending.mockRejectedValue(new Error('Network error'));

    renderWithQueryClient(<RetryQueuePanel />);

    await waitFor(() => {
      expect(screen.getByTestId('retry-queue-error')).toBeInTheDocument();
    });

    expect(screen.getByText('載入失敗')).toBeInTheDocument();
    expect(screen.getByText('Network error')).toBeInTheDocument();
  });

  it('shows empty state when no pending retries', async () => {
    mockRetryService.getPending.mockResolvedValue({
      items: [],
      stats: { totalPending: 0, totalSucceeded: 0, totalFailed: 0 },
    });

    renderWithQueryClient(<RetryQueuePanel />);

    await waitFor(() => {
      expect(screen.getByTestId('retry-queue-empty')).toBeInTheDocument();
    });

    expect(screen.getByText('目前沒有待重試項目')).toBeInTheDocument();
  });

  it('renders retry items with countdown timers', async () => {
    mockRetryService.getPending.mockResolvedValue({
      items: mockRetryItems,
      stats: { totalPending: 2, totalSucceeded: 0, totalFailed: 0 },
    });

    renderWithQueryClient(<RetryQueuePanel />);

    await waitFor(() => {
      expect(screen.getByTestId('retry-queue-panel')).toBeInTheDocument();
    });

    // Check stats
    expect(screen.getByTestId('retry-stats')).toHaveTextContent('2 個待重試');

    // Check retry items
    expect(screen.getByTestId('retry-item-retry-1')).toBeInTheDocument();
    expect(screen.getByTestId('retry-item-retry-2')).toBeInTheDocument();

    // Check task types
    expect(screen.getByText('解析')).toBeInTheDocument();
    expect(screen.getByText('取得元資料')).toBeInTheDocument();

    // Check error messages
    expect(screen.getByText('TMDb timeout')).toBeInTheDocument();
    expect(screen.getByText('Rate limited')).toBeInTheDocument();

    // Check attempt counts
    expect(screen.getByText('1/4 次')).toBeInTheDocument();
    expect(screen.getByText('2/4 次')).toBeInTheDocument();
  });

  it('triggers immediate retry when button is clicked', async () => {
    mockRetryService.getPending.mockResolvedValue({
      items: [mockRetryItems[0]],
      stats: { totalPending: 1, totalSucceeded: 0, totalFailed: 0 },
    });
    mockRetryService.triggerImmediate.mockResolvedValue({
      id: 'retry-1',
      message: '已觸發立即重試',
    });

    renderWithQueryClient(<RetryQueuePanel />);

    await waitFor(() => {
      expect(screen.getByTestId('retry-item-retry-1')).toBeInTheDocument();
    });

    const triggerButton = screen.getByTestId('trigger-retry-retry-1');
    fireEvent.click(triggerButton);

    await waitFor(() => {
      expect(mockRetryService.triggerImmediate).toHaveBeenCalledWith('retry-1');
    });
  });

  it('cancels retry when button is clicked', async () => {
    mockRetryService.getPending.mockResolvedValue({
      items: [mockRetryItems[0]],
      stats: { totalPending: 1, totalSucceeded: 0, totalFailed: 0 },
    });
    mockRetryService.cancel.mockResolvedValue(undefined);

    renderWithQueryClient(<RetryQueuePanel />);

    await waitFor(() => {
      expect(screen.getByTestId('retry-item-retry-1')).toBeInTheDocument();
    });

    const cancelButton = screen.getByTestId('cancel-retry-retry-1');
    fireEvent.click(cancelButton);

    await waitFor(() => {
      expect(mockRetryService.cancel).toHaveBeenCalledWith('retry-1');
    });
  });

  it('shows header with title and icon', async () => {
    mockRetryService.getPending.mockResolvedValue({
      items: [],
      stats: { totalPending: 0, totalSucceeded: 0, totalFailed: 0 },
    });

    renderWithQueryClient(<RetryQueuePanel />);

    await waitFor(() => {
      expect(screen.getByText('重試隊列')).toBeInTheDocument();
    });
  });

  it('applies custom className', async () => {
    mockRetryService.getPending.mockResolvedValue({
      items: [],
      stats: { totalPending: 0, totalSucceeded: 0, totalFailed: 0 },
    });

    renderWithQueryClient(<RetryQueuePanel className="custom-class" />);

    await waitFor(() => {
      expect(screen.getByTestId('retry-queue-panel')).toHaveClass('custom-class');
    });
  });

  it('allows refetch on error', async () => {
    mockRetryService.getPending.mockRejectedValueOnce(new Error('Network error'));

    renderWithQueryClient(<RetryQueuePanel />);

    await waitFor(() => {
      expect(screen.getByTestId('retry-queue-error')).toBeInTheDocument();
    });

    // Mock successful response for retry
    mockRetryService.getPending.mockResolvedValueOnce({
      items: [],
      stats: { totalPending: 0, totalSucceeded: 0, totalFailed: 0 },
    });

    const retryButton = screen.getByText('重試');
    fireEvent.click(retryButton);

    await waitFor(() => {
      expect(mockRetryService.getPending).toHaveBeenCalledTimes(2);
    });
  });
});
