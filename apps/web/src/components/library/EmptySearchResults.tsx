import { Search } from 'lucide-react';

interface EmptySearchResultsProps {
  query: string;
  onClear: () => void;
}

export function EmptySearchResults({ query, onClear }: EmptySearchResultsProps) {
  return (
    <div
      className="flex flex-col items-center justify-center py-16 text-center animate-in fade-in duration-500 delay-500 fill-mode-backwards"
      data-testid="empty-search-results"
    >
      <Search className="h-12 w-12 text-slate-500 mb-4" aria-hidden="true" />
      <h3 className="text-lg font-medium text-slate-300 mb-2">找不到相關結果</h3>
      <p className="text-sm text-slate-400 mb-6">搜尋「{query}」沒有找到匹配的電影或影集</p>
      <ul className="text-sm text-slate-500 mb-6 space-y-1 text-left">
        <li>• 試試不同的關鍵字或新增媒體</li>
        <li>• 嘗試使用繁體中文或英文搜尋</li>
        <li>• 檢查拼寫是否正確</li>
      </ul>
      <button
        onClick={onClear}
        className="px-4 py-2 bg-slate-700 text-slate-300 rounded-lg text-sm
                   hover:bg-slate-600 hover:text-white transition-colors
                   focus:outline-none focus:ring-2 focus:ring-blue-500"
        type="button"
      >
        清除搜尋
      </button>
    </div>
  );
}
