import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { SortField, SortOrder } from '../../services/downloadService';
import { DownloadSortSheet } from './DownloadSortSheet';

// The same eight the desktop select lists (DownloadsBrowseV2 SORT_OPTIONS); the
// sheet never keeps a copy of its own — it renders whatever it is handed.
const OPTIONS: { field: SortField; order: SortOrder; label: string }[] = [
  { field: 'added_on', order: 'desc', label: '加入時間（新到舊）' },
  { field: 'added_on', order: 'asc', label: '加入時間（舊到新）' },
  { field: 'name', order: 'asc', label: '名稱（A–Z）' },
  { field: 'name', order: 'desc', label: '名稱（Z–A）' },
  { field: 'progress', order: 'desc', label: '進度（多到少）' },
  { field: 'progress', order: 'asc', label: '進度（少到多）' },
  { field: 'status', order: 'asc', label: '狀態' },
  { field: 'status', order: 'desc', label: '狀態（反向）' },
];

const tokens = (el: Element) => (el.getAttribute('class') ?? '').split(/\s+/).filter(Boolean);

function renderSheet(value = 'added_on:desc') {
  const onChange = vi.fn();
  const onOpenChange = vi.fn();
  render(
    <DownloadSortSheet
      open
      onOpenChange={onOpenChange}
      options={OPTIONS}
      value={value}
      onChange={onChange}
    />
  );
  return { onChange, onOpenChange };
}

describe('DownloadSortSheet (dsr-4b-1 D10-M-v2)', () => {
  it('is a radiogroup named 排序方式 with the eight options and exactly one checked — the value', () => {
    renderSheet('name:desc');
    const group = screen.getByRole('radiogroup', { name: '排序方式' });
    const radios = within(group).getAllByRole('radio');
    expect(radios.map((r) => r.textContent)).toEqual(OPTIONS.map((o) => o.label));
    const checked = radios.filter((r) => r.getAttribute('aria-checked') === 'true');
    expect(checked).toHaveLength(1);
    expect(checked[0]).toHaveTextContent('名稱（Z–A）');
  });

  it('carries its own testid, the 排序 title and the subtitle as the dialog description', () => {
    renderSheet();
    const sheet = screen.getByTestId('download-sort-sheet');
    expect(screen.getByRole('dialog')).toHaveAccessibleName('排序');
    expect(screen.getByRole('dialog')).toHaveAccessibleDescription('選擇下載清單的排序方式');
    expect(within(sheet).getByText('選擇下載清單的排序方式')).toBeInTheDocument();
  });

  it('a value no row carries still leaves the first row in the tab order', async () => {
    renderSheet('size:desc');
    const radios = screen.getAllByRole('radio');
    expect(radios.filter((r) => r.getAttribute('aria-checked') === 'true')).toHaveLength(0);
    expect(radios.map((r) => r.getAttribute('tabindex'))).toEqual([
      '0',
      '-1',
      '-1',
      '-1',
      '-1',
      '-1',
      '-1',
      '-1',
    ]);
    await waitFor(() => expect(radios[0]).toHaveFocus());
  });

  it('picking 名稱（A–Z） → onChange("name:asc") and closes', async () => {
    const user = userEvent.setup();
    const { onChange, onOpenChange } = renderSheet();
    await user.click(screen.getByRole('radio', { name: '名稱（A–Z）' }));
    expect(onChange).toHaveBeenCalledWith('name:asc');
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('picking the row that is already checked still closes', async () => {
    const user = userEvent.setup();
    const { onChange, onOpenChange } = renderSheet();
    await user.click(screen.getByRole('radio', { name: '加入時間（新到舊）' }));
    expect(onChange).toHaveBeenCalledWith('added_on:desc');
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('roving tabindex: only the checked row is in the tab order, and it takes the initial focus', async () => {
    renderSheet('progress:desc');
    const radios = screen.getAllByRole('radio');
    expect(radios.map((r) => r.getAttribute('tabindex'))).toEqual([
      '-1',
      '-1',
      '-1',
      '-1',
      '0',
      '-1',
      '-1',
      '-1',
    ]);
    await waitFor(() =>
      expect(screen.getByRole('radio', { name: '進度（多到少）' })).toHaveFocus()
    );
  });

  it('ArrowDown / ArrowUp move focus and wrap at both ends; Enter picks', async () => {
    const user = userEvent.setup();
    const { onChange, onOpenChange } = renderSheet('status:desc');
    const last = screen.getByRole('radio', { name: '狀態（反向）' });
    await waitFor(() => expect(last).toHaveFocus());

    await user.keyboard('{ArrowDown}');
    expect(screen.getByRole('radio', { name: '加入時間（新到舊）' })).toHaveFocus();
    await user.keyboard('{ArrowUp}');
    expect(last).toHaveFocus();
    await user.keyboard('{ArrowUp}');
    expect(screen.getByRole('radio', { name: '狀態' })).toHaveFocus();
    // Moving focus does not pick anything — only Enter/Space/click does.
    expect(onChange).not.toHaveBeenCalled();

    await user.keyboard('{Enter}');
    expect(onChange).toHaveBeenCalledWith('status:asc');
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('Space picks too', async () => {
    const user = userEvent.setup();
    const { onChange } = renderSheet();
    await waitFor(() =>
      expect(screen.getByRole('radio', { name: '加入時間（新到舊）' })).toHaveFocus()
    );
    await user.keyboard('{ArrowDown}');
    await user.keyboard(' ');
    expect(onChange).toHaveBeenCalledWith('added_on:asc');
  });

  it('every row keeps a 20px icon slot — a check on the checked row, an empty spacer on the rest', () => {
    renderSheet('name:asc');
    for (const radio of screen.getAllByRole('radio')) {
      const slot = radio.firstElementChild!;
      expect(tokens(slot)).toContain('size-5');
      expect(tokens(slot)).toContain('shrink-0');
      expect(slot).toHaveAttribute('aria-hidden', 'true');
      if (radio.getAttribute('aria-checked') === 'true') {
        expect(slot.tagName.toLowerCase()).toBe('svg');
      } else {
        expect(slot.tagName.toLowerCase()).toBe('span');
        expect(slot).toBeEmptyDOMElement();
      }
    }
  });

  it('checked row wears the accent, the others the primary text at 500', () => {
    renderSheet('name:asc');
    const checked = screen.getByRole('radio', { name: '名稱（A–Z）' });
    const other = screen.getByRole('radio', { name: '名稱（Z–A）' });
    expect(tokens(checked)).toEqual(
      expect.arrayContaining([
        'bg-[var(--accent-subtle)]',
        'font-semibold',
        'text-[var(--accent-text)]',
        'min-h-[52px]',
      ])
    );
    expect(tokens(other)).toEqual(
      expect.arrayContaining(['font-medium', 'text-[var(--text-primary)]', 'min-h-[52px]'])
    );
    expect(tokens(other)).not.toContain('bg-[var(--accent-subtle)]');
  });
});
