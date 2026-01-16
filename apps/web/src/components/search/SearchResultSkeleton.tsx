import { cn } from '../../lib/utils';

interface SearchResultSkeletonProps {
  count?: number;
  className?: string;
}

export function SearchResultSkeleton({
  count = 5,
  className,
}: SearchResultSkeletonProps) {
  return (
    <div className={cn('space-y-4', className)}>
      {Array.from({ length: count }).map((_, index) => (
        <div
          key={index}
          className="animate-pulse flex space-x-4 p-4 bg-slate-800 rounded-lg"
        >
          {/* Poster skeleton */}
          <div className="w-16 h-24 bg-slate-700 rounded flex-shrink-0" />

          {/* Content skeleton */}
          <div className="flex-1 space-y-3 py-1">
            <div className="h-5 bg-slate-700 rounded w-3/4" />
            <div className="h-4 bg-slate-700 rounded w-1/2" />
            <div className="flex items-center space-x-2">
              <div className="h-4 bg-slate-700 rounded w-16" />
              <div className="h-5 bg-slate-700 rounded-full w-14" />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
