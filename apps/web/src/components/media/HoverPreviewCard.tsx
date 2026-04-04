import { getGenreNames } from '../../lib/genres';

export interface HoverPreviewCardProps {
  title: string;
  originalTitle?: string;
  overview?: string;
  genreIds?: number[];
}

export function HoverPreviewCard({
  title,
  originalTitle,
  overview,
  genreIds = [],
}: HoverPreviewCardProps) {
  const genres = getGenreNames(genreIds, 3);

  return (
    <div
      data-testid="hover-preview-card"
      className="absolute left-0 right-0 top-full z-10 mt-2 hidden rounded-lg bg-[var(--bg-secondary)] p-3 shadow-xl lg:block"
    >
      {/* Original title if different */}
      {originalTitle && originalTitle !== title && (
        <p data-testid="original-title" className="mb-1 text-xs text-[var(--text-secondary)]">
          {originalTitle}
        </p>
      )}

      {/* Genres */}
      {genres.length > 0 && (
        <div data-testid="genres-container" className="mb-2 flex flex-wrap gap-1">
          {genres.map((genre) => (
            <span
              key={genre}
              className="rounded bg-[var(--bg-tertiary)] px-2 py-0.5 text-xs text-[var(--text-secondary)]"
            >
              {genre}
            </span>
          ))}
        </div>
      )}

      {/* Overview */}
      {overview && (
        <p data-testid="overview" className="line-clamp-3 text-xs text-[var(--text-secondary)]">
          {overview}
        </p>
      )}
    </div>
  );
}
