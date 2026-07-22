import { createFileRoute } from '@tanstack/react-router';
import { DownloadsBrowseV2 } from '../components/downloads/DownloadsBrowseV2';
import type { FilterStatus } from '../services/downloadService';

interface DownloadsSearch {
  filter?: FilterStatus;
  page?: number;
  pageSize?: number;
}

const PAGE_SIZE_OPTIONS = [50, 100, 200, 500] as const;

export const Route = createFileRoute('/downloads')({
  // ux3-cutover-3: legacy branch removed — DownloadsBrowseV2 is the only render
  // (the 下載 bottom-4 tab is wired in navModel.ts).
  validateSearch: (search: Record<string, unknown>): DownloadsSearch => {
    const validFilters = ['all', 'downloading', 'paused', 'completed', 'seeding', 'error'];
    const filter = validFilters.includes(search.filter as string)
      ? (search.filter as FilterStatus)
      : undefined;
    const page = Number(search.page) > 0 ? Number(search.page) : undefined;
    const pageSize = PAGE_SIZE_OPTIONS.includes(
      Number(search.pageSize) as (typeof PAGE_SIZE_OPTIONS)[number]
    )
      ? Number(search.pageSize)
      : undefined;
    return { filter, page, pageSize };
  },
  component: DownloadsBrowseV2,
});
