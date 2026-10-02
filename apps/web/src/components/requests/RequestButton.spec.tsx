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

// --- Story 13-2b: the detail page's tv 想要 opens the season/episode tree ---
vi.mock('./SeasonEpisodeTreeDialog', () => ({
  SeasonEpisodeTreeDialog: ({
    open,
    onConfirm,
    submitError,
  }: {
    open: boolean;
    onConfirm: (p: unknown) => void;
    submitError?: string | null;
  }) =>
    open ? (
      <div data-testid="tree-stub">
        {submitError && <p data-testid="tree-error">{submitError}</p>}
        <button
          onClick={() => onConfirm({ whole: false, seasons: [2], episodes: { '1': [3, 4] } })}
        >
          pick
        </button>
        <button onClick={() => onConfirm({ whole: true })}>pick-all</button>
      </div>
    ) : null,
}));

describe('RequestButton pickEpisodes (13-2b)', () => {
  beforeEach(() => {
    vi.mocked(requestService.createRequest).mockReset();
  });

  it('tv + pickEpisodes: 想要 opens the tree instead of requesting at once', async () => {
    renderButton({ mediaType: 'tv', pickEpisodes: true });
    await userEvent.click(screen.getByTestId('request-button'));
    expect(screen.getByTestId('tree-stub')).toBeInTheDocument();
    expect(requestService.createRequest).not.toHaveBeenCalled();
  });

  it('the tree’s selection rides into createRequest', async () => {
    vi.mocked(requestService.createRequest).mockResolvedValue({} as never);
    renderButton({ mediaType: 'tv', pickEpisodes: true });
    await userEvent.click(screen.getByTestId('request-button'));
    await userEvent.click(screen.getByText('pick'));
    await waitFor(() =>
      expect(requestService.createRequest).toHaveBeenCalledWith({
        tmdbId: 550,
        mediaType: 'tv',
        seasons: [2],
        episodes: { '1': [3, 4] },
      })
    );
  });

  it('全選 (whole) sends NO selection — the same wire as one click', async () => {
    vi.mocked(requestService.createRequest).mockResolvedValue({} as never);
    renderButton({ mediaType: 'tv', pickEpisodes: true });
    await userEvent.click(screen.getByTestId('request-button'));
    await userEvent.click(screen.getByText('pick-all'));
    await waitFor(() =>
      expect(requestService.createRequest).toHaveBeenCalledWith({ tmdbId: 550, mediaType: 'tv' })
    );
  });

  it('movies and card contexts stay one-click', async () => {
    vi.mocked(requestService.createRequest).mockResolvedValue({} as never);
    renderButton({ mediaType: 'movie', pickEpisodes: true });
    await userEvent.click(screen.getByTestId('request-button'));
    expect(screen.queryByTestId('tree-stub')).toBeNull();
    await waitFor(() => expect(requestService.createRequest).toHaveBeenCalledOnce());
  });
});

describe('RequestButton 13-2c', () => {
  beforeEach(() => {
    vi.mocked(requestService.createRequest).mockReset();
  });

  it('secondary variant with a custom label (B4p-D 想要更多集數)', () => {
    renderButton({
      mediaType: 'tv',
      pickEpisodes: true,
      variant: 'secondary',
      label: '想要更多集數',
    });
    const btn = screen.getByTestId('request-button');
    expect(btn).toHaveTextContent('想要更多集數');
    expect(btn.className).toContain('bg-[var(--bg-tertiary)]');
    expect(btn.className).not.toContain('bg-[var(--accent-primary)]');
  });

  it('a failed create keeps the tree open with the reason, instead of closing and losing the picks', async () => {
    vi.mocked(requestService.createRequest).mockRejectedValue(
      new RequestApiError('第 1 季第 2 集已在媒體庫中', 'REQUEST_INVALID_SELECTION')
    );
    renderButton({ mediaType: 'tv', pickEpisodes: true });
    await userEvent.click(screen.getByTestId('request-button'));
    await userEvent.click(screen.getByText('pick'));
    expect(await screen.findByTestId('tree-error')).toHaveTextContent('第 1 季第 2 集已在媒體庫中');
    expect(screen.getByTestId('tree-stub')).toBeInTheDocument();
    expect(screen.queryByTestId('request-toast')).toBeNull();
  });

  it('a successful create closes the tree and shows the toast', async () => {
    vi.mocked(requestService.createRequest).mockResolvedValue({} as never);
    renderButton({ mediaType: 'tv', pickEpisodes: true });
    await userEvent.click(screen.getByTestId('request-button'));
    await userEvent.click(screen.getByText('pick'));
    await waitFor(() => expect(screen.queryByTestId('tree-stub')).toBeNull());
    expect(await screen.findByTestId('request-toast')).toBeInTheDocument();
  });
});
