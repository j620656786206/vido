import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import type { DownloadImportStatus } from '../../services/downloadService';

// TanStack <Link> → a plain anchor with the params filled in, so the chip renders without a router.
vi.mock('@tanstack/react-router', () => ({
  Link: ({
    to,
    params,
    children,
    ...props
  }: {
    to: string;
    params: Record<string, string>;
    children: React.ReactNode;
  }) => (
    <a href={to.replace('$type', params.type).replace('$id', params.id)} {...props}>
      {children}
    </a>
  ),
}));

import { ImportStatusChip, describeImportStatus } from './ImportStatusChip';

const movie = (over: Partial<DownloadImportStatus>): DownloadImportStatus => ({
  state: 'in_library',
  source: 'radarr',
  mediaType: 'movie',
  ...over,
});

describe('describeImportStatus (dl-import-2 · D12-D-v2)', () => {
  it.each([
    [movie({ state: 'in_library', mediaId: 'm1' }), '已入庫', '已入庫', 'success'],
    [movie({ state: 'awaiting_scan' }), 'Radarr 已匯入 · Vido 還沒有', '已匯入', 'info'],
    [movie({ state: 'awaiting_import' }), '等 Radarr 匯入', '待匯入', 'neutral'],
    [movie({ state: 'import_failed' }), 'Radarr 匯入失敗', '匯入失敗', 'warning'],
    [movie({ state: 'import_ignored' }), '已略過匯入', '已略過', 'neutral'],
  ])('%#: %o', (status, label, short, tone) => {
    const copy = describeImportStatus(status);
    expect(copy?.label).toBe(label);
    expect(copy?.short).toBe(short);
    expect(copy?.tone).toBe(tone);
  });

  it('a partly scanned season pack counts episodes', () => {
    const copy = describeImportStatus({
      state: 'awaiting_scan',
      source: 'sonarr',
      mediaType: 'tv',
      mediaId: 's1',
      episodesImported: 9,
      episodesInLibrary: 6,
    });
    expect(copy?.label).toBe('Vido 有 6/9 集');
    expect(copy?.short).toBe('6/9 集');
  });

  it('a season pack Vido has none of yet does not say「Vido 有 0/9 集」', () => {
    const copy = describeImportStatus({
      state: 'awaiting_scan',
      source: 'sonarr',
      mediaType: 'tv',
      episodesImported: 9,
      episodesInLibrary: 0,
    });
    expect(copy?.label).toBe('Sonarr 已匯入 · Vido 還沒有');
  });

  it('a state this build does not know says nothing', () => {
    expect(
      describeImportStatus(movie({ state: 'future_state' as DownloadImportStatus['state'] }))
    ).toBeNull();
  });
});

describe('ImportStatusChip', () => {
  it('links to the detail page when Vido has the title', () => {
    render(<ImportStatusChip status={movie({ state: 'in_library', mediaId: 'm1' })} />);
    // label-in-name (WCAG 2.5.3): the accessible name starts with the visible words
    const link = screen.getByRole('link', {
      name: /^已入庫：Radarr 已匯入，Vido 媒體庫裡有，查看詳情$/,
    });
    expect(link).toHaveAttribute('href', '/media/movie/m1');
    expect(link).toHaveTextContent('已入庫');
    expect(link.className).toContain('bg-[var(--success-tint)]');
  });

  it('a partly scanned season pack links to the series', () => {
    render(
      <ImportStatusChip
        status={{
          state: 'awaiting_scan',
          source: 'sonarr',
          mediaType: 'tv',
          mediaId: 's1',
          episodesImported: 9,
          episodesInLibrary: 6,
        }}
      />
    );
    expect(screen.getByRole('link')).toHaveAttribute('href', '/media/tv/s1');
  });

  it('「Vido 還沒有」is never a link, even when Vido has the series', () => {
    render(
      <ImportStatusChip
        status={{
          state: 'awaiting_scan',
          source: 'sonarr',
          mediaType: 'tv',
          mediaId: 's1',
          episodesImported: 9,
          episodesInLibrary: 0,
        }}
      />
    );
    expect(screen.queryByRole('link')).toBeNull();
    expect(screen.getByTestId('download-import-status')).toHaveTextContent(
      'Sonarr 已匯入 · Vido 還沒有'
    );
  });

  it('an unexpected media type is not linked to a page that does not exist', () => {
    render(
      <ImportStatusChip
        status={movie({ mediaType: 'music' as DownloadImportStatus['mediaType'], mediaId: 'x' })}
      />
    );
    expect(screen.queryByRole('link')).toBeNull();
  });

  it('a long card label truncates instead of running past a narrow card', () => {
    render(<ImportStatusChip status={movie({ state: 'awaiting_scan' })} />);
    const chip = screen.getByTestId('download-import-status');
    expect(chip).toHaveClass('max-w-full', 'min-w-0');
    expect(chip).not.toHaveClass('shrink-0');
    expect(screen.getByText('Radarr 已匯入 · Vido 還沒有')).toHaveClass('truncate');
  });

  it('is not a link when there is nothing in Vido to open', () => {
    render(<ImportStatusChip status={movie({ state: 'awaiting_import' })} />);
    expect(screen.queryByRole('link')).toBeNull();
    expect(screen.getByTestId('download-import-status')).toHaveTextContent('等 Radarr 匯入');
  });

  it('the table variant is short, and still tells a screen reader the whole sentence', () => {
    render(<ImportStatusChip status={movie({ state: 'import_failed' })} variant="table" />);
    const chip = screen.getByTestId('download-import-status');
    expect(chip).toHaveTextContent('匯入失敗');
    // the sentence is in a real sr-only node, not aria-hidden text
    const sentence = screen.getByText('：Radarr 把這個下載標記為失敗，要到 Radarr 看原因');
    expect(sentence).toHaveClass('sr-only');
    expect(sentence.closest('[aria-hidden="true"]')).toBeNull();
    expect(chip.className).toContain('text-[var(--warning-text)]');
  });

  it('renders nothing for an unknown state', () => {
    const { container } = render(
      <ImportStatusChip
        status={movie({ state: 'future_state' as DownloadImportStatus['state'] })}
      />
    );
    expect(container).toBeEmptyDOMElement();
  });
});
