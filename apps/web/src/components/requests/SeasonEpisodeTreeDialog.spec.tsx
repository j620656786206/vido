import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>();
  return {
    ...actual,
    Link: ({ children, onClick }: { children: React.ReactNode; onClick?: () => void }) => (
      <a href="/discover?view=requests" onClick={onClick}>
        {children}
      </a>
    ),
  };
});
vi.mock('../../services/tmdb', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../services/tmdb')>();
  return {
    ...actual,
    tmdbService: {
      ...actual.tmdbService,
      getTVShowDetails: vi.fn(),
      getSeasonDetails: vi.fn(),
    },
  };
});
vi.mock('../../services/requestService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../services/requestService')>();
  return {
    ...actual,
    requestService: { ...actual.requestService, getCoverage: vi.fn() },
  };
});

import { tmdbService } from '../../services/tmdb';
import { requestService, type RequestCoverage } from '../../services/requestService';
import type { TVShowDetails } from '../../types/tmdb';
import { SeasonEpisodeTreeDialog } from './SeasonEpisodeTreeDialog';

const show = {
  id: 1429,
  name: '進擊的巨人',
  seasons: [
    { id: 0, name: '特別篇', seasonNumber: 0, episodeCount: 8 },
    { id: 1, name: '第 1 季', seasonNumber: 1, episodeCount: 8 },
    { id: 2, name: '第 2 季', seasonNumber: 2, episodeCount: 12 },
  ],
} as unknown as TVShowDetails;

const noCoverage: RequestCoverage = {
  owned: {},
  requestedSeasons: [],
  requestedEpisodes: {},
  wholeSeriesRequested: false,
  activeRequest: false,
};

function seasonOne() {
  return {
    id: 1,
    name: '第 1 季',
    seasonNumber: 1,
    episodes: Array.from({ length: 8 }, (_, i) => ({
      id: 100 + i,
      episodeNumber: i + 1,
      seasonNumber: 1,
      name: `第 ${i + 1} 集`,
      airDate: null,
    })),
  };
}

function renderTree() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onConfirm = vi.fn();
  render(
    <QueryClientProvider client={qc}>
      <SeasonEpisodeTreeDialog
        open
        onOpenChange={vi.fn()}
        tmdbId={1429}
        title="進擊的巨人"
        onConfirm={onConfirm}
      />
    </QueryClientProvider>
  );
  return { onConfirm };
}

describe('SeasonEpisodeTreeDialog (L3-D-v2 He04g)', () => {
  beforeEach(() => {
    vi.mocked(tmdbService.getTVShowDetails).mockReset().mockResolvedValue(show);
    vi.mocked(tmdbService.getSeasonDetails).mockReset().mockResolvedValue(seasonOne());
    vi.mocked(requestService.getCoverage).mockReset().mockResolvedValue(noCoverage);
  });

  it('lists the master row + real seasons (no specials) with Mono counts', async () => {
    renderTree();
    expect(await screen.findByTestId('season-tree')).toBeInTheDocument();
    expect(screen.getByTestId('season-tree-master')).toHaveTextContent('整部影集2 季 · 20 集');
    expect(screen.getByTestId('season-row-1')).toBeInTheDocument();
    expect(screen.getByTestId('season-row-2')).toBeInTheDocument();
    expect(screen.queryByTestId('season-row-0')).toBeNull();
    expect(screen.getByTestId('season-tree-confirm')).toBeDisabled();
  });

  it('loads a season’s episodes only when it is expanded', async () => {
    renderTree();
    await screen.findByTestId('season-tree');
    expect(tmdbService.getSeasonDetails).not.toHaveBeenCalled();

    await userEvent.click(screen.getByTestId('season-expand-1'));
    expect(await screen.findByTestId('episode-row-1-1')).toBeInTheDocument();
    expect(tmdbService.getSeasonDetails).toHaveBeenCalledWith(1429, 1);
    // First six, then 顯示其餘 2 集.
    expect(screen.queryByTestId('episode-row-1-7')).toBeNull();
    await userEvent.click(screen.getByRole('button', { name: /顯示其餘/ }));
    expect(screen.getByTestId('episode-row-1-8')).toBeInTheDocument();
  });

  it('RED LINE: 整部影集 on a show with nothing owned confirms as a whole-title request', async () => {
    const { onConfirm } = renderTree();
    await screen.findByTestId('season-tree');
    await userEvent.click(within(screen.getByTestId('season-tree-master')).getByRole('checkbox'));
    expect(screen.getByTestId('season-tree-summary')).toHaveTextContent('已選 2 季 · 0 集');
    await userEvent.click(screen.getByTestId('season-tree-confirm'));
    await waitFor(() => expect(onConfirm).toHaveBeenCalledWith({ whole: true }));
  });

  it('owned episodes are locked with 已入庫; picking around them sends exact episodes', async () => {
    vi.mocked(requestService.getCoverage).mockResolvedValue({
      ...noCoverage,
      owned: { '1': [1, 2] },
    });
    const { onConfirm } = renderTree();
    await screen.findByTestId('season-tree');
    await userEvent.click(screen.getByTestId('season-expand-1'));
    const owned = await screen.findByTestId('episode-row-1-1');
    expect(owned).toHaveTextContent('已入庫');
    expect(within(owned).getByRole('checkbox')).toBeDisabled();
    expect(within(owned).getByRole('checkbox')).toBeChecked();

    // Season 1 = its 6 missing episodes; season 2 untouched.
    await userEvent.click(within(screen.getByTestId('season-row-1')).getAllByRole('checkbox')[0]);
    await userEvent.click(within(screen.getByTestId('episode-row-1-4')).getByRole('checkbox'));
    expect(screen.getByTestId('season-tree-summary')).toHaveTextContent('已選 0 季 · 5 集');
    await userEvent.click(screen.getByTestId('season-tree-confirm'));
    await waitFor(() =>
      expect(onConfirm).toHaveBeenCalledWith({
        whole: false,
        seasons: [],
        episodes: { '1': [3, 5, 6, 7, 8] },
      })
    );
  });

  it('a season checkbox becomes mixed when only some episodes are picked', async () => {
    renderTree();
    await screen.findByTestId('season-tree');
    await userEvent.click(screen.getByTestId('season-expand-1'));
    await userEvent.click(
      within(await screen.findByTestId('episode-row-1-2')).getByRole('checkbox')
    );
    const seasonBox = within(screen.getByTestId('season-row-1')).getAllByRole(
      'checkbox'
    )[0] as HTMLInputElement;
    expect(seasonBox.indeterminate).toBe(true);
    const master = within(screen.getByTestId('season-tree-master')).getByRole(
      'checkbox'
    ) as HTMLInputElement;
    expect(master.indeterminate).toBe(true);
  });

  it('an open request for the show blocks the tree (⚖️ A)', async () => {
    vi.mocked(requestService.getCoverage).mockResolvedValue({ ...noCoverage, activeRequest: true });
    renderTree();
    expect(await screen.findByTestId('season-tree-active')).toHaveTextContent('已有進行中的請求');
    expect(screen.getByText('查看清單')).toBeInTheDocument();
    expect(screen.queryByTestId('season-tree')).toBeNull();
  });

  it('coverage failure is fail-soft: the tree still opens, without badges', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    vi.mocked(requestService.getCoverage).mockRejectedValue(new Error('down'));
    renderTree();
    expect(await screen.findByTestId('season-tree')).toBeInTheDocument();
    await waitFor(() => expect(warn).toHaveBeenCalled());
    warn.mockRestore();
  });

  it('no seasons → an honest empty state; details failure → retry', async () => {
    vi.mocked(tmdbService.getTVShowDetails).mockResolvedValueOnce({ ...show, seasons: [] });
    renderTree();
    expect(await screen.findByTestId('season-tree-empty')).toBeInTheDocument();
  });

  it('details failure shows an error with 重試', async () => {
    vi.mocked(tmdbService.getTVShowDetails).mockRejectedValue(new Error('tmdb down'));
    renderTree();
    const err = await screen.findByTestId('season-tree-error');
    expect(within(err).getByRole('button', { name: '重試' })).toBeInTheDocument();
  });

  it('CR: a selection that resolves to nothing is refused, never sent as the whole title', async () => {
    // TMDb's season list is behind the show's episode_count: 8 counted, the
    // season endpoint lists 3, and all three are owned.
    vi.mocked(requestService.getCoverage).mockResolvedValue({
      ...noCoverage,
      owned: { '1': [1, 2, 3] },
    });
    vi.mocked(tmdbService.getSeasonDetails).mockResolvedValue({
      ...seasonOne(),
      episodes: seasonOne().episodes.slice(0, 3),
    });
    const { onConfirm } = renderTree();
    await screen.findByTestId('season-tree');
    await userEvent.click(within(screen.getByTestId('season-row-1')).getAllByRole('checkbox')[0]);
    await userEvent.click(screen.getByTestId('season-tree-confirm'));
    expect(await screen.findByRole('alert')).toHaveTextContent('選到的集數都已入庫或已請求');
    expect(onConfirm).not.toHaveBeenCalled();
  });
});
