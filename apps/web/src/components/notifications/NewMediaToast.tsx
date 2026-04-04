interface NewMediaToastProps {
  title: string;
  posterUrl?: string;
  mediaType: 'movie' | 'tv';
}

const mediaTypeLabels: Record<string, string> = {
  movie: '電影',
  tv: '影集',
};

export function NewMediaToast({ title, posterUrl, mediaType }: NewMediaToastProps) {
  return (
    <div className="flex items-center gap-3 rounded-lg bg-[var(--bg-secondary)] p-3 shadow-lg">
      {/* Poster thumbnail */}
      <div className="h-12 w-8 shrink-0 overflow-hidden rounded bg-[var(--bg-tertiary)]">
        {posterUrl ? (
          <img src={posterUrl} alt={title} className="h-full w-full object-cover" />
        ) : (
          <div
            className="flex h-full items-center justify-center text-[var(--text-muted)]"
            data-testid="poster-placeholder"
          >
            <span className="text-xs">🎬</span>
          </div>
        )}
      </div>

      {/* Content */}
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium text-[var(--text-primary)]">已新增至媒體庫</p>
        <p className="truncate text-xs text-[var(--text-secondary)]">{title}</p>
        <span className="text-[10px] text-[var(--text-secondary)]">
          {mediaTypeLabels[mediaType]}
        </span>
      </div>

      {/* Success indicator */}
      <span className="shrink-0 text-emerald-400">✓</span>
    </div>
  );
}
