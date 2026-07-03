import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DownloadCardV2 } from './DownloadCardV2';
import type { Download } from '../../services/downloadService';

const base: Download = {
  hash: 'abc123',
  name: '測試電影 Test.Movie.2024.1080p.BluRay',
  size: 5_000_000_000,
  progress: 0.42,
  downloadSpeed: 1_500_000,
  uploadSpeed: 800_000,
  eta: 3600,
  status: 'downloading',
  addedOn: '2026-07-01T00:00:00Z',
  seeds: 5,
  peers: 2,
  downloaded: 2_100_000_000,
  uploaded: 0,
  ratio: 0,
  savePath: '/downloads',
};

describe('DownloadCardV2 (ux3-4-3 AC2)', () => {
  it('renders the status token pill with the v2 label + accent tint', () => {
    render(<DownloadCardV2 download={base} />);
    const pill = screen.getByTestId('download-status-abc123');
    expect(pill).toHaveTextContent('下載中');
    expect(pill.className).toContain('bg-[var(--accent-tint)]');
    expect(pill.className).toContain('text-[var(--accent-text)]');
  });

  it('renders the progress percent as a Mono numeric + an accessible progressbar', () => {
    render(<DownloadCardV2 download={base} />);
    const pct = screen.getByText('42.0%');
    expect(pct).toHaveClass('font-mono');
    expect(pct).toHaveClass('tabular-nums');

    const bar = screen.getByRole('progressbar');
    expect(bar).toHaveAttribute('aria-valuenow', '42');
    expect(bar).toHaveAttribute('aria-valuemin', '0');
    expect(bar).toHaveAttribute('aria-valuemax', '100');
  });

  it('a downloading item shows ↓ speed + ETA as Mono numerics', () => {
    render(<DownloadCardV2 download={base} />);
    const speed = screen.getByText(/↓/);
    expect(speed).toHaveClass('font-mono');
    expect(speed.textContent).toContain('MB/s');
    expect(screen.getByText(/ETA/)).toBeInTheDocument();
  });

  it('a completed item shows the 已完成 token and no download-speed row', () => {
    render(<DownloadCardV2 download={{ ...base, status: 'completed', progress: 1 }} />);
    expect(screen.getByTestId('download-status-abc123')).toHaveTextContent('已完成');
    expect(screen.queryByText(/↓/)).toBeNull();
    expect(screen.getByText('100.0%')).toBeInTheDocument();
  });

  it('renders no action cluster when no handlers are provided (4-3a display-only usage)', () => {
    render(<DownloadCardV2 download={base} />);
    expect(screen.queryByRole('button', { name: /暫停/ })).toBeNull();
    expect(screen.queryByRole('button', { name: /移除/ })).toBeNull();
  });
});

describe('DownloadCardV2 — actions + selection (ux3-4-3b AC3/AC5)', () => {
  it('a downloading item shows 暫停 → calls onPause(hash)', async () => {
    const onPause = vi.fn();
    render(<DownloadCardV2 download={base} onPause={onPause} onResume={vi.fn()} />);
    await userEvent.click(screen.getByRole('button', { name: /暫停/ }));
    expect(onPause).toHaveBeenCalledWith('abc123');
  });

  it('a paused item shows 繼續 → calls onResume(hash)', async () => {
    const onResume = vi.fn();
    render(
      <DownloadCardV2
        download={{ ...base, status: 'paused' }}
        onPause={vi.fn()}
        onResume={onResume}
      />
    );
    await userEvent.click(screen.getByRole('button', { name: /繼續/ }));
    expect(onResume).toHaveBeenCalledWith('abc123');
  });

  it('the remove button opens a confirm dialog; 連同檔案刪除 calls onRemove(hash, true)', async () => {
    const onRemove = vi.fn();
    render(<DownloadCardV2 download={base} onRemove={onRemove} />);

    await userEvent.click(screen.getByRole('button', { name: /移除/ }));
    // dialog opened (Radix → aria-modal dialog, focus-trapped)
    expect(await screen.findByRole('dialog')).toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: '移除（連同檔案刪除）' }));
    expect(onRemove).toHaveBeenCalledWith('abc123', true);
  });

  it('保留檔案 calls onRemove(hash, false)', async () => {
    const onRemove = vi.fn();
    render(<DownloadCardV2 download={base} onRemove={onRemove} />);
    await userEvent.click(screen.getByRole('button', { name: /移除/ }));
    await userEvent.click(await screen.findByRole('button', { name: '移除（保留檔案）' }));
    expect(onRemove).toHaveBeenCalledWith('abc123', false);
  });

  it('select mode renders a checkbox that toggles selection', async () => {
    const onSelectChange = vi.fn();
    render(
      <DownloadCardV2 download={base} selectable selected={false} onSelectChange={onSelectChange} />
    );
    await userEvent.click(screen.getByRole('checkbox', { name: /選取/ }));
    expect(onSelectChange).toHaveBeenCalledWith('abc123', true);
  });
});
