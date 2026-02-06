import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ServiceHealthBanner } from './ServiceHealthBanner';
import type { ServicesHealth } from './types';

const mockServices: ServicesHealth = {
  tmdb: {
    name: 'tmdb',
    displayName: 'TMDb API',
    status: 'degraded',
    lastCheck: '2024-01-15T10:00:00Z',
    lastSuccess: '2024-01-15T09:50:00Z',
    errorCount: 2,
  },
  douban: {
    name: 'douban',
    displayName: 'Douban Scraper',
    status: 'healthy',
    lastCheck: '2024-01-15T10:00:00Z',
    lastSuccess: '2024-01-15T10:00:00Z',
    errorCount: 0,
  },
  wikipedia: {
    name: 'wikipedia',
    displayName: 'Wikipedia API',
    status: 'healthy',
    lastCheck: '2024-01-15T10:00:00Z',
    lastSuccess: '2024-01-15T10:00:00Z',
    errorCount: 0,
  },
  ai: {
    name: 'ai',
    displayName: 'AI Parser',
    status: 'down',
    lastCheck: '2024-01-15T10:00:00Z',
    lastSuccess: '2024-01-15T08:00:00Z',
    errorCount: 5,
    message: 'quota exceeded',
  },
};

describe('ServiceHealthBanner', () => {
  it('returns null for normal level', () => {
    const { container } = render(<ServiceHealthBanner level="normal" />);
    expect(container.firstChild).toBeNull();
  });

  it('renders partial degradation banner', () => {
    render(<ServiceHealthBanner level="partial" />);
    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByText(/部分服務暫時降級中/)).toBeInTheDocument();
  });

  it('renders minimal degradation banner', () => {
    render(<ServiceHealthBanner level="minimal" />);
    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByText(/多項服務無法使用/)).toBeInTheDocument();
  });

  it('renders offline banner', () => {
    render(<ServiceHealthBanner level="offline" />);
    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByText(/無法連線到外部服務/)).toBeInTheDocument();
  });

  it('displays custom message', () => {
    render(<ServiceHealthBanner level="partial" message="自訂訊息" />);
    expect(screen.getByText('自訂訊息')).toBeInTheDocument();
  });

  it('shows affected services', () => {
    render(<ServiceHealthBanner level="partial" services={mockServices} />);
    expect(screen.getByText(/受影響服務/)).toBeInTheDocument();
    expect(screen.getByText(/TMDb API/)).toBeInTheDocument();
    expect(screen.getByText(/AI Parser/)).toBeInTheDocument();
  });

  it('calls onDismiss when close button clicked', () => {
    const onDismiss = vi.fn();
    render(<ServiceHealthBanner level="partial" onDismiss={onDismiss} />);

    const closeButton = screen.getByRole('button', { name: /關閉通知/ });
    fireEvent.click(closeButton);

    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it('does not show dismiss button when onDismiss not provided', () => {
    render(<ServiceHealthBanner level="partial" />);
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('applies custom className', () => {
    render(<ServiceHealthBanner level="partial" className="custom-class" />);
    expect(screen.getByRole('alert')).toHaveClass('custom-class');
  });
});
