// Design ref: ux-design.pen Screen HP-1 Homepage Desktop (sAaCR)
import { useEffect, useMemo, useRef } from 'react';
import { useQuery } from '@tanstack/react-query';
import { X } from 'lucide-react';
import tmdbService from '../../services/tmdb';
import type { MediaType, VideosResponse } from '../../types/tmdb';
import { pickBestTrailer } from '../../lib/trailers';

const YOUTUBE_EMBED_BASE = 'https://www.youtube-nocookie.com/embed/';

export interface TrailerModalProps {
  open: boolean;
  onClose: () => void;
  mediaType: MediaType;
  tmdbId: number;
  title: string;
}

// Selector for elements eligible for keyboard focus inside the dialog.
const FOCUSABLE_SELECTOR =
  'a[href], button:not([disabled]), iframe, [tabindex]:not([tabindex="-1"])';

export function TrailerModal({ open, onClose, mediaType, tmdbId, title }: TrailerModalProps) {
  const dialogRef = useRef<HTMLDivElement | null>(null);
  const closeButtonRef = useRef<HTMLButtonElement | null>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);

  const { data, isLoading, isError } = useQuery<VideosResponse, Error>({
    queryKey: ['tmdb', 'videos', mediaType, tmdbId],
    queryFn: () =>
      mediaType === 'movie'
        ? tmdbService.getMovieVideos(tmdbId)
        : tmdbService.getTVShowVideos(tmdbId),
    enabled: open && tmdbId > 0,
    staleTime: 30 * 60 * 1000, // 30m
    // Keep failures snappy — empty-state fallback should appear within ~1 retry,
    // not 4+ seconds of silent backoff. (Code review L1.)
    retry: 1,
  });

  const trailer = useMemo(() => pickBestTrailer(data?.results), [data]);

  // H2 fix: focus management for aria-modal dialog.
  // 1. On open: remember the trigger and move focus into the dialog.
  // 2. While open: trap Tab cycles inside the dialog.
  // 3. On close: restore focus to the original trigger.
  useEffect(() => {
    if (!open) return;
    previousFocusRef.current = document.activeElement as HTMLElement | null;
    // Move focus to the close button as a safe initial target.
    closeButtonRef.current?.focus();
    return () => {
      previousFocusRef.current?.focus?.();
    };
  }, [open]);

  // Escape-key close + focus trap (AC #6 + H2).
  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
        return;
      }
      if (e.key !== 'Tab' || !dialogRef.current) return;
      const focusable = Array.from(
        dialogRef.current.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)
      ).filter((el) => !el.hasAttribute('disabled'));
      if (focusable.length === 0) {
        e.preventDefault();
        return;
      }
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      const active = document.activeElement as HTMLElement | null;
      if (e.shiftKey && active === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && active === last) {
        e.preventDefault();
        first.focus();
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      ref={dialogRef}
      role="dialog"
      aria-modal="true"
      aria-label={`${title} 預告片`}
      data-testid="trailer-modal"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/85 p-4"
    >
      {/* Mouse-only dismiss affordance; keyboard users close via Escape. */}
      <div
        aria-hidden="true"
        data-testid="trailer-modal-backdrop"
        className="absolute inset-0"
        onClick={onClose}
      />
      <div className="relative w-full max-w-4xl">
        <button
          ref={closeButtonRef}
          onClick={onClose}
          aria-label="關閉預告片"
          data-testid="trailer-modal-close"
          className="absolute -top-10 right-0 flex h-8 w-8 items-center justify-center rounded-full bg-white/20 text-white transition-colors hover:bg-white/40"
        >
          <X className="h-5 w-5" />
        </button>

        {isLoading && (
          <div
            data-testid="trailer-modal-loading"
            className="flex aspect-video w-full items-center justify-center rounded-lg bg-[var(--bg-secondary)] text-[var(--text-secondary)]"
          >
            載入預告片中…
          </div>
        )}

        {!isLoading && (isError || !trailer) && (
          <div
            data-testid="trailer-modal-empty"
            className="flex aspect-video w-full items-center justify-center rounded-lg bg-[var(--bg-secondary)] text-[var(--text-secondary)]"
          >
            找不到預告片
          </div>
        )}

        {!isLoading && trailer && (
          <div className="aspect-video w-full overflow-hidden rounded-lg bg-black">
            <iframe
              src={`${YOUTUBE_EMBED_BASE}${trailer.key}?autoplay=1`}
              title={`${title} 預告片`}
              allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
              allowFullScreen
              data-testid="trailer-modal-iframe"
              className="h-full w-full"
            />
          </div>
        )}
      </div>
    </div>
  );
}
