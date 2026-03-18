import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { ServiceStatusCard } from './ServiceStatusCard';
import type { ServiceStatus } from '../../services/serviceStatusService';

const connectedService: ServiceStatus = {
  name: 'tmdb',
  displayName: 'TMDb API',
  status: 'connected',
  message: '已連線',
  lastSuccessAt: '2026-02-10T14:30:00Z',
  lastCheckAt: '2026-02-10T14:30:00Z',
  responseTimeMs: 45,
};

const errorService: ServiceStatus = {
  name: 'qbittorrent',
  displayName: 'qBittorrent',
  status: 'disconnected',
  message: 'connection refused',
  lastSuccessAt: '2026-02-10T14:00:00Z',
  lastCheckAt: '2026-02-10T14:30:00Z',
  responseTimeMs: 5000,
  errorMessage: 'connection refused',
};

const unconfiguredService: ServiceStatus = {
  name: 'ai',
  displayName: 'AI 服務',
  status: 'unconfigured',
  message: '未設定',
  lastSuccessAt: null,
  lastCheckAt: '2026-02-10T14:30:00Z',
  responseTimeMs: 0,
};

const rateLimitedService: ServiceStatus = {
  name: 'tmdb',
  displayName: 'TMDb API',
  status: 'rate_limited',
  message: '速率限制中',
  lastSuccessAt: '2026-02-10T14:29:00Z',
  lastCheckAt: '2026-02-10T14:30:00Z',
  responseTimeMs: 230,
  errorMessage: 'TMDb API rate limit exceeded',
};

describe('ServiceStatusCard', () => {
  it('renders connected service correctly', () => {
    const onTest = vi.fn();
    render(
      React.createElement(ServiceStatusCard, {
        service: connectedService,
        onTest,
        isTesting: false,
      })
    );

    expect(screen.getByText('TMDb API')).toBeInTheDocument();
    expect(screen.getByText('已連線')).toBeInTheDocument();
    expect(screen.getByText('45ms')).toBeInTheDocument();
    expect(screen.getByTestId('service-card-tmdb')).toBeInTheDocument();
  });

  it('renders disconnected service with detail toggle', async () => {
    const user = userEvent.setup();
    const onTest = vi.fn();
    render(
      React.createElement(ServiceStatusCard, {
        service: errorService,
        onTest,
        isTesting: false,
      })
    );

    expect(screen.getByText('qBittorrent')).toBeInTheDocument();
    expect(screen.getByText('已斷線')).toBeInTheDocument();

    // Detail toggle should be visible
    const toggle = screen.getByTestId('detail-toggle-qbittorrent');
    expect(toggle).toBeInTheDocument();

    // Click to expand
    await user.click(toggle);
    expect(screen.getByTestId('detail-panel-qbittorrent')).toBeInTheDocument();
    expect(screen.getByText(/connection refused/)).toBeInTheDocument();
  });

  it('renders unconfigured service without detail toggle', () => {
    const onTest = vi.fn();
    render(
      React.createElement(ServiceStatusCard, {
        service: unconfiguredService,
        onTest,
        isTesting: false,
      })
    );

    expect(screen.getByText('AI 服務')).toBeInTheDocument();
    expect(screen.getByText('未設定')).toBeInTheDocument();
    expect(screen.queryByTestId('detail-toggle-ai')).not.toBeInTheDocument();
  });

  it('renders rate limited service', () => {
    const onTest = vi.fn();
    render(
      React.createElement(ServiceStatusCard, {
        service: rateLimitedService,
        onTest,
        isTesting: false,
      })
    );

    expect(screen.getByText('速率限制')).toBeInTheDocument();
  });

  it('calls onTest when test button is clicked', async () => {
    const user = userEvent.setup();
    const onTest = vi.fn();
    render(
      React.createElement(ServiceStatusCard, {
        service: connectedService,
        onTest,
        isTesting: false,
      })
    );

    await user.click(screen.getByTestId('test-btn-tmdb'));
    expect(onTest).toHaveBeenCalledWith('tmdb');
  });

  it('disables test button when testing', () => {
    const onTest = vi.fn();
    render(
      React.createElement(ServiceStatusCard, {
        service: connectedService,
        onTest,
        isTesting: true,
      })
    );

    const btn = screen.getByTestId('test-btn-tmdb');
    expect(btn).toBeDisabled();
  });

  it('does not show response time for disconnected services', () => {
    const onTest = vi.fn();
    render(
      React.createElement(ServiceStatusCard, {
        service: errorService,
        onTest,
        isTesting: false,
      })
    );

    expect(screen.queryByText('5000ms')).not.toBeInTheDocument();
  });
});
