import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { NfoLocalizeAction } from './NfoLocalizeAction';
import { NfoLocalizeApiError, NFO_ERROR_CODES } from '../../services/nfoLocalizerService';

const navigate = vi.fn();
vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => navigate,
}));

const localizeMovieNfo = vi.fn();
const localizeSeriesNfo = vi.fn();
vi.mock('../../services/nfoLocalizerService', async () => {
  const actual = await vi.importActual<typeof import('../../services/nfoLocalizerService')>(
    '../../services/nfoLocalizerService'
  );
  return {
    ...actual,
    nfoLocalizerService: {
      localizeMovieNfo: (...args: unknown[]) => localizeMovieNfo(...args),
      localizeSeriesNfo: (...args: unknown[]) => localizeSeriesNfo(...args),
      localizeEpisodeNfo: vi.fn(),
    },
  };
});

function renderAction(props: Partial<React.ComponentProps<typeof NfoLocalizeAction>> = {}) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <NfoLocalizeAction mediaType="movie" id="m-1" hasFilePath {...props} />
    </QueryClientProvider>
  );
}

const singleOk = { path: '/media/x.nfo', backupPath: '', replaced: false };

beforeEach(() => {
  vi.clearAllMocks();
  localizeMovieNfo.mockResolvedValue(singleOk);
  localizeSeriesNfo.mockResolvedValue(singleOk);
});

describe('NfoLocalizeAction — entry', () => {
  it('renders the labelled button (never an icon-only guess)', () => {
    renderAction();
    const btn = screen.getByTestId('action-localize-nfo');
    expect(btn).toHaveTextContent('在地化資訊');
  });

  it('renders nothing without a file path — there is nowhere to put a .nfo', () => {
    const { container } = renderAction({ hasFilePath: false });
    expect(container).toBeEmptyDOMElement();
  });
});

// 🔴 The single most important guarantee in this story. The backend gate exists
// to stop a curated tvshow.nfo being overwritten without consent; a frontend
// that fired on click — or that hard-coded confirm_replace — would defeat it
// while every other test still passed.
describe('NfoLocalizeAction — the confirm gate', () => {
  it('clicking the button calls NOTHING, it only opens the dialog', async () => {
    const user = userEvent.setup();
    renderAction({ mediaType: 'tv', id: 's-1' });

    await user.click(screen.getByTestId('action-localize-nfo'));

    expect(await screen.findByTestId('nfo-localize-dialog')).toBeInTheDocument();
    expect(localizeSeriesNfo).not.toHaveBeenCalled();
    expect(localizeMovieNfo).not.toHaveBeenCalled();
  });

  it('cancelling calls NOTHING', async () => {
    const user = userEvent.setup();
    renderAction({ mediaType: 'tv', id: 's-1' });

    await user.click(screen.getByTestId('action-localize-nfo'));
    await user.click(await screen.findByRole('button', { name: '取消' }));

    expect(localizeSeriesNfo).not.toHaveBeenCalled();
  });

  it('pressing Escape calls NOTHING', async () => {
    const user = userEvent.setup();
    renderAction({ mediaType: 'tv', id: 's-1' });

    await user.click(screen.getByTestId('action-localize-nfo'));
    await screen.findByTestId('nfo-localize-dialog');
    await user.keyboard('{Escape}');

    await waitFor(() =>
      expect(screen.queryByTestId('nfo-localize-dialog')).not.toBeInTheDocument()
    );
    expect(localizeSeriesNfo).not.toHaveBeenCalled();
  });

  it('only the confirm button sends confirm_replace: true', async () => {
    const user = userEvent.setup();
    renderAction({ mediaType: 'tv', id: 's-1' });

    await user.click(screen.getByTestId('action-localize-nfo'));
    await user.click(await screen.findByTestId('nfo-confirm'));

    await waitFor(() =>
      expect(localizeSeriesNfo).toHaveBeenCalledWith('s-1', {
        confirmReplace: true,
        includeEpisodes: false,
      })
    );
  });

  // Sally overruled the original spec: a movie run is additive but still spends
  // LLM budget, and 2026-08-19「花錢須同意」means paid work needs an explicit yes.
  it('movies get a dialog too — a paid action never fires on one click', async () => {
    const user = userEvent.setup();
    renderAction({ mediaType: 'movie', id: 'm-1' });

    await user.click(screen.getByTestId('action-localize-nfo'));

    expect(await screen.findByTestId('nfo-localize-dialog')).toBeInTheDocument();
    expect(localizeMovieNfo).not.toHaveBeenCalled();
  });
});

describe('NfoLocalizeAction — dialog copy', () => {
  it('movie: promises the original is safe AND that it costs', async () => {
    const user = userEvent.setup();
    renderAction({ mediaType: 'movie' });
    await user.click(screen.getByTestId('action-localize-nfo'));

    const dialog = await screen.findByTestId('nfo-localize-dialog');
    expect(dialog).toHaveTextContent('將資訊在地化為繁體中文');
    expect(dialog).toHaveTextContent('不會覆寫你現有的 .nfo —— 會寫進另一個播放器同樣認得的檔名');
    expect(dialog).toHaveTextContent('會使用 AI 翻譯額度');
    expect(screen.getByTestId('nfo-confirm')).toHaveTextContent('開始在地化');
    expect(screen.queryByTestId('nfo-include-episodes')).not.toBeInTheDocument();
  });

  it('tv: states the overwrite, names the backup, and does not say a vague 確定', async () => {
    const user = userEvent.setup();
    renderAction({ mediaType: 'tv' });
    await user.click(screen.getByTestId('action-localize-nfo'));

    const dialog = await screen.findByTestId('nfo-localize-dialog');
    expect(dialog).toHaveTextContent('影集只有一個檔名可用，這會覆寫現有的 tvshow.nfo。');
    expect(dialog).toHaveTextContent(
      '原始檔會先備份成 tvshow.nfo.orig；之後再執行也不會覆蓋這份備份。'
    );
    expect(dialog).toHaveTextContent('會使用 AI 翻譯額度');

    const confirm = screen.getByTestId('nfo-confirm');
    expect(confirm).toHaveTextContent('備份並覆寫');
    expect(confirm).not.toHaveTextContent('確定');
  });

  // Checked by default would let a first attempt cost 24x.
  it('tv: the per-episode option is UNCHECKED by default and warns about cost', async () => {
    const user = userEvent.setup();
    renderAction({ mediaType: 'tv' });
    await user.click(screen.getByTestId('action-localize-nfo'));

    const cb = await screen.findByTestId('nfo-include-episodes');
    expect(cb).not.toBeChecked();
    expect(screen.getByTestId('nfo-localize-dialog')).toHaveTextContent(
      '每一集各翻譯一次，額度用量會明顯增加。'
    );
  });

  // CR M1 — cost-sensitive: a tick from a cancelled dialog must not survive
  // into the next opening. Every open starts unchecked.
  it('re-opening the dialog resets the per-episode option to unchecked', async () => {
    const user = userEvent.setup();
    renderAction({ mediaType: 'tv' });

    await user.click(screen.getByTestId('action-localize-nfo'));
    await user.click(await screen.findByTestId('nfo-include-episodes'));
    expect(screen.getByTestId('nfo-include-episodes')).toBeChecked();
    await user.click(screen.getByRole('button', { name: '取消' }));

    await user.click(screen.getByTestId('action-localize-nfo'));
    expect(await screen.findByTestId('nfo-include-episodes')).not.toBeChecked();
  });

  // CR M3 — AC #5.5 asks for focus management to be ASSERTED, not assumed.
  it('moves focus into the dialog on open and back to the trigger on close', async () => {
    const user = userEvent.setup();
    renderAction({ mediaType: 'tv' });
    const trigger = screen.getByTestId('action-localize-nfo');
    // Keyboard activation keeps focus on the trigger, which is what Radix's
    // FocusScope records and restores (TrailerModal.spec precedent).
    trigger.focus();
    expect(document.activeElement).toBe(trigger);

    await user.keyboard('{Enter}');
    const dialog = await screen.findByTestId('nfo-localize-dialog');
    await waitFor(() => expect(dialog).toContainElement(document.activeElement as HTMLElement));

    await user.keyboard('{Escape}');
    await waitFor(() => expect(document.activeElement).toBe(trigger));
  });

  it('ticking the option forwards includeEpisodes: true', async () => {
    const user = userEvent.setup();
    renderAction({ mediaType: 'tv', id: 's-9' });

    await user.click(screen.getByTestId('action-localize-nfo'));
    await user.click(await screen.findByTestId('nfo-include-episodes'));
    await user.click(screen.getByTestId('nfo-confirm'));

    await waitFor(() =>
      expect(localizeSeriesNfo).toHaveBeenCalledWith('s-9', {
        confirmReplace: true,
        includeEpisodes: true,
      })
    );
  });
});

describe('NfoLocalizeAction — result pill', () => {
  async function confirmAs(mediaType: 'movie' | 'tv') {
    const user = userEvent.setup();
    renderAction({ mediaType });
    await user.click(screen.getByTestId('action-localize-nfo'));
    await user.click(await screen.findByTestId('nfo-confirm'));
    return user;
  }

  it('movie success replaces the button in place', async () => {
    await confirmAs('movie');

    const pill = await screen.findByTestId('nfo-result-ok');
    expect(pill).toHaveTextContent('已寫入繁中資訊');
    expect(pill).toHaveAttribute('role', 'status');
    expect(pill).toHaveAttribute('aria-live', 'polite');
    expect(screen.queryByTestId('action-localize-nfo')).not.toBeInTheDocument();
  });

  it('tv success names where the original went', async () => {
    localizeSeriesNfo.mockResolvedValue({
      path: '/media/Buffy/tvshow.nfo',
      backupPath: '/media/Buffy/tvshow.nfo.orig',
      replaced: true,
    });
    await confirmAs('tv');

    expect(await screen.findByTestId('nfo-result-ok')).toHaveTextContent(
      '已覆寫，原檔已備份為 .nfo.orig'
    );
  });

  // CR H1 — the pill follows `replaced`, not the media type. A show with no
  // tvshow.nfo yet has NOTHING backed up; claiming a .orig exists is a lie.
  it('tv with no pre-existing file must NOT claim a backup was taken', async () => {
    localizeSeriesNfo.mockResolvedValue({
      path: '/media/Buffy/tvshow.nfo',
      backupPath: '',
      replaced: false,
    });
    await confirmAs('tv');

    const pill = await screen.findByTestId('nfo-result-ok');
    expect(pill).toHaveTextContent('已寫入繁中資訊');
    expect(pill).not.toHaveTextContent('.orig');
  });

  // CR H1 — and the mirror: a movie whose two slots were both occupied IS
  // replaced (backend backup-and-replace branch); saying "written" understates.
  it('movie with both slots occupied reports the replace + backup', async () => {
    localizeMovieNfo.mockResolvedValue({
      path: '/media/x.nfo',
      backupPath: '/media/x.nfo.orig',
      replaced: true,
    });
    await confirmAs('movie');

    expect(await screen.findByTestId('nfo-result-ok')).toHaveTextContent(
      '已覆寫，原檔已備份為 .nfo.orig'
    );
  });

  it('a clean whole-show run reads as success', async () => {
    localizeSeriesNfo.mockResolvedValue({
      show: singleOk,
      episodes: [],
      succeeded: 24,
      failed: 0,
      skipped: 0,
    });
    await confirmAs('tv');

    const pill = await screen.findByTestId('nfo-result-batch');
    expect(pill).toHaveTextContent('影集資訊已更新 · 24 集完成');
    expect(pill).not.toHaveTextContent('略過');
  });

  it('skipped episodes are surfaced as 略過, not as failures', async () => {
    localizeSeriesNfo.mockResolvedValue({
      show: singleOk,
      episodes: [],
      succeeded: 22,
      failed: 0,
      skipped: 2,
    });
    await confirmAs('tv');

    expect(await screen.findByTestId('nfo-result-batch')).toHaveTextContent(
      '影集資訊已更新 · 22 集完成、2 集略過'
    );
  });

  it('failures are counted separately from skips', async () => {
    localizeSeriesNfo.mockResolvedValue({
      show: singleOk,
      episodes: [],
      succeeded: 20,
      failed: 1,
      skipped: 3,
    });
    await confirmAs('tv');

    expect(await screen.findByTestId('nfo-result-batch')).toHaveTextContent(
      '20 集完成、3 集略過、1 集失敗'
    );
  });
});

describe('NfoLocalizeAction — failure modes', () => {
  it('503 offers a way to fix it, not just a complaint', async () => {
    localizeMovieNfo.mockRejectedValue(
      new NfoLocalizeApiError('disabled', NFO_ERROR_CODES.disabled)
    );
    const user = userEvent.setup();
    renderAction({ mediaType: 'movie' });
    await user.click(screen.getByTestId('action-localize-nfo'));
    await user.click(await screen.findByTestId('nfo-confirm'));

    const pill = await screen.findByTestId('nfo-result-disabled');
    expect(pill).toHaveTextContent('尚未設定翻譯服務');

    await user.click(screen.getByRole('button', { name: '前往設定' }));
    expect(navigate).toHaveBeenCalledWith({ to: '/settings/keys' });
  });

  it('400 tells the user the actionable thing: scan first', async () => {
    localizeMovieNfo.mockRejectedValue(
      new NfoLocalizeApiError('no path', NFO_ERROR_CODES.missingPath)
    );
    const user = userEvent.setup();
    renderAction({ mediaType: 'movie' });
    await user.click(screen.getByTestId('action-localize-nfo'));
    await user.click(await screen.findByTestId('nfo-confirm'));

    expect(await screen.findByTestId('nfo-result-error')).toHaveTextContent('請先掃描媒體庫');
  });

  // CR M2 — the backend's 500 message is English; the user reads zh-TW (Rule 3).
  it('500 shows a zh-TW message, not the raw English server text', async () => {
    localizeMovieNfo.mockRejectedValue(
      new NfoLocalizeApiError('Failed to localize metadata', NFO_ERROR_CODES.failed)
    );
    const user = userEvent.setup();
    renderAction({ mediaType: 'movie' });
    await user.click(screen.getByTestId('action-localize-nfo'));
    await user.click(await screen.findByTestId('nfo-confirm'));

    const pill = await screen.findByTestId('nfo-result-error');
    expect(pill).toHaveTextContent('在地化失敗，請查看伺服器記錄');
    expect(pill).not.toHaveTextContent('Failed to localize');
  });

  it('an unknown code falls through to the server message', async () => {
    localizeMovieNfo.mockRejectedValue(new NfoLocalizeApiError('disk full', 'SOMETHING_NEW'));
    const user = userEvent.setup();
    renderAction({ mediaType: 'movie' });
    await user.click(screen.getByTestId('action-localize-nfo'));
    await user.click(await screen.findByTestId('nfo-confirm'));

    expect(await screen.findByTestId('nfo-result-error')).toHaveTextContent('disk full');
  });

  // CR H2 — an error must not be a dead end: the user gets the button back
  // without reloading the page.
  it('the error pill offers 重試, which restores the button', async () => {
    localizeMovieNfo.mockRejectedValue(new NfoLocalizeApiError('boom', NFO_ERROR_CODES.failed));
    const user = userEvent.setup();
    renderAction({ mediaType: 'movie' });
    await user.click(screen.getByTestId('action-localize-nfo'));
    await user.click(await screen.findByTestId('nfo-confirm'));

    await user.click(await screen.findByTestId('nfo-retry'));

    expect(screen.getByTestId('action-localize-nfo')).toBeInTheDocument();
    expect(screen.queryByTestId('nfo-result-error')).not.toBeInTheDocument();
    // Retrying is still gated — nothing fired on the way back.
    expect(localizeMovieNfo).toHaveBeenCalledTimes(1);
  });

  // Reaching 409 from the UI means the confirm gate leaked — it must be visible,
  // not swallowed, so the bug is diagnosable.
  it('409 is shown rather than silently ignored', async () => {
    localizeSeriesNfo.mockRejectedValue(
      new NfoLocalizeApiError('not confirmed', NFO_ERROR_CODES.notConfirmed)
    );
    const user = userEvent.setup();
    renderAction({ mediaType: 'tv' });
    await user.click(screen.getByTestId('action-localize-nfo'));
    await user.click(await screen.findByTestId('nfo-confirm'));

    const pill = await screen.findByTestId('nfo-result-error');
    expect(pill).toHaveTextContent('未經確認的請求被拒絕');
    expect(pill).toHaveTextContent('程式錯誤');
  });
});
