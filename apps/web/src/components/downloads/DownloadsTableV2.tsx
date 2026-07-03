// Design ref: ux-design.pen Screen D7-D-v2 (w3ipb)
import { ArrowUp, ArrowDown } from 'lucide-react';
import type { Download, SortField, SortOrder } from '../../services/downloadService';
import { cn } from '../../lib/utils';
import { getDownloadStatus } from './downloadStatus';
import { DownloadRowActions } from './DownloadRowActions';
import { formatSpeed, formatSize, formatETA, formatProgress } from './formatters';

interface DownloadsTableV2Props {
  items: Download[];
  sortField: SortField;
  sortOrder: SortOrder;
  onSort: (field: SortField) => void;
  selected: Set<string>;
  onSelectChange: (hash: string, selected: boolean) => void;
  onSelectAll: () => void;
  onClearAll: () => void;
  onPause?: (hash: string) => void;
  onResume?: (hash: string) => void;
  onRemove?: (hash: string, deleteFiles: boolean) => void;
}

interface Col {
  key: string;
  label: string;
  sort?: SortField; // only name/status/progress are API-sortable (SortField); size/speed/eta are not
  thClass?: string;
}

const COLUMNS: Col[] = [
  { key: 'name', label: '名稱', sort: 'name' },
  { key: 'status', label: '狀態', sort: 'status', thClass: 'w-24' },
  { key: 'size', label: '大小', thClass: 'w-28 text-right' },
  { key: 'progress', label: '進度', sort: 'progress', thClass: 'w-44' },
  { key: 'speed', label: '速度', thClass: 'w-28 text-right' },
  { key: 'eta', label: 'ETA', thClass: 'w-24 text-right' },
  { key: 'actions', label: '操作', thClass: 'w-20 text-right' },
];

function ariaSort(active: boolean, order: SortOrder): 'ascending' | 'descending' | 'none' {
  if (!active) return 'none';
  return order === 'asc' ? 'ascending' : 'descending';
}

function progressFill(d: Download): string {
  if (d.status === 'error') return 'bg-[var(--error)]';
  if (d.progress >= 1 || d.status === 'completed' || d.status === 'seeding')
    return 'bg-[var(--success)]';
  return 'bg-[var(--accent-primary)]';
}

function speedCell(d: Download): string {
  if (d.status === 'downloading') return `↓ ${formatSpeed(d.downloadSpeed)}`;
  if (d.status === 'seeding') return `↑ ${formatSpeed(d.uploadSpeed)}`;
  return '—';
}

/**
 * DownloadsTableV2 — the dense sortable desktop Table view (ux3-4-4 / D7-D-v2). An alternate rendering
 * of the SAME page data the card List uses: reuses downloadStatus + formatters + DownloadRowActions,
 * and drives the SAME sortField/sortOrder + selection state as the List (so the toolbar sort control and
 * the column headers are two controls over one state). Semantic <table> in an overflow-x container;
 * sortable headers carry aria-sort; the checkbox column is persistent (no select-mode toggle in Table).
 */
export function DownloadsTableV2({
  items,
  sortField,
  sortOrder,
  onSort,
  selected,
  onSelectChange,
  onSelectAll,
  onClearAll,
  onPause,
  onResume,
  onRemove,
}: DownloadsTableV2Props) {
  const allSelected = items.length > 0 && items.every((d) => selected.has(d.hash));
  const someSelected = items.some((d) => selected.has(d.hash));

  return (
    <div
      data-testid="downloads-table-v2"
      className="overflow-x-auto rounded-[var(--radius-lg)] border border-[var(--border-subtle)]"
    >
      <table className="w-full border-collapse text-sm">
        <thead>
          <tr className="border-b border-[var(--border-subtle)] bg-[var(--bg-secondary)] text-left text-[var(--text-secondary)]">
            <th scope="col" className="w-10 px-3 py-2">
              <input
                type="checkbox"
                aria-label={allSelected ? '取消全選' : '全選'}
                checked={allSelected}
                ref={(el) => {
                  if (el) el.indeterminate = !allSelected && someSelected;
                }}
                onChange={() => (allSelected ? onClearAll() : onSelectAll())}
                className="h-4 w-4 accent-[var(--accent-primary)]"
              />
            </th>
            {COLUMNS.map((col) => {
              const active = col.sort === sortField;
              return (
                <th
                  key={col.key}
                  scope="col"
                  aria-sort={col.sort ? ariaSort(active, sortOrder) : undefined}
                  className={cn('px-3 py-2 font-medium', col.thClass)}
                >
                  {col.sort ? (
                    <button
                      type="button"
                      onClick={() => onSort(col.sort as SortField)}
                      data-testid={`downloads-sort-${col.sort}`}
                      className="inline-flex items-center gap-1 transition-colors hover:text-[var(--text-primary)]"
                    >
                      {col.label}
                      {active &&
                        (sortOrder === 'asc' ? <ArrowUp size={14} /> : <ArrowDown size={14} />)}
                    </button>
                  ) : (
                    col.label
                  )}
                </th>
              );
            })}
          </tr>
        </thead>
        <tbody>
          {items.map((d) => {
            const status = getDownloadStatus(d.status);
            const pct = Math.round(d.progress * 100);
            const isSel = selected.has(d.hash);
            return (
              <tr
                key={d.hash}
                data-testid={`downloads-table-row-${d.hash}`}
                className={cn(
                  'border-b border-[var(--border-subtle)] transition-colors hover:bg-[var(--bg-secondary)]/50',
                  isSel && 'bg-[var(--accent-tint)]'
                )}
              >
                <td className="px-3 py-2">
                  <input
                    type="checkbox"
                    checked={isSel}
                    onChange={(e) => onSelectChange(d.hash, e.target.checked)}
                    aria-label={`選取 ${d.name}`}
                    className="h-4 w-4 accent-[var(--accent-primary)]"
                  />
                </td>
                <td className="px-3 py-2">
                  <span className="line-clamp-2 font-medium text-[var(--text-primary)]">
                    {d.name}
                  </span>
                </td>
                <td className="px-3 py-2">
                  <span
                    data-testid={`download-status-${d.hash}`}
                    className={cn(
                      'inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium',
                      status.className
                    )}
                  >
                    {status.label}
                  </span>
                </td>
                <td className="px-3 py-2 text-right font-mono tabular-nums text-[var(--text-secondary)]">
                  {formatSize(d.size)}
                </td>
                <td className="px-3 py-2">
                  <div className="flex items-center gap-2">
                    <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-[var(--bg-tertiary)]">
                      <div
                        className={cn('h-full rounded-full', progressFill(d))}
                        style={{ width: `${Math.min(pct, 100)}%` }}
                        role="progressbar"
                        aria-label={`${d.name} 下載進度`}
                        aria-valuenow={pct}
                        aria-valuemin={0}
                        aria-valuemax={100}
                      />
                    </div>
                    <span className="shrink-0 font-mono text-xs tabular-nums text-[var(--text-secondary)]">
                      {formatProgress(d.progress)}
                    </span>
                  </div>
                </td>
                <td className="px-3 py-2 text-right font-mono tabular-nums text-[var(--text-secondary)]">
                  {speedCell(d)}
                </td>
                <td className="px-3 py-2 text-right font-mono tabular-nums text-[var(--text-secondary)]">
                  {d.status === 'downloading' ? formatETA(d.eta) : '—'}
                </td>
                <td className="px-3 py-2">
                  <div className="flex justify-end">
                    <DownloadRowActions
                      download={d}
                      onPause={onPause}
                      onResume={onResume}
                      onRemove={onRemove}
                    />
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
