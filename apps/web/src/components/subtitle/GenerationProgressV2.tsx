// Implements: Component/GenerationProgress-v2 (XkGvG)
// Source: ux-design.pen (Pencil app)
/**
 * Route C generation stepper (ux3-subtitle-v2 AC 3, Component Library sJzat row
 * luza9). Renders the FROZEN stage list 提取音訊 → 轉錄中 → 翻譯中 → 簡轉繁 →
 * AI校正 → 完成 (fixture vocabulary — renaming breaks fixture↔baseline mapping)
 * plus the failed-at-stage state with 重試.
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
 */
import { Check, LoaderCircle, X, CircleAlert, RotateCcw } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { GenerationPhase } from '../../hooks/useGenerationProgress';

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
  /** Stage that was live when the failure arrived — labels 失敗於{stage}. */
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
  /** Renders the 重試 action in the failed state. */
  onRetry?: () => void;
}

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
        <Check className="h-3.5 w-3.5 text-[var(--success)]" aria-hidden="true" />
      )}
      {state === 'active' && (
        <LoaderCircle
          className="h-3.5 w-3.5 animate-spin text-[var(--accent-text)] motion-reduce:animate-none"
          aria-hidden="true"
        />
      )}
      {state === 'failed' && <X className="h-3.5 w-3.5 text-[var(--error)]" aria-hidden="true" />}
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
  onRetry,
}: GenerationProgressV2Props) {
  const states = stepStates(phase, failedPhase);
  const failedStageName =
    phase === 'failed' ? GENERATION_STAGES[PHASE_INDEX[failedPhase ?? 'extracting']] : null;
  const pctText =
    percentage !== null && percentage !== undefined ? `${Math.round(percentage)}%` : null;

  return (
    <div data-testid="generation-progress-v2" className="flex flex-col gap-4">
      {/* Stepper — mobile (<sm) = vertical full-width rows per F3-M-v2 (k8sJl4 `fS5is`,
          Sally gate MUST-FIX 2026-07-05): [22px circle + 13px label + spacer + Mono pct].
          Desktop (sm+) keeps the original horizontal stepper — every sm: class computes
          IDENTICALLY to the pre-fix desktop DOM (zero darwin-baseline diff). */}
      <ol
        className="flex flex-col gap-1.5 sm:flex-row sm:items-start sm:justify-center sm:gap-0"
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
                    'mt-[10px] h-0.5 w-5 sm:w-7',
                    states[i - 1] === 'done' ? 'bg-[var(--success)]' : 'bg-[var(--border-subtle)]'
                  )}
                />
              )}
              <span
                data-testid={`gen-stage-${stage}`}
                data-state={state}
                className="flex w-full flex-row items-center gap-2.5 sm:w-[72px] sm:flex-col sm:gap-1"
              >
                <StepMark state={state} />
                <span
                  className={cn(
                    'text-[13px] sm:text-xs',
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
          className="text-center text-[13px] text-[var(--text-secondary)]"
        >
          {message}
        </p>
      )}

      {/* Failed panel: 失敗於{stage} + server error + 重試 (F4-D-v2 U8rRtv). */}
      {phase === 'failed' && (
        <div
          data-testid="gen-failed-panel"
          className="flex items-center gap-2 rounded-[var(--radius-md)] bg-[var(--error-tint)] p-3"
        >
          <CircleAlert className="h-4 w-4 shrink-0 text-[var(--error)]" aria-hidden="true" />
          <p className="flex-1 text-[13px] text-[var(--error-text)]">
            失敗於{failedStageName}
            {error ? `：${error}` : ''}
          </p>
          {onRetry && (
            <button
              type="button"
              onClick={onRetry}
              data-testid="gen-retry"
              className="flex min-h-[44px] shrink-0 items-center gap-1.5 rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-4 text-sm font-medium text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-pressed)]"
            >
              <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" />
              重試
            </button>
          )}
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
