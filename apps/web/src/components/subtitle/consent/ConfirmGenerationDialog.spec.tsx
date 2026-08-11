import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ConfirmGenerationDialog } from './ConfirmGenerationDialog';
import type { ConsentTotals } from './consentSelection';

const baseTotals: ConsentTotals = {
  candidateCount: 142,
  selectedCount: 18,
  selectedExtractCount: 6,
  selectedAsrCount: 12,
  selectedExtractUsd: 0.18,
  selectedAsrUsd: 4.32,
  selectedTotalUsd: 4.5,
  overBudget: false,
  feasibleCount: 18,
};

function renderDialog(totals: Partial<ConsentTotals> = {}, budgetUsd = 5) {
  const onConfirm = vi.fn();
  const onCancel = vi.fn();
  render(
    <ConfirmGenerationDialog
      open
      totals={{ ...baseTotals, ...totals }}
      budgetUsd={budgetUsd}
      onConfirm={onConfirm}
      onCancel={onCancel}
    />
  );
  return { onConfirm, onCancel };
}

describe('ConfirmGenerationDialog (F16/F19)', () => {
  it('[P0 F16] under budget: breakdown lines, neutral hint, 確認並開始', () => {
    const { onConfirm } = renderDialog();
    expect(screen.getByTestId('consent-confirm-asr-usd').textContent).toContain('$4.32');
    expect(screen.getByTestId('consent-confirm-extract-usd').textContent).toContain('$0.18');
    expect(screen.getByTestId('consent-confirm-total-usd').textContent).toBe('$4.50');
    const hint = screen.getByTestId('consent-confirm-hint');
    expect(hint.textContent).toContain('自動暫停');
    expect(hint.textContent).toContain('$5.00');
    expect(hint.textContent).not.toContain('絕不');
    const btn = screen.getByTestId('consent-confirm-start');
    expect(btn.textContent).toContain('確認並開始');
    fireEvent.click(btn);
    expect(onConfirm).toHaveBeenCalledOnce();
  });

  it('[P0 F19] over budget: warning hint with feasible count, 仍要開始', () => {
    renderDialog({
      selectedCount: 96,
      selectedExtractCount: 0,
      selectedAsrCount: 96,
      selectedExtractUsd: 0,
      selectedAsrUsd: 25.8,
      selectedTotalUsd: 25.8,
      overBudget: true,
      feasibleCount: 18,
    });
    const hint = screen.getByTestId('consent-confirm-hint');
    expect(hint.textContent).toContain('已超過上限');
    expect(hint.textContent).toContain('預計可完成約');
    expect(hint.textContent).toContain('18');
    expect(screen.getByTestId('consent-confirm-start').textContent).toContain('仍要開始');
    expect(screen.getByTestId('consent-confirm-total-usd').textContent).toBe('$25.80');
  });

  it('取消 dispatches onCancel', () => {
    const { onCancel, onConfirm } = renderDialog();
    fireEvent.click(screen.getByTestId('consent-confirm-cancel'));
    expect(onCancel).toHaveBeenCalledOnce();
    expect(onConfirm).not.toHaveBeenCalled();
  });
});
