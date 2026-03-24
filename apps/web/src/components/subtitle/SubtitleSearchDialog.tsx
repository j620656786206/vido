import { useState, useCallback, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog';
import { Button } from '../ui/button';
import { Input } from '../ui/input';
import { Checkbox } from '../ui/checkbox';
import { Label } from '../ui/label';
import { Switch } from '../ui/switch';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '../ui/table';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '../ui/popover';
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
}: SubtitleSearchDialogProps) {
  const [query, setQuery] = useState(mediaTitle);
  const [selectedProviders, setSelectedProviders] = useState<string[]>(
    PROVIDERS.map((p) => p.id),
  );

  // CN Conversion Policy (AC #9, #10, #11)
  // Default: OFF for CN content, ON for non-CN content
  const isCNContent = productionCountry?.includes('CN') ?? false;
  const [convertToTraditional, setConvertToTraditional] = useState(!isCNContent);

  // Reset query and toggle when dialog opens for different media (M7 fix)
  useEffect(() => {
    if (open) {
      setQuery(mediaTitle);
      setConvertToTraditional(!isCNContent);
    }
  }, [open, mediaTitle, isCNContent]);

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
      download({
        media_id: mediaId,
        media_type: mediaType,
        media_file_path: mediaFilePath,
        subtitle_id: result.id,
        provider: result.source,
        resolution: mediaResolution,
        convert_to_traditional: convertToTraditional,
      });
    },
    [download, mediaId, mediaType, mediaFilePath, mediaResolution, convertToTraditional],
  );

  const toggleProvider = useCallback((providerId: string) => {
    setSelectedProviders((prev) =>
      prev.includes(providerId)
        ? prev.filter((p) => p !== providerId)
        : [...prev, providerId],
    );
  }, []);

  const scoreColor = (score: number) => {
    if (score > 0.7) return 'text-green-600 bg-green-600/10 border border-green-600/40';
    if (score > 0.4) return 'text-yellow-600 bg-yellow-600/10 border border-yellow-600/40';
    return 'text-red-600 bg-red-600/10 border border-red-600/40';
  };

  const SortableHeader = ({
    field,
    children,
  }: {
    field: SortField;
    children: React.ReactNode;
  }) => (
    <TableHead
      className="cursor-pointer select-none hover:bg-muted/50"
      onClick={() => toggleSort(field)}
    >
      <div className="flex items-center gap-1">
        {children}
        {sortBy === field && (
          <span className="text-xs">{sortOrder === 'desc' ? '▼' : '▲'}</span>
        )}
      </div>
    </TableHead>
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>搜尋字幕</DialogTitle>
        </DialogHeader>

        {/* Search Form */}
        <div className="space-y-4">
          <div className="flex gap-2">
            <Input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="輸入搜尋關鍵字..."
              className="flex-1"
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
            />
            <Button onClick={handleSearch} disabled={isSearching}>
              {isSearching ? '搜尋中...' : '搜尋'}
            </Button>
          </div>

          {/* Provider Checkboxes + 繁體轉換 Toggle */}
          <div className="flex items-center justify-between">
            <div className="flex gap-4">
              {PROVIDERS.map((p) => (
                <div key={p.id} className="flex items-center gap-2">
                  <Checkbox
                    id={`provider-${p.id}`}
                    checked={selectedProviders.includes(p.id)}
                    onCheckedChange={() => toggleProvider(p.id)}
                  />
                  <Label htmlFor={`provider-${p.id}`} className="text-sm">
                    {p.label}
                  </Label>
                </div>
              ))}
            </div>

            {/* 繁體轉換 Toggle (AC #9, #10, #11) */}
            <div className="flex items-center gap-2">
              <Label htmlFor="convert-toggle" className="text-sm">
                繁體轉換
              </Label>
              <Switch
                id="convert-toggle"
                checked={convertToTraditional}
                onCheckedChange={setConvertToTraditional}
              />
            </div>
          </div>
        </div>

        {/* Search Error (M8 fix) */}
        {searchError && (
          <div className="mt-2 p-3 text-sm text-red-600 bg-red-50 rounded-md border border-red-200">
            搜尋失敗：{searchError.message}
          </div>
        )}

        {/* Results Table — columns per UX design Flow I */}
        {resultCount > 0 && (
          <div className="mt-4">
            <p className="text-sm text-muted-foreground mb-2">
              找到 {resultCount} 個結果
            </p>
            <Table>
              <TableHeader>
                <TableRow>
                  <SortableHeader field="source">來源</SortableHeader>
                  <SortableHeader field="language">語言</SortableHeader>
                  <TableHead>字幕名稱</TableHead>
                  <TableHead>格式</TableHead>
                  <SortableHeader field="score">評分</SortableHeader>
                  <SortableHeader field="downloads">下載數</SortableHeader>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {results.map((result) => (
                  <TableRow key={`${result.source}-${result.id}`}>
                    <TableCell className="font-medium">
                      {result.source}
                    </TableCell>
                    <TableCell>{result.language}</TableCell>
                    <TableCell className="max-w-[200px] truncate" title={result.filename}>
                      {result.filename}
                    </TableCell>
                    <TableCell className="uppercase text-xs text-muted-foreground">
                      {result.format || '-'}
                    </TableCell>
                    <TableCell>
                      <span
                        className={`inline-flex items-center justify-center px-2 py-0.5 rounded text-xs font-medium ${scoreColor(result.score)}`}
                      >
                        {(result.score * 100).toFixed(0)}%
                      </span>
                    </TableCell>
                    <TableCell>{result.downloads}</TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        {/* Preview */}
                        <Popover>
                          <PopoverTrigger asChild>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() =>
                                preview({
                                  subtitleId: result.id,
                                  provider: result.source,
                                })
                              }
                              disabled={previewingId === result.id}
                            >
                              {previewingId === result.id ? '...' : '預覽'}
                            </Button>
                          </PopoverTrigger>
                          <PopoverContent className="w-96">
                            <div className="space-y-1">
                              <p className="text-sm font-medium">字幕預覽</p>
                              {previewDataMap[result.id]?.lines?.map(
                                (line, i) => (
                                  <p
                                    key={i}
                                    className="text-xs text-muted-foreground font-mono"
                                  >
                                    {line}
                                  </p>
                                ),
                              )}
                            </div>
                          </PopoverContent>
                        </Popover>

                        {/* Download — per-row state (M2 fix) */}
                        {downloadedIds.has(result.id) ? (
                          <Button
                            variant="outline"
                            size="sm"
                            disabled
                            className="text-green-600"
                          >
                            ✓
                          </Button>
                        ) : (
                          <Button
                            variant="default"
                            size="sm"
                            onClick={() => handleDownload(result)}
                            disabled={downloadingIds.has(result.id)}
                          >
                            {downloadingIds.has(result.id) ? '...' : '下載'}
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}

        {/* Empty state */}
        {resultCount === 0 && !isSearching && !searchError && (
          <div className="text-center py-8 text-muted-foreground">
            點擊「搜尋」開始查找字幕
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
