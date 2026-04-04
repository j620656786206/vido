import { useEffect } from 'react';
import { Plus, X } from 'lucide-react';
import type { StepProps } from './SetupWizard';

interface LibraryEntry {
  id: string;
  path: string;
  contentType: 'movie' | 'series';
}

export function MediaLibrarySetupStep({ data, onUpdate, onNext, onBack }: StepProps) {
  const defaultLibrary: LibraryEntry = {
    id: globalThis.crypto.randomUUID(),
    path: '',
    contentType: 'movie',
  };
  const libraries: LibraryEntry[] = (data.libraries as LibraryEntry[] | undefined) || [
    defaultLibrary,
  ];

  useEffect(() => {
    if (!data.libraries) {
      onUpdate({ libraries: [defaultLibrary] } as Record<string, unknown>);
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const updateLibraries = (updated: LibraryEntry[]) => {
    onUpdate({ libraries: updated } as Record<string, unknown>);
  };

  const addLibrary = () => {
    updateLibraries([
      ...libraries,
      { id: globalThis.crypto.randomUUID(), path: '', contentType: 'movie' },
    ]);
  };

  const removeLibrary = (index: number) => {
    if (libraries.length <= 1) return;
    updateLibraries(libraries.filter((_, i) => i !== index));
  };

  const updateEntry = (index: number, field: keyof LibraryEntry, value: string) => {
    const updated = libraries.map((lib, i) => (i === index ? { ...lib, [field]: value } : lib));
    updateLibraries(updated);
  };

  const hasEmptyPath = libraries.some((lib) => !lib.path.trim());

  return (
    <div data-testid="media-library-step">
      <h2 className="mb-2 text-lg font-semibold text-[var(--text-primary)]">媒體庫設定</h2>
      <p className="mb-6 text-sm text-[var(--text-secondary)]">
        設定您的媒體資料夾路徑和類型。至少需要一個媒體庫。
      </p>

      <div className="mb-4 space-y-3">
        {libraries.map((lib, index) => (
          <div
            key={lib.id}
            className="rounded-lg border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 p-4"
            data-testid={`library-entry-${index}`}
          >
            <div className="mb-3">
              <label className="mb-1 block text-xs font-medium text-[var(--text-secondary)]">
                資料夾路徑
              </label>
              <input
                type="text"
                value={lib.path}
                onChange={(e) => updateEntry(index, 'path', e.target.value)}
                placeholder="/media/movies"
                className="w-full rounded-md border border-[var(--border-subtle)]/50 bg-[var(--bg-primary)]/60 px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
                data-testid={`library-path-${index}`}
              />
            </div>

            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <label className="text-xs font-medium text-[var(--text-secondary)]">類型</label>
                <select
                  value={lib.contentType}
                  onChange={(e) => updateEntry(index, 'contentType', e.target.value)}
                  className="rounded-md border border-[var(--border-subtle)]/50 bg-[var(--bg-primary)]/60 px-3 py-1.5 text-sm text-[var(--text-primary)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
                  data-testid={`library-type-${index}`}
                >
                  <option value="movie">🎬 電影</option>
                  <option value="series">📺 影集</option>
                </select>
              </div>

              {libraries.length > 1 && (
                <button
                  type="button"
                  onClick={() => removeLibrary(index)}
                  className="rounded-md p-1 text-[var(--text-muted)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--error)]"
                  data-testid={`library-remove-${index}`}
                  aria-label="移除此媒體庫"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>
          </div>
        ))}
      </div>

      <button
        type="button"
        onClick={addLibrary}
        className="mb-6 flex w-full items-center justify-center gap-2 rounded-lg border border-dashed border-[var(--border-subtle)]/50 py-2.5 text-sm text-[var(--text-secondary)] transition-colors hover:border-[var(--accent-primary)]/50 hover:text-[var(--accent-primary)]"
        data-testid="add-library-button"
      >
        <Plus className="h-4 w-4" />
        新增媒體庫
      </button>

      <div className="flex gap-3">
        <button
          type="button"
          onClick={onBack}
          className="rounded-lg border border-[var(--border-subtle)]/50 px-4 py-2.5 text-sm font-medium text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-secondary)]"
          data-testid="back-button"
        >
          上一步
        </button>
        <button
          type="button"
          onClick={onNext}
          disabled={hasEmptyPath}
          className="flex-1 rounded-lg bg-[var(--accent-primary)] px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-pressed)] disabled:cursor-not-allowed disabled:opacity-50"
          data-testid="next-button"
        >
          下一步
        </button>
      </div>
    </div>
  );
}
