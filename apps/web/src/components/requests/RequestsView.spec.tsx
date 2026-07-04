import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';

vi.mock('../../services/requestService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../services/requestService')>();
  return {
    ...actual,
    requestService: { ...actual.requestService, listRequests: vi.fn() },
  };
});

import { requestService, type MediaRequest } from '../../services/requestService';
import { RequestsView } from './RequestsView';

const row = (over: Partial<MediaRequest> = {}): MediaRequest => ({
  id: 'r1',
  tmdbId: 550,
  mediaType: 'movie',
  title: '沙丘：第二部',
  status: 'pending',
  fulfilmentSource: null,
  externalId: null,
  seasons: null,
  episodes: null,
  errorMessage: null,
  requestedAt: '2026-06-28T10:00:00Z',
  updatedAt: '2026-06-28T10:00:00Z',
  ...over,
});

function renderView(onExplore = vi.fn()) {
  // retryDelay 0: the view's own `retry: 1` wins over `retry: false`, but the
  // delay stays a client default — zero it so the auto-retry lands within waitFor.
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false, retryDelay: 0 } },
  });
  render(
    <QueryClientProvider client={qc}>
      <RequestsView onExplore={onExplore} />
    </QueryClientProvider>
  );
  return { onExplore };
}

describe('RequestsView (N4 states — design L1/L5/L6/L7)', () => {
  beforeEach(() => {
    vi.mocked(requestService.listRequests).mockReset();
  });

  it('L5 — shows the list skeleton while loading', () => {
    vi.mocked(requestService.listRequests).mockReturnValue(new Promise(() => {}));
    renderView();
    expect(screen.getByTestId('requests-skeleton')).toBeInTheDocument();
  });

  it('L1 — renders rows + Mono count header', async () => {
    vi.mocked(requestService.listRequests).mockResolvedValue([
      row(),
      row({ id: 'r2', tmdbId: 1399, mediaType: 'tv', title: '熊家餐館 S3', status: 'searching' }),
    ]);
    renderView();

    await waitFor(() => expect(screen.getAllByTestId('request-row')).toHaveLength(2));
    expect(screen.getByText('想要清單')).toBeInTheDocument();
    expect(screen.getByText('2')).toBeInTheDocument();
  });

  it('L6 — empty is distinct from failure: 尚無請求 + 前往探索 fires onExplore', async () => {
    vi.mocked(requestService.listRequests).mockResolvedValue([]);
    const user = userEvent.setup();
    const { onExplore } = renderView();

    await waitFor(() => expect(screen.getByTestId('requests-empty')).toBeInTheDocument());
    expect(screen.getByText('尚無請求')).toBeInTheDocument();
    await user.click(screen.getByTestId('requests-go-explore'));
    expect(onExplore).toHaveBeenCalled();
  });

  it('L7 — fetch failure fails soft: 無法載入請求狀態 + 重試 refetches', async () => {
    // The view's useQuery sets retry: 1, so the first failure auto-retries
    // once — reject BOTH attempts to land in the error state, then succeed
    // on the manual 重試 refetch.
    vi.mocked(requestService.listRequests)
      .mockRejectedValueOnce(new Error('boom'))
      .mockRejectedValueOnce(new Error('boom'))
      .mockResolvedValueOnce([row()]);
    const user = userEvent.setup();
    renderView();

    await waitFor(() => expect(screen.getByTestId('requests-error')).toBeInTheDocument());
    expect(screen.getByText('無法載入請求狀態')).toBeInTheDocument();

    await user.click(screen.getByTestId('requests-retry'));
    await waitFor(() => expect(screen.getAllByTestId('request-row')).toHaveLength(1));
  });
});
