import { Link } from '@tanstack/react-router';
import { cn } from '../../lib/utils';
import { useRecentMedia } from '../../hooks/useDashboardData';
import type { RecentMedia } from '../../services/mediaService';

interface RecentMediaPanelProps {
  className?: string;
}

export function RecentMediaPanel({ className }: RecentMediaPanelProps) {
  const { data: recentMedia, isLoading } = useRecentMedia(8);

  return (
    <div
      className={cn('rounded-lg border border-slate-700 bg-slate-800/50', className)}
      data-testid="recent-media-panel"
    >
      {/* Header */}
      <div className="flex items-center justify-between border-b border-slate-700 px-4 py-3">
        <h2 className="text-lg font-semibold text-slate-100">最近新增</h2>
      </div>

      {/* Content */}
      <div className="px-4 py-3">
        {isLoading ? (
          <div className="grid grid-cols-4 gap-3" data-testid="recent-media-loading">
            {Array.from({ length: 8 }).map((_, i) => (
              <div key={i} className="animate-pulse">
                <div className="aspect-[2/3] rounded-lg bg-slate-700" />
                <div className="mt-1 h-3 w-3/4 rounded bg-slate-700" />
                <div className="mt-1 h-3 w-1/2 rounded bg-slate-700" />
              </div>
            ))}
          </div>
        ) : recentMedia?.length === 0 ? (
          <div className="py-6 text-center text-sm text-slate-400">媒體庫中還沒有內容</div>
        ) : (
          <div className="grid grid-cols-4 gap-3">
            {recentMedia?.map((media) => (
              <MediaCard key={media.id} media={media} />
            ))}
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="border-t border-slate-700 px-4 py-2">
        <Link to="/library" className="text-sm text-blue-400 hover:text-blue-300 hover:underline">
          查看全部媒體庫 →
        </Link>
      </div>
    </div>
  );
}

function MediaCard({ media }: { media: RecentMedia }) {
  return (
    <Link to="/media/$id" params={{ id: media.id }} className="group relative">
      <div className="aspect-[2/3] overflow-hidden rounded-lg bg-slate-700">
        {media.posterUrl ? (
          <img
            src={media.posterUrl}
            alt={media.title}
            className="h-full w-full object-cover transition-transform group-hover:scale-105"
            loading="lazy"
          />
        ) : (
          <div className="flex h-full items-center justify-center text-slate-500">
            <span className="text-2xl">🎬</span>
          </div>
        )}
        {media.justAdded && (
          <span className="absolute right-1 top-1 rounded bg-blue-600 px-1.5 py-0.5 text-[10px] font-medium text-white">
            剛剛新增
          </span>
        )}
      </div>
      <p className="mt-1 truncate text-xs text-slate-200">{media.title}</p>
      {media.year && <p className="text-xs text-slate-400">{media.year}</p>}
    </Link>
  );
}
