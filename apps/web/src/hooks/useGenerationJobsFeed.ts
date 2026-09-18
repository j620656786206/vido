/**
 * Lazy SSE generation-jobs feed hook (Story ux3-ai-2 AC 4/5, reworked by
 * dsr-6d-c-2) — the generation WORKSPACE's live surface. Clones the
 * `useGenerationProgress` SSE anatomy (lazy §8, double-nested `parsed.data`
 * unwrap, snakeToCamel, 10s reconnect, `mountedRef`, latest-ref connect) but
 * listens UNFILTERED to derive two outputs from the one shared stream:
 *
 *   - `feed`: a session-scoped event log. It records WHAT HAPPENED; the words
 *     and colours are the view's job (generationEventCopy.ts). The rules:
 *       · one row per film per stage — later frames of the same stage update
 *         that row in place (translation fires every 10 cues: 150 frames for
 *         one film used to be 150 rows);
 *       · a stage row stops being `live` in one of two ways: `passed` (the film
 *         moved on to its next stage, or finished) or `stopped` (it failed, the
 *         batch ended, or the stream reopened — events in the gap are gone for
 *         good, the hub has no replay). Only `passed` may be drawn as a tick;
 *         a stopped step did not finish (dsr-6d-c-2 CR H2);
 *       · a BATCH item's result comes only from the batch's `changed_item`.
 *         Its own `transcription_failed` arrives BEFORE the batch pauses it at
 *         the budget ceiling or cancels it, so trusting that event would log a
 *         paused film as failed while the queue beside it says paused;
 *       · "batch member" means the ONE film the batch is running right now (it
 *         runs them one at a time): a queued film run from its detail page is
 *         a single job, and so is any event that carries a backend `title` —
 *         only solo runs send one (CR M4);
 *       · `subtitle_progress` is only read for that member — the subtitle
 *         search engine and the manual download reuse that event name, in
 *         English, for work that is not generation — and its terminals only
 *         end a step the pipeline itself started (CR M8).
 *     NO timestamps — SSE payloads carry none, so the log is order-only
 *     (Rule 23-clean). Row keys use a monotonic `seq`, never `Date.now()`.
 *   - `singleJobs`: in-flight transcriptions that are NOT part of a batch
 *     (detail-page runs). Batch members never enter it: after 關閉 the workspace
 *     falls back to single mode, and a stale batch item there was a fake
 *     「進行中任務」 (dsr-6d-c-1 CR M6). A single job whose terminal event the hub
 *     dropped still lingers (disc-2026-09-workspace-single-job-lost-terminal).
 *
 * §8 lazy-SSE: NO connect on mount — the workspace calls `startTracking()` while
 * the view is active AND visible; `stop()` closes the stream on leave/hide.
 */
import { useCallback, useEffect, useReducer, useRef } from 'react';
import { snakeToCamel } from '../utils/caseTransform';
import type { GenerationPhase } from './useGenerationProgress';
import type {
  GenerationBatchItemReason,
  GenerationBatchItemState,
  GenerationBatchItemStatus,
} from '../services/subtitleService';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';
const SSE_RECONNECT_MS = 10000;
/** Cap the feed so a long-running batch never grows an unbounded array. */
export const FEED_CAP = 200;

/**
 * A stage a film can be in. `track` is the pipeline's embedded-subtitle
 * extraction (D6 `subtitle_progress` `extracting`) — not audio.
 */
export type FeedStage = 'extracting' | 'transcribing' | 'translating' | 'track';

export type BatchTerminalStatus = 'complete' | 'cancelled' | 'error' | 'budget_ceiling';

export type FeedStageState = 'live' | 'passed' | 'stopped';

interface ItemRowBase {
  /** Monotonic React key (deterministic, wall-clock-free — Rule 23). */
  seq: number;
  /** UUID-string media row id (movie or episode). */
  mediaId: string;
  /** Raw title ('' = unknown); the view composes it with seriesTitle. */
  title: string;
  seriesTitle: string;
}

export type FeedRow =
  | (ItemRowBase & {
      kind: 'stage';
      stage: FeedStage;
      /**
       * `live` — the film's current stage (accent + spinner);
       * `passed` — it moved on to the next stage or finished (a tick is true);
       * `stopped` — it ended without finishing this step, or we lost track of
       * it (failure, batch end, stream gap). Never drawn as done.
       */
      state: FeedStageState;
      /** Translation only; null once the row is no longer live. */
      percentage: number | null;
      /** Started by a D6 `subtitle_progress` stage (pipeline), not a transcription event. */
      pipeline: boolean;
    })
  | (ItemRowBase & { kind: 'done' })
  | (ItemRowBase & {
      kind: 'failed';
      /** Batch item: the backend reason. Single job: null. */
      reason: GenerationBatchItemReason | null;
      /** The raw backend error, for the view to map — never displayed as-is. */
      error: string | null;
    })
  | {
      seq: number;
      kind: 'batch';
      batchId: string;
      status: BatchTerminalStatus;
      successCount: number;
      failCount: number;
      budgetUsd: number;
    };

export interface SingleJobState {
  mediaId: string;
  phase: GenerationPhase;
  /** Backend-resolved title (dsr-6d-c-2 AC #8); '' = unknown. */
  title: string;
  message: string;
  percentage: number | null;
}

interface MemberInfo {
  title: string;
  seriesTitle: string;
}

export interface GenerationJobsFeedState {
  feed: FeedRow[];
  singleJobs: Record<string, SingleJobState>;
  seq: number;
  connected: boolean;
  /**
   * The film(s) the batch is running right now — in practice one. Added by
   * `changed_item: running` / `seedBatch`, removed by its done/failed
   * changed_item, cleared at the batch terminal or `endBatch()`.
   */
  members: Record<string, MemberInfo>;
  /** A member's own transcription_failed text, used when its reason is `error`. */
  memberErrors: Record<string, string>;
  /** `${batchId}:${status}` terminals already written. */
  batchRowsSeen: string[];
}

const initialState: GenerationJobsFeedState = {
  feed: [],
  singleJobs: {},
  seq: 0,
  connected: false,
  members: {},
  memberErrors: {},
  batchRowsSeen: [],
};

/** Camelized transcription_* payload (envelope-unwrapped). */
interface TranscriptionPayload {
  mediaId?: string;
  title?: string;
  phase?: string;
  percentage?: number;
  message?: string;
  error?: string;
}
/** Camelized D6 subtitle_progress payload. */
interface PipelinePayload {
  mediaId?: string;
  stage?: string;
}
interface ChangedItem {
  mediaId: string;
  title: string;
  seriesTitle?: string;
  status: GenerationBatchItemStatus;
  reason: GenerationBatchItemReason;
}
/** Camelized generation_batch_progress payload, `items` stripped before conversion. */
interface BatchPayload {
  batchId?: string;
  status?: string;
  successCount?: number;
  failCount?: number;
  budgetUsd?: number;
  changedItem?: ChangedItem | null;
}

type ActivePhase = 'extracting' | 'transcribing' | 'translating';

const TRANSCRIPTION_EVENTS: ReadonlyArray<{ event: string; phase: ActivePhase }> = [
  { event: 'transcription_extracting', phase: 'extracting' },
  { event: 'transcription_progress', phase: 'transcribing' },
  { event: 'translation_progress', phase: 'translating' },
];

/**
 * D6 stages that get a row. `probing`/`placing`/`converting` are momentary and
 * always followed by another stage; the terminals are ignored because a member's
 * result comes from the batch.
 */
const D6_STAGE: Readonly<Record<string, FeedStage>> = {
  extracting: 'track',
  translating: 'translating',
};

const BATCH_TERMINALS = new Set<string>(['complete', 'cancelled', 'error', 'budget_ceiling']);

type Action =
  | { type: 'OPEN' }
  | { type: 'CLOSED' }
  | {
      type: 'STAGE';
      stage: FeedStage;
      mediaId: string;
      payload: TranscriptionPayload;
      pipeline: boolean;
    }
  | { type: 'SINGLE_DONE'; payload: TranscriptionPayload }
  | { type: 'SINGLE_FAILED'; payload: TranscriptionPayload }
  | { type: 'PIPELINE_END'; mediaId: string; ok: boolean }
  | { type: 'BATCH'; payload: BatchPayload }
  | { type: 'SEED'; items: GenerationBatchItemState[] }
  | { type: 'END_BATCH' };

/** A backend title that is really the media id is no title at all. */
function honestTitle(title: string | undefined, mediaId: string): string {
  return title && title !== mediaId ? title : '';
}

type DistributiveOmit<T, K extends keyof T> = T extends unknown ? Omit<T, K> : never;
type NewRow = DistributiveOmit<FeedRow, 'seq'>;

/** Append one row, assigning its seq and enforcing FEED_CAP (drop-oldest). */
function append(feed: FeedRow[], seq: number, row: NewRow): { feed: FeedRow[]; seq: number } {
  const next = [...feed, { ...row, seq: seq + 1 } as FeedRow];
  return { feed: next.length > FEED_CAP ? next.slice(next.length - FEED_CAP) : next, seq: seq + 1 };
}

/** Every live row matching `which` stops claiming to run, as `passed` or `stopped`. */
function settle(
  feed: FeedRow[],
  which: (row: Extract<FeedRow, { kind: 'stage' }>) => boolean,
  state: 'passed' | 'stopped'
): FeedRow[] {
  let changed = false;
  const next = feed.map((row) => {
    if (row.kind === 'stage' && row.state === 'live' && which(row)) {
      changed = true;
      return { ...row, state, percentage: null };
    }
    return row;
  });
  return changed ? next : feed;
}

const ofFilm = (mediaId: string) => (row: { mediaId: string }) => row.mediaId === mediaId;

/** The film's current live row, if any. */
function liveRow(feed: FeedRow[], mediaId: string) {
  for (let i = feed.length - 1; i >= 0; i -= 1) {
    const row = feed[i];
    if (row.kind === 'stage' && row.state === 'live' && row.mediaId === mediaId) {
      return { row, index: i };
    }
  }
  return null;
}

/** The film's most recent row of any kind. */
function lastRowOf(feed: FeedRow[], mediaId: string): FeedRow | null {
  for (let i = feed.length - 1; i >= 0; i -= 1) {
    const row = feed[i];
    if (row.kind !== 'batch' && row.mediaId === mediaId) return row;
  }
  return null;
}

/** Fill in the title of rows written before the film's name was known. */
function backfill(feed: FeedRow[], names: Record<string, MemberInfo>): FeedRow[] {
  let changed = false;
  const next = feed.map((row) => {
    if (row.kind === 'batch' || row.title) return row;
    const info = names[row.mediaId];
    if (!info?.title) return row;
    changed = true;
    return { ...row, title: info.title, seriesTitle: info.seriesTitle };
  });
  return changed ? next : feed;
}

function withoutKey<T>(record: Record<string, T>, key: string): Record<string, T> {
  if (!(key in record)) return record;
  const next = { ...record };
  delete next[key];
  return next;
}

/** Everything the batch left behind ends: its running film's step, its membership. */
function endMembership(state: GenerationJobsFeedState): GenerationJobsFeedState {
  const ids = state.members;
  const feed = settle(state.feed, (row) => row.mediaId in ids, 'stopped');
  if (feed === state.feed && Object.keys(ids).length === 0) return state;
  return { ...state, feed, members: {}, memberErrors: {} };
}

function reducer(state: GenerationJobsFeedState, action: Action): GenerationJobsFeedState {
  switch (action.type) {
    case 'OPEN':
      // (Re)opened: whatever was live before cannot be vouched for any more.
      return { ...state, connected: true, feed: settle(state.feed, () => true, 'stopped') };

    case 'CLOSED':
      return state.connected ? { ...state, connected: false } : state;

    case 'STAGE': {
      const { stage, mediaId, payload, pipeline } = action;
      // Only a solo run sends a title (dsr-6d-c-2 AC #8): such an event is never
      // the batch's, even for a film the batch also lists.
      const soloTitle = pipeline ? '' : honestTitle(payload.title, mediaId);
      const member = soloTitle ? undefined : state.members[mediaId];
      if (pipeline && !member) return state; // not generation, or not ours
      const title = member ? member.title : soloTitle || state.singleJobs[mediaId]?.title || '';
      const seriesTitle = member ? member.seriesTitle : '';
      const percentage =
        stage === 'translating' && !pipeline && typeof payload.percentage === 'number'
          ? Math.round(payload.percentage)
          : null;

      const singleJobs = member
        ? state.singleJobs
        : {
            ...state.singleJobs,
            [mediaId]: {
              mediaId,
              phase: (stage === 'track' ? 'extracting' : stage) as GenerationPhase,
              title,
              message: payload.message ?? state.singleJobs[mediaId]?.message ?? '',
              percentage: percentage ?? state.singleJobs[mediaId]?.percentage ?? null,
            },
          };

      const current = liveRow(state.feed, mediaId);
      if (current && current.row.kind === 'stage' && current.row.stage === stage) {
        // Same stage again: update in place, no new row, no new seq.
        const feed = [...state.feed];
        feed[current.index] = { ...current.row, percentage, title: current.row.title || title };
        return { ...state, feed, singleJobs };
      }
      // A new stage: the one before it is behind the film now.
      const moved = settle(state.feed, ofFilm(mediaId), 'passed');
      const next = append(moved, state.seq, {
        kind: 'stage',
        mediaId,
        title,
        seriesTitle,
        stage,
        state: 'live',
        percentage,
        pipeline,
      });
      return { ...state, ...next, singleJobs };
    }

    case 'SINGLE_DONE':
    case 'SINGLE_FAILED': {
      const mediaId = action.payload.mediaId;
      if (!mediaId) return state;
      const ok = action.type === 'SINGLE_DONE';
      const soloTitle = honestTitle(action.payload.title, mediaId);
      const feed = settle(state.feed, ofFilm(mediaId), ok ? 'passed' : 'stopped');
      if (!soloTitle && state.members[mediaId]) {
        // A batch member: its result comes from changed_item. Remember why it
        // failed in case the batch only says `error`.
        const memberErrors =
          !ok && action.payload.error
            ? { ...state.memberErrors, [mediaId]: action.payload.error }
            : state.memberErrors;
        return { ...state, feed, memberErrors };
      }
      const title = soloTitle || state.singleJobs[mediaId]?.title || '';
      const row: NewRow = ok
        ? { kind: 'done', mediaId, title, seriesTitle: '' }
        : {
            kind: 'failed',
            mediaId,
            title,
            seriesTitle: '',
            reason: null,
            error: action.payload.error ?? action.payload.message ?? null,
          };
      const next = append(feed, state.seq, row);
      return { ...state, ...next, singleJobs: withoutKey(state.singleJobs, mediaId) };
    }

    case 'PIPELINE_END': {
      if (!state.members[action.mediaId]) return state;
      // Only end a step the PIPELINE started: the subtitle search engine emits
      // the same terminal names for the same film (CR M8).
      const feed = settle(
        state.feed,
        (row) => row.mediaId === action.mediaId && row.pipeline,
        action.ok ? 'passed' : 'stopped'
      );
      return feed === state.feed ? state : { ...state, feed };
    }

    case 'SEED': {
      // The probe is authoritative about what runs NOW: replace, don't merge —
      // a batch that ended while we were away must not keep its members.
      const running = action.items.filter((it) => it.status === 'running');
      const ended = endMembership({
        ...state,
        members: Object.fromEntries(
          Object.entries(state.members).filter(([id]) => !running.some((it) => it.mediaId === id))
        ),
      });
      const members: Record<string, MemberInfo> = {};
      const singleJobs = { ...ended.singleJobs };
      for (const it of running) {
        members[it.mediaId] = { title: it.title, seriesTitle: it.seriesTitle ?? '' };
        // Its events may have arrived before we knew it was the batch's.
        delete singleJobs[it.mediaId];
      }
      const memberErrors = Object.fromEntries(
        Object.entries(state.memberErrors).filter(([id]) => id in members)
      );
      return { ...ended, members, memberErrors, singleJobs, feed: backfill(ended.feed, members) };
    }

    case 'END_BATCH':
      return endMembership(state);

    case 'BATCH': {
      const { payload } = action;
      const status = payload.status ?? '';
      const batchId = payload.batchId ?? '';
      let { feed, seq, members, singleJobs, memberErrors } = state;

      const changed = payload.changedItem;
      if (changed?.mediaId) {
        const id = changed.mediaId;
        const info = { title: changed.title ?? '', seriesTitle: changed.seriesTitle ?? '' };
        feed = backfill(feed, { [id]: info });
        if (changed.status === 'running') {
          // A solo run already on this film means the batch cannot run it (the
          // backend's single flight) — it will report busy_elsewhere; keep the
          // single job as it is.
          if (!singleJobs[id]) members = { ...members, [id]: info };
        } else if (changed.status === 'done' || changed.status === 'failed') {
          const busy = changed.reason === 'busy_elsewhere';
          // busy_elsewhere: the live row (if any) is someone else's single job.
          if (!busy) {
            feed = settle(feed, ofFilm(id), changed.status === 'done' ? 'passed' : 'stopped');
          }
          const row: NewRow =
            changed.status === 'done'
              ? { kind: 'done', mediaId: id, ...info }
              : {
                  kind: 'failed',
                  mediaId: id,
                  ...info,
                  reason: changed.reason,
                  error: changed.reason === 'error' ? (memberErrors[id] ?? null) : null,
                };
          // The same result may already be in the log (a changed_item: running
          // lost to the hub made the run look like a single job).
          const last = lastRowOf(feed, id);
          if (!(last && last.kind === row.kind && row.kind === 'done')) {
            const next = append(feed, seq, row);
            feed = next.feed;
            seq = next.seq;
          }
          members = withoutKey(members, id);
          memberErrors = withoutKey(memberErrors, id);
          // busy_elsewhere: the film IS running — as someone else's single job.
          if (!busy) singleJobs = withoutKey(singleJobs, id);
        }
      }

      let batchRowsSeen = state.batchRowsSeen;
      const key = `${batchId}:${status}`;
      if (BATCH_TERMINALS.has(status) && !batchRowsSeen.includes(key)) {
        const ids = members;
        feed = settle(feed, (row) => row.mediaId in ids, 'stopped');
        const next = append(feed, seq, {
          kind: 'batch',
          batchId,
          status: status as BatchTerminalStatus,
          successCount: payload.successCount ?? 0,
          failCount: payload.failCount ?? 0,
          budgetUsd: payload.budgetUsd ?? 0,
        });
        feed = next.feed;
        seq = next.seq;
        batchRowsSeen = [...batchRowsSeen.slice(-19), key];
        members = {};
        memberErrors = {};
      }

      return { ...state, feed, seq, members, singleJobs, memberErrors, batchRowsSeen };
    }

    default:
      return state;
  }
}

export function useGenerationJobsFeed() {
  const [state, dispatch] = useReducer(reducer, initialState);
  const esRef = useRef<EventSource | null>(null);
  const reconnectRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const mountedRef = useRef(true);
  const connectRef = useRef<() => void>(() => {});

  const closeSSE = useCallback(() => {
    if (esRef.current) {
      esRef.current.close();
      esRef.current = null;
    }
    if (reconnectRef.current) {
      clearTimeout(reconnectRef.current);
      reconnectRef.current = undefined;
    }
  }, []);

  /** Unwrap the envelope; convert only what the reducer reads. */
  const unwrap = useCallback((e: MessageEvent): Record<string, unknown> | null => {
    try {
      const parsed = JSON.parse(e.data);
      const data = parsed.data ?? parsed;
      return data && typeof data === 'object' ? (data as Record<string, unknown>) : null;
    } catch {
      return null; // ignore malformed frames
    }
  }, []);

  const connect = useCallback(() => {
    if (reconnectRef.current) {
      clearTimeout(reconnectRef.current);
      reconnectRef.current = undefined;
    }
    if (esRef.current) esRef.current.close();
    const es = new EventSource(`${API_BASE_URL}/events`);
    esRef.current = es;

    es.onopen = () => {
      if (mountedRef.current) dispatch({ type: 'OPEN' });
    };

    for (const { event, phase } of TRANSCRIPTION_EVENTS) {
      es.addEventListener(event, (e: MessageEvent) => {
        if (!mountedRef.current) return;
        const raw = unwrap(e);
        if (!raw) return;
        const payload = snakeToCamel<TranscriptionPayload>(raw);
        if (!payload.mediaId) return;
        dispatch({
          type: 'STAGE',
          stage: phase,
          mediaId: payload.mediaId,
          payload,
          pipeline: false,
        });
      });
    }
    es.addEventListener('transcription_complete', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      const raw = unwrap(e);
      if (raw) dispatch({ type: 'SINGLE_DONE', payload: snakeToCamel<TranscriptionPayload>(raw) });
    });
    es.addEventListener('transcription_failed', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      const raw = unwrap(e);
      if (raw)
        dispatch({ type: 'SINGLE_FAILED', payload: snakeToCamel<TranscriptionPayload>(raw) });
    });
    // D6 pipeline stages — the reducer keeps them for batch members only.
    es.addEventListener('subtitle_progress', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      const raw = unwrap(e);
      if (!raw) return;
      const payload = snakeToCamel<PipelinePayload>(raw);
      if (!payload.mediaId || !payload.stage) return;
      const stage = D6_STAGE[payload.stage];
      if (stage) {
        dispatch({ type: 'STAGE', stage, mediaId: payload.mediaId, payload: {}, pipeline: true });
      } else if (['complete', 'failed', 'skipped'].includes(payload.stage)) {
        dispatch({
          type: 'PIPELINE_END',
          mediaId: payload.mediaId,
          ok: payload.stage === 'complete',
        });
      }
    });
    es.addEventListener('generation_batch_progress', (e: MessageEvent) => {
      if (!mountedRef.current) return;
      const raw = unwrap(e);
      if (!raw) return;
      // The terminal frame carries the WHOLE queue (2,400 films on a select-all);
      // this log never reads it, so it is dropped before any conversion.
      const { items: _items, ...rest } = raw;
      dispatch({ type: 'BATCH', payload: snakeToCamel<BatchPayload>(rest) });
    });

    es.onerror = () => {
      if (!mountedRef.current) return;
      dispatch({ type: 'CLOSED' });
      es.close();
      if (reconnectRef.current) clearTimeout(reconnectRef.current);
      reconnectRef.current = setTimeout(() => {
        if (mountedRef.current) connectRef.current();
      }, SSE_RECONNECT_MS);
    };
  }, [unwrap]);

  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  useEffect(() => {
    mountedRef.current = true;
    // NO connect on mount (§8) — the workspace calls startTracking() when active+visible.
    return () => {
      mountedRef.current = false;
      closeSSE();
    };
  }, [closeSSE]);

  const startTracking = useCallback(() => {
    if (!esRef.current || esRef.current.readyState === 2) connect();
  }, [connect]);

  const stop = useCallback(() => {
    closeSSE();
    dispatch({ type: 'CLOSED' });
  }, [closeSSE]);

  /**
   * Tell the log what the batch is running NOW (the status probe's
   * `progress.items`; only `running` ones count). Replaces the membership —
   * needed when the page opens mid-batch, where the running film's events
   * would otherwise count as an untitled single job. `batchId` names the batch
   * for the caller's benefit; membership itself ends with the batch terminal.
   */
  const seedBatch = useCallback((_batchId: string, items: GenerationBatchItemState[]) => {
    dispatch({ type: 'SEED', items });
  }, []);

  /**
   * No batch is running (the status probe said so). Ends any membership left
   * behind by a terminal event this stream never saw — tab hidden, a reconnect
   * gap, a dropped frame — so a later detail-page run of that film is a single
   * job again (CR H1).
   */
  const endBatch = useCallback(() => {
    dispatch({ type: 'END_BATCH' });
  }, []);

  return {
    feed: state.feed,
    singleJobs: state.singleJobs,
    /** The jobs stream is open right now (onopen seen, no error since). */
    connected: state.connected,
    startTracking,
    stop,
    seedBatch,
    endBatch,
  };
}
