// Story 11-4 — saved filter presets server state (Rule 5: TanStack Query, not Zustand).
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  filterPresetService,
  type CreateFilterPresetRequest,
  type FilterPreset,
} from '../services/filterPresetService';

export const filterPresetKeys = {
  all: ['filter-presets'] as const,
};

/** List all saved filter presets. */
export function useFilterPresets() {
  return useQuery<FilterPreset[], Error>({
    queryKey: filterPresetKeys.all,
    queryFn: async () => (await filterPresetService.getAll()).presets,
    staleTime: 5 * 60 * 1000,
  });
}

/** Create a preset; invalidates the list on success. */
export function useCreateFilterPreset() {
  const queryClient = useQueryClient();
  return useMutation<FilterPreset, Error, CreateFilterPresetRequest>({
    mutationFn: (req) => filterPresetService.create(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: filterPresetKeys.all });
    },
  });
}

/** Delete a preset by id; invalidates the list on success. */
export function useDeleteFilterPreset() {
  const queryClient = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: (id) => filterPresetService.remove(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: filterPresetKeys.all });
    },
  });
}
