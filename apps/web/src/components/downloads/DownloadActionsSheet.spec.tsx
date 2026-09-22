import { describe, it, expect, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { Download, TorrentStatus } from '../../services/downloadService';
import { DownloadActionsSheet } from './DownloadActionsSheet';

const NAME = 'Dune.Part.Two.2024.2160p.UHD.BluRay.mkv';
const make = (status: TorrentStatus, progress = 0.624): Download => ({
  hash: 'h1',
  name: NAME,
  size: 8_100_000_000,
  progress,
  downloadSpeed: 1_500_000,
  uploadSpeed: 100_000,
  eta: 3600,
  status,
  addedOn: '2026-06-30T13:14:00Z',
  seeds: 1,
  peers: 1,
  downloaded: 0,
  uploaded: 0,
  ratio: 0,
  savePath: '/dl',
});

function setup(status: TorrentStatus, opts: { details?: boolean } = { details: true }) {
  const spies = {
    onOpenChange: vi.fn(),
    onPause: vi.fn(),
    onResume: vi.fn(),
    onRemove: vi.fn(),
    onShowDetails: opts.details ? vi.fn() : undefined,
    onRequestDeleteWithFiles: vi.fn(),
  };
  render(<DownloadActionsSheet open download={make(status)} {...spies} />);
  return { user: userEvent.setup(), sheet: screen.getByTestId('download-actions-sheet'), ...spies };
}

const rows = (sheet: HTMLElement) =>
  within(sheet)
    .getAllByRole('button')
    .map((b) => b.textContent);

describe('DownloadActionsSheet (dsr-4b-2 D8-M-v2)', () => {
  it('is titled with the torrent name and shows「status · percent」under it', () => {
    const { sheet } = setup('downloading');
    expect(screen.getByRole('dialog')).toHaveAccessibleName(NAME);
    expect(within(sheet).getByText('下載中 · 62.4%')).toBeInTheDocument();
  });

  it.each([
    ['paused', '繼續'],
    ['error', '重試'],
    ['stalled', '暫停'],
    ['queued', '暫停'],
  ] as const)('%s → the state row reads %s', (status, label) => {
    const { sheet } = setup(status);
    expect(rows(sheet)).toEqual(['詳細資訊', label, '移除（保留檔案）', '移除（連同檔案刪除）']);
  });

  it('a completed torrent has no state row', () => {
    const { sheet } = setup('completed');
    expect(rows(sheet)).toEqual(['詳細資訊', '移除（保留檔案）', '移除（連同檔案刪除）']);
  });

  it('no onShowDetails → no 詳細資訊 row', () => {
    const { sheet } = setup('downloading', { details: false });
    expect(rows(sheet)).toEqual(['暫停', '移除（保留檔案）', '移除（連同檔案刪除）']);
  });

  it('the state row runs its handler and closes', async () => {
    const s = setup('paused');
    await s.user.click(within(s.sheet).getByRole('button', { name: '繼續' }));
    expect(s.onResume).toHaveBeenCalledWith('h1');
    expect(s.onOpenChange).toHaveBeenCalledWith(false);
  });

  it('移除（保留檔案）removes without asking and closes', async () => {
    const s = setup('downloading');
    await s.user.click(within(s.sheet).getByRole('button', { name: '移除（保留檔案）' }));
    expect(s.onRemove).toHaveBeenCalledWith('h1', false);
    expect(s.onOpenChange).toHaveBeenCalledWith(false);
  });

  it('移除（連同檔案刪除）only asks the owner to confirm — it removes nothing itself and does not close', async () => {
    const s = setup('downloading');
    await s.user.click(within(s.sheet).getByRole('button', { name: '移除（連同檔案刪除）' }));
    expect(s.onRequestDeleteWithFiles).toHaveBeenCalledTimes(1);
    expect(s.onRemove).not.toHaveBeenCalled();
    expect(s.onOpenChange).not.toHaveBeenCalled();
  });

  it('詳細資訊 is a handoff too — the owner closes this sheet', async () => {
    const s = setup('downloading');
    await s.user.click(within(s.sheet).getByRole('button', { name: '詳細資訊' }));
    expect(s.onShowDetails).toHaveBeenCalledTimes(1);
    expect(s.onOpenChange).not.toHaveBeenCalled();
  });

  it('every row is at least 52px tall and the delete row wears the error text', () => {
    const { sheet } = setup('downloading');
    for (const b of within(sheet).getAllByRole('button')) {
      expect(b.className.split(/\s+/)).toContain('min-h-[52px]');
    }
    expect(
      within(sheet).getByRole('button', { name: '移除（連同檔案刪除）' }).className.split(/\s+/)
    ).toContain('text-[var(--error-text)]');
  });
});
