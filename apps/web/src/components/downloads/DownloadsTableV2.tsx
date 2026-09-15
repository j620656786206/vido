// Design ref: ux-design.pen Screen D7-D-v2 (w3ipb) · D12-D-v2 (tvp15)
import { ArrowUp, ArrowDown, ChevronsUpDown } from 'lucide-react';
import type { Download, SortField, SortOrder } from '../../services/downloadService';
import { cn } from '../../lib/utils';
import { getDownloadTone } from './downloadStatus';
import { DownloadStatusPill } from './DownloadCardV2';
import { DownloadRowActions } from './DownloadRowActions';
import { ImportStatusChip } from './ImportStatusChip';
import { formatDownloadMeta, formatProgress } from './formatters';

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
  sort?: SortField; // only name/status/progress are API-sortable (SortField); speed/eta/size are not
  thClass?: string;
}

// D7-D-v2 column order. Only the API-sortable columns carry a sort control — the design's sort
// glyphs on 速度 / ETA / 大小 promised a sort the backend cannot do and were removed (dsr-4).
const COLUMNS: Col[] = [
  { key: 'name', label: '名稱', sort: 'name' },
  { key: 'status', label: '狀態', sort: 'status', thClass: 'w-28' },
  { key: 'progress', label: '進度', sort: 'progress', thClass: 'w-44' },
  { key: 'speed', label: '速度', thClass: 'w-32' },
  { key: 'eta', label: 'ETA', thClass: 'w-20' },
  { key: 'size', label: '大小', thClass: 'w-32' },
  { key: 'actions', label: '動作', thClass: 'w-28' },
];

function ariaSort(active: boolean, order: SortOrder): 'ascending' | 'descending' | 'none' {
  if (!active) return 'none';
  return order === 'asc' ? 'ascending' : 'descending';
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
      className="overflow-x-auto rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)]"
    >
      <table className="w-full min-w-[960px] table-fixed border-collapse text-sm">
        <thead>
          <tr className="border-b border-[var(--border-subtle)] bg-[var(--bg-tertiary)] text-left text-xs text-[var(--text-secondary)]">
            <th scope="col" className="w-11 px-3 py-3">
              <input
                type="checkbox"
                aria-label={allSelected ? '取消全選' : '全選'}
                checked={allSelected}
                ref={(el) => {
                  if (el) el.indeterminate = !allSelected && someSelected;
                }}
                onChange={() => (allSelected ? onClearAll() : onSelectAll())}
                className="size-5 accent-[var(--accent-primary)]"
              />
            </th>
            {COLUMNS.map((col) => {
              const active = col.sort === sortField;
              return (
                <th
                  key={col.key}
                  scope="col"
                  aria-sort={col.sort ? ariaSort(active, sortOrder) : undefined}
                  className={cn('px-3 py-3 font-semibold', col.thClass)}
                >
                  {col.sort ? (
                    <button
                      type="button"
                      onClick={() => onSort(col.sort as SortField)}
                      data-testid={`downloads-sort-${col.sort}`}
                      className="inline-flex items-center gap-1 transition-colors hover:text-[var(--text-primary)]"
                    >
                      {col.label}
                      {active ? (
                        sortOrder === 'asc' ? (
                          <ArrowUp className="size-3" aria-hidden="true" />
                        ) : (
                          <ArrowDown className="size-3" aria-hidden="true" />
                        )
                      ) : (
                        <ChevronsUpDown
                          className="size-3 text-[var(--text-muted)]"
                          aria-hidden="true"
                        />
                      )}
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
            const pct = Math.round(d.progress * 100);
            const isSel = selected.has(d.hash);
            const tone = getDownloadTone(d);
            const meta = formatDownloadMeta(d);
            return (
              <tr
                key={d.hash}
                data-testid={`downloads-table-row-${d.hash}`}
                className={cn(
                  'border-b border-[var(--border-subtle)] transition-colors last:border-b-0',
                  isSel ? 'bg-[var(--accent-subtle)]' : 'hover:bg-[var(--bg-tertiary)]/40'
                )}
              >
                <td className="px-3 py-1">
                  <input
                    type="checkbox"
                    checked={isSel}
                    onChange={(e) => onSelectChange(d.hash, e.target.checked)}
                    aria-label={`選取 ${d.name}`}
                    className="size-5 accent-[var(--accent-primary)]"
                  />
                </td>
                <td className="px-3 py-1">
                  <div className="flex min-w-0 items-center gap-2">
                    <span
                      className="shrink-0 rounded-full bg-[var(--bg-tertiary)] px-2 py-0.5 text-xs text-[var(--text-muted)]"
                      title="qBittorrent"
                    >
                      qB
                    </span>
                    <span
                      className="truncate font-medium text-[var(--text-primary)]"
                      title={d.name}
                    >
                      {d.name}
                    </span>
                  </div>
                </td>
                <td className="px-3 py-1">
                  {/* Under the status pill, not in the name cell: the name keeps its width and every
                      row's name starts in the same place (dl-import-2 CR). */}
                  <div className="flex flex-col items-start gap-1">
                    <DownloadStatusPill download={d} />
                    {d.importStatus && <ImportStatusChip status={d.importStatus} variant="table" />}
                  </div>
                </td>
                <td className="px-3 py-1">
                  <div className="flex items-center gap-2">
                    <div className="h-[5px] flex-1 overflow-hidden rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)]">
                      <div
                        className={cn('h-full rounded-[var(--radius-sm)]', tone.fill)}
                        style={{ width: `${Math.min(pct, 100)}%` }}
                        role="progressbar"
                        aria-label={`${d.name} 下載進度`}
                        aria-valuenow={pct}
                        aria-valuemin={0}
                        aria-valuemax={100}
                      />
                    </div>
                    <span
                      className={cn(
                        'shrink-0 font-mono text-xs font-semibold tabular-nums',
                        tone.text
                      )}
                    >
                      {formatProgress(d.progress)}
                    </span>
                  </div>
                </td>
                <td className="whitespace-nowrap px-3 py-1 font-mono text-xs tabular-nums">
                  <span className="block text-[var(--text-secondary)]">{meta.down}</span>
                  <span className="block text-[var(--text-muted)]">{meta.up}</span>
                </td>
                <td className="whitespace-nowrap px-3 py-1 font-mono text-xs tabular-nums text-[var(--text-secondary)]">
                  {meta.eta}
                </td>
                <td className="whitespace-nowrap px-3 py-1 font-mono text-xs tabular-nums text-[var(--text-secondary)]">
                  {meta.sizeCompact}
                </td>
                <td className="px-3 py-1">
                  <DownloadRowActions
                    download={d}
                    variant="table"
                    onPause={onPause}
                    onResume={onResume}
                    onRemove={onRemove}
                  />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
