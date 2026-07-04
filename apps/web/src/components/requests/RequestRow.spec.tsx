import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
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

  it('tv rows read 影集', () => {
    render(<RequestRow request={row({ mediaType: 'tv', title: '熊家餐館 S3' })} />);
    expect(screen.getByText('影集')).toBeInTheDocument();
  });

  it('failed rows surface error_message', () => {
    render(<RequestRow request={row({ status: 'failed', errorMessage: '找不到種子' })} />);
    expect(screen.getByText('找不到種子')).toBeInTheDocument();
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
});
