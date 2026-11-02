import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('../../services/subtitleService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../services/subtitleService')>();
  return {
    ...actual,
    subtitleService: {
      ...actual.subtitleService,
      checkBurnedInSeason: vi.fn(),
      clearBurnedInSeason: vi.fn(),
    },
  };
});

import { BurnedInSubtitleRow } from './BurnedInSubtitleRow';
import { subtitleService } from '../../services/subtitleService';
import { detailKeys } from '../../hooks/useMediaDetails';

const mockedCheck = vi.mocked(subtitleService.checkBurnedInSeason);
const mockedClear = vi.mocked(subtitleService.clearBurnedInSeason);

function renderRow() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
  render(
    <QueryClientProvider client={queryClient}>
      <BurnedInSubtitleRow label="簡中" pillClass="" seriesId="xianni" seasonNumber={1} />
    </QueryClientProvider>
  );
  return { invalidate, user: userEvent.setup() };
}

async function choose(user: ReturnType<typeof userEvent.setup>, item: string) {
  await user.click(screen.getByTestId('burned-in-correct'));
  await user.click(await screen.findByTestId(item));
}

beforeEach(() => vi.clearAllMocks());

describe('BurnedInSubtitleRow — 判斷錯了？ (disc-2026-11-burned-in-subs-c2)', () => {
  it('opens the two choices; 重新檢查 carries its price', async () => {
    const { user } = renderRow();
    await user.click(screen.getByTestId('burned-in-correct'));
    expect(await screen.findByTestId('burned-in-clear')).toHaveTextContent('這季沒有畫面字幕');
    expect(screen.getByTestId('burned-in-recheck')).toHaveTextContent('重新檢查這季');
    expect(screen.getByTestId('burned-in-recheck')).toHaveTextContent('會花約 US$0.002');
  });

  it('這季沒有畫面字幕 → PUT for the season, then refetches the season list', async () => {
    mockedClear.mockResolvedValue({ episodesUpdated: 151 });
    const { user, invalidate } = renderRow();
    await choose(user, 'burned-in-clear');
    await waitFor(() =>
      expect(invalidate).toHaveBeenCalledWith({ queryKey: detailKeys.seasonEpisodes('xianni', 1) })
    );
    expect(mockedClear).toHaveBeenCalledWith({ seriesId: 'xianni', season: 1 });
    expect(mockedCheck).not.toHaveBeenCalled();
  });

  it('重新檢查這季 → POST check, busy while it runs, then refetches', async () => {
    let finish: (
      v: Awaited<ReturnType<typeof subtitleService.checkBurnedInSeason>>
    ) => void = () => {};
    mockedCheck.mockReturnValue(new Promise((resolve) => (finish = resolve)));
    const { user, invalidate } = renderRow();
    await choose(user, 'burned-in-recheck');

    const trigger = screen.getByTestId('burned-in-correct');
    await waitFor(() => expect(trigger).toHaveTextContent('檢查中…'));
    expect(trigger).toBeDisabled();
    expect(trigger).toHaveAttribute('aria-busy', 'true');
    expect(mockedCheck).toHaveBeenCalledWith({ seriesId: 'xianni', season: 1 });

    finish({ result: 'zh-Hans', episodesUpdated: 151, costUsd: 0.0023, checkedEpisodeIds: ['e1'] });
    await waitFor(() => expect(trigger).toHaveTextContent('判斷錯了？'));
    expect(invalidate).toHaveBeenCalledWith({ queryKey: detailKeys.seasonEpisodes('xianni', 1) });
    expect(screen.queryByTestId('burned-in-undecided')).not.toBeInTheDocument();
  });

  it('a re-check that could not decide says so and keeps the answer', async () => {
    mockedCheck.mockResolvedValue({
      result: null,
      reason: 'frames_unavailable',
      episodesUpdated: 0,
      costUsd: 0,
      checkedEpisodeIds: ['e1'],
    });
    const { user } = renderRow();
    await choose(user, 'burned-in-recheck');
    expect(await screen.findByTestId('burned-in-undecided')).toHaveTextContent(
      '這次沒看出結果，先維持原本的判斷'
    );
  });

  it('a failure is said in plain words with the server reason', async () => {
    mockedCheck.mockRejectedValue(
      new Error('還沒設定 Claude 金鑰，沒辦法檢查字幕是不是燒在畫面上。')
    );
    const { user } = renderRow();
    await choose(user, 'burned-in-recheck');
    const alert = await screen.findByTestId('burned-in-error');
    expect(alert).toHaveAttribute('role', 'alert');
    expect(alert).toHaveTextContent('沒辦法重新檢查：還沒設定 Claude 金鑰');

    // Trying the other choice clears the old error.
    mockedClear.mockResolvedValue({ episodesUpdated: 3 });
    await choose(user, 'burned-in-clear');
    await waitFor(() => expect(screen.queryByTestId('burned-in-error')).not.toBeInTheDocument());
  });

  it('a failed correction says which one failed', async () => {
    mockedClear.mockRejectedValue(new Error('season not found'));
    const { user } = renderRow();
    await choose(user, 'burned-in-clear');
    expect(await screen.findByTestId('burned-in-error')).toHaveTextContent(
      '沒辦法改成沒有畫面字幕：season not found'
    );
  });
});
