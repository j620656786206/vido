import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { RequestRow } from './RequestRow';
import type { MediaRequest, RequestStatus } from '../../services/requestService';

const row = (
  over: Partial<MediaRequest> & { progress?: number } = {}
): MediaRequest & {
  progress?: number;
} => ({
  id: 'r1',
  tmdbId: 550,
  mediaType: 'movie',
  title: '沙丘：第二部',
  status: 'pending',
  fulfilmentSource: null,
  externalId: null,
  seasons: null,
  episodes: null,
  errorMessage: null,
  requestedAt: '2026-06-28T10:00:00Z',
  updatedAt: '2026-06-28T10:00:00Z',
  ...over,
});

describe('RequestRow', () => {
  it('renders title, type, and the Mono date (design L1 metaRow)', () => {
    render(<RequestRow request={row()} />);
    expect(screen.getByText('沙丘：第二部')).toBeInTheDocument();
    expect(screen.getByText('電影')).toBeInTheDocument();
    expect(screen.getByText('2026-06-28')).toBeInTheDocument();
  });

  it.each<[RequestStatus, string]>([
    ['pending', '想要'],
    ['searching', '搜尋中'],
    ['downloading', '下載中'],
    ['completed', '已入庫'],
    ['failed', '失敗'],
  ])('maps status %s through the DL-v2 §2.5 token map → %s', (status, label) => {
    render(<RequestRow request={row({ status })} />);
    expect(screen.getByTestId(`request-status-${status}`)).toHaveTextContent(label);
  });

  it('[bugfix-h] shows the local calendar day, not the UTC one', () => {
    // Pin the zone so this fails on a UTC CI runner too: 23:30Z is already
    // the 29th in Taipei, while slicing the text would show the 28th.
    const prevTZ = process.env.TZ;
    process.env.TZ = 'Asia/Taipei';
    try {
      render(<RequestRow request={row({ requestedAt: '2026-06-28T23:30:00Z' })} />);
      expect(screen.getByText('2026-06-29')).toBeInTheDocument();
    } finally {
      process.env.TZ = prevTZ;
    }
  });

  it('tv rows read 影集', () => {
    render(<RequestRow request={row({ mediaType: 'tv', title: '熊家餐館 S3' })} />);
    expect(screen.getByText('影集')).toBeInTheDocument();
  });

  it('failed rows surface error_message', () => {
    render(<RequestRow request={row({ status: 'failed', errorMessage: '找不到種子' })} />);
    // Rendered for both layouts (13-7b): under the title below md, in the
    // action cluster from md up — CSS shows exactly one.
    expect(screen.getAllByText('找不到種子').length).toBeGreaterThan(0);
  });

  it('Mono progress % renders only when downloading AND progress exists (13-3b slot)', () => {
    const { rerender } = render(
      <RequestRow request={row({ status: 'downloading', progress: 0.45 })} />
    );
    expect(screen.getByText('45%')).toBeInTheDocument();

    rerender(<RequestRow request={row({ status: 'downloading' })} />);
    expect(screen.queryByText(/%$/)).not.toBeInTheDocument();

    rerender(<RequestRow request={row({ status: 'pending', progress: 0.45 })} />);
    expect(screen.queryByText('45%')).not.toBeInTheDocument();
  });

  it('exposes the live % as a Mono progressbar (13-3b AC #4 a11y)', () => {
    render(<RequestRow request={row({ status: 'downloading', progress: 0.45 })} />);
    const bar = screen.getByRole('progressbar');
    expect(bar).toHaveTextContent('45%');
    expect(bar).toHaveAttribute('aria-valuenow', '45');
    expect(bar).toHaveAttribute('aria-valuemin', '0');
    expect(bar).toHaveAttribute('aria-valuemax', '100');
    // Mono, tabular, text-xs per AC #4 (DownloadCardV2 progressbar convention).
    expect(bar.className).toContain('font-mono');
    expect(bar.className).toContain('tabular-nums');
    expect(bar.className).toContain('text-xs');
  });

  it('the status pill announces politely on async transitions (aria-live)', () => {
    render(<RequestRow request={row({ status: 'searching' })} />);
    const pill = screen.getByTestId('request-status-searching');
    expect(pill).toHaveAttribute('role', 'status');
    expect(pill).toHaveAttribute('aria-live', 'polite');
  });

  // ⚖️ Alexyu 2026-09-11（dsr-11）：搜尋中改成泥金。赭色＝「你要求了，但它沒發生」，
  // 而搜尋中正在發生。改完後 searching 與 downloading 同為泥金——兩者都是「正在跑」，
  // 區別交給標籤與百分比，不靠顏色。七月 13-0 的「暫態處理中家族」被這條取代。
  it('[dsr-11] searching wears 泥金, not 赭', () => {
    render(<RequestRow request={row({ status: 'searching' })} />);
    const pill = screen.getByTestId('request-status-searching');
    expect(pill.className).toContain('--accent-tint');
    expect(pill.className).not.toContain('--warning');
  });

  describe('13-7b action-area (design matrix: 取消 on pending, 重試 on failed)', () => {
    const statuses: RequestStatus[] = [
      'pending',
      'searching',
      'downloading',
      'completed',
      'failed',
    ];

    it.each(statuses)('%s shows exactly its drawn action', (status) => {
      render(<RequestRow request={row({ status })} onCancel={vi.fn()} onRetry={vi.fn()} />);
      expect(!!screen.queryByTestId('request-cancel-btn')).toBe(status === 'pending');
      expect(!!screen.queryByTestId('request-retry-btn')).toBe(status === 'failed');
    });

    it('no handler → no button (static/gallery uses stay inert)', () => {
      render(<RequestRow request={row({ status: 'pending' })} />);
      expect(screen.queryByTestId('request-cancel-btn')).toBeNull();
    });

    it('clicks fire the handlers', async () => {
      const onCancel = vi.fn();
      const onRetry = vi.fn();
      const { unmount } = render(<RequestRow request={row()} onCancel={onCancel} />);
      await userEvent.click(screen.getByRole('button', { name: /取消請求/ }));
      expect(onCancel).toHaveBeenCalledOnce();
      unmount();

      render(<RequestRow request={row({ status: 'failed' })} onRetry={onRetry} />);
      await userEvent.click(screen.getByRole('button', { name: /重試請求/ }));
      expect(onRetry).toHaveBeenCalledOnce();
    });

    it('busy disables the button (no double-fire)', () => {
      render(<RequestRow request={row()} onCancel={vi.fn()} busy />);
      expect(screen.getByTestId('request-cancel-btn')).toBeDisabled();
    });

    it('the failed caption sits beside 重試 on md+ and under the title below md', () => {
      render(
        <RequestRow
          request={row({ status: 'failed', errorMessage: '找不到可用來源' })}
          onRetry={vi.fn()}
        />
      );
      const captions = screen.getAllByText('找不到可用來源');
      expect(captions).toHaveLength(2);
      expect(screen.getByTestId('request-fail-caption')).toHaveClass('hidden', 'md:inline');
      expect(captions.find((c) => c.tagName === 'P')).toHaveClass('md:hidden');
    });
  });

  describe('13-2b range label', () => {
    it('a partial request shows its seasons / episodes in the meta row', () => {
      render(
        <RequestRow
          request={row({ mediaType: 'tv', seasons: '[1,2]', episodes: '{"3":[1,2,3]}' })}
        />
      );
      expect(screen.getByTestId('request-range')).toHaveTextContent('第 1、2 季 · 第 3 季 3 集');
    });

    it('a whole-title request shows no range', () => {
      render(<RequestRow request={row({ mediaType: 'tv' })} />);
      expect(screen.queryByTestId('request-range')).toBeNull();
    });
  });
});
