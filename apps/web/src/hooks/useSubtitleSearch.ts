import { useState, useCallback, useEffect, useRef } from 'react';
import { useMutation } from '@tanstack/react-query';
import {
  subtitleService,
  type SubtitleSearchParams,
  type SubtitleSearchResult,
  type SubtitleDownloadParams,
  type SubtitlePreviewResult,
} from '../services/subtitleService';
import { snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export type SortField = 'score' | 'language' | 'source' | 'downloads' | 'group';
export type SortOrder = 'asc' | 'desc';

// SSE subtitle_progress event data shape (camelCase after snakeToCamel transform)
interface SubtitleProgressEvent {
  mediaId: string;
  mediaType: string;
  stage: string;
  message: string;
}

export function useSubtitleSearch() {
  const [results, setResults] = useState<SubtitleSearchResult[]>([]);
  const [sortBy, setSortBy] = useState<SortField>('score');
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');
  const [downloadedIds, setDownloadedIds] = useState<Set<string>>(new Set());
  // Per-row download tracking (M2 fix)
  const [downloadingIds, setDownloadingIds] = useState<Set<string>>(new Set());
  // Per-row preview tracking (M3 fix)
  const [previewDataMap, setPreviewDataMap] = useState<Record<string, SubtitlePreviewResult>>({});
  const [previewingId, setPreviewingId] = useState<string | null>(null);
  // Per-row download error tracking (M1 fix — CR pass 2)
  const [downloadErrorMap, setDownloadErrorMap] = useState<Record<string, string>>({});
  // SSE subtitle progress (Task 7.5)
  const [downloadStage, setDownloadStage] = useState<string | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  // Connect to SSE when any download is in progress
  useEffect(() => {
    if (downloadingIds.size === 0) {
      // No active downloads — disconnect SSE
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      setDownloadStage(null);
      return;
    }

    // Already connected
    if (eventSourceRef.current) return;

    const es = new EventSource(`${API_BASE_URL}/events`);
    eventSourceRef.current = es;

    es.addEventListener('subtitle_progress', (e: MessageEvent) => {
      try {
        const parsed = JSON.parse(e.data);
        const data = snakeToCamel<SubtitleProgressEvent>(parsed.data || parsed);
        setDownloadStage(data.stage);
      } catch {
        // Ignore parse errors
      }
    });

    es.onerror = () => {
      es.close();
      eventSourceRef.current = null;
    };

    return () => {
      es.close();
      eventSourceRef.current = null;
    };
  }, [downloadingIds.size]);

  // Search mutation
  const searchMutation = useMutation({
    mutationFn: (params: SubtitleSearchParams) => subtitleService.searchSubtitles(params),
    onSuccess: (data) => {
      setResults(data || []);
      setDownloadedIds(new Set());
      setDownloadingIds(new Set());
      setPreviewDataMap({});
      setDownloadErrorMap({});
    },
  });

  // Download mutation (per-row tracking)
  const downloadMutation = useMutation({
    mutationFn: (params: SubtitleDownloadParams) => subtitleService.downloadSubtitle(params),
    onMutate: (variables) => {
      setDownloadingIds((prev) => new Set(prev).add(variables.subtitleId));
    },
    onSuccess: (_data, variables) => {
      setDownloadedIds((prev) => new Set(prev).add(variables.subtitleId));
      setDownloadingIds((prev) => {
        const next = new Set(prev);
        next.delete(variables.subtitleId);
        return next;
      });
      // Clear any previous error for this row
      setDownloadErrorMap((prev) => {
        const next = { ...prev };
        delete next[variables.subtitleId];
        return next;
      });
    },
    onError: (error, variables) => {
      setDownloadingIds((prev) => {
        const next = new Set(prev);
        next.delete(variables.subtitleId);
        return next;
      });
      // Track error per-row
      setDownloadErrorMap((prev) => ({
        ...prev,
        [variables.subtitleId]: error instanceof Error ? error.message : '下載失敗',
      }));
    },
  });

  // Preview mutation (per-row tracking)
  const previewMutation = useMutation({
    mutationFn: (params: { subtitleId: string; provider: string }) =>
      subtitleService.previewSubtitle({
        subtitleId: params.subtitleId,
        provider: params.provider,
      }),
    onMutate: (variables) => {
      setPreviewingId(variables.subtitleId);
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
    [sortBy]
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
    downloadErrorMap,
    downloadedIds,

    // Preview (per-row)
    preview: previewMutation.mutateAsync,
    previewDataMap,
    previewingId,
    isPreviewing: previewMutation.isPending,

    // SSE progress (Task 7.5)
    downloadStage,
  };
}
