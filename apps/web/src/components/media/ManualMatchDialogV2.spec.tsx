import { useState } from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ManualMatchDialogV2 } from './ManualMatchDialogV2';
import { metadataService, type ManualSearchResultItem } from '../../services/metadata';
import { ApiError } from '../../lib/apiError';

vi.mock('../../services/metadata', () => ({
  metadataService: { manualSearch: vi.fn(), applyMetadata: vi.fn() },
}));

const yourName: ManualSearchResultItem = {
  id: 'tmdb-372058',
  source: 'tmdb',
  title: '君の名は。',
  titleZhTw: '你的名字',
  year: 2016,
  mediaType: 'movie',
};
const weathering: ManualSearchResultItem = {
  id: 'tmdb-568160',
  source: 'tmdb',
  title: '天気の子',
  titleZhTw: '天氣之子',
  year: 2019,
  mediaType: 'movie',
};

function renderDialog(over: Partial<React.ComponentProps<typeof ManualMatchDialogV2>> = {}) {
  const props = {
    open: true,
    onOpenChange: vi.fn(),
    mediaId: 'm-1',
    mediaType: 'movie' as const,
    initialQuery: 'Kimi no Na wa',
    ...over,
  };
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <ManualMatchDialogV2 {...props} />
    </QueryClientProvider>
  );
  return props;
}

describe('ManualMatchDialogV2', () => {
  beforeEach(() => {
    vi.mocked(metadataService.manualSearch).mockReset();
    vi.mocked(metadataService.applyMetadata).mockReset();
    vi.mocked(metadataService.manualSearch).mockResolvedValue({
      results: [yourName, weathering],
      totalCount: 2,
      searchedSources: ['tmdb'],
    });
  });

  it('opens prefilled and locked: TMDb only, this item’s type, no pickers', async () => {
    renderDialog();
    expect(await screen.findByRole('dialog')).toHaveTextContent('手動選片');
    expect(screen.getByRole('searchbox')).toHaveValue('Kimi no Na wa');
    await waitFor(() =>
      expect(metadataService.manualSearch).toHaveBeenCalledWith({
        query: 'Kimi no Na wa',
        mediaType: 'movie',
        source: 'tmdb',
      })
    );
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '影集' })).not.toBeInTheDocument();
  });

  it('a series searches TV', async () => {
    renderDialog({ mediaType: 'series', initialQuery: 'Breaking Bad' });
    await waitFor(() =>
      expect(metadataService.manualSearch).toHaveBeenCalledWith({
        query: 'Breaking Bad',
        mediaType: 'tv',
        source: 'tmdb',
      })
    );
  });

  it('does not search while closed', () => {
    renderDialog({ open: false });
    expect(metadataService.manualSearch).not.toHaveBeenCalled();
  });

  it('lists results with the Chinese title, the original title and the year', async () => {
    renderDialog();
    const rows = await screen.findAllByTestId('manual-match-result');
    expect(rows).toHaveLength(2);
    expect(rows[0]).toHaveTextContent('你的名字');
    expect(rows[0]).toHaveTextContent('君の名は。');
    expect(rows[0]).toHaveTextContent('2016');
    // No poster URL → the name-hash gradient tile with the first letter.
    expect(rows[0].querySelector('[data-testid="manual-match-poster-fallback"]')).toHaveTextContent(
      '你'
    );
    expect(rows[0]).not.toHaveTextContent(/TMDB|tmdb/);
  });

  it('says so when nothing matches', async () => {
    vi.mocked(metadataService.manualSearch).mockResolvedValue({
      results: [],
      totalCount: 0,
      searchedSources: ['tmdb'],
    });
    renderDialog();
    expect(await screen.findByText('找不到符合的作品，換個關鍵字試試。')).toBeInTheDocument();
  });

  it('picking a result asks before replacing, then applies it with the result type', async () => {
    vi.mocked(metadataService.applyMetadata).mockResolvedValue({
      success: true,
      mediaId: 'm-1',
      mediaType: 'movie',
      title: '你的名字',
      source: 'tmdb',
      tmdbId: 372058,
      parseStatus: 'success',
    });
    const props = renderDialog();
    expect(screen.queryByTestId('manual-match-apply')).not.toBeInTheDocument();

    fireEvent.click((await screen.findAllByTestId('manual-match-result'))[0]);
    expect(screen.getAllByTestId('manual-match-result')[0]).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByTestId('manual-match-confirm')).toHaveTextContent(
      '套用「你的名字（2016）」？這會用它的片名、海報、簡介取代目前的資料，之後自動比對不會再改它。'
    );

    fireEvent.click(screen.getByTestId('manual-match-apply'));

    await waitFor(() =>
      expect(metadataService.applyMetadata).toHaveBeenCalledWith({
        mediaId: 'm-1',
        mediaType: 'movie',
        selectedItem: { id: 'tmdb-372058', source: 'tmdb', mediaType: 'movie' },
      })
    );
    await waitFor(() => expect(props.onOpenChange).toHaveBeenCalledWith(false));
  });

  it('while applying, the button says 套用中… and cannot be pressed again', async () => {
    vi.mocked(metadataService.applyMetadata).mockReturnValue(new Promise(() => {}));
    renderDialog();
    fireEvent.click((await screen.findAllByTestId('manual-match-result'))[0]);
    fireEvent.click(screen.getByTestId('manual-match-apply'));
    await waitFor(() => expect(screen.getByTestId('manual-match-apply')).toBeDisabled());
    expect(screen.getByTestId('manual-match-apply')).toHaveTextContent('套用中…');
  });

  it.each([
    [
      new ApiError('busy', 409, 'ENRICHMENT_ALREADY_RUNNING'),
      '媒體庫正在比對其他檔案，請稍後再試。',
      null,
    ],
    [new ApiError('nope', 404, 'TMDB_NOT_FOUND'), 'TMDb 上找不到這部作品，請換一筆。', null],
    [
      new ApiError('db locked', 500, 'DB_QUERY_FAILED'),
      '套用失敗，請稍後再試。',
      'DB_QUERY_FAILED',
    ],
  ])(
    'a failed apply (%s) says something true, never the raw message',
    async (error, sentence, code) => {
      vi.mocked(metadataService.applyMetadata).mockRejectedValue(error);
      const props = renderDialog();
      fireEvent.click((await screen.findAllByTestId('manual-match-result'))[0]);
      fireEvent.click(screen.getByTestId('manual-match-apply'));

      const alert = await screen.findByRole('alert');
      expect(alert).toHaveTextContent(sentence);
      expect(alert).not.toHaveTextContent(error.message);
      const pill = screen.queryByTestId('manual-match-error-code');
      if (code) expect(pill).toHaveTextContent(code);
      else expect(pill).toBeNull();
      expect(props.onOpenChange).not.toHaveBeenCalledWith(false);
    }
  );

  it('picking another result clears the error about the previous pick', async () => {
    vi.mocked(metadataService.applyMetadata).mockRejectedValue(
      new ApiError('nope', 404, 'TMDB_NOT_FOUND')
    );
    renderDialog();
    const rows = await screen.findAllByTestId('manual-match-result');
    fireEvent.click(rows[0]);
    fireEvent.click(screen.getByTestId('manual-match-apply'));
    expect(await screen.findByRole('alert')).toHaveTextContent('TMDb 上找不到這部作品');

    fireEvent.click(screen.getAllByTestId('manual-match-result')[1]);
    await waitFor(() => expect(screen.queryByRole('alert')).not.toBeInTheDocument());
    expect(screen.getByTestId('manual-match-confirm')).toHaveTextContent('天氣之子');
  });

  it('closing returns keyboard focus to the button that opened it', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    function Host() {
      const [open, setOpen] = useState(false);
      return (
        <QueryClientProvider client={client}>
          <button type="button" onClick={() => setOpen(true)}>
            手動選片
          </button>
          <ManualMatchDialogV2
            open={open}
            onOpenChange={setOpen}
            mediaId="m-1"
            mediaType="movie"
            initialQuery="Kimi no Na wa"
          />
        </QueryClientProvider>
      );
    }
    render(<Host />);
    const opener = screen.getByRole('button', { name: '手動選片' });
    opener.focus();
    fireEvent.click(opener);
    await screen.findByTestId('manual-match-dialog');

    fireEvent.click(screen.getByRole('button', { name: '取消' }));

    await waitFor(() =>
      expect(screen.queryByTestId('manual-match-dialog')).not.toBeInTheDocument()
    );
    expect(opener).toHaveFocus();
  });

  it('a failed search is not an empty one', async () => {
    vi.mocked(metadataService.manualSearch).mockRejectedValue(
      new ApiError('down', 502, 'TMDB_SERVER_ERROR')
    );
    renderDialog();
    expect(await screen.findByText('搜尋暫時無法使用，請稍後再試。')).toBeInTheDocument();
    expect(screen.queryByText('找不到符合的作品，換個關鍵字試試。')).not.toBeInTheDocument();
  });

  it('typing a new query searches again', async () => {
    renderDialog();
    await waitFor(() => expect(metadataService.manualSearch).toHaveBeenCalledTimes(1));
    fireEvent.change(screen.getByRole('searchbox'), { target: { value: 'Tenki no Ko' } });
    await waitFor(
      () =>
        expect(metadataService.manualSearch).toHaveBeenLastCalledWith({
          query: 'Tenki no Ko',
          mediaType: 'movie',
          source: 'tmdb',
        }),
      { timeout: 2000 }
    );
  });
});
