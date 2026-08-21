// Design ref: ux-design.pen Screen E5-D (hUVYm) · E5-M (P0P82x) · J4-D (sPzZT) · J5-D (alrIw)
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

  // 補審 M4 — capability honor. The auto-generator that honours this opt-in is
  // built only when the API runs in `pipeline` mode, and the shipped default is
  // `legacy`. Offering the checkbox there is a promise nothing keeps: the user
  // ticks it, the save succeeds, and no subtitle is ever produced.
  //
  // `!== false` on purpose: an API that does not report the capability at all
  // reads as unknown, and unknown keeps the control — hiding a shipped feature
  // on a missing field would be the worse failure.
  const autoSubtitleSupported = data?.autoSubtitleSupported !== false;

  const [name, setName] = useState('');
  const [contentType, setContentType] = useState<'movie' | 'series'>('movie');
  const [autoSubtitle, setAutoSubtitle] = useState(false);
  const [newPath, setNewPath] = useState('');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (existingLibrary) {
      setName(existingLibrary.name);
      setContentType(existingLibrary.contentType);
      setAutoSubtitle(existingLibrary.autoSubtitle ?? false);
    }
  }, [existingLibrary]);

  const handleSave = async () => {
    setError(null);
    try {
      if (isEditMode && libraryId) {
        await updateLibrary.mutateAsync({
          id: libraryId,
          name,
          contentType,
          // Omitted when unsupported: the field is optional on update, and
          // omitting leaves whatever the library already had untouched rather
          // than silently clearing an opt-in made while the pipeline was on.
          ...(autoSubtitleSupported ? { autoSubtitle } : {}),
        });
      } else {
        await createLibrary.mutateAsync({
          name,
          contentType,
          ...(autoSubtitleSupported ? { autoSubtitle } : {}),
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
        className="w-full max-w-md rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-primary)] p-6 shadow-xl"
        data-testid="library-edit-modal"
      >
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-[var(--text-primary)]">
            {isEditMode ? '編輯媒體庫' : '新增媒體庫'}
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
          <div className="mb-4 rounded-lg bg-red-900/30 px-3 py-2 text-sm text-[var(--error)]">
            {error}
          </div>
        )}

        <div className="mb-4 space-y-4">
          <div>
            <label
              htmlFor="library-name-input"
              className="mb-1 block text-sm font-medium text-[var(--text-secondary)]"
            >
              名稱
            </label>
            <input
              id="library-name-input"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="我的電影"
              className="w-full rounded-md border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
              data-testid="library-name-input"
            />
          </div>

          <div>
            <label
              htmlFor="library-type-select"
              className="mb-1 block text-sm font-medium text-[var(--text-secondary)]"
            >
              類型
            </label>
            <select
              id="library-type-select"
              value={contentType}
              onChange={(e) => setContentType(e.target.value as 'movie' | 'series')}
              className="w-full rounded-md border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
              data-testid="library-type-select"
            >
              <option value="movie">電影</option>
              <option value="series">影集</option>
            </select>
          </div>

          {/* Paths section (edit mode or single path for create) */}
          <div>
            <label
              htmlFor="library-path-input"
              className="mb-1 block text-sm font-medium text-[var(--text-secondary)]"
            >
              資料夾路徑
            </label>

            {isEditMode && existingLibrary?.paths && (
              <div className="mb-2 space-y-1">
                {existingLibrary.paths.map((p) => (
                  <div
                    key={p.id}
                    className="flex items-center justify-between rounded-md bg-[var(--bg-secondary)] px-3 py-1.5"
                  >
                    <span className="truncate font-mono text-xs text-[var(--text-secondary)]">
                      {p.path}
                    </span>
                    <button
                      type="button"
                      onClick={() => handleRemovePath(p.id)}
                      aria-label={`移除路徑 ${p.path}`}
                      className="ml-2 rounded p-0.5 text-[var(--text-muted)] hover:text-[var(--error)]"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </div>
                ))}
              </div>
            )}

            <div className="flex gap-2">
              <input
                id="library-path-input"
                type="text"
                value={newPath}
                onChange={(e) => setNewPath(e.target.value)}
                placeholder="/media/movies"
                className="flex-1 rounded-md border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
                data-testid="library-path-input"
              />
              {isEditMode && (
                <button
                  type="button"
                  onClick={handleAddPath}
                  disabled={!newPath.trim() || addPath.isPending}
                  aria-label="新增路徑"
                  className="rounded-md bg-[var(--bg-tertiary)] px-3 py-2 text-sm text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] disabled:opacity-50"
                >
                  <Plus className="h-4 w-4" />
                </button>
              )}
            </div>
          </div>

          {/* Story 9R-10b AC #2 — free auto-generation opt-in.
              Last field and behind a divider on purpose: the three above describe
              what the library IS, this one describes what Vido DOES with it.
              The copy is frozen by the 2026-08-19 ruling 「花錢須同意」 and must
              keep saying both halves — free work happens, paid work waits — and
              must never imply that scanning itself produces subtitles. */}
          <div
            className="border-t border-[var(--border-subtle)]/50 pt-4"
            data-testid="library-auto-subtitle-field"
          >
            {/* The label WRAPS the input so the whole row is the hit area, and
                still carries htmlFor like every other field in this modal. The
                row is min-h-[44px] because this is the one control here that a
                phone user has to hit deliberately. */}
            <label
              htmlFor="library-auto-subtitle-checkbox"
              className={`flex min-h-[44px] items-start gap-2.5 py-1.5 ${
                autoSubtitleSupported ? 'cursor-pointer' : 'cursor-not-allowed'
              }`}
            >
              <input
                id="library-auto-subtitle-checkbox"
                type="checkbox"
                checked={autoSubtitle}
                disabled={!autoSubtitleSupported}
                // A11y pre-flight (9R-10b-M4): a DISABLED input is skipped by
                // the tab order entirely, so the notice explaining WHY it is
                // disabled would never reach a keyboard/screen-reader user
                // without being programmatically tied to the control. Browse
                // mode still reads the control, and now reads its reason too.
                aria-describedby={
                  autoSubtitleSupported ? undefined : 'library-auto-subtitle-unsupported-notice'
                }
                onChange={(e) => setAutoSubtitle(e.target.checked)}
                className={`mt-0.5 h-5 w-5 shrink-0 rounded accent-[var(--accent-primary)] ${
                  autoSubtitleSupported ? 'cursor-pointer' : 'cursor-not-allowed opacity-60'
                }`}
                data-testid="library-auto-subtitle-checkbox"
              />
              <span
                className={`text-sm font-medium leading-relaxed ${
                  autoSubtitleSupported
                    ? 'text-[var(--text-primary)]'
                    : 'text-[var(--text-disabled)]'
                }`}
              >
                新檔入庫後，自動完成免費的字幕處理
              </span>
            </label>

            {/* 9R-10b-M4 (design J5-D block D) — the unsupported notice sits
                between the checkbox and the description on purpose: the eye
                lands on a greyed control and asks "why", so the answer comes
                first; the description below then answers "what would it do",
                which is what makes the user decide whether to go and change
                the variable at all.
                Both sentences are frozen. The SECOND one is verbatim from the
                API's own 409 suggestion (subtitle_pipeline_handler.go:113) —
                same action, same words, so a user who hit that error over the
                API recognises the sentence here. */}
            {!autoSubtitleSupported && (
              <div
                id="library-auto-subtitle-unsupported-notice"
                className="my-2 rounded-[var(--radius-sm)] border-l-4 border-[var(--info)] bg-[var(--info)]/10 p-3"
                data-testid="library-auto-subtitle-unsupported-notice"
              >
                <p className="text-[13px] font-medium leading-relaxed text-[var(--text-primary)]">
                  字幕生成管線尚未啟用，這個選項無法變更。
                </p>
                <p className="mt-1 text-xs leading-relaxed text-[var(--text-secondary)]">
                  請將{' '}
                  <span
                    className="rounded bg-[var(--bg-tertiary)] px-1 py-0.5 font-mono text-[var(--text-primary)]"
                    data-testid="library-auto-subtitle-env-var"
                  >
                    VIDO_SUBTITLE_PIPELINE_MODE
                  </span>{' '}
                  設為 pipeline 後重啟伺服器。
                </p>
              </div>
            )}

            <div className="space-y-1.5 pl-[30px]">
              <p
                className={`text-xs leading-relaxed ${
                  autoSubtitleSupported
                    ? 'text-[var(--text-secondary)]'
                    : 'text-[var(--text-disabled)]'
                }`}
              >
                影片內建繁體中文字幕會直接沿用，簡體字幕自動轉成繁體。這些都在本機執行，不會產生費用。
              </p>
              <p
                className={`text-xs leading-relaxed ${
                  autoSubtitleSupported
                    ? 'text-[var(--text-secondary)]'
                    : 'text-[var(--text-disabled)]'
                }`}
              >
                需要 AI
                翻譯或語音辨識的影片不會自動處理，它們會留在「產生字幕」清單裡，標好預估金額等你確認。
              </p>
            </div>
          </div>
        </div>

        <div className="flex gap-3">
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
            className="flex-1 rounded-lg bg-[var(--accent-primary)] px-4 py-2 text-sm font-medium text-white hover:bg-[var(--accent-pressed)] disabled:opacity-50"
            data-testid="library-save-button"
          >
            {isSaving ? '儲存中...' : isEditMode ? '儲存變更' : '建立'}
          </button>
        </div>
      </div>
    </div>
  );
}
