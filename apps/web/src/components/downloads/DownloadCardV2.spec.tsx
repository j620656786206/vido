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

describe('DownloadCardV2 (ux3-4-3 AC2 · dsr-4 D1-D-v2)', () => {
  it('renders the status token pill with the v2 label + accent tint', () => {
    render(<DownloadCardV2 download={base} />);
    const pill = screen.getByTestId('download-status-abc123');
    expect(pill).toHaveTextContent('下載中');
    expect(pill.className).toContain('bg-[var(--accent-tint)]');
    expect(pill.className).toContain('text-[var(--accent-text)]');
  });

  it('a paused torrent wears the neutral pill — pausing is what the user asked for, not 赭', () => {
    render(<DownloadCardV2 download={{ ...base, status: 'paused' }} />);
    const pill = screen.getByTestId('download-status-abc123');
    expect(pill.className).toContain('bg-[var(--bg-tertiary)]');
    expect(pill.className).not.toContain('warning');
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

  it('a downloading torrent shows ↓ / ↑ / ETA / done-of-total, all Mono', () => {
    render(<DownloadCardV2 download={base} />);
    const meta = screen.getByTestId('download-meta-abc123');
    expect(meta).toHaveClass('font-mono');
    expect(meta).toHaveTextContent('↓ 1.4 MB/s');
    expect(meta).toHaveTextContent('↑ 781.3 KB/s');
    expect(meta).toHaveTextContent('1h 0m');
    expect(meta.textContent).toMatch(/GB \/ [\d.]+ GB/);
  });

  it('speeds are neutral text, never the 已完成 green (disc-2026-08-download-speed-wears-completed-green)', () => {
    render(<DownloadCardV2 download={base} />);
    const meta = screen.getByTestId('download-meta-abc123');
    expect(meta).toHaveClass('text-[var(--text-secondary)]');
    expect(meta.innerHTML).not.toContain('success');
  });

  it('a completed torrent shows 已完成, dashes where a number does not apply, and the plain size', () => {
    render(
      <DownloadCardV2
        download={{ ...base, status: 'completed', progress: 1, downloadSpeed: 0, uploadSpeed: 0 }}
      />
    );
    expect(screen.getByTestId('download-status-abc123')).toHaveTextContent('已完成');
    const meta = screen.getByTestId('download-meta-abc123');
    expect(meta).toHaveTextContent('↓ —');
    expect(meta).toHaveTextContent('↑ —');
    expect(meta).toHaveTextContent('4.66 GB');
    expect(meta.textContent).not.toContain('/');
    expect(screen.getByText('100.0%')).toBeInTheDocument();
  });

  it('shows the qBittorrent source chip and an inert NZBGet slot', () => {
    render(<DownloadCardV2 download={base} />);
    expect(screen.getByText('qBittorrent')).toBeInTheDocument();
    expect(screen.getByText('NZBGet').closest('[aria-disabled]')).toHaveAttribute(
      'aria-disabled',
      'true'
    );
  });

  it('renders no action cluster when no handlers are provided (4-3a display-only usage)', () => {
    render(<DownloadCardV2 download={base} />);
    expect(screen.queryByRole('button', { name: /暫停/ })).toBeNull();
    expect(screen.queryByRole('button', { name: /更多動作/ })).toBeNull();
  });
});

describe('DownloadCardV2 — actions + selection (ux3-4-3b AC3/AC5)', () => {
  it('wires the shared row actions — 暫停 calls onPause(hash)', async () => {
    const onPause = vi.fn();
    render(<DownloadCardV2 download={base} onPause={onPause} onResume={vi.fn()} />);
    await userEvent.click(screen.getByRole('button', { name: /^暫停/ }));
    expect(onPause).toHaveBeenCalledWith('abc123');
  });

  it('select mode renders a checkbox that toggles selection', async () => {
    const onSelectChange = vi.fn();
    render(
      <DownloadCardV2 download={base} selectable selected={false} onSelectChange={onSelectChange} />
    );
    await userEvent.click(screen.getByRole('checkbox', { name: /選取/ }));
    expect(onSelectChange).toHaveBeenCalledWith('abc123', true);
  });

  it('a selected card gets the gold outline', () => {
    render(<DownloadCardV2 download={base} selectable selected onSelectChange={vi.fn()} />);
    expect(screen.getByTestId('download-card-v2-abc123')).toHaveClass(
      'border-[var(--accent-primary)]'
    );
  });
});
