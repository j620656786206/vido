// Design ref: ux-design.pen Screen H3 (Paqlk)
/**
 * Explore block create/edit modal — Story 10.3.
 */

import { cloneElement, useEffect, useId, useState } from 'react';
import { Check, X } from 'lucide-react';
import { cn } from '../../lib/utils';
import { genreIdsFor, genreLabel, parseGenreIdList } from '../../lib/genres';
import { useCreateExploreBlock, useUpdateExploreBlock } from '../../hooks/useExploreBlocks';
import type { ExploreBlock, ExploreBlockContentType } from '../../services/exploreBlockService';
import { getSortOptions } from './exploreBlockSort';

interface ExploreBlockEditModalProps {
  block?: ExploreBlock; // undefined = create mode
  onClose: () => void;
}

export function ExploreBlockEditModal({ block, onClose }: ExploreBlockEditModalProps) {
  const createBlock = useCreateExploreBlock();
  const updateBlock = useUpdateExploreBlock();
  const isEditMode = !!block;

  const [name, setName] = useState(block?.name ?? '');
  const [contentType, setContentType] = useState<ExploreBlockContentType>(
    block?.contentType ?? 'movie'
  );
  const [genreIds, setGenreIds] = useState<number[]>(() => parseGenreIdList(block?.genreIds));
  const [language, setLanguage] = useState(block?.language ?? '');
  const [region, setRegion] = useState(block?.region ?? '');
  const [sortBy, setSortBy] = useState(block?.sortBy ?? 'popularity.desc');
  const [maxItems, setMaxItems] = useState(block?.maxItems ?? 20);
  const [error, setError] = useState<string | null>(null);
  const genreLabelId = useId();
  // The content type's TMDb genres, plus any already-picked id they do not
  // contain (hand-typed before the chip picker) — shown so it is never
  // dropped silently; un-picking it removes it.
  const genreChoices = [
    ...genreIdsFor(contentType),
    ...genreIds.filter((id) => !genreIdsFor(contentType).includes(id)),
  ];

  useEffect(() => {
    if (block) {
      setName(block.name);
      setContentType(block.contentType);
      setGenreIds(parseGenreIdList(block.genreIds));
      setLanguage(block.language);
      setRegion(block.region);
      setSortBy(block.sortBy || 'popularity.desc');
      setMaxItems(block.maxItems);
    }
  }, [block]);

  // L1 fix: close modal on Escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

  // H1 fix: reset sort when content type changes to avoid invalid TMDb sort_by
  const handleContentTypeChange = (newType: ExploreBlockContentType) => {
    setContentType(newType);
    // Keep genres both types have (動畫, 劇情…); drop the other type's own
    // (TV has no 28 動作 — TMDb ignores it). An id neither list knows was
    // typed by hand before the chip picker: keep it, never drop it silently.
    const known = new Set([...genreIdsFor('movie'), ...genreIdsFor('tv')]);
    const nextList = genreIdsFor(newType);
    setGenreIds((prev) => prev.filter((id) => nextList.includes(id) || !known.has(id)));
    const validOptions = getSortOptions(newType);
    if (!validOptions.some((opt) => opt.value === sortBy)) {
      setSortBy('popularity.desc');
    }
  };

  const handleSave = async () => {
    setError(null);
    // M3 fix: validate maxItems range before submitting
    const clampedMaxItems = Math.max(1, Math.min(40, maxItems || 20));
    if (clampedMaxItems !== maxItems) setMaxItems(clampedMaxItems);
    try {
      const payload = {
        name,
        contentType,
        genreIds: genreIds.join(','),
        language,
        region,
        sortBy,
        maxItems: clampedMaxItems,
      };
      if (isEditMode && block) {
        await updateBlock.mutateAsync({ id: block.id, ...payload });
      } else {
        await createBlock.mutateAsync(payload);
      }
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : '操作失敗');
    }
  };

  const isSaving = createBlock.isPending || updateBlock.isPending;

  return (
    /* --overlay-scrim is the modal-backdrop token and stays DARK in both themes:
       a paper modal on paper ground needs the same boundary a dark one does.
       Was black/60; the token is 70%. */
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-[var(--overlay-scrim)]"
      role="dialog"
      aria-modal="true"
      aria-labelledby="explore-block-modal-title"
    >
      <div
        className="w-full max-w-md rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-primary)] p-6 shadow-[var(--shadow-xl)]"
        data-testid="explore-block-edit-modal"
      >
        <div className="mb-4 flex items-center justify-between">
          <h3
            id="explore-block-modal-title"
            className="text-lg font-semibold text-[var(--text-primary)]"
          >
            {isEditMode ? '編輯探索區塊' : '新增探索區塊'}
          </h3>
          <button
            type="button"
            onClick={onClose}
            aria-label="關閉"
            className="rounded p-1 text-[var(--text-muted)] hover:bg-[var(--bg-secondary)] hover:text-[var(--text-secondary)]"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {error && (
          <div
            role="alert"
            className="mb-4 rounded-lg bg-[var(--error-tint)] px-3 py-2 text-sm text-[var(--error-text)]"
          >
            {error}
          </div>
        )}

        <div className="space-y-4">
          <Field label="區塊名稱">
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="例如：熱門台劇"
              data-testid="explore-block-name-input"
              className="w-full rounded-md border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
            />
          </Field>

          <Field label="內容類型">
            <select
              value={contentType}
              onChange={(e) => handleContentTypeChange(e.target.value as ExploreBlockContentType)}
              data-testid="explore-block-type-select"
              className="w-full rounded-md border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
            >
              <option value="movie">電影</option>
              <option value="tv">影集</option>
            </select>
          </Field>

          {/* H3 genreField (JtzjF): the content type's genres as toggle chips —
              the v2 FilterChip language of I1-D-v2 (FilterPanel), not H3's pre-v2
              palette. Saved as the same comma-separated id string as before. */}
          <div>
            <span
              id={genreLabelId}
              className="mb-1 block text-sm font-medium text-[var(--text-secondary)]"
            >
              類型篩選
            </span>
            <div
              role="group"
              aria-labelledby={genreLabelId}
              className="flex flex-wrap gap-1.5"
              data-testid="explore-block-genre-chips"
            >
              {genreChoices.map((id) => {
                const active = genreIds.includes(id);
                return (
                  <button
                    key={id}
                    type="button"
                    aria-pressed={active}
                    data-testid={`explore-block-genre-${id}`}
                    onClick={() =>
                      setGenreIds((prev) =>
                        prev.includes(id) ? prev.filter((g) => g !== id) : [...prev, id]
                      )
                    }
                    className={cn(
                      'inline-flex h-9 items-center gap-1 rounded-full border px-3 text-sm transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]',
                      active
                        ? 'border-[var(--accent-primary)] bg-[var(--accent-primary)]/15 text-[var(--accent-text)]'
                        : 'border-transparent bg-[var(--bg-tertiary)] text-[var(--text-secondary)] hover:text-[var(--text-primary)]'
                    )}
                  >
                    {active && <Check className="h-3.5 w-3.5" aria-hidden="true" />}
                    {genreLabel(id)}
                  </button>
                );
              })}
            </div>
            <p className="mt-1 text-xs text-[var(--text-muted)]">不選＝不限類型</p>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <Field label="語言">
              <input
                type="text"
                value={language}
                onChange={(e) => setLanguage(e.target.value)}
                placeholder="zh-TW"
                data-testid="explore-block-language-input"
                className="w-full rounded-md border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
              />
            </Field>
            <Field label="地區">
              <input
                type="text"
                value={region}
                onChange={(e) => setRegion(e.target.value)}
                placeholder="TW"
                data-testid="explore-block-region-input"
                className="w-full rounded-md border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
              />
            </Field>
          </div>

          <Field label="排序">
            <select
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value)}
              data-testid="explore-block-sort-select"
              className="w-full rounded-md border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
            >
              {getSortOptions(contentType).map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </Field>

          <Field label="最大項目數（1–40）">
            <input
              type="number"
              min={1}
              max={40}
              value={maxItems}
              onChange={(e) => setMaxItems(Number(e.target.value))}
              data-testid="explore-block-max-items-input"
              className="w-full rounded-md border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
            />
          </Field>
        </div>

        <div className="mt-6 flex gap-3">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-[var(--border-subtle)]/50 px-4 py-2 text-sm text-[var(--text-secondary)] hover:bg-[var(--bg-secondary)]"
          >
            取消
          </button>
          <button
            type="button"
            onClick={handleSave}
            disabled={!name.trim() || isSaving}
            data-testid="explore-block-save-button"
            className="flex-1 rounded-lg bg-[var(--accent-primary)] px-4 py-2 text-sm font-medium text-[var(--text-on-accent)] hover:bg-[var(--accent-pressed)] disabled:opacity-50"
          >
            {isSaving ? '儲存中...' : isEditMode ? '儲存變更' : '儲存區塊'}
          </button>
        </div>
      </div>
    </div>
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactElement<{ id?: string }>;
}) {
  // Associate the visible label with its (single) form control so screen
  // readers announce the field name (jsx-a11y/label-has-associated-control).
  const id = useId();
  return (
    <div>
      <label htmlFor={id} className="mb-1 block text-sm font-medium text-[var(--text-secondary)]">
        {label}
      </label>
      {cloneElement(children, { id })}
    </div>
  );
}
