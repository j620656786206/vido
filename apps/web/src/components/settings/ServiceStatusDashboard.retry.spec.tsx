/**
 * dsr-3c — 重試 against a REAL QueryClient.
 *
 * The main spec mocks useServiceStatuses, and a mock can hold "error, not
 * fetching" forever. A real failed query with nothing cached drops back to
 * pending the moment it is refetched (project_tanstack_refetch_no_data_resets_pending):
 * without the `retrying` latch, 重試 would swap the error state for the
 * skeleton and 重試中… would never be seen.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const getAllStatuses = vi.fn();

vi.mock('../../services/serviceStatusService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../services/serviceStatusService')>();
  return {
    ...actual,
    serviceStatusService: {
      ...actual.serviceStatusService,
      getAllStatuses: () => getAllStatuses(),
    },
  };
});

import { ServiceStatusDashboard } from './ServiceStatusDashboard';

function renderDashboard() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <ServiceStatusDashboard />
    </QueryClientProvider>
  );
}

describe('ServiceStatusDashboard — 重試 (real QueryClient)', () => {
  beforeEach(() => getAllStatuses.mockReset());

  it('stays on the error state showing 重試中… while the retry is in flight, then shows the cards', async () => {
    getAllStatuses.mockRejectedValueOnce(new Error('Failed to fetch'));
    let release: (v: unknown) => void = () => {};
    getAllStatuses.mockImplementationOnce(() => new Promise((resolve) => (release = resolve)));
    renderDashboard();

    fireEvent.click(await screen.findByRole('button', { name: '重試' }));

    expect(screen.queryByTestId('status-loading')).toBeNull();
    expect(screen.getByTestId('status-error')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '重試中…' })).toHaveAttribute(
      'aria-disabled',
      'true'
    );
    expect(screen.queryByText('Failed to fetch')).toBeNull();

    release({
      services: [
        {
          name: 'tmdb',
          displayName: 'TMDb API',
          status: 'connected',
          message: 'ok',
          lastSuccessAt: null,
          lastCheckAt: '2026-02-10T14:30:00Z',
          responseTimeMs: 40,
        },
      ],
    });
    await waitFor(() => expect(screen.getByTestId('service-card-tmdb')).toBeInTheDocument());
    expect(screen.queryByTestId('status-error')).toBeNull();
  });
});
