// Implements: <utility — no .pen counterpart>
/**
 * Words and small rules for the AI 花費明細 page (disc-2026-11-spend-history-d),
 * straight from Sally's design table in disc-2026-11-spend-history-c-design.md.
 *
 * - 「語音辨識」, never 聽寫 — the activity hub's SpendCard already says it.
 * - Anything not recorded reads 「沒記錄」; a real $0 still reads $0.00
 *   (PRODUCT.md 原則 1).
 * - ⚠ only when the actual is MORE than 1.5× the estimate, decided with exact
 *   decimal arithmetic (lib/currency), never a float compare.
 */
import { divideUsd, gtUsd, percentOfUsd, subUsd } from '../../lib/currency';
import type {
  ChartCategory,
  RunHistoryItem,
  RunHistorySummary,
} from '../../services/subtitleRunHistoryService';
import { creditProviderName } from '../subtitle/creditStop';
import { failureCopyForReason } from '../subtitle/runFailureCopy';

/** Chart order, bottom of the stack first — also the chip and legend order. */
export const CHART_CATEGORIES: readonly ChartCategory[] = [
  'chinese_track',
  'translate_english',
  'asr',
  'legacy',
] as const;

export const CATEGORY_LABEL: Readonly<Record<ChartCategory, string>> = {
  chinese_track: '片內中文字幕',
  translate_english: '翻譯英文字幕',
  asr: '語音辨識＋翻譯',
  legacy: '舊紀錄',
};

/**
 * One colour per chart category, existing tokens only. 語音辨識 wears the
 * accent (the biggest spender), 翻譯 靛青, the near-free 片內字幕 青碧 and the
 * unclassified 舊紀錄 a neutral grey so it never reads as a lane of its own.
 * `--warning` stays free for the ⚠ over-estimate mark.
 */
export const CATEGORY_FILL: Readonly<Record<ChartCategory, string>> = {
  chinese_track: 'fill-[var(--success)]',
  translate_english: 'fill-[var(--info)]',
  asr: 'fill-[var(--accent-primary)]',
  legacy: 'fill-[var(--text-muted)]',
};

export const CATEGORY_SWATCH: Readonly<Record<ChartCategory, string>> = {
  chinese_track: 'bg-[var(--success)]',
  translate_english: 'bg-[var(--info)]',
  asr: 'bg-[var(--accent-primary)]',
  legacy: 'bg-[var(--text-muted)]',
};

/** The list's 處理方式 words (the -c table, route＋source → words). */
export function routeLabel(item: Pick<RunHistoryItem, 'route' | 'source' | 'category'>): string {
  switch (item.route) {
    case 'deliver_direct':
      return '直接用片內繁中';
    case 'convert_then_deliver':
      return '片內簡中轉繁體';
    case 'translate':
      if (item.source === 'embedded_text') return '翻譯片內英文字幕';
      if (item.source === 'sidecar') return '翻譯片旁英文字幕';
      return '翻譯英文字幕';
    case 'asr':
      return '語音辨識＋翻譯';
    case 'no_text_source':
    case 'skip':
      return '沒有可用字幕';
    default:
      // Route not recorded (September's rows) — or one this build does not know.
      return item.category === 'no_subtitle' ? '沒有可用字幕' : '舊紀錄（沒分類）';
  }
}

export interface RunStatusView {
  label: string;
  /** Token class for the label. */
  tone: string;
  /** The failure reason in words, when there is one worth showing. */
  detail: string | null;
}

/**
 * 完成／失敗／中途停止／略過. A cancelled run is `failed` on the wire with
 * reason `cancelled` — the design names it 中途停止 on its own.
 */
export function runStatus(item: Pick<RunHistoryItem, 'status' | 'failureReason'>): RunStatusView {
  if (item.status === 'completed') {
    return { label: '完成', tone: 'text-[var(--text-secondary)]', detail: null };
  }
  if (item.status === 'skipped') {
    return { label: '略過', tone: 'text-[var(--text-muted)]', detail: null };
  }
  if (item.failureReason === 'cancelled') {
    return { label: '中途停止', tone: 'text-[var(--warning-text)]', detail: null };
  }
  return {
    label: '失敗',
    tone: 'text-[var(--error-text)]',
    detail: item.failureReason ? failureCopyForReason(item.failureReason).text : null,
  };
}

/** 「Lioness S1E1」 for an episode; null when the media was deleted. */
export function mediaTitle(item: Pick<RunHistoryItem, 'media'>): string | null {
  const { title, type, season, episode } = item.media;
  if (!title) return null;
  if (type === 'episode' && season != null && episode != null) {
    return `${title} S${season}E${episode}`;
  }
  return title;
}

/**
 * The models a run used: the translation model, then the ASR engine and model
 * (「ElevenLabs scribe_v2」). Empty when neither was recorded — the caller
 * renders 「—」.
 */
export function runModels(
  item: Pick<RunHistoryItem, 'translationModel' | 'asrProvider' | 'asrModel'>
): string[] {
  const out: string[] = [];
  if (item.translationModel) out.push(item.translationModel);
  if (item.asrProvider || item.asrModel) {
    const engine = item.asrProvider ? creditProviderName(item.asrProvider) : '';
    out.push(
      [engine === 'AI' ? item.asrProvider : engine, item.asrModel].filter(Boolean).join(' ')
    );
  }
  return out;
}

/**
 * How far over the estimate a run went, as a whole percent — only when the
 * actual is MORE than 1.5× the estimate (「比預估高 N%」). undefined otherwise,
 * including when either side was not recorded or the estimate is 0.
 */
export function overEstimatePercent(
  item: Pick<RunHistoryItem, 'spentUsd' | 'estimateUsd'>
): number | undefined {
  const { spentUsd, estimateUsd } = item;
  if (spentUsd == null || estimateUsd == null || !gtUsd(estimateUsd, 0)) return undefined;
  // actual − estimate > estimate / 2  ⇔  actual > 1.5 × estimate, exactly.
  const over = subUsd(spentUsd, estimateUsd);
  if (!gtUsd(over, divideUsd(estimateUsd, 2))) return undefined;
  return percentOfUsd(over, estimateUsd);
}

/**
 * 預估準度: 「實際是預估的 N%」 over the runs that recorded both, or undefined
 * when there is nothing to compare (the caller writes 「還沒有可比的紀錄」).
 */
export function estimateAccuracyPercent(summary: RunHistorySummary): number | undefined {
  const { comparableRuns, comparableSpentUsd, comparableEstimateUsd } = summary;
  if (comparableRuns <= 0 || comparableSpentUsd == null || comparableEstimateUsd == null) {
    return undefined;
  }
  return percentOfUsd(comparableSpentUsd, comparableEstimateUsd);
}

const MONTH_RE = /^(\d{4})-(0[1-9]|1[0-2])$/;

/** Whether a string is a `YYYY-MM` month the API accepts. */
export function isMonth(value: unknown): value is string {
  return typeof value === 'string' && MONTH_RE.test(value);
}

/** `YYYY-MM` shifted by `delta` months. */
export function shiftMonth(month: string, delta: number): string {
  const m = MONTH_RE.exec(month);
  if (!m) return month;
  const index = Number(m[1]) * 12 + (Number(m[2]) - 1) + delta;
  const year = Math.floor(index / 12);
  return `${year}-${String((index % 12) + 1).padStart(2, '0')}`;
}

/** 「2026 年 11 月」. */
export function monthLabel(month: string): string {
  const m = MONTH_RE.exec(month);
  return m ? `${m[1]} 年 ${Number(m[2])} 月` : month;
}

/** 「11 月 3 日」 from a series `YYYY-MM-DD`. */
export function dayLabel(date: string): string {
  const [, mm, dd] = date.split('-');
  return mm && dd ? `${Number(mm)} 月 ${Number(dd)} 日` : date;
}
