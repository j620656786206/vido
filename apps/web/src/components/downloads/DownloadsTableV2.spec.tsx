import { describe, it, expect, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
// TanStack <Link> → a plain anchor with the params filled in (the import status chip links out).
vi.mock('@tanstack/react-router', () => ({
  Link: ({
    to,
    params,
    children,
    ...props
  }: {
    to: string;
    params: Record<string, string>;
    children: React.ReactNode;
  }) => (
    <a href={to.replace('$type', params.type).replace('$id', params.id)} {...props}>
      {children}
    </a>
  ),
}));

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
    const speed = within(screen.getByTestId('downloads-table-row-a')).getByText(/↓ 1\.4 MB\/s/);
    expect(speed.closest('td')).toHaveClass('font-mono');
    expect(speed.closest('td')).toHaveClass('tabular-nums');
    // a completed row says "—" instead of a zero speed
    expect(
      within(screen.getByTestId('downloads-table-row-b')).getByText('↓ —')
    ).toBeInTheDocument();
  });

  it('columns follow D7-D-v2: ☑ 名稱 狀態 進度 速度 ETA 大小 動作', () => {
    renderTable();
    const headers = screen.getAllByRole('columnheader').map((th) => th.textContent?.trim());
    expect(headers).toEqual(['', '名稱', '狀態', '進度', '速度', 'ETA', '大小', '動作']);
  });

  it('sortable headers carry aria-sort; the active field shows its direction (AC3)', () => {
    renderTable({ sortField: 'name', sortOrder: 'asc' });
    expect(screen.getByRole('columnheader', { name: /名稱/ })).toHaveAttribute(
      'aria-sort',
      'ascending'
    );
    expect(screen.getByRole('columnheader', { name: /狀態/ })).toHaveAttribute('aria-sort', 'none');
    // columns the API cannot sort expose no aria-sort (and no sort button)
    for (const name of ['速度', 'ETA', '大小']) {
      expect(screen.getByRole('columnheader', { name })).not.toHaveAttribute('aria-sort');
    }
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

  it('row actions reuse DownloadRowActions — pause + ⋯ → confirmed delete (AC5)', async () => {
    const { onPause, onRemove } = renderTable();
    const row = screen.getByTestId('downloads-table-row-a');
    await userEvent.click(within(row).getByRole('button', { name: /^暫停/ }));
    expect(onPause).toHaveBeenCalledWith('a');

    await userEvent.click(within(row).getByRole('button', { name: /更多動作/ }));
    await userEvent.click(await screen.findByRole('menuitem', { name: '移除（連同檔案刪除）' }));
    await userEvent.click(await screen.findByRole('button', { name: '刪除檔案' }));
    expect(onRemove).toHaveBeenCalledWith('a', true);
  });
});

describe('DownloadsTableV2 — import status (dl-import-2)', () => {
  it('shows the short chip under the status pill', () => {
    const rows = [
      item({
        hash: 'p',
        name: 'Show.S01.mkv',
        status: 'seeding',
        progress: 1,
        importStatus: {
          state: 'awaiting_scan',
          source: 'sonarr',
          mediaType: 'tv',
          mediaId: 's1',
          episodesImported: 9,
          episodesInLibrary: 6,
        },
      }),
    ];
    render(
      <DownloadsTableV2
        items={rows}
        sortField="added_on"
        sortOrder="desc"
        onSort={vi.fn()}
        selected={new Set()}
        onSelectChange={vi.fn()}
        onSelectAll={vi.fn()}
        onClearAll={vi.fn()}
      />
    );
    const chip = within(screen.getByTestId('downloads-table-row-p')).getByTestId(
      'download-import-status'
    );
    expect(chip).toHaveTextContent('6/9 集');
    // under the status pill, so the name keeps its column
    expect(chip.closest('td')).toContainElement(screen.getByTestId('download-status-p'));
    expect(chip).toHaveAttribute('href', '/media/tv/s1');
  });
});
