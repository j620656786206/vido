// Time-bomb-exempt: new Date().getFullYear() is a fallback default for the year field; gallery fixture always passes initialData.year so the path is unreachable in baseline render (Sally)
// Design ref: ux-design.pen Screen B13p-D 修改資訊 (AFuPx)
// Design ref: ux-design.pen Screen B13p-M 修改資訊 (oktn2)
/**
 * MetadataEditorDialog — 修改資訊 (Story 3.8 AC1/AC4; rebuilt by poster-upload-a).
 *
 * One component, two looks (ui/mobileSheet): a centred 760px dialog from `sm:`,
 * a bottom sheet with a pinned footer below. The left column shows the current
 * poster read-only; changing it is poster-upload-b.
 */

import { useEffect, useRef, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Loader2, X } from 'lucide-react';
import { cn } from '../../lib/utils';
import { genreNamesFor } from '../../lib/genres';
import { useUpdateMetadata, useUploadPoster } from '../../hooks/useMetadataEditor';
import { Dialog, DialogClose, DialogContent, DialogTitle } from '../ui/Dialog';
import { MOBILE_SHEET_CONTENT, SheetGrabber } from '../ui/mobileSheet';
import { GenreSelector } from './GenreSelector';
import { CastEditor } from './CastEditor';
import { PosterField, type PosterChoice, type PosterPhase } from './PosterField';

const metadataSchema = z.object({
  title: z.string().min(1, '片名為必填'),
  titleEnglish: z.string().optional(),
  // A cleared number input is NaN; without its own message zod says
  // "Expected number, received nan".
  year: z
    .number({ invalid_type_error: '請輸入年份' })
    .min(1900, '年份必須大於 1900')
    .max(2100, '年份必須小於 2100'),
  genres: z.array(z.string()),
  director: z.string().optional(),
  cast: z.array(z.string()),
  overview: z.string().optional(),
});

export type MetadataFormData = z.infer<typeof metadataSchema>;

export interface MediaMetadata {
  id: string;
  mediaType: 'movie' | 'series';
  title: string;
  titleEnglish?: string;
  year?: number;
  genres?: string[];
  director?: string;
  cast?: string[];
  overview?: string;
  /** The stored poster path (TMDb path, `/posters/…` upload, or an absolute URL). */
  posterUrl?: string;
}

export interface MetadataEditorDialogProps {
  isOpen: boolean;
  onClose: () => void;
  mediaId: string;
  mediaType: 'movie' | 'series';
  initialData: MediaMetadata;
  onSuccess: () => void;
}

const FORM_ID = 'metadata-editor-form';

const INPUT =
  'h-11 w-full rounded-[var(--radius-md)] border bg-[var(--bg-secondary)] px-3 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] transition-colors focus:border-transparent focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)] sm:h-9';
const LABEL = 'mb-1 block text-xs text-[var(--text-secondary)]';

function toFormValues(data: MediaMetadata): MetadataFormData {
  return {
    title: data.title || '',
    titleEnglish: data.titleEnglish || '',
    year: data.year || new Date().getFullYear(),
    genres: data.genres || [],
    director: data.director || '',
    cast: data.cast || [],
    overview: data.overview || '',
  };
}

export function MetadataEditorDialog({
  isOpen,
  onClose,
  mediaId,
  mediaType,
  initialData,
  onSuccess,
}: MetadataEditorDialogProps) {
  // Radix returns focus to a DialogTrigger; 修改資訊 is a plain button on the
  // detail page, so remember what had focus and go back to it.
  const openerRef = useRef<HTMLElement | null>(null);
  useEffect(() => {
    if (isOpen && document.activeElement instanceof HTMLElement) {
      openerRef.current = document.activeElement;
    }
  }, [isOpen]);

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        data-testid="metadata-editor-dialog"
        aria-describedby={undefined}
        // An edit form: a stray click on the scrim must not throw the edits away.
        // Esc, ✕ and 取消 still close it.
        onPointerDownOutside={(event) => event.preventDefault()}
        onInteractOutside={(event) => event.preventDefault()}
        // Esc in a field that uses it itself (CastEditor's add box) must not also
        // close the dialog; Radix hears Esc on the document before React does.
        onEscapeKeyDown={(event) => {
          if (event.target instanceof HTMLElement && event.target.dataset.escapeLocal) {
            event.preventDefault();
          }
        }}
        // Radix would focus the first tabbable — the ✕. Start in 片名 instead: the
        // dialog opens to be edited, and a ring on ✕ reads as "about to close".
        onOpenAutoFocus={(event) => {
          const title = document.getElementById('metadata-title');
          if (title) {
            event.preventDefault();
            title.focus();
          }
        }}
        onCloseAutoFocus={(event) => {
          const opener = openerRef.current;
          if (opener?.isConnected) {
            event.preventDefault();
            opener.focus();
          }
        }}
        // ui/Dialog's own ✕ says "Close" and is 16px; the header draws a 44px 關閉.
        closeClassName="hidden"
        className={cn(
          'flex max-h-[92vh] flex-col gap-0 overflow-hidden p-0',
          MOBILE_SHEET_CONTENT,
          'sm:bottom-auto sm:left-1/2 sm:right-auto sm:top-1/2 sm:max-h-[85vh] sm:w-[calc(100vw-4rem)] sm:max-w-[760px] sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-[var(--radius-lg)] sm:border sm:border-[var(--border-subtle)]'
        )}
      >
        {/* Mounted only while open, so every open starts from initialData. */}
        <EditorBody
          mediaId={mediaId}
          mediaType={mediaType}
          initialData={initialData}
          onClose={onClose}
          onSuccess={onSuccess}
        />
      </DialogContent>
    </Dialog>
  );
}

function EditorBody({
  mediaId,
  mediaType,
  initialData,
  onClose,
  onSuccess,
}: Omit<MetadataEditorDialogProps, 'isOpen'>) {
  const updateMutation = useUpdateMetadata();
  const {
    register,
    handleSubmit,
    formState: { errors, isDirty },
    watch,
    setValue,
  } = useForm<MetadataFormData>({
    resolver: zodResolver(metadataSchema),
    defaultValues: toFormValues(initialData),
  });

  const genres = watch('genres');
  const cast = watch('cast');
  const uploadMutation = useUploadPoster();
  const [posterChoice, setPosterChoice] = useState<PosterChoice | null>(null);
  const [posterPhase, setPosterPhase] = useState<PosterPhase>('idle');
  // Set once the picked file is on the server, so a retry after a FIELD
  // failure saves the fields without uploading the image again.
  const posterUploaded = useRef(false);
  // The poster went up but the fields did not: 取消 can no longer undo the
  // poster, so the footer must stop promising that it will.
  const [posterSavedFieldsFailed, setPosterSavedFieldsFailed] = useState(false);

  const urlNotReady = posterChoice?.kind === 'url' && posterChoice.status !== 'ok';
  const busy = posterPhase === 'uploading' || updateMutation.isPending;
  const canSave = !busy && !urlNotReady && (isDirty || posterChoice !== null);

  // B′14 rule 3: upload the poster first; only when it is on the server are the
  // fields saved. A failed upload saves nothing and keeps the dialog open.
  const onSubmit = async (data: MetadataFormData) => {
    if (posterChoice?.kind === 'file' && !posterUploaded.current) {
      setPosterPhase('uploading');
      try {
        await uploadMutation.mutateAsync({ mediaId, mediaType, file: posterChoice.file });
        posterUploaded.current = true;
        setPosterPhase('idle');
      } catch {
        setPosterPhase('failed');
        return;
      }
    }
    // A poster-only change must not rewrite the fields: a metadata save also
    // marks the item as hand-edited.
    const fieldsToSave = isDirty || posterChoice?.kind === 'url';
    try {
      if (fieldsToSave) {
        await updateMutation.mutateAsync({
          id: mediaId,
          mediaType,
          ...data,
          // Only a pasted URL goes through the metadata write; an uploaded
          // file already stored its own path and must not be overwritten.
          ...(posterChoice?.kind === 'url' ? { posterUrl: posterChoice.url } : {}),
        });
      }
      onSuccess();
      onClose();
    } catch {
      // Shown in the footer from updateMutation.error.
      if (posterUploaded.current) setPosterSavedFieldsFailed(true);
    }
  };

  return (
    <>
      <SheetGrabber />
      <div className="flex h-14 shrink-0 items-center justify-between border-b border-[var(--border-subtle)] pl-4 pr-2 sm:pl-6">
        <DialogTitle className="text-base font-semibold">修改資訊</DialogTitle>
        <DialogClose
          aria-label="關閉"
          className="flex h-11 w-11 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]"
        >
          <X className="h-[18px] w-[18px]" aria-hidden="true" />
        </DialogClose>
      </div>

      <form
        id={FORM_ID}
        // zod owns the messages (片名為必填…) — not the browser's own bubbles.
        noValidate
        onSubmit={handleSubmit(onSubmit)}
        className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-4 sm:flex-row sm:gap-6 sm:p-6"
      >
        {/* Poster column (B′13 left, B′14 states). */}
        <PosterField
          currentPoster={initialData.posterUrl ?? null}
          choice={posterChoice}
          onChoiceChange={(next) => {
            setPosterChoice(next);
            setPosterPhase('idle');
            posterUploaded.current = false;
            setPosterSavedFieldsFailed(false);
          }}
          phase={posterPhase}
        />

        {/* Fields (B′13 right). DOM order is the desktop order; the phone moves
            導演 up beside 年份 (B13p-M) with `order`. */}
        <fieldset
          disabled={busy}
          className="grid min-w-0 flex-1 grid-cols-2 content-start gap-x-3 gap-y-4 sm:grid-cols-[1fr_120px]"
        >
          <div className="order-1 col-span-2 sm:col-span-1">
            <label htmlFor="metadata-title" className={LABEL}>
              片名 <span className="text-[var(--error-text)]">*</span>
            </label>
            <input
              id="metadata-title"
              type="text"
              {...register('title')}
              aria-invalid={errors.title ? true : undefined}
              aria-describedby={errors.title ? 'metadata-title-error' : undefined}
              className={cn(
                INPUT,
                errors.title ? 'border-[var(--error)]' : 'border-[var(--border-subtle)]'
              )}
            />
            {errors.title && (
              <p id="metadata-title-error" className="mt-1 text-xs text-[var(--error-text)]">
                {errors.title.message}
              </p>
            )}
          </div>

          <div className="order-2">
            <label htmlFor="metadata-year" className={LABEL}>
              年份
            </label>
            <input
              id="metadata-year"
              type="number"
              {...register('year', { valueAsNumber: true })}
              aria-invalid={errors.year ? true : undefined}
              aria-describedby={errors.year ? 'metadata-year-error' : undefined}
              min={1900}
              max={2100}
              className={cn(
                INPUT,
                errors.year ? 'border-[var(--error)]' : 'border-[var(--border-subtle)]'
              )}
            />
            {errors.year && (
              <p id="metadata-year-error" className="mt-1 text-xs text-[var(--error-text)]">
                {errors.year.message}
              </p>
            )}
          </div>

          <div className="order-4 col-span-2 sm:order-3">
            <label htmlFor="metadata-title-english" className={LABEL}>
              英文片名
            </label>
            <input
              id="metadata-title-english"
              type="text"
              {...register('titleEnglish')}
              className={cn(INPUT, 'border-[var(--border-subtle)]')}
            />
          </div>

          <div className="order-5 col-span-2 sm:order-4">
            <span id="metadata-genres-label" className={LABEL}>
              類型
            </span>
            <GenreSelector
              labelId="metadata-genres-label"
              selected={genres}
              options={genreNamesFor(mediaType)}
              onChange={(next) => setValue('genres', next, { shouldDirty: true })}
            />
          </div>

          <div className="order-3 sm:order-5 sm:col-span-2">
            <label htmlFor="metadata-director" className={LABEL}>
              導演
            </label>
            <input
              id="metadata-director"
              type="text"
              {...register('director')}
              className={cn(INPUT, 'border-[var(--border-subtle)]')}
            />
          </div>

          <div className="order-6 col-span-2">
            <span id="metadata-cast-label" className={LABEL}>
              演員
            </span>
            <CastEditor
              labelId="metadata-cast-label"
              cast={cast}
              onChange={(next) => setValue('cast', next, { shouldDirty: true })}
            />
          </div>

          <div className="order-7 col-span-2">
            <label htmlFor="metadata-overview" className={LABEL}>
              簡介
            </label>
            <textarea
              id="metadata-overview"
              {...register('overview')}
              rows={4}
              className="w-full resize-none rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] transition-colors focus:border-transparent focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)]"
            />
          </div>
        </fieldset>
      </form>

      <div className="flex shrink-0 flex-col gap-3 border-t border-[var(--border-subtle)] px-4 pb-6 pt-4 sm:flex-row sm:items-center sm:px-6 sm:pb-4">
        {updateMutation.error ? (
          <p role="alert" className="text-sm text-[var(--error-text)] sm:flex-1">
            更新失敗：{updateMutation.error.message}
          </p>
        ) : null}
        {posterSavedFieldsFailed ? (
          <p role="status" className="text-xs text-[var(--text-secondary)] sm:flex-1">
            新海報已經換上了；其他欄位還沒存，請再按一次「儲存」。
          </p>
        ) : (
          posterChoice &&
          !updateMutation.error && (
            <p className="hidden text-xs text-[var(--text-secondary)] sm:block sm:flex-1">
              新海報會在按「儲存」後換上；按「取消」就不會動到目前的海報。
            </p>
          )
        )}
        <div className="flex gap-3 sm:ml-auto">
          <button
            type="button"
            onClick={onClose}
            className="h-11 flex-1 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] sm:h-9 sm:flex-none"
          >
            取消
          </button>
          <button
            type="submit"
            form={FORM_ID}
            disabled={!canSave}
            className="flex h-11 flex-1 items-center justify-center gap-2 rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-5 text-sm font-semibold text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-pressed)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] disabled:cursor-not-allowed disabled:opacity-50 sm:h-9 sm:flex-none"
          >
            {updateMutation.isPending && (
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
            )}
            {busy ? '儲存中…' : '儲存'}
          </button>
        </div>
      </div>
    </>
  );
}

export default MetadataEditorDialog;
