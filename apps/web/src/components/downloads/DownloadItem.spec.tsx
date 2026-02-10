import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DownloadItem } from './DownloadItem';
import type { Download } from '../../services/downloadService';

const mockDownload: Download = {
  hash: 'abc123def456',
  name: '[SubGroup] Movie Name (2024) [1080p]',
  size: 4294967296,
  progress: 0.85,
  downloadSpeed: 10485760,
  uploadSpeed: 524288,
  eta: 600,
  status: 'downloading',
  addedOn: '2026-01-15T10:00:00Z',
  seeds: 10,
  peers: 5,
  downloaded: 3650722201,
  uploaded: 104857600,
  ratio: 0.03,
  savePath: '/downloads/movies',
};

describe('DownloadItem', () => {
  it('renders torrent name', () => {
    render(<DownloadItem download={mockDownload} expanded={false} onToggleExpand={() => {}} />);
    expect(screen.getByText('[SubGroup] Movie Name (2024) [1080p]')).toBeTruthy();
  });

  it('renders progress bar', () => {
    render(<DownloadItem download={mockDownload} expanded={false} onToggleExpand={() => {}} />);
    const progressBar = screen.getByRole('progressbar');
    expect(progressBar).toBeTruthy();
    expect(progressBar.getAttribute('aria-valuenow')).toBe('85');
  });

  it('renders progress percentage', () => {
    render(<DownloadItem download={mockDownload} expanded={false} onToggleExpand={() => {}} />);
    expect(screen.getByText('85.0%')).toBeTruthy();
  });

  it('renders download speed for downloading status', () => {
    render(<DownloadItem download={mockDownload} expanded={false} onToggleExpand={() => {}} />);
    expect(screen.getByText(/10\.0 MB\/s/)).toBeTruthy();
  });

  it('renders upload speed for seeding status', () => {
    const seedingDownload = { ...mockDownload, status: 'seeding' as const, uploadSpeed: 1048576 };
    render(<DownloadItem download={seedingDownload} expanded={false} onToggleExpand={() => {}} />);
    expect(screen.getByText(/1\.0 MB\/s/)).toBeTruthy();
  });

  it('renders completed text for completed status', () => {
    const completedDownload = { ...mockDownload, status: 'completed' as const, progress: 1 };
    render(
      <DownloadItem download={completedDownload} expanded={false} onToggleExpand={() => {}} />
    );
    expect(screen.getByText('完成')).toBeTruthy();
  });

  it('calls onToggleExpand when clicked', async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    render(<DownloadItem download={mockDownload} expanded={false} onToggleExpand={onToggle} />);

    await user.click(screen.getByRole('button'));
    expect(onToggle).toHaveBeenCalledTimes(1);
  });

  it('sets aria-expanded true when expanded', () => {
    render(<DownloadItem download={mockDownload} expanded={true} onToggleExpand={() => {}} />);
    expect(screen.getByRole('button').getAttribute('aria-expanded')).toBe('true');
  });

  it('sets aria-expanded false when collapsed', () => {
    render(<DownloadItem download={mockDownload} expanded={false} onToggleExpand={() => {}} />);
    expect(screen.getByRole('button').getAttribute('aria-expanded')).toBe('false');
  });

  it('renders status icon', () => {
    render(<DownloadItem download={mockDownload} expanded={false} onToggleExpand={() => {}} />);
    expect(screen.getByText('下載中')).toBeTruthy();
  });
});
