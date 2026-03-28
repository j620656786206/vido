import { useState, useRef, useCallback } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';
import { PosterCard } from '../media/PosterCard';
import { PosterCardSkeleton } from '../media/PosterCardSkeleton';
import { PosterCardMenu } from './PosterCardMenu';
import { useDeleteLibraryItem, useReparseItem, useExportItem } from '../../hooks/useLibrary';
import type { LibraryItem } from '../../types/library';

interface LibraryGridProps {
  items: LibraryItem[];
  isLoading?: boolean;
  totalItems?: number;
  density?: 'small' | 'medium' | 'large';
  highlightQuery?: string;
  selectionMode?: boolean;
  selectedIds?: Set<string>;
  onSelect?: (id: string, e: React.MouseEvent) => void;
}

const DENSITY_CONFIG = {
  small: { minWidth: 150, skeletonCount: 18 },
  medium: { minWidth: 200, skeletonCount: 12 },
  large: { minWidth: 250, skeletonCount: 8 },
};

// Static Tailwind classes — dynamic template literals get purged at build time
const GRID_CLASS: Record<string, string> = {
  small: 'grid grid-cols-2 gap-3 md:grid-cols-[repeat(auto-fill,minmax(150px,1fr))] md:gap-4',
  medium: 'grid grid-cols-2 gap-3 md:grid-cols-[repeat(auto-fill,minmax(200px,1fr))] md:gap-4',
  large: 'grid grid-cols-2 gap-3 md:grid-cols-[repeat(auto-fill,minmax(250px,1fr))] md:gap-4',
};

function getItemProps(item: LibraryItem) {
  if (item.type === 'movie' && item.movie) {
    const m = item.movie;
    return {
      id: m.id,
      itemId: m.id,
      itemType: 'movie' as const,
      type: 'movie' as const,
      title: m.title,
      originalTitle: m.originalTitle,
      posterPath: m.posterPath ?? null,
      releaseDate: m.releaseDate,
      voteAverage: m.voteAverage,
      overview: m.overview,
      metadataSource: m.metadataSource,
    };
  }
  if (item.type === 'series' && item.series) {
    const s = item.series;
    return {
      id: s.id,
      itemId: s.id,
      itemType: 'series' as const,
      type: 'tv' as const,
      title: s.title,
      originalTitle: s.originalTitle,
      posterPath: s.posterPath ?? null,
      releaseDate: s.firstAirDate,
      voteAverage: s.voteAverage,
      overview: s.overview,
      metadataSource: s.metadataSource,
    };
  }
  return null;
}

export function LibraryGrid({
  items,
  isLoading,
  totalItems = 0,
  density = 'medium',
  highlightQuery,
  selectionMode,
  selectedIds,
  onSelect,
}: LibraryGridProps) {
  const parentRef = useRef<HTMLDivElement>(null);
  const config = DENSITY_CONFIG[density];
  const useVirtual = totalItems > 1000;

  const [menuState, setMenuState] = useState<{
    itemId: string;
    itemType: 'movie' | 'series';
  } | null>(null);

  const deleteMutation = useDeleteLibraryItem();
  const reparseMutation = useReparseItem();
  const exportMutation = useExportItem();

  const handleMenuClick = useCallback(
    (itemId: string, itemType: 'movie' | 'series') => (_e: React.MouseEvent) => {
      setMenuState({ itemId, itemType });
    },
    []
  );

  const handleCloseMenu = useCallback(() => setMenuState(null), []);

  const handleViewDetails = useCallback(() => {
    // TODO: Open detail panel (Story 5.6)
    setMenuState(null);
  }, []);

  const handleReparse = useCallback(() => {
    if (menuState) {
      reparseMutation.mutate({ type: menuState.itemType, id: menuState.itemId });
    }
    setMenuState(null);
  }, [menuState, reparseMutation]);

  const handleExport = useCallback(() => {
    if (menuState) {
      exportMutation.mutate({ type: menuState.itemType, id: menuState.itemId });
    }
    setMenuState(null);
  }, [menuState, exportMutation]);

  const handleDelete = useCallback(() => {
    if (menuState) {
      deleteMutation.mutate({ type: menuState.itemType, id: menuState.itemId });
    }
    setMenuState(null);
  }, [menuState, deleteMutation]);

  const gridStyle = GRID_CLASS[density];

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
    return (
      <VirtualGrid
        items={items}
        parentRef={parentRef}
        density={density}
        highlightQuery={highlightQuery}
        onMenuClick={handleMenuClick}
        menuState={menuState}
        onCloseMenu={handleCloseMenu}
        onViewDetails={handleViewDetails}
        onReparse={handleReparse}
        onExport={handleExport}
        onDelete={handleDelete}
        selectionMode={selectionMode}
        selectedIds={selectedIds}
        onSelect={onSelect}
      />
    );
  }

  return (
    <div data-testid="library-grid" className={gridStyle}>
      {items.map((item, index) => {
        const props = getItemProps(item);
        if (!props) return null;
        const { itemId, itemType, ...cardProps } = props;
        return (
          <div key={`${props.type}-${props.id}-${index}`} className="relative">
            <PosterCard
              {...cardProps}
              highlightQuery={highlightQuery}
              onMenuClick={selectionMode ? undefined : handleMenuClick(itemId, itemType)}
              selectable={selectionMode}
              selected={selectionMode && selectedIds?.has(itemId)}
              onSelect={selectionMode ? (e) => onSelect?.(itemId, e) : undefined}
            />
            {!selectionMode && menuState?.itemId === itemId && (
              <PosterCardMenu
                isOpen={true}
                onClose={handleCloseMenu}
                onViewDetails={handleViewDetails}
                onReparse={handleReparse}
                onExport={handleExport}
                onDelete={handleDelete}
              />
            )}
          </div>
        );
      })}
    </div>
  );
}

interface VirtualGridProps {
  items: LibraryItem[];
  parentRef: React.RefObject<HTMLDivElement | null>;
  density: 'small' | 'medium' | 'large';
  highlightQuery?: string;
  onMenuClick: (itemId: string, itemType: 'movie' | 'series') => (e: React.MouseEvent) => void;
  menuState: { itemId: string; itemType: 'movie' | 'series' } | null;
  onCloseMenu: () => void;
  onViewDetails: () => void;
  onReparse: () => void;
  onExport: () => void;
  onDelete: () => void;
  selectionMode?: boolean;
  selectedIds?: Set<string>;
  onSelect?: (id: string, e: React.MouseEvent) => void;
}

function VirtualGrid({
  items,
  parentRef,
  density,
  highlightQuery,
  onMenuClick,
  menuState,
  onCloseMenu,
  onViewDetails,
  onReparse,
  onExport,
  onDelete,
  selectionMode,
  selectedIds,
  onSelect,
}: VirtualGridProps) {
  const config = DENSITY_CONFIG[density];
  // Use window.innerWidth when available, fallback to 1200
  const viewportWidth = typeof window !== 'undefined' ? window.innerWidth : 1200;
  const estimatedColumns = Math.max(2, Math.floor(viewportWidth / config.minWidth));
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
              className={GRID_CLASS[density]}
            >
              {rowItems.map((item, colIndex) => {
                const props = getItemProps(item);
                if (!props) return null;
                const { itemId, itemType, ...cardProps } = props;
                return (
                  <div
                    key={`${props.type}-${props.id}-${startIndex + colIndex}`}
                    className="relative"
                  >
                    <PosterCard
                      {...cardProps}
                      highlightQuery={highlightQuery}
                      onMenuClick={selectionMode ? undefined : onMenuClick(itemId, itemType)}
                      selectable={selectionMode}
                      selected={selectionMode && selectedIds?.has(itemId)}
                      onSelect={selectionMode ? (e) => onSelect?.(itemId, e) : undefined}
                    />
                    {!selectionMode && menuState?.itemId === itemId && (
                      <PosterCardMenu
                        isOpen={true}
                        onClose={onCloseMenu}
                        onViewDetails={onViewDetails}
                        onReparse={onReparse}
                        onExport={onExport}
                        onDelete={onDelete}
                      />
                    )}
                  </div>
                );
              })}
            </div>
          );
        })}
      </div>
    </div>
  );
}
