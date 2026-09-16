// Design ref: ux-design.pen Screen B7-M (7UnDy)
// Test-only today (visual gallery; the v2 detail page does not render it) — B7-M has
// no live implementer, so this pointer is honest. Revival: dsr-2b-flow-b-no-metadata-states.
// Old reference was the deleted desktop frame 4e (wQOkg).
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
      <Loader2
        className="h-10 w-10 animate-spin text-[var(--accent-text)]"
        data-testid="pending-spinner"
      />

      {/* Primary message */}
      <h2 className="mt-5 text-lg font-semibold text-[var(--text-primary)]">正在搜尋電影資訊⋯</h2>

      {/* Secondary description */}
      <p className="mt-2 text-sm text-[var(--text-secondary)]">
        系統正在比對檔案名稱與 TMDb 資料庫
      </p>

      {/* Progress bar */}
      <div className="mt-5 h-1 w-full max-w-xs overflow-hidden rounded-full bg-[var(--bg-tertiary)]">
        <div
          className="h-full animate-pulse rounded-full bg-[var(--accent-primary)]"
          style={{ width: '60%' }}
          data-testid="pending-progress"
        />
      </div>

      {/* Filename hint */}
      <p
        className="mt-4 max-w-xs truncate font-mono text-xs text-[var(--text-muted)]"
        title={filename}
        data-testid="pending-filename"
      >
        {filename}
      </p>
    </div>
  );
}

export default FallbackPending;
