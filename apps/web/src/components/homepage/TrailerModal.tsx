import { useEffect, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { X } from 'lucide-react';
import tmdbService from '../../services/tmdb';
import type { MediaType, Video, VideosResponse } from '../../types/tmdb';

const YOUTUBE_EMBED_BASE = 'https://www.youtube-nocookie.com/embed/';
const VALID_VIDEO_KEY = /^[a-zA-Z0-9_-]+$/;

export interface TrailerModalProps {
  open: boolean;
  onClose: () => void;
  mediaType: MediaType;
  tmdbId: number;
  title: string;
}

// Story 10-2 AC #6 — picks the best YouTube trailer (official → newest).
// Returns null when no embeddable trailer exists.
export function pickBestTrailer(results: Video[] | undefined): Video | null {
  if (!results || results.length === 0) return null;

  const youtubeTrailers = results.filter(
    (v) => v.site === 'YouTube' && v.type === 'Trailer' && VALID_VIDEO_KEY.test(v.key)
  );
  if (youtubeTrailers.length === 0) return null;

  // Prefer official; among the same officiality, prefer the most recent.
  return [...youtubeTrailers].sort((a, b) => {
    if (a.official !== b.official) return a.official ? -1 : 1;
    return (b.publishedAt || '').localeCompare(a.publishedAt || '');
  })[0];
}

export function TrailerModal({ open, onClose, mediaType, tmdbId, title }: TrailerModalProps) {
  const { data, isLoading, isError } = useQuery<VideosResponse, Error>({
    queryKey: ['tmdb', 'videos', mediaType, tmdbId],
    queryFn: () =>
      mediaType === 'movie'
        ? tmdbService.getMovieVideos(tmdbId)
        : tmdbService.getTVShowVideos(tmdbId),
    enabled: open && tmdbId > 0,
    staleTime: 30 * 60 * 1000, // 30m
  });

  const trailer = useMemo(() => pickBestTrailer(data?.results), [data]);

  // Escape-key close (AC #6).
  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={`${title} 預告片`}
      data-testid="trailer-modal"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/85 p-4"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="relative w-full max-w-4xl">
        <button
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
