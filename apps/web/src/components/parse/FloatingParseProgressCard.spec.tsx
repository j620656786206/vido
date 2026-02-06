/**
 * FloatingParseProgressCard Tests (Story 3.10 - Task 4)
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FloatingParseProgressCard } from './FloatingParseProgressCard';

// Mock useParseProgress hook
const mockProgress = {
  taskId: 'task-123',
  filename: 'test-movie.mkv',
  status: 'pending' as const,
  steps: [
    { name: 'filename_extract', label: '解析檔名', status: 'success' as const },
    { name: 'tmdb_search', label: '搜尋 TMDb', status: 'in_progress' as const },
    { name: 'douban_search', label: '搜尋豆瓣', status: 'pending' as const },
    { name: 'wikipedia_search', label: '搜尋 Wikipedia', status: 'pending' as const },
    { name: 'ai_retry', label: 'AI 重試', status: 'pending' as const },
    { name: 'download_poster', label: '下載海報', status: 'pending' as const },
  ],
  currentStep: 1,
  percentage: 16,
  startedAt: new Date().toISOString(),
};

let mockUseParseProgress = vi.fn();

vi.mock('./useParseProgress', () => ({
  useParseProgress: (...args: unknown[]) => mockUseParseProgress(...args),
}));

describe('FloatingParseProgressCard', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mockUseParseProgress = vi.fn().mockReturnValue({
      progress: mockProgress,
      status: 'pending',
      error: null,
      isConnected: true,
      isReconnecting: false,
      disconnect: vi.fn(),
      reconnect: vi.fn(),
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders the floating card', () => {
    render(<FloatingParseProgressCard taskId="task-123" onClose={vi.fn()} />);

    expect(screen.getByTestId('floating-parse-progress-card')).toBeInTheDocument();
  });

  it('displays "正在解析..." when parsing', () => {
    render(<FloatingParseProgressCard taskId="task-123" onClose={vi.fn()} />);

    expect(screen.getByText('正在解析...')).toBeInTheDocument();
  });

  it('shows progress percentage', () => {
    render(<FloatingParseProgressCard taskId="task-123" onClose={vi.fn()} />);

    expect(screen.getByText('16%')).toBeInTheDocument();
  });

  it('displays filename', () => {
    render(<FloatingParseProgressCard taskId="task-123" onClose={vi.fn()} />);

    expect(screen.getByText(/test-movie.mkv/)).toBeInTheDocument();
  });

  it('shows progress bar with correct value', () => {
    render(<FloatingParseProgressCard taskId="task-123" onClose={vi.fn()} />);

    const progressBar = screen.getByRole('progressbar');
    expect(progressBar).toHaveAttribute('aria-valuenow', '16');
    expect(progressBar).toHaveAttribute('aria-valuemin', '0');
    expect(progressBar).toHaveAttribute('aria-valuemax', '100');
  });

  it('renders layered progress indicator', () => {
    render(<FloatingParseProgressCard taskId="task-123" onClose={vi.fn()} />);

    expect(screen.getByTestId('layered-progress-indicator')).toBeInTheDocument();
    expect(screen.getByText('解析檔名')).toBeInTheDocument();
    expect(screen.getByText('搜尋 TMDb')).toBeInTheDocument();
  });

  it('calls onClose when close button clicked', async () => {
    vi.useRealTimers();
    const onClose = vi.fn();
    const user = userEvent.setup();

    render(<FloatingParseProgressCard taskId="task-123" onClose={onClose} />);

    await user.click(screen.getByTestId('close-button'));
    expect(onClose).toHaveBeenCalledTimes(1);
    vi.useFakeTimers();
  });

  it('minimizes when minimize button clicked', async () => {
    vi.useRealTimers();
    const user = userEvent.setup();

    render(<FloatingParseProgressCard taskId="task-123" onClose={vi.fn()} />);

    // Initially expanded - should show layered progress
    expect(screen.getByTestId('layered-progress-indicator')).toBeInTheDocument();

    await user.click(screen.getByTestId('minimize-button'));

    // After minimize - should not show layered progress
    expect(screen.queryByTestId('layered-progress-indicator')).not.toBeInTheDocument();
    vi.useFakeTimers();
  });

  it('displays success state when parse completes', () => {
    mockUseParseProgress.mockReturnValue({
      progress: {
        ...mockProgress,
        status: 'success',
        percentage: 100,
        result: {
          mediaId: 'movie-456',
          title: 'Test Movie',
          year: 2024,
          metadataSource: 'tmdb',
        },
      },
      status: 'success',
      error: null,
      isConnected: true,
      isReconnecting: false,
      disconnect: vi.fn(),
      reconnect: vi.fn(),
    });

    render(<FloatingParseProgressCard taskId="task-123" onClose={vi.fn()} />);

    expect(screen.getByText('✅ 解析完成！')).toBeInTheDocument();
    expect(screen.getByText(/Test Movie/)).toBeInTheDocument();
    expect(screen.getByText(/2024/)).toBeInTheDocument();
    expect(screen.getByText('來源：TMDb')).toBeInTheDocument();
  });

  it('auto-dismisses after success', () => {
    const onClose = vi.fn();
    mockUseParseProgress.mockReturnValue({
      progress: { ...mockProgress, status: 'success', percentage: 100 },
      status: 'success',
      error: null,
      isConnected: true,
      isReconnecting: false,
      disconnect: vi.fn(),
      reconnect: vi.fn(),
    });

    render(
      <FloatingParseProgressCard taskId="task-123" onClose={onClose} autoDismissDelay={3000} />
    );

    expect(onClose).not.toHaveBeenCalled();

    act(() => {
      vi.advanceTimersByTime(3000);
    });

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('does not auto-dismiss when autoDismissDelay is 0', () => {
    const onClose = vi.fn();
    mockUseParseProgress.mockReturnValue({
      progress: { ...mockProgress, status: 'success', percentage: 100 },
      status: 'success',
      error: null,
      isConnected: true,
      isReconnecting: false,
      disconnect: vi.fn(),
      reconnect: vi.fn(),
    });

    render(<FloatingParseProgressCard taskId="task-123" onClose={onClose} autoDismissDelay={0} />);

    act(() => {
      vi.advanceTimersByTime(5000);
    });

    expect(onClose).not.toHaveBeenCalled();
  });

  it('displays failure state with error details', () => {
    mockUseParseProgress.mockReturnValue({
      progress: {
        ...mockProgress,
        status: 'failed',
        steps: [
          { name: 'filename_extract', label: '解析檔名', status: 'success' as const },
          {
            name: 'tmdb_search',
            label: '搜尋 TMDb',
            status: 'failed' as const,
            error: 'API timeout',
          },
          { name: 'douban_search', label: '搜尋豆瓣', status: 'failed' as const },
          { name: 'wikipedia_search', label: '搜尋 Wikipedia', status: 'pending' as const },
          { name: 'ai_retry', label: 'AI 重試', status: 'pending' as const },
          { name: 'download_poster', label: '下載海報', status: 'pending' as const },
        ],
      },
      status: 'failed',
      error: null,
      isConnected: true,
      isReconnecting: false,
      disconnect: vi.fn(),
      reconnect: vi.fn(),
    });

    render(<FloatingParseProgressCard taskId="task-123" onClose={vi.fn()} />);

    expect(screen.getByText('❌ 解析失敗')).toBeInTheDocument();
    expect(screen.getByTestId('error-details-panel')).toBeInTheDocument();
    expect(screen.getByText('手動搜尋')).toBeInTheDocument();
  });

  it('calls action callbacks on failure', async () => {
    vi.useRealTimers();
    const onManualSearch = vi.fn();
    const onEditFilename = vi.fn();
    const onSkip = vi.fn();
    const user = userEvent.setup();

    mockUseParseProgress.mockReturnValue({
      progress: {
        ...mockProgress,
        status: 'failed',
        steps: [{ name: 'tmdb_search', label: '搜尋 TMDb', status: 'failed' as const }],
      },
      status: 'failed',
      error: null,
      isConnected: true,
      isReconnecting: false,
      disconnect: vi.fn(),
      reconnect: vi.fn(),
    });

    render(
      <FloatingParseProgressCard
        taskId="task-123"
        onClose={vi.fn()}
        onManualSearch={onManualSearch}
        onEditFilename={onEditFilename}
        onSkip={onSkip}
      />
    );

    await user.click(screen.getByTestId('manual-search-button'));
    expect(onManualSearch).toHaveBeenCalledTimes(1);

    await user.click(screen.getByTestId('edit-filename-button'));
    expect(onEditFilename).toHaveBeenCalledTimes(1);

    await user.click(screen.getByTestId('skip-button'));
    expect(onSkip).toHaveBeenCalledTimes(1);
    vi.useFakeTimers();
  });

  it('shows connection error message', () => {
    mockUseParseProgress.mockReturnValue({
      progress: mockProgress,
      status: 'pending',
      error: new Error('Connection failed'),
      isConnected: false,
      isReconnecting: true,
      disconnect: vi.fn(),
      reconnect: vi.fn(),
    });

    render(<FloatingParseProgressCard taskId="task-123" onClose={vi.fn()} />);

    expect(screen.getByText(/連線中斷/)).toBeInTheDocument();
  });

  it('has correct ARIA attributes', () => {
    render(<FloatingParseProgressCard taskId="task-123" onClose={vi.fn()} />);

    const card = screen.getByTestId('floating-parse-progress-card');
    expect(card).toHaveAttribute('role', 'status');
    expect(card).toHaveAttribute('aria-live', 'polite');
  });

  it('close button has accessible label', () => {
    render(<FloatingParseProgressCard taskId="task-123" onClose={vi.fn()} />);

    expect(screen.getByTestId('close-button')).toHaveAttribute('aria-label', '關閉進度卡片');
  });
});
