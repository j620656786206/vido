import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DownloadRowActions } from './DownloadRowActions';
import type { Download, TorrentStatus } from '../../services/downloadService';

// The hook, not matchMedia: `false` is what the global test-setup stub yields, so every existing
// test below runs exactly as before; the phone block flips it.
const h = vi.hoisted(() => ({ isPhone: false }));
vi.mock('../../hooks/useIsPhone', () => ({ useIsPhone: () => h.isPhone }));

const NAME = 'Test.Movie.2024.mkv';

const make = (status: TorrentStatus): Download => ({
  hash: 'h1',
  name: NAME,
  size: 5_000_000_000,
  progress: 0.42,
  downloadSpeed: 1_500_000,
  uploadSpeed: 0,
  eta: 3600,
  status,
  addedOn: '2026-07-01T00:00:00Z',
  seeds: 1,
  peers: 1,
  downloaded: 0,
  uploaded: 0,
  ratio: 0,
  savePath: '/dl',
});

function setup(status: TorrentStatus) {
  const spies = { onPause: vi.fn(), onResume: vi.fn(), onRemove: vi.fn() };
  const user = userEvent.setup();
  render(<DownloadRowActions download={make(status)} {...spies} />);
  return { user, ...spies };
}

async function openMenu(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole('button', { name: `更多動作：${NAME}` }));
}

describe('DownloadRowActions — the state button (dsr-4 D1-D-v2)', () => {
  it.each([
    ['downloading', '暫停', 'onPause'],
    ['seeding', '暫停', 'onPause'],
    ['stalled', '暫停', 'onPause'],
    ['queued', '暫停', 'onPause'],
    ['checking', '暫停', 'onPause'],
    ['paused', '繼續', 'onResume'],
    ['error', '重試', 'onResume'],
  ] as const)('%s → %s', async (status, label, handler) => {
    const s = setup(status);
    await s.user.click(screen.getByRole('button', { name: `${label} ${NAME}` }));
    expect(s[handler]).toHaveBeenCalledWith('h1');
  });

  it('a completed torrent has no state button — only the ⋯ menu', () => {
    setup('completed');
    expect(screen.queryByRole('button', { name: /^(暫停|繼續|重試) / })).toBeNull();
    expect(screen.getByRole('button', { name: `更多動作：${NAME}` })).toBeInTheDocument();
  });

  it('renders nothing when no handlers are provided', () => {
    const { container } = render(<DownloadRowActions download={make('downloading')} />);
    expect(container).toBeEmptyDOMElement();
  });
});

describe('DownloadRowActions — the ⋯ menu (dsr-4 D3-D-v2)', () => {
  it('repeats the state action inside the menu', async () => {
    const s = setup('paused');
    await openMenu(s.user);
    await s.user.click(await screen.findByRole('menuitem', { name: '繼續' }));
    expect(s.onResume).toHaveBeenCalledWith('h1');
  });

  it('移除（保留檔案）removes right away — the files stay, so there is nothing to confirm', async () => {
    const s = setup('downloading');
    await openMenu(s.user);
    await s.user.click(await screen.findByRole('menuitem', { name: '移除（保留檔案）' }));
    expect(s.onRemove).toHaveBeenCalledWith('h1', false);
    expect(screen.queryByRole('dialog')).toBeNull();
  });

  it('移除（連同檔案刪除）asks first; 刪除檔案 removes the torrent with its files', async () => {
    const s = setup('downloading');
    await openMenu(s.user);
    await s.user.click(await screen.findByRole('menuitem', { name: '移除（連同檔案刪除）' }));

    const dialog = await screen.findByRole('dialog', { name: '移除並刪除檔案？' });
    expect(dialog).toHaveTextContent(NAME);
    expect(s.onRemove).not.toHaveBeenCalled();

    await s.user.click(screen.getByRole('button', { name: '刪除檔案' }));
    expect(s.onRemove).toHaveBeenCalledWith('h1', true);
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
  });

  it('Esc closes the confirm without removing anything', async () => {
    const s = setup('downloading');
    await openMenu(s.user);
    await s.user.click(await screen.findByRole('menuitem', { name: '移除（連同檔案刪除）' }));
    await screen.findByRole('dialog');
    await s.user.keyboard('{Escape}');
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(s.onRemove).not.toHaveBeenCalled();
  });

  it('closing the confirm hands focus back to ⋯, not to the top of the page', async () => {
    const s = setup('downloading');
    await openMenu(s.user);
    await s.user.click(await screen.findByRole('menuitem', { name: '移除（連同檔案刪除）' }));
    await s.user.click(await screen.findByRole('button', { name: '取消' }));
    await waitFor(() =>
      expect(screen.getByRole('button', { name: `更多動作：${NAME}` })).toHaveFocus()
    );
  });

  it('a torrent whose size is still unknown (magnet metadata) shows no「0 B」in the confirm', async () => {
    const onRemove = vi.fn();
    const user = userEvent.setup();
    render(
      <DownloadRowActions download={{ ...make('downloading'), size: 0 }} onRemove={onRemove} />
    );
    await user.click(screen.getByRole('button', { name: `更多動作：${NAME}` }));
    await user.click(await screen.findByRole('menuitem', { name: '移除（連同檔案刪除）' }));
    const dialog = await screen.findByRole('dialog');
    expect(dialog).toHaveTextContent(NAME);
    expect(dialog.textContent).not.toContain('0 B');
  });

  it('取消 in the confirm removes nothing', async () => {
    const s = setup('completed');
    await openMenu(s.user);
    await s.user.click(await screen.findByRole('menuitem', { name: '移除（連同檔案刪除）' }));
    await s.user.click(await screen.findByRole('button', { name: '取消' }));
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(s.onRemove).not.toHaveBeenCalled();
  });
});

describe('DownloadRowActions — phone ⋯ reports upward (dsr-4b-2 D8-M-v2)', () => {
  beforeEach(() => {
    h.isPhone = true;
  });
  afterEach(() => {
    h.isPhone = false;
  });

  it('with onOpenActions the ⋯ is a plain button that reports (hash, itself) — no menu', async () => {
    const onOpenActions = vi.fn();
    const user = userEvent.setup();
    render(
      <DownloadRowActions
        download={make('downloading')}
        onPause={vi.fn()}
        onRemove={vi.fn()}
        onOpenActions={onOpenActions}
      />
    );
    const more = screen.getByRole('button', { name: `更多動作：${NAME}` });
    expect(more).toHaveAttribute('aria-haspopup', 'dialog');
    await user.click(more);
    expect(onOpenActions).toHaveBeenCalledWith('h1', more);
    expect(screen.queryByRole('menu')).toBeNull();
    expect(screen.queryByRole('menuitem')).toBeNull();
  });

  it('without onOpenActions a phone still gets the dropdown — it works at any width', async () => {
    const s = setup('downloading');
    await openMenu(s.user);
    expect(await screen.findByRole('menuitem', { name: '移除（保留檔案）' })).toBeInTheDocument();
  });

  it('the table keeps the dropdown even when onOpenActions is passed', async () => {
    const user = userEvent.setup();
    render(
      <DownloadRowActions
        download={make('downloading')}
        variant="table"
        onPause={vi.fn()}
        onRemove={vi.fn()}
        onOpenActions={vi.fn()}
      />
    );
    await user.click(screen.getByRole('button', { name: `更多動作：${NAME}` }));
    expect(await screen.findByRole('menuitem', { name: '移除（保留檔案）' })).toBeInTheDocument();
  });
});
