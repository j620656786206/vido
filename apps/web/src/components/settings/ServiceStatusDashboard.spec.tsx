import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
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
});
