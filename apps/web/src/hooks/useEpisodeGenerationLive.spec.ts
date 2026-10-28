import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import {
  EPISODE_LIVE_COALESCE_MS,
  EPISODE_LIVE_CONNECT_TIMEOUT_MS,
  EPISODE_LIVE_POLL_MS,
  EPISODE_START_CONFIRM_MS,
  EPISODE_START_POLL_MS,
  useEpisodeGenerationLive,
} from './useEpisodeGenerationLive';
import type { MergedEpisode } from '../types/library';

class MockEventSource {
  static instances: MockEventSource[] = [];
  listeners: Record<string, ((e: MessageEvent) => void)[]> = {};
  closed = false;
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  constructor(public url: string) {
    MockEventSource.instances.push(this);
  }
  addEventListener(type: string, cb: (e: MessageEvent) => void) {
    (this.listeners[type] ??= []).push(cb);
  }
  close() {
    this.closed = true;
  }
  emit(type: string, data: unknown) {
    this.listeners[type]?.forEach((cb) =>
      cb(new MessageEvent(type, { data: JSON.stringify({ type, data }) }))
    );
  }
}

const openStreams = () => MockEventSource.instances.filter((es) => !es.closed);

const ep = (
  n: number,
  subtitleStatus: string,
  over: Partial<MergedEpisode> = {}
): MergedEpisode => ({
  episodeNumber: n,
  name: `第 ${n} 集`,
  hasLocalFile: true,
  episodeId: `ep-${n}`,
  filePath: `/media/S01E0${n}.mkv`,
  subtitleStatus,
  chineseSubtitle: 'zh_hant',
  ...over,
});

type Props = { episodes: MergedEpisode[] | undefined; enabled: boolean };

function setup(initial: Props) {
  const refetch = vi.fn();
  const view = renderHook((p: Props) => useEpisodeGenerationLive({ ...p, refetch }), {
    initialProps: initial,
  });
  return { refetch, ...view };
}

describe('useEpisodeGenerationLive (disc-2026-10-episode-row-live-progress-a)', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    MockEventSource.instances = [];
    (globalThis as Record<string, unknown>).EventSource = MockEventSource;
  });
  afterEach(() => {
    delete (globalThis as Record<string, unknown>).EventSource;
    vi.useRealTimers();
  });

  it('is silent when nothing runs: no stream, no polling (AC #4)', () => {
    const { refetch } = setup({ episodes: [ep(1, 'found'), ep(2, 'not_searched')], enabled: true });
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_POLL_MS * 4));
    expect(MockEventSource.instances).toHaveLength(0);
    expect(refetch).not.toHaveBeenCalled();
  });

  it('refetches at once when a run starts, then fast until the server shows it (AC #1)', () => {
    const { result, refetch, rerender } = setup({ episodes: [ep(1, 'found')], enabled: true });

    act(() => result.current.markStarted('ep-1'));
    expect(refetch).toHaveBeenCalledTimes(1);
    expect(openStreams()).toHaveLength(1);

    // The first answer raced the run's own write — still the old ✓.
    act(() => vi.advanceTimersByTime(EPISODE_START_POLL_MS));
    expect(refetch).toHaveBeenCalledTimes(2);

    // The server now reports it in flight → back to the slow safety poll.
    rerender({ episodes: [ep(1, 'extracting')], enabled: true });
    refetch.mockClear();
    act(() => vi.advanceTimersByTime(EPISODE_START_POLL_MS * 3));
    expect(refetch).not.toHaveBeenCalled();
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_POLL_MS));
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it('stops waiting for an unconfirmed start after the confirm window', () => {
    const { result, refetch } = setup({ episodes: [ep(1, 'found')], enabled: true });
    act(() => result.current.markStarted('ep-1'));
    act(() => vi.advanceTimersByTime(EPISODE_START_CONFIRM_MS));
    expect(openStreams()).toHaveLength(0);
    refetch.mockClear();
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_POLL_MS * 2));
    expect(refetch).not.toHaveBeenCalled();
  });

  it('polls only while a row is in flight, and stops once it is not (AC #3, #4)', () => {
    const { refetch, rerender } = setup({
      episodes: [ep(1, 'translating'), ep(2, 'found')],
      enabled: true,
    });
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_POLL_MS));
    expect(refetch).toHaveBeenCalledTimes(1);
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_POLL_MS));
    expect(refetch).toHaveBeenCalledTimes(2);

    rerender({ episodes: [ep(1, 'found'), ep(2, 'found')], enabled: true });
    expect(openStreams()).toHaveLength(0);
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_POLL_MS * 3));
    expect(refetch).toHaveBeenCalledTimes(2);
  });

  it('skips the poll while the tab is hidden and resumes when it is visible', () => {
    const visibility = vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden');
    const { refetch } = setup({ episodes: [ep(1, 'extracting')], enabled: true });
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_POLL_MS * 2));
    expect(refetch).not.toHaveBeenCalled();
    visibility.mockReturnValue('visible');
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_POLL_MS));
    expect(refetch).toHaveBeenCalledTimes(1);
    visibility.mockRestore();
  });

  it('does nothing while the season is collapsed, even with a row in flight', () => {
    const { refetch } = setup({ episodes: [ep(1, 'extracting')], enabled: false });
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_POLL_MS * 2));
    expect(MockEventSource.instances).toHaveLength(0);
    expect(refetch).not.toHaveBeenCalled();
  });

  it('refetches on a terminal event for an episode of this season (AC #2, #5)', () => {
    const { refetch } = setup({ episodes: [ep(1, 'extracting'), ep(2, 'found')], enabled: true });
    const es = MockEventSource.instances[0];
    expect(es.url).toBe('/api/v1/events');

    act(() => es.emit('transcription_complete', { media_id: 'ep-1', job_id: 'j1' }));
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_COALESCE_MS));
    expect(refetch).toHaveBeenCalledTimes(1);

    act(() => es.emit('transcription_failed', { media_id: 'ep-2' }));
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_COALESCE_MS));
    expect(refetch).toHaveBeenCalledTimes(2);

    act(() => es.emit('subtitle_progress', { media_id: 'ep-1', stage: 'failed' }));
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_COALESCE_MS));
    expect(refetch).toHaveBeenCalledTimes(3);
  });

  it('turns the end of a run into one refetch, not one per event', () => {
    const { refetch } = setup({ episodes: [ep(1, 'extracting')], enabled: true });
    const es = MockEventSource.instances[0];
    act(() => {
      es.emit('subtitle_progress', { media_id: 'ep-1', stage: 'complete' });
      es.emit('transcription_complete', { media_id: 'ep-1' });
    });
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_COALESCE_MS));
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it('ignores events for other seasons/titles and mid-run stages of running rows', () => {
    const { refetch } = setup({ episodes: [ep(1, 'extracting')], enabled: true });
    const es = MockEventSource.instances[0];
    act(() => {
      es.emit('transcription_complete', { media_id: 'someone-else' });
      es.emit('subtitle_progress', { media_id: 'movie-9', stage: 'complete' });
      es.emit('subtitle_progress', { media_id: 'ep-1', stage: 'translating' });
      es.emit('subtitle_progress', { media_id: 'ep-1' }); // malformed: no stage
    });
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_COALESCE_MS * 4));
    expect(refetch).not.toHaveBeenCalled();
  });

  it('confirms a pending start from its first stage past probing', () => {
    const { result, refetch } = setup({ episodes: [ep(1, 'found')], enabled: true });
    act(() => result.current.markStarted('ep-1'));
    refetch.mockClear();
    const es = MockEventSource.instances[0];

    act(() => es.emit('subtitle_progress', { media_id: 'ep-1', stage: 'probing' }));
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_COALESCE_MS));
    expect(refetch).not.toHaveBeenCalled();

    act(() => es.emit('subtitle_progress', { media_id: 'ep-1', stage: 'extracting' }));
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_COALESCE_MS));
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it('a run that ends before the server ever showed it settles on its terminal event', () => {
    const { result, refetch } = setup({ episodes: [ep(1, 'not_searched')], enabled: true });
    act(() => result.current.markStarted('ep-1'));
    const es = MockEventSource.instances[0];
    act(() => es.emit('transcription_complete', { media_id: 'ep-1' }));
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_COALESCE_MS));
    // Nothing pending and nothing in flight → quiet again.
    expect(openStreams()).toHaveLength(0);
    refetch.mockClear();
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_POLL_MS * 2));
    expect(refetch).not.toHaveBeenCalled();
  });

  it('closes a stream that never opens and keeps polling (tunnel)', () => {
    const { refetch } = setup({ episodes: [ep(1, 'extracting')], enabled: true });
    const es = MockEventSource.instances[0];
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_CONNECT_TIMEOUT_MS));
    expect(es.closed).toBe(true);
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_POLL_MS));
    expect(refetch).toHaveBeenCalled();
  });

  it('keeps an opened stream past the connect timeout, and closes it on error', () => {
    setup({ episodes: [ep(1, 'extracting')], enabled: true });
    const es = MockEventSource.instances[0];
    act(() => es.onopen?.());
    act(() => vi.advanceTimersByTime(EPISODE_LIVE_CONNECT_TIMEOUT_MS * 2));
    expect(es.closed).toBe(false);
    act(() => es.onerror?.());
    expect(es.closed).toBe(true);
  });

  it('closes the stream and stops timers on unmount', () => {
    const { refetch, unmount, result } = setup({ episodes: [ep(1, 'extracting')], enabled: true });
    act(() => result.current.markStarted('ep-2'));
    refetch.mockClear();
    unmount();
    expect(openStreams()).toHaveLength(0);
    act(() => vi.advanceTimersByTime(EPISODE_START_CONFIRM_MS + EPISODE_LIVE_POLL_MS));
    expect(refetch).not.toHaveBeenCalled();
  });
});
