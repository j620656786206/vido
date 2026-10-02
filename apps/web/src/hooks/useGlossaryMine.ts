/**
 * 「從官方字幕學譯名」 hooks (sub-7-5b). The status is server state; while a
 * run is in flight it is polled every 3 s (C9-D note ②), and the start
 * mutation invalidates it so the button flips to 學習中… at once — the
 * backend claims the miner before answering 202, so that first read already
 * says running.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { glossaryMineService, type MineStatus } from '../services/glossaryMineService';

export const glossaryMineQueryKeys = {
  status: ['settings', 'glossary-mine'] as const,
};

export const GLOSSARY_MINE_POLL_MS = 3000;

export function useGlossaryMineStatus() {
  return useQuery<MineStatus, Error>({
    queryKey: glossaryMineQueryKeys.status,
    queryFn: () => glossaryMineService.status(),
    refetchInterval: (query) => (query.state.data?.running ? GLOSSARY_MINE_POLL_MS : false),
  });
}

export function useStartGlossaryMine() {
  const queryClient = useQueryClient();
  return useMutation<{ started: boolean }, Error, void>({
    mutationFn: () => glossaryMineService.startSweep(),
    onSettled: () => {
      // 202 or 409 (already running): either way the truth is on the server.
      void queryClient.invalidateQueries({ queryKey: glossaryMineQueryKeys.status });
    },
  });
}
