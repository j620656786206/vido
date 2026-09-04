// Design ref: ux-design.pen Screen F16-D-v2 (gmOt6) · F16-M-v2 (x45wBO) · F19-D-v2 (KThbY) · F19-M-v2 (IMQO6)
/**
 * F16 金額確認 / F19 超出上限確認 (sub-4-3 AC #4) — the LAST gate before money
 * is spent. Breakdown figures come from the SAME ConsentTotals the list panel
 * renders (三處金額同源). Over-budget flips the tint to warning (one shade
 * deeper than F16's neutral — the third design round widened the contrast),
 * the total to warning orange and the primary button to 仍要開始.
 *
 * Soft-ceiling honesty: the copy says 自動暫停/預計/約 — never "絕不超過".
 *
 * sub-6-8b adds the 「選擇翻譯模型」 block above the breakdown, and with it the
 * tallest this dialog ever gets: three model rows plus the breakdown can
 * exceed a phone viewport, and a confirm button pushed off-screen is a dead
 * end. So the BODY scrolls (max-h-[85vh] + overflow-y-auto) while the title
 * bar and the action row stay put. Switching model here re-prices the F15
 * summary bar and footer behind the dialog too — one selector feeds all of
 * them, so they can never disagree about what this batch costs.
 *
 * MOBILE (sub-6-8b): a bottom sheet, not a centred dialog. The shared
 * DialogContent base is `w-full max-w-lg`, and `w-full` on a fixed element is
 * 100% of the VIEWPORT — so below 448px the "centred" dialog was already
 * full-bleed, its rounded corners flush against both screen edges. Rather than
 * paper over that with a side margin, this follows the sheet vocabulary F15-M
 * (fdu4y) established for the very screen this dialog opens from: handle,
 * top-rounded, pinned to the bottom, action row within thumb reach. Desktop is
 * untouched — every change is behind `sm:`.
 */
import { Loader2 } from 'lucide-react';
import { Dialog, DialogContent, DialogTitle } from '../../ui/Dialog';
import { cn } from '../../../lib/utils';
import { usd } from '../../../lib/currency';
import { ModelPicker } from './ModelPicker';
import type { ConsentTotals, ModelChoice } from './consentSelection';

export interface ConfirmGenerationDialogProps {
  open: boolean;
  totals: ConsentTotals;
  budgetUsd: number;
  confirming?: boolean;
  /**
   * sub-6-8b: the models this deployment can run, priced for THIS batch.
   * Empty (or absent) renders no picker — a deployment with one reachable
   * model has no question to ask, and a failed catalog fetch must not invent
   * one.
   */
  modelChoices?: ModelChoice[];
  selectedModelId?: string;
  onModelChange?: (modelId: string) => void;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmGenerationDialog({
  open,
  totals,
  budgetUsd,
  confirming = false,
  modelChoices = [],
  selectedModelId = '',
  onModelChange,
  onConfirm,
  onCancel,
}: ConfirmGenerationDialogProps) {
  const overBudget = totals.overBudget;
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onCancel();
      }}
    >
      <DialogContent
        data-testid="consent-confirm-dialog"
        aria-describedby={undefined}
        className={cn(
          'flex max-h-[85vh] flex-col gap-0 overflow-hidden p-0',
          // Mobile: bottom sheet (F16-M-v2 / F19-M-v2) — same geometry the
          // consent list uses one step earlier in the same flow.
          'bottom-0 left-0 right-0 top-auto w-full max-w-none translate-x-0 translate-y-0 rounded-b-none rounded-t-[var(--radius-xl)]',
          'sm:bottom-auto sm:left-1/2 sm:right-auto sm:top-1/2 sm:w-full sm:max-w-md sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-[var(--radius-xl)]'
        )}
      >
        {/* Mobile bottom-sheet drag handle (F15-M precedent, sm:hidden). */}
        <div className="flex shrink-0 justify-center pb-1 pt-2 sm:hidden">
          <span aria-hidden="true" className="h-1 w-9 rounded-full bg-[var(--bg-tertiary)]" />
        </div>

        <div className="flex h-14 shrink-0 items-center border-b border-[var(--border-subtle)] pl-4 pr-12 sm:pl-6">
          <DialogTitle className="text-base font-semibold">確認產生字幕</DialogTitle>
        </div>

        <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-4 sm:p-6">
          <p className="flex items-center gap-[3px] text-sm text-[var(--text-primary)]">
            即將為 <span className="font-mono tabular-nums">{totals.selectedCount}</span>
            部影片產生字幕
          </p>

          {modelChoices.length > 0 && onModelChange && (
            <ModelPicker
              choices={modelChoices}
              selectedModelId={selectedModelId}
              onSelect={onModelChange}
              disabled={confirming}
            />
          )}

          <div className="flex flex-col gap-2 text-[13px] text-[var(--text-secondary)]">
            <p className="flex items-center justify-between">
              <span className="flex items-center gap-[3px]">
                語音辨識 <span className="font-mono tabular-nums">{totals.selectedAsrCount}</span>{' '}
                部
              </span>
              <span data-testid="consent-confirm-asr-usd" className="font-mono tabular-nums">
                預估 {usd(totals.selectedAsrUsd)}
              </span>
            </p>
            <p className="flex items-center justify-between">
              <span className="flex items-center gap-[3px]">
                抽取 + 翻譯
                <span className="font-mono tabular-nums">{totals.selectedExtractCount}</span> 部
              </span>
              <span data-testid="consent-confirm-extract-usd" className="font-mono tabular-nums">
                預估 {usd(totals.selectedExtractUsd)}
              </span>
            </p>
            <p className="flex items-center justify-between border-t border-[var(--border-subtle)] pt-2 text-[var(--text-primary)]">
              <span>合計預估</span>
              <span
                data-testid="consent-confirm-total-usd"
                className={cn(
                  'font-mono font-semibold tabular-nums',
                  overBudget ? 'text-[var(--warning-text)]' : 'text-[var(--text-primary)]'
                )}
              >
                {usd(totals.selectedTotalUsd)}
              </span>
            </p>
          </div>

          <div
            data-testid="consent-confirm-hint"
            className={cn(
              'rounded-[var(--radius-md)] p-3 text-[13px]',
              overBudget
                ? 'bg-[var(--warning-tint)] text-[var(--text-primary)]'
                : 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)]'
            )}
          >
            {overBudget ? (
              <>
                預估金額已超過上限
                <span className="font-mono tabular-nums"> {usd(budgetUsd)}</span>
                。系統會依序處理，累計達到上限時自動暫停，預計可完成約
                <span className="font-mono tabular-nums"> {totals.feasibleCount} </span>
                部；未完成的項目會保留，可提高上限後續跑。
              </>
            ) : (
              <>
                預估金額僅供參考，實際費用依內容長度而定。累計達到上限
                <span className="font-mono tabular-nums"> {usd(budgetUsd)} </span>
                時會自動暫停，已完成的部分會保留。
              </>
            )}
          </div>
        </div>

        <div className="flex shrink-0 items-center justify-end gap-3 border-t border-[var(--border-subtle)] px-4 py-3.5 pb-[max(0.875rem,env(safe-area-inset-bottom))] sm:px-6 sm:pb-3.5">
          <button
            type="button"
            onClick={onCancel}
            data-testid="consent-confirm-cancel"
            className="flex min-h-[44px] items-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)]"
          >
            取消
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={confirming}
            data-testid="consent-confirm-start"
            className="flex min-h-[44px] items-center gap-2 rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-6 text-sm font-medium text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-pressed)] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {confirming && (
              <Loader2
                className="h-4 w-4 animate-spin motion-reduce:animate-none"
                aria-hidden="true"
              />
            )}
            {overBudget ? '仍要開始' : '確認並開始'}
          </button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
