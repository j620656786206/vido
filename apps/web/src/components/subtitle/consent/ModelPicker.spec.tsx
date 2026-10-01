import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ModelPicker } from './ModelPicker';
import type { ModelChoice } from './consentSelection';

const SONNET: ModelChoice = {
  id: 'claude-sonnet-5',
  displayName: 'Claude Sonnet 5',
  isDefault: true,
  qualityGrade: 'A',
  qualityNote: 'Vido 實測 2026-09（eval-1 盲測，10,304 句）',
  isBestGrade: true,
  totalUsd: 0.53,
  minutes: 11,
};

const HAIKU: ModelChoice = {
  id: 'claude-haiku-4-5',
  displayName: 'Claude Haiku 4.5',
  isDefault: false,
  qualityGrade: 'B',
  qualityNote: 'Vido 實測 2026-09（eval-1 盲測，10,304 句）',
  isBestGrade: false,
  totalUsd: 0.21,
  minutes: 9,
  deltaUsd: 0.32,
  deltaPercent: 60,
};

const UNGRADED: ModelChoice = {
  id: 'gemini-2.5-flash',
  displayName: 'Gemini 2.5 Flash',
  isDefault: false,
  isBestGrade: false,
  totalUsd: 0.06,
  deltaUsd: 0.47,
  deltaPercent: 89,
};

function renderPicker(selectedModelId = SONNET.id, choices = [SONNET, HAIKU, UNGRADED]) {
  const onSelect = vi.fn();
  render(<ModelPicker choices={choices} selectedModelId={selectedModelId} onSelect={onSelect} />);
  return { onSelect };
}

describe('ModelPicker (F16/F19 翻譯模型)', () => {
  it('[P0 AC #1] every row states this batch price, the measured grade and the rough time', () => {
    renderPicker();

    // dsr-6e-2 ②: exact text — 「這批約 $0.53」 CONTAINS 「約 $0.53」, so a
    // substring match would let the old copy through.
    expect(screen.getByTestId('consent-model-usd-claude-sonnet-5').textContent).toBe('約 $0.53');
    expect(screen.getByTestId('consent-model-grade-claude-sonnet-5')).toHaveTextContent('品質 A');
    expect(screen.getByTestId('consent-model-minutes-claude-sonnet-5')).toHaveTextContent(
      '約 11 分鐘'
    );
    expect(screen.getByTestId('consent-model-usd-claude-haiku-4-5').textContent).toBe('約 $0.21');
  });

  it('[dsr-6e-2] the grade badge is a NEUTRAL pill — a grade is a property, not a state', () => {
    renderPicker();
    for (const id of ['claude-sonnet-5', 'claude-haiku-4-5']) {
      const cls = screen.getByTestId(`consent-model-grade-${id}`).className;
      expect(cls).toContain('bg-[var(--bg-tertiary)]');
      expect(cls).toContain('text-[var(--text-secondary)]');
      expect(cls).toContain('rounded-full');
      expect(cls).not.toMatch(/--success-(tint|text)/);
    }
  });

  it('[dsr-6e-2] type scale: title Body 600, name Body, amount Body 700 — no 13px', () => {
    renderPicker();
    const container = screen.getByTestId('consent-model-picker');
    expect(screen.getByText('選擇翻譯模型').className).toContain('text-sm font-semibold');
    expect(screen.getByTestId('consent-model-usd-claude-sonnet-5').className).toContain(
      'font-bold'
    );
    expect(container.innerHTML).not.toContain('text-[13px]');
  });

  it('[P0 AC #1] an unevaluated model says 尚未評測 — never a blank where a grade goes', () => {
    renderPicker();

    const badge = screen.getByTestId('consent-model-grade-gemini-2.5-flash');
    expect(badge).toHaveTextContent('尚未評測');
    expect(badge.textContent).not.toContain('品質');
    // No measurement, no duration claim.
    expect(screen.queryByTestId('consent-model-minutes-gemini-2.5-flash')).not.toBeInTheDocument();
    // sub-7-8c: without a preview handler the dead 「可花約 $0.01」 copy is gone too —
    // the row says nothing it cannot act on.
    expect(screen.getByTestId('consent-model-option-gemini-2.5-flash').textContent).not.toContain(
      '試跑'
    );
  });

  it('[P0 AC #4] the selected non-default row spells the gap out in money', () => {
    renderPicker(HAIKU.id);
    expect(screen.getByTestId('consent-model-note-claude-haiku-4-5')).toHaveTextContent(
      '比 Claude Sonnet 5 省 $0.32（60%）'
    );
    // Only the SELECTED row carries the comparison — three of them at once is noise.
    expect(screen.queryByTestId('consent-model-note-gemini-2.5-flash')).not.toBeInTheDocument();
  });

  it('[AC #4] a DEARER model states its premium instead of hiding it', () => {
    const opus: ModelChoice = {
      id: 'claude-opus-4-8',
      displayName: 'Claude Opus 4.8',
      isDefault: false,
      isBestGrade: false,
      totalUsd: 2.4,
      deltaUsd: -1.87,
      deltaPercent: 353,
    };
    renderPicker(opus.id, [SONNET, opus]);
    expect(screen.getByTestId('consent-model-note-claude-opus-4-8')).toHaveTextContent(
      '比 Claude Sonnet 5 多 $1.87（353%）'
    );
  });

  it('[AC #4] 「品質最穩」 is claimed only by a default row that actually holds the top grade', () => {
    // A CLAUDE_MODEL override can make the B-grade model the default; the copy
    // must not follow it into a false claim.
    const haikuAsDefault: ModelChoice = { ...HAIKU, isDefault: true, deltaUsd: undefined };
    renderPicker(haikuAsDefault.id, [{ ...SONNET, isDefault: false }, haikuAsDefault]);
    expect(screen.queryByTestId('consent-model-note-claude-haiku-4-5')).not.toBeInTheDocument();
  });

  it('[P0 AC #5] the group is a labelled radiogroup and selecting reports the id', () => {
    const { onSelect } = renderPicker();

    expect(screen.getByRole('radiogroup', { name: '翻譯模型' })).toBeInTheDocument();
    expect(screen.getAllByRole('radio')).toHaveLength(3);

    fireEvent.click(screen.getByLabelText(/Claude Haiku 4.5/));
    expect(onSelect).toHaveBeenCalledWith('claude-haiku-4-5');
  });

  it('renders nothing when there is no choice to make', () => {
    render(<ModelPicker choices={[]} selectedModelId="" onSelect={vi.fn()} />);
    expect(screen.queryByTestId('consent-model-picker')).not.toBeInTheDocument();
  });

  it('a start already in flight locks the quote', () => {
    const onSelect = vi.fn();
    render(
      <ModelPicker
        choices={[SONNET, HAIKU]}
        selectedModelId={SONNET.id}
        onSelect={onSelect}
        disabled
      />
    );
    expect(screen.getByLabelText(/Claude Haiku 4.5/)).toBeDisabled();
  });
});

// ─── sub-7-8c: 「試跑 20 句」 (J10 four states) ───────────────────────────────

const OPUS: ModelChoice = {
  id: 'claude-opus-4-8',
  displayName: 'Claude Opus 4.8',
  isDefault: false,
  isBestGrade: false,
  totalUsd: 7.5,
  minutes: 12,
  previewEstimateUsd: 0.06,
};

describe('ModelPicker 試跑 20 句 (sub-7-8c AC #4)', () => {
  function renderWithPreview(
    states: Parameters<typeof ModelPicker>[0]['previewStates'] = {},
    choices = [SONNET, OPUS],
    disabled = false
  ) {
    const onSelect = vi.fn();
    const onPreview = vi.fn();
    render(
      <ModelPicker
        choices={choices}
        selectedModelId={SONNET.id}
        onSelect={onSelect}
        previewStates={states}
        onPreview={onPreview}
        disabled={disabled}
      />
    );
    return { onSelect, onPreview };
  }

  it('① idle: the ungraded row carries a small cost button with the backend estimate, graded rows do not', () => {
    const { onSelect, onPreview } = renderWithPreview();
    const btn = screen.getByTestId('consent-model-preview-claude-opus-4-8');
    expect(btn).toHaveTextContent('試跑 20 句');
    expect(screen.getByTestId('consent-model-preview-claude-opus-4-8-amount').textContent).toBe(
      '$0.06'
    );
    expect(btn.getAttribute('data-cost-status')).toBe('ready');
    expect(btn.className).toContain('h-6');
    expect(screen.queryByTestId('consent-model-preview-claude-sonnet-5')).not.toBeInTheDocument();

    // Clicking the button must NOT tick the radio it sits beside.
    fireEvent.click(btn);
    expect(onPreview).toHaveBeenCalledWith('claude-opus-4-8');
    expect(onSelect).not.toHaveBeenCalled();
  });

  it('① no estimate → the button is unavailable, and never shows $0.00 (J9 ⑤)', () => {
    const { onPreview } = renderWithPreview({}, [
      SONNET,
      { ...OPUS, previewEstimateUsd: undefined },
    ]);
    const btn = screen.getByTestId('consent-model-preview-claude-opus-4-8');
    expect(btn.getAttribute('data-cost-status')).toBe('unavailable');
    expect(btn).toBeDisabled();
    expect(btn.textContent).not.toContain('$0.00');
    fireEvent.click(btn);
    expect(onPreview).not.toHaveBeenCalled();
  });

  it('② running: busy button with a skeleton amount and the 「約 1 分鐘」 line', () => {
    const { onPreview } = renderWithPreview({ 'claude-opus-4-8': { status: 'running' } });
    const btn = screen.getByTestId('consent-model-preview-claude-opus-4-8');
    expect(btn).toHaveAttribute('aria-busy', 'true');
    expect(
      screen.getByTestId('consent-model-preview-claude-opus-4-8-amount-skeleton')
    ).toBeInTheDocument();
    expect(screen.getByTestId('consent-model-preview-running-claude-opus-4-8')).toHaveTextContent(
      '約 1 分鐘，請勿關閉視窗'
    );
    fireEvent.click(btn);
    expect(onPreview).not.toHaveBeenCalled();
  });

  it('③ done: one 「你的實測」 line — no letter, no badge colour — plus 再試一次', () => {
    const result = {
      modelId: 'claude-opus-4-8',
      cues: 20,
      zeroRate: 0.05,
      naturalRate: 0.7,
      costUsd: 0.05,
      judgeModel: 'claude-sonnet-5',
      gradedAt: '2026-10-01T12:00:00Z',
    };
    const { onPreview } = renderWithPreview({ 'claude-opus-4-8': { status: 'done', result } });
    expect(screen.getByTestId('consent-model-local-grade-claude-opus-4-8').textContent).toBe(
      '你的實測：0 分 5%・2 分 70%・花 $0.05'
    );
    // The badge stays 尚未評測: this is the user's 20 cues, not Vido's 200.
    expect(screen.getByTestId('consent-model-grade-claude-opus-4-8')).toHaveTextContent('尚未評測');
    expect(screen.getByTestId('consent-model-option-claude-opus-4-8').textContent).not.toMatch(
      /品質 [ABC]/
    );
    expect(screen.queryByTestId('consent-model-preview-claude-opus-4-8')).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId('consent-model-preview-again-claude-opus-4-8'));
    expect(onPreview).toHaveBeenCalledWith('claude-opus-4-8');
  });

  it('③ a result the SERVER stored (local_grade on the catalog row) renders the same line', () => {
    renderWithPreview({}, [
      SONNET,
      {
        ...OPUS,
        localGrade: {
          modelId: 'claude-opus-4-8',
          cues: 20,
          zeroRate: 0.1,
          naturalRate: 0.6,
          costUsd: 0.04,
          judgeModel: 'claude-sonnet-5',
          gradedAt: '2026-10-01T12:00:00Z',
          incomplete: 'translate: budget',
        },
      },
    ]);
    expect(screen.getByTestId('consent-model-local-grade-claude-opus-4-8').textContent).toBe(
      '到預算上限才停：0 分 10%・2 分 60%・花 $0.04'
    );
  });

  it('④ failed: the button comes back with the reason beside it', () => {
    renderWithPreview({
      'claude-opus-4-8': {
        status: 'failed',
        message: '剛剛試跑過，稍後再試',
        unpaid: true,
        code: 'AI_PREVIEW_TOO_SOON',
      },
    });
    expect(screen.getByTestId('consent-model-preview-claude-opus-4-8')).toBeInTheDocument();
    expect(screen.getByTestId('consent-model-preview-error-claude-opus-4-8')).toHaveTextContent(
      '剛剛試跑過，稍後再試'
    );
  });

  it('a locked picker (start in flight) also locks the try-out', () => {
    const { onPreview } = renderWithPreview({}, [SONNET, OPUS], true);
    fireEvent.click(screen.getByTestId('consent-model-preview-claude-opus-4-8'));
    expect(onPreview).not.toHaveBeenCalled();
  });
});
