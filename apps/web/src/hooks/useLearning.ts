/**
 * Learning hooks using TanStack Query (Story 3.9)
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  learningService,
  type PatternListResponse,
  type PatternStats,
  type LearnPatternParams,
  type LearnedPattern,
} from '../services/learning';

// Query keys following project conventions
export const learningKeys = {
  all: ['learning'] as const,
  patterns: () => [...learningKeys.all, 'patterns'] as const,
  stats: () => [...learningKeys.all, 'stats'] as const,
};

/**
 * Hook for fetching learned patterns with stats (AC3)
 */
export function useLearningPatterns() {
  return useQuery<PatternListResponse, Error>({
    queryKey: learningKeys.patterns(),
    queryFn: () => learningService.listPatterns(),
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

/**
 * Hook for fetching pattern statistics
 */
export function useLearningStats() {
  return useQuery<PatternStats, Error>({
    queryKey: learningKeys.stats(),
    queryFn: () => learningService.getStats(),
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

/**
 * Hook for learning a new pattern from a correction (AC1)
 * Invalidates patterns list on success
 */
export function useLearnPattern() {
  const queryClient = useQueryClient();

  return useMutation<LearnedPattern, Error, LearnPatternParams>({
    mutationFn: (params) => learningService.learnPattern(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: learningKeys.patterns() });
      queryClient.invalidateQueries({ queryKey: learningKeys.stats() });
    },
  });
}

/**
 * Hook for deleting a learned pattern (AC3)
 * Invalidates patterns list on success
 */
export function useDeletePattern() {
  const queryClient = useQueryClient();

  return useMutation<void, Error, string>({
    mutationFn: (id) => learningService.deletePattern(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: learningKeys.patterns() });
      queryClient.invalidateQueries({ queryKey: learningKeys.stats() });
    },
  });
}
