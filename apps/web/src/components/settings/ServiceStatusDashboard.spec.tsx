import React from 'react';
import { render, screen, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ServiceStatusDashboard } from './ServiceStatusDashboard';

vi.mock('../../hooks/useServiceStatus', () => ({
  useServiceStatuses: vi.fn(),
  useTestServiceConnection: vi.fn(),
}));

import { useServiceStatuses, useTestServiceConnection } from '../../hooks/useServiceStatus';

const mockUseServiceStatuses = vi.mocked(useServiceStatuses);
const mockUseTestServiceConnection = vi.mocked(useTestServiceConnection);

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(React.createElement(QueryClientProvider, { client: queryClient }, ui));
}

beforeEach(() => {
  mockUseTestServiceConnection.mockReturnValue({
    mutateAsync: vi.fn(),
    isPending: false,
  } as any);
});

describe('ServiceStatusDashboard', () => {
  it('renders loading state', () => {
    mockUseServiceStatuses.mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    } as any);

    renderWithQuery(React.createElement(ServiceStatusDashboard));
    expect(screen.getByTestId('status-loading')).toBeInTheDocument();
  });

  it('renders error state', () => {
    mockUseServiceStatuses.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('Network error'),
    } as any);

    renderWithQuery(React.createElement(ServiceStatusDashboard));
    expect(screen.getByTestId('status-error')).toBeInTheDocument();
    expect(screen.getByText('無法載入服務狀態')).toBeInTheDocument();
  });

  it('renders service cards when data loads', () => {
    mockUseServiceStatuses.mockReturnValue({
      data: {
        services: [
          {
            name: 'tmdb',
            displayName: 'TMDb API',
            status: 'connected',
            message: '已連線',
            lastSuccessAt: '2026-02-10T14:30:00Z',
            lastCheckAt: '2026-02-10T14:30:00Z',
            responseTimeMs: 45,
          },
          {
            name: 'ai',
            displayName: 'AI 服務',
            status: 'unconfigured',
            message: '未設定',
            lastSuccessAt: null,
            lastCheckAt: '2026-02-10T14:30:00Z',
            responseTimeMs: 0,
          },
        ],
      },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(ServiceStatusDashboard));
    expect(screen.getByTestId('service-status-dashboard')).toBeInTheDocument();
    expect(screen.getByTestId('service-card-tmdb')).toBeInTheDocument();
    expect(screen.getByTestId('service-card-ai')).toBeInTheDocument();
    expect(screen.getByText('TMDb API')).toBeInTheDocument();
    expect(screen.getByText('AI 服務')).toBeInTheDocument();
  });

  it('renders empty state when no services', () => {
    mockUseServiceStatuses.mockReturnValue({
      data: { services: [] },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(ServiceStatusDashboard));
    expect(screen.getByTestId('status-empty')).toBeInTheDocument();
  });

  it('shows response time for connected services', () => {
    mockUseServiceStatuses.mockReturnValue({
      data: {
        services: [
          {
            name: 'tmdb',
            displayName: 'TMDb API',
            status: 'connected',
            message: '已連線',
            lastSuccessAt: '2026-02-10T14:30:00Z',
            lastCheckAt: '2026-02-10T14:30:00Z',
            responseTimeMs: 45,
          },
        ],
      },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(ServiceStatusDashboard));
    expect(screen.getByText('45ms')).toBeInTheDocument();
  });

  it('renders correct status indicators', () => {
    mockUseServiceStatuses.mockReturnValue({
      data: {
        services: [
          {
            name: 'tmdb',
            displayName: 'TMDb API',
            status: 'connected',
            message: '已連線',
            lastSuccessAt: null,
            lastCheckAt: '2026-02-10T14:30:00Z',
            responseTimeMs: 45,
          },
          {
            name: 'qbittorrent',
            displayName: 'qBittorrent',
            status: 'disconnected',
            message: 'connection refused',
            lastSuccessAt: null,
            lastCheckAt: '2026-02-10T14:30:00Z',
            responseTimeMs: 0,
            errorMessage: 'connection refused',
          },
        ],
      },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(ServiceStatusDashboard));
    expect(screen.getByText('已連線')).toBeInTheDocument();
    expect(screen.getByText('已斷線')).toBeInTheDocument();
  });

  it('[P1] renders header text', () => {
    mockUseServiceStatuses.mockReturnValue({
      data: { services: [] },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(ServiceStatusDashboard));
    expect(screen.getByText('服務狀態')).toBeInTheDocument();
    expect(screen.getByText('監控外部服務連線狀態')).toBeInTheDocument();
  });

  it('[P1] calls mutateAsync when test button is clicked on a service card', async () => {
    const user = userEvent.setup();
    const mockMutateAsync = vi.fn().mockResolvedValue({});
    mockUseTestServiceConnection.mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
    } as any);
    mockUseServiceStatuses.mockReturnValue({
      data: {
        services: [
          {
            name: 'tmdb',
            displayName: 'TMDb API',
            status: 'connected',
            message: '已連線',
            lastSuccessAt: '2026-02-10T14:30:00Z',
            lastCheckAt: '2026-02-10T14:30:00Z',
            responseTimeMs: 45,
          },
        ],
      },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(ServiceStatusDashboard));
    await user.click(screen.getByTestId('test-btn-tmdb'));
    expect(mockMutateAsync).toHaveBeenCalledWith('tmdb');
  });

  it('[P2] shows error message text from API error', () => {
    mockUseServiceStatuses.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('Connection timeout'),
    } as any);

    renderWithQuery(React.createElement(ServiceStatusDashboard));
    expect(screen.getByText('Connection timeout')).toBeInTheDocument();
  });

  it('[P1] renders all three service types together', () => {
    mockUseServiceStatuses.mockReturnValue({
      data: {
        services: [
          {
            name: 'qbittorrent',
            displayName: 'qBittorrent',
            status: 'connected',
            message: '已連線',
            lastSuccessAt: '2026-02-10T14:30:00Z',
            lastCheckAt: '2026-02-10T14:30:00Z',
            responseTimeMs: 30,
          },
          {
            name: 'tmdb',
            displayName: 'TMDb API',
            status: 'rate_limited',
            message: '速率限制中',
            lastSuccessAt: '2026-02-10T14:29:00Z',
            lastCheckAt: '2026-02-10T14:30:00Z',
            responseTimeMs: 230,
          },
          {
            name: 'ai',
            displayName: 'AI 服務',
            status: 'error',
            message: 'API key invalid',
            lastSuccessAt: null,
            lastCheckAt: '2026-02-10T14:30:00Z',
            responseTimeMs: 0,
            errorMessage: 'API key invalid',
          },
        ],
      },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(ServiceStatusDashboard));
    expect(screen.getByTestId('service-card-qbittorrent')).toBeInTheDocument();
    expect(screen.getByTestId('service-card-tmdb')).toBeInTheDocument();
    expect(screen.getByTestId('service-card-ai')).toBeInTheDocument();
  });

  it('[P1] shows error message when test connection fails', async () => {
    const user = userEvent.setup();
    const mockMutateAsync = vi.fn().mockRejectedValue(new Error('Service unreachable'));
    mockUseTestServiceConnection.mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
    } as any);
    mockUseServiceStatuses.mockReturnValue({
      data: {
        services: [
          {
            name: 'tmdb',
            displayName: 'TMDb API',
            status: 'connected',
            message: '已連線',
            lastSuccessAt: '2026-02-10T14:30:00Z',
            lastCheckAt: '2026-02-10T14:30:00Z',
            responseTimeMs: 45,
          },
        ],
      },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(ServiceStatusDashboard));
    await user.click(screen.getByTestId('test-btn-tmdb'));
    expect(mockMutateAsync).toHaveBeenCalledWith('tmdb');
    expect(screen.getByTestId('test-error')).toBeInTheDocument();
    expect(screen.getByText('Service unreachable')).toBeInTheDocument();
  });

  it('[P1] clears test error on next successful test', async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const mockMutateAsync = vi.fn().mockImplementation(() => {
      callCount++;
      if (callCount === 1) return Promise.reject(new Error('Failed'));
      return Promise.resolve({});
    });
    mockUseTestServiceConnection.mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
    } as any);
    mockUseServiceStatuses.mockReturnValue({
      data: {
        services: [
          {
            name: 'tmdb',
            displayName: 'TMDb API',
            status: 'connected',
            message: '已連線',
            lastSuccessAt: '2026-02-10T14:30:00Z',
            lastCheckAt: '2026-02-10T14:30:00Z',
            responseTimeMs: 45,
          },
        ],
      },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(ServiceStatusDashboard));
    // First click fails
    await user.click(screen.getByTestId('test-btn-tmdb'));
    expect(screen.getByTestId('test-error')).toBeInTheDocument();

    // Second click succeeds — error should clear
    await user.click(screen.getByTestId('test-btn-tmdb'));
    expect(screen.queryByTestId('test-error')).not.toBeInTheDocument();
  });

  describe('AC3: Status change notifications', () => {
    afterEach(() => {
      vi.restoreAllMocks();
    });

    it('[P1] shows notification when service status changes between renders', () => {
      const initialServices = [
        {
          name: 'tmdb',
          displayName: 'TMDb API',
          status: 'connected' as const,
          message: '已連線',
          lastSuccessAt: '2026-02-10T14:30:00Z',
          lastCheckAt: '2026-02-10T14:30:00Z',
          responseTimeMs: 45,
        },
      ];

      const mockReturn = {
        data: { services: initialServices },
        isLoading: false,
        error: null,
      };
      mockUseServiceStatuses.mockReturnValue(mockReturn as any);

      const { rerender } = renderWithQuery(React.createElement(ServiceStatusDashboard));

      // No notification on first render
      expect(screen.queryByTestId('status-change-notification')).not.toBeInTheDocument();

      // Simulate status change on next polling cycle
      const changedServices = [
        {
          ...initialServices[0],
          status: 'disconnected' as const,
          message: 'connection refused',
          errorMessage: 'connection refused',
        },
      ];
      mockUseServiceStatuses.mockReturnValue({
        data: { services: changedServices },
        isLoading: false,
        error: null,
      } as any);

      rerender(
        React.createElement(
          QueryClientProvider,
          {
            client: new QueryClient({ defaultOptions: { queries: { retry: false } } }),
          },
          React.createElement(ServiceStatusDashboard)
        )
      );

      const notification = screen.getByTestId('status-change-notification');
      expect(notification).toBeInTheDocument();
      expect(notification).toHaveTextContent('TMDb API：已連線 → 已斷線');
    });

    it('[P1] dismisses notification when close button is clicked', async () => {
      const user = userEvent.setup();
      const initialServices = [
        {
          name: 'tmdb',
          displayName: 'TMDb API',
          status: 'connected' as const,
          message: '已連線',
          lastSuccessAt: '2026-02-10T14:30:00Z',
          lastCheckAt: '2026-02-10T14:30:00Z',
          responseTimeMs: 45,
        },
      ];

      mockUseServiceStatuses.mockReturnValue({
        data: { services: initialServices },
        isLoading: false,
        error: null,
      } as any);

      const { rerender } = renderWithQuery(React.createElement(ServiceStatusDashboard));

      // Trigger status change
      mockUseServiceStatuses.mockReturnValue({
        data: {
          services: [{ ...initialServices[0], status: 'error' as const, message: 'timeout' }],
        },
        isLoading: false,
        error: null,
      } as any);

      rerender(
        React.createElement(
          QueryClientProvider,
          {
            client: new QueryClient({ defaultOptions: { queries: { retry: false } } }),
          },
          React.createElement(ServiceStatusDashboard)
        )
      );

      expect(screen.getByTestId('status-change-notification')).toBeInTheDocument();

      await user.click(screen.getByTestId('dismiss-notification'));
      expect(screen.queryByTestId('status-change-notification')).not.toBeInTheDocument();
    });

    it('[P1] shows multiple status changes in one notification', () => {
      const initialServices = [
        {
          name: 'tmdb',
          displayName: 'TMDb API',
          status: 'connected' as const,
          message: '已連線',
          lastSuccessAt: '2026-02-10T14:30:00Z',
          lastCheckAt: '2026-02-10T14:30:00Z',
          responseTimeMs: 45,
        },
        {
          name: 'qbittorrent',
          displayName: 'qBittorrent',
          status: 'disconnected' as const,
          message: 'refused',
          lastSuccessAt: null,
          lastCheckAt: '2026-02-10T14:30:00Z',
          responseTimeMs: 0,
          errorMessage: 'refused',
        },
      ];

      mockUseServiceStatuses.mockReturnValue({
        data: { services: initialServices },
        isLoading: false,
        error: null,
      } as any);

      const { rerender } = renderWithQuery(React.createElement(ServiceStatusDashboard));

      // Both services change
      mockUseServiceStatuses.mockReturnValue({
        data: {
          services: [
            { ...initialServices[0], status: 'rate_limited' as const, message: '速率限制中' },
            { ...initialServices[1], status: 'connected' as const, message: '已連線' },
          ],
        },
        isLoading: false,
        error: null,
      } as any);

      rerender(
        React.createElement(
          QueryClientProvider,
          {
            client: new QueryClient({ defaultOptions: { queries: { retry: false } } }),
          },
          React.createElement(ServiceStatusDashboard)
        )
      );

      const notification = screen.getByTestId('status-change-notification');
      expect(notification).toBeInTheDocument();
      expect(screen.getByText(/TMDb API：已連線 → 速率限制/)).toBeInTheDocument();
      expect(screen.getByText(/qBittorrent：已斷線 → 已連線/)).toBeInTheDocument();
    });

    it('[P2] auto-dismisses notification after 5 seconds', () => {
      vi.useFakeTimers();

      const initialServices = [
        {
          name: 'tmdb',
          displayName: 'TMDb API',
          status: 'connected' as const,
          message: '已連線',
          lastSuccessAt: '2026-02-10T14:30:00Z',
          lastCheckAt: '2026-02-10T14:30:00Z',
          responseTimeMs: 45,
        },
      ];

      mockUseServiceStatuses.mockReturnValue({
        data: { services: initialServices },
        isLoading: false,
        error: null,
      } as any);

      const { rerender } = renderWithQuery(React.createElement(ServiceStatusDashboard));

      mockUseServiceStatuses.mockReturnValue({
        data: {
          services: [
            { ...initialServices[0], status: 'disconnected' as const, message: 'refused' },
          ],
        },
        isLoading: false,
        error: null,
      } as any);

      rerender(
        React.createElement(
          QueryClientProvider,
          {
            client: new QueryClient({ defaultOptions: { queries: { retry: false } } }),
          },
          React.createElement(ServiceStatusDashboard)
        )
      );

      expect(screen.getByTestId('status-change-notification')).toBeInTheDocument();

      // Advance past 5 seconds
      act(() => {
        vi.advanceTimersByTime(5000);
      });

      expect(screen.queryByTestId('status-change-notification')).not.toBeInTheDocument();

      vi.useRealTimers();
    });

    it('[P2] does not show notification when same status is returned', () => {
      const services = [
        {
          name: 'tmdb',
          displayName: 'TMDb API',
          status: 'connected' as const,
          message: '已連線',
          lastSuccessAt: '2026-02-10T14:30:00Z',
          lastCheckAt: '2026-02-10T14:30:00Z',
          responseTimeMs: 45,
        },
      ];

      mockUseServiceStatuses.mockReturnValue({
        data: { services },
        isLoading: false,
        error: null,
      } as any);

      const { rerender } = renderWithQuery(React.createElement(ServiceStatusDashboard));

      // Re-render with same status (different responseTimeMs but same status)
      mockUseServiceStatuses.mockReturnValue({
        data: { services: [{ ...services[0], responseTimeMs: 50 }] },
        isLoading: false,
        error: null,
      } as any);

      rerender(
        React.createElement(
          QueryClientProvider,
          {
            client: new QueryClient({ defaultOptions: { queries: { retry: false } } }),
          },
          React.createElement(ServiceStatusDashboard)
        )
      );

      expect(screen.queryByTestId('status-change-notification')).not.toBeInTheDocument();
    });

    it('[P2] does not show notification when a new service appears', () => {
      const initialServices = [
        {
          name: 'tmdb',
          displayName: 'TMDb API',
          status: 'connected' as const,
          message: '已連線',
          lastSuccessAt: '2026-02-10T14:30:00Z',
          lastCheckAt: '2026-02-10T14:30:00Z',
          responseTimeMs: 45,
        },
      ];

      mockUseServiceStatuses.mockReturnValue({
        data: { services: initialServices },
        isLoading: false,
        error: null,
      } as any);

      const { rerender } = renderWithQuery(React.createElement(ServiceStatusDashboard));

      // A new service appears (wasn't in previous map)
      mockUseServiceStatuses.mockReturnValue({
        data: {
          services: [
            ...initialServices,
            {
              name: 'ai',
              displayName: 'AI 服務',
              status: 'unconfigured' as const,
              message: '未設定',
              lastSuccessAt: null,
              lastCheckAt: '2026-02-10T14:30:00Z',
              responseTimeMs: 0,
            },
          ],
        },
        isLoading: false,
        error: null,
      } as any);

      rerender(
        React.createElement(
          QueryClientProvider,
          {
            client: new QueryClient({ defaultOptions: { queries: { retry: false } } }),
          },
          React.createElement(ServiceStatusDashboard)
        )
      );

      // No notification — new service is not a "change"
      expect(screen.queryByTestId('status-change-notification')).not.toBeInTheDocument();
    });
  });

  it('[P2] shows fallback error message for non-Error rejection', async () => {
    const user = userEvent.setup();
    const mockMutateAsync = vi.fn().mockRejectedValue('string error');
    mockUseTestServiceConnection.mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
    } as any);
    mockUseServiceStatuses.mockReturnValue({
      data: {
        services: [
          {
            name: 'tmdb',
            displayName: 'TMDb API',
            status: 'connected',
            message: '已連線',
            lastSuccessAt: '2026-02-10T14:30:00Z',
            lastCheckAt: '2026-02-10T14:30:00Z',
            responseTimeMs: 45,
          },
        ],
      },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(ServiceStatusDashboard));
    await user.click(screen.getByTestId('test-btn-tmdb'));
    expect(screen.getByTestId('test-error')).toBeInTheDocument();
    expect(screen.getByText('測試連線失敗')).toBeInTheDocument();
  });
});
