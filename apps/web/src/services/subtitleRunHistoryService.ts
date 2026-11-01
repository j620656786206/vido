/**
 * Per-run subtitle spend history client (disc-2026-11-spend-history-d) for
 * GET /api/v1/subtitles/runs (disc-2026-11-spend-history-b [@contract-v1]).
 * The backend speaks snake_case; this boundary camelCases it (Rule 18),
 * exactly like subtitleSpendService.
 *
 * Honesty markers the UI must carry through, never round away:
 *  - every amount that was not recorded is `null`, not 0 — a run's
 *    `spentUsd` / `estimateUsd`, and the `comparable*` totals when no run had
 *    both (PRODUCT.md 原則 1);
 *  - `summary.unpricedRuns` counts runs whose spend is null;
 *  - failed and cancelled runs are in `items`, `summary` and `series` — they
 *    spent money too.
 *
 * The month groups runs by START time (the API's rule), which is why its total
 * can differ from the hub card's `/subtitles/spend` (completion time,
 * completed runs only).
 */
import { snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

/** The five buckets of the -c design table; `no_subtitle` never enters the chart. */
export type RunCategory = 'chinese_track' | 'translate_english' | 'asr' | 'legacy' | 'no_subtitle';

/** The four categories the chart stacks (and the chips filter by). */
export type ChartCategory = Exclude<RunCategory, 'no_subtitle'>;

export interface RunHistorySummary {
  spentUsd: number;
  runs: number;
  unpricedRuns: number;
  comparableRuns: number;
  comparableSpentUsd: number | null;
  comparableEstimateUsd: number | null;
}

export interface RunHistoryDay {
  /** `YYYY-MM-DD` in the server zone. */
  date: string;
  spentUsd: number;
  runs: number;
  byCategory: Record<ChartCategory, number>;
}

export interface RunHistoryMedia {
  id: string;
  type: 'movie' | 'episode' | string;
  /** null when the media was deleted. For an episode, the series title. */
  title: string | null;
  season: number | null;
  episode: number | null;
}

export interface RunHistoryItem {
  id: string;
  startedAt: string;
  completedAt: string | null;
  status: 'completed' | 'failed' | 'skipped' | string;
  failureReason: string;
  media: RunHistoryMedia;
  route: string | null;
  source: string | null;
  category: RunCategory;
  spentUsd: number | null;
  estimateUsd: number | null;
  translationModel: string | null;
  asrProvider: string | null;
  asrModel: string | null;
  batchId: string | null;
  triggeredBy: string | null;
}

export interface RunHistoryPage {
  /** `YYYY-MM` — the month the server aggregated. */
  month: string;
  summary: RunHistorySummary;
  series: RunHistoryDay[];
  items: RunHistoryItem[];
  page: number;
  pageSize: number;
  total: number;
}

export interface RunHistoryQuery {
  /** `YYYY-MM`; omitted = the server's current month. */
  month?: string;
  /** Filters `items` only — `summary` and `series` are the whole month. */
  category?: ChartCategory;
  page?: number;
  pageSize?: number;
}

/**
 * snakeToCamel rewrites every KEY — including `by_category`'s, which are
 * category enum values (`chinese_track` → `chineseTrack`). The boundary puts
 * them back, so `byCategory[item.category]` and the chips use one vocabulary.
 */
type WireRunHistoryPage = Omit<RunHistoryPage, 'series'> & {
  series: Array<
    Omit<RunHistoryDay, 'byCategory'> & {
      byCategory: { chineseTrack: number; translateEnglish: number; asr: number; legacy: number };
    }
  >;
};

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

export const subtitleRunHistoryService = {
  async getRuns(query: RunHistoryQuery = {}): Promise<RunHistoryPage> {
    const params = new URLSearchParams();
    if (query.month) params.set('month', query.month);
    if (query.category) params.set('category', query.category);
    if (query.page) params.set('page', String(query.page));
    if (query.pageSize) params.set('page_size', String(query.pageSize));
    const qs = params.toString();
    const raw = await fetchApi<WireRunHistoryPage>(`/subtitles/runs${qs ? `?${qs}` : ''}`);
    return {
      ...raw,
      series: (raw.series ?? []).map((d) => ({
        ...d,
        byCategory: {
          chinese_track: d.byCategory?.chineseTrack ?? 0,
          translate_english: d.byCategory?.translateEnglish ?? 0,
          asr: d.byCategory?.asr ?? 0,
          legacy: d.byCategory?.legacy ?? 0,
        },
      })),
      items: raw.items ?? [],
    };
  },
};

export default subtitleRunHistoryService;
