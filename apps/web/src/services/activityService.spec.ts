import { describe, it, expect, vi, afterEach } from 'vitest';
import { activityService } from './activityService';

afterEach(() => {
  vi.unstubAllGlobals();
});

// disc-2026-10-activity-subtitle-run-time-b AC #1 — the -a story's
// [@contract-v1] wire keys arrive camelCased on RecentEvent.
describe('activityService.getActivity — subtitle recent event wire shape', () => {
  it('maps every [@contract-v1] key, keeps spent_usd 0 as 0, leaves absent keys absent', async () => {
    const body = {
      success: true,
      data: {
        active_jobs: { status: 'ok', jobs: [] },
        pending: { status: 'ok', parse_count: 0 },
        downloads: { status: 'ok', downloading: 0, queued: 0, errored: 0, paused: 0, total: 0 },
        recent: {
          status: 'ok',
          events: [
            {
              kind: 'subtitle',
              result: 'completed',
              at: '2026-10-19T01:25:01Z',
              media_type: 'episode',
              media_title: '末日光明',
              season_number: 1,
              episode_number: 2,
              route: 'translate',
              cue_count: 568,
              duration_seconds: 211,
              spent_usd: 0.197439,
            },
            {
              kind: 'subtitle',
              result: 'failed',
              at: '2026-10-19T01:00:00Z',
              media_type: 'movie',
              media_title: '',
              duration_seconds: 4,
              spent_usd: 0,
              failure_reason: 'target_not_writable',
            },
          ],
        },
      },
    };
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(body) })
    );
    const got = await activityService.getActivity();
    expect(got.recent.events[0]).toEqual({
      kind: 'subtitle',
      result: 'completed',
      at: '2026-10-19T01:25:01Z',
      mediaType: 'episode',
      mediaTitle: '末日光明',
      seasonNumber: 1,
      episodeNumber: 2,
      route: 'translate',
      cueCount: 568,
      durationSeconds: 211,
      spentUsd: 0.197439,
    });
    const failed = got.recent.events[1];
    expect(failed.spentUsd).toBe(0);
    expect(failed.mediaTitle).toBe('');
    expect(failed.failureReason).toBe('target_not_writable');
    expect('cueCount' in failed).toBe(false);
  });
});
