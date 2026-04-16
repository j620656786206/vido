/**
 * Explore block create/edit modal — Story 10.3.
 */

import { useEffect, useState } from 'react';
import { X } from 'lucide-react';
import { useCreateExploreBlock, useUpdateExploreBlock } from '../../hooks/useExploreBlocks';
import type { ExploreBlock, ExploreBlockContentType } from '../../services/exploreBlockService';

interface ExploreBlockEditModalProps {
  block?: ExploreBlock; // undefined = create mode
  onClose: () => void;
}

const SORT_OPTIONS: Array<{ value: string; label: string }> = [
  { value: 'popularity.desc', label: '熱門度（高→低）' },
  { value: 'vote_average.desc', label: '評分（高→低）' },
  { value: 'primary_release_date.desc', label: '發行日期（新→舊）' },
  { value: 'first_air_date.desc', label: '首播日期（新→舊）' },
  { value: 'revenue.desc', label: '票房（高→低）' },
];

export function ExploreBlockEditModal({ block, onClose }: ExploreBlockEditModalProps) {
  const createBlock = useCreateExploreBlock();
  const updateBlock = useUpdateExploreBlock();
  const isEditMode = !!block;

  const [name, setName] = useState(block?.name ?? '');
  const [contentType, setContentType] = useState<ExploreBlockContentType>(
    block?.contentType ?? 'movie'
  );
  const [genreIds, setGenreIds] = useState(block?.genreIds ?? '');
  const [language, setLanguage] = useState(block?.language ?? '');
  const [region, setRegion] = useState(block?.region ?? '');
  const [sortBy, setSortBy] = useState(block?.sortBy ?? 'popularity.desc');
  const [maxItems, setMaxItems] = useState(block?.maxItems ?? 20);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (block) {
      setName(block.name);
      setContentType(block.contentType);
      setGenreIds(block.genreIds);
      setLanguage(block.language);
      setRegion(block.region);
      setSortBy(block.sortBy || 'popularity.desc');
      setMaxItems(block.maxItems);
    }
  }, [block]);

  const handleSave = async () => {
    setError(null);
    try {
      const payload = {
        name,
        contentType,
        genreIds,
        language,
        region,
        sortBy,
        maxItems,
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
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div
        className="w-full max-w-md rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-primary)] p-6 shadow-xl"
        data-testid="explore-block-edit-modal"
      >
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-[var(--text-primary)]">
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
            className="mb-4 rounded-lg bg-red-900/30 px-3 py-2 text-sm text-[var(--error)]"
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
              onChange={(e) => setContentType(e.target.value as ExploreBlockContentType)}
              data-testid="explore-block-type-select"
              className="w-full rounded-md border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
            >
              <option value="movie">🎬 電影</option>
              <option value="tv">📺 影集</option>
            </select>
          </Field>

          <Field label="類型 ID（逗號分隔 TMDb genre IDs，可留空）">
            <input
              type="text"
              value={genreIds}
              onChange={(e) => setGenreIds(e.target.value)}
              placeholder="例如：28,12"
              data-testid="explore-block-genre-input"
              className="w-full rounded-md border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
            />
          </Field>

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
              {SORT_OPTIONS.map((opt) => (
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
            className="flex-1 rounded-lg bg-[var(--accent-primary)] px-4 py-2 text-sm font-medium text-white hover:bg-[var(--accent-pressed)] disabled:opacity-50"
          >
            {isSaving ? '儲存中...' : isEditMode ? '儲存變更' : '儲存區塊'}
          </button>
        </div>
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="mb-1 block text-sm font-medium text-[var(--text-secondary)]">{label}</label>
      {children}
    </div>
  );
}
