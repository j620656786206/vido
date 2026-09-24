// Design ref: ux-design.pen Screen B14-D 換海報・狀態與規則 (AM0xm)
/**
 * PosterField — the 海報 column of 修改資訊 (poster-upload-b, B′13 left + B′14).
 *
 * Picking an image only CHOOSES it; nothing is uploaded until the dialog's
 * 儲存 (B′14 rule 2). The field reports the choice up and draws whatever phase
 * the dialog is in (uploading / failed). Every B′14 state is reachable from the
 * keyboard: the drop target is a convenience, 「換一張圖片」 is the real control.
 */

import { useEffect, useId, useRef, useState } from 'react';
import { ImageIcon, Loader2, Upload } from 'lucide-react';
import { cn } from '../../lib/utils';
import { getImageUrl } from '../../lib/image';

export const ACCEPTED_POSTER_TYPES = ['image/jpeg', 'image/png', 'image/webp'];
export const MAX_POSTER_BYTES = 5 * 1024 * 1024;

export type PosterChoice =
  | { kind: 'file'; file: File }
  | { kind: 'url'; url: string; status: 'loading' | 'ok' | 'broken' };

export type PosterPhase = 'idle' | 'uploading' | 'failed';

export const POSTER_COPY = {
  badType: '只能用 JPG、PNG 或 WebP 圖片。',
  tooBig: '這張圖超過 5 MB，請換一張小一點的。',
  uploadFailed: '海報沒有上傳成功，這次的修改都還沒儲存。請再按一次「儲存」。',
  badUrl: '這個網址打不開圖片。',
  pending: '新海報・尚未儲存',
  hintDesktop: 'JPG、PNG 或 WebP，5 MB 以內。也可以直接把圖片拖到海報上。',
  hintPhone: 'JPG、PNG 或 WebP，5 MB 以內。新海報會在按「儲存」後換上。',
} as const;

export interface PosterFieldProps {
  /** The stored poster path (TMDb path, `/posters/…` upload, or absolute URL). */
  currentPoster?: string | null;
  choice: PosterChoice | null;
  onChoiceChange: (choice: PosterChoice | null) => void;
  phase: PosterPhase;
  /** Gallery only: open in a state that otherwise needs a drag or a bad file. */
  initialState?: { dragging?: boolean; error?: 'badType' | 'tooBig' };
}

function validate(file: File): 'badType' | 'tooBig' | null {
  if (!ACCEPTED_POSTER_TYPES.includes(file.type)) return 'badType';
  if (file.size > MAX_POSTER_BYTES) return 'tooBig';
  return null;
}

export function PosterField({
  currentPoster,
  choice,
  onChoiceChange,
  phase,
  initialState,
}: PosterFieldProps) {
  const labelId = useId();
  const errorId = useId();
  const urlInputId = useId();
  const fileRef = useRef<HTMLInputElement>(null);
  const [urlMode, setUrlMode] = useState(choice?.kind === 'url');
  const [dragging, setDragging] = useState(initialState?.dragging ?? false);
  const [fileError, setFileError] = useState<'badType' | 'tooBig' | null>(
    initialState?.error ?? null
  );
  // dragleave fires when the pointer crosses into a child; count enter/leave
  // pairs so the drop state does not flicker (story "已知陷阱").
  const dragDepth = useRef(0);
  const [currentFailed, setCurrentFailed] = useState(false);

  // A picked file previews from a blob URL, released when it is replaced,
  // cancelled or the field unmounts.
  // (Created in an effect, not a memo: StrictMode re-runs effects, and a memoised
  // URL would be revoked by the first cleanup and never recreated.)
  const file = choice?.kind === 'file' ? choice.file : null;
  const [fileUrl, setFileUrl] = useState<string | null>(null);
  useEffect(() => {
    if (!file) {
      setFileUrl(null);
      return;
    }
    const url = URL.createObjectURL(file);
    setFileUrl(url);
    return () => URL.revokeObjectURL(url);
  }, [file]);

  const current = currentFailed ? null : getImageUrl(currentPoster ?? null, 'w342');
  const preview = fileUrl ?? (choice?.kind === 'url' ? choice.url : null) ?? current;
  const isNew = choice !== null;
  // A pasted URL that does not load previews as the empty frame, not as a
  // broken-image glyph; typing a new URL resets it to 'loading' and retries.
  const urlBroken = choice?.kind === 'url' && choice.status === 'broken';
  const busy = phase === 'uploading';

  const pick = (file: File | undefined) => {
    if (!file) return;
    const problem = validate(file);
    setFileError(problem);
    if (!problem) onChoiceChange({ kind: 'file', file });
  };

  const error =
    phase === 'failed'
      ? POSTER_COPY.uploadFailed
      : fileError
        ? POSTER_COPY[fileError]
        : choice?.kind === 'url' && choice.status === 'broken'
          ? POSTER_COPY.badUrl
          : null;

  const secondaryLink =
    'flex h-11 items-center text-sm text-[var(--accent-text)] underline underline-offset-2 hover:no-underline focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] disabled:opacity-50 sm:h-auto sm:self-start';

  return (
    <section
      aria-labelledby={labelId}
      data-testid="metadata-editor-poster-column"
      className="grid shrink-0 grid-cols-[104px_1fr] content-start items-start gap-x-4 gap-y-2 sm:w-[184px] sm:grid-cols-1 sm:gap-y-3"
    >
      <span
        id={labelId}
        className="col-start-2 row-start-1 text-xs text-[var(--text-secondary)] sm:col-start-1 sm:row-start-auto"
      >
        海報
      </span>

      {isNew && (
        <span className="col-start-2 row-start-2 justify-self-start rounded-full bg-[var(--accent-tint)] px-2 py-0.5 text-xs font-semibold text-[var(--accent-text)] sm:col-start-1 sm:row-start-auto">
          {POSTER_COPY.pending}
        </span>
      )}

      <div
        data-testid="poster-drop-target"
        data-dragging={dragging || undefined}
        onDragEnter={(e) => {
          if (busy) return;
          e.preventDefault();
          dragDepth.current += 1;
          setDragging(true);
        }}
        onDragOver={(e) => {
          if (!busy) e.preventDefault();
        }}
        onDragLeave={() => {
          dragDepth.current = Math.max(0, dragDepth.current - 1);
          if (dragDepth.current === 0) setDragging(false);
        }}
        onDrop={(e) => {
          e.preventDefault();
          dragDepth.current = 0;
          setDragging(false);
          if (busy) return;
          setUrlMode(false);
          pick(e.dataTransfer.files?.[0]);
        }}
        className="relative col-start-1 row-span-4 row-start-1 h-[156px] w-[104px] overflow-hidden rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] sm:row-span-1 sm:row-start-auto sm:h-[276px] sm:w-[184px]"
      >
        {preview && !urlBroken ? (
          <img
            key={preview}
            src={preview}
            alt={isNew ? '新的海報' : '目前的海報'}
            data-testid="metadata-editor-poster"
            className="h-full w-full object-cover"
            onLoad={() => {
              if (choice?.kind === 'url' && choice.status !== 'ok') {
                onChoiceChange({ ...choice, status: 'ok' });
              }
            }}
            onError={() => {
              if (choice?.kind === 'url') {
                if (choice.status !== 'broken') onChoiceChange({ ...choice, status: 'broken' });
              } else if (!isNew) {
                setCurrentFailed(true);
              }
            }}
          />
        ) : (
          <div className="flex h-full w-full flex-col items-center justify-center gap-1 text-[var(--text-muted)]">
            <ImageIcon className="h-6 w-6" aria-hidden="true" />
            <span className="text-xs">還沒有海報</span>
          </div>
        )}
        {/* The frame is drawn ABOVE the image: an inset ring on the box itself
            is painted under its children and the poster would hide it. */}
        {(dragging || isNew) && (
          <span
            aria-hidden="true"
            className={cn(
              'pointer-events-none absolute inset-0 rounded-[var(--radius-md)] ring-2 ring-inset',
              phase === 'failed' && !dragging
                ? 'ring-[var(--error)]'
                : 'ring-[var(--accent-primary)]'
            )}
          />
        )}
        {(dragging || busy) && (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-1 bg-[var(--overlay-scrim)] p-2 text-center text-[var(--text-on-scrim)]">
            {busy ? (
              <Loader2 className="h-6 w-6 animate-spin" aria-hidden="true" />
            ) : (
              <Upload className="h-6 w-6" aria-hidden="true" />
            )}
            <span className="text-xs font-semibold">{busy ? '上傳中…' : '放開就用這張'}</span>
          </div>
        )}
      </div>

      <div className="col-start-2 row-start-3 flex min-w-0 flex-col gap-1 sm:col-start-1 sm:row-start-auto sm:gap-3">
        <input
          ref={fileRef}
          type="file"
          accept={ACCEPTED_POSTER_TYPES.join(',')}
          className="sr-only"
          tabIndex={-1}
          aria-hidden="true"
          data-testid="poster-file-input"
          onChange={(e) => {
            pick(e.target.files?.[0]);
            // Picking the same file again must fire change again.
            e.target.value = '';
          }}
        />
        {urlMode ? (
          <>
            <label htmlFor={urlInputId} className="sr-only">
              圖片網址
            </label>
            <input
              id={urlInputId}
              type="url"
              inputMode="url"
              disabled={busy}
              placeholder="https://…/poster.jpg"
              defaultValue={choice?.kind === 'url' ? choice.url : ''}
              aria-invalid={choice?.kind === 'url' && choice.status === 'broken' ? true : undefined}
              aria-describedby={error ? errorId : undefined}
              onChange={(e) => {
                const url = e.target.value.trim();
                onChoiceChange(url ? { kind: 'url', url, status: 'loading' } : null);
              }}
              className="h-11 w-full rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-transparent focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)] sm:h-9"
            />
            <button
              type="button"
              disabled={busy}
              onClick={() => {
                setUrlMode(false);
                if (choice?.kind === 'url') onChoiceChange(null);
              }}
              className={secondaryLink}
            >
              改用上傳
            </button>
          </>
        ) : (
          <>
            <button
              type="button"
              disabled={busy}
              aria-describedby={error ? errorId : undefined}
              onClick={() => fileRef.current?.click()}
              className="flex h-11 w-full items-center justify-center gap-2 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-4 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] disabled:opacity-50 sm:h-9"
            >
              <Upload className="h-4 w-4" aria-hidden="true" />
              {preview ? '換一張圖片' : '上傳圖片'}
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={() => {
                setUrlMode(true);
                setFileError(null);
                if (choice?.kind === 'file') onChoiceChange(null);
              }}
              className={secondaryLink}
            >
              改用圖片網址
            </button>
          </>
        )}
      </div>

      <p
        id={errorId}
        aria-live="polite"
        className={cn(
          'col-start-2 row-start-4 text-xs text-[var(--error-text)] sm:col-start-1 sm:row-start-auto',
          !error && 'sr-only'
        )}
      >
        {error}
      </p>

      <p className="col-span-2 text-xs text-[var(--text-muted)] sm:col-span-1">
        <span className="hidden sm:inline">{POSTER_COPY.hintDesktop}</span>
        <span className="sm:hidden">{POSTER_COPY.hintPhone}</span>
      </p>
    </section>
  );
}

export default PosterField;
