import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

// The state components use TanStack <Link>; stub it to a plain anchor so they render without a router.
vi.mock('@tanstack/react-router', () => ({
  Link: ({ to, children, ...props }: { to: string; children: React.ReactNode }) => (
    <a href={to} {...props}>
      {children}
    </a>
  ),
}));

import { DownloadsSkeletonV2, DownloadsEmptyV2, DownloadsQbtErrorV2 } from './DownloadsStatesV2';

describe('DownloadsSkeletonV2 (ux3-4-3 AC6)', () => {
  it('renders an aria-busy card-shaped skeleton', () => {
    render(<DownloadsSkeletonV2 />);
    expect(screen.getByTestId('downloads-skeleton-v2')).toHaveAttribute('aria-busy', 'true');
  });
});

describe('DownloadsEmptyV2 (ux3-4-3 AC6)', () => {
  it('the all filter shows a distinct no-downloads message + a 前往探索 affordance', () => {
    render(<DownloadsEmptyV2 filter="all" />);
    expect(screen.getByText('目前沒有下載任務')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '前往探索' })).toHaveAttribute('href', '/discover');
  });

  it('a non-all filter shows a switch-filter hint and no 前往探索', () => {
    render(<DownloadsEmptyV2 filter="downloading" />);
    expect(screen.getByText('沒有正在下載的任務')).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: '前往探索' })).toBeNull();
  });
});

describe('DownloadsQbtErrorV2 (ux3-4-3 AC6 — fail-soft)', () => {
  it('renders the unreachable card with 重試 + 前往設定 and fires onRetry', async () => {
    const onRetry = vi.fn();
    render(<DownloadsQbtErrorV2 onRetry={onRetry} />);

    expect(screen.getByRole('alert')).toHaveTextContent('無法連線到 qBittorrent');
    expect(screen.getByRole('link', { name: '前往設定' })).toHaveAttribute('href', '/settings');

    await userEvent.click(screen.getByRole('button', { name: '重試' }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('surfaces a backend message when provided', () => {
    render(<DownloadsQbtErrorV2 onRetry={vi.fn()} message="qBittorrent 認證失敗" />);
    expect(screen.getByText('qBittorrent 認證失敗')).toBeInTheDocument();
  });
});
