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
 *       · a row stops being `live` when its film moves on, ends, when the batch
 *         ends, or when the stream reopens (events in the gap are gone for
 *         good — the hub has no replay — so nothing may keep claiming it runs);
 *       · a BATCH item's result comes only from the batch's `changed_item`.
 *         Its own `transcription_failed` arrives BEFORE the batch pauses it at
 *         the budget ceiling or cancels it, so trusting that event would log a
 *         paused film as failed while the queue beside it says paused;
 *       · `subtitle_progress` is only read for batch members — the subtitle
 *         search engine and the manual download reuse that event name, in
 *         English, for work that is not generation.
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
      /** Still the film's current stage (accent + spinner) vs a step it has passed. */
      live: boolean;
      /** Translation only; null once the row is no longer live. */
      percentage: number | null;
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
  /** Films in the batch that is running now; cleared at its terminal. */
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
  | { type: 'PIPELINE_END'; mediaId: string }
  | { type: 'BATCH'; payload: BatchPayload }
  | { type: 'SEED'; items: GenerationBatchItemState[] };

/** A backend title that is really the media id is no title at all. */
function honestTitle(title: string | undefined, mediaId: string): string {
  return title && title !== mediaId ? title : '';
}

/** Append rows, assigning seqs and enforcing FEED_CAP (drop-oldest). */
function append(
  state: GenerationJobsFeedState,
  feed: FeedRow[],
  rows: Array<DistributiveOmit<FeedRow, 'seq'>>
): { feed: FeedRow[]; seq: number } {
  let seq = state.seq;
  const next = [...feed];
  for (const row of rows) {
    seq += 1;
    next.push({ ...row, seq } as FeedRow);
  }
  return { feed: next.length > FEED_CAP ? next.slice(next.length - FEED_CAP) : next, seq };
}

type DistributiveOmit<T, K extends keyof T> = T extends unknown ? Omit<T, K> : never;

/** Every live row matching `which` stops claiming to run. */
function demote(feed: FeedRow[], which: (row: FeedRow) => boolean): FeedRow[] {
  let changed = false;
  const next = feed.map((row) => {
    if (row.kind === 'stage' && row.live && which(row)) {
      changed = true;
      return { ...row, live: false, percentage: null };
    }
    return row;
  });
  return changed ? next : feed;
}

const forFilm = (mediaId: string) => (row: FeedRow) =>
  row.kind !== 'batch' && row.mediaId === mediaId;

/** The film's current live row, if any. */
function liveRow(feed: FeedRow[], mediaId: string) {
  for (let i = feed.length - 1; i >= 0; i -= 1) {
    const row = feed[i];
    if (row.kind === 'stage' && row.live && row.mediaId === mediaId) return { row, index: i };
  }
  return null;
}

/** Fill the title of a film's earlier rows that were written before it was known. */
function backfill(feed: FeedRow[], mediaId: string, info: MemberInfo): FeedRow[] {
  if (!info.title) return feed;
  let changed = false;
  const next = feed.map((row) => {
    if (row.kind !== 'batch' && row.mediaId === mediaId && !row.title) {
      changed = true;
      return { ...row, title: info.title, seriesTitle: info.seriesTitle };
    }
    return row;
  });
  return changed ? next : feed;
}

function withoutKey<T>(record: Record<string, T>, key: string): Record<string, T> {
  if (!(key in record)) return record;
  const next = { ...record };
  delete next[key];
  return next;
}

function reducer(state: GenerationJobsFeedState, action: Action): GenerationJobsFeedState {
  switch (action.type) {
    case 'OPEN':
      // (Re)opened: whatever was live before cannot be vouched for any more.
      return { ...state, connected: true, feed: demote(state.feed, () => true) };

    case 'CLOSED':
      return state.connected ? { ...state, connected: false } : state;

    case 'STAGE': {
      const { stage, mediaId, payload, pipeline } = action;
      const member = state.members[mediaId];
      if (pipeline && !member) return state; // not generation, or not ours
      const title = member ? member.title : honestTitle(payload.title, mediaId);
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
              title: title || state.singleJobs[mediaId]?.title || '',
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
      const demoted = demote(state.feed, forFilm(mediaId));
      const next = append(state, demoted, [
        { kind: 'stage', mediaId, title, seriesTitle, stage, live: true, percentage },
      ]);
      return { ...state, ...next, singleJobs };
    }

    case 'SINGLE_DONE':
    case 'SINGLE_FAILED': {
      const mediaId = action.payload.mediaId;
      if (!mediaId) return state;
      const demoted = demote(state.feed, forFilm(mediaId));
      if (state.members[mediaId]) {
        // A batch member: its result comes from changed_item. Remember why it
        // failed in case the batch only says `error`.
        const memberErrors =
          action.type === 'SINGLE_FAILED' && action.payload.error
            ? { ...state.memberErrors, [mediaId]: action.payload.error }
            : state.memberErrors;
        return { ...state, feed: demoted, memberErrors };
      }
      const title =
        honestTitle(action.payload.title, mediaId) || state.singleJobs[mediaId]?.title || '';
      const row: DistributiveOmit<FeedRow, 'seq'> =
        action.type === 'SINGLE_DONE'
          ? { kind: 'done', mediaId, title, seriesTitle: '' }
          : {
              kind: 'failed',
              mediaId,
              title,
              seriesTitle: '',
              reason: null,
              error: action.payload.error ?? action.payload.message ?? null,
            };
      const next = append(state, demoted, [row]);
      return { ...state, ...next, singleJobs: withoutKey(state.singleJobs, mediaId) };
    }

    case 'PIPELINE_END': {
      if (!state.members[action.mediaId]) return state;
      return { ...state, feed: demote(state.feed, forFilm(action.mediaId)) };
    }

    case 'SEED': {
      let members = state.members;
      let feed = state.feed;
      let singleJobs = state.singleJobs;
      for (const it of action.items) {
        const info = { title: it.title, seriesTitle: it.seriesTitle ?? '' };
        members = { ...members, [it.mediaId]: info };
        feed = backfill(feed, it.mediaId, info);
        singleJobs = withoutKey(singleJobs, it.mediaId);
      }
      return { ...state, members, feed, singleJobs };
    }

    case 'BATCH': {
      const { payload } = action;
      const status = payload.status ?? '';
      const batchId = payload.batchId ?? '';
      let { feed, members, singleJobs, memberErrors } = state;
      let seqState = state;

      const changed = payload.changedItem;
      if (changed?.mediaId) {
        const id = changed.mediaId;
        const info = { title: changed.title ?? '', seriesTitle: changed.seriesTitle ?? '' };
        members = { ...members, [id]: info };
        feed = backfill(feed, id, info);
        singleJobs = withoutKey(singleJobs, id);
        if (changed.status === 'done' || changed.status === 'failed') {
          feed = demote(feed, forFilm(id));
          const row: DistributiveOmit<FeedRow, 'seq'> =
            changed.status === 'done'
              ? { kind: 'done', mediaId: id, ...info }
              : {
                  kind: 'failed',
                  mediaId: id,
                  ...info,
                  reason: changed.reason,
                  error: changed.reason === 'error' ? (memberErrors[id] ?? null) : null,
                };
          const next = append(seqState, feed, [row]);
          feed = next.feed;
          seqState = { ...seqState, seq: next.seq };
          memberErrors = withoutKey(memberErrors, id);
        }
      }

      let batchRowsSeen = state.batchRowsSeen;
      const key = `${batchId}:${status}`;
      if (BATCH_TERMINALS.has(status) && !batchRowsSeen.includes(key)) {
        feed = demote(feed, (row) => row.kind !== 'batch' && row.mediaId in members);
        const next = append(seqState, feed, [
          {
            kind: 'batch',
            batchId,
            status: status as BatchTerminalStatus,
            successCount: payload.successCount ?? 0,
            failCount: payload.failCount ?? 0,
            budgetUsd: payload.budgetUsd ?? 0,
          },
        ]);
        feed = next.feed;
        seqState = { ...seqState, seq: next.seq };
        batchRowsSeen = [...batchRowsSeen.slice(-19), key];
        members = {};
        memberErrors = {};
      }

      return {
        ...state,
        feed,
        seq: seqState.seq,
        members,
        singleJobs,
        memberErrors,
        batchRowsSeen,
      };
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
        dispatch({ type: 'PIPELINE_END', mediaId: payload.mediaId });
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
   * Tell the log which films belong to the batch that is running now (the
   * status probe's `progress.items`). Needed when the page opens mid-batch: the
   * running item's events would otherwise count as a single job, untitled.
   */
  const seedBatch = useCallback((_batchId: string, items: GenerationBatchItemState[]) => {
    dispatch({ type: 'SEED', items });
  }, []);

  return {
    feed: state.feed,
    singleJobs: state.singleJobs,
    /** The jobs stream is open right now (onopen seen, no error since). */
    connected: state.connected,
    startTracking,
    stop,
    seedBatch,
  };
}
