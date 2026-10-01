// Design ref: ux-design.pen Screen F16-D-v2 (gmOt6) · F16-M-v2 (x45wBO) · F19-D-v2 (KThbY) · J10 (ctRsy)
/**
 * 「選擇翻譯模型」 radio list inside the F16/F19 confirm dialog (sub-6-8b
 * AC #1/#4/#5).
 *
 * The whole point of this block is that the 2.7× price gap between Sonnet and
 * Haiku is a choice the user SEES and makes, so every row states all three
 * things that differ — this batch's price, the MEASURED quality grade, and the
 * rough wall-clock time — and the cheaper option is never hidden behind a
 * default. Equally, an unevaluated model shows 「尚未評測」 rather than a blank
 * where a grade would go: an absent grade is a fact about our testing, not a
 * claim of parity.
 *
 * Figures come from `modelChoices()` — the ONE selector that owns the money
 * math for these screens. Nothing is computed here.
 */
import type * as React from 'react';
import { usd } from '../../../lib/currency';
import { cn } from '../../../lib/utils';
import { ButtonCost } from '../../ui/ButtonCost';
import type { ModelPreviewRowState } from '../../../hooks/useModelPreview';
import type { ModelLocalGrade } from '../../../services/subtitleService';
import type { ModelChoice } from './consentSelection';

export interface ModelPickerProps {
  choices: ModelChoice[];
  selectedModelId: string;
  onSelect: (modelId: string) => void;
  /** Locked while a paid start is in flight — the quote must not move under it. */
  disabled?: boolean;
  /**
   * sub-7-8c: 「試跑 20 句」 per ungraded row (J10). Absent → the rows just
   * say 尚未評測 and the button is not rendered at all.
   */
  previewStates?: Record<string, ModelPreviewRowState>;
  onPreview?: (modelId: string) => void;
}

/** 「你的實測：0 分 5%・2 分 70%・花 $0.05」 (J10 ③), with the incomplete variant. */
export function localGradeLine(g: ModelLocalGrade): string {
  const pct = (v: number) => `${Math.round(v * 100)}%`;
  const body = `0 分 ${pct(g.zeroRate)}・2 分 ${pct(g.naturalRate)}・花 ${usd(g.costUsd)}`;
  return g.incomplete ? `到預算上限才停：${body}` : `你的實測：${body}`;
}

/**
 * Grade badge tone — NEUTRAL for every grade (dsr-6e-2, Alexyu 2026-09-20).
 *
 * 「品質 A」 used to wear jade. But a grade is a property of the model (with its
 * provenance in the accessible name), not something that happened — the same
 * line that made TechBadge neutral. The letter and the note carry the
 * difference; an unmeasured model is only a step quieter, never blank.
 */
function gradeTint(grade?: string): string {
  return grade
    ? 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)]'
    : 'bg-[var(--bg-tertiary)] text-[var(--text-muted)]';
}

/**
 * The one-line justification under the SELECTED row (AC #4). Both directions
 * are stated: a dearer model is a real choice too, and hiding its premium
 * would be the same omission this screen exists to prevent. Empty string =
 * nothing worth saying (an identically-priced alternative).
 */
function selectionNote(choice: ModelChoice, defaultName?: string): string {
  if (choice.deltaUsd !== undefined && choice.deltaUsd !== 0 && defaultName) {
    const verb = choice.deltaUsd > 0 ? '省' : '多';
    const pct = choice.deltaPercent !== undefined ? `（${choice.deltaPercent}%）` : '';
    return `比 ${defaultName} ${verb} ${usd(Math.abs(choice.deltaUsd))}${pct}`;
  }
  // Only the row that actually holds the top MEASURED grade may claim it — a
  // CLAUDE_MODEL override can make a B-grade model the default, and this copy
  // must not follow it into a false claim.
  if (choice.isDefault && choice.isBestGrade) return 'eval-1 實測品質最穩';
  return '';
}

export function ModelPicker({
  choices,
  selectedModelId,
  onSelect,
  disabled,
  previewStates,
  onPreview,
}: ModelPickerProps) {
  if (choices.length === 0) return null;

  const defaultName = choices.find((c) => c.isDefault)?.displayName;

  return (
    <div className="flex flex-col gap-2" data-testid="consent-model-picker">
      <p className="text-sm font-semibold text-[var(--text-primary)]">選擇翻譯模型</p>
      <div
        role="radiogroup"
        aria-label="翻譯模型"
        className="flex flex-col gap-1.5 rounded-[var(--radius-md)] border border-[var(--border-subtle)] p-1.5"
      >
        {choices.map((choice) => {
          const checked = choice.id === selectedModelId;
          const note = checked ? selectionNote(choice, defaultName) : '';
          return (
            <label
              key={choice.id}
              data-testid={`consent-model-option-${choice.id}`}
              data-selected={checked ? 'true' : 'false'}
              className={cn(
                'flex min-h-[44px] cursor-pointer flex-col gap-1 rounded-[var(--radius-sm)] px-2.5 py-2 transition-colors',
                checked ? 'bg-[var(--bg-tertiary)]' : 'hover:bg-[var(--bg-tertiary)]',
                disabled && 'cursor-not-allowed opacity-60'
              )}
            >
              <span className="flex items-center gap-2.5">
                <input
                  type="radio"
                  name="consent-translation-model"
                  value={choice.id}
                  checked={checked}
                  disabled={disabled}
                  onChange={() => onSelect(choice.id)}
                  className="h-4 w-4 shrink-0 accent-[var(--accent-primary)] disabled:cursor-not-allowed"
                />
                <span className="min-w-0 flex-1 truncate text-sm text-[var(--text-primary)]">
                  {choice.displayName}
                  {choice.isDefault && (
                    <span className="ml-1.5 text-xs text-[var(--text-muted)]">（預設）</span>
                  )}
                </span>
                <span
                  data-testid={`consent-model-usd-${choice.id}`}
                  className="shrink-0 font-mono text-sm font-bold tabular-nums text-[var(--text-primary)]"
                >
                  約 {usd(choice.totalUsd)}
                </span>
              </span>

              <span className="flex flex-wrap items-center gap-x-2 gap-y-1.5 pl-[26px] text-xs text-[var(--text-secondary)]">
                <span
                  data-testid={`consent-model-grade-${choice.id}`}
                  title={choice.qualityNote}
                  // CR L8: `title` alone is a hover tooltip — unreachable by
                  // keyboard, unreliable for screen readers, and meaningless on
                  // the mobile sheet. The provenance is the whole reason the
                  // grade is trustworthy, so it also goes in the accessible name.
                  aria-label={
                    choice.qualityGrade
                      ? `品質 ${choice.qualityGrade}${choice.qualityNote ? `（${choice.qualityNote}）` : ''}`
                      : '尚未評測'
                  }
                  className={cn('rounded-full px-1.5 py-0.5', gradeTint(choice.qualityGrade))}
                >
                  {choice.qualityGrade ? `品質 ${choice.qualityGrade}` : '尚未評測'}
                </span>
                {choice.minutes !== undefined && (
                  <span
                    data-testid={`consent-model-minutes-${choice.id}`}
                    className="font-mono tabular-nums"
                  >
                    約 {choice.minutes} 分鐘
                  </span>
                )}
                {!choice.qualityGrade && onPreview && (
                  <PreviewControl
                    choice={choice}
                    state={previewStates?.[choice.id] ?? { status: 'idle' }}
                    disabled={!!disabled}
                    onPreview={onPreview}
                  />
                )}
              </span>

              {note !== '' && (
                <span
                  data-testid={`consent-model-note-${choice.id}`}
                  className="pl-[26px] text-xs text-[var(--text-muted)]"
                >
                  {note}
                </span>
              )}
            </label>
          );
        })}
      </div>
    </div>
  );
}

/**
 * The 「試跑 20 句」 control on an ungraded row — J10's four states:
 * ① idle: a small ButtonCost carrying the estimate (J9: the $ is the mark);
 * ② running: the same button busy with a skeleton amount, plus 「約 1 分鐘」;
 * ③ done: one 「你的實測」 line (no letter, no badge colour) and 「再試一次」;
 * ④ failed: back to ① with the reason beside it.
 *
 * `preventDefault` on the button matters: it sits inside the row's <label>,
 * and a button click would otherwise activate the label and tick the radio.
 */
function PreviewControl({
  choice,
  state,
  disabled,
  onPreview,
}: {
  choice: ModelChoice;
  state: ModelPreviewRowState;
  disabled: boolean;
  onPreview: (modelId: string) => void;
}) {
  const estimate = choice.previewEstimateUsd ?? 0;
  // A result stored on the server (catalog refetch) and one just returned
  // this session are the same fact; the session one is fresher.
  const result = state.status === 'done' ? state.result : choice.localGrade;
  const running = state.status === 'running';

  const tryOut = (event: React.MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (!disabled && !running) onPreview(choice.id);
  };

  if (result && !running) {
    return (
      <span className="flex items-center gap-2 max-sm:basis-full">
        <span
          data-testid={`consent-model-local-grade-${choice.id}`}
          className="text-[var(--text-secondary)]"
        >
          {localGradeLine(result)}
        </span>
        <button
          type="button"
          data-testid={`consent-model-preview-again-${choice.id}`}
          disabled={disabled || estimate <= 0}
          onClick={tryOut}
          className="text-[var(--text-muted)] underline-offset-2 hover:underline disabled:cursor-not-allowed disabled:no-underline"
        >
          再試一次
        </button>
        {state.status === 'failed' && (
          <span
            data-testid={`consent-model-preview-error-${choice.id}`}
            className="text-[var(--danger-text)]"
          >
            {state.message}
          </span>
        )}
      </span>
    );
  }

  return (
    <span className="flex items-center gap-2 max-sm:basis-full">
      <ButtonCost
        data-testid={`consent-model-preview-${choice.id}`}
        label="試跑 20 句"
        cost={
          running
            ? { status: 'loading' }
            : estimate > 0
              ? { status: 'ready', usd: estimate, approximate: false }
              : { status: 'unavailable' }
        }
        busy={running}
        aria-disabled={disabled ? true : undefined}
        onClick={tryOut}
        // Row-sized variant of the J9 button: 24px tall, 12px type, same tints.
        className="min-h-0 h-6 gap-1.5 rounded-[var(--radius-sm)] px-2 text-xs font-semibold max-sm:h-7"
      />
      {running && (
        <span data-testid={`consent-model-preview-running-${choice.id}`}>
          約 1 分鐘，請勿關閉視窗
        </span>
      )}
      {state.status === 'failed' && (
        <span
          data-testid={`consent-model-preview-error-${choice.id}`}
          className="text-[var(--danger-text)]"
        >
          {state.message}
        </span>
      )}
    </span>
  );
}
