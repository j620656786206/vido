import { Link } from '@tanstack/react-router';
import { SearchX, File, Folder, HardDrive, Clock3, CircleAlert, Search } from 'lucide-react';

interface FallbackFailedProps {
  title: string;
  mediaType?: 'movie' | 'tv';
  filePath?: string;
  fileSize?: number;
  createdAt?: string;
  parseStatus?: string;
  onEditClick: () => void;
}

function formatFileSize(bytes: number): string {
  return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB';
}

function parseStatusLabel(status: string | undefined): string {
  if (status === 'failed') return '比對失敗';
  return '尚未比對';
}

export function FallbackFailed({
  title,
  mediaType = 'movie',
  filePath,
  fileSize,
  createdAt,
  parseStatus,
  onEditClick,
}: FallbackFailedProps) {
  // Extract filename from path
  const fileName = filePath?.split('/').pop() ?? '';
  // Extract directory from path
  const dirPath = filePath?.substring(0, filePath.lastIndexOf('/') + 1) ?? '';
  // Search query: strip file extension from title
  const searchQuery = title.replace(/\.\w{2,4}$/, '');

  return (
    <div data-testid="fallback-failed" className="px-4 py-6 md:px-6">
      {/* Inline icon + title row */}
      <div className="flex items-center gap-3">
        <SearchX className="h-6 w-6 flex-shrink-0 text-gray-400" />
        <h2 className="text-lg font-semibold text-white" data-testid="fallback-failed-title">
          {mediaType === 'tv' ? '我們找不到這部電視節目的資料' : '我們找不到這部電影的資料'}
        </h2>
      </div>

      {/* Secondary description */}
      <p className="mt-2 text-sm text-gray-400">你可以手動搜尋，或等待系統自動比對</p>

      {/* File info section (AC #4) */}
      <div className="mt-6" data-testid="fallback-file-info">
        <h3 className="mb-3 text-sm font-semibold text-gray-400">檔案資訊</h3>

        <div className="space-y-2.5">
          {fileName && (
            <div className="flex items-center gap-2.5 text-sm" data-testid="file-info-name">
              <File className="h-4 w-4 flex-shrink-0 text-gray-500" />
              <span className="truncate font-mono text-gray-300" title={fileName}>
                {fileName}
              </span>
            </div>
          )}

          {dirPath && (
            <div className="flex items-center gap-2.5 text-sm" data-testid="file-info-path">
              <Folder className="h-4 w-4 flex-shrink-0 text-gray-500" />
              <span className="truncate font-mono text-gray-300" title={dirPath}>
                {dirPath}
              </span>
            </div>
          )}

          {fileSize != null && fileSize > 0 && (
            <div className="flex items-center gap-2.5 text-sm" data-testid="file-info-size">
              <HardDrive className="h-4 w-4 flex-shrink-0 text-gray-500" />
              <span className="font-mono text-gray-300">{formatFileSize(fileSize)}</span>
            </div>
          )}

          {createdAt && (
            <div className="flex items-center gap-2.5 text-sm" data-testid="file-info-date">
              <Clock3 className="h-4 w-4 flex-shrink-0 text-gray-500" />
              <span className="text-gray-300">{new Date(createdAt).toLocaleString('zh-TW')}</span>
            </div>
          )}

          <div className="flex items-center gap-2.5 text-sm" data-testid="file-info-status">
            <CircleAlert className="h-4 w-4 flex-shrink-0 text-amber-500" />
            <span className="text-amber-400">{parseStatusLabel(parseStatus)}</span>
          </div>
        </div>
      </div>

      {/* CTA section (AC #5) */}
      <div className="mt-6 space-y-3" data-testid="fallback-cta">
        <Link
          to="/search"
          search={{ q: searchQuery }}
          className="flex w-full items-center justify-center gap-2 rounded-lg bg-blue-600 px-4 py-3 text-sm font-medium text-white transition-colors hover:bg-blue-700"
          data-testid="cta-search-metadata"
        >
          <Search className="h-4 w-4" />
          搜尋 Metadata
        </Link>

        <button
          onClick={onEditClick}
          className="w-full text-center text-sm font-medium text-blue-400 transition-colors hover:text-blue-300"
          data-testid="cta-manual-edit"
        >
          手動編輯
        </button>
      </div>
    </div>
  );
}

export default FallbackFailed;
