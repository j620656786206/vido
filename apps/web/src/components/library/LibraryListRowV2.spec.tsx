import React from 'react';
import { act, fireEvent, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router';
import { LibraryListRowV2, rowAriaLabel, ZH_UNMARKED_HINT } from './LibraryListRowV2';
import { filenameToGradient } from '../../utils/filenameToGradient';

vi.mock('../../utils/filenameToGradient', () => ({
  filenameToGradient: vi.fn(() => ['rgb(1, 2, 3)', 'rgb(4, 5, 6)']),
}));
const gradientSpy = vi.mocked(filenameToGradient);
import type { LibraryMovie } from '../../types/library';

const movie = (over: Partial<LibraryMovie> = {}): LibraryMovie => ({
  id: 'm1',
  title: '執法記錄',
  releaseDate: '2026-01-01',
  genres: ['恐怖'],
  parseStatus: 'success',
  chineseSubtitle: 'none',
  videoResolution: '3840x2160',
  videoCodec: 'H.265',
  audioCodec: 'DTS',
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
  ...over,
});

type RowProps = React.ComponentProps<typeof LibraryListRowV2>;

const base: RowProps = {
  id: 'm1',
  type: 'movie',
  title: '執法記錄',
  posterPath: null,
  meta: '2026 · 恐怖',
  voteAverage: 6,
  media: movie(),
};

function renderRow(over: Partial<RowProps> = {}) {
  const props = { ...base, ...over };
  const rootRoute = createRootRoute({
    component: () => (
      <ul>
        <LibraryListRowV2 {...props} />
      </ul>
    ),
  });
  const detail = createRoute({
    getParentRoute: () => rootRoute,
    path: '/media/$type/$id',
    component: () => <p>detail page</p>,
  });
  const router = createRouter({
    routeTree: rootRoute.addChildren([detail]),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  });
  render(<RouterProvider router={router} />);
  return router;
}

const FAILED_FILE = '[ItachiUchiha] Chainsaw Man – The Movie – Reze-hen.mkv';

describe('LibraryListRowV2', () => {
  it('links to the detail page and speaks the whole row in column order (AC #9)', async () => {
    renderRow();
    const link = await screen.findByTestId('list-row-v2-m1');
    expect(link).toHaveAttribute('href', '/media/movie/m1');
    expect(link).toHaveAttribute(
      'aria-label',
      '執法記錄，2026 · 恐怖，字幕：缺中文，評分 6.0，4K · H.265 · DTS'
    );
  });

  it.each([
    ['zh_hant', undefined, '繁中', 'success'],
    ['zh_hans', undefined, '簡中', 'info'],
    ['zh', undefined, '中文', 'success'],
    ['none', undefined, '缺中文', 'warning'],
    ['none', 'no_text_source', '無字幕源', 'bg-tertiary'],
    ['none', 'skipped', '已略過', 'bg-tertiary'],
    ['none', 'untranslated', '未翻譯', 'bg-tertiary'],
  ] as const)(
    'chineseSubtitle=%s subtitleStatus=%s → a %s pill (AC #5)',
    async (zh, status, label, tint) => {
      renderRow({ media: movie({ chineseSubtitle: zh, subtitleStatus: status }) });
      const cell = await screen.findByTestId('list-row-subtitle-m1');
      const pill = within(cell).getByText(label);
      expect(pill.className).toContain('rounded-full');
      expect(pill.className).toContain(tint);
    }
  );

  it.each([
    ['unknown verdict', { chineseSubtitle: 'unknown' as const }],
    ['no verdict at all', { chineseSubtitle: undefined }],
    [
      'still being worked on',
      { chineseSubtitle: 'unknown' as const, subtitleStatus: 'translating' },
    ],
  ])('writes a muted 不知道 (not a pill) when the state is %s', async (_, over) => {
    renderRow({ media: movie(over) });
    const cell = await screen.findByTestId('list-row-subtitle-m1');
    const text = within(cell).getByText('不知道');
    expect(text.className).toContain('text-[var(--text-muted)]');
    expect(text.className).not.toContain('rounded-full');
    expect(screen.getByTestId('list-row-v2-m1').getAttribute('aria-label')).toContain(
      '字幕：不知道'
    );
  });

  it('shows the 中文 hint in a tooltip and in the spoken name (AC #5, #9)', async () => {
    renderRow({ media: movie({ chineseSubtitle: 'zh' }) });
    const link = await screen.findByTestId('list-row-v2-m1');
    expect(link.getAttribute('aria-label')).toContain(`字幕：中文（${ZH_UNMARKED_HINT}）`);
    const badge = within(screen.getByTestId('list-row-subtitle-m1')).getByText('中文');
    // The badge is not a second focus stop inside the link.
    expect(badge).not.toHaveAttribute('tabindex');
    expect(screen.queryByText(ZH_UNMARKED_HINT)).not.toBeInTheDocument();
    await userEvent.hover(badge);
    expect(await screen.findByText(ZH_UNMARKED_HINT, {}, { timeout: 2000 })).toBeInTheDocument();
  });

  it('has no tooltip on the other badges', async () => {
    renderRow({ media: movie({ chineseSubtitle: 'zh_hant' }) });
    const badge = within(await screen.findByTestId('list-row-subtitle-m1')).getByText('繁中');
    await userEvent.hover(badge);
    await act(async () => {
      await new Promise((r) => setTimeout(r, 800));
    });
    expect(screen.queryByText(ZH_UNMARKED_HINT)).not.toBeInTheDocument();
  });

  it('shows a filled star and one decimal (AC #6)', async () => {
    renderRow({ voteAverage: 8.06 });
    const link = await screen.findByTestId('list-row-v2-m1');
    expect(link).toHaveTextContent('8.1');
    expect(link.querySelector('svg.lucide-star')?.getAttribute('class')).toContain(
      'fill-[var(--warning)]'
    );
  });

  // The 評分 column keeping its width without a rating is a layout fact — pinned by
  // tests/e2e/library-list.spec.ts (規格 starts at the same x on a row with no rating).
  it('leaves the rating out when there is none', async () => {
    renderRow({ voteAverage: 0 });
    const link = await screen.findByTestId('list-row-v2-m1');
    expect(link.querySelector('svg.lucide-star')).toBeNull();
    expect(link.getAttribute('aria-label')).toBe(
      '執法記錄，2026 · 恐怖，字幕：缺中文，4K · H.265 · DTS'
    );
  });

  it('writes the spec as one mono line with the tier, full text in title (AC #4)', async () => {
    renderRow({ media: movie({ videoResolution: '3840x2076', audioCodec: 'E-AC-3' }) });
    const spec = await screen.findByTestId('list-row-spec-m1');
    expect(spec).toHaveTextContent('4K · H.265 · E-AC-3');
    expect(spec).toHaveAttribute('title', '4K · H.265 · E-AC-3');
    expect(spec.className).toContain('font-mono');
    expect(spec.className).toContain('text-[var(--text-secondary)]');
    expect(spec.className).not.toContain('accent');
  });

  it('writes a muted — when the file told us no spec, and leaves it out of the spoken name', async () => {
    renderRow({
      media: movie({ videoResolution: undefined, videoCodec: undefined, audioCodec: undefined }),
    });
    const spec = await screen.findByTestId('list-row-spec-m1');
    expect(spec).toHaveTextContent('—');
    expect(spec).not.toHaveAttribute('title');
    expect(spec.className).toContain('text-[var(--text-muted)]');
    expect(screen.getByTestId('list-row-v2-m1').getAttribute('aria-label')).toBe(
      '執法記錄，2026 · 恐怖，字幕：缺中文，評分 6.0'
    );
  });

  it('draws the poster card gradient + initial when a matched title has no poster (AC #8)', async () => {
    renderRow({ title: '[字幕組] 牧師的最後誘惑' });
    const thumb = await screen.findByTestId('list-row-thumb-fallback');
    expect(thumb).toHaveTextContent('字');
    // jsdom's CSSOM drops any gradient background, so this proves the poster card's
    // hash is asked for this title; the visual baseline shows the drawn tile.
    expect(gradientSpy).toHaveBeenCalledWith('[字幕組] 牧師的最後誘惑');
    expect(thumb.className).toContain('text-[var(--text-on-scrim)]');
    expect(thumb).toHaveAttribute('aria-hidden', 'true');
  });

  it('shows the poster when there is one', async () => {
    renderRow({ posterPath: '/abc.jpg' });
    const link = await screen.findByTestId('list-row-v2-m1');
    expect(link.querySelector('img')?.getAttribute('src')).toContain('/abc.jpg');
    expect(screen.queryByTestId('list-row-thumb-fallback')).not.toBeInTheDocument();
  });

  describe('failed match (AC #7)', () => {
    const failed = (over: Partial<LibraryMovie> = {}) =>
      movie({
        parseStatus: 'failed',
        chineseSubtitle: 'unknown',
        videoResolution: undefined,
        videoCodec: undefined,
        audioCodec: undefined,
        filePath: `/media/movies/${FAILED_FILE}`,
        ...over,
      });

    it('says B10p’s sentence for a film, names the file and keeps one link', async () => {
      renderRow({
        title: '[ItachiUchiha] Chainsaw Man',
        voteAverage: undefined,
        posterPath: '/should-not-show.jpg',
        media: failed(),
      });
      const link = await screen.findByTestId('list-row-v2-m1');
      expect(link).toHaveTextContent('沒有找到這部電影的資料');
      expect(link).not.toHaveTextContent('[ItachiUchiha] Chainsaw Man[');
      // 失敗 pill: one copy per layout, CSS shows one.
      expect(within(link).getAllByText('失敗')).toHaveLength(2);
      expect(within(link).getAllByText('失敗')[0].className).toContain('error-tint');
      expect(link).toHaveAttribute(
        'aria-label',
        `沒有找到這部電影的資料，比對失敗，檔名 ${FAILED_FILE}，字幕：不知道`
      );
      expect(link.querySelectorAll('a, button')).toHaveLength(0);
      // Film strip thumb, never the poster.
      expect(link.querySelector('img')).toBeNull();
      expect(link.querySelector('svg.lucide-film')).not.toBeNull();
    });

    it('cuts the file name in the middle and keeps the last 12 characters', async () => {
      renderRow({ media: failed() });
      const name = await screen.findByTestId('list-row-filename');
      expect(name).toHaveAttribute('title', FAILED_FILE);
      const [head, tail] = Array.from(name.children);
      expect(head).toHaveTextContent('[ItachiUchiha] Chainsaw Man – The Movie –');
      expect(head.className).toContain('text-ellipsis');
      expect(head.className).toContain('whitespace-pre');
      expect(tail.textContent).toBe('Reze-hen.mkv');
      expect(tail.className).toContain('shrink-0');
      expect(name.className).toContain('font-mono');
      expect(name.className).toContain('text-[var(--text-muted)]');
    });

    it('says 影集 for a series and falls back to the title when there is no path', async () => {
      renderRow({
        type: 'tv',
        title: 'Some.Show.S01',
        media: failed({ filePath: undefined }),
      });
      const link = await screen.findByTestId('list-row-v2-m1');
      expect(link).toHaveTextContent('沒有找到這部影集的資料');
      expect(screen.getByTestId('list-row-filename')).toHaveAttribute('title', 'Some.Show.S01');
      expect(screen.getByTestId('list-row-filename').textContent).toBe('Some.Show.S01');
    });

    it('still shows the subtitle, rating and spec it does have', async () => {
      renderRow({
        voteAverage: 7.2,
        media: failed({ chineseSubtitle: 'zh_hant', audioCodec: 'AAC' }),
      });
      const link = await screen.findByTestId('list-row-v2-m1');
      expect(within(screen.getByTestId('list-row-subtitle-m1')).getByText('繁中')).toBeVisible();
      expect(screen.getByTestId('list-row-spec-m1')).toHaveTextContent('AAC');
      expect(link.getAttribute('aria-label')).toBe(
        `沒有找到這部電影的資料，比對失敗，檔名 ${FAILED_FILE}，字幕：繁中，評分 7.2，AAC`
      );
    });
  });

  it('shows 整理中 and says it; 已入庫 never shows (AC #11)', async () => {
    renderRow({ media: movie({ parseStatus: 'pending' }) });
    const link = await screen.findByTestId('list-row-v2-m1');
    expect(within(link).getAllByText('整理中')).toHaveLength(2);
    expect(link.getAttribute('aria-label')).toBe(
      '執法記錄，整理中，2026 · 恐怖，字幕：缺中文，評分 6.0，4K · H.265 · DTS'
    );
  });

  it('never shows 已入庫', async () => {
    renderRow();
    const link = await screen.findByTestId('list-row-v2-m1');
    expect(within(link).queryByText('已入庫')).not.toBeInTheDocument();
  });

  it('draws the focus ring inside the row, so the clipped frame cannot cut it (AC #9)', async () => {
    renderRow();
    const link = await screen.findByTestId('list-row-v2-m1');
    expect(link.className).toContain('focus-visible:outline-offset-[-2px]');
  });

  it('in selection mode toggles instead of navigating (AC #10)', async () => {
    const onSelect = vi.fn();
    const router = renderRow({ selectable: true, selected: true, onSelect });
    const link = await screen.findByTestId('list-row-v2-m1');
    expect(link).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByTestId('row-select-indicator-m1')).toHaveAttribute('aria-hidden', 'true');
    expect(link.className).toContain('bg-[var(--accent-subtle)]');
    fireEvent.click(link);
    expect(onSelect).toHaveBeenCalledTimes(1);
    expect(router.state.location.pathname).toBe('/');
  });

  it('a selected row keeps its tint under the mouse; only unselected rows get the hover tint', async () => {
    renderRow({ selectable: true, selected: true, onSelect: vi.fn() });
    const selected = await screen.findByTestId('list-row-v2-m1');
    expect(selected.className).not.toContain('hover:bg-');
  });

  it('has no aria-pressed outside selection mode', async () => {
    renderRow();
    expect(await screen.findByTestId('list-row-v2-m1')).not.toHaveAttribute('aria-pressed');
    expect(screen.queryByTestId('row-select-indicator-m1')).not.toBeInTheDocument();
  });
});

describe('rowAriaLabel', () => {
  it('skips an empty meta and keeps the order', () => {
    expect(
      rowAriaLabel({
        heading: '片',
        failed: false,
        fileName: 'x.mkv',
        pending: false,
        meta: '',
        subtitleLabel: '繁中',
        zhUnmarked: false,
        rating: null,
        spec: null,
      })
    ).toBe('片，字幕：繁中');
  });
});
