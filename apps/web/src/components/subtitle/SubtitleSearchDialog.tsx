import { useState, useCallback } from 'react';
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
  open,
  onOpenChange,
}: SubtitleSearchDialogProps) {
  const [query, setQuery] = useState(mediaTitle);
  const [selectedProviders, setSelectedProviders] = useState<string[]>(
    PROVIDERS.map((p) => p.id),
  );

  const {
    search,
    isSearching,
    results,
    resultCount,
    sortBy,
    sortOrder,
    toggleSort,
    download,
    isDownloading,
    downloadedIds,
    preview,
    previewData,
    isPreviewing,
  } = useSubtitleSearch();

  const handleSearch = useCallback(() => {
    search({
      mediaId,
      mediaType,
      providers: selectedProviders,
      query,
    });
  }, [search, mediaId, mediaType, selectedProviders, query]);

  const handleDownload = useCallback(
    (result: SubtitleSearchResult) => {
      download({
        mediaId,
        mediaType,
        mediaFilePath,
        subtitleId: result.ID,
        provider: result.Source,
        resolution: mediaResolution,
      });
    },
    [download, mediaId, mediaType, mediaFilePath, mediaResolution],
  );

  const toggleProvider = useCallback((providerId: string) => {
    setSelectedProviders((prev) =>
      prev.includes(providerId)
        ? prev.filter((p) => p !== providerId)
        : [...prev, providerId],
    );
  }, []);

  const scoreColor = (score: number) => {
    if (score > 0.7) return 'text-green-600';
    if (score > 0.4) return 'text-yellow-600';
    return 'text-red-600';
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

          {/* Provider Checkboxes */}
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
        </div>

        {/* Results Table */}
        {resultCount > 0 && (
          <div className="mt-4">
            <p className="text-sm text-muted-foreground mb-2">
              找到 {resultCount} 個結果
            </p>
            <Table>
              <TableHeader>
                <TableRow>
                  <SortableHeader field="Source">來源</SortableHeader>
                  <SortableHeader field="Language">語言</SortableHeader>
                  <SortableHeader field="Group">字幕組</SortableHeader>
                  <TableHead>檔名</TableHead>
                  <SortableHeader field="score">評分</SortableHeader>
                  <SortableHeader field="Downloads">下載數</SortableHeader>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {results.map((result) => (
                  <TableRow key={`${result.Source}-${result.ID}`}>
                    <TableCell className="font-medium">
                      {result.Source}
                    </TableCell>
                    <TableCell>{result.Language}</TableCell>
                    <TableCell>{result.Group || '-'}</TableCell>
                    <TableCell className="max-w-[200px] truncate" title={result.Filename}>
                      {result.Filename}
                    </TableCell>
                    <TableCell>
                      <span className={scoreColor(result.score)}>
                        {(result.score * 100).toFixed(0)}%
                      </span>
                    </TableCell>
                    <TableCell>{result.Downloads}</TableCell>
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
                                  subtitleId: result.ID,
                                  provider: result.Source,
                                })
                              }
                              disabled={isPreviewing}
                            >
                              預覽
                            </Button>
                          </PopoverTrigger>
                          <PopoverContent className="w-96">
                            <div className="space-y-1">
                              <p className="text-sm font-medium">字幕預覽</p>
                              {previewData?.lines?.map((line, i) => (
                                <p key={i} className="text-xs text-muted-foreground font-mono">
                                  {line}
                                </p>
                              ))}
                            </div>
                          </PopoverContent>
                        </Popover>

                        {/* Download */}
                        {downloadedIds.has(result.ID) ? (
                          <Button variant="outline" size="sm" disabled className="text-green-600">
                            ✓
                          </Button>
                        ) : (
                          <Button
                            variant="default"
                            size="sm"
                            onClick={() => handleDownload(result)}
                            disabled={isDownloading}
                          >
                            {isDownloading ? '...' : '下載'}
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
        {resultCount === 0 && !isSearching && (
          <div className="text-center py-8 text-muted-foreground">
            點擊「搜尋」開始查找字幕
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
