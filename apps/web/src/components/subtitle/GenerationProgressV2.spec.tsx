import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { GenerationProgressV2, GENERATION_STAGES } from './GenerationProgressV2';

describe('GenerationProgressV2', () => {
  it('renders ALL six frozen stage names (fixture vocabulary — do not rename)', () => {
    render(<GenerationProgressV2 phase="extracting" />);

    expect(GENERATION_STAGES).toEqual(['提取音訊', '轉錄中', '翻譯中', '簡轉繁', 'AI校正', '完成']);
    for (const stage of GENERATION_STAGES) {
      expect(screen.getByTestId(`gen-stage-${stage}`)).toBeInTheDocument();
      expect(screen.getByText(stage)).toBeInTheDocument();
    }
  });

  it('marks the wire phase mapping: transcribing → 轉錄中 active, 提取音訊 done', () => {
    render(<GenerationProgressV2 phase="transcribing" />);

    expect(screen.getByTestId('gen-stage-提取音訊')).toHaveAttribute('data-state', 'done');
    expect(screen.getByTestId('gen-stage-轉錄中')).toHaveAttribute('data-state', 'active');
    expect(screen.getByTestId('gen-stage-翻譯中')).toHaveAttribute('data-state', 'pending');
    expect(screen.getByTestId('gen-stage-簡轉繁')).toHaveAttribute('data-state', 'pending');
  });

  it('shows the translating percentage in Mono under the active stage', () => {
    render(<GenerationProgressV2 phase="translating" percentage={62.5} />);

    expect(screen.getByTestId('gen-stage-翻譯中')).toHaveAttribute('data-state', 'active');
    const pct = screen.getByText('63%');
    expect(pct).toBeInTheDocument();
    expect(pct.className).toContain('font-mono');
    expect(pct.className).toContain('tabular-nums');
  });

  it('advances 簡轉繁/AI校正/完成 ATOMICALLY on complete (no dedicated wire phase)', () => {
    render(<GenerationProgressV2 phase="complete" />);

    for (const stage of GENERATION_STAGES) {
      expect(screen.getByTestId(`gen-stage-${stage}`)).toHaveAttribute('data-state', 'done');
    }
  });

  it('renders 失敗於{stage} + server error + 重試 in the failed state', () => {
    const onRetry = vi.fn();
    render(
      <GenerationProgressV2
        phase="failed"
        failedPhase="translating"
        error="AI 服務逾時"
        onRetry={onRetry}
      />
    );

    expect(screen.getByTestId('gen-stage-翻譯中')).toHaveAttribute('data-state', 'failed');
    expect(screen.getByTestId('gen-failed-panel')).toHaveTextContent('失敗於翻譯中：AI 服務逾時');

    fireEvent.click(screen.getByTestId('gen-retry'));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('renders the server-supplied message verbatim (Rule 23 — no local clock text)', () => {
    render(
      <GenerationProgressV2
        phase="transcribing"
        message="正在轉錄音訊（Whisper large-v3）— 12:34 / 45:10"
      />
    );

    expect(screen.getByTestId('gen-stage-message')).toHaveTextContent(
      '正在轉錄音訊（Whisper large-v3）— 12:34 / 45:10'
    );
  });

  it('renders NO cost line when cost props are absent (9R-17 dormant slot)', () => {
    render(<GenerationProgressV2 phase="transcribing" />);
    expect(screen.queryByTestId('gen-cost-line')).not.toBeInTheDocument();
  });

  it('renders the cost line with Mono numerics when BOTH cost props are provided', () => {
    render(
      <GenerationProgressV2 phase="transcribing" costUsedText="$0.42" costLimitText="$5.00" />
    );

    const line = screen.getByTestId('gen-cost-line');
    expect(line).toHaveTextContent('本次用量：$0.42 / 上限 $5.00');
    expect(screen.getByText('$0.42').className).toContain('font-mono');
  });
});
