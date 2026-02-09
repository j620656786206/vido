import { cn } from '../../lib/utils';

export interface UnidentifiedFileCardProps {
  filename: string;
  attemptedSources?: string[];
  onManualSearch: () => void;
  onEditFilename: () => void;
  onSkip: () => void;
  className?: string;
}

const sourceLabels: Record<string, string> = {
  tmdb: 'TMDb',
  douban: 'Douban',
  wikipedia: 'Wikipedia',
  ai: 'AI',
  regex: 'Regex',
};

export function UnidentifiedFileCard({
  filename,
  attemptedSources = [],
  onManualSearch,
  onEditFilename,
  onSkip,
  className,
}: UnidentifiedFileCardProps) {
  return (
    <div
      className={cn(
        'rounded-lg border-2 border-dashed border-gray-600 bg-gray-900/50 p-6',
        className
      )}
      role="article"
      aria-label={`無法識別的檔案：${filename}`}
    >
      <div className="flex flex-col items-center space-y-4 text-center">
        {/* File Icon */}
        <div className="flex h-16 w-16 items-center justify-center rounded-lg bg-gray-800">
          <svg
            className="h-8 w-8 text-gray-500"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1.5}
              d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
            />
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1.5}
              d="M12 8v.01M12 12v.01"
            />
          </svg>
        </div>

        {/* Filename */}
        <div className="space-y-1">
          <p className="max-w-[300px] truncate font-mono text-sm text-gray-400" title={filename}>
            {filename}
          </p>
          <p className="text-lg font-medium text-gray-200">無法自動識別</p>
        </div>

        {/* Attempted Sources */}
        {attemptedSources.length > 0 && (
          <div className="flex flex-wrap items-center justify-center gap-2 text-xs text-gray-500">
            <span>已嘗試：</span>
            {attemptedSources.map((source, index) => (
              <span key={source} className="flex items-center gap-1">
                <span className="text-red-400">{sourceLabels[source] || source}</span>
                <span className="text-red-500">✗</span>
                {index < attemptedSources.length - 1 && <span className="text-gray-600">→</span>}
              </span>
            ))}
          </div>
        )}

        {/* Action Buttons */}
        <div className="flex w-full max-w-[200px] flex-col gap-2">
          <button
            onClick={onManualSearch}
            className="flex w-full items-center justify-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-gray-900"
          >
            <svg
              className="h-4 w-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
              />
            </svg>
            手動搜尋
          </button>

          <button
            onClick={onEditFilename}
            className="flex w-full items-center justify-center gap-2 rounded-lg border border-gray-600 bg-transparent px-4 py-2 text-sm font-medium text-gray-300 transition-colors hover:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2 focus:ring-offset-gray-900"
          >
            <svg
              className="h-4 w-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
              />
            </svg>
            編輯檔名
          </button>

          <button
            onClick={onSkip}
            className="w-full px-4 py-2 text-sm text-gray-500 transition-colors hover:text-gray-400 focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2 focus:ring-offset-gray-900"
          >
            稍後處理
          </button>
        </div>
      </div>
    </div>
  );
}
