// Implements: <utility — no .pen counterpart>
/**
 * What the paid 生成字幕 / 重試 buttons show, and the one line that explains it
 * (story dsr-6a AC #5; J9-D six states + the SM supplement in J9-D §C).
 *
 * ONE pure function so the main button and both retry buttons can never
 * disagree about whether a click costs money, how much, or why it is blocked.
 *
 * The invariant (DESIGN.md「沒有金額，就沒有可按的按鈕」): the only clickable
 * state is `ready`, and `ready` always carries an amount. Every other state is
 * `loading` (not clickable yet) or `unavailable` (disabled, with a reason).
 */
import type { ButtonCostState } from '../ui/ButtonCost';
import type { TranscriptionEstimate } from '../../services/transcriptionService';

export type HelperTone = 'muted' | 'secondary';

export interface GenerateHelper {
  text: string;
  /** muted while the button can be pressed; secondary when it is blocked
   *  (DESIGN.md: the CONTROL is disabled, the explanation is not). */
  tone: HelperTone;
  /** Render the 前往設定 link to /settings/keys after the text. */
  settingsLink: boolean;
}

export interface RetryNote {
  text: string;
  settingsLink: boolean;
}

export interface GenerateCostView {
  cost: ButtonCostState;
  helper: GenerateHelper;
  /** What the retry surfaces say beside 重試 (the start-error panel, and the
   *  failed-run footer hint since dsr-6b) — the block reason, the ≈ explanation or
   *  the English-only line. null = nothing. */
  retryNote: RetryNote | null;
}

/** The subset of a TanStack query result this function reads. */
export interface EstimateQueryState {
  data: TranscriptionEstimate | undefined;
  isError: boolean;
  error: unknown;
}

export const SERIES_LINE = '請於下方分集清單逐集生成';
export const DEFAULT_LINE = '語音辨識＋AI 翻譯，約需數分鐘';
export const TRANSLATE_ONLY_LINE = '僅需翻譯，不再重跑語音辨識——這次很快也很便宜';
export const ENGLISH_ONLY_LINE = '僅能產生英文字幕——尚未設定翻譯金鑰';
export const SELF_HOSTED_PREFIX = '語音辨識：自架（不另計費）。';
export const FALLBACK_RUNTIME_LINE = '片長未知（估 45 分）——實際費用依內容長度而定';
export const ESTIMATE_FAILED_LINE = '暫時算不出費用，因此先不開放。重新整理或稍後再試。';
export const FILE_UNREADABLE_LINE =
  '讀不到影片檔案，因此先不開放。請確認檔案還在，或重新掃描媒體庫。';
export const ASR_NOT_CONFIGURED_LINE =
  '生成字幕需要雲端語音辨識（ASR）金鑰。請至金鑰設定儲存後即可使用。';
export const TRANSLATION_KEY_MISSING_LINE = '尚未設定翻譯金鑰';

function blocked(text: string, settingsLink: boolean): GenerateCostView {
  return {
    cost: { status: 'unavailable' },
    helper: { text, tone: 'secondary', settingsLink },
    retryNote: { text, settingsLink },
  };
}

export function deriveGenerateCostView(input: {
  mediaType: 'movie' | 'series' | 'episode';
  subtitleStatus?: string;
  estimate: EstimateQueryState;
}): GenerateCostView {
  const { mediaType, subtitleStatus, estimate } = input;

  if (mediaType === 'series') {
    // J3-D: a series is a container — generation happens per episode.
    return {
      cost: { status: 'unavailable' },
      helper: { text: SERIES_LINE, tone: 'secondary', settingsLink: false },
      retryNote: null,
    };
  }

  const data = estimate.data;
  const errorCode = estimate.isError
    ? (estimate.error as { code?: unknown } | null)?.code
    : undefined;

  // The file being gone is a definitive answer, not a transient failure: it wins
  // even over a price already on screen — that click could only fail again.
  if (errorCode === 'VALIDATION_REQUIRED_FIELD') {
    return blocked(FILE_UNREADABLE_LINE, false);
  }

  if (!data) {
    if (estimate.isError) {
      return blocked(ESTIMATE_FAILED_LINE, false);
    }
    // J9-D ④: the line does not change while the price is on its way — it is
    // read from what the page already knows.
    return {
      cost: { status: 'loading' },
      helper: {
        text: subtitleStatus === 'untranslated' ? TRANSLATE_ONLY_LINE : DEFAULT_LINE,
        tone: 'muted',
        settingsLink: false,
      },
      retryNote: null,
    };
  }

  const translateOnly = data.plan === 'translate_only';

  if (!translateOnly && !data.asrAvailable) {
    return blocked(ASR_NOT_CONFIGURED_LINE, true); // J9-D ⑥
  }
  if (translateOnly && !data.translationConfigured) {
    // The run would resume, skip the translate leg and produce nothing — a
    // clickable $0.00 here would be a button that lies (SM supplement ②).
    return blocked(TRANSLATION_KEY_MISSING_LINE, true);
  }

  const approximate = data.runtimeSource === 'fallback';
  const cost: ButtonCostState = { status: 'ready', usd: data.estimatedUsd, approximate };
  const retryNote: RetryNote | null = approximate
    ? { text: FALLBACK_RUNTIME_LINE, settingsLink: false }
    : null;

  if (translateOnly) {
    return {
      cost,
      retryNote,
      helper: { text: TRANSLATE_ONLY_LINE, tone: 'muted', settingsLink: false },
    };
  }
  if (!data.translationConfigured) {
    // Degraded ≠ blocked (sub-2-2d): an English subtitle beats none. With
    // self-hosted ASR the amount is $0.00, and a zero must say why (SM supplement ①)
    // — on the retry surfaces too (start-error panel, failed-run footer hint).
    const text = data.selfHostedAsr
      ? `${SELF_HOSTED_PREFIX}${ENGLISH_ONLY_LINE}`
      : ENGLISH_ONLY_LINE;
    return {
      cost,
      retryNote: { text, settingsLink: true },
      helper: { text, tone: 'muted', settingsLink: true },
    };
  }
  if (approximate) {
    return {
      cost,
      retryNote,
      helper: { text: FALLBACK_RUNTIME_LINE, tone: 'muted', settingsLink: false },
    };
  }
  return { cost, retryNote, helper: { text: DEFAULT_LINE, tone: 'muted', settingsLink: false } };
}
