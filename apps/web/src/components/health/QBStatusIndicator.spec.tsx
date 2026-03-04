import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QBStatusIndicator } from './QBStatusIndicator';

// Mock the hook
vi.mock('../../hooks/useConnectionHealth', () => ({
  useQBConnectionHealth: vi.fn(),
}));

import { useQBConnectionHealth } from '../../hooks/useConnectionHealth';

const mockUseQBConnectionHealth = vi.mocked(useQBConnectionHealth);

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

describe('QBStatusIndicator', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows loading state', () => {
    mockUseQBConnectionHealth.mockReturnValue({
      data: undefined,
      isLoading: true,
    } as ReturnType<typeof useQBConnectionHealth>);

    renderWithQuery(<QBStatusIndicator />);
    expect(screen.getByText('連線中...')).toBeInTheDocument();
  });

  it('shows healthy status', () => {
    mockUseQBConnectionHealth.mockReturnValue({
      data: {
        name: 'qbittorrent',
        displayName: 'qBittorrent',
        status: 'healthy',
        lastCheck: new Date().toISOString(),
        lastSuccess: new Date().toISOString(),
        errorCount: 0,
      },
      isLoading: false,
    } as ReturnType<typeof useQBConnectionHealth>);

    renderWithQuery(<QBStatusIndicator />);
    expect(screen.getByRole('button', { name: 'qBittorrent 已連線' })).toBeInTheDocument();
  });

  it('shows degraded status', () => {
    mockUseQBConnectionHealth.mockReturnValue({
      data: {
        name: 'qbittorrent',
        displayName: 'qBittorrent',
        status: 'degraded',
        lastCheck: new Date().toISOString(),
        lastSuccess: new Date().toISOString(),
        errorCount: 1,
        message: 'timeout',
      },
      isLoading: false,
    } as ReturnType<typeof useQBConnectionHealth>);

    renderWithQuery(<QBStatusIndicator />);
    expect(screen.getByRole('button', { name: 'qBittorrent 連線不穩定' })).toBeInTheDocument();
  });

  it('shows disconnected status with last success time', () => {
    const fiveMinutesAgo = new Date(Date.now() - 5 * 60 * 1000).toISOString();
    mockUseQBConnectionHealth.mockReturnValue({
      data: {
        name: 'qbittorrent',
        displayName: 'qBittorrent',
        status: 'down',
        lastCheck: new Date().toISOString(),
        lastSuccess: fiveMinutesAgo,
        errorCount: 3,
        message: 'connection refused',
      },
      isLoading: false,
    } as ReturnType<typeof useQBConnectionHealth>);

    renderWithQuery(<QBStatusIndicator />);
    const button = screen.getByRole('button');
    expect(button).toHaveAttribute('aria-label', 'qBittorrent 未連線');
  });

  it('calls onClick when clicked', async () => {
    const user = userEvent.setup();
    const handleClick = vi.fn();

    mockUseQBConnectionHealth.mockReturnValue({
      data: {
        name: 'qbittorrent',
        displayName: 'qBittorrent',
        status: 'healthy',
        lastCheck: new Date().toISOString(),
        lastSuccess: new Date().toISOString(),
        errorCount: 0,
      },
      isLoading: false,
    } as ReturnType<typeof useQBConnectionHealth>);

    renderWithQuery(<QBStatusIndicator onClick={handleClick} />);
    await user.click(screen.getByRole('button'));
    expect(handleClick).toHaveBeenCalledOnce();
  });

  it('shows down status when data is undefined', () => {
    mockUseQBConnectionHealth.mockReturnValue({
      data: undefined,
      isLoading: false,
    } as ReturnType<typeof useQBConnectionHealth>);

    renderWithQuery(<QBStatusIndicator />);
    expect(screen.getByRole('button', { name: 'qBittorrent 未連線' })).toBeInTheDocument();
  });

  it('[P2] shows "上次" text when status is down with valid lastSuccess', () => {
    const tenMinutesAgo = new Date(Date.now() - 10 * 60 * 1000).toISOString();
    mockUseQBConnectionHealth.mockReturnValue({
      data: {
        name: 'qbittorrent',
        displayName: 'qBittorrent',
        status: 'down',
        lastCheck: new Date().toISOString(),
        lastSuccess: tenMinutesAgo,
        errorCount: 3,
        message: 'connection refused',
      },
      isLoading: false,
    } as ReturnType<typeof useQBConnectionHealth>);

    renderWithQuery(<QBStatusIndicator />);
    expect(screen.getByText(/上次：10 分鐘前/)).toBeInTheDocument();
  });

  it('[P2] does not show "上次" text when status is healthy', () => {
    mockUseQBConnectionHealth.mockReturnValue({
      data: {
        name: 'qbittorrent',
        displayName: 'qBittorrent',
        status: 'healthy',
        lastCheck: new Date().toISOString(),
        lastSuccess: new Date().toISOString(),
        errorCount: 0,
      },
      isLoading: false,
    } as ReturnType<typeof useQBConnectionHealth>);

    renderWithQuery(<QBStatusIndicator />);
    expect(screen.queryByText(/上次：/)).not.toBeInTheDocument();
  });

  it('[P2] includes last success in title attribute when down', () => {
    const threeMinutesAgo = new Date(Date.now() - 3 * 60 * 1000).toISOString();
    mockUseQBConnectionHealth.mockReturnValue({
      data: {
        name: 'qbittorrent',
        displayName: 'qBittorrent',
        status: 'down',
        lastCheck: new Date().toISOString(),
        lastSuccess: threeMinutesAgo,
        errorCount: 5,
      },
      isLoading: false,
    } as ReturnType<typeof useQBConnectionHealth>);

    renderWithQuery(<QBStatusIndicator />);
    const button = screen.getByRole('button');
    expect(button.title).toContain('上次連線：3 分鐘前');
  });

  it('[P2] shows "剛剛" for very recent lastSuccess when down', () => {
    const justNow = new Date(Date.now() - 10 * 1000).toISOString();
    mockUseQBConnectionHealth.mockReturnValue({
      data: {
        name: 'qbittorrent',
        displayName: 'qBittorrent',
        status: 'down',
        lastCheck: new Date().toISOString(),
        lastSuccess: justNow,
        errorCount: 3,
      },
      isLoading: false,
    } as ReturnType<typeof useQBConnectionHealth>);

    renderWithQuery(<QBStatusIndicator />);
    expect(screen.getByText(/上次：剛剛/)).toBeInTheDocument();
  });
});
