import type { ApiResponse } from '../types/tmdb';
import { snakeToCamel, camelToSnake } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, options);

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.error?.message || `API request failed: ${response.status}`);
  }

  const data: ApiResponse<T> = await response.json();

  if (!data.success) {
    throw new Error(data.error?.message || 'API request failed');
  }

  return snakeToCamel<T>(data.data);
}

// --- Types (camelCase frontend convention, transformed at API boundary) ---

export interface SubtitleSearchParams {
  mediaId: string;
  mediaType: 'movie' | 'series';
  providers?: string[];
  query?: string;
}

export interface SubtitleScoreBreakdown {
  language: number;
  resolution: number;
  sourceTrust: number;
  group: number;
  downloads: number;
}

export interface SubtitleSearchResult {
  id: string;
  source: string;
  filename: string;
  language: string;
  downloadUrl: string;
  downloads: number;
  group: string;
  resolution: string;
  format: string;
  score: number;
  scoreBreakdown: SubtitleScoreBreakdown;
}

export interface SubtitleDownloadParams {
  mediaId: string;
  mediaType: 'movie' | 'series';
  mediaFilePath: string;
  subtitleId: string;
  provider: string;
  resolution?: string;
  convertToTraditional?: boolean;
  score?: number;
}

export interface SubtitleDownloadResult {
  subtitlePath: string;
  language: string;
  score: number;
}

export interface SubtitlePreviewParams {
  subtitleId: string;
  provider: string;
}

export interface SubtitlePreviewResult {
  lines: string[];
  language: string;
}

// --- Batch types (Story 8-11) ---
// NOTE: contract reconciled against the ACTUAL Story 8-9 backend (Rule 20 ack):
//  - `season_id` is a STRING on the wire (subtitle_handler.go BatchStartRequest), not a number.
//  - GET /batch/status returns { running, progress? } — NOT a bare progress object.

export type BatchScope = 'library' | 'season';

export interface BatchStartParams {
  scope: BatchScope;
  /** Required when scope === 'season'. String id per backend contract. */
  seasonId?: string;
}

export interface BatchStartResult {
  batchId: string;
  totalItems: number;
}

/** Live progress shape mirrored from the `subtitle_batch_progress` SSE payload. */
export interface BatchProgress {
  batchId: string;
  totalItems: number;
  currentIndex: number;
  currentItem: string;
  successCount: number;
  failCount: number;
  status: 'running' | 'complete' | 'cancelled' | 'error';
}

/** GET /subtitles/batch/status response (camelCase). */
export interface BatchStatusResponse {
  running: boolean;
  progress?: BatchProgress;
}

/**
 * Outcome of startBatch: either the batch started (202) or one was already
 * running (409), in which case the in-progress snapshot is surfaced instead of
 * throwing (AC #7).
 */
export type StartBatchOutcome =
  | { conflict: false; result: BatchStartResult }
  | { conflict: true; progress: BatchProgress };

export interface BatchCancelResult {
  cancelled: boolean;
}

// --- Generation-batch types (Story ux3-subtitle-v2-batch, consumes 9R-16) ---
// Contract: 9R-16 AC #1 [@contract-v3] (bumped by sub-4-2: additive budget_usd
// + items[].media_type, scope=selected accepts mixed movie/episode UUIDs);
// AC #2/#3/#7/#9 stay [@contract-v2]. Media ids are UUID STRINGS end-to-end
// (movie/episode PKs are uuid.New().String(); 9R-18 ruling).

export type GenerationBatchScope = 'missing' | 'selected';

/** Internal media-type vocabulary (movie|episode — NOT the TMDB movie|tv pair). */
export type GenerationMediaType = 'movie' | 'episode';

/** One enumerated queue item from the 202 start response (`items[]`). */
export interface GenerationBatchItem {
  /** UUID string media row id on the wire ([@contract-v3]). */
  mediaId: string;
  title: string;
  /** movie|episode — additive since sub-4-2 ([@contract-v3]). */
  mediaType: GenerationMediaType;
}

export interface GenerationBatchStartParams {
  scope: GenerationBatchScope;
  /**
   * Required iff scope === 'selected'. UUID string ids — movies AND episodes
   * may be mixed since sub-4-2 (D1 ruling, [@contract-v3]). The backend still
   * REJECTS the whole request with 400 if ANY id resolves against neither
   * table or has no media file (reject-not-filter: the consented list IS the
   * confirmed amount).
   */
  mediaIds?: string[];
  /**
   * User-approved batch ceiling in USD (sub-4-2 AC #1). Must be > 0 when
   * present; absent falls back to the server-side AI_RUN_BUDGET_USD default.
   * The consent flow ALWAYS sends the on-screen value (WYSIWYG consent — the
   * number the user confirmed is the ceiling that gets enforced).
   */
  budgetUsd?: number;
  /**
   * Translation model for THIS batch (sub-6-8a AC #4, additive on the sub-4-2
   * `[@contract-v3]` envelope). Same WYSIWYG rule as `budgetUsd`: the consent
   * flow sends the id whose price the user just read, so the quote and the
   * charge can never come from different models. Absent = the deployment
   * default. An id outside `GET /settings/models` is rejected with 400 at the
   * boundary — never silently downgraded.
   */
  modelId?: string;
}

// --- Translation models (sub-6-8a AC #2 [@contract-v1], consumed by sub-6-8b) ---

/** Coarse speed/quality band a picker may group by. */
export type TranslationModelTier = 'fast' | 'balanced' | 'max';

/**
 * One selectable translation model, from `GET /api/v1/settings/models`.
 *
 * NOTE (Rule 18 boundary quirk): the wire also carries `input_per_1m` /
 * `output_per_1m`, which survive snakeToCamel as `inputPer_1m` /
 * `outputPer_1m` — the transform only upper-cases a letter after `_`, and `_1`
 * is a digit. They are deliberately NOT declared here: this screen prices a
 * batch from `estimatesByModel` (the backend's own cost model), never from
 * per-token rates it would have to re-derive.
 */
export interface TranslationModelInfo {
  id: string;
  provider: string;
  displayName: string;
  tier: TranslationModelTier;
  /**
   * The deployment's pre-selected model — stamped from the server's EFFECTIVE
   * model, so a `CLAUDE_MODEL` override drives it (not a hard-coded default).
   */
  isDefault: boolean;
  /**
   * MEASURED grade, present only for models Vido has blind-scored. Absent
   * means 尚未評測 — the UI must say so, never imply parity.
   */
  qualityGrade?: string;
  /** Provenance for the grade (which eval, which corpus). */
  qualityNote?: string;
}

/**
 * `GET /settings/models` body. `models` may legitimately be EMPTY (no AI key
 * configured) — that is a 200, not an error, and the picker simply has nothing
 * to offer.
 */
export interface TranslationModelList {
  models: TranslationModelInfo[];
  defaultModelId: string;
}

// --- Generation-candidates types (sub-4-3, consumes sub-4-1 read side) ---
// GET /subtitles/generation-candidates is ALWAYS 200 with a state envelope —
// F14 (analyzing) and F15 (list) render from the same payload; `result` is
// present only when status === 'ready'. The SSE stream carries counts only,
// never the result (fetch it via GET on the `ready` transition).

export type GenerationCandidateRoute = 'extract' | 'asr' | 'skip';

export type CandidateAnalysisStatus = 'idle' | 'analyzing' | 'ready' | 'cancelled' | 'error';

/** One analyzed candidate. `estimatedUsd` renders VERBATIM — the 2026-08-11
 * §5-sexies ruling bans a "免費" rounding presentation on the extract route. */
export interface GenerationCandidate {
  mediaId: string;
  mediaType: GenerationMediaType;
  /** Display title; episodes already carry the SxxEyy form from the backend. */
  title: string;
  route: GenerationCandidateRoute;
  runtimeMinutes: number;
  /** false → the estimate used the 45-minute fallback; UI prefixes ≈. */
  runtimeKnown: boolean;
  estimatedUsd: number;
  /**
   * sub-6-1 (additive): false when the backend's real write probe of the media
   * folder failed — the pipeline would refuse this row before spending, so it
   * is never pre-selected, its checkbox is disabled and its estimate stays out
   * of every total. `blocker` carries the zh-TW reason. Optional: a
   * pre-sub-6-1 server omits both and the row is treated as writable.
   */
  writable?: boolean;
  /** Rule 7 code (SUBTITLE_TARGET_NOT_WRITABLE); the sentence is composed here. */
  blocker?: string;
  /** Base name of the refused folder — never an absolute path. */
  blockerDir?: string;
  /**
   * Series identity (sub-5-3, additive on the sub-4-1 [@contract-v1] envelope)
   * — episodes only; absent/empty on movies and on pre-sub-5-3 servers.
   * Group by `seriesId`, NEVER by season: S00 specials are a legal season 0.
   * seriesTitle degrades to '' when the backend lookup failed (未知影集).
   */
  seriesId?: string;
  seriesTitle?: string;
  seasonNumber?: number;
  episodeNumber?: number;
  /**
   * Row identity (sub-6-10a, additive on the same [@contract-v1] envelope).
   * All four are absent on a pre-sub-6-10a server, so every consumer must
   * treat them as optional and fall back to what it rendered before.
   *
   * `runtimeSource` says which rung of the ladder produced `runtimeMinutes`:
   * 'ffprobe' (measured from the file), 'tmdb' (the published cut) or
   * 'fallback' (the 45-minute assumption — `runtimeKnown` is false).
   */
  runtimeSource?: 'ffprobe' | 'tmdb' | 'fallback';
  /**
   * TMDb poster path — compose the CDN URL as elsewhere. An EPISODE carries
   * its SERIES' poster (its own still is a frame grab, not an identity).
   * Empty/absent → draw an initial, never a broken image.
   */
  posterPath?: string;
  /**
   * false → this row's identity did not come from TMDb, so `title` is derived
   * from the filename. The UI marks it unverified rather than presenting a
   * scene-release string as a film.
   */
  tmdbMatched?: boolean;
  /**
   * What the row should READ as: the TMDb title when matched, otherwise the
   * filename parser's cleaned-up guess. `title` keeps its old meaning, so an
   * older client renders exactly what it rendered before.
   */
  displayTitle?: string;
}

export interface GenerationCandidateSummary {
  extractCount: number;
  asrCount: number;
  skippedCount: number;
  estimatedTotalUsd: number;
  selfHostedAsr: boolean;
  /** sub-6-1 (additive): rows listed with writable=false; absent on old servers. */
  unwritableCount?: number;
}

/**
 * What one model would cost for this sweep (sub-6-8a AC #3). `perCandidate`
 * (keyed by media id) lets the picker re-price every visible row when the user
 * switches model — without a second sweep and without the FE re-implementing
 * the backend's cost model.
 */
export interface ModelEstimate {
  totalUsd: number;
  /**
   * WRITABLE candidates only — the same set `totalUsd` sums, so the visible
   * rows add up to the footer figure. An unwritable row is absent (the
   * pipeline would refuse it before spending).
   */
  perCandidate?: Record<string, number>;
}

export interface GenerationCandidateResult {
  candidates: GenerationCandidate[];
  summary: GenerationCandidateSummary;
  /**
   * sub-6-8a AC #3 (additive): this sweep priced under every model the
   * deployment can actually run, keyed by model id. Absent on a pre-sub-6-8a
   * server — the picker then falls back to a single default-model row priced
   * from `candidate.estimatedUsd`, which is what those servers quote.
   *
   * It lives on the RESULT, not on the snapshot, because it IS the quote: a
   * cancelled or failed sweep clears `result` and these numbers vanish with
   * it. A stale price is the one thing this screen must never show.
   */
  estimatesByModel?: Record<string, ModelEstimate>;
  /** Wall-clock cost of the same choice, in whole minutes, keyed by model id. */
  estimatedMinutesByModel?: Record<string, number>;
}

/** The GET state envelope. */
export interface CandidateAnalysisSnapshot {
  status: CandidateAnalysisStatus;
  analyzed: number;
  total: number;
  result?: GenerationCandidateResult;
  analyzedAt?: string;
  error?: string;
  /**
   * Operator-configured AI_RUN_BUDGET_USD (sub-5-1 AC #5/#6) — the F15
   * budget-input PREFILL source, present on every snapshot state. Optional:
   * a pre-sub-5-1 server omits it and the FE falls back to its constant.
   * WYSIWYG consent is unchanged — the value SENT is always the on-screen one.
   */
  defaultBudgetUsd?: number;
}

/**
 * Outcome of startCandidateAnalysis: 202 started, or 409
 * TRANSCRIPTION_ANALYSIS_RUNNING — an analysis already running is a JOIN, not
 * an error (discriminated union instead of throwing; transcriptionService
 * precedent). Other non-2xx throw.
 */
export type StartCandidateAnalysisOutcome = { started: true } | { alreadyRunning: true };

/**
 * 202 start response — `batchId` is null on the empty-missing-scope 200
 * (`{total_items: 0, items: []}` — nothing to do is not an error).
 */
export interface GenerationBatchStartResult {
  batchId: string | null;
  totalItems: number;
  items: GenerationBatchItem[];
}

export type GenerationBatchStatus =
  | 'running'
  | 'complete'
  | 'cancelled'
  | 'error'
  | 'budget_ceiling';

/** Progress snapshot — mirrors the `generation_batch_progress` SSE payload (11 keys). */
export interface GenerationBatchProgress {
  batchId: string;
  totalItems: number;
  currentIndex: number;
  /** UUID string movie id of the in-flight item — joins UI rows directly ([@contract-v2]). */
  currentMediaId: string;
  currentItem: string;
  successCount: number;
  failCount: number;
  pausedCount: number;
  status: GenerationBatchStatus;
  spentUsd: number;
  budgetUsd: number;
}

/**
 * GET /subtitles/generation-batch/status response. NOTE (fetch-batch parity):
 * after ANY terminal state this probe returns `{running: false, progress: null}`
 * — terminal snapshots (incl. budget_ceiling counts) arrive only via SSE.
 */
export interface GenerationBatchStatusResponse {
  running: boolean;
  progress?: GenerationBatchProgress | null;
}

export interface GenerationBatchCancelResult {
  cancelled: boolean;
  running: boolean;
}

/** GET /subtitles/generation-batch/preview?scope=missing — the F8 缺字幕 count. */
export interface GenerationBatchPreviewResult {
  /** Movies-only — semantics FROZEN (what scope=missing actually runs). */
  totalItems: number;
  /**
   * Library-wide missing count INCLUDING episodes (sub-5-1 AC #7 additive
   * key) — what the consent list will actually show, so the F17 toast reads
   * this. Optional: absent on a pre-sub-5-1 server (fall back to totalItems).
   */
  totalItemsIncludingEpisodes?: number;
}

/**
 * Outcome of startGenerationBatch: started (202 / empty 200), or a batch was
 * already running (409 TRANSCRIPTION_BATCH_RUNNING → progress rides the error
 * body, recover-and-attach instead of throwing).
 */
export type StartGenerationBatchOutcome =
  | { conflict: false; result: GenerationBatchStartResult }
  | { conflict: true; progress: GenerationBatchProgress };

// --- Service ---

export const subtitleService = {
  async searchSubtitles(params: SubtitleSearchParams): Promise<SubtitleSearchResult[]> {
    return fetchApi<SubtitleSearchResult[]>('/subtitles/search', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(camelToSnake(params)),
    });
  },

  async downloadSubtitle(params: SubtitleDownloadParams): Promise<SubtitleDownloadResult> {
    return fetchApi<SubtitleDownloadResult>('/subtitles/download', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(camelToSnake(params)),
    });
  },

  async previewSubtitle(params: SubtitlePreviewParams): Promise<SubtitlePreviewResult> {
    return fetchApi<SubtitlePreviewResult>('/subtitles/preview', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(camelToSnake(params)),
    });
  },

  // --- Batch (Story 8-11, consumes Story 8-9 backend) ---

  /**
   * POST /subtitles/batch. Returns the started batch on 202, or the in-progress
   * snapshot on 409 (AC #7) — never throws on a conflict. Other non-2xx throw.
   */
  async startBatch(params: BatchStartParams): Promise<StartBatchOutcome> {
    const response = await fetch(`${API_BASE_URL}/subtitles/batch`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(camelToSnake(params)),
    });

    const json = await response.json().catch(() => ({}) as Record<string, unknown>);

    if (response.status === 409) {
      return {
        conflict: true,
        progress: snakeToCamel<BatchProgress>((json as ApiResponse<unknown>).data),
      };
    }

    if (!response.ok || !(json as ApiResponse<unknown>).success) {
      const err = (json as ApiResponse<unknown>).error;
      throw new Error(err?.message || `API request failed: ${response.status}`);
    }

    return {
      conflict: false,
      result: snakeToCamel<BatchStartResult>((json as ApiResponse<unknown>).data),
    };
  },

  /** GET /subtitles/batch/status — current batch status (AC #7 recovery path). */
  async getBatchStatus(): Promise<BatchStatusResponse> {
    return fetchApi<BatchStatusResponse>('/subtitles/batch/status');
  },

  /** POST /subtitles/batch/cancel — stops the active batch (AC #5). Idempotent. */
  async cancelBatch(): Promise<BatchCancelResult> {
    return fetchApi<BatchCancelResult>('/subtitles/batch/cancel', {
      method: 'POST',
    });
  },

  // --- Generation batch (Story ux3-subtitle-v2-batch, consumes 9R-16) ---

  /**
   * POST /subtitles/generation-batch. 202 → started ({batch_id, total_items,
   * items[]}); empty missing scope → 200 {total_items: 0, items: []} (batchId
   * null); 409 TRANSCRIPTION_BATCH_RUNNING → in-progress snapshot from the
   * error body (never throws on conflict). Other non-2xx throw.
   */
  async startGenerationBatch(
    params: GenerationBatchStartParams
  ): Promise<StartGenerationBatchOutcome> {
    const response = await fetch(`${API_BASE_URL}/subtitles/generation-batch`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(camelToSnake(params)),
    });

    const json = await response.json().catch(() => ({}) as Record<string, unknown>);

    if (response.status === 409) {
      return {
        conflict: true,
        progress: snakeToCamel<GenerationBatchProgress>((json as ApiResponse<unknown>).data),
      };
    }

    if (!response.ok || !(json as ApiResponse<unknown>).success) {
      const err = (json as ApiResponse<unknown>).error;
      throw new Error(err?.message || `API request failed: ${response.status}`);
    }

    const data = snakeToCamel<{
      batchId?: string;
      totalItems: number;
      items: GenerationBatchItem[];
    }>((json as ApiResponse<unknown>).data);
    return {
      conflict: false,
      result: {
        batchId: data.batchId ?? null,
        totalItems: data.totalItems,
        items: data.items ?? [],
      },
    };
  },

  /**
   * GET /settings/models — the translation models this deployment can run
   * (sub-6-8a AC #2 `[@contract-v1]`). Lives here rather than in
   * keySettingsService because its only consumer is the generation consent
   * flow: the settings page and the confirm dialog ask the same question and
   * must get the same answer.
   */
  async getModels(): Promise<TranslationModelList> {
    return fetchApi<TranslationModelList>('/settings/models');
  },

  /** GET /subtitles/generation-batch/status — on-open recovery probe. */
  async getGenerationBatchStatus(): Promise<GenerationBatchStatusResponse> {
    return fetchApi<GenerationBatchStatusResponse>('/subtitles/generation-batch/status');
  },

  /** POST /subtitles/generation-batch/cancel — idempotent; queued items never start. */
  async cancelGenerationBatch(): Promise<GenerationBatchCancelResult> {
    return fetchApi<GenerationBatchCancelResult>('/subtitles/generation-batch/cancel', {
      method: 'POST',
    });
  },

  /** GET /subtitles/generation-batch/preview?scope=missing — the F8 idle count. */
  async previewGenerationBatch(): Promise<GenerationBatchPreviewResult> {
    return fetchApi<GenerationBatchPreviewResult>(
      '/subtitles/generation-batch/preview?scope=missing'
    );
  },

  // --- Generation candidates (sub-4-3, consumes sub-4-1) ---

  /** GET /subtitles/generation-candidates — always 200 with the state envelope. */
  async getGenerationCandidates(): Promise<CandidateAnalysisSnapshot> {
    return fetchApi<CandidateAnalysisSnapshot>('/subtitles/generation-candidates');
  },

  /**
   * POST /subtitles/generation-candidates/analyze. 202 → started; 409
   * TRANSCRIPTION_ANALYSIS_RUNNING → join the running analysis (not an
   * error, never throws on that pair). Other non-2xx throw.
   */
  async startCandidateAnalysis(): Promise<StartCandidateAnalysisOutcome> {
    const response = await fetch(`${API_BASE_URL}/subtitles/generation-candidates/analyze`, {
      method: 'POST',
    });
    const json = await response.json().catch(() => ({}) as Record<string, unknown>);
    const envelope = json as ApiResponse<unknown>;

    if (response.status === 409 && envelope.error?.code === 'TRANSCRIPTION_ANALYSIS_RUNNING') {
      return { alreadyRunning: true };
    }
    if (!response.ok || !envelope.success) {
      throw new Error(envelope.error?.message || `API request failed: ${response.status}`);
    }
    return { started: true };
  },

  /** POST /subtitles/generation-candidates/analyze/cancel — idempotent. */
  async cancelCandidateAnalysis(): Promise<{ cancelled: boolean }> {
    return fetchApi<{ cancelled: boolean }>('/subtitles/generation-candidates/analyze/cancel', {
      method: 'POST',
    });
  },
};
