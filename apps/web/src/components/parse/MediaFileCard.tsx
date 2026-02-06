/**
 * MediaFileCard Component (Story 3.10 - Task 8)
 * Displays a local media file with parse status indicator
 * AC1: Status Icons in File List
 */

import { Film, Tv, Loader2 } from 'lucide-react';
import { cn } from '../../lib/utils';
import { ParseStatusBadge, ParsingStatusBadge } from './ParseStatusBadge';
import { CompactErrorSummary } from './ErrorDetailsPanel';
import type { ParseStatus, ParseStep } from './types';

export interface MediaFile {
  id: string;
  filename: string;
  path: string;
  size: number;
  mediaType?: 'movie' | 'tv';
  parseStatus: ParseStatus;
  parsedInfo?: {
    title?: string;
    year?: number;
    season?: number;
    episode?: number;
  };
  parseSteps?: ParseStep[];
  posterPath?: string;
}

export interface MediaFileCardProps {
  /** The media file to display */
  file: MediaFile;
  /** Whether this file is currently being parsed */
  isParsing?: boolean;
  /** Called when the card is clicked */
  onClick?: () => void;
  /** Additional CSS classes */
  className?: string;
}

/**
 * Card component for displaying a local media file with parse status
 * Implements AC1: Status Icons in File List
 */
export function MediaFileCard({
  file,
  isParsing = false,
  onClick,
  className,
}: MediaFileCardProps) {
  const displayTitle = file.parsedInfo?.title || extractTitleFromFilename(file.filename);
  const year = file.parsedInfo?.year;
  const isTV = file.mediaType === 'tv';

  return (
    <div
      className={cn(
        'relative flex flex-col rounded-lg border bg-slate-800/50 overflow-hidden',
        'cursor-pointer hover:bg-slate-800 transition-colors',
        file.parseStatus === 'failed' && 'border-red-500/30',
        file.parseStatus === 'needs_ai' && 'border-yellow-500/30',
        file.parseStatus === 'success' && 'border-slate-700',
        file.parseStatus === 'pending' && 'border-slate-700',
        className
      )}
      onClick={onClick}
      data-testid="media-file-card"
      data-status={file.parseStatus}
    >
      {/* Status Badge in top-right */}
      <div className="absolute top-2 right-2 z-10">
        {isParsing ? (
          <ParsingStatusBadge size="sm" />
        ) : (
          <ParseStatusBadge status={file.parseStatus} size="sm" />
        )}
      </div>

      {/* Poster Area */}
      <div className="aspect-[2/3] bg-slate-700/50 flex items-center justify-center">
        {file.posterPath ? (
          <img
            src={file.posterPath}
            alt={displayTitle}
            className="w-full h-full object-cover"
          />
        ) : (
          <div className="flex flex-col items-center text-slate-500">
            {isParsing ? (
              <Loader2 className="h-12 w-12 animate-spin text-blue-500" />
            ) : isTV ? (
              <Tv className="h-12 w-12" />
            ) : (
              <Film className="h-12 w-12" />
            )}
          </div>
        )}
      </div>

      {/* Content */}
      <div className="flex-1 p-3">
        {/* Title */}
        <h3
          className="text-sm font-medium text-white line-clamp-2 mb-1"
          title={file.filename}
        >
          {displayTitle}
        </h3>

        {/* Metadata line */}
        <div className="flex items-center gap-2 text-xs text-slate-400 mb-1">
          {year && <span>{year}</span>}
          {file.mediaType && (
            <span>{file.mediaType === 'tv' ? '影集' : '電影'}</span>
          )}
          {file.parsedInfo?.season && <span>S{file.parsedInfo.season}</span>}
          {file.parsedInfo?.episode && <span>E{file.parsedInfo.episode}</span>}
        </div>

        {/* File size */}
        <div className="text-xs text-slate-500">{formatFileSize(file.size)}</div>

        {/* Error summary for failed parses */}
        {file.parseStatus === 'failed' && file.parseSteps && (
          <CompactErrorSummary steps={file.parseSteps} className="mt-2" />
        )}
      </div>
    </div>
  );
}

/**
 * List view row for media file
 */
export function MediaFileRow({
  file,
  isParsing = false,
  onClick,
  className,
}: MediaFileCardProps) {
  const displayTitle = file.parsedInfo?.title || extractTitleFromFilename(file.filename);
  const year = file.parsedInfo?.year;

  return (
    <div
      className={cn(
        'flex items-center gap-4 px-4 py-3 rounded-lg',
        'cursor-pointer hover:bg-slate-800 transition-colors',
        'border-b border-slate-700/50 last:border-0',
        className
      )}
      onClick={onClick}
      data-testid="media-file-row"
      data-status={file.parseStatus}
    >
      {/* Status Icon */}
      <div className="flex-shrink-0">
        {isParsing ? (
          <ParsingStatusBadge size="sm" />
        ) : (
          <ParseStatusBadge status={file.parseStatus} size="sm" />
        )}
      </div>

      {/* Title and info */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-medium text-white truncate">{displayTitle}</h3>
          {year && <span className="text-xs text-slate-400 flex-shrink-0">{year}</span>}
        </div>
        <div className="text-xs text-slate-500 truncate">{file.filename}</div>
      </div>

      {/* File size */}
      <div className="text-xs text-slate-400 flex-shrink-0">
        {formatFileSize(file.size)}
      </div>
    </div>
  );
}

/**
 * Grid of media files with status indicators
 */
export function MediaFileGrid({
  files,
  parsingIds = [],
  onFileClick,
  className,
}: {
  files: MediaFile[];
  parsingIds?: string[];
  onFileClick?: (file: MediaFile) => void;
  className?: string;
}) {
  return (
    <div
      className={cn(
        'grid grid-cols-2 gap-3',
        'md:grid-cols-[repeat(auto-fill,minmax(160px,1fr))] md:gap-4',
        'lg:grid-cols-[repeat(auto-fill,minmax(200px,1fr))]',
        className
      )}
      data-testid="media-file-grid"
    >
      {files.map((file) => (
        <MediaFileCard
          key={file.id}
          file={file}
          isParsing={parsingIds.includes(file.id)}
          onClick={() => onFileClick?.(file)}
        />
      ))}
    </div>
  );
}

/**
 * List of media files with status indicators
 */
export function MediaFileList({
  files,
  parsingIds = [],
  onFileClick,
  className,
}: {
  files: MediaFile[];
  parsingIds?: string[];
  onFileClick?: (file: MediaFile) => void;
  className?: string;
}) {
  return (
    <div className={cn('flex flex-col', className)} data-testid="media-file-list">
      {files.map((file) => (
        <MediaFileRow
          key={file.id}
          file={file}
          isParsing={parsingIds.includes(file.id)}
          onClick={() => onFileClick?.(file)}
        />
      ))}
    </div>
  );
}

// Utility functions

function extractTitleFromFilename(filename: string): string {
  let title = filename.replace(/\.[^.]+$/, '');
  title = title.replace(/[._]/g, ' ');
  title = title
    .replace(/\b(1080p|720p|480p|2160p|4k)\b/gi, '')
    .replace(/\b(x264|x265|h264|h265|hevc|avc)\b/gi, '')
    .replace(/\b(bluray|bdrip|dvdrip|webrip|hdtv)\b/gi, '')
    .replace(/\b(aac|ac3|dts|mp3)\b/gi, '')
    .replace(/\[(.*?)\]/g, '')
    .replace(/\((.*?)\)/g, '')
    .replace(/\s{2,}/g, ' ')
    .trim();
  return title || filename;
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

export default MediaFileCard;
