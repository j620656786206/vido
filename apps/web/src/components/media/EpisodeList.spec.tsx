import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { EpisodeList } from './EpisodeList';
import type { MergedEpisode } from '../../types/library';

const episodes: MergedEpisode[] = [
  {
    episodeNumber: 1,
    name: '第一集',
    airDate: '2024-01-05',
    runtime: 24,
    hasLocalFile: true,
    episodeId: 'ep-uuid-1',
    subtitleStatus: 'found',
    subtitleLanguage: 'zh-Hant',
    filePath: '/m/S01E01.mkv',
  },
  {
    episodeNumber: 2,
    name: '第二集',
    airDate: '2024-01-12',
    runtime: 24,
    hasLocalFile: true,
    episodeId: 'ep-uuid-2',
    filePath: '/m/S01E02.mkv',
    subtitleStatus: 'not_found',
  },
  {
    episodeNumber: 3,
    name: '第三集',
    hasLocalFile: false, // no local file → no subtitle indicator (AC #6)
  },
];

describe('EpisodeList', () => {
  it('renders an SxxExx code, title, air date and runtime per episode', () => {
    render(<EpisodeList episodes={episodes} seasonNumber={1} />);

    expect(screen.getByText('S01E01')).toBeInTheDocument();
    expect(screen.getByText('第一集')).toBeInTheDocument();
    expect(screen.getByText('2024-01-05')).toBeInTheDocument();
    // dsr-2 AC #9: 分, not 分鐘 — B4p-D writes 「24 分」, and the movie meta row says 「107 分」.
    expect(screen.getAllByText('24 分').length).toBeGreaterThan(0);
    expect(screen.queryByText(/分鐘/)).not.toBeInTheDocument();
    expect(screen.getByText('S01E03')).toBeInTheDocument();
  });

  it('shows a subtitle status indicator only for episodes with a local file (AC #6)', () => {
    render(<EpisodeList episodes={episodes} seasonNumber={1} />);

    // ep1 found + ep2 not_found each carry an indicator button; ep3 has none.
    const indicators = screen.getAllByTestId('episode-subtitle-indicator');
    expect(indicators).toHaveLength(2);
    expect(screen.getByLabelText('已找到字幕')).toBeInTheDocument();
    expect(screen.getByLabelText('找不到字幕')).toBeInTheDocument();
  });

  it('renders the loading skeleton when isLoading', () => {
    render(<EpisodeList episodes={[]} seasonNumber={1} isLoading />);
    expect(screen.getByTestId('episode-list-skeleton')).toBeInTheDocument();
    expect(screen.queryByTestId('episode-list')).not.toBeInTheDocument();
  });

  it('renders a retry-able error state when isError (AC #7)', () => {
    const onRetry = vi.fn();
    render(<EpisodeList episodes={[]} seasonNumber={1} isError onRetry={onRetry} />);

    const errorBox = screen.getByTestId('episode-list-error');
    expect(errorBox).toBeInTheDocument();
    expect(screen.getByRole('alert')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '重試' }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('renders an empty message when there are no episodes', () => {
    render(<EpisodeList episodes={[]} seasonNumber={1} />);
    expect(screen.getByText('此季沒有劇集資料。')).toBeInTheDocument();
  });
});

// ─── Story sub-1-7b — the 5 pipeline states (spec: flow-j-specs/j2-d) ───

const ep = (subtitleStatus: string, episodeNumber = 1): MergedEpisode => ({
  episodeNumber,
  name: `第 ${episodeNumber} 集`,
  hasLocalFile: true,
  subtitleStatus,
  filePath: `/m/S01E0${episodeNumber}.mkv`,
});

describe('EpisodeList — subtitle-pipeline status icons (sub-1-7b AC #3)', () => {
  it.each([
    ['probing', '偵測字幕軌中'],
    ['extracting', '抽取內嵌字幕中'],
    ['translating', '翻譯字幕中'],
    ['no_text_source', '無可用的文字字幕軌'],
    ['skipped', '已略過（字幕軌語言不符）'],
    // 10th value (sub-2-2b CR M1): unreachable for episodes until 9R-10a, added
    // ahead so a settled verdict never falls back to 尚未搜尋字幕's bare Minus.
    ['untranslated', '已生成英文字幕，尚未翻譯'],
  ])('renders %s with the long-form accessible name "%s"', (subtitleStatus, label) => {
    render(<EpisodeList episodes={[ep(subtitleStatus)]} seasonNumber={1} />);
    // The icon carries no visible text, so the accessible name is where the full
    // explanation lives — this is where "已略過 must not read as broken" is solved.
    expect(screen.getByRole('button', { name: label })).toBeInTheDocument();
  });

  it('spins for the three in-flight states and NOT for the terminal ones', () => {
    const inFlight = ['probing', 'extracting', 'translating'];
    const terminal = ['no_text_source', 'skipped'];

    for (const status of inFlight) {
      const { unmount } = render(<EpisodeList episodes={[ep(status)]} seasonNumber={1} />);
      expect(screen.getByTestId('episode-subtitle-indicator').querySelector('svg')).toHaveClass(
        'animate-spin'
      );
      unmount();
    }
    for (const status of terminal) {
      const { unmount } = render(<EpisodeList episodes={[ep(status)]} seasonNumber={1} />);
      expect(screen.getByTestId('episode-subtitle-indicator').querySelector('svg')).not.toHaveClass(
        'animate-spin'
      );
      unmount();
    }
  });

  it('tints in-flight states with accent, terminal verdicts with muted (sub-1-7a AC #3)', () => {
    const { unmount } = render(<EpisodeList episodes={[ep('translating')]} seasonNumber={1} />);
    // accent is RESERVED for in-progress (Sally 2026-07-05).
    expect(screen.getByTestId('episode-subtitle-indicator')).toHaveClass(
      'text-[var(--accent-text)]'
    );
    unmount();

    render(<EpisodeList episodes={[ep('no_text_source')]} seasonNumber={1} />);
    expect(screen.getByTestId('episode-subtitle-indicator')).toHaveClass(
      'text-[var(--text-muted)]'
    );
  });

  it('re-tints the pre-existing searching state to accent (sub-1-7a AC #5 ruling)', () => {
    render(<EpisodeList episodes={[ep('searching')]} seasonNumber={1} />);
    const icon = screen.getByTestId('episode-subtitle-indicator');
    // Was --warning; two colours for one meaning next to the three new spinners
    // read as a distinction that does not exist.
    expect(icon).toHaveClass('text-[var(--accent-text)]');
    expect(icon).not.toHaveClass('text-[var(--warning)]');
    expect(icon.querySelector('svg')).toHaveClass('animate-spin');
  });

  it('still short-circuits on !hasLocalFile before any status lookup', () => {
    render(
      <EpisodeList
        episodes={[
          { episodeNumber: 1, name: '第 1 集', hasLocalFile: false, subtitleStatus: 'skipped' },
        ]}
        seasonNumber={1}
      />
    );
    expect(screen.queryByTestId('episode-subtitle-indicator')).not.toBeInTheDocument();
  });

  it('falls back to not_searched for an unrecognised value (belt and braces)', () => {
    render(<EpisodeList episodes={[ep('some_future_state')]} seasonNumber={1} />);
    expect(screen.getByRole('button', { name: '尚未搜尋字幕' })).toBeInTheDocument();
  });
});

/**
 * The guard that was missing when sub-1-7b first shipped: `skipped` and
 * `not_searched` rendered pixel-identically (both `Minus` + `--text-muted`) and
 * nothing failed, because every test asserted each state in isolation. A human
 * had to notice. These assert the states AGAINST EACH OTHER instead.
 *
 * Icon grammar (J2-D, Sally 2026-08-04): a CIRCLED glyph means the pipeline has
 * a settled answer for this file; a BARE glyph means it does not yet.
 */
describe('EpisodeList — icon grammar: settled verdicts vs not-yet (J2-D)', () => {
  const glyphOf = (subtitleStatus: string): string => {
    const { unmount } = render(<EpisodeList episodes={[ep(subtitleStatus)]} seasonNumber={1} />);
    const svg = screen.getByTestId('episode-subtitle-indicator').querySelector('svg');
    // lucide stamps `lucide-<kebab-icon-name>` alongside the base `lucide` class.
    const glyph = [...(svg?.classList ?? [])].find((c) => c.startsWith('lucide-')) ?? '';
    unmount();
    return glyph;
  };

  it('skipped and not_searched do NOT share a glyph', () => {
    // The exact regression that shipped: same glyph, same tint, told apart only
    // by the accessible name — invisible to anyone scanning the list.
    expect(glyphOf('skipped')).not.toBe(glyphOf('not_searched'));
  });

  it('settled verdicts use a circled glyph; not-yet states use a bare one', () => {
    expect(glyphOf('found')).toBe('lucide-circle-check');
    expect(glyphOf('not_found')).toBe('lucide-circle-x');
    expect(glyphOf('no_text_source')).toBe('lucide-circle-x');
    expect(glyphOf('skipped')).toBe('lucide-circle-slash');
    // Settled-but-incomplete (glyph provisional pending γ ratification).
    expect(glyphOf('untranslated')).toBe('lucide-circle-dashed');

    expect(glyphOf('not_searched')).toBe('lucide-minus');
    expect(glyphOf('probing')).toBe('lucide-loader-circle');
  });

  it('untranslated and not_searched do NOT share a glyph (the skipped-precedent class)', () => {
    expect(glyphOf('untranslated')).not.toBe(glyphOf('not_searched'));
  });

  it('no_text_source and skipped share the muted tint — a deliberate family, not a bug', () => {
    // Both mean "no subtitle is coming"; the user's next step is the same. What
    // must be instant is settled-vs-pending, not which settled reason. Sally
    // ratified the resemblance, so a future reader must not "fix" it apart.
    for (const status of ['no_text_source', 'skipped']) {
      const { unmount } = render(<EpisodeList episodes={[ep(status)]} seasonNumber={1} />);
      expect(screen.getByTestId('episode-subtitle-indicator')).toHaveClass(
        'text-[var(--text-muted)]'
      );
      unmount();
    }
  });
});

// ── 9R-10c — the per-episode subtitle entry (design J3-D `Z54xAd`) ────────

/** Every subtitle_status the ladder can produce (sub-1-2 [@contract-v2]). */
const ALL_STATUSES = [
  'found',
  'not_found',
  'not_searched',
  'searching',
  'probing',
  'extracting',
  'translating',
  'no_text_source',
  'skipped',
  'untranslated',
] as const;

describe('EpisodeList — 管理字幕 entry (9R-10c)', () => {
  it('renders no action at all when the caller passes no handler (existing callers unchanged)', () => {
    render(<EpisodeList episodes={episodes} seasonNumber={1} />);
    expect(screen.queryByTestId('episode-manage-subtitle')).not.toBeInTheDocument();
  });

  it('renders the action only for rows WITH a local file', () => {
    render(<EpisodeList episodes={episodes} seasonNumber={1} onManageSubtitle={vi.fn()} />);

    // episodes[0] and [1] have a local file; [2] does not.
    expect(screen.getAllByTestId('episode-manage-subtitle')).toHaveLength(2);
    // The TMDb-only episode must not offer an action it cannot fulfil.
    expect(screen.queryByLabelText('管理 S01E03 的字幕')).not.toBeInTheDocument();
  });

  it('gives each action an accessible name containing its SxxExx code', () => {
    render(<EpisodeList episodes={episodes} seasonNumber={1} onManageSubtitle={vi.fn()} />);

    // Without the code every one of these buttons would read "管理字幕" and be
    // indistinguishable to a screen-reader user (design ruling-line-4).
    expect(screen.getByLabelText('管理 S01E01 的字幕')).toBeInTheDocument();
    expect(screen.getByLabelText('管理 S01E02 的字幕')).toBeInTheDocument();
  });

  it('raises onManageSubtitle with the episode that was clicked', () => {
    const onManageSubtitle = vi.fn();
    render(
      <EpisodeList episodes={episodes} seasonNumber={1} onManageSubtitle={onManageSubtitle} />
    );

    fireEvent.click(screen.getByLabelText('管理 S01E02 的字幕'));
    expect(onManageSubtitle).toHaveBeenCalledTimes(1);
    expect(onManageSubtitle).toHaveBeenCalledWith(episodes[1]);
  });

  it('meets the 44px touch target', () => {
    render(<EpisodeList episodes={episodes} seasonNumber={1} onManageSubtitle={vi.fn()} />);
    expect(screen.getAllByTestId('episode-manage-subtitle')[0].className).toContain('min-h-[44px]');
  });

  // J3-D's ruling is that the action is IDENTICAL across all ten statuses.
  // A status-dependent action would encode state twice (the indicator already
  // does it) and make buttons flicker down a 25-episode list. This pins the
  // ruling so a future "helpful" refinement has to argue with a test.
  it.each(ALL_STATUSES)('renders the action for subtitle_status=%s (uniform by ruling)', (s) => {
    const { unmount } = render(
      <EpisodeList
        episodes={[
          {
            episodeNumber: 7,
            name: 'ep',
            hasLocalFile: true,
            episodeId: 'ep-uuid-7',
            filePath: '/m/S02E07.mkv',
            subtitleStatus: s,
          },
        ]}
        seasonNumber={2}
        onManageSubtitle={vi.fn()}
      />
    );

    expect(screen.getByLabelText('管理 S02E07 的字幕')).toBeInTheDocument();
    unmount();
  });
});

// ── CR M3 — the gate must be ONE predicate, not two that can disagree ─────

describe('EpisodeList — canManageEpisodeSubtitle gate (CR M3)', () => {
  // The realistic failure: this pair of stories ships BE and FE separately, so a
  // frontend can meet a backend older than 9R-10a. `episode_id` is omitempty, so
  // every row would have a file but no address. Gating the BUTTON on
  // hasLocalFile while the DIALOG needed episodeId rendered buttons that
  // clicked and silently did nothing.
  it('renders no action when the row has a file but NO episodeId (backend predates 9R-10a)', () => {
    render(
      <EpisodeList
        episodes={[{ episodeNumber: 1, name: 'ep', hasLocalFile: true, filePath: '/m/e1.mkv' }]}
        seasonNumber={1}
        onManageSubtitle={vi.fn()}
      />
    );

    expect(screen.queryByTestId('episode-manage-subtitle')).not.toBeInTheDocument();
  });

  it('renders no action when the row has an episodeId but no filePath', () => {
    render(
      <EpisodeList
        episodes={[{ episodeNumber: 1, name: 'ep', hasLocalFile: true, episodeId: 'e1' }]}
        seasonNumber={1}
        onManageSubtitle={vi.fn()}
      />
    );

    expect(screen.queryByTestId('episode-manage-subtitle')).not.toBeInTheDocument();
  });

  it('renders the action once all three are present', () => {
    render(
      <EpisodeList
        episodes={[
          {
            episodeNumber: 1,
            name: 'ep',
            hasLocalFile: true,
            episodeId: 'e1',
            filePath: '/m/e1.mkv',
          },
        ]}
        seasonNumber={1}
        onManageSubtitle={vi.fn()}
      />
    );

    expect(screen.getByTestId('episode-manage-subtitle')).toBeInTheDocument();
  });
});

// ─── disc-2026-10-episode-list-subtitle-badge-b — J11-D ─────────────────────
// The real shapes of 《末日光明》(See), as the badge-a API sends them.

const see = (n: number, extra: Partial<MergedEpisode>): MergedEpisode => ({
  episodeNumber: n,
  name: `第 ${n} 集`,
  hasLocalFile: true,
  episodeId: `see-${n}`,
  filePath: `/tv/See/S01/See.S01E0${n}.mkv`,
  subtitleStatus: 'not_searched',
  embeddedSubtitlesRead: true,
  chineseSubtitleSources: [],
  ...extra,
});

const indicator = (code: string) =>
  document.querySelector(`[data-testid="episode-subtitle-indicator"][data-episode="${code}"]`);

describe('EpisodeList — Chinese-subtitle verdict (badge-b, J11-D)', () => {
  it.each<[string, MergedEpisode, string, string]>([
    [
      'S01E02 zh_hant from the zh-TW file + Vido',
      see(2, {
        chineseSubtitle: 'zh_hant',
        chineseSubtitleSources: [
          { kind: 'vido', language: 'zh-Hant', label: 'zh-Hant' },
          { kind: 'sidecar', language: 'zh-Hant', label: 'zh-TW' },
        ],
      }),
      '有繁中字幕，Vido 生成・旁邊的 zh-TW 檔',
      'text-[var(--success-text)]',
    ],
    [
      'S02E01 zh_hant from the file',
      see(1, {
        chineseSubtitle: 'zh_hant',
        chineseSubtitleSources: [
          { kind: 'embedded', language: 'zh-Hant', label: '繁體中文' },
          { kind: 'embedded', language: 'zh-Hant', label: 'Traditional' },
        ],
      }),
      '有繁中字幕，片內字幕',
      'text-[var(--success-text)]',
    ],
    [
      'zh — Chinese, script untold (Alexyu 1A: reads like zh_hant)',
      see(3, {
        chineseSubtitle: 'zh',
        chineseSubtitleSources: [{ kind: 'embedded', language: 'zh-unknown', label: '' }],
      }),
      '有中文字幕（繁簡未知），片內字幕',
      'text-[var(--success-text)]',
    ],
    [
      'zh_hans',
      see(5, { chineseSubtitle: 'zh_hans' }),
      '只有簡中字幕，可以在「管理字幕」轉成繁中',
      'text-[var(--error-text)]',
    ],
    [
      'S01E01 none',
      see(1, { chineseSubtitle: 'none' }),
      '缺中文字幕，只有英文等其他語言——可以生成',
      'text-[var(--error-text)]',
    ],
    [
      'none before the file was read (Alexyu 2A)',
      see(1, { chineseSubtitle: 'none', embeddedSubtitlesRead: false }),
      '缺中文字幕，還在讀片內字幕，讀完可能會變',
      'text-[var(--error-text)]',
    ],
    [
      'unknown',
      see(8, { chineseSubtitle: 'unknown', embeddedSubtitlesRead: false }),
      '還沒檢查這一集的字幕，下次掃描媒體庫時會讀',
      'text-[var(--text-muted)]',
    ],
  ])('%s', (_name, episode, sentence, color) => {
    render(<EpisodeList episodes={[episode]} seasonNumber={1} />);
    const button = screen.getByRole('button', { name: sentence });
    expect(button).toHaveClass(color);
    expect(button).not.toHaveAttribute('title');
  });

  it('a subtitle being made outranks the verdict', () => {
    render(
      <EpisodeList
        episodes={[see(3, { chineseSubtitle: 'none', subtitleStatus: 'translating' })]}
        seasonNumber={1}
      />
    );
    const button = screen.getByRole('button', { name: '字幕生成中，打開「管理字幕」看進度' });
    expect(button.querySelector('svg')).toHaveClass('animate-spin');
  });

  it('the verdict replaces Vido’s own search record (S01E03: a zh-TW file, never searched)', () => {
    render(
      <EpisodeList
        episodes={[
          see(3, {
            chineseSubtitle: 'zh_hant',
            chineseSubtitleSources: [{ kind: 'sidecar', language: 'zh-Hant', label: 'zh-TW' }],
          }),
        ]}
        seasonNumber={1}
      />
    );
    expect(screen.queryByRole('button', { name: '尚未搜尋字幕' })).not.toBeInTheDocument();
    expect(indicator('S01E03')).toHaveAccessibleName('有繁中字幕，旁邊的 zh-TW 檔');
  });

  it('pressing the icon does not open 管理字幕', () => {
    const onManage = vi.fn();
    render(
      <EpisodeList
        episodes={[see(1, { chineseSubtitle: 'none' })]}
        seasonNumber={1}
        onManageSubtitle={onManage}
      />
    );
    fireEvent.click(indicator('S01E01')!);
    expect(onManage).not.toHaveBeenCalled();
  });

  it('no local file → no indicator, whatever the verdict', () => {
    render(
      <EpisodeList
        episodes={[{ ...see(4, { chineseSubtitle: 'zh_hant' }), hasLocalFile: false }]}
        seasonNumber={1}
      />
    );
    expect(screen.queryByTestId('episode-subtitle-indicator')).not.toBeInTheDocument();
  });
});

it('an unrecognised verdict falls back to Vido’s own record, not "not checked" (CR L4)', () => {
  render(
    <EpisodeList
      episodes={[
        see(1, {
          chineseSubtitle: 'zh_future' as unknown as MergedEpisode['chineseSubtitle'],
          subtitleStatus: 'found',
        }),
      ]}
      seasonNumber={1}
    />
  );
  expect(screen.getByRole('button', { name: '已找到字幕' })).toBeInTheDocument();
});
