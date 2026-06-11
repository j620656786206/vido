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
