/**
 * ParseFailureCard Component (Story 3.7 - Task 7)
 * Displays a media file that failed automatic parsing with manual search option
 */

import { useState } from 'react';
import { Film, Tv, Search, AlertTriangle } from 'lucide-react';
import { cn } from '../../lib/utils';
import { ManualSearchDialog, FallbackStatusDisplay } from '../manual-search';
import type { FallbackStatus } from '../manual-search/ManualSearchDialog';

/**
 * Represents a local media file with parse results
 */
export interface LocalMediaFile {
  id: string;
  filename: string;
  path: string;
  size: number;
  parsedInfo?: {
    title?: string;
    year?: number;
    mediaType?: 'movie' | 'tv';
    season?: number;
    episode?: number;
  };
  metadataStatus: 'pending' | 'success' | 'failed';
  fallbackStatus?: FallbackStatus;
}

export interface ParseFailureCardProps {
  file: LocalMediaFile;
  onMetadataApplied?: () => void;
  className?: string;
}

/**
 * Card component for displaying a media file that failed automatic parsing.
 * Provides a "Manual Search" button to allow the user to find and apply metadata.
 * Implements UX-4: Failure handling friendliness
 */
export function ParseFailureCard({
  file,
  onMetadataApplied,
  className,
}: ParseFailureCardProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  // Use parsed title or extract from filename
  const initialQuery =
    file.parsedInfo?.title ||
    extractTitleFromFilename(file.filename);

  const handleSuccess = () => {
    setIsDialogOpen(false);
    onMetadataApplied?.();
  };

  return (
    <>
      <div
        className={cn(
          'relative flex flex-col rounded-lg border border-amber-500/30 bg-slate-800/50 overflow-hidden',
          className
        )}
        data-testid="parse-failure-card"
      >
        {/* Warning indicator */}
        <div className="absolute top-2 right-2 z-10">
          <span className="flex items-center gap-1 rounded-full bg-amber-500/20 px-2 py-1 text-xs text-amber-400">
            <AlertTriangle className="h-3 w-3" />
            無法識別
          </span>
        </div>

        {/* Poster placeholder */}
        <div className="aspect-[2/3] bg-slate-700/50 flex items-center justify-center">
          {file.parsedInfo?.mediaType === 'tv' ? (
            <Tv className="h-16 w-16 text-slate-500" />
          ) : (
            <Film className="h-16 w-16 text-slate-500" />
          )}
        </div>

        {/* Content */}
        <div className="flex-1 p-3">
          {/* Filename/Title */}
          <h3
            className="text-sm font-medium text-white line-clamp-2 mb-1"
            title={file.filename}
          >
            {file.parsedInfo?.title || file.filename}
          </h3>

          {/* Parsed info if available */}
          {file.parsedInfo && (
            <div className="flex items-center gap-2 text-xs text-slate-400 mb-2">
              {file.parsedInfo.year && (
                <span>{file.parsedInfo.year}</span>
              )}
              {file.parsedInfo.mediaType && (
                <span className="capitalize">
                  {file.parsedInfo.mediaType === 'tv' ? '影集' : '電影'}
                </span>
              )}
              {file.parsedInfo.season && (
                <span>S{file.parsedInfo.season}</span>
              )}
              {file.parsedInfo.episode && (
                <span>E{file.parsedInfo.episode}</span>
              )}
            </div>
          )}

          {/* Fallback status (compact view) */}
          {file.fallbackStatus &&
            file.fallbackStatus.attempts &&
            file.fallbackStatus.attempts.length > 0 && (
              <div className="text-xs text-slate-500 mb-2">
                已嘗試 {file.fallbackStatus.attempts.length} 個來源
              </div>
            )}

          {/* UX-4: Guidance message */}
          <p className="text-xs text-slate-400 mb-3">
            自動識別失敗，請手動搜尋正確的 Metadata
          </p>

          {/* Manual Search button (AC1: Manual Search Dialog) */}
          <button
            onClick={() => setIsDialogOpen(true)}
            className="w-full flex items-center justify-center gap-2 rounded-lg bg-blue-600 hover:bg-blue-700 px-3 py-2 text-sm font-medium text-white transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500"
            data-testid="manual-search-button"
          >
            <Search className="h-4 w-4" />
            手動搜尋
          </button>
        </div>
      </div>

      {/* Manual Search Dialog */}
      <ManualSearchDialog
        isOpen={isDialogOpen}
        onClose={() => setIsDialogOpen(false)}
        initialQuery={initialQuery}
        mediaId={file.id}
        fallbackStatus={file.fallbackStatus}
        onSuccess={handleSuccess}
      />
    </>
  );
}

/**
 * Extracts a human-readable title from a filename.
 * Removes common patterns like resolution, encoding, year in brackets, etc.
 */
function extractTitleFromFilename(filename: string): string {
  // Remove file extension
  let title = filename.replace(/\.[^.]+$/, '');

  // Replace common separators with spaces
  title = title.replace(/[._]/g, ' ');

  // Remove common patterns
  title = title
    .replace(/\b(1080p|720p|480p|2160p|4k)\b/gi, '')
    .replace(/\b(x264|x265|h264|h265|hevc|avc)\b/gi, '')
    .replace(/\b(bluray|bdrip|dvdrip|webrip|hdtv)\b/gi, '')
    .replace(/\b(aac|ac3|dts|mp3)\b/gi, '')
    .replace(/\[(.*?)\]/g, '') // Remove content in brackets
    .replace(/\((.*?)\)/g, '') // Remove content in parentheses
    .replace(/\s{2,}/g, ' ') // Collapse multiple spaces
    .trim();

  return title || filename;
}

export default ParseFailureCard;
