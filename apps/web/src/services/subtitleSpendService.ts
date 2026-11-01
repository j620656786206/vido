/**
 * Monthly subtitle AI spend client (sub-7-6c) for GET /api/v1/subtitles/spend
 * (sub-7-6b [@contract-v1]). The backend speaks snake_case; this boundary
 * camelCases it (Rule 18), exactly like activityService.
 *
 * Honesty markers the UI must carry through, never round away:
 *  - `unpricedRuns`: completed runs with no recorded amount — counted apart,
 *    never shown as $0;
 *  - `unroutedRuns` / `unroutedUsd`: rows older than the ledger columns
 *    (sub-7-6a). Real money, kept out of the translated bucket because nobody
 *    recorded WHICH lane they ran on;
 *  - `cacheSavedUsdEstimate` is null (not 0) when no run measured its split;
 *  - `skippedSavedRuntimeAssumed`: at least one skipped item's length was a
 *    guess, so that saving wears `≈` — the ONE meaning of ≈ in this product
 *    (lib/currency.ts usdWithEstimate).
 *  - `parseCalls` / `parseUsd` (backlog-ai-parse-reads-env-keys-only-a/-b):
 *    paid AI calls made OUTSIDE subtitle runs — filename parsing, fansub
 *    parsing and keyword generation together. Not part of `byModel` (that
 *    table counts subtitle runs). An older API that predates the two fields
 *    reads as 0 calls, not undefined.
 *  - `burnedInCalls` / `burnedInUsd` (disc-2026-11-burned-in-subs-b2): the
 *    burned-in subtitle checks (Claude reading an episode's frames), listed
 *    apart from parsing — `parseCalls` no longer includes them. An older API
 *    reads as 0 the same way.
 */
import { snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export interface SpendByModel {
  modelId: string;
  runs: number;
  usd: number;
}

/**
 * One consent batch's receipt (`by_batch`, present only when the request named
 * a `batch_id`). `usd` counts every run — a failed or budget-paused run still
 * spent its money; `cueCount` / `cacheHitCues` count COMPLETED runs only.
 * `cacheHitCues` is null when no run of the batch measured its cache split.
 * `modelIds` is present only when the batch changed model midway.
 */
export interface SubtitleBatchReceipt {
  batchId: string;
  runs: number;
  completedRuns: number;
  usd: number;
  cueCount: number;
  cacheHitCues: number | null;
  cacheMeasuredRuns: number;
  modelId: string;
  modelIds?: string[];
}

export interface SubtitleSpendSummary {
  period: 'month' | string;
  /** Start of the server's calendar month, RFC3339 in the server zone. */
  from: string;
  /** Start of the next month (exclusive). */
  to: string;
  translatedRuns: number;
  translatedUsd: number;
  asrRuns: number;
  asrUsd: number;
  unpricedRuns: number;
  unroutedRuns: number;
  unroutedUsd: number;
  skippedDeliverCount: number;
  skippedSavedUsdEstimate: number;
  skippedSavedRuntimeAssumed: boolean;
  cacheHitCues: number;
  cacheMeasuredRuns: number;
  cacheSavedUsdEstimate: number | null;
  byModel: SpendByModel[];
  byBatch?: SubtitleBatchReceipt;
  /** Paid filename-parsing AI calls this month (0 when none, or an older API). */
  parseCalls: number;
  /** What those calls cost, USD. */
  parseUsd: number;
  /** Paid burned-in subtitle checks this month (0 when none, or an older API). */
  burnedInCalls: number;
  /** What those checks cost, USD. */
  burnedInUsd: number;
}

/** The fields an older API may omit; each reads as 0. */
type OptionalSpendFields = 'parseCalls' | 'parseUsd' | 'burnedInCalls' | 'burnedInUsd';

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: { code: string; message: string };
}

async function fetchApi<T>(endpoint: string): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    headers: { 'Content-Type': 'application/json' },
  });

  const data: ApiResponse<T> = await response.json();

  if (!response.ok || !data.success) {
    throw new Error(data.error?.message || `API request failed: ${response.status}`);
  }
  if (data.data === undefined) {
    throw new Error('API response missing data field');
  }

  return snakeToCamel(data.data);
}

export const subtitleSpendService = {
  /**
   * This calendar month's spend. With `batchId`, the response also carries
   * `byBatch` — that batch's receipt, whatever month it ran in.
   */
  async getMonthSummary(batchId?: string): Promise<SubtitleSpendSummary> {
    const params = new URLSearchParams({ period: 'month' });
    if (batchId) params.set('batch_id', batchId);
    const summary = await fetchApi<
      Omit<SubtitleSpendSummary, OptionalSpendFields> &
        Partial<Pick<SubtitleSpendSummary, OptionalSpendFields>>
    >(`/subtitles/spend?${params.toString()}`);
    // An API older than backlog-ai-parse-reads-env-keys-only-a (parse_*) or
    // disc-2026-11-burned-in-subs-b2 (burned_in_*) omits those fields: that is
    // "no such calls recorded", i.e. 0.
    return {
      ...summary,
      parseCalls: summary.parseCalls ?? 0,
      parseUsd: summary.parseUsd ?? 0,
      burnedInCalls: summary.burnedInCalls ?? 0,
      burnedInUsd: summary.burnedInUsd ?? 0,
    };
  },
};

export default subtitleSpendService;
