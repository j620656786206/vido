import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
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

describe('GlossaryRowV2', () => {
  it('renders the term pair — src in Mono, zh in default (Noto) font', () => {
    render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={noop} onDelete={noop} />);

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
  ] as const)('renders the %s source badge as %s', (source, label) => {
    render(
      <GlossaryRowV2 term={term({ source })} onConfirm={noop} onEdit={noop} onDelete={noop} />
    );
    expect(screen.getByText(label)).toBeInTheDocument();
  });

  it('marks unconfirmed rows visually distinct (未確認 badge + 確認 action)', () => {
    render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={noop} onDelete={noop} />);

    expect(screen.getByTestId('glossary-unconfirmed-t1')).toHaveTextContent('未確認');
    expect(screen.getByTestId('glossary-confirm-t1')).toBeInTheDocument();
  });

  it('hides the 未確認 badge and 確認 action on confirmed rows', () => {
    render(
      <GlossaryRowV2
        term={term({ confirmed: true })}
        onConfirm={noop}
        onEdit={noop}
        onDelete={noop}
      />
    );

    expect(screen.queryByTestId('glossary-unconfirmed-t1')).not.toBeInTheDocument();
    expect(screen.queryByTestId('glossary-confirm-t1')).not.toBeInTheDocument();
  });

  it('calls onConfirm with the term id', () => {
    const onConfirm = vi.fn();
    render(<GlossaryRowV2 term={term()} onConfirm={onConfirm} onEdit={noop} onDelete={noop} />);

    fireEvent.click(screen.getByTestId('glossary-confirm-t1'));
    expect(onConfirm).toHaveBeenCalledWith('t1');
  });

  it('edit flow: 編輯 → input → 儲存 calls onEdit with the new zh text', () => {
    const onEdit = vi.fn();
    render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={onEdit} onDelete={noop} />);

    fireEvent.click(screen.getByTestId('glossary-edit-t1'));
    const input = screen.getByTestId('glossary-edit-input-t1');
    fireEvent.change(input, { target: { value: '魔神獸' } });
    fireEvent.click(screen.getByTestId('glossary-save-t1'));

    expect(onEdit).toHaveBeenCalledWith('t1', '魔神獸');
  });

  it('edit flow: unchanged text does NOT call onEdit', () => {
    const onEdit = vi.fn();
    render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={onEdit} onDelete={noop} />);

    fireEvent.click(screen.getByTestId('glossary-edit-t1'));
    fireEvent.click(screen.getByTestId('glossary-save-t1'));

    expect(onEdit).not.toHaveBeenCalled();
  });

  it('delete is gated behind a Radix confirm dialog (destructive confirm, AC 7)', () => {
    const onDelete = vi.fn();
    render(<GlossaryRowV2 term={term()} onConfirm={noop} onEdit={noop} onDelete={onDelete} />);

    fireEvent.click(screen.getByTestId('glossary-delete-t1'));
    expect(onDelete).not.toHaveBeenCalled(); // dialog first, never direct

    expect(screen.getByText('刪除詞彙')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('glossary-delete-confirm-t1'));
    expect(onDelete).toHaveBeenCalledWith('t1');
  });

  it('disables actions while busy', () => {
    render(<GlossaryRowV2 term={term()} busy onConfirm={noop} onEdit={noop} onDelete={noop} />);

    expect(screen.getByTestId('glossary-confirm-t1')).toBeDisabled();
    expect(screen.getByTestId('glossary-edit-t1')).toBeDisabled();
    expect(screen.getByTestId('glossary-delete-t1')).toBeDisabled();
  });
});
