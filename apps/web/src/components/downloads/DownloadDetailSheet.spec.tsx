import { describe, it, expect, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { Download, TorrentStatus } from '../../services/downloadService';
import { DownloadDetailSheet } from './DownloadDetailSheet';

const NAME = 'Dune.Part.Two.2024.2160p.UHD.BluRay.mkv';
const HASH = '8f3ac2e1b4d59a7c6e0f21d8b93a45c7e1d6f0a2';
const PATH = '/volume1/media/movies/Dune Part Two (2024)/';
// downloaded: 0 with progress: 0.5 — the trap: a sheet that reads `downloaded` says「0 B」while
// the card next to it says 2.00 GB.
const make = (status: TorrentStatus): Download => ({
  hash: HASH,
  name: NAME,
  size: 4 * 1024 ** 3,
  progress: 0.5,
  downloadSpeed: 1_048_576,
  uploadSpeed: 1024,
  eta: 600,
  status,
  addedOn: '2026-06-30T13:14:00Z',
  seeds: 1,
  peers: 1,
  downloaded: 0,
  uploaded: 0,
  ratio: 0,
  savePath: PATH,
});

function setup(status: TorrentStatus) {
  const spies = {
    onOpenChange: vi.fn(),
    onPause: vi.fn(),
    onResume: vi.fn(),
    onOpenActions: vi.fn(),
  };
  render(<DownloadDetailSheet open download={make(status)} {...spies} />);
  return { user: userEvent.setup(), sheet: screen.getByTestId('download-detail-sheet'), ...spies };
}

describe('DownloadDetailSheet (dsr-4b-2 D9-M-v2)', () => {
  it('eight labelled cells, in the draft order, all read from the list item', () => {
    const { sheet } = setup('downloading');
    const dts = within(sheet)
      .getAllByRole('term')
      .map((t) => t.textContent);
    expect(dts).toEqual([
      '下載速度',
      '上傳速度',
      '進度',
      '剩餘時間',
      '來源',
      '加入時間',
      'Hash',
      '儲存路徑',
    ]);
    const dds = within(sheet)
      .getAllByRole('definition')
      .map((d) => d.textContent);
    expect(dds[0]).toBe('1.0 MB/s');
    expect(dds[1]).toBe('1.0 KB/s');
    expect(dds[0]).not.toMatch(/[↓↑]/);
    expect(dds[2]).toBe('2.00 GB / 4.00 GB');
    expect(dds[2]).not.toMatch(/^0 B/);
    expect(dds[3]).toBe('10m 0s');
    expect(dds[4]).toBe('qBittorrent');
    expect(dds[5]).toMatch(/^2026-0[67]-\d\d \d\d:\d\d$/);
    expect(dds[6]).toBe(HASH);
    expect(dds[7]).toBe(PATH);
  });

  it('title is the torrent name; the status pill and an accessible progressbar are there', () => {
    const { sheet } = setup('downloading');
    expect(screen.getByRole('dialog')).toHaveAccessibleName(NAME);
    expect(within(sheet).getByTestId(`download-status-${HASH}`)).toHaveTextContent('下載中');
    expect(within(sheet).getByRole('progressbar')).toHaveAttribute('aria-valuenow', '50');
    expect(within(sheet).getByText('50.0%')).toBeInTheDocument();
  });

  it('the primary button is the state action; pressing it does NOT close the sheet', async () => {
    const s = setup('downloading');
    await s.user.click(within(s.sheet).getByRole('button', { name: `暫停 ${NAME}` }));
    expect(s.onPause).toHaveBeenCalledWith(HASH);
    expect(s.onOpenChange).not.toHaveBeenCalled();
  });

  it.each([
    ['paused', '繼續', 'onResume'],
    ['error', '重試', 'onResume'],
  ] as const)('%s → primary is %s', async (status, label, handler) => {
    const s = setup(status);
    await s.user.click(within(s.sheet).getByRole('button', { name: `${label} ${NAME}` }));
    expect(s[handler]).toHaveBeenCalledWith(HASH);
  });

  it('⋯ hands off to the actions sheet', async () => {
    const s = setup('downloading');
    await s.user.click(within(s.sheet).getByRole('button', { name: `更多動作：${NAME}` }));
    expect(s.onOpenActions).toHaveBeenCalledTimes(1);
    expect(s.onOpenChange).not.toHaveBeenCalled();
  });

  it('a completed torrent has only a full-width 更多動作 — no primary button', async () => {
    const s = setup('completed');
    const buttons = within(s.sheet).getAllByRole('button');
    expect(buttons).toHaveLength(1);
    expect(buttons[0]).toHaveAccessibleName(`更多動作：${NAME}`);
    expect(buttons[0]).toHaveTextContent('更多動作');
    await s.user.click(buttons[0]);
    expect(s.onOpenActions).toHaveBeenCalledTimes(1);
  });

  it('hash and path can break anywhere so a 40-char hash never widens the sheet', () => {
    const { sheet } = setup('downloading');
    const dds = within(sheet).getAllByRole('definition');
    expect(dds[6].className.split(/\s+/)).toEqual(
      expect.arrayContaining(['break-all', 'font-mono'])
    );
    expect(dds[7].className.split(/\s+/)).toContain('break-all');
  });
});
