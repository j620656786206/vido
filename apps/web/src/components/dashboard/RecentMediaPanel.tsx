import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { ChevronDown, Film } from 'lucide-react';
import { cn } from '../../lib/utils';
import { useRecentMedia } from '../../hooks/useDashboardData';
import type { RecentMedia } from '../../services/mediaService';

interface RecentMediaPanelProps {
  className?: string;
}

export function RecentMediaPanel({ className }: RecentMediaPanelProps) {
  const { data: recentMedia, isLoading } = useRecentMedia(8);
  const [isExpanded, setIsExpanded] = useState(true);

  return (
    <div
      className={cn(
        'rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-secondary)]/50',
        className
      )}
      data-testid="recent-media-panel"
    >
      {/* Collapsible Header (AC4) */}
      <button
        type="button"
        onClick={() => setIsExpanded(!isExpanded)}
        className="flex w-full items-center justify-between border-b border-[var(--border-subtle)] px-4 py-3 text-left lg:cursor-default"
        aria-expanded={isExpanded}
        aria-controls="recent-media-content"
      >
        <div className="flex items-center gap-2">
          <Film className="h-5 w-5 text-[var(--text-secondary)]" />
          <h2 className="text-lg font-semibold text-[var(--text-primary)]">最近新增</h2>
        </div>
        {/* Chevron only visible on mobile */}
        <ChevronDown
          className={cn(
            'h-5 w-5 text-[var(--text-secondary)] transition-transform lg:hidden',
            isExpanded && 'rotate-180'
          )}
        />
      </button>

      {/* Collapsible Content (AC4) */}
      <div
        id="recent-media-content"
        className={cn(
          'overflow-hidden transition-all duration-300',
          isExpanded
            ? 'max-h-[2000px] opacity-100'
            : 'max-h-0 opacity-0 lg:max-h-none lg:opacity-100'
        )}
      >
        {/* Content */}
        <div className="px-4 py-3">
          {isLoading ? (
            <div className="grid grid-cols-4 gap-3" data-testid="recent-media-loading">
              {Array.from({ length: 8 }).map((_, i) => (
                <div key={i} className="animate-pulse">
                  <div className="aspect-[2/3] rounded-lg bg-[var(--bg-tertiary)]" />
                  <div className="mt-1 h-3 w-3/4 rounded bg-[var(--bg-tertiary)]" />
                  <div className="mt-1 h-3 w-1/2 rounded bg-[var(--bg-tertiary)]" />
                </div>
              ))}
            </div>
          ) : recentMedia?.length === 0 ? (
            <div className="py-6 text-center text-sm text-[var(--text-secondary)]">
              媒體庫中還沒有內容
            </div>
          ) : (
            <div className="grid grid-cols-4 gap-3">
              {recentMedia?.map((media) => (
                <MediaCard key={media.id} media={media} />
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="border-t border-[var(--border-subtle)] px-4 py-2">
          <Link
            to="/library"
            className="text-sm text-[var(--accent-primary)] hover:text-blue-300 hover:underline"
          >
            查看全部媒體庫 →
          </Link>
        </div>
      </div>
    </div>
  );
}

function MediaCard({ media }: { media: RecentMedia }) {
  return (
    <Link to="/media/$id" params={{ id: media.id }} className="group relative">
      <div className="aspect-[2/3] overflow-hidden rounded-lg bg-[var(--bg-tertiary)]">
        {media.posterUrl ? (
          <img
            src={media.posterUrl}
            alt={media.title}
            className="h-full w-full object-cover transition-transform group-hover:scale-105"
            loading="lazy"
          />
        ) : (
          <div className="flex h-full items-center justify-center text-[var(--text-muted)]">
            <span className="text-2xl">🎬</span>
          </div>
        )}
        {media.justAdded && (
          <span className="absolute right-1 top-1 rounded bg-[var(--accent-primary)] px-1.5 py-0.5 text-[10px] font-medium text-white">
            剛剛新增
          </span>
        )}
        {/* Hover Quick Action (AC5) */}
        <div className="absolute inset-0 flex items-center justify-center bg-black/50 opacity-0 transition-opacity group-hover:opacity-100">
          <span
            className="rounded-full bg-[var(--accent-primary)] p-2 text-white shadow-lg"
            aria-label={`查看 ${media.title} 詳情`}
          >
            <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
              />
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
              />
            </svg>
          </span>
        </div>
      </div>
      <p className="mt-1 truncate text-xs text-[var(--text-primary)]">{media.title}</p>
      {media.year && <p className="text-xs text-[var(--text-secondary)]">{media.year}</p>}
    </Link>
  );
}
