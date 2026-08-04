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
    expect(screen.getAllByText('24 分鐘').length).toBeGreaterThan(0);
    expect(screen.getByText('S01E03')).toBeInTheDocument();
  });

  it('shows a subtitle status indicator only for episodes with a local file (AC #6)', () => {
    render(<EpisodeList episodes={episodes} seasonNumber={1} />);

    // ep1 found + ep2 not_found each carry a role=status indicator; ep3 has none.
    const indicators = screen.getAllByRole('status');
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
  ])('renders %s with the long-form accessible name "%s"', (subtitleStatus, label) => {
    render(<EpisodeList episodes={[ep(subtitleStatus)]} seasonNumber={1} />);
    // The icon carries no visible text, so the accessible name is where the full
    // explanation lives — this is where "已略過 must not read as broken" is solved.
    expect(screen.getByRole('status', { name: label })).toBeInTheDocument();
  });

  it('spins for the three in-flight states and NOT for the terminal ones', () => {
    const inFlight = ['probing', 'extracting', 'translating'];
    const terminal = ['no_text_source', 'skipped'];

    for (const status of inFlight) {
      const { unmount } = render(<EpisodeList episodes={[ep(status)]} seasonNumber={1} />);
      expect(screen.getByRole('status').querySelector('svg')).toHaveClass('animate-spin');
      unmount();
    }
    for (const status of terminal) {
      const { unmount } = render(<EpisodeList episodes={[ep(status)]} seasonNumber={1} />);
      expect(screen.getByRole('status').querySelector('svg')).not.toHaveClass('animate-spin');
      unmount();
    }
  });

  it('tints in-flight states with accent, terminal verdicts with muted (sub-1-7a AC #3)', () => {
    const { unmount } = render(<EpisodeList episodes={[ep('translating')]} seasonNumber={1} />);
    // accent is RESERVED for in-progress (Sally 2026-07-05).
    expect(screen.getByRole('status')).toHaveClass('text-[var(--accent-text)]');
    unmount();

    render(<EpisodeList episodes={[ep('no_text_source')]} seasonNumber={1} />);
    expect(screen.getByRole('status')).toHaveClass('text-[var(--text-muted)]');
  });

  it('re-tints the pre-existing searching state to accent (sub-1-7a AC #5 ruling)', () => {
    render(<EpisodeList episodes={[ep('searching')]} seasonNumber={1} />);
    const icon = screen.getByRole('status');
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
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
  });

  it('falls back to not_searched for an unrecognised value (belt and braces)', () => {
    render(<EpisodeList episodes={[ep('some_future_state')]} seasonNumber={1} />);
    expect(screen.getByRole('status', { name: '尚未搜尋字幕' })).toBeInTheDocument();
  });
});
