import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import {
  ConfirmGenerationDialog,
  type ConfirmGenerationDialogProps,
} from './ConfirmGenerationDialog';
import type { ConsentTotals, ModelChoice } from './consentSelection';

const baseTotals: ConsentTotals = {
  candidateCount: 142,
  selectableCount: 142,
  unwritableCount: 0,
  unpricedCount: 0,
  unpricedSelectedCount: 0,
  selectedCount: 18,
  selectedExtractCount: 6,
  selectedAsrCount: 12,
  selectedExtractUsd: 0.18,
  selectedAsrUsd: 4.32,
  selectedTotalUsd: 4.5,
  overBudget: false,
  feasibleCount: 18,
  visibleSelectableCount: 142,
  visibleSelectedCount: 18,
  visibleSelectedTotalUsd: 4.5,
  hasEstimatedRows: false,
  estimatedRowCount: 0,
  cutMediaId: null,
  pausedIds: new Set<string>(),
};

const MODEL_CHOICES: ModelChoice[] = [
  {
    id: 'claude-sonnet-5',
    displayName: 'Claude Sonnet 5',
    isDefault: true,
    qualityGrade: 'A',
    isBestGrade: true,
    totalUsd: 4.5,
    minutes: 11,
  },
  {
    id: 'claude-haiku-4-5',
    displayName: 'Claude Haiku 4.5',
    isDefault: false,
    qualityGrade: 'B',
    isBestGrade: false,
    totalUsd: 1.7,
    minutes: 9,
    deltaUsd: 2.8,
    deltaPercent: 62,
  },
];

function renderDialog(
  totals: Partial<ConsentTotals> = {},
  budgetUsd = 5,
  picker?: Partial<ConfirmGenerationDialogProps>
) {
  const onConfirm = vi.fn();
  const onCancel = vi.fn();
  render(
    <ConfirmGenerationDialog
      open
      totals={{ ...baseTotals, ...totals }}
      budgetUsd={budgetUsd}
      onConfirm={onConfirm}
      onCancel={onCancel}
      {...picker}
    />
  );
  return { onConfirm, onCancel };
}

describe('ConfirmGenerationDialog (F16/F19)', () => {
  it("[dsr-6f-1] phone sheet shell: slides up, and the 44px ✕ sits on this dialog's 56px title row", () => {
    renderDialog();
    const shell = screen.getByTestId('consent-confirm-dialog');
    const t = (el: Element) => el.className.split(/\s+/);
    expect(t(shell)).toContain('max-sm:data-[state=open]:animate-sheet-enter');
    const close = screen.getByText('Close').closest('button')!;
    // Grabber 16 + half of h-14 (28) − half of 44 (22) = 22. It moves with the
    // header: change the title row's height and this token must change too.
    expect(t(close)).toEqual(expect.arrayContaining(['max-sm:h-11', 'max-sm:top-[22px]']));
  });

  it('[P0 F16] under budget: breakdown lines, neutral hint, 確認並開始', () => {
    const { onConfirm } = renderDialog();
    // dsr-6e-2: the testid wraps the AMOUNT only (「預估」 is its own muted node).
    expect(screen.getByTestId('consent-confirm-asr-usd').textContent).toBe('$4.32');
    expect(screen.getByTestId('consent-confirm-extract-usd').textContent).toBe('$0.18');
    for (const id of ['consent-confirm-asr-usd', 'consent-confirm-extract-usd']) {
      expect(screen.getByTestId(id).className).toContain('text-[var(--text-primary)]');
    }
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
    // dsr-6e-2: money wears no status colour, over budget included — the ochre
    // hint block and 仍要開始 carry the state.
    const total = screen.getByTestId('consent-confirm-total-usd').className;
    expect(total).toContain('text-[var(--text-primary)]');
    expect(total).toContain('text-base');
    expect(total).toContain('font-bold');
    expect(total).not.toContain('warning');
    expect(hint.className).toContain('bg-[var(--warning-tint)]');
    expect(hint.className).toContain('text-[var(--text-primary)]');
  });

  it('[dsr-6e-2] shell 480 + hairline, lead is BodyLg 600, primary is 14/600, no 13px', () => {
    renderDialog();
    const shell = screen.getByTestId('consent-confirm-dialog');
    expect(shell.className).toContain('sm:max-w-[480px]');
    expect(shell.className).toContain('sm:rounded-[var(--radius-lg)]');
    expect(shell.className).toContain('sm:border-[var(--border-subtle)]');
    expect(screen.getByText(/即將為/).className).toContain('text-base font-semibold');
    const start = screen.getByTestId('consent-confirm-start').className;
    expect(start).toContain('font-semibold');
    expect(start).toContain('px-5');
    expect(shell.innerHTML).not.toContain('text-[13px]');
  });

  it('取消 dispatches onCancel', () => {
    const { onCancel, onConfirm } = renderDialog();
    fireEvent.click(screen.getByTestId('consent-confirm-cancel'));
    expect(onCancel).toHaveBeenCalledOnce();
    expect(onConfirm).not.toHaveBeenCalled();
  });
});

describe('ConfirmGenerationDialog — 翻譯模型 (sub-6-8b)', () => {
  it('[P0 AC #1] renders the picker above the breakdown with the default pre-selected', () => {
    const onModelChange = vi.fn();
    renderDialog({}, 5, {
      modelChoices: MODEL_CHOICES,
      selectedModelId: 'claude-sonnet-5',
      onModelChange,
    });

    expect(screen.getByTestId('consent-model-picker')).toBeInTheDocument();
    expect(screen.getByTestId('consent-model-option-claude-sonnet-5')).toHaveAttribute(
      'data-selected',
      'true'
    );
    // The breakdown still renders the ONE totals value it was handed.
    expect(screen.getByTestId('consent-confirm-total-usd')).toHaveTextContent('$4.50');
  });

  it('[AC #4] 仍要開始 / 確認並開始 wording is untouched by the model choice — inform, never nag', () => {
    renderDialog({}, 5, {
      modelChoices: MODEL_CHOICES,
      selectedModelId: 'claude-haiku-4-5',
      onModelChange: vi.fn(),
    });
    expect(screen.getByTestId('consent-confirm-start').textContent).toContain('確認並開始');
  });

  it('no catalog → no picker, and the dialog is otherwise unchanged', () => {
    renderDialog();
    expect(screen.queryByTestId('consent-model-picker')).not.toBeInTheDocument();
    expect(screen.getByTestId('consent-confirm-total-usd')).toHaveTextContent('$4.50');
  });
});

describe('ConfirmGenerationDialog — a total built out of guesses (sub-6-12 AC #4)', () => {
  const estimated: ConsentTotals = {
    ...baseTotals,
    hasEstimatedRows: true,
    estimatedRowCount: 12,
  };

  it('marks the total and says how many rows made it soft', () => {
    render(
      <ConfirmGenerationDialog
        open
        totals={estimated}
        budgetUsd={5}
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );
    expect(screen.getByTestId('consent-confirm-total-usd')).toHaveTextContent('≈ $4.50');
    expect(screen.getByTestId('consent-confirm-estimated-note')).toHaveTextContent(
      '其中 12 部片長未知，以 45 分鐘估算'
    );
  });

  it('says nothing when every selected runtime was measured', () => {
    render(
      <ConfirmGenerationDialog
        open
        totals={baseTotals}
        budgetUsd={5}
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    );
    expect(screen.getByTestId('consent-confirm-total-usd').textContent).toBe('$4.50');
    expect(screen.queryByTestId('consent-confirm-estimated-note')).toBeNull();
  });
});
