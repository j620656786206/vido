// Implements: <utility — no .pen counterpart>
/**
 * useEpisodeGenerationLive (disc-2026-10-episode-row-live-progress-a)
 *
 * Keeps ONE expanded season's episode list fresh while subtitles are being
 * generated for it, so the J11 spinner on a row appears when a run starts and
 * turns into the result when it ends — without opening 管理字幕 and without a
 * page reload. It never draws anything itself and never edits the cached rows:
 * the server's `subtitle_status` is the only thing the list shows, this hook
 * only decides WHEN to ask for it again.
 *
 * Why it is needed: the season query used to be refetched only by the
 * 管理字幕 dialog's own terminal callbacks, and the dialog unmounts (closing its
 * stream) the moment it is closed — start eight episodes, close the dialog
 * each time, and no row ever changed.
 *
 * How:
 *   - `markStarted(episodeId)` — the dialog's run was accepted (or was already
 *     running). Refetch now, then every EPISODE_START_POLL_MS until the server
 *     reports that episode in flight, a terminal event for it arrives, or
 *     EPISODE_START_CONFIRM_MS passes. The fast phase exists because the
 *     server answers the click BEFORE its run goroutine writes `extracting`
 *     (solo_runner.go Start → ProcessItem), so the first refetch can miss it.
 *   - While any row is in flight (or a start is unconfirmed) one `/events`
 *     stream is open. A terminal event (`transcription_complete` /
 *     `transcription_failed`, or `subtitle_progress` complete/failed/skipped)
 *     for an episode of THIS season refetches — the backend writes the final
 *     status before it sends those, so the refetch sees the result. Events for
 *     other seasons or titles are ignored (they carry only the episode id).
 *   - A slow safety poll (EPISODE_LIVE_POLL_MS) runs alongside, because the
 *     hub drops frames for a slow client and the Cloudflare tunnel can hold a
 *     stream closed. A stream that has not opened in
 *     EPISODE_LIVE_CONNECT_TIMEOUT_MS is closed rather than left hanging — a
 *     hung one held up the page's other responses
 *     (disc-2026-10-reparse-dialog-stuck-over-tunnel); an error after opening
 *     closes it too, and the poll carries on.
 *   - Nothing in flight → no stream, no timer.
 */
import { useCallback, useEffect, useRef, useState } from 'react';
import type { MergedEpisode } from '../types/library';
import { isSubtitleInFlight } from '../utils/libraryStatus';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

/** Safety poll while a row is in flight (the stream is the fast path). */
export const EPISODE_LIVE_POLL_MS = 15_000;
/** Poll while a just-started run has not shown up on the server yet. */
export const EPISODE_START_POLL_MS = 2_000;
/** Give up waiting for a start to show up after this long. */
export const EPISODE_START_CONFIRM_MS = 30_000;
/** A stream that has not opened by then is closed (tunnel, see above). */
export const EPISODE_LIVE_CONNECT_TIMEOUT_MS = 3_000;
/** Events that arrive together (the pipeline's complete, then the job's)
 *  become one refetch. */
export const EPISODE_LIVE_COALESCE_MS = 250;

const TERMINAL_STAGES = new Set(['complete', 'failed', 'skipped']);

interface Options {
  /** The season's rows as the server last sent them. */
  episodes: MergedEpisode[] | undefined;
  /** The season query's refetch. */
  refetch: () => unknown;
  /** The season is expanded (its query is enabled). Off → nothing runs. */
  enabled: boolean;
}

export interface EpisodeGenerationLive {
  /** A run for this episode was accepted (or was already running). */
  markStarted: (episodeId: string) => void;
}

function mediaIdOf(e: MessageEvent): { mediaId: string; stage: string } | null {
  try {
    const parsed = JSON.parse(e.data);
    const data = parsed?.data ?? parsed;
    if (!data || typeof data !== 'object' || typeof data.media_id !== 'string') return null;
    return { mediaId: data.media_id, stage: typeof data.stage === 'string' ? data.stage : '' };
  } catch {
    return null; // ignore malformed frames
  }
}

export function useEpisodeGenerationLive({
  episodes,
  refetch,
  enabled,
}: Options): EpisodeGenerationLive {
  // Episodes whose start the server has not confirmed yet.
  const [pending, setPending] = useState<ReadonlySet<string>>(() => new Set());
  const pendingTimersRef = useRef(new Map<string, ReturnType<typeof setTimeout>>());

  const refetchRef = useRef(refetch);
  const seasonIdsRef = useRef<ReadonlySet<string>>(new Set());
  const pendingRef = useRef(pending);
  const coalesceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const inFlight = (episodes ?? []).some(
    (ep) => ep.hasLocalFile && !!ep.episodeId && isSubtitleInFlight(ep.subtitleStatus)
  );
  const active = enabled && (inFlight || pending.size > 0);
  const pollMs = !enabled
    ? null
    : pending.size > 0
      ? EPISODE_START_POLL_MS
      : inFlight
        ? EPISODE_LIVE_POLL_MS
        : null;

  useEffect(() => {
    refetchRef.current = refetch;
    pendingRef.current = pending;
    seasonIdsRef.current = new Set(
      (episodes ?? []).flatMap((ep) => (ep.episodeId ? [ep.episodeId] : []))
    );
  });

  const settle = useCallback((episodeId: string) => {
    const timer = pendingTimersRef.current.get(episodeId);
    if (timer) clearTimeout(timer);
    pendingTimersRef.current.delete(episodeId);
    setPending((prev) => {
      if (!prev.has(episodeId)) return prev;
      const next = new Set(prev);
      next.delete(episodeId);
      return next;
    });
  }, []);

  const scheduleRefetch = useCallback(() => {
    if (coalesceRef.current) return;
    coalesceRef.current = setTimeout(() => {
      coalesceRef.current = null;
      void refetchRef.current();
    }, EPISODE_LIVE_COALESCE_MS);
  }, []);

  const markStarted = useCallback(
    (episodeId: string) => {
      const old = pendingTimersRef.current.get(episodeId);
      if (old) clearTimeout(old);
      pendingTimersRef.current.set(
        episodeId,
        setTimeout(() => settle(episodeId), EPISODE_START_CONFIRM_MS)
      );
      setPending((prev) => (prev.has(episodeId) ? prev : new Set(prev).add(episodeId)));
      void refetchRef.current();
    },
    [settle]
  );

  // The server caught up: a pending episode now reads in flight.
  useEffect(() => {
    if (pending.size === 0 || !episodes) return;
    for (const ep of episodes) {
      if (ep.episodeId && pending.has(ep.episodeId) && isSubtitleInFlight(ep.subtitleStatus)) {
        settle(ep.episodeId);
      }
    }
  }, [episodes, pending, settle]);

  // Poll — only while something runs. Skipped in a hidden tab; the query's
  // focus refetch (30s staleTime) catches up when the user comes back.
  useEffect(() => {
    if (pollMs === null) return;
    const id = setInterval(() => {
      if (typeof document !== 'undefined' && document.visibilityState === 'hidden') return;
      void refetchRef.current();
    }, pollMs);
    return () => clearInterval(id);
  }, [pollMs]);

  // The stream — only while something runs.
  useEffect(() => {
    if (!active || typeof EventSource === 'undefined') return;
    const es = new EventSource(`${API_BASE_URL}/events`);
    let opened = false;
    const connectTimer = setTimeout(() => {
      if (!opened) es.close();
    }, EPISODE_LIVE_CONNECT_TIMEOUT_MS);
    es.onopen = () => {
      opened = true;
      clearTimeout(connectTimer);
    };
    es.onerror = () => {
      clearTimeout(connectTimer);
      es.close();
    };

    const ours = (id: string) => seasonIdsRef.current.has(id) || pendingRef.current.has(id);
    const onTerminal = (e: MessageEvent) => {
      const frame = mediaIdOf(e);
      if (!frame || !ours(frame.mediaId)) return;
      settle(frame.mediaId);
      scheduleRefetch();
    };
    es.addEventListener('transcription_complete', onTerminal);
    es.addEventListener('transcription_failed', onTerminal);
    es.addEventListener('subtitle_progress', (e: MessageEvent) => {
      const frame = mediaIdOf(e);
      if (!frame || !ours(frame.mediaId)) return;
      if (TERMINAL_STAGES.has(frame.stage)) {
        settle(frame.mediaId);
        scheduleRefetch();
        return;
      }
      // A just-started run reporting a stage past `probing` has already
      // written `extracting` (process_item.go writes it before that frame),
      // so asking now confirms the start without waiting for the poll.
      if (frame.stage !== 'probing' && pendingRef.current.has(frame.mediaId)) scheduleRefetch();
    });

    return () => {
      clearTimeout(connectTimer);
      es.close();
    };
  }, [active, settle, scheduleRefetch]);

  // Unmount: no timer outlives the season row.
  useEffect(() => {
    const timers = pendingTimersRef.current;
    return () => {
      for (const t of timers.values()) clearTimeout(t);
      timers.clear();
      if (coalesceRef.current) clearTimeout(coalesceRef.current);
    };
  }, []);

  return { markStarted };
}
