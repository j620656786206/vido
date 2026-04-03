import { Loader2 } from 'lucide-react';

interface FallbackPendingProps {
  filename: string;
}

export function FallbackPending({ filename }: FallbackPendingProps) {
  return (
    <div
      data-testid="fallback-pending"
      className="flex flex-col items-center px-6 py-8 text-center"
    >
      {/* Spinner */}
      <Loader2 className="h-10 w-10 animate-spin text-blue-400" data-testid="pending-spinner" />

      {/* Primary message */}
      <h2 className="mt-5 text-lg font-semibold text-white">正在搜尋電影資訊⋯</h2>

      {/* Secondary description */}
      <p className="mt-2 text-sm text-gray-400">系統正在比對檔案名稱與 TMDb 資料庫</p>

      {/* Progress bar */}
      <div className="mt-5 h-1 w-full max-w-xs overflow-hidden rounded-full bg-gray-700">
        <div
          className="h-full animate-pulse rounded-full bg-blue-500"
          style={{ width: '60%' }}
          data-testid="pending-progress"
        />
      </div>

      {/* Filename hint */}
      <p
        className="mt-4 max-w-xs truncate font-mono text-xs text-gray-500"
        title={filename}
        data-testid="pending-filename"
      >
        {filename}
      </p>
    </div>
  );
}

export default FallbackPending;
