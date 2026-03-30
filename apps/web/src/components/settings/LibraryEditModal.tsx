/**
 * Library Edit/Create Modal for Settings page (Story 7b-4)
 */

import { useState, useEffect } from 'react';
import { X, Plus } from 'lucide-react';
import {
  useMediaLibraries,
  useCreateLibrary,
  useUpdateLibrary,
  useAddLibraryPath,
  useRemoveLibraryPath,
} from '../../hooks/useMediaLibrary';

interface LibraryEditModalProps {
  libraryId?: string; // undefined = create mode
  onClose: () => void;
}

export function LibraryEditModal({ libraryId, onClose }: LibraryEditModalProps) {
  const { data } = useMediaLibraries();
  const createLibrary = useCreateLibrary();
  const updateLibrary = useUpdateLibrary();
  const addPath = useAddLibraryPath();
  const removePath = useRemoveLibraryPath();

  const isEditMode = !!libraryId;
  const existingLibrary = data?.libraries?.find((l) => l.id === libraryId);

  const [name, setName] = useState('');
  const [contentType, setContentType] = useState<'movie' | 'series'>('movie');
  const [newPath, setNewPath] = useState('');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (existingLibrary) {
      setName(existingLibrary.name);
      setContentType(existingLibrary.contentType);
    }
  }, [existingLibrary]);

  const handleSave = async () => {
    setError(null);
    try {
      if (isEditMode && libraryId) {
        await updateLibrary.mutateAsync({ id: libraryId, name, contentType });
      } else {
        await createLibrary.mutateAsync({
          name,
          contentType,
          paths: newPath.trim() ? [newPath.trim()] : undefined,
        });
      }
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Operation failed');
    }
  };

  const handleAddPath = async () => {
    if (!libraryId || !newPath.trim()) return;
    setError(null);
    try {
      await addPath.mutateAsync({ libraryId, path: newPath.trim() });
      setNewPath('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add path');
    }
  };

  const handleRemovePath = async (pathId: string) => {
    if (!libraryId) return;
    try {
      await removePath.mutateAsync({ libraryId, pathId });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove path');
    }
  };

  const isSaving = createLibrary.isPending || updateLibrary.isPending;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div
        className="w-full max-w-md rounded-lg border border-slate-700 bg-slate-900 p-6 shadow-xl"
        data-testid="library-edit-modal"
      >
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-slate-100">
            {isEditMode ? '編輯媒體庫' : '新增媒體庫'}
          </h3>
          <button
            type="button"
            onClick={onClose}
            className="rounded p-1 text-slate-500 hover:bg-slate-800 hover:text-slate-300"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {error && (
          <div className="mb-4 rounded-lg bg-red-900/30 px-3 py-2 text-sm text-red-400">
            {error}
          </div>
        )}

        <div className="mb-4 space-y-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-slate-300">名稱</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="我的電影"
              className="w-full rounded-md border border-slate-600/50 bg-slate-800/60 px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              data-testid="library-name-input"
            />
          </div>

          <div>
            <label className="mb-1 block text-sm font-medium text-slate-300">類型</label>
            <select
              value={contentType}
              onChange={(e) => setContentType(e.target.value as 'movie' | 'series')}
              className="w-full rounded-md border border-slate-600/50 bg-slate-800/60 px-3 py-2 text-sm text-slate-200 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              data-testid="library-type-select"
            >
              <option value="movie">🎬 電影</option>
              <option value="series">📺 影集</option>
            </select>
          </div>

          {/* Paths section (edit mode or single path for create) */}
          <div>
            <label className="mb-1 block text-sm font-medium text-slate-300">資料夾路徑</label>

            {isEditMode && existingLibrary?.paths && (
              <div className="mb-2 space-y-1">
                {existingLibrary.paths.map((p) => (
                  <div
                    key={p.id}
                    className="flex items-center justify-between rounded-md bg-slate-800 px-3 py-1.5"
                  >
                    <span className="truncate font-mono text-xs text-slate-400">{p.path}</span>
                    <button
                      type="button"
                      onClick={() => handleRemovePath(p.id)}
                      className="ml-2 rounded p-0.5 text-slate-500 hover:text-red-400"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </div>
                ))}
              </div>
            )}

            <div className="flex gap-2">
              <input
                type="text"
                value={newPath}
                onChange={(e) => setNewPath(e.target.value)}
                placeholder="/media/movies"
                className="flex-1 rounded-md border border-slate-600/50 bg-slate-800/60 px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                data-testid="library-path-input"
              />
              {isEditMode && (
                <button
                  type="button"
                  onClick={handleAddPath}
                  disabled={!newPath.trim() || addPath.isPending}
                  className="rounded-md bg-slate-700 px-3 py-2 text-sm text-slate-300 hover:bg-slate-600 disabled:opacity-50"
                >
                  <Plus className="h-4 w-4" />
                </button>
              )}
            </div>
          </div>
        </div>

        <div className="flex gap-3">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-slate-600/50 px-4 py-2 text-sm text-slate-300 hover:bg-slate-800"
          >
            取消
          </button>
          <button
            type="button"
            onClick={handleSave}
            disabled={!name.trim() || isSaving}
            className="flex-1 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            data-testid="library-save-button"
          >
            {isSaving ? '儲存中...' : isEditMode ? '儲存變更' : '建立'}
          </button>
        </div>
      </div>
    </div>
  );
}
