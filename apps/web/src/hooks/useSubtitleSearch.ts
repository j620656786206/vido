import { useState, useCallback } from 'react';
import { useMutation } from '@tanstack/react-query';
import {
  subtitleService,
  type SubtitleSearchParams,
  type SubtitleSearchResult,
  type SubtitleDownloadParams,
  type SubtitlePreviewResult,
} from '../services/subtitleService';

export type SortField = 'score' | 'language' | 'source' | 'downloads' | 'group';
export type SortOrder = 'asc' | 'desc';

export function useSubtitleSearch() {
  const [results, setResults] = useState<SubtitleSearchResult[]>([]);
  const [sortBy, setSortBy] = useState<SortField>('score');
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');
  const [downloadedIds, setDownloadedIds] = useState<Set<string>>(new Set());
  // Per-row download tracking (M2 fix)
  const [downloadingIds, setDownloadingIds] = useState<Set<string>>(new Set());
  // Per-row preview tracking (M3 fix)
  const [previewDataMap, setPreviewDataMap] = useState<
    Record<string, SubtitlePreviewResult>
  >({});
  const [previewingId, setPreviewingId] = useState<string | null>(null);

  // Search mutation
  const searchMutation = useMutation({
    mutationFn: (params: SubtitleSearchParams) =>
      subtitleService.searchSubtitles(params),
    onSuccess: (data) => {
      setResults(data || []);
      setDownloadedIds(new Set());
      setDownloadingIds(new Set());
      setPreviewDataMap({});
    },
  });

  // Download mutation (per-row tracking)
  const downloadMutation = useMutation({
    mutationFn: (params: SubtitleDownloadParams) => {
      setDownloadingIds((prev) => new Set(prev).add(params.subtitle_id));
      return subtitleService.downloadSubtitle(params);
    },
    onSuccess: (_data, variables) => {
      setDownloadedIds((prev) => new Set(prev).add(variables.subtitle_id));
      setDownloadingIds((prev) => {
        const next = new Set(prev);
        next.delete(variables.subtitle_id);
        return next;
      });
    },
    onError: (_error, variables) => {
      setDownloadingIds((prev) => {
        const next = new Set(prev);
        next.delete(variables.subtitle_id);
        return next;
      });
    },
  });

  // Preview mutation (per-row tracking)
  const previewMutation = useMutation({
    mutationFn: (params: { subtitleId: string; provider: string }) => {
      setPreviewingId(params.subtitleId);
      return subtitleService.previewSubtitle({
        subtitle_id: params.subtitleId,
        provider: params.provider,
      });
    },
    onSuccess: (data, variables) => {
      setPreviewDataMap((prev) => ({
        ...prev,
        [variables.subtitleId]: data,
      }));
      setPreviewingId(null);
    },
    onError: () => {
      setPreviewingId(null);
    },
  });

  // Sort results
  const sortedResults = [...results].sort((a, b) => {
    const multiplier = sortOrder === 'desc' ? -1 : 1;
    switch (sortBy) {
      case 'score':
        return (a.score - b.score) * multiplier;
      case 'downloads':
        return (a.downloads - b.downloads) * multiplier;
      case 'language':
        return a.language.localeCompare(b.language) * multiplier;
      case 'source':
        return a.source.localeCompare(b.source) * multiplier;
      case 'group':
        return (a.group || '').localeCompare(b.group || '') * multiplier;
      default:
        return 0;
    }
  });

  // Toggle sort
  const toggleSort = useCallback(
    (field: SortField) => {
      if (sortBy === field) {
        setSortOrder((prev) => (prev === 'desc' ? 'asc' : 'desc'));
      } else {
        setSortBy(field);
        setSortOrder('desc');
      }
    },
    [sortBy],
  );

  return {
    // Search
    search: searchMutation.mutate,
    isSearching: searchMutation.isPending,
    searchError: searchMutation.error,

    // Results
    results: sortedResults,
    resultCount: results.length,

    // Sort
    sortBy,
    sortOrder,
    toggleSort,

    // Download (per-row)
    download: downloadMutation.mutate,
    downloadingIds,
    downloadError: downloadMutation.error,
    downloadedIds,

    // Preview (per-row)
    preview: previewMutation.mutateAsync,
    previewDataMap,
    previewingId,
    isPreviewing: previewMutation.isPending,
  };
}
