import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { Download, TorrentStatus } from '../../services/downloadService';

const h = vi.hoisted(() => ({
  navigate: vi.fn(),
  search: {} as Record<string, unknown>,
  useDownloads: vi.fn(),
  pause: vi.fn(),
  resume: vi.fn(),
  remove: vi.fn(),
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
  useDownloadCounts: () => ({
    data: { all: 2, downloading: 1, paused: 1, completed: 0, seeding: 0, error: 0 },
  }),
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
  useQBittorrentConfig: () => ({ data: { configured: true } }),
}));

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
