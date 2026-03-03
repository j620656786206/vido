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
    expect(screen.getByText('[SubGroup] Movie Name (2024) [1080p]')).toBeInTheDocument();
  });

  it('renders progress bar', () => {
    render(<DownloadItem download={mockDownload} expanded={false} onToggleExpand={() => {}} />);
    const progressBar = screen.getByRole('progressbar');
    expect(progressBar).toBeInTheDocument();
    expect(progressBar.getAttribute('aria-valuenow')).toBe('85');
  });

  it('renders progress percentage', () => {
    render(<DownloadItem download={mockDownload} expanded={false} onToggleExpand={() => {}} />);
    expect(screen.getByText('85.0%')).toBeInTheDocument();
  });

  it('renders download speed for downloading status', () => {
    render(<DownloadItem download={mockDownload} expanded={false} onToggleExpand={() => {}} />);
    expect(screen.getByText(/10\.0 MB\/s/)).toBeInTheDocument();
  });

  it('renders upload speed for seeding status', () => {
    const seedingDownload = { ...mockDownload, status: 'seeding' as const, uploadSpeed: 1048576 };
    render(<DownloadItem download={seedingDownload} expanded={false} onToggleExpand={() => {}} />);
    expect(screen.getByText(/1\.0 MB\/s/)).toBeInTheDocument();
  });

  it('renders completed text for completed status', () => {
    const completedDownload = { ...mockDownload, status: 'completed' as const, progress: 1 };
    render(
      <DownloadItem download={completedDownload} expanded={false} onToggleExpand={() => {}} />
    );
    expect(screen.getByText('完成')).toBeInTheDocument();
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
    expect(screen.getByText('下載中')).toBeInTheDocument();
  });

  it('renders parse status badge when parseStatus is present', () => {
    const downloadWithParse: Download = {
      ...mockDownload,
      status: 'completed',
      progress: 1,
      parseStatus: { status: 'completed', mediaId: 'media-123' },
    };
    render(
      <DownloadItem download={downloadWithParse} expanded={false} onToggleExpand={() => {}} />
    );
    expect(screen.getByTestId('download-parse-status-badge')).toBeInTheDocument();
    expect(screen.getByText('已入庫')).toBeInTheDocument();
  });

  it('does not render parse status badge when parseStatus is absent', () => {
    render(<DownloadItem download={mockDownload} expanded={false} onToggleExpand={() => {}} />);
    expect(screen.queryByTestId('download-parse-status-badge')).toBeNull();
  });

  it('shows parse status badge for failed parsing', () => {
    const downloadWithFailed: Download = {
      ...mockDownload,
      status: 'completed',
      progress: 1,
      parseStatus: { status: 'failed', errorMessage: 'parse error' },
    };
    render(
      <DownloadItem download={downloadWithFailed} expanded={false} onToggleExpand={() => {}} />
    );
    expect(screen.getByText('解析失敗')).toBeInTheDocument();
  });
});
