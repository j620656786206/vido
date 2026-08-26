// Design ref: ux-design.pen Screen 7 Search + Filter Desktop (rsAxf)
import { Search } from 'lucide-react';

interface EmptySearchResultsProps {
  query: string;
  onClear: () => void;
}

/**
 * The entrance animation here was DELETED rather than revived.
 *
 * This element carried `animate-in fade-in delay-500 fill-mode-backwards` —
 * tailwindcss-animate classes for a plugin this project never installed, so it
 * has always rendered instantly. Making them work would have been a
 * REGRESSION: a 500ms delay before an answer the user is actively waiting for,
 * then a 320ms fade on top of that.
 *
 * DESIGN.md's Motion section licenses motion for feedback, for a state change,
 * or for work genuinely in flight. "The page finished loading" is none of the
 * three, and an Operate surface must never make someone wait through an
 * entrance. If a fast query ever flashes this state, the fix belongs in the
 * query gating, not in a delay.
 */
export function EmptySearchResults({ query, onClear }: EmptySearchResultsProps) {
  return (
    <div
      className="flex flex-col items-center justify-center py-16 text-center"
      data-testid="empty-search-results"
    >
      <Search className="h-12 w-12 text-[var(--text-muted)] mb-4" aria-hidden="true" />
      <h3 className="text-lg font-medium text-[var(--text-secondary)] mb-2">找不到相關結果</h3>
      <p className="text-sm text-[var(--text-secondary)] mb-6">
        搜尋「{query}」沒有找到匹配的電影或影集
      </p>
      <ul className="text-sm text-[var(--text-muted)] mb-6 space-y-1 text-left">
        <li>• 試試不同的關鍵字或新增媒體</li>
        <li>• 嘗試使用繁體中文或英文搜尋</li>
        <li>• 檢查拼寫是否正確</li>
      </ul>
      <button
        onClick={onClear}
        className="px-4 py-2 bg-[var(--bg-tertiary)] text-[var(--text-secondary)] rounded-lg text-sm
                   hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] transition-colors
                   focus:outline-none focus:ring-2 focus:ring-[var(--accent-primary)]"
        type="button"
      >
        清除搜尋
      </button>
    </div>
  );
}
