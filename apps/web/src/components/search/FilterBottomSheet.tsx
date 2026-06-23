// Design ref: ux-design.pen Screen AS-4 Filter Bottom Sheet Mobile (oypj1)
// Source: ux-design.pen (Pencil app)
import { useEffect, useRef, useState } from 'react';
import { cn } from '../../lib/utils';
import { FilterPanel } from './FilterPanel';
import { type DiscoverFilters, type DiscoverMediaType } from '../../lib/discoverFilters';
import { useDiscoverResults } from '../../hooks/useDiscoverResults';

interface FilterBottomSheetProps {
  isOpen: boolean;
  onClose: () => void;
  /** Currently committed filters (the sheet edits a local draft of these). */
  filters: DiscoverFilters;
  /** Commit the drafted filters (AC #6 — chips/results update after 套用篩選). */
  onApply: (next: DiscoverFilters) => void;
  /** Media type in scope — drives the live draft result count. */
  mediaType: DiscoverMediaType;
  /**
   * ux3-3-2 AC #9: `v2` restyles the sheet to the v2 design (radius-xl corner +
   * overlay-scrim backdrop) and debounces the draft year inputs, while KEEPING the
   * batch apply. `default` preserves the legacy styling byte-unchanged.
   */
  variant?: 'default' | 'v2';
}

/**
 * Mobile filter UI rendered as a slide-up bottom sheet (AC #6, Task 4.1). Edits a
 * local draft so the grid doesn't thrash on every tap; "套用篩選" commits, "清除全部"
 * resets the draft.
 */
export function FilterBottomSheet({
  isOpen,
  onClose,
  filters,
  onApply,
  mediaType,
  variant = 'default',
}: FilterBottomSheetProps) {
  const isV2 = variant === 'v2';
  const [draft, setDraft] = useState<DiscoverFilters>(filters);
  const sheetRef = useRef<HTMLDivElement>(null);

  // Live result count for the DRAFT (not the committed filters) — re-queries as
  // the user edits, gated to only run while the sheet is open (AC #6).
  const { totalResults, isLoading: isCounting } = useDiscoverResults(draft, mediaType, 1, isOpen);

  // Reset the draft to the committed filters each time the sheet opens.
  useEffect(() => {
    if (isOpen) setDraft(filters);
  }, [isOpen, filters]);

  // Lock body scroll while the sheet is open.
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [isOpen]);

  // Keyboard a11y: Escape closes the dialog; move focus into the sheet on open.
  useEffect(() => {
    if (!isOpen) return;
    sheetRef.current?.focus();
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const handleApply = () => {
    onApply(draft);
    onClose();
  };

  const handleClear = () => {
    setDraft({ genre: [], platform: [], sortBy: draft.sortBy });
  };

  return (
    <div className="fixed inset-0 z-[60] lg:hidden" role="dialog" aria-modal="true">
      {/* Backdrop */}
      <button
        type="button"
        aria-label="關閉篩選"
        onClick={onClose}
        className={cn(
          'absolute inset-0 backdrop-blur-sm',
          isV2 ? 'bg-[var(--overlay-scrim)]' : 'bg-black/50'
        )}
        data-testid="filter-sheet-backdrop"
      />

      {/* Sheet */}
      <div
        ref={sheetRef}
        tabIndex={-1}
        className={cn(
          'absolute inset-x-0 bottom-0 max-h-[85vh] overflow-y-auto',
          isV2 ? 'rounded-t-[var(--radius-xl)]' : 'rounded-t-2xl',
          'bg-[var(--bg-primary)] shadow-2xl outline-none'
        )}
        data-testid="filter-bottom-sheet"
      >
        {/* Drag handle */}
        <div className="flex justify-center py-2">
          <span className="h-1 w-10 rounded-full bg-[var(--border-subtle)]" aria-hidden="true" />
        </div>

        {/* Header */}
        <div className="flex items-center justify-between px-4 pb-3">
          <h2 className="text-lg font-semibold text-white">篩選條件</h2>
          <button
            onClick={handleClear}
            data-testid="filter-sheet-clear"
            className="text-sm text-[var(--accent-primary)] hover:underline"
          >
            清除全部
          </button>
        </div>

        {/* Controls */}
        <div className="px-4 pb-4">
          <FilterPanel filters={draft} onChange={setDraft} debounceMs={isV2 ? 350 : 0} />
        </div>

        {/* Apply */}
        <div className="sticky bottom-0 border-t border-[var(--border-subtle)] bg-[var(--bg-primary)] p-4">
          <button
            onClick={handleApply}
            data-testid="filter-sheet-apply"
            className="w-full rounded-lg bg-[var(--accent-primary)] py-3 text-sm font-semibold text-white transition-opacity hover:opacity-90"
          >
            套用篩選{isCounting ? '' : `（${totalResults} 部結果）`}
          </button>
        </div>
      </div>
    </div>
  );
}
