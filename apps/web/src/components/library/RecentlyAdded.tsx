import { Link } from '@tanstack/react-router';
import { PosterCard } from '../media/PosterCard';
import { PosterCardSkeleton } from '../media/PosterCardSkeleton';
import { useRecentlyAdded } from '../../hooks/useLibrary';
import type { LibraryItem } from '../../types/library';

const SEVEN_DAYS_MS = 7 * 24 * 60 * 60 * 1000;

function isWithin7Days(dateStr: string): boolean {
  return Date.now() - new Date(dateStr).getTime() < SEVEN_DAYS_MS;
}

function getCardProps(item: LibraryItem) {
  if (item.type === 'movie' && item.movie) {
    const m = item.movie;
    return {
      id: m.tmdb_id ?? 0,
      type: 'movie' as const,
      title: m.title,
      originalTitle: m.original_title,
      posterPath: m.poster_path ?? null,
      releaseDate: m.release_date,
      voteAverage: m.vote_average,
      overview: m.overview,
      metadataSource: m.metadata_source,
      isNew: isWithin7Days(m.created_at),
    };
  }
  if (item.type === 'series' && item.series) {
    const s = item.series;
    return {
      id: s.tmdb_id ?? 0,
      type: 'tv' as const,
      title: s.title,
      originalTitle: s.original_title,
      posterPath: s.poster_path ?? null,
      releaseDate: s.first_air_date,
      voteAverage: s.vote_average,
      overview: s.overview,
      metadataSource: s.metadata_source,
      isNew: isWithin7Days(s.created_at),
    };
  }
  return null;
}

export function RecentlyAdded() {
  const { data, isLoading } = useRecentlyAdded(20);

  if (!isLoading && (!data || data.length === 0)) {
    return null;
  }

  return (
    <section data-testid="recently-added-section" className="mb-8">
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-xl font-semibold text-white">最近新增</h2>
        <Link
          to="/library"
          search={{ sortBy: 'created_at', sortOrder: 'desc' }}
          className="text-sm text-blue-400 hover:underline"
        >
          查看全部 &gt;
        </Link>
      </div>
      <div className="flex gap-4 overflow-x-auto pb-4 scrollbar-thin">
        {isLoading
          ? Array.from({ length: 8 }).map((_, i) => (
              <div key={i} className="w-[180px] flex-shrink-0">
                <PosterCardSkeleton />
              </div>
            ))
          : data?.map((item, index) => {
              const props = getCardProps(item);
              if (!props) return null;
              return (
                <div key={`${props.type}-${props.id}-${index}`} className="w-[180px] flex-shrink-0">
                  <PosterCard {...props} />
                </div>
              );
            })}
      </div>
    </section>
  );
}
