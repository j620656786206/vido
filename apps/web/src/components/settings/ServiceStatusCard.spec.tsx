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
  // Freshness used to live ONLY inside 顯示詳情, and the toggle does not exist
  // for connected/unconfigured — the two states a returning user checks most.
  // A green dot that cannot say WHEN it was verified is asking to be trusted.
  describe('freshness is unconditional', () => {
    it.each([
      ['connected', connectedService, 'service-card-tmdb', 'last-check-tmdb'],
      ['unconfigured', unconfiguredService, 'service-card-ai', 'last-check-ai'],
    ])(
      'shows 最後檢查 for a %s service without expanding anything',
      (_label, service, cardId, checkId) => {
        render(
          React.createElement(ServiceStatusCard, {
            service: service as ServiceStatus,
            onTest: vi.fn(),
            isTesting: false,
          })
        );

        expect(screen.getByTestId(cardId)).toBeInTheDocument();
        expect(screen.getByTestId(checkId)).toHaveTextContent('最後檢查');
      }
    );

    it('falls back to — rather than printing an empty time', () => {
      render(
        React.createElement(ServiceStatusCard, {
          service: { ...connectedService, lastCheckAt: '' },
          onTest: vi.fn(),
          isTesting: false,
        })
      );

      expect(screen.getByTestId('last-check-tmdb')).toHaveTextContent('最後檢查 —');
    });
  });

  it('renders connected service correctly', () => {
    const onTest = vi.fn();
    render(
      React.createElement(ServiceStatusCard, {
        service: connectedService,
        onTest,
        isTesting: false,
      })
    );

    expect(screen.getByText('TMDB')).toBeInTheDocument();
    expect(screen.getByText('中繼資料與海報')).toBeInTheDocument();
    expect(screen.getByText('已連線')).toBeInTheDocument();
    expect(screen.getByText('回應 45 ms')).toBeInTheDocument();
    expect(screen.getByTestId('service-card-tmdb')).toBeInTheDocument();
  });

  // dsr-3c: the backend's English names never reach the page.
  it.each([
    ['douban', 'Douban Scraper', '豆瓣'],
    ['wikipedia', 'Wikipedia API', '維基百科'],
    ['ai', 'AI Parser', 'AI 解析'],
    ['tmdb', 'TMDb API', 'TMDB'],
  ])('shows %s by its Chinese name, not %s', (name, displayName, zh) => {
    const { container } = render(
      React.createElement(ServiceStatusCard, {
        service: { ...connectedService, name, displayName },
        onTest: vi.fn(),
        isTesting: false,
      })
    );
    expect(screen.getByRole('heading', { name: zh })).toBeInTheDocument();
    expect(container).not.toHaveTextContent(displayName);
  });

  it('falls back to displayName for a service the label table does not know', () => {
    render(
      React.createElement(ServiceStatusCard, {
        service: { ...connectedService, name: 'sonarr', displayName: 'Sonarr' },
        onTest: vi.fn(),
        isTesting: false,
      })
    );
    expect(screen.getByRole('heading', { name: 'Sonarr' })).toBeInTheDocument();
  });

  it.each([
    ['connected', '--success-tint', '--success-text', '已連線'],
    ['rate_limited', '--warning-tint', '--warning-text', '速率限制'],
    ['error', '--error-tint', '--error-text', '錯誤'],
    ['disconnected', '--error-tint', '--error-text', '已斷線'],
    ['unconfigured', '--bg-tertiary', '--text-muted', '未設定'],
  ])('%s → a pill in %s / %s that says %s', (status, bg, fg, text) => {
    render(
      React.createElement(ServiceStatusCard, {
        service: { ...connectedService, status: status as ServiceStatus['status'] },
        onTest: vi.fn(),
        isTesting: false,
      })
    );
    const pill = screen.getByTestId('status-pill-tmdb');
    expect(pill).toHaveTextContent(text);
    expect(pill.className).toContain(`bg-[var(${bg})]`);
    expect(pill.className).toContain(`text-[var(${fg})]`);
  });

  it('the icon-only re-check button is named after the service in Chinese', () => {
    render(
      React.createElement(ServiceStatusCard, {
        service: { ...connectedService, name: 'douban', displayName: 'Douban Scraper' },
        onTest: vi.fn(),
        isTesting: false,
      })
    );
    const btn = screen.getByRole('button', { name: '重新檢查 豆瓣' });
    expect(btn).toHaveAttribute('data-testid', 'test-btn-douban');
    expect(btn).toHaveTextContent('');
  });

  // 顯示詳情 retired in dsr-3c: the raw error now lives in the dashboard banner's 技術細節.
  it('has no per-card 顯示詳情 toggle and does not print the backend error', () => {
    const { container } = render(
      React.createElement(ServiceStatusCard, {
        service: errorService,
        onTest: vi.fn(),
        isTesting: false,
      })
    );
    expect(screen.queryByText('顯示詳情')).toBeNull();
    expect(screen.queryByTestId('detail-toggle-qbittorrent')).toBeNull();
    expect(container).not.toHaveTextContent('connection refused');
  });

  it('renders disconnected service', () => {
    render(
      React.createElement(ServiceStatusCard, {
        service: errorService,
        onTest: vi.fn(),
        isTesting: false,
      })
    );

    expect(screen.getByText('qBittorrent')).toBeInTheDocument();
    expect(screen.getByText('已斷線')).toBeInTheDocument();
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

  it('disables test button and spins while testing', () => {
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
    expect(screen.getByTestId('test-spinner-tmdb').getAttribute('class')).toContain(
      'motion-safe:animate-spin'
    );
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

    expect(screen.queryByText(/5000/)).not.toBeInTheDocument();
    expect(screen.getByText('回應 —')).toBeInTheDocument();
  });

  it('[P2] does not show response time when responseTimeMs is 0 for connected service', () => {
    const onTest = vi.fn();
    render(
      React.createElement(ServiceStatusCard, {
        service: { ...connectedService, responseTimeMs: 0 },
        onTest,
        isTesting: false,
      })
    );

    expect(screen.queryByText(/0 ms/)).not.toBeInTheDocument();
    expect(screen.getByText('回應 —')).toBeInTheDocument();
  });
});
