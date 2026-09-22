import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { Download, TorrentStatus } from '../../services/downloadService';

const h = vi.hoisted(() => ({
  navigate: vi.fn(),
  search: {} as Record<string, unknown>,
  useDownloads: vi.fn(),
  pause: vi.fn(),
  resume: vi.fn(),
  remove: vi.fn(),
  qbtConfig: { configured: true } as { configured: boolean } | undefined,
  counts: undefined as Record<string, number> | undefined,
  isPhone: false,
}));

vi.mock('@tanstack/react-router', () => ({
  getRouteApi: () => ({ useSearch: () => h.search, useNavigate: () => h.navigate }),
  Link: ({ to, children, ...props }: { to: string; children: React.ReactNode }) => (
    <a href={to} {...props}>
      {children}
    </a>
  ),
}));
vi.mock('../../hooks/useDownloads', () => ({
  useDownloads: (...args: unknown[]) => h.useDownloads(...args),
  useDownloadCounts: () => ({ data: h.counts }),
  usePageVisibility: () => true,
}));
vi.mock('../../hooks/useDownloadActions', () => ({
  useDownloadActions: () => ({
    pause: { mutate: h.pause },
    resume: { mutate: h.resume },
    remove: { mutate: h.remove },
  }),
}));
vi.mock('../../hooks/useDownloadProgress', () => ({
  useDownloadProgress: () => ({ startTracking: vi.fn(), stopTracking: vi.fn() }),
}));
vi.mock('../../hooks/useQBittorrent', () => ({
  useQBittorrentConfig: () => ({ data: h.qbtConfig }),
}));
// The hook, not matchMedia: `false` is what the global test-setup stub already yields, so
// every other spec here runs exactly as before; the rotation test flips it.
vi.mock('../../hooks/useIsPhone', () => ({ useIsPhone: () => h.isPhone }));

import { DownloadsBrowseV2 } from './DownloadsBrowseV2';

const dl = (hash: string, status: TorrentStatus): Download => ({
  hash,
  name: `${hash}.mkv`,
  size: 1_000_000_000,
  progress: 0.5,
  downloadSpeed: 1_000_000,
  uploadSpeed: 0,
  eta: 60,
  status,
  addedOn: '2026-09-01T00:00:00Z',
  seeds: 1,
  peers: 1,
  downloaded: 0,
  uploaded: 0,
  ratio: 0,
  savePath: '/dl',
});

const PAGE = {
  items: [dl('a', 'downloading'), dl('b', 'paused')],
  page: 1,
  pageSize: 100,
  totalItems: 2,
  totalPages: 1,
};

const realMatchMedia = window.matchMedia;

beforeEach(() => {
  vi.clearAllMocks();
  h.search = {};
  h.isPhone = false;
  h.qbtConfig = { configured: true };
  h.counts = { all: 2, downloading: 1, paused: 1, completed: 0, seeding: 0, error: 0 };
  localStorage.clear();
  h.useDownloads.mockReturnValue({ data: PAGE, isLoading: false, error: null, refetch: vi.fn() });
});

afterEach(() => {
  window.matchMedia = realMatchMedia;
});

describe('DownloadsBrowseV2 — list select mode (dsr-4 D1-D-v2 / D2-D-v2)', () => {
  it('選取 swaps the toolbar for the batch bar; 取消 in the header brings it back', async () => {
    const user = userEvent.setup();
    render(<DownloadsBrowseV2 />);
    expect(screen.getByText('管理所有下載任務')).toBeInTheDocument();
    expect(screen.getByRole('combobox', { name: '排序方式' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '選取' }));
    expect(screen.getByText('批次選取模式')).toBeInTheDocument();
    expect(screen.getByTestId('downloads-batch-bar')).toBeInTheDocument();
    expect(screen.queryByRole('combobox', { name: '排序方式' })).toBeNull();

    await user.click(screen.getByRole('button', { name: '取消' }));
    expect(screen.queryByTestId('downloads-batch-bar')).toBeNull();
    expect(screen.getByRole('button', { name: '選取' })).toBeInTheDocument();
  });

  it('全選 → 批次繼續 resumes every selected torrent', async () => {
    const user = userEvent.setup();
    render(<DownloadsBrowseV2 />);
    await user.click(screen.getByRole('button', { name: '選取' }));
    await user.click(screen.getByRole('button', { name: '全選' }));
    await user.click(screen.getByRole('button', { name: '批次繼續' }));
    expect(h.resume).toHaveBeenCalledWith(['a', 'b']);
  });

  it('switching filter drops the selection, so a batch action cannot reach rows no longer shown', async () => {
    const user = userEvent.setup();
    render(<DownloadsBrowseV2 />);
    await user.click(screen.getByRole('button', { name: '選取' }));
    await user.click(screen.getByRole('button', { name: '全選' }));
    const bar = screen.getByTestId('downloads-batch-bar');
    expect(within(bar).getByText('2')).toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: /已暫停/ }));
    expect(h.navigate).toHaveBeenCalled();
    expect(within(bar).getByText('0')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '批次暫停' })).toBeDisabled();
  });
});

describe('DownloadsBrowseV2 — a selection never reaches rows that are gone (dsr-4 CR)', () => {
  it('a selected torrent that leaves the page on its own drops out of the count and the batch call', async () => {
    const user = userEvent.setup();
    const { rerender } = render(<DownloadsBrowseV2 />);
    await user.click(screen.getByRole('button', { name: '選取' }));
    await user.click(screen.getByRole('button', { name: '全選' }));
    expect(within(screen.getByTestId('downloads-batch-bar')).getByText('2')).toBeInTheDocument();

    // b finishes and falls out of the list on the next refresh
    h.useDownloads.mockReturnValue({
      data: { ...PAGE, items: [PAGE.items[0]], totalItems: 1 },
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });
    rerender(<DownloadsBrowseV2 />);
    expect(within(screen.getByTestId('downloads-batch-bar')).getByText('1')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '批次暫停' }));
    expect(h.pause).toHaveBeenCalledWith(['a']);
  });

  it('changing the sort drops the selection (a new sort can bring a different page)', async () => {
    const user = userEvent.setup();
    render(<DownloadsBrowseV2 />);
    await user.click(screen.getByRole('button', { name: '選取' }));
    await user.click(screen.getByRole('button', { name: '全選' }));
    await user.click(screen.getByRole('button', { name: '取消' }));

    await user.selectOptions(screen.getByRole('combobox', { name: '排序方式' }), 'name:asc');
    await user.click(screen.getByRole('button', { name: '選取' }));
    expect(within(screen.getByTestId('downloads-batch-bar')).getByText('0')).toBeInTheDocument();
  });

  it('選取 is unavailable while there is nothing to select', () => {
    h.useDownloads.mockReturnValue({
      data: { ...PAGE, items: [], totalItems: 0 },
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });
    render(<DownloadsBrowseV2 />);
    expect(screen.getByRole('button', { name: '選取' })).toBeDisabled();
  });
});

describe('DownloadsBrowseV2 — toolbar + states', () => {
  it('the sort control sets field and direction together', async () => {
    const user = userEvent.setup();
    render(<DownloadsBrowseV2 />);
    await user.selectOptions(screen.getByRole('combobox', { name: '排序方式' }), 'name:asc');
    expect(h.useDownloads).toHaveBeenLastCalledWith('all', 'name', 'asc', 1, 100);
  });

  it('the seeding chip uses the same word as the status pill (做種)', () => {
    render(<DownloadsBrowseV2 />);
    expect(screen.getByRole('tab', { name: /^做種/ })).toBeInTheDocument();
  });

  it('the page summary reads the real range', () => {
    render(<DownloadsBrowseV2 />);
    expect(screen.getByTestId('downloads-page-summary')).toHaveTextContent('1–2 / 2');
  });

  it('when qBittorrent cannot be reached the toolbar hides and only the fail-soft card shows', () => {
    h.useDownloads.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('connection refused'),
      refetch: vi.fn(),
    });
    render(<DownloadsBrowseV2 />);
    expect(screen.getByRole('alert')).toHaveTextContent('無法連線到 qBittorrent');
    expect(screen.queryByRole('button', { name: '選取' })).toBeNull();
    expect(screen.queryByRole('combobox', { name: '排序方式' })).toBeNull();
  });

  it('qBittorrent never set up → the not-configured card, no 重試, no alert, chips say「—」', () => {
    h.qbtConfig = { configured: false };
    h.counts = undefined;
    h.useDownloads.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });
    render(<DownloadsBrowseV2 />);
    expect(screen.getByTestId('downloads-qbt-not-configured-v2')).toBeInTheDocument();
    expect(screen.queryByTestId('downloads-qbt-error-v2')).toBeNull();
    expect(screen.queryByRole('alert')).toBeNull();
    expect(screen.queryByRole('button', { name: '重試' })).toBeNull();
    expect(screen.queryByRole('button', { name: '選取' })).toBeNull();
    expect(screen.getByRole('tab', { name: /全部/ })).toHaveTextContent('全部—');
    // no counts means no error chip either — not a made-up「錯誤 0」
    expect(screen.queryByRole('tab', { name: /錯誤/ })).toBeNull();
  });

  it('table view (desktop) shows the task count where 選取 was — its checkboxes are always on', async () => {
    window.matchMedia = ((query: string) => ({
      matches: true,
      media: query,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    })) as unknown as typeof window.matchMedia;
    const user = userEvent.setup();
    render(<DownloadsBrowseV2 />);

    await user.click(screen.getByRole('button', { name: '表格檢視' }));
    expect(screen.getByTestId('downloads-table-v2')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '表格檢視' })).toHaveAttribute(
      'aria-pressed',
      'true'
    );
    expect(screen.getByText('筆任務', { exact: false })).toHaveTextContent('共 2 筆任務');
    expect(screen.queryByRole('button', { name: '選取' })).toBeNull();
  });
});

/** Whole class tokens — never substrings (`sm:hidden` is inside `max-sm:hidden`). */
const tokens = (el: Element) => (el.getAttribute('class') ?? '').split(/\s+/).filter(Boolean);

// jsdom does not evaluate media queries: these pin the CSS tokens that decide the
// phone layout. The layout itself is measured by tests/e2e/downloads-mobile.spec.ts.
describe('DownloadsBrowseV2 — phone sort sheet + chip row (dsr-4b-1 D1-M-v2 / D10-M-v2)', () => {
  it('a 排序 button sits in the page header, phone-only (sm:hidden)', () => {
    render(<DownloadsBrowseV2 />);
    const btn = screen.getByTestId('downloads-sort-btn');
    expect(btn).toHaveAccessibleName('排序');
    expect(btn).toHaveAttribute('aria-haspopup', 'dialog');
    expect(btn).toHaveAttribute('aria-expanded', 'false');
    expect(tokens(btn)).toContain('sm:hidden');
    expect(btn.closest('header')).not.toBeNull();
  });

  it('no 排序 button in select mode (取消 takes that spot)', async () => {
    const user = userEvent.setup();
    render(<DownloadsBrowseV2 />);
    await user.click(screen.getByRole('button', { name: '選取' }));
    expect(screen.queryByTestId('downloads-sort-btn')).toBeNull();
    expect(screen.getByRole('button', { name: '取消' })).toBeInTheDocument();
  });

  it('no 排序 button when qBittorrent cannot be reached', () => {
    h.useDownloads.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('connection refused'),
      refetch: vi.fn(),
    });
    render(<DownloadsBrowseV2 />);
    expect(screen.queryByTestId('downloads-sort-btn')).toBeNull();
  });

  it('the native select stays in the DOM, hidden only below sm', () => {
    render(<DownloadsBrowseV2 />);
    const select = screen.getByRole('combobox', { name: '排序方式' });
    expect(tokens(select.closest('label')!)).toContain('max-sm:hidden');
  });

  it('排序 opens the sheet; picking 名稱（A–Z） sorts through the same handler and closes it', async () => {
    const user = userEvent.setup();
    render(<DownloadsBrowseV2 />);
    expect(screen.queryByTestId('download-sort-sheet')).toBeNull();

    await user.click(screen.getByTestId('downloads-sort-btn'));
    const sheet = await screen.findByTestId('download-sort-sheet');
    expect(screen.getByTestId('downloads-sort-btn')).toHaveAttribute('aria-expanded', 'true');
    expect(within(sheet).getByRole('radio', { name: '加入時間（新到舊）' })).toHaveAttribute(
      'aria-checked',
      'true'
    );

    await user.click(within(sheet).getByRole('radio', { name: '名稱（A–Z）' }));
    expect(h.useDownloads).toHaveBeenLastCalledWith('all', 'name', 'asc', 1, 100);
    await waitFor(() => expect(screen.queryByTestId('download-sort-sheet')).toBeNull());
    // …and the desktop select agrees: one state, two controls.
    expect(screen.getByRole('combobox', { name: '排序方式' })).toHaveValue('name:asc');
  });

  it('closing the sheet returns focus to 排序 even when the tap never focused it (WebKit)', async () => {
    // fireEvent.click does not move focus — like Safari, where tapping a button leaves
    // focus on <body>. Base UI's default would then "restore" focus to <body>.
    render(<DownloadsBrowseV2 />);
    const btn = screen.getByTestId('downloads-sort-btn');
    fireEvent.click(btn);
    const sheet = await screen.findByTestId('download-sort-sheet');
    await waitFor(() => expect(within(sheet).getAllByRole('radio')[0]).toHaveFocus());
    fireEvent.keyDown(document.activeElement!, { key: 'Escape' });
    await waitFor(() => expect(screen.queryByTestId('download-sort-sheet')).toBeNull());
    await waitFor(() => expect(btn).toHaveFocus());
  });

  it('if qBittorrent drops while the sheet is open, the sheet goes with the 排序 button', async () => {
    const user = userEvent.setup();
    const { rerender } = render(<DownloadsBrowseV2 />);
    await user.click(screen.getByTestId('downloads-sort-btn'));
    expect(await screen.findByTestId('download-sort-sheet')).toBeInTheDocument();

    h.useDownloads.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('connection refused'),
      refetch: vi.fn(),
    });
    rerender(<DownloadsBrowseV2 />);
    expect(screen.queryByTestId('downloads-sort-btn')).toBeNull();
    await waitFor(() => expect(screen.queryByTestId('download-sort-sheet')).toBeNull());
    // The button and the select are both gone — focus lands on the page heading, not <body>.
    await waitFor(() =>
      expect(screen.getByRole('heading', { level: 1, name: '下載' })).toHaveFocus()
    );
  });

  it('rotating past sm (phone → wider) closes a sheet left open', async () => {
    h.isPhone = true;
    const user = userEvent.setup();
    const { rerender } = render(<DownloadsBrowseV2 />);
    await user.click(screen.getByTestId('downloads-sort-btn'));
    expect(await screen.findByTestId('download-sort-sheet')).toBeInTheDocument();

    h.isPhone = false;
    rerender(<DownloadsBrowseV2 />);
    await waitFor(() => expect(screen.queryByTestId('download-sort-sheet')).toBeNull());
  });

  it('the sheet lists exactly what the select lists, in the same order — one SORT_OPTIONS', async () => {
    const user = userEvent.setup();
    render(<DownloadsBrowseV2 />);
    const selectLabels = within(screen.getByRole('combobox', { name: '排序方式' }))
      .getAllByRole('option')
      .map((o) => o.textContent);
    await user.click(screen.getByTestId('downloads-sort-btn'));
    const sheet = await screen.findByTestId('download-sort-sheet');
    expect(
      within(sheet)
        .getAllByRole('radio')
        .map((r) => r.textContent)
    ).toEqual(selectLabels);
    expect(selectLabels).toHaveLength(8);
  });

  it('the chip row scrolls sideways on a phone; each chip keeps its width', () => {
    render(<DownloadsBrowseV2 />);
    const row = screen.getByRole('tablist', { name: '下載狀態篩選' });
    expect(tokens(row)).toEqual(
      expect.arrayContaining([
        'flex-wrap',
        'max-sm:flex-nowrap',
        'max-sm:overflow-x-auto',
        'max-sm:-my-1',
        'max-sm:py-1',
        'max-sm:-mx-4',
        'max-sm:px-4',
        'max-sm:scroll-px-4',
        'max-sm:overscroll-x-contain',
      ])
    );
    for (const chip of within(row).getAllByRole('tab')) {
      expect(tokens(chip)).toContain('max-sm:shrink-0');
    }
  });
});

// h.remove is (hashes, deleteFiles) via `actions.remove.mutate({ hashes, deleteFiles })`.
describe('DownloadsBrowseV2 — phone card sheets: one overlay at a time (dsr-4b-2 D8-M / D9-M)', () => {
  const NAME_A = 'a.mkv';
  beforeEach(() => {
    h.isPhone = true;
  });

  async function openActions(user: ReturnType<typeof userEvent.setup>) {
    await user.click(screen.getByRole('button', { name: `更多動作：${NAME_A}` }));
    return screen.findByTestId('download-actions-sheet');
  }

  it('⋯ on a card opens the actions sheet, titled with that torrent; no menu anywhere', async () => {
    const user = userEvent.setup();
    render(<DownloadsBrowseV2 />);
    const sheet = await openActions(user);
    expect(screen.getByRole('dialog')).toHaveAccessibleName(NAME_A);
    expect(within(sheet).getByText('下載中 · 50.0%')).toBeInTheDocument();
    expect(screen.queryByRole('menu')).toBeNull();
  });

  it('詳細資訊 → the actions sheet goes, the detail sheet comes; ⋯ there goes back', async () => {
    const user = userEvent.setup();
    render(<DownloadsBrowseV2 />);
    const actions = await openActions(user);
    await user.click(within(actions).getByRole('button', { name: '詳細資訊' }));
    const detail = await screen.findByTestId('download-detail-sheet');
    await waitFor(() => expect(screen.queryByTestId('download-actions-sheet')).toBeNull());
    expect(within(detail).getByText('Hash')).toBeInTheDocument();

    await user.click(within(detail).getByRole('button', { name: `更多動作：${NAME_A}` }));
    await screen.findByTestId('download-actions-sheet');
    await waitFor(() => expect(screen.queryByTestId('download-detail-sheet')).toBeNull());
  });

  it('連同檔案刪除 → the sheet closes first, the confirm shows THAT torrent; 刪除檔案 removes with files', async () => {
    const user = userEvent.setup();
    render(<DownloadsBrowseV2 />);
    const actions = await openActions(user);
    await user.click(within(actions).getByRole('button', { name: '移除（連同檔案刪除）' }));
    const confirm = await screen.findByRole('dialog', { name: '移除並刪除檔案？' });
    await waitFor(() => expect(screen.queryByTestId('download-actions-sheet')).toBeNull());
    expect(within(confirm).getByText(/a\.mkv/)).toBeInTheDocument();
    await user.click(within(confirm).getByRole('button', { name: '刪除檔案' }));
    expect(h.remove).toHaveBeenCalledWith({ hashes: ['a'], deleteFiles: true });
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
  });

  it('移除（保留檔案）removes at once, no confirm, sheet closes', async () => {
    const user = userEvent.setup();
    render(<DownloadsBrowseV2 />);
    const actions = await openActions(user);
    await user.click(within(actions).getByRole('button', { name: '移除（保留檔案）' }));
    expect(h.remove).toHaveBeenCalledWith({ hashes: ['a'], deleteFiles: false });
    expect(screen.queryByRole('dialog', { name: '移除並刪除檔案？' })).toBeNull();
    await waitFor(() => expect(screen.queryByTestId('download-actions-sheet')).toBeNull());
  });

  it('the open sheet follows the list: a refetch that flips the status flips the pill', async () => {
    const user = userEvent.setup();
    const { rerender } = render(<DownloadsBrowseV2 />);
    const actions = await openActions(user);
    await user.click(within(actions).getByRole('button', { name: '詳細資訊' }));
    const detail = await screen.findByTestId('download-detail-sheet');
    expect(within(detail).getByTestId('download-status-a')).toHaveTextContent('下載中');

    h.useDownloads.mockReturnValue({
      data: { ...PAGE, items: [dl('a', 'paused'), PAGE.items[1]] },
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });
    rerender(<DownloadsBrowseV2 />);
    expect(within(detail).getByTestId('download-status-a')).toHaveTextContent('已暫停');
    expect(within(detail).getByRole('button', { name: `繼續 ${NAME_A}` })).toBeInTheDocument();
  });

  it('the item leaving the list closes its sheet; focus lands on the heading, not <body>', async () => {
    const user = userEvent.setup();
    const { rerender } = render(<DownloadsBrowseV2 />);
    await openActions(user);
    h.useDownloads.mockReturnValue({
      data: { ...PAGE, items: [PAGE.items[1]], totalItems: 1 },
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });
    rerender(<DownloadsBrowseV2 />);
    await waitFor(() => expect(screen.queryByTestId('download-actions-sheet')).toBeNull());
    await waitFor(() =>
      expect(screen.getByRole('heading', { level: 1, name: '下載' })).toHaveFocus()
    );
  });

  it('closing normally returns focus to the ⋯ that opened it', async () => {
    render(<DownloadsBrowseV2 />);
    const more = screen.getByRole('button', { name: `更多動作：${NAME_A}` });
    fireEvent.click(more); // no focus moves — Safari-style
    const sheet = await screen.findByTestId('download-actions-sheet');
    await waitFor(() => expect(within(sheet).getAllByRole('button')[0]).toHaveFocus());
    fireEvent.keyDown(document.activeElement!, { key: 'Escape' });
    await waitFor(() => expect(screen.queryByTestId('download-actions-sheet')).toBeNull());
    await waitFor(() => expect(more).toHaveFocus());
  });

  it('isPhone true → false closes any open card sheet', async () => {
    const user = userEvent.setup();
    const { rerender } = render(<DownloadsBrowseV2 />);
    await openActions(user);
    h.isPhone = false;
    rerender(<DownloadsBrowseV2 />);
    await waitFor(() => expect(screen.queryByTestId('download-actions-sheet')).toBeNull());
  });

  it('desktop (isPhone false) still gets the dropdown with no 詳細資訊 in it', async () => {
    h.isPhone = false;
    const user = userEvent.setup();
    render(<DownloadsBrowseV2 />);
    await user.click(screen.getByRole('button', { name: `更多動作：${NAME_A}` }));
    expect(await screen.findByRole('menu')).toBeInTheDocument();
    expect(screen.queryByRole('menuitem', { name: '詳細資訊' })).toBeNull();
    expect(screen.queryByTestId('download-actions-sheet')).toBeNull();
  });
});
