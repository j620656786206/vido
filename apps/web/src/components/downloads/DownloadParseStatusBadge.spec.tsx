import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DownloadParseStatusBadge } from './DownloadParseStatusBadge';

describe('DownloadParseStatusBadge', () => {
  it('renders nothing when parseStatus is undefined', () => {
    const { container } = render(<DownloadParseStatusBadge />);
    expect(container.firstChild).toBeNull();
  });

  it('renders "解析中..." for pending status', () => {
    render(<DownloadParseStatusBadge parseStatus={{ status: 'pending' }} />);
    expect(screen.getByRole('status')).toBeInTheDocument();
    expect(screen.getByText('解析中...')).toBeInTheDocument();
  });

  it('renders "解析中..." for processing status with spinner', () => {
    render(<DownloadParseStatusBadge parseStatus={{ status: 'processing' }} />);
    const badge = screen.getByRole('status');
    expect(badge).toBeInTheDocument();
    expect(screen.getByText('解析中...')).toBeInTheDocument();
    expect(badge.getAttribute('data-status')).toBe('processing');
  });

  it('renders "已入庫" for completed status with mediaId', () => {
    render(
      <DownloadParseStatusBadge parseStatus={{ status: 'completed', mediaId: 'media-123' }} />
    );
    expect(screen.getByText('已入庫')).toBeInTheDocument();
    expect(screen.getByRole('status').getAttribute('data-status')).toBe('completed');
  });

  it('renders "已解析" for completed status without mediaId', () => {
    render(<DownloadParseStatusBadge parseStatus={{ status: 'completed' }} />);
    expect(screen.getByText('已解析')).toBeInTheDocument();
  });

  it('renders "解析失敗" for failed status', () => {
    render(
      <DownloadParseStatusBadge
        parseStatus={{ status: 'failed', errorMessage: 'could not parse' }}
      />
    );
    expect(screen.getByText('解析失敗')).toBeInTheDocument();
    expect(screen.getByRole('status').getAttribute('data-status')).toBe('failed');
  });

  it('renders "已跳過" for skipped status', () => {
    render(<DownloadParseStatusBadge parseStatus={{ status: 'skipped' }} />);
    expect(screen.getByText('已跳過')).toBeInTheDocument();
  });

  it('has correct accessibility attributes', () => {
    render(<DownloadParseStatusBadge parseStatus={{ status: 'processing' }} />);
    const badge = screen.getByRole('status');
    expect(badge.getAttribute('aria-label')).toContain('解析');
  });

  it('applies custom className', () => {
    render(
      <DownloadParseStatusBadge
        parseStatus={{ status: 'completed', mediaId: 'media-1' }}
        className="custom-class"
      />
    );
    expect(screen.getByRole('status').className).toContain('custom-class');
  });
});
