import { useState, useCallback, useEffect, useRef } from 'react';
import { Search, X, ChevronDown, ChevronUp, Eye, Download, Check, Loader2 } from 'lucide-react';
import { cn } from '../../lib/utils';
import { useSubtitleSearch, type SortField } from '../../hooks/useSubtitleSearch';
import type { SubtitleSearchResult } from '../../services/subtitleService';

interface SubtitleSearchDialogProps {
  mediaId: string;
  mediaType: 'movie' | 'series';
  mediaTitle: string;
  mediaFilePath: string;
  mediaResolution?: string;
  productionCountry?: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDownloadSuccess?: () => void;
}

const PROVIDERS = [
  { id: 'assrt', label: 'Assrt (射手網)' },
  { id: 'opensubtitles', label: 'OpenSubtitles' },
  { id: 'zimuku', label: 'Zimuku (字幕庫)' },
];

export function SubtitleSearchDialog({
  mediaId,
  mediaType,
  mediaTitle,
  mediaFilePath,
  mediaResolution,
  productionCountry,
  open,
  onOpenChange,
  onDownloadSuccess,
}: SubtitleSearchDialogProps) {
  const [query, setQuery] = useState(mediaTitle);
  const [selectedProviders, setSelectedProviders] = useState<string[]>(
    PROVIDERS.map((p) => p.id),
  );
  const [previewOpen, setPreviewOpen] = useState<string | null>(null);
  const [toast, setToast] = useState<string | null>(null);
  const dialogRef = useRef<HTMLDivElement>(null);

  // CN Conversion Policy (AC #9, #10, #11)
  const isCNContent = productionCountry?.includes('CN') ?? false;
  const [convertToTraditional, setConvertToTraditional] = useState(!isCNContent);

  // Reset state when dialog opens for different media (M7 fix)
  useEffect(() => {
    if (open) {
      setQuery(mediaTitle);
      setConvertToTraditional(!isCNContent);
      setPreviewOpen(null);
      setToast(null);
    }
  }, [open, mediaTitle, isCNContent]);

  // Escape key to close
  useEffect(() => {
    if (!open) return;
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onOpenChange(false);
    };
    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [open, onOpenChange]);

  // Auto-hide toast
  useEffect(() => {
    if (!toast) return;
    const timer = setTimeout(() => setToast(null), 3000);
    return () => clearTimeout(timer);
  }, [toast]);

  const {
    search,
    isSearching,
    searchError,
    results,
    resultCount,
    sortBy,
    sortOrder,
    toggleSort,
    download,
    downloadingIds,
    downloadedIds,
    downloadErrorMap,
    preview,
    previewDataMap,
    previewingId,
  } = useSubtitleSearch();

  const handleSearch = useCallback(() => {
    search({
      media_id: mediaId,
      media_type: mediaType,
      providers: selectedProviders,
      query,
    });
  }, [search, mediaId, mediaType, selectedProviders, query]);

  const handleDownload = useCallback(
    (result: SubtitleSearchResult) => {
      download(
        {
          media_id: mediaId,
          media_type: mediaType,
          media_file_path: mediaFilePath,
          subtitle_id: result.id,
          provider: result.source,
          resolution: mediaResolution,
          convert_to_traditional: convertToTraditional,
          score: result.score,
        },
        {
          onSuccess: () => {
            setToast('字幕下載成功');
            onDownloadSuccess?.();
          },
        },
      );
    },
    [download, mediaId, mediaType, mediaFilePath, mediaResolution, convertToTraditional, onDownloadSuccess],
  );

  const toggleProvider = useCallback((providerId: string) => {
    setSelectedProviders((prev) =>
      prev.includes(providerId)
        ? prev.filter((p) => p !== providerId)
        : [...prev, providerId],
    );
  }, []);

  const scoreColor = (score: number) => {
    if (score > 0.7) return 'text-green-400 bg-green-400/10 border-green-400/40';
    if (score > 0.4) return 'text-yellow-400 bg-yellow-400/10 border-yellow-400/40';
    return 'text-red-400 bg-red-400/10 border-red-400/40';
  };

  const SortIcon = ({ field }: { field: SortField }) => {
    if (sortBy !== field) return null;
    return sortOrder === 'desc'
      ? <ChevronDown className="inline h-3 w-3" />
      : <ChevronUp className="inline h-3 w-3" />;
  };

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-black/60 pt-[10vh]"
      onClick={(e) => {
        if (e.target === e.currentTarget) onOpenChange(false);
      }}
      role="dialog"
      aria-modal="true"
      aria-labelledby="subtitle-search-title"
      data-testid="subtitle-search-dialog"
    >
      <div
        ref={dialogRef}
        className="mx-4 w-full max-w-4xl max-h-[80vh] overflow-y-auto rounded-xl bg-slate-800 shadow-2xl"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-700 px-6 py-4">
          <h2 id="subtitle-search-title" className="text-lg font-semibold text-white">
            搜尋字幕
          </h2>
          <button
            onClick={() => onOpenChange(false)}
            className="rounded-lg p-1 text-slate-400 hover:bg-slate-700 hover:text-white"
            aria-label="關閉"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="p-6 space-y-4">
          {/* Search Form */}
          <div className="flex gap-2">
            <input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="輸入搜尋關鍵字..."
              className="flex-1 rounded-lg border border-slate-600 bg-slate-700 px-3 py-2 text-sm text-white placeholder-slate-400 focus:border-blue-500 focus:outline-none"
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              data-testid="subtitle-search-input"
            />
            <button
              onClick={handleSearch}
              disabled={isSearching}
              className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:opacity-50"
              data-testid="subtitle-search-btn"
            >
              {isSearching ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Search className="h-4 w-4" />
              )}
              {isSearching ? '搜尋中...' : '搜尋'}
            </button>
          </div>

          {/* Provider Checkboxes + 繁體轉換 Toggle */}
          <div className="flex items-center justify-between">
            <div className="flex gap-4">
              {PROVIDERS.map((p) => (
                <label key={p.id} className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={selectedProviders.includes(p.id)}
                    onChange={() => toggleProvider(p.id)}
                    className="rounded border-slate-600 bg-slate-700 text-blue-600 focus:ring-blue-500"
                    data-testid={`provider-${p.id}`}
                  />
                  <span className="text-sm text-slate-300">{p.label}</span>
                </label>
              ))}
            </div>

            {/* 繁體轉換 Toggle (AC #9, #10, #11) */}
            <label className="flex items-center gap-2 cursor-pointer" data-testid="convert-toggle">
              <span className="text-sm text-slate-300">繁體轉換</span>
              <button
                role="switch"
                aria-checked={convertToTraditional}
                onClick={() => setConvertToTraditional((v) => !v)}
                className={cn(
                  'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
                  convertToTraditional ? 'bg-blue-600' : 'bg-slate-600',
                )}
              >
                <span
                  className={cn(
                    'inline-block h-4 w-4 rounded-full bg-white transition-transform',
                    convertToTraditional ? 'translate-x-6' : 'translate-x-1',
                  )}
                />
              </button>
            </label>
          </div>

          {/* Search Error (M8) */}
          {searchError && (
            <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-400">
              搜尋失敗：{searchError.message}
            </div>
          )}

          {/* Results Table */}
          {resultCount > 0 && (
            <div>
              <p className="mb-2 text-sm text-slate-400">
                找到 {resultCount} 個結果
              </p>
              <div className="overflow-x-auto rounded-lg border border-slate-700">
                <table className="w-full text-sm" data-testid="subtitle-results-table">
                  <thead>
                    <tr className="border-b border-slate-700 bg-slate-800/50">
                      <th
                        className="cursor-pointer px-3 py-2 text-left text-xs font-medium text-slate-400 hover:text-white"
                        onClick={() => toggleSort('source')}
                      >
                        來源 <SortIcon field="source" />
                      </th>
                      <th
                        className="cursor-pointer px-3 py-2 text-left text-xs font-medium text-slate-400 hover:text-white"
                        onClick={() => toggleSort('language')}
                      >
                        語言 <SortIcon field="language" />
                      </th>
                      <th className="px-3 py-2 text-left text-xs font-medium text-slate-400">
                        字幕名稱
                      </th>
                      <th className="px-3 py-2 text-left text-xs font-medium text-slate-400">
                        格式
                      </th>
                      <th
                        className="cursor-pointer px-3 py-2 text-center text-xs font-medium text-slate-400 hover:text-white"
                        onClick={() => toggleSort('score')}
                      >
                        評分 <SortIcon field="score" />
                      </th>
                      <th
                        className="cursor-pointer px-3 py-2 text-right text-xs font-medium text-slate-400 hover:text-white"
                        onClick={() => toggleSort('downloads')}
                      >
                        下載數 <SortIcon field="downloads" />
                      </th>
                      <th className="px-3 py-2 text-right text-xs font-medium text-slate-400">
                        操作
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {results.map((result) => (
                      <tr
                        key={`${result.source}-${result.id}`}
                        className="border-b border-slate-700/50 hover:bg-slate-700/30"
                        data-testid={`subtitle-row-${result.id}`}
                      >
                        <td className="px-3 py-2 font-medium text-slate-200">
                          {result.source}
                        </td>
                        <td className="px-3 py-2 text-slate-300">{result.language}</td>
                        <td
                          className="max-w-[200px] truncate px-3 py-2 text-slate-300"
                          title={result.filename}
                        >
                          {result.filename}
                        </td>
                        <td className="px-3 py-2 text-xs uppercase text-slate-500">
                          {result.format || '-'}
                        </td>
                        <td className="px-3 py-2 text-center">
                          <span
                            className={cn(
                              'inline-flex items-center justify-center rounded border px-2 py-0.5 text-xs font-medium',
                              scoreColor(result.score),
                            )}
                          >
                            {(result.score * 100).toFixed(0)}%
                          </span>
                        </td>
                        <td className="px-3 py-2 text-right text-slate-400">
                          {result.downloads}
                        </td>
                        <td className="px-3 py-2 text-right">
                          <div className="flex items-center justify-end gap-1">
                            {/* Preview */}
                            <div className="relative">
                              <button
                                onClick={() => {
                                  if (previewOpen === result.id) {
                                    setPreviewOpen(null);
                                  } else {
                                    setPreviewOpen(result.id);
                                    preview({
                                      subtitleId: result.id,
                                      provider: result.source,
                                    });
                                  }
                                }}
                                disabled={previewingId === result.id}
                                className="rounded-md border border-slate-600 px-2 py-1 text-xs text-slate-300 transition-colors hover:bg-slate-700 disabled:opacity-50"
                                data-testid={`preview-btn-${result.id}`}
                              >
                                {previewingId === result.id ? (
                                  <Loader2 className="h-3 w-3 animate-spin" />
                                ) : (
                                  <Eye className="h-3 w-3" />
                                )}
                              </button>
                              {/* Preview Popover */}
                              {previewOpen === result.id && previewDataMap[result.id] && (
                                <div className="absolute right-0 top-8 z-10 w-80 rounded-lg border border-slate-600 bg-slate-800 p-3 shadow-xl">
                                  <p className="mb-2 text-xs font-medium text-slate-300">字幕預覽</p>
                                  {previewDataMap[result.id].lines.map((line, i) => (
                                    <p key={i} className="font-mono text-xs text-slate-400">
                                      {line}
                                    </p>
                                  ))}
                                </div>
                              )}
                            </div>

                            {/* Download — per-row state */}
                            {downloadedIds.has(result.id) ? (
                              <button
                                disabled
                                className="rounded-md border border-green-600/40 bg-green-600/10 px-2 py-1 text-xs text-green-400"
                              >
                                <Check className="h-3 w-3" />
                              </button>
                            ) : (
                              <button
                                onClick={() => handleDownload(result)}
                                disabled={downloadingIds.has(result.id)}
                                className="rounded-md bg-blue-600 px-2 py-1 text-xs text-white transition-colors hover:bg-blue-700 disabled:opacity-50"
                                data-testid={`download-btn-${result.id}`}
                              >
                                {downloadingIds.has(result.id) ? (
                                  <Loader2 className="h-3 w-3 animate-spin" />
                                ) : (
                                  <Download className="h-3 w-3" />
                                )}
                              </button>
                            )}
                          </div>
                          {/* Per-row download error (Task 9.5) */}
                          {downloadErrorMap[result.id] && (
                            <p className="mt-1 text-xs text-red-400">
                              {downloadErrorMap[result.id]}
                            </p>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Empty state */}
          {resultCount === 0 && !isSearching && !searchError && (
            <div className="py-8 text-center text-slate-500" data-testid="subtitle-empty-state">
              點擊「搜尋」開始查找字幕
            </div>
          )}
        </div>
      </div>

      {/* Toast notification (Task 9.6) */}
      {toast && (
        <div className="fixed bottom-6 right-6 z-[60] rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white shadow-lg">
          {toast}
        </div>
      )}
    </div>
  );
}
