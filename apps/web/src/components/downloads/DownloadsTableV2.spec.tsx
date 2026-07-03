import { describe, it, expect, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DownloadsTableV2 } from './DownloadsTableV2';
import type { Download, SortField, SortOrder } from '../../services/downloadService';

const item = (over: Partial<Download> = {}): Download => ({
  hash: 'a',
  name: 'Alpha.mkv',
  size: 5_000_000_000,
  progress: 0.42,
  downloadSpeed: 1_500_000,
  uploadSpeed: 0,
  eta: 3600,
  status: 'downloading',
  addedOn: '2026-07-01T00:00:00Z',
  seeds: 1,
  peers: 1,
  downloaded: 0,
  uploaded: 0,
  ratio: 0,
  savePath: '/dl',
  ...over,
});

const items = [
  item({ hash: 'a', name: 'Alpha.mkv', status: 'downloading', progress: 0.42 }),
  item({ hash: 'b', name: 'Bravo.mkv', status: 'completed', progress: 1 }),
];

function renderTable(
  over: { sortField?: SortField; sortOrder?: SortOrder; selected?: Set<string> } = {}
) {
  const spies = {
    onSort: vi.fn(),
    onSelectChange: vi.fn(),
    onSelectAll: vi.fn(),
    onClearAll: vi.fn(),
    onPause: vi.fn(),
    onResume: vi.fn(),
    onRemove: vi.fn(),
  };
  render(
    <DownloadsTableV2
      items={items}
      sortField={over.sortField ?? 'added_on'}
      sortOrder={over.sortOrder ?? 'desc'}
      selected={over.selected ?? new Set()}
      {...spies}
    />
  );
  return spies;
}

describe('DownloadsTableV2 (ux3-4-4 AC2/3/4/5)', () => {
  it('renders a semantic table with a row per download + status token + Mono numerics', () => {
    renderTable();
    expect(screen.getByTestId('downloads-table-v2')).toBeInTheDocument();
    expect(screen.getByRole('table')).toBeInTheDocument();
    expect(screen.getByTestId('downloads-table-row-a')).toBeInTheDocument();
    expect(screen.getByTestId('download-status-a')).toHaveTextContent('下載中');
    // downloading row's speed cell is a Mono numeric
    const speed = screen.getByText(/↓/);
    expect(speed).toHaveClass('font-mono');
    expect(speed).toHaveClass('tabular-nums');
  });

  it('sortable headers carry aria-sort; the active field shows its direction (AC3)', () => {
    renderTable({ sortField: 'name', sortOrder: 'asc' });
    expect(screen.getByRole('columnheader', { name: /名稱/ })).toHaveAttribute(
      'aria-sort',
      'ascending'
    );
    expect(screen.getByRole('columnheader', { name: /狀態/ })).toHaveAttribute('aria-sort', 'none');
    // a non-sortable column (大小) exposes no aria-sort
    expect(screen.getByRole('columnheader', { name: '大小' })).not.toHaveAttribute('aria-sort');
  });

  it('clicking a sortable header calls onSort with the field (AC3)', async () => {
    const { onSort } = renderTable();
    await userEvent.click(screen.getByTestId('downloads-sort-progress'));
    expect(onSort).toHaveBeenCalledWith('progress');
  });

  it('a row checkbox toggles selection; the header checkbox selects all (AC4)', async () => {
    const { onSelectChange, onSelectAll } = renderTable();
    await userEvent.click(screen.getByRole('checkbox', { name: '選取 Alpha.mkv' }));
    expect(onSelectChange).toHaveBeenCalledWith('a', true);
    await userEvent.click(screen.getByRole('checkbox', { name: '全選' }));
    expect(onSelectAll).toHaveBeenCalled();
  });

  it('when all rows are selected the header checkbox clears all', async () => {
    const { onClearAll } = renderTable({ selected: new Set(['a', 'b']) });
    await userEvent.click(screen.getByRole('checkbox', { name: '取消全選' }));
    expect(onClearAll).toHaveBeenCalled();
  });

  it('row actions reuse DownloadRowActions — pause + destructive remove (AC5)', async () => {
    const { onPause, onRemove } = renderTable();
    const row = screen.getByTestId('downloads-table-row-a');
    await userEvent.click(within(row).getByRole('button', { name: /暫停/ }));
    expect(onPause).toHaveBeenCalledWith('a');

    await userEvent.click(within(row).getByRole('button', { name: /移除/ }));
    await userEvent.click(await screen.findByRole('button', { name: '移除（連同檔案刪除）' }));
    expect(onRemove).toHaveBeenCalledWith('a', true);
  });
});
