import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { DatabaseUnavailableBanner } from './DatabaseUnavailableBanner';

function renderBanner() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <DatabaseUnavailableBanner />
    </QueryClientProvider>
  );
}

function mockHealth(body: unknown, status = 200) {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue({
      ok: status < 400,
      status,
      json: () => Promise.resolve(body),
    })
  );
}

describe('DatabaseUnavailableBanner', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders nothing while the database is healthy', async () => {
    mockHealth({ status: 'healthy', database: { status: 'healthy' } });
    renderBanner();

    await waitFor(() => expect(fetch).toHaveBeenCalled());
    expect(screen.queryByTestId('database-unavailable-banner')).not.toBeInTheDocument();
  });

  it('shows the single outage message when the database is unhealthy (503 body)', async () => {
    mockHealth({ status: 'unhealthy', database: { status: 'unhealthy' } }, 503);
    renderBanner();

    const banner = await screen.findByTestId('database-unavailable-banner');
    expect(banner).toHaveTextContent('資料庫目前無法使用');
    expect(banner).toHaveAttribute('role', 'alert');
  });

  it('stays quiet on a network-level fetch failure', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')));
    renderBanner();

    await waitFor(() => expect(fetch).toHaveBeenCalled());
    expect(screen.queryByTestId('database-unavailable-banner')).not.toBeInTheDocument();
  });
});
