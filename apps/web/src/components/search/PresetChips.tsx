// Design ref: ux-design.pen Screen AS-1 - Advanced Search Filter Desktop (NWxok)
// Source: ux-design.pen (Pencil app) — presetBar (dPbq2)
import { useRef, useState } from 'react';
import {
  parseFiltersFromSearch,
  type DiscoverFilters,
  type DiscoverSearch,
} from '../../lib/discoverFilters';
import { useDeleteFilterPreset, useFilterPresets } from '../../hooks/useFilterPresets';
import type { FilterPreset } from '../../services/filterPresetService';

const LONG_PRESS_MS = 500;

interface PresetChipsProps {
  /**
   * Applies a saved preset's filters to the discover URL state. Wired to
   * `useFilterState().setFilters` from Story 11-2 in the discover route (AC #3).
   */
  onApplyPreset: (filters: DiscoverFilters) => void;
  className?: string;
}

/** Parse a stored preset's JSON-string filters back into a DiscoverFilters object. */
function parsePresetFilters(preset: FilterPreset): DiscoverFilters {
  try {
    const search = JSON.parse(preset.filters) as DiscoverSearch;
    return parseFiltersFromSearch(search);
  } catch {
    // Corrupt payload — fall back to an empty filter set rather than throwing.
    return parseFiltersFromSearch({});
  }
}

/**
 * Quick-access row of saved filter presets shown above the filter area (AC #2).
 * Clicking a chip restores its saved filter combination (AC #3). Right-click or
 * long-press opens a delete confirmation (AC #4); deletion persists via the DB
 * (AC #5).
 */
export function PresetChips({ onApplyPreset, className }: PresetChipsProps) {
  const { data: presets } = useFilterPresets();
  const deletePreset = useDeleteFilterPreset();
  const [pendingDelete, setPendingDelete] = useState<FilterPreset | null>(null);
  const longPressTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Set when a long-press fires so the synthesized click that follows touchend
  // is swallowed instead of also applying the preset (Task 3.4).
  const didLongPress = useRef(false);

  if (!presets || presets.length === 0) return null;

  const startLongPress = (preset: FilterPreset) => {
    didLongPress.current = false;
    longPressTimer.current = setTimeout(() => {
      didLongPress.current = true;
      setPendingDelete(preset);
    }, LONG_PRESS_MS);
  };
  const cancelLongPress = () => {
    if (longPressTimer.current) {
      clearTimeout(longPressTimer.current);
      longPressTimer.current = null;
    }
  };

  const handleConfirmDelete = async () => {
    if (!pendingDelete) return;
    try {
      await deletePreset.mutateAsync(pendingDelete.id);
    } finally {
      setPendingDelete(null);
    }
  };

  return (
    <>
      <div
        className={`flex flex-wrap items-center gap-2 ${className ?? ''}`}
        data-testid="preset-chips"
      >
        <span className="text-xs text-[var(--text-muted)]">快速篩選:</span>
        {presets.map((preset) => (
          <button
            key={preset.id}
            data-testid={`preset-chip-${preset.id}`}
            onClick={() => {
              // A long-press already opened the delete dialog; touchend still
              // synthesizes this click — swallow it so we don't apply the preset
              // underneath the dialog (Task 3.4).
              if (didLongPress.current) {
                didLongPress.current = false;
                return;
              }
              onApplyPreset(parsePresetFilters(preset));
            }}
            onContextMenu={(e) => {
              e.preventDefault();
              setPendingDelete(preset);
            }}
            onTouchStart={() => startLongPress(preset)}
            onTouchEnd={cancelLongPress}
            onTouchMove={cancelLongPress}
            title={`套用「${preset.name}」（右鍵或長按可刪除）`}
            className="rounded-full border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3.5 py-1 text-xs font-medium text-[var(--text-secondary)] hover:border-[var(--accent-primary)] hover:text-white"
          >
            {preset.name}
          </button>
        ))}
      </div>

      {pendingDelete && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
          role="dialog"
          aria-modal="true"
          aria-labelledby="preset-delete-title"
          data-testid="preset-delete-dialog"
          onClick={(e) => {
            if (e.target === e.currentTarget) setPendingDelete(null);
          }}
        >
          <div className="mx-4 w-full max-w-sm rounded-xl bg-[var(--bg-secondary)] p-6 shadow-2xl">
            <h3
              id="preset-delete-title"
              className="mb-2 text-center text-lg font-semibold text-white"
            >
              刪除預設
            </h3>
            <p className="mb-4 text-center text-sm text-[var(--text-secondary)]">
              確定要刪除「{pendingDelete.name}」嗎？
            </p>
            <div className="flex gap-3">
              <button
                type="button"
                onClick={() => setPendingDelete(null)}
                data-testid="preset-delete-cancel"
                className="flex-1 rounded-lg bg-[var(--bg-tertiary)] px-4 py-2 text-sm font-medium text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]"
              >
                取消
              </button>
              <button
                type="button"
                onClick={handleConfirmDelete}
                disabled={deletePreset.isPending}
                data-testid="preset-delete-confirm"
                className="flex-1 rounded-lg bg-[var(--error)] px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
              >
                {deletePreset.isPending ? '刪除中...' : '刪除'}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
