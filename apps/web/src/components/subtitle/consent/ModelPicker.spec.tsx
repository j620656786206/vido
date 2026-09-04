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

    expect(screen.getByTestId('consent-model-usd-claude-sonnet-5')).toHaveTextContent(
      '這批約 $0.53'
    );
    expect(screen.getByTestId('consent-model-grade-claude-sonnet-5')).toHaveTextContent('品質 A');
    expect(screen.getByTestId('consent-model-minutes-claude-sonnet-5')).toHaveTextContent(
      '約 11 分鐘'
    );
    expect(screen.getByTestId('consent-model-usd-claude-haiku-4-5')).toHaveTextContent(
      '這批約 $0.21'
    );
  });

  it('[P0 AC #1] an unevaluated model says 尚未評測 and offers the cheap try-out — never a blank where a grade goes', () => {
    renderPicker();

    const badge = screen.getByTestId('consent-model-grade-gemini-2.5-flash');
    expect(badge).toHaveTextContent('尚未評測');
    expect(badge.textContent).not.toContain('品質');
    expect(screen.getByTestId('consent-model-option-gemini-2.5-flash').textContent).toContain(
      '可花約 $0.01 試跑 20 句'
    );
    // No measurement, no duration claim.
    expect(screen.queryByTestId('consent-model-minutes-gemini-2.5-flash')).not.toBeInTheDocument();
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
