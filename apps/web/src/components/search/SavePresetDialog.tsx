// Design ref: ux-design.pen Screen AS-3 - Save Filter Preset Modal (i74p2)
// Source: ux-design.pen (Pencil app)
import { useEffect, useState } from 'react';
import { X } from 'lucide-react';
import { useCreateFilterPreset } from '../../hooks/useFilterPresets';
import {
  activeFilterChips,
  serializeFilters,
  type DiscoverFilters,
} from '../../lib/discoverFilters';

const PRESET_NAME_MAX_LENGTH = 30;

interface SavePresetDialogProps {
  /** The filter combination to be saved (AC #1). */
  filters: DiscoverFilters;
  onClose: () => void;
}

/**
 * Modal that prompts for a preset name and saves the current filter state as a
 * named preset (AC #1, Story 11-4 Task 2). Mirrors Screen AS-3: title, helper
 * text, name field (required, ≤30 chars — Task 2.3), a read-only preview of the
 * filters being saved, and cancel/save actions.
 */
export function SavePresetDialog({ filters, onClose }: SavePresetDialogProps) {
  const createPreset = useCreateFilterPreset();
  const [name, setName] = useState('');
  const [error, setError] = useState<string | null>(null);

  // Close on Escape (matches sibling modals).
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  const previewChips = activeFilterChips(filters);

  const handleSave = async () => {
    setError(null);
    const trimmed = name.trim();
    if (!trimmed) {
      setError('請輸入預設名稱');
      return;
    }
    if (trimmed.length > PRESET_NAME_MAX_LENGTH) {
      setError(`預設名稱最多 ${PRESET_NAME_MAX_LENGTH} 個字`);
      return;
    }
    try {
      await createPreset.mutateAsync({
        name: trimmed,
        // Store the URL-param-shaped filters as a JSON string (see service docs).
        filters: JSON.stringify(serializeFilters(filters)),
      });
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : '儲存失敗');
    }
  };

  const isSaving = createPreset.isPending;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
      role="dialog"
      aria-modal="true"
      aria-labelledby="save-preset-modal-title"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        className="mx-4 w-full max-w-md rounded-2xl border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-6 shadow-xl"
        data-testid="save-preset-dialog"
      >
        {/* Header */}
        <div className="mb-2 flex items-center justify-between">
          <h3 id="save-preset-modal-title" className="text-lg font-bold text-[var(--text-primary)]">
            儲存篩選條件
          </h3>
          <button
            type="button"
            onClick={onClose}
            aria-label="關閉"
            className="rounded p-1 text-[var(--text-muted)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-secondary)]"
          >
            <X className="h-[18px] w-[18px]" />
          </button>
        </div>

        <p className="mb-4 text-[13px] text-[var(--text-secondary)]">
          將目前的篩選條件儲存為快速存取預設
        </p>

        {error && (
          <div
            role="alert"
            data-testid="save-preset-error"
            className="mb-4 rounded-lg bg-red-900/30 px-3 py-2 text-sm text-[var(--error)]"
          >
            {error}
          </div>
        )}

        {/* Name field */}
        <div className="mb-4">
          <label
            htmlFor="preset-name-input"
            className="mb-1.5 block text-[13px] font-medium text-[var(--text-secondary)]"
          >
            預設名稱
          </label>
          <input
            id="preset-name-input"
            type="text"
            value={name}
            maxLength={PRESET_NAME_MAX_LENGTH}
            onChange={(e) => setName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleSave();
            }}
            placeholder="例：高評分韓劇"
            autoFocus
            data-testid="preset-name-input"
            className="w-full rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-tertiary)] px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
          />
        </div>

        {/* Preview of the filters being saved */}
        <div className="mb-5">
          <p className="mb-1.5 text-xs text-[var(--text-muted)]">包含的篩選條件：</p>
          <div className="flex flex-wrap gap-1.5" data-testid="save-preset-preview">
            {previewChips.length === 0 ? (
              <span className="text-xs text-[var(--text-muted)]">（無）</span>
            ) : (
              previewChips.map((chip) => (
                <span
                  key={chip.key}
                  className="rounded-full bg-[var(--accent-primary)] px-2 py-0.5 text-[11px] text-white"
                >
                  {chip.label}
                </span>
              ))
            )}
          </div>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-3">
          <button
            type="button"
            onClick={onClose}
            data-testid="save-preset-cancel"
            className="rounded-lg border border-[var(--border-subtle)] px-5 py-2 text-sm font-medium text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]"
          >
            取消
          </button>
          <button
            type="button"
            onClick={handleSave}
            disabled={!name.trim() || isSaving}
            data-testid="save-preset-confirm"
            className="rounded-lg bg-[var(--accent-primary)] px-5 py-2 text-sm font-semibold text-white hover:bg-[var(--accent-pressed)] disabled:opacity-50"
          >
            {isSaving ? '儲存中...' : '儲存'}
          </button>
        </div>
      </div>
    </div>
  );
}
