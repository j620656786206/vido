import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
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
});
