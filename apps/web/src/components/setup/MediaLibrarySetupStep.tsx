import { Film, Tv, Plus, X } from 'lucide-react';
import type { StepProps } from './SetupWizard';

interface LibraryEntry {
  path: string;
  contentType: 'movie' | 'series';
}

export function MediaLibrarySetupStep({ data, onUpdate, onNext, onBack }: StepProps) {
  const libraries: LibraryEntry[] = (data.libraries as LibraryEntry[] | undefined) || [
    { path: '', contentType: 'movie' },
  ];

  const updateLibraries = (updated: LibraryEntry[]) => {
    onUpdate({ libraries: updated } as Record<string, unknown>);
  };

  const addLibrary = () => {
    updateLibraries([...libraries, { path: '', contentType: 'movie' }]);
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
      <h2 className="mb-2 text-lg font-semibold text-slate-100">媒體庫設定</h2>
      <p className="mb-6 text-sm text-slate-400">
        設定您的媒體資料夾路徑和類型。至少需要一個媒體庫。
      </p>

      <div className="mb-4 space-y-3">
        {libraries.map((lib, index) => (
          <div
            key={index}
            className="rounded-lg border border-slate-600/50 bg-slate-800/60 p-4"
            data-testid={`library-entry-${index}`}
          >
            <div className="mb-3">
              <label className="mb-1 block text-xs font-medium text-slate-400">資料夾路徑</label>
              <input
                type="text"
                value={lib.path}
                onChange={(e) => updateEntry(index, 'path', e.target.value)}
                placeholder="/media/movies"
                className="w-full rounded-md border border-slate-600/50 bg-slate-900/60 px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                data-testid={`library-path-${index}`}
              />
            </div>

            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <label className="text-xs font-medium text-slate-400">類型</label>
                <select
                  value={lib.contentType}
                  onChange={(e) => updateEntry(index, 'contentType', e.target.value)}
                  className="rounded-md border border-slate-600/50 bg-slate-900/60 px-3 py-1.5 text-sm text-slate-200 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
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
                  className="rounded-md p-1 text-slate-500 transition-colors hover:bg-slate-700 hover:text-red-400"
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
        className="mb-6 flex w-full items-center justify-center gap-2 rounded-lg border border-dashed border-slate-600/50 py-2.5 text-sm text-slate-400 transition-colors hover:border-blue-500/50 hover:text-blue-400"
        data-testid="add-library-button"
      >
        <Plus className="h-4 w-4" />
        新增媒體庫
      </button>

      <div className="flex gap-3">
        <button
          type="button"
          onClick={onBack}
          className="rounded-lg border border-slate-600/50 px-4 py-2.5 text-sm font-medium text-slate-300 transition-colors hover:bg-slate-800"
          data-testid="back-button"
        >
          上一步
        </button>
        <button
          type="button"
          onClick={onNext}
          disabled={hasEmptyPath}
          className="flex-1 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
          data-testid="next-button"
        >
          下一步
        </button>
      </div>
    </div>
  );
}
