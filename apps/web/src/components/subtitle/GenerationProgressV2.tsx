// Implements: Component/GenerationProgress-v2 (XkGvG)
// Source: ux-design.pen (Pencil app)
/**
 * Route C generation stepper (ux3-subtitle-v2 AC 3, Component Library sJzat row
 * luza9). Renders the FROZEN stage list 提取音訊 → 轉錄中 → 翻譯中 → 簡轉繁 →
 * AI校正 → 完成 (fixture vocabulary — renaming breaks fixture↔baseline mapping)
 * plus the failed-at-stage panel.
 *
 * Wire-phase mapping (transcription_service.go): `extracting`→提取音訊,
 * `transcribing`→轉錄中, `translating` (+percentage 0–100)→翻譯中, `complete`→完成.
 * 簡轉繁 and AI校正 have NO dedicated wire phase today — they advance ATOMICALLY
 * when `transcription_complete` arrives (the backend runs OpenCC + AI correction
 * inside the pipeline between `translating` and `complete`).
 *
 * Rule 23: this component reads NO ambient clock (`Date.now()`/`new Date()`) —
 * every timing/ETA/progress text is the server-supplied SSE `message`/`percentage`.
 *
 * Cost/quota slot (9R-17 dormant): optional `costUsedText`/`costLimitText` props;
 * renders NOTHING when absent — no BE cost surface exists today, do not invent.
 *
 * 重試 is NOT here: F4-D-v2 puts it in the dialog footer next to 稍後再試
 * (story dsr-6b AC #6), so the paid button and its price belong to the dialog.
 */
import { Check, LoaderCircle, X, CircleAlert } from 'lucide-react';
import { cn } from '../../lib/utils';
import {
  GENERATION_FAILED_FALLBACK,
  type GenerationPhase,
} from '../../hooks/useGenerationProgress';

/** FROZEN stage names (design handoff + AC 3) — also the gallery fixture vocabulary. */
export const GENERATION_STAGES = [
  '提取音訊',
  '轉錄中',
  '翻譯中',
  '簡轉繁',
  'AI校正',
  '完成',
] as const;

type ActivePhase = 'extracting' | 'transcribing' | 'translating';

const PHASE_INDEX: Record<ActivePhase, number> = {
  extracting: 0,
  transcribing: 1,
  translating: 2,
};

export interface GenerationProgressV2Props {
  /** Current pipeline phase (from useGenerationProgress; 'idle' renders 提取音訊 as active-waiting). */
  phase: GenerationPhase;
  /** Stage that was live when the failure arrived — names the panel's {stage}失敗. */
  failedPhase?: ActivePhase | null;
  /** translation_progress percentage (0–100 float). Mono numerals. */
  percentage?: number | null;
  /** Server-supplied progress/error text (SSE payload — the ONLY timing source, Rule 23). */
  message?: string;
  /** Error detail from transcription_failed. */
  error?: string | null;
  /** Optional cost slot (9R-17 dormant): both must be present to render the line. */
  costUsedText?: string;
  costLimitText?: string;
  /**
   * Desktop horizontal alignment of the stage row. Default 'center' is the
   * standalone dialog (F3/F4) — unchanged. 'start' is the batch queue row
   * (F8-D-v2), where the stepper sits under the poster's left edge.
   */
  align?: 'center' | 'start';
}

/** F4-D-v2 `pjXCe`: the failure named with the stepper's own words. */
const FAILED_STAGE_TEXT: Record<ActivePhase, string> = {
  extracting: '提取音訊失敗',
  transcribing: '轉錄失敗',
  translating: '翻譯失敗',
};

/** A detail that STARTS in CJK is a sentence written for people (the D6 family's
 *  「字幕生成失敗：…」「已略過：…」). A Go error that merely carries a Chinese file
 *  path — `ffprobe timeout: /media/電影/…` — is still machine text. */
const STARTS_CJK_RE = /^[\u3000-\u30ff\u3400-\u4dbf\u4e00-\u9fff\uac00-\ud7af\uff00-\uffef]/;

type StepState = 'done' | 'active' | 'pending' | 'failed';

function stepStates(phase: GenerationPhase, failedPhase?: ActivePhase | null): StepState[] {
  if (phase === 'complete') {
    // 簡轉繁 / AI校正 / 完成 flip atomically on transcription_complete.
    return GENERATION_STAGES.map(() => 'done');
  }
  if (phase === 'failed') {
    const failedIdx = PHASE_INDEX[failedPhase ?? 'extracting'];
    return GENERATION_STAGES.map((_, i) =>
      i < failedIdx ? 'done' : i === failedIdx ? 'failed' : 'pending'
    );
  }
  const activeIdx = phase === 'idle' ? 0 : PHASE_INDEX[phase as ActivePhase];
  return GENERATION_STAGES.map((_, i) =>
    i < activeIdx ? 'done' : i === activeIdx ? 'active' : 'pending'
  );
}

function StepMark({ state }: { state: StepState }) {
  return (
    <span
      className={cn(
        'flex h-[22px] w-[22px] shrink-0 items-center justify-center rounded-full',
        state === 'done' && 'bg-[var(--success-tint)]',
        state === 'active' && 'bg-[var(--accent-tint)]',
        state === 'failed' && 'bg-[var(--error-tint)]',
        state === 'pending' && 'bg-[var(--bg-tertiary)]'
      )}
    >
      {state === 'done' && (
        <Check className="h-3.5 w-3.5 text-[var(--success-text)]" aria-hidden="true" />
      )}
      {state === 'active' && (
        <LoaderCircle
          className="h-3.5 w-3.5 animate-spin text-[var(--accent-text)] motion-reduce:animate-none"
          aria-hidden="true"
        />
      )}
      {state === 'failed' && (
        <X className="h-3.5 w-3.5 text-[var(--error-text)]" aria-hidden="true" />
      )}
      {state === 'pending' && (
        <span className="h-1.5 w-1.5 rounded-full bg-[var(--text-muted)]" aria-hidden="true" />
      )}
    </span>
  );
}

export function GenerationProgressV2({
  phase,
  failedPhase = null,
  percentage = null,
  message,
  error,
  costUsedText,
  costLimitText,
  align = 'center',
}: GenerationProgressV2Props) {
  const states = stepStates(phase, failedPhase);
  const failedText = FAILED_STAGE_TEXT[failedPhase ?? 'extracting'];
  const failedDetail = error && error !== GENERATION_FAILED_FALLBACK ? error : null;
  const pctText =
    percentage !== null && percentage !== undefined ? `${Math.round(percentage)}%` : null;

  return (
    <div data-testid="generation-progress-v2" className="flex flex-col gap-4">
      {/* Stepper — mobile (<sm) = vertical full-width rows per F3-M-v2 (k8sJl4 `fS5is`,
          Sally gate MUST-FIX 2026-07-05): [22px circle + Body 14 label + spacer + Mono pct].
          Desktop (sm+) keeps the original horizontal stepper — every sm: class computes
          IDENTICALLY to the pre-fix desktop DOM (zero darwin-baseline diff). */}
      <ol
        className={cn(
          'flex flex-col gap-1.5 sm:flex-row sm:items-start sm:gap-0',
          align === 'start' ? 'sm:justify-start' : 'sm:justify-center'
        )}
        aria-label="字幕生成進度"
      >
        {GENERATION_STAGES.map((stage, i) => {
          const state = states[i];
          return (
            <li key={stage} className="flex w-full items-start sm:w-auto">
              {i > 0 && (
                <span
                  aria-hidden="true"
                  className={cn(
                    'hidden sm:block',
                    // Connector is desktop-only (hidden below sm) — XkGvG `ITuZl` 26×2.
                    'mt-[10px] h-0.5 sm:w-[26px]',
                    states[i - 1] === 'done' ? 'bg-[var(--success)]' : 'bg-[var(--border-subtle)]'
                  )}
                />
              )}
              <span
                data-testid={`gen-stage-${stage}`}
                data-state={state}
                className="flex w-full flex-row items-center gap-2.5 max-sm:p-1 sm:w-[72px] sm:flex-col sm:gap-1.5"
              >
                <StepMark state={state} />
                <span
                  className={cn(
                    // Phone: Body 14 (F3-M-v2 IdGB2). Desktop: Label 12 / 1.5.
                    'text-sm sm:text-xs sm:leading-normal',
                    state === 'active' && 'font-semibold text-[var(--accent-text)]',
                    state === 'failed' && 'font-semibold text-[var(--error-text)]',
                    state === 'done' && 'text-[var(--text-secondary)]',
                    state === 'pending' && 'text-[var(--text-muted)]'
                  )}
                >
                  {stage}
                </span>
                {state === 'active' && pctText && (
                  <span className="ml-auto font-mono text-[11px] tabular-nums text-[var(--accent-text)] sm:ml-0">
                    {pctText}
                  </span>
                )}
              </span>
            </li>
          );
        })}
      </ol>

      {/* Stage detail — server-supplied text only (Rule 23: no local clock). */}
      {message && phase !== 'failed' && (
        <p
          data-testid="gen-stage-message"
          className="text-center text-sm text-[var(--text-secondary)]"
        >
          {message}
        </p>
      )}

      {/* Failed panel (F4-D-v2 vgChD): {stage}失敗 + the server's error on its own line.
          重試 lives in the dialog footer (dg5rH), not here. */}
      {phase === 'failed' && (
        <div
          data-testid="gen-failed-panel"
          className="flex items-center gap-2 rounded-[var(--radius-md)] bg-[var(--error-tint)] p-3"
        >
          <CircleAlert className="h-4 w-4 shrink-0 text-[var(--error-text)]" aria-hidden="true" />
          <div className="flex min-w-0 flex-1 flex-col gap-1">
            <p className="text-sm text-[var(--error-text)]">{failedText}</p>
            {failedDetail && (
              <span
                data-testid="gen-failed-detail"
                className={cn(
                  'text-xs text-[var(--error-text)]',
                  // A Go error string is machine text: verbatim, Mono, wraps anywhere.
                  !STARTS_CJK_RE.test(failedDetail) && 'break-all font-mono'
                )}
              >
                {failedDetail}
              </span>
            )}
          </div>
        </div>
      )}

      {/* Cost/quota slot — dormant until 9R-17; renders nothing when props absent. */}
      {costUsedText && costLimitText && (
        <p data-testid="gen-cost-line" className="text-center text-xs text-[var(--text-secondary)]">
          本次用量：
          <span className="font-mono font-semibold tabular-nums text-[var(--text-primary)]">
            {costUsedText}
          </span>
          <span className="font-mono text-[var(--text-muted)]"> / </span>
          上限 <span className="font-mono tabular-nums">{costLimitText}</span>
        </p>
      )}
    </div>
  );
}
