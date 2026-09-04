// Design ref: ux-design.pen Screen F16-D-v2 (gmOt6) · F19-D-v2 (KThbY)
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
import { usd } from '../../../lib/currency';
import { cn } from '../../../lib/utils';
import type { ModelChoice } from './consentSelection';

export interface ModelPickerProps {
  choices: ModelChoice[];
  selectedModelId: string;
  onSelect: (modelId: string) => void;
  /** Locked while a paid start is in flight — the quote must not move under it. */
  disabled?: boolean;
}

/** Grade badge tint: only a MEASURED grade earns a colour. */
function gradeTint(grade?: string): string {
  if (!grade) return 'bg-[var(--bg-tertiary)] text-[var(--text-muted)]';
  if (grade === 'A') return 'bg-[var(--success-tint)] text-[var(--success-text)]';
  return 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)]';
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

export function ModelPicker({ choices, selectedModelId, onSelect, disabled }: ModelPickerProps) {
  if (choices.length === 0) return null;

  const defaultName = choices.find((c) => c.isDefault)?.displayName;

  return (
    <div className="flex flex-col gap-2" data-testid="consent-model-picker">
      <p className="text-[13px] font-medium text-[var(--text-primary)]">選擇翻譯模型</p>
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
                <span className="min-w-0 flex-1 truncate text-[13px] text-[var(--text-primary)]">
                  {choice.displayName}
                  {choice.isDefault && (
                    <span className="ml-1.5 text-[11px] text-[var(--text-muted)]">（預設）</span>
                  )}
                </span>
                <span
                  data-testid={`consent-model-usd-${choice.id}`}
                  className="shrink-0 font-mono text-[13px] font-semibold tabular-nums text-[var(--text-primary)]"
                >
                  這批約 {usd(choice.totalUsd)}
                </span>
              </span>

              <span className="flex items-center gap-2 pl-[26px] text-[11px] text-[var(--text-secondary)]">
                <span
                  data-testid={`consent-model-grade-${choice.id}`}
                  title={choice.qualityNote}
                  className={cn(
                    'rounded-[var(--radius-sm)] px-1.5 py-0.5',
                    gradeTint(choice.qualityGrade)
                  )}
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
                {/* P1-8 (sub-7-8) will make this actionable; today it is the
                    honest answer to「為什麼這格是空的」. */}
                {!choice.qualityGrade && <span>· 可花約 $0.01 試跑 20 句</span>}
              </span>

              {note !== '' && (
                <span
                  data-testid={`consent-model-note-${choice.id}`}
                  className="pl-[26px] text-[11px] text-[var(--text-muted)]"
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
