import { useRef, useMemo } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';
import { PosterCard } from '../media/PosterCard';
import { PosterCardSkeleton } from '../media/PosterCardSkeleton';
import type { LibraryItem } from '../../types/library';

interface LibraryGridProps {
  items: LibraryItem[];
  isLoading?: boolean;
  totalItems?: number;
  density?: 'small' | 'medium' | 'large';
}

const DENSITY_CONFIG = {
  small: { minWidth: 150, skeletonCount: 18 },
  medium: { minWidth: 200, skeletonCount: 12 },
  large: { minWidth: 250, skeletonCount: 8 },
};

function getItemProps(item: LibraryItem) {
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
    };
  }
  return null;
}

export function LibraryGrid({
  items,
  isLoading,
  totalItems = 0,
  density = 'medium',
}: LibraryGridProps) {
  const parentRef = useRef<HTMLDivElement>(null);
  const config = DENSITY_CONFIG[density];
  const useVirtual = totalItems > 1000;

  const gridStyle = useMemo(
    () =>
      `grid grid-cols-2 gap-3 md:grid-cols-[repeat(auto-fill,minmax(${config.minWidth}px,1fr))] md:gap-4`,
    [config.minWidth]
  );

  if (isLoading) {
    return (
      <div data-testid="library-grid-loading" className={gridStyle}>
        {Array.from({ length: config.skeletonCount }).map((_, i) => (
          <PosterCardSkeleton key={i} />
        ))}
      </div>
    );
  }

  if (items.length === 0) {
    return null;
  }

  // Virtual scrolling for large libraries (>1000 items)
  if (useVirtual) {
    return <VirtualGrid items={items} parentRef={parentRef} density={density} />;
  }

  return (
    <div data-testid="library-grid" className={gridStyle}>
      {items.map((item, index) => {
        const props = getItemProps(item);
        if (!props) return null;
        return <PosterCard key={`${props.type}-${props.id}-${index}`} {...props} />;
      })}
    </div>
  );
}

function VirtualGrid({
  items,
  parentRef,
  density,
}: {
  items: LibraryItem[];
  parentRef: React.RefObject<HTMLDivElement | null>;
  density: 'small' | 'medium' | 'large';
}) {
  const config = DENSITY_CONFIG[density];
  // Estimate columns based on common viewport widths
  const estimatedColumns = Math.max(2, Math.floor(1200 / config.minWidth));
  const rowCount = Math.ceil(items.length / estimatedColumns);

  const virtualizer = useVirtualizer({
    count: rowCount,
    getScrollElement: () => parentRef.current,
    estimateSize: () => config.minWidth * 1.5 + 16 + 40, // aspect 2:3 + gap + title
    overscan: 3,
  });

  return (
    <div ref={parentRef} className="h-[calc(100vh-200px)] overflow-auto">
      <div
        style={{
          height: `${virtualizer.getTotalSize()}px`,
          width: '100%',
          position: 'relative',
        }}
      >
        {virtualizer.getVirtualItems().map((virtualRow) => {
          const startIndex = virtualRow.index * estimatedColumns;
          const rowItems = items.slice(startIndex, startIndex + estimatedColumns);

          return (
            <div
              key={virtualRow.key}
              style={{
                position: 'absolute',
                top: 0,
                left: 0,
                width: '100%',
                height: `${virtualRow.size}px`,
                transform: `translateY(${virtualRow.start}px)`,
              }}
              className={`grid grid-cols-2 gap-3 md:grid-cols-[repeat(auto-fill,minmax(${config.minWidth}px,1fr))] md:gap-4`}
            >
              {rowItems.map((item, colIndex) => {
                const props = getItemProps(item);
                if (!props) return null;
                return (
                  <PosterCard
                    key={`${props.type}-${props.id}-${startIndex + colIndex}`}
                    {...props}
                  />
                );
              })}
            </div>
          );
        })}
      </div>
    </div>
  );
}
