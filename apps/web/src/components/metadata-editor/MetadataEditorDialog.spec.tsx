/**
 * MetadataEditorDialog Tests (Story 3.8 AC1/AC4; rebuilt by poster-upload-a)
 */

import { useState } from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MetadataEditorDialog } from './MetadataEditorDialog';
import type { MediaMetadata } from './MetadataEditorDialog';
import { useUpdateMetadata } from '../../hooks/useMetadataEditor';

const mockMutateAsync = vi.fn();
vi.mock('../../hooks/useMetadataEditor', () => ({
  useUpdateMetadata: vi.fn(() => ({
    mutateAsync: mockMutateAsync,
    isPending: false,
    error: null,
  })),
}));

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

describe('MetadataEditorDialog', () => {
  const defaultInitialData: MediaMetadata = {
    id: 'test-media-id',
    mediaType: 'movie',
    title: '鬼滅之刃',
    titleEnglish: 'Demon Slayer',
    year: 2019,
    genres: ['動畫', '動作'],
    director: '外崎春雄',
    cast: ['花江夏樹', '鬼頭明里'],
    overview: '大正時代的日本，善良的少年炭治郎...',
    posterUrl: '/abc123.jpg',
  };

  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    mediaId: 'test-media-id',
    mediaType: 'movie' as const,
    initialData: defaultInitialData,
    onSuccess: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(useUpdateMetadata).mockImplementation(
      () =>
        ({
          mutateAsync: mockMutateAsync,
          isPending: false,
          error: null,
        }) as unknown as ReturnType<typeof useUpdateMetadata>
    );
    mockMutateAsync.mockResolvedValue({});
  });

  it('renders the 修改資訊 dialog only while open', () => {
    const { rerender } = renderWithProviders(<MetadataEditorDialog {...defaultProps} />);
    expect(screen.getByRole('dialog', { name: '修改資訊' })).toBeInTheDocument();
    rerender(
      <QueryClientProvider client={new QueryClient()}>
        <MetadataEditorDialog {...defaultProps} isOpen={false} />
      </QueryClientProvider>
    );
    expect(screen.queryByTestId('metadata-editor-dialog')).toBeNull();
  });

  it('pre-fills every field and names each one by its visible label', () => {
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);
    expect(screen.getByLabelText(/片名/, { selector: '#metadata-title' })).toHaveValue('鬼滅之刃');
    expect(screen.getByLabelText('英文片名')).toHaveValue('Demon Slayer');
    expect(screen.getByLabelText('年份')).toHaveValue(2019);
    expect(screen.getByLabelText('導演')).toHaveValue('外崎春雄');
    expect(screen.getByLabelText('簡介')).toHaveValue(defaultInitialData.overview);
    expect(screen.getByRole('group', { name: '類型' })).toBeInTheDocument();
    expect(screen.getByRole('group', { name: '演員' })).toBeInTheDocument();
  });

  it('shows the current poster read-only, and no poster-URL text field', () => {
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);
    const column = screen.getByTestId('metadata-editor-poster-column');
    expect(within(column).getByRole('img', { name: '目前的海報' })).toHaveAttribute(
      'src',
      'https://image.tmdb.org/t/p/w342/abc123.jpg'
    );
    expect(screen.queryByLabelText('海報圖片網址')).toBeNull();
    expect(within(column).queryByRole('button')).toBeNull();
  });

  it('shows 還沒有海報 when there is no poster', () => {
    renderWithProviders(
      <MetadataEditorDialog
        {...defaultProps}
        initialData={{ ...defaultInitialData, posterUrl: undefined }}
      />
    );
    const column = screen.getByTestId('metadata-editor-poster-column');
    expect(within(column).getByText('還沒有海報')).toBeInTheDocument();
    expect(within(column).queryByRole('img')).toBeNull();
  });

  it('falls back to 還沒有海報 when the poster fails to load', () => {
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);
    fireEvent.error(screen.getByRole('img', { name: '目前的海報' }));
    expect(screen.getByText('還沒有海報')).toBeInTheDocument();
  });

  it('keeps the stored zh-TW genres — including ones the menu does not list — and saves names, not keys', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <MetadataEditorDialog
        {...defaultProps}
        initialData={{ ...defaultInitialData, genres: ['劇情', '奇幻冒險'] }}
      />
    );
    const group = screen.getByRole('group', { name: '類型' });
    expect(within(group).getByText('劇情')).toBeInTheDocument();
    expect(within(group).getByText('奇幻冒險')).toBeInTheDocument();

    await user.click(within(group).getByRole('button', { name: '類型' }));
    const menu = await screen.findByRole('menu');
    expect(within(menu).queryByRole('menuitem', { name: '劇情' })).toBeNull();
    expect(within(menu).queryByRole('menuitem', { name: /drama|animation/i })).toBeNull();
    await user.click(within(menu).getByRole('menuitem', { name: '動畫' }));
    await user.click(screen.getByRole('button', { name: '移除類型：劇情' }));

    await user.click(screen.getByRole('button', { name: '儲存' }));
    await waitFor(() => expect(mockMutateAsync).toHaveBeenCalled());
    const sent = mockMutateAsync.mock.calls[0][0];
    expect(sent.genres).toEqual(['奇幻冒險', '動畫']);
    expect(sent).not.toHaveProperty('posterUrl');
  });

  it('offers TV genres for a series', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <MetadataEditorDialog
        {...defaultProps}
        mediaType="series"
        initialData={{ ...defaultInitialData, mediaType: 'series', genres: [] }}
      />
    );
    await user.click(screen.getByRole('button', { name: '類型' }));
    const menu = await screen.findByRole('menu');
    expect(within(menu).getByRole('menuitem', { name: '脫口秀' })).toBeInTheDocument();
    expect(within(menu).queryByRole('menuitem', { name: '電視電影' })).toBeNull();
  });

  it('adds and removes cast; blanks and repeats are ignored; Esc cancels the box without closing the dialog', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderWithProviders(<MetadataEditorDialog {...defaultProps} onClose={onClose} />);
    const group = screen.getByRole('group', { name: '演員' });

    await user.click(within(group).getByRole('button', { name: '演員' }));
    await user.type(screen.getByRole('textbox', { name: '新增演員' }), '下野紘{Enter}');
    await user.type(screen.getByRole('textbox', { name: '新增演員' }), '花江夏樹{Enter}   {Enter}');
    expect(within(group).getAllByText('花江夏樹')).toHaveLength(1);
    expect(within(group).getByText('下野紘')).toBeInTheDocument();

    await user.keyboard('{Escape}');
    expect(screen.queryByRole('textbox', { name: '新增演員' })).toBeNull();
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByRole('dialog', { name: '修改資訊' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '移除演員：下野紘' }));
    expect(within(group).queryByText('下野紘')).toBeNull();
  });

  it('closes with 關閉, 取消 and Esc — but not with a click on the scrim', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderWithProviders(<MetadataEditorDialog {...defaultProps} onClose={onClose} />);

    // The scrim is ui/Dialog's overlay — the sibling of the dialog in the portal.
    const scrim = screen.getByRole('dialog').previousElementSibling as HTMLElement;
    await userEvent.setup({ pointerEventsCheck: 0 }).click(scrim);
    expect(onClose).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: '關閉' }));
    await user.click(screen.getByRole('button', { name: '取消' }));
    await user.keyboard('{Escape}');
    expect(onClose).toHaveBeenCalledTimes(3);
  });

  it('opens with focus in 片名, not on ✕', async () => {
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);
    await waitFor(() => expect(screen.getByDisplayValue('鬼滅之刃')).toHaveFocus());
  });

  it('returns focus to the button that opened it', async () => {
    const user = userEvent.setup();
    function Host() {
      const [open, setOpen] = useState(false);
      return (
        <>
          <button type="button" onClick={() => setOpen(true)}>
            修改資訊
          </button>
          <MetadataEditorDialog {...defaultProps} isOpen={open} onClose={() => setOpen(false)} />
        </>
      );
    }
    renderWithProviders(<Host />);
    const opener = screen.getByRole('button', { name: '修改資訊' });
    await user.click(opener);
    await user.click(await screen.findByRole('button', { name: '取消' }));
    await waitFor(() => expect(opener).toHaveFocus());
  });

  it('shows the required-title error under the field and links it', async () => {
    const user = userEvent.setup();
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);
    const title = screen.getByDisplayValue('鬼滅之刃');
    await user.clear(title);
    await user.click(screen.getByRole('button', { name: '儲存' }));
    expect(await screen.findByText('片名為必填')).toHaveAttribute('id', 'metadata-title-error');
    expect(title).toHaveAttribute('aria-invalid', 'true');
    expect(title).toHaveAttribute('aria-describedby', 'metadata-title-error');
    expect(mockMutateAsync).not.toHaveBeenCalled();
  });

  it('rejects a year below 1900', async () => {
    const user = userEvent.setup();
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);
    const year = screen.getByDisplayValue('2019');
    await user.clear(year);
    await user.type(year, '1800');
    await user.click(screen.getByRole('button', { name: '儲存' }));
    expect(await screen.findByText('年份必須大於 1900')).toBeInTheDocument();
  });

  it('says 請輸入年份 in Chinese when the year is cleared', async () => {
    const user = userEvent.setup();
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);
    await user.clear(screen.getByDisplayValue('2019'));
    await user.click(screen.getByRole('button', { name: '儲存' }));
    expect(await screen.findByText('請輸入年份')).toBeInTheDocument();
    expect(mockMutateAsync).not.toHaveBeenCalled();
  });

  it('keeps an actor that was typed but not yet Entered when 儲存 is pressed', async () => {
    const user = userEvent.setup();
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);
    await user.click(screen.getByRole('button', { name: '演員' }));
    await user.type(screen.getByRole('textbox', { name: '新增演員' }), '下野紘');
    await user.click(screen.getByRole('button', { name: '儲存' }));
    await waitFor(() => expect(mockMutateAsync).toHaveBeenCalled());
    expect(mockMutateAsync.mock.calls[0][0].cast).toEqual(['花江夏樹', '鬼頭明里', '下野紘']);
  });

  it('saves the edited fields, then reports success and closes', async () => {
    const user = userEvent.setup();
    const onSuccess = vi.fn();
    const onClose = vi.fn();
    renderWithProviders(
      <MetadataEditorDialog {...defaultProps} onSuccess={onSuccess} onClose={onClose} />
    );
    const title = screen.getByDisplayValue('鬼滅之刃');
    await user.clear(title);
    await user.type(title, '鬼滅之刃：無限城篇');
    await user.click(screen.getByRole('button', { name: '儲存' }));

    await waitFor(() =>
      expect(mockMutateAsync).toHaveBeenCalledWith(
        expect.objectContaining({
          id: 'test-media-id',
          mediaType: 'movie',
          title: '鬼滅之刃：無限城篇',
          genres: ['動畫', '動作'],
          cast: ['花江夏樹', '鬼頭明里'],
        })
      )
    );
    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalled();
      expect(onClose).toHaveBeenCalled();
    });
  });

  it('keeps 儲存 disabled until something changes', () => {
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);
    expect(screen.getByRole('button', { name: '儲存' })).toBeDisabled();
  });

  it('shows 儲存中… while saving and the error when it fails, staying open', () => {
    vi.mocked(useUpdateMetadata).mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: true,
      error: null,
    } as unknown as ReturnType<typeof useUpdateMetadata>);
    const { unmount } = renderWithProviders(<MetadataEditorDialog {...defaultProps} />);
    expect(screen.getByRole('button', { name: '儲存中…' })).toBeDisabled();
    unmount();

    vi.mocked(useUpdateMetadata).mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
      error: new Error('database is locked'),
    } as unknown as ReturnType<typeof useUpdateMetadata>);
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);
    expect(screen.getByRole('alert')).toHaveTextContent('更新失敗：database is locked');
    expect(screen.getByRole('dialog', { name: '修改資訊' })).toBeInTheDocument();
  });
});
