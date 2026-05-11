// AC #1 [@contract-v1] (bugfix-10-5): empty-library 3-state classifier.
// Priority order is load-bearing: A (no-qbt) > B (no-folder) > C (ready-for-scan).
// Any pending query → 'loading' so callers can keep showing the skeleton instead of an empty-state.

export type EmptyLibraryState = 'loading' | 'no-qbt' | 'no-folder' | 'ready-for-scan';

export interface ClassifyEmptyStateInput {
  qbtConfigured: boolean | undefined;
  mediaLibrariesCount: number;
  itemsCount: number;
  isLoading: boolean;
}

export function classifyEmptyState({
  qbtConfigured,
  mediaLibrariesCount,
  itemsCount: _itemsCount,
  isLoading,
}: ClassifyEmptyStateInput): EmptyLibraryState {
  if (isLoading || qbtConfigured === undefined) return 'loading';
  if (qbtConfigured === false) return 'no-qbt';
  if (mediaLibrariesCount === 0) return 'no-folder';
  return 'ready-for-scan';
}
