import { useState, useCallback } from 'react';
import { useMutation } from '@tanstack/react-query';
import {
  subtitleService,
  type SubtitleSearchParams,
  type SubtitleSearchResult,
  type SubtitleDownloadParams,
  type SubtitlePreviewResult,
} from '../services/subtitleService';

export type SortField = 'score' | 'Language' | 'Source' | 'Downloads' | 'Group';
export type SortOrder = 'asc' | 'desc';

export function useSubtitleSearch() {
  const [results, setResults] = useState<SubtitleSearchResult[]>([]);
  const [sortBy, setSortBy] = useState<SortField>('score');
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');
  const [downloadedIds, setDownloadedIds] = useState<Set<string>>(new Set());

  // Search mutation
  const searchMutation = useMutation({
    mutationFn: (params: SubtitleSearchParams) =>
      subtitleService.searchSubtitles(params),
    onSuccess: (data) => {
      setResults(data || []);
      setDownloadedIds(new Set());
    },
  });

  // Download mutation
  const downloadMutation = useMutation({
    mutationFn: (params: SubtitleDownloadParams) =>
      subtitleService.downloadSubtitle(params),
    onSuccess: (_data, variables) => {
      setDownloadedIds((prev) => new Set(prev).add(variables.subtitleId));
    },
  });

  // Preview mutation
  const previewMutation = useMutation({
    mutationFn: (params: { subtitleId: string; provider: string }) =>
      subtitleService.previewSubtitle(params),
  });

  // Sort results
  const sortedResults = [...results].sort((a, b) => {
    const multiplier = sortOrder === 'desc' ? -1 : 1;
    switch (sortBy) {
      case 'score':
        return (a.score - b.score) * multiplier;
      case 'Downloads':
        return (a.Downloads - b.Downloads) * multiplier;
      case 'Language':
        return a.Language.localeCompare(b.Language) * multiplier;
      case 'Source':
        return a.Source.localeCompare(b.Source) * multiplier;
      case 'Group':
        return (a.Group || '').localeCompare(b.Group || '') * multiplier;
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

    // Download
    download: downloadMutation.mutate,
    isDownloading: downloadMutation.isPending,
    downloadError: downloadMutation.error,
    downloadedIds,

    // Preview
    preview: previewMutation.mutateAsync,
    previewData: previewMutation.data as SubtitlePreviewResult | undefined,
    isPreviewing: previewMutation.isPending,
  };
}
