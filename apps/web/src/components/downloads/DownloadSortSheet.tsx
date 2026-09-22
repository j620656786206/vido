// Design ref: ux-design.pen Screen D10-M-v2 (JxMWL)
/**
 * The phone sort sheet (dsr-4b-1). Only exists below 640px — the desktop keeps
 * its native select — so it is a `ui/Sheet` (Base UI), not a `ui/Dialog` with
 * the `MOBILE_SHEET_*` shell (see the `ui/mobileSheet.tsx` header).
 *
 * It lists exactly what the desktop select lists: `options` is the page's
 * SORT_OPTIONS, handed in, never re-declared here. A single choice that applies
 * at once reads as a radio group; picking closes the sheet, picking the row that
 * is already checked closes it too. Every row keeps a 20px icon slot, so the
 * label never jumps sideways when the check moves (the D10-M draft did, 32px).
 */
import { useRef, type ComponentProps, type KeyboardEvent } from 'react';
import { Check } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { SortField, SortOrder } from '../../services/downloadService';
import { Sheet } from '../ui/Sheet';

interface DownloadSortSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Where focus lands once the sheet has closed — the sort button that opened it. */
  finalFocus?: ComponentProps<typeof Sheet>['finalFocus'];
  options: ReadonlyArray<{ field: SortField; order: SortOrder; label: string }>;
  /** `field:order`, the same string the desktop select uses as its value. */
  value: string;
  onChange: (value: string) => void;
}

export function DownloadSortSheet({
  open,
  onOpenChange,
  finalFocus,
  options,
  value,
  onChange,
}: DownloadSortSheetProps) {
  const rowsRef = useRef<(HTMLButtonElement | null)[]>([]);

  const hasChecked = options.some((o) => `${o.field}:${o.order}` === value);

  const pick = (next: string) => {
    onChange(next);
    onOpenChange(false);
  };

  // Up/Down walk the rows and wrap at both ends; Enter/Space are the button's own click.
  const onKeyDown = (e: KeyboardEvent<HTMLButtonElement>, at: number) => {
    if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return;
    e.preventDefault();
    const n = options.length;
    rowsRef.current[(at + (e.key === 'ArrowDown' ? 1 : -1) + n) % n]?.focus();
  };

  return (
    <Sheet
      open={open}
      onOpenChange={onOpenChange}
      title="排序"
      testId="download-sort-sheet"
      titleClassName="mb-1 px-5"
      description="選擇下載清單的排序方式"
      descriptionClassName="px-5 pb-3"
      className="p-0 pt-2 pb-[max(1.25rem,env(safe-area-inset-bottom))]"
      finalFocus={finalFocus}
    >
      <div aria-hidden="true" className="h-px bg-[var(--border-subtle)]" />
      <div role="radiogroup" aria-label="排序方式" className="flex flex-col gap-0.5 px-2 py-1">
        {options.map((o, i) => {
          const optionValue = `${o.field}:${o.order}`;
          const checked = optionValue === value;
          // A value no row carries (a sort only the table headers can make) must not
          // leave the group with nothing to Tab into — the first row stands in.
          const inTabOrder = checked || (!hasChecked && i === 0);
          return (
            <button
              key={optionValue}
              ref={(el) => {
                rowsRef.current[i] = el;
              }}
              type="button"
              role="radio"
              aria-checked={checked}
              tabIndex={inTabOrder ? 0 : -1}
              onClick={() => pick(optionValue)}
              onKeyDown={(e) => onKeyDown(e, i)}
              className={cn(
                'flex min-h-[52px] items-center gap-3 rounded-[var(--radius-md)] px-3 text-left text-base transition-colors',
                checked
                  ? 'bg-[var(--accent-subtle)] font-semibold text-[var(--accent-text)]'
                  : 'font-medium text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]'
              )}
            >
              {checked ? (
                <Check className="size-5 shrink-0" aria-hidden="true" />
              ) : (
                <span className="size-5 shrink-0" aria-hidden="true" />
              )}
              <span>{o.label}</span>
            </button>
          );
        })}
      </div>
    </Sheet>
  );
}
