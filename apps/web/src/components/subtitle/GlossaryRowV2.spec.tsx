import { describe, it, expect, vi } from 'vitest';
import { act, render, screen, fireEvent, waitFor } from '@testing-library/react';
import { GlossaryRowV2 } from './GlossaryRowV2';
import type { GlossaryTerm } from '../../services/glossaryService';

const term = (overrides: Partial<GlossaryTerm> = {}): GlossaryTerm => ({
  id: 't1',
  mediaId: '42',
  termSrc: 'Demogorgon',
  termZh: '魔王獸',
  language: 'zh-Hant',
  source: 'subtitle',
  confirmed: false,
  createdAt: '2026-07-01T00:00:00Z',
  updatedAt: '2026-07-01T00:00:00Z',
  ...overrides,
});

const noop = () => undefined;
// dsr-6c AC #5: onEdit reports the save result so a failed save keeps the row in edit mode.
const resolved = () => Promise.resolve();

/** Exact class-set equality — `toHaveClass` alone passes on a superset. */
function expectExactClasses(el: HTMLElement, classes: string) {
  const expected = classes.split(' ');
  expect(el).toHaveClass(...expected);
  expect(el.classList).toHaveLength(expected.length);
}

function startEditing(value: string) {
  fireEvent.click(screen.getByTestId('glossary-edit-t1'));
  const input = screen.getByTestId('glossary-edit-input-t1');
  fireEvent.change(input, { target: { value } });
  return input;
}

describe('GlossaryRowV2', () => {
  it('renders the term pair — src in Mono, zh in default (Noto) font', () => {
    render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={resolved} onDelete={noop} />);

    const src = screen.getByText('Demogorgon');
    expect(src).toBeInTheDocument();
    expect(src.className).toContain('font-mono');
    const zh = screen.getByText('魔王獸');
    expect(zh).toBeInTheDocument();
    expect(zh.className).not.toContain('font-mono');
  });

  it.each([
    ['subtitle', '字幕'],
    ['metadata', '中繼資料'],
    ['manual', '手動'],
    // sub-7-1 AC #4: the two provenances the shared TMDb drawer brings in.
    ['official_subtitle', '官方字幕'],
    ['community', '社群'],
  ] as const)('renders the %s source badge as %s', (source, label) => {
    render(
      <GlossaryRowV2 term={term({ source })} onConfirm={noop} onEdit={resolved} onDelete={noop} />
    );
    expect(screen.getByText(label)).toBeInTheDocument();
  });

  it.each(['subtitle', 'metadata', 'manual', 'official_subtitle', 'community'] as const)(
    'the %s source badge is a neutral pill — a source is a property, not a state (dsr-6c ruling ①)',
    (source) => {
      render(
        <GlossaryRowV2 term={term({ source })} onConfirm={noop} onEdit={resolved} onDelete={noop} />
      );
      expectExactClasses(
        screen.getByTestId('glossary-source-t1'),
        'shrink-0 rounded-full bg-[var(--bg-tertiary)] px-2.5 py-1 text-[11px] text-[var(--text-secondary)]'
      );
    }
  );

  it('marks unconfirmed rows visually distinct (未確認 badge + 確認 action)', () => {
    render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={resolved} onDelete={noop} />);

    const badge = screen.getByTestId('glossary-unconfirmed-t1');
    expect(badge).toHaveTextContent(/^未確認$/);
    // 未確認 keeps ochre: unconfirmed terms already feed translation (confirmedOnly=false).
    expectExactClasses(
      badge,
      'shrink-0 rounded-full bg-[var(--warning-tint)] px-2.5 py-1 text-[11px] text-[var(--warning-text)]'
    );
    expect(screen.getByTestId('glossary-confirm-t1')).toBeInTheDocument();
  });

  it('hides the 未確認 badge and 確認 action on confirmed rows, but keeps 編輯 (dsr-6c ruling ②)', () => {
    render(
      <GlossaryRowV2
        term={term({ confirmed: true })}
        onConfirm={noop}
        onEdit={resolved}
        onDelete={noop}
      />
    );

    expect(screen.queryByTestId('glossary-unconfirmed-t1')).not.toBeInTheDocument();
    expect(screen.queryByTestId('glossary-confirm-t1')).not.toBeInTheDocument();
    expect(screen.getByTestId('glossary-edit-t1')).toBeInTheDocument();
  });

  it('names 確認 and 編輯 after the term so a screen reader can tell rows apart', () => {
    render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={resolved} onDelete={noop} />);

    expect(screen.getByRole('button', { name: '確認 Demogorgon' })).toBe(
      screen.getByTestId('glossary-confirm-t1')
    );
    expect(screen.getByRole('button', { name: '編輯 Demogorgon' })).toBe(
      screen.getByTestId('glossary-edit-t1')
    );
  });

  it('calls onConfirm with the term id', () => {
    const onConfirm = vi.fn();
    render(<GlossaryRowV2 term={term()} onConfirm={onConfirm} onEdit={resolved} onDelete={noop} />);

    fireEvent.click(screen.getByTestId('glossary-confirm-t1'));
    expect(onConfirm).toHaveBeenCalledWith('t1');
  });

  it('edit flow: 編輯 → input → 儲存 calls onEdit with the new zh text', async () => {
    const onEdit = vi.fn().mockResolvedValue(undefined);
    render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={onEdit} onDelete={noop} />);

    startEditing('魔神獸');
    fireEvent.click(screen.getByTestId('glossary-save-t1'));

    expect(onEdit).toHaveBeenCalledWith('t1', '魔神獸');
    await waitFor(() =>
      expect(screen.queryByTestId('glossary-edit-input-t1')).not.toBeInTheDocument()
    );
  });

  it('edit flow: unchanged text does NOT call onEdit', () => {
    const onEdit = vi.fn().mockResolvedValue(undefined);
    render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={onEdit} onDelete={noop} />);

    fireEvent.click(screen.getByTestId('glossary-edit-t1'));
    fireEvent.click(screen.getByTestId('glossary-save-t1'));

    expect(onEdit).not.toHaveBeenCalled();
    expect(screen.queryByTestId('glossary-edit-input-t1')).not.toBeInTheDocument();
  });

  it('a failed save keeps the row in edit mode with the typed text (dsr-6c AC #5)', async () => {
    const onEdit = vi.fn().mockRejectedValue(new Error('500'));
    render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={onEdit} onDelete={noop} />);

    startEditing('魔神獸');
    fireEvent.click(screen.getByTestId('glossary-save-t1'));

    await waitFor(() => expect(onEdit).toHaveBeenCalledTimes(1));
    // Let the rejection settle inside act, then the row must still be editing.
    await act(async () => {});
    expect(screen.getByTestId('glossary-edit-input-t1')).toHaveValue('魔神獸');
  });

  it('while its own save is in flight, Esc and 取消 wait — a late failure still hands back the text (CR MED-1)', async () => {
    let rejectSave: (err: Error) => void = () => undefined;
    const onEdit = vi.fn(
      () =>
        new Promise<void>((_resolve, reject) => {
          rejectSave = reject;
        })
    );
    render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={onEdit} onDelete={noop} />);

    const input = startEditing('魔神獸');
    fireEvent.keyDown(input, { key: 'Enter' });
    expect(onEdit).toHaveBeenCalledTimes(1);

    fireEvent.keyDown(input, { key: 'Escape' });
    fireEvent.click(screen.getByTestId('glossary-cancel-t1'));
    expect(screen.getByTestId('glossary-edit-input-t1')).toHaveValue('魔神獸');

    await act(async () => rejectSave(new Error('500')));
    expect(screen.getByTestId('glossary-edit-input-t1')).toHaveValue('魔神獸');
  });

  it('儲存 and 取消 use aria-disabled (not disabled) while saving, so focus is not dropped to <body>', async () => {
    const onEdit = vi.fn(() => new Promise<void>(() => undefined));
    render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={onEdit} onDelete={noop} />);

    startEditing('魔神獸');
    const save = screen.getByTestId('glossary-save-t1');
    fireEvent.click(save);

    await waitFor(() => expect(save).toHaveAttribute('aria-disabled', 'true'));
    expect(save).not.toBeDisabled();
    expect(screen.getByTestId('glossary-cancel-t1')).toHaveAttribute('aria-disabled', 'true');
    // A second click while saving does not send again.
    fireEvent.click(save);
    expect(onEdit).toHaveBeenCalledTimes(1);
  });

  it('the edit input matches Component/TextField/Default (aaiLz) without copying its size', () => {
    render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={resolved} onDelete={noop} />);

    fireEvent.click(screen.getByTestId('glossary-edit-t1'));
    expectExactClasses(
      screen.getByTestId('glossary-edit-input-t1'),
      'w-32 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent-primary)] focus:outline-none'
    );
  });

  it('marks the row as an inline editor only while editing (panel Esc guard, dsr-6c AC #6)', () => {
    render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={resolved} onDelete={noop} />);

    const row = screen.getByTestId('glossary-row-t1');
    expect(row).not.toHaveAttribute('data-glossary-inline-editor');
    fireEvent.click(screen.getByTestId('glossary-edit-t1'));
    expect(row).toHaveAttribute('data-glossary-inline-editor', '');
  });

  describe('keyboard (dsr-6c AC #6, #7)', () => {
    it('Enter saves', async () => {
      const onEdit = vi.fn().mockResolvedValue(undefined);
      render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={onEdit} onDelete={noop} />);

      fireEvent.keyDown(startEditing('魔神獸'), { key: 'Enter' });
      expect(onEdit).toHaveBeenCalledWith('t1', '魔神獸');
      await waitFor(() =>
        expect(screen.queryByTestId('glossary-edit-input-t1')).not.toBeInTheDocument()
      );
    });

    it.each([
      ['isComposing', { key: 'Enter', isComposing: true }],
      ['keyCode 229 (Safari)', { key: 'Enter', keyCode: 229 }],
    ])('an Enter that only picks an IME candidate (%s) does not save', (_label, init) => {
      const onEdit = vi.fn().mockResolvedValue(undefined);
      render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={onEdit} onDelete={noop} />);

      fireEvent.keyDown(startEditing('魔ㄕㄣˊ'), init);
      expect(onEdit).not.toHaveBeenCalled();
      expect(screen.getByTestId('glossary-edit-input-t1')).toBeInTheDocument();
    });

    it('Enter while busy does not save', () => {
      const onEdit = vi.fn().mockResolvedValue(undefined);
      const { rerender } = render(
        <GlossaryRowV2 term={term()} onConfirm={noop} onEdit={onEdit} onDelete={noop} />
      );

      const input = startEditing('魔神獸');
      rerender(
        <GlossaryRowV2 term={term()} busy onConfirm={noop} onEdit={onEdit} onDelete={noop} />
      );
      fireEvent.keyDown(input, { key: 'Enter' });
      expect(onEdit).not.toHaveBeenCalled();
    });

    it.each([
      ['the input', 'glossary-edit-input-t1'],
      ['儲存', 'glossary-save-t1'],
      ['取消', 'glossary-cancel-t1'],
    ])('Esc on %s cancels the edit and restores the text', (_label, testId) => {
      const onEdit = vi.fn().mockResolvedValue(undefined);
      render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={onEdit} onDelete={noop} />);

      startEditing('魔神獸');
      fireEvent.keyDown(screen.getByTestId(testId), { key: 'Escape' });

      expect(screen.queryByTestId('glossary-edit-input-t1')).not.toBeInTheDocument();
      expect(screen.getByText('魔王獸')).toBeInTheDocument();
      expect(onEdit).not.toHaveBeenCalled();
    });

    it('an Esc that only drops an IME candidate does not cancel the edit', () => {
      render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={resolved} onDelete={noop} />);

      const input = startEditing('魔ㄕ');
      fireEvent.keyDown(input, { key: 'Escape', isComposing: true });
      expect(screen.getByTestId('glossary-edit-input-t1')).toHaveValue('魔ㄕ');
    });
  });

  it('delete is a text button 「刪除」, not an icon (nDSEd Q11NpX)', () => {
    render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={resolved} onDelete={noop} />);

    const del = screen.getByTestId('glossary-delete-t1');
    expect(del).toHaveTextContent(/^刪除$/);
    expect(del.querySelector('svg')).toBeNull();
    expect(del).toHaveAccessibleName('刪除 Demogorgon');
    expectExactClasses(
      del,
      'flex min-h-[44px] shrink-0 items-center px-2.5 text-sm text-[var(--error-text)] hover:underline disabled:opacity-50'
    );
  });

  it('delete is gated behind a Radix confirm dialog (destructive confirm, AC 7)', () => {
    const onDelete = vi.fn();
    render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={resolved} onDelete={onDelete} />);

    fireEvent.click(screen.getByTestId('glossary-delete-t1'));
    expect(onDelete).not.toHaveBeenCalled(); // dialog first, never direct

    expect(screen.getByText('刪除詞彙')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('glossary-delete-confirm-t1'));
    expect(onDelete).toHaveBeenCalledWith('t1');
  });

  it('the delete confirm dialog follows Component/DialogFrame (m6KMPr) — 480 wide, radius-lg, H3 title', () => {
    render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={resolved} onDelete={noop} />);

    fireEvent.click(screen.getByTestId('glossary-delete-t1'));
    expect(screen.getByTestId('glossary-delete-dialog-t1')).toHaveClass(
      'max-w-[480px]',
      'rounded-[var(--radius-lg)]',
      'flex',
      'flex-col',
      'gap-4'
    );
    expect(screen.getByText('刪除詞彙')).toHaveClass('text-xl');
    expect(screen.getByRole('button', { name: '取消' })).toHaveClass('px-5', 'font-medium');
    expect(screen.getByTestId('glossary-delete-confirm-t1')).toHaveClass('px-5', 'font-semibold');
  });

  it('disables actions while busy', () => {
    render(<GlossaryRowV2 term={term()} busy onConfirm={noop} onEdit={resolved} onDelete={noop} />);

    expect(screen.getByTestId('glossary-confirm-t1')).toBeDisabled();
    expect(screen.getByTestId('glossary-edit-t1')).toBeDisabled();
    expect(screen.getByTestId('glossary-delete-t1')).toBeDisabled();
  });
});
