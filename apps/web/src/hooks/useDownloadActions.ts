/**
 * Download action mutations (ux3-4-3b AC3/AC5) — pause / resume / remove, single OR batch.
 *
 * `confirmed against [@contract-v1]` (ux3-4-2): `POST /downloads/:hash/pause|resume`,
 * `DELETE /downloads/:hash?deleteFiles=`. The BE HTTP API is SINGLE-HASH ONLY (ux3-4-2's
 * slice-accepting methods live at the qBittorrent Go-client layer, not the HTTP surface), so a
 * batch op is N parallel single-hash requests via `Promise.allSettled` — correct for a single-user
 * NAS with a handful of downloads. If any request fails the mutation throws (→ optimistic rollback);
 * the onSettled invalidate re-fetches ground truth either way, so the UI self-corrects.
 *
 * Each mutation optimistically patches every cached downloads list page (pause→paused,
 * resume→downloading, remove→dropped) and rolls back on error.
 */
import { useMutation, useQueryClient, type QueryClient } from '@tanstack/react-query';
import {
  downloadService,
  type PaginatedDownloads,
  type TorrentStatus,
} from '../services/downloadService';
import { downloadKeys } from './useDownloads';

type ActionKind = 'pause' | 'resume' | 'remove';

export interface RemoveVars {
  hashes: string[];
  deleteFiles: boolean;
}

type CacheSnapshot = [readonly unknown[], PaginatedDownloads | undefined];
interface ActionContext {
  snapshots: CacheSnapshot[];
}

// Optimistically patch every cached downloads list page; returns the prior snapshots for rollback.
function patchListCache(
  queryClient: QueryClient,
  hashes: Set<string>,
  kind: ActionKind
): CacheSnapshot[] {
  const queries = queryClient.getQueryCache().findAll({ queryKey: [...downloadKeys.all, 'list'] });
  const snapshots: CacheSnapshot[] = [];

  for (const q of queries) {
    const key = q.queryKey;
    const old = q.state.data as PaginatedDownloads | undefined;
    snapshots.push([key, old]);
    if (!old) continue;

    let items = old.items;
    if (kind === 'remove') {
      items = old.items.filter((it) => !hashes.has(it.hash));
    } else {
      const nextStatus: TorrentStatus = kind === 'pause' ? 'paused' : 'downloading';
      items = old.items.map((it) => (hashes.has(it.hash) ? { ...it, status: nextStatus } : it));
    }

    const removed = old.items.length - items.length;
    queryClient.setQueryData<PaginatedDownloads>(key, {
      ...old,
      items,
      totalItems: Math.max(0, old.totalItems - removed),
    });
  }

  return snapshots;
}

// Run a single-hash action across all hashes; throw if any rejects (batch = N requests, no batch API).
function runForHashes(fn: (hash: string) => Promise<void>) {
  return async (hashes: string[]): Promise<void> => {
    const results = await Promise.allSettled(hashes.map((h) => fn(h)));
    const failed = results.filter((r): r is PromiseRejectedResult => r.status === 'rejected');
    if (failed.length > 0) {
      const reason = failed[0].reason;
      const msg = reason instanceof Error ? reason.message : `${failed.length} 個操作失敗`;
      throw new Error(failed.length > 1 ? `${failed.length} 個操作失敗：${msg}` : msg);
    }
  };
}

export function useDownloadActions() {
  const queryClient = useQueryClient();

  const optimistic = (kind: ActionKind) => ({
    onMutate: async (vars: string[] | RemoveVars): Promise<ActionContext> => {
      const hashes = Array.isArray(vars) ? vars : vars.hashes;
      await queryClient.cancelQueries({ queryKey: downloadKeys.all });
      return { snapshots: patchListCache(queryClient, new Set(hashes), kind) };
    },
    onError: (_err: Error, _vars: string[] | RemoveVars, ctx?: ActionContext) => {
      ctx?.snapshots.forEach(([key, data]) => queryClient.setQueryData(key, data));
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: downloadKeys.all });
    },
  });

  const pause = useMutation<void, Error, string[], ActionContext>({
    mutationFn: runForHashes((h) => downloadService.pauseDownload(h)),
    ...optimistic('pause'),
  });

  const resume = useMutation<void, Error, string[], ActionContext>({
    mutationFn: runForHashes((h) => downloadService.resumeDownload(h)),
    ...optimistic('resume'),
  });

  const remove = useMutation<void, Error, RemoveVars, ActionContext>({
    mutationFn: ({ hashes, deleteFiles }) =>
      runForHashes((h) => downloadService.removeDownload(h, deleteFiles))(hashes),
    ...optimistic('remove'),
  });

  return { pause, resume, remove };
}
