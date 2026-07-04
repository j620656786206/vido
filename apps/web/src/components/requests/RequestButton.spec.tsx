import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';

const navigateMock = vi.fn();
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>();
  return { ...actual, useNavigate: () => navigateMock };
});

vi.mock('../../services/requestService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../services/requestService')>();
  return {
    ...actual,
    requestService: { ...actual.requestService, createRequest: vi.fn() },
  };
});

import { requestService, RequestApiError } from '../../services/requestService';
import { RequestButton } from './RequestButton';

function renderButton(over: Partial<React.ComponentProps<typeof RequestButton>> = {}) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const parentClick = vi.fn();
  const utils = render(
    <QueryClientProvider client={qc}>
      {/* Card contexts mount the button inside a <Link>; the anchor stands in
          for it so the stopPropagation guard is what's actually asserted. */}
      <a href="/somewhere" onClick={parentClick} data-testid="card-link">
        <RequestButton
          tmdbId={550}
          mediaType="movie"
          title="鬥陣俱樂部"
          owned={false}
          requested={false}
          {...over}
        />
      </a>
    </QueryClientProvider>
  );
  return { ...utils, parentClick };
}

describe('RequestButton', () => {
  beforeEach(() => {
    vi.mocked(requestService.createRequest).mockReset();
    navigateMock.mockReset();
  });

  it('已入庫 — owned renders the success pill, no button (L2 state 3)', () => {
    renderButton({ owned: true });
    expect(screen.getByTestId('request-pill-owned')).toHaveTextContent('已入庫');
    expect(screen.queryByTestId('request-button')).not.toBeInTheDocument();
  });

  it('已請求·處理中 — requested renders the info pill, non-actionable (L2 state 2)', () => {
    renderButton({ requested: true });
    expect(screen.getByTestId('request-pill-requested')).toHaveTextContent('已請求 · 處理中');
    expect(screen.queryByTestId('request-button')).not.toBeInTheDocument();
  });

  it('可請求 — click fires the create mutation and never navigates the card link (AC #1/#2)', async () => {
    vi.mocked(requestService.createRequest).mockResolvedValue({
      id: 'server',
      tmdbId: 550,
      mediaType: 'movie',
      title: '鬥陣俱樂部',
      status: 'pending',
      fulfilmentSource: null,
      externalId: null,
      seasons: null,
      episodes: null,
      errorMessage: null,
      requestedAt: '2026-07-04T12:00:00Z',
      updatedAt: '2026-07-04T12:00:00Z',
    });
    const user = userEvent.setup();
    const { parentClick } = renderButton();

    await user.click(screen.getByTestId('request-button'));

    await waitFor(() =>
      expect(requestService.createRequest).toHaveBeenCalledWith({ tmdbId: 550, mediaType: 'movie' })
    );
    expect(parentClick).not.toHaveBeenCalled();
  });

  it('success shows the L8 toast; 查看清單 deep-links to ?view=requests', async () => {
    vi.mocked(requestService.createRequest).mockResolvedValue({
      id: 'server',
      tmdbId: 550,
      mediaType: 'movie',
      title: 'x',
      status: 'pending',
      fulfilmentSource: null,
      externalId: null,
      seasons: null,
      episodes: null,
      errorMessage: null,
      requestedAt: '2026-07-04T12:00:00Z',
      updatedAt: '2026-07-04T12:00:00Z',
    });
    const user = userEvent.setup();
    renderButton();

    await user.click(screen.getByTestId('request-button'));
    await waitFor(() =>
      expect(screen.getByTestId('request-toast')).toHaveTextContent('已加入想要清單')
    );

    await user.click(screen.getByTestId('request-toast-view'));
    expect(navigateMock).toHaveBeenCalledWith({ to: '/discover', search: { view: 'requests' } });
  });

  it('non-duplicate errors surface the backend zh-TW message as an alert (AC #4)', async () => {
    vi.mocked(requestService.createRequest).mockRejectedValue(
      new RequestApiError('此片已在媒體庫中', 'REQUEST_ALREADY_IN_LIBRARY')
    );
    const user = userEvent.setup();
    renderButton();

    await user.click(screen.getByTestId('request-button'));

    await waitFor(() => {
      const toast = screen.getByTestId('request-toast');
      expect(toast).toHaveAttribute('role', 'alert');
      expect(toast).toHaveTextContent('此片已在媒體庫中');
    });
  });

  it('REQUEST_DUPLICATE settles into the requested state with a success toast, not an error (AC #4)', async () => {
    vi.mocked(requestService.createRequest).mockRejectedValue(
      new RequestApiError('已有進行中的請求', 'REQUEST_DUPLICATE')
    );
    const user = userEvent.setup();
    renderButton();

    await user.click(screen.getByTestId('request-button'));

    await waitFor(() => {
      const toast = screen.getByTestId('request-toast');
      expect(toast).toHaveAttribute('role', 'status');
      expect(toast).toHaveTextContent('已加入想要清單');
    });
  });
});
