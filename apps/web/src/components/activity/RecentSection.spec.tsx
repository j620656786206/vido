import { describe, it, expect, vi } from 'vitest';
import { render, screen, within, fireEvent } from '@testing-library/react';
import { RecentSection } from './RecentSection';
import type { RecentEvent } from '../../services/activityService';

const NOW = Date.parse('2026-10-19T12:00:00Z');
const minsAgo = (m: number) => new Date(NOW - m * 60_000).toISOString();

/** K1-D-v2 (kMeWS): 字幕完成 · 解析失敗 · 字幕沒做成, newest first. */
const EVENTS: RecentEvent[] = [
  {
    kind: 'subtitle',
    result: 'completed',
    at: minsAgo(2),
    mediaType: 'episode',
    mediaTitle: '末日光明',
    seasonNumber: 1,
    episodeNumber: 2,
    route: 'translate',
    cueCount: 568,
    durationSeconds: 180,
    spentUsd: 0.27,
  },
  { kind: 'parse', result: 'failed', detail: 'The.Movie.2024.mkv', at: minsAgo(60) },
  {
    kind: 'subtitle',
    result: 'failed',
    at: minsAgo(180),
    mediaType: 'episode',
    mediaTitle: '末日光明',
    seasonNumber: 1,
    episodeNumber: 1,
    route: 'translate',
    durationSeconds: 4,
    spentUsd: 0,
    failureReason: 'target_not_writable',
  },
];

function renderSection(events: RecentEvent[] = EVENTS) {
  return render(
    <RecentSection section={{ status: 'ok', events }} onRetry={() => undefined} now={NOW} />
  );
}

describe('RecentSection (disc-2026-10-activity-subtitle-run-time-b)', () => {
  it('mixed parse + subtitle rows get their own titles, in the backend order', () => {
    renderSection();
    const rows = screen.getAllByTestId('activity-recent-row');
    expect(rows).toHaveLength(3);
    expect(rows.map((r) => r.getAttribute('data-kind'))).toEqual(['subtitle', 'parse', 'subtitle']);
    expect(within(rows[0]).getByRole('heading')).toHaveTextContent('字幕完成');
    expect(rows[0]).toHaveTextContent('末日光明 S01E02 · 翻譯 568 句');
    expect(within(rows[1]).getByRole('heading')).toHaveTextContent('解析失敗');
    expect(rows[1]).toHaveTextContent('The.Movie.2024.mkv');
    expect(within(rows[2]).getByRole('heading')).toHaveTextContent('字幕沒做成');
    expect(rows[2]).toHaveTextContent('末日光明 S01E01 · 影片所在的資料夾寫不進字幕');
  });

  it('each subtitle row has exactly one meta node; the parse row has none', () => {
    renderSection();
    const [done, parse, failed] = screen.getAllByTestId('activity-recent-row');
    expect(within(done).getAllByTestId('activity-recent-meta')).toHaveLength(1);
    expect(within(done).getByTestId('activity-recent-meta')).toHaveTextContent('3 分鐘 · US$0.27');
    expect(within(failed).getAllByTestId('activity-recent-meta')).toHaveLength(1);
    expect(within(failed).getByTestId('activity-recent-meta')).toHaveTextContent('4 秒 · 沒花錢');
    expect(within(parse).queryByTestId('activity-recent-meta')).toBeNull();
    // The meta text is in the DOM once — not a desktop copy + a phone copy.
    expect(screen.getAllByText('3 分鐘 · US$0.27')).toHaveLength(1);
  });

  it('the meta is read as 「花了…」 by screen readers; the visible text is unchanged (1A)', () => {
    renderSection();
    const [done, parse, failed] = screen.getAllByTestId('activity-recent-row');
    const doneMeta = within(done).getByTestId('activity-recent-meta');
    const failedMeta = within(failed).getByTestId('activity-recent-meta');
    expect(doneMeta.textContent).toBe('花了3 分鐘 · US$0.27');
    expect(failedMeta.textContent).toBe('花了4 秒 · 沒花錢');
    // Screen-reader only: hidden by sr-only, a child of the meta node.
    const lead = within(doneMeta).getByText('花了');
    expect(lead).toHaveClass('sr-only');
    expect(lead.parentElement).toBe(doneMeta);
    expect(within(failedMeta).getByText('花了')).toHaveClass('sr-only');
    // Inside the meta, before the amount and before the time — not on the time.
    const rowText = done.textContent ?? '';
    expect(rowText.indexOf('花了')).toBeGreaterThanOrEqual(0);
    expect(rowText.indexOf('花了')).toBeLessThan(rowText.indexOf('3 分鐘'));
    expect(rowText.indexOf('3 分鐘')).toBeLessThan(rowText.indexOf('2 分鐘前'));
    const time = within(done).getByTestId('activity-recent-time');
    expect(within(time).queryByText('花了')).toBeNull();
    expect(time.textContent).toBe('2 分鐘前');
    // Once per subtitle row — no second copy for the other breakpoint.
    expect(within(done).getAllByText('花了')).toHaveLength(1);
    // A parse row never gets it.
    expect(within(parse).queryByText('花了')).toBeNull();
  });

  it('only 「沒花錢」 to say → no 「花了」 (「花了沒花錢」 is not Chinese)', () => {
    renderSection([{ ...EVENTS[2], durationSeconds: undefined, spentUsd: 0 }]);
    const meta = screen.getByTestId('activity-recent-meta');
    expect(meta.textContent).toBe('沒花錢');
    expect(within(meta).queryByText('花了')).toBeNull();
    expect(meta.querySelector('.sr-only')).toBeNull();
  });

  it('a fixed `now` pins every relative time', () => {
    renderSection();
    expect(screen.getAllByTestId('activity-recent-time').map((t) => t.textContent)).toEqual([
      '2 分鐘前',
      '1 小時前',
      '3 小時前',
    ]);
  });

  it('a subtitle row with neither duration nor spend renders no meta node', () => {
    renderSection([{ ...EVENTS[0], durationSeconds: undefined, spentUsd: undefined }]);
    expect(screen.queryByTestId('activity-recent-meta')).toBeNull();
    expect(screen.getByTestId('activity-recent-time')).toHaveTextContent('2 分鐘前');
  });

  it('a subtitle row never shows the raw `detail` even if one arrives', () => {
    renderSection([
      {
        ...EVENTS[2],
        detail: 'place: placer: write failed: open /media/tv/Peacemaker: permission denied',
      },
    ]);
    expect(screen.queryByText(/permission denied/)).toBeNull();
  });

  it('an unknown kind keeps the parse row shape (one time line, no meta)', () => {
    renderSection([{ kind: 'scan', result: 'completed', detail: 'x.mkv', at: minsAgo(5) }]);
    const row = screen.getByTestId('activity-recent-row');
    expect(row).toHaveTextContent('解析完成');
    expect(row.getAttribute('data-kind')).toBe('scan');
    expect(screen.queryByTestId('activity-recent-meta')).toBeNull();
  });

  it('the subtitle description may wrap on a phone; the parse one stays truncated', () => {
    renderSection();
    const [done, parse] = screen.getAllByTestId('activity-recent-row');
    const detailOf = (row: HTMLElement, text: string) =>
      within(row).getByText(text, { exact: false });
    expect(detailOf(done, '翻譯 568 句').className).toContain('sm:truncate');
    expect(detailOf(done, '翻譯 568 句').className).not.toMatch(/(^| )truncate( |$)/);
    expect(detailOf(parse, 'The.Movie.2024.mkv').className).toMatch(/(^| )truncate( |$)/);
  });

  it('no events → nothing; unavailable → the inline error with retry', () => {
    const { container, rerender } = renderSection([]);
    expect(container).toBeEmptyDOMElement();
    const onRetry = vi.fn();
    rerender(<RecentSection section={{ status: 'unavailable', events: [] }} onRetry={onRetry} />);
    const err = screen.getByTestId('activity-recent-error');
    fireEvent.click(within(err).getByRole('button'));
    expect(onRetry).toHaveBeenCalled();
  });

  it('no arbitrary px font sizes (ActivityHub.spec rule)', () => {
    const { container } = renderSection();
    const offScale = Array.from(container.querySelectorAll('*')).filter((el) =>
      /text-\[\d+px\]/.test((el as HTMLElement).className)
    );
    expect(offScale).toEqual([]);
  });
});

describe('ActivityRow — rows without meta are unchanged (AC #7)', () => {
  it('a parse row keeps the one-line flex layout and truncating detail', () => {
    renderSection([EVENTS[1]]);
    const row = screen.getByTestId('activity-recent-row');
    const line = row.firstElementChild as HTMLElement;
    expect(line.className).toBe('flex items-center gap-3.5');
    expect(within(row).getByText('The.Movie.2024.mkv').className).toBe(
      'truncate text-xs text-[var(--text-secondary)]'
    );
    expect(within(row).getByTestId('activity-recent-time').parentElement?.className).toBe(
      'shrink-0 text-right'
    );
  });

  it('a subtitle row with no meta still wraps its description on a phone', () => {
    renderSection([{ ...EVENTS[2], durationSeconds: undefined, spentUsd: undefined }]);
    const detail = screen.getByText(/影片所在的資料夾寫不進字幕/);
    expect(detail.className).toContain('break-words');
    expect(detail.className).toContain('sm:truncate');
  });
});
