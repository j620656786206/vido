import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
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

  // dsr-6b AC #6 — the failed panel names the stage the way the stepper does, and
  // 重試 lives in the dialog footer now (F4-D-v2 dg5rH), not in this panel.
  it('names the failed stage with the stepper vocabulary and has no retry of its own', () => {
    render(<GenerationProgressV2 phase="failed" failedPhase="translating" />);

    expect(screen.getByTestId('gen-stage-翻譯中')).toHaveAttribute('data-state', 'failed');
    const panel = screen.getByTestId('gen-failed-panel');
    expect(panel.querySelector('p')?.textContent).toBe('翻譯失敗');
    expect(screen.queryByTestId('gen-retry')).toBeNull();
  });

  it.each([
    ['extracting', '提取音訊失敗'],
    ['transcribing', '轉錄失敗'],
    ['translating', '翻譯失敗'],
  ] as const)('failed at %s → %s', (failedPhase, text) => {
    render(<GenerationProgressV2 phase="failed" failedPhase={failedPhase} />);
    expect(screen.getByTestId('gen-failed-panel').querySelector('p')?.textContent).toBe(text);
  });

  it('shows a machine error verbatim on its own Mono line', () => {
    render(
      <GenerationProgressV2
        phase="failed"
        failedPhase="translating"
        error="translate: context deadline exceeded"
      />
    );
    const detail = screen.getByTestId('gen-failed-detail');
    expect(detail.textContent).toBe('translate: context deadline exceeded');
    expect(detail.className).toContain('font-mono');
    expect(detail.className).toContain('break-all');
  });

  // /ship CR L1 — a machine error that merely CONTAINS a CJK file path is still
  // machine text; only a sentence that starts in CJK is written for people.
  it('keeps a machine error Mono even when it carries a Chinese file path', () => {
    render(
      <GenerationProgressV2
        phase="failed"
        failedPhase="extracting"
        error="ffprobe timeout: /media/電影/沙丘.2160p.WEB-DL.mkv"
      />
    );
    expect(screen.getByTestId('gen-failed-detail').className).toContain('font-mono');
  });

  it('shows a Chinese error in the normal typeface, not Mono', () => {
    render(
      <GenerationProgressV2 phase="failed" failedPhase="translating" error="字幕生成失敗：逾時" />
    );
    const detail = screen.getByTestId('gen-failed-detail');
    expect(detail.textContent).toBe('字幕生成失敗：逾時');
    expect(detail.className).not.toContain('font-mono');
  });

  it.each([[''], ['生成失敗']])('hides the detail line when the error is %j', (error) => {
    render(<GenerationProgressV2 phase="failed" failedPhase="extracting" error={error} />);
    expect(screen.queryByTestId('gen-failed-detail')).toBeNull();
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

  describe('mobile vertical stepper variant (F3-M-v2 k8sJl4 — Sally gate MUST-FIX)', () => {
    it('the list stacks vertically at <sm and restores the horizontal stepper at sm+', () => {
      render(<GenerationProgressV2 phase="translating" percentage={62.5} />);

      const ol = screen.getByRole('list', { name: '字幕生成進度' });
      // Mobile-first: vertical stack, gap-1.5 rows.
      expect(ol.className).toContain('flex-col');
      expect(ol.className).toContain('gap-1.5');
      // Desktop restores the EXACT pre-fix layout (zero baseline diff).
      expect(ol.className).toContain('sm:flex-row');
      expect(ol.className).toContain('sm:items-start');
      expect(ol.className).toContain('sm:justify-center');
      expect(ol.className).toContain('sm:gap-0');
    });

    it('each stage is a full-width row [circle + 13px label + spacer + Mono pct] at <sm, 72px column at sm+', () => {
      render(<GenerationProgressV2 phase="translating" percentage={62.5} />);

      const stage = screen.getByTestId('gen-stage-翻譯中');
      expect(stage.className).toContain('w-full');
      expect(stage.className).toContain('flex-row');
      expect(stage.className).toContain('items-center');
      expect(stage.className).toContain('gap-2.5');
      expect(stage.className).toContain('sm:w-[72px]');
      expect(stage.className).toContain('sm:flex-col');
      // dsr-6b: XkGvG column gap 6 — exact token, `toContain('sm:gap-1')` would
      // also match sm:gap-1.5 and prove nothing.
      expect(stage.className.split(' ')).toContain('sm:gap-1.5');
      expect(stage.className.split(' ')).not.toContain('sm:gap-1');

      // Label 13px on mobile (dsr-6f owns it), Label 12 / 1.5 on desktop.
      const label = screen.getByText('翻譯中');
      expect(label.className.split(' ')).toEqual(
        expect.arrayContaining(['text-[13px]', 'sm:text-xs', 'sm:leading-normal'])
      );

      // Mono pct right-aligned via the ml-auto spacer on mobile only.
      const pct = screen.getByText('63%');
      expect(pct.className).toContain('ml-auto');
      expect(pct.className).toContain('sm:ml-0');
    });

    it('connectors are hidden on mobile and visible at sm+ (rows need no connecting line)', () => {
      render(<GenerationProgressV2 phase="transcribing" />);

      const ol = screen.getByRole('list', { name: '字幕生成進度' });
      const connectors = Array.from(ol.querySelectorAll('span[aria-hidden="true"]')).filter((el) =>
        el.className.includes('h-0.5')
      );
      expect(connectors).toHaveLength(GENERATION_STAGES.length - 1); // 5 connectors between 6 stages
      for (const connector of connectors) {
        expect(connector.className).toContain('hidden');
        expect(connector.className).toContain('sm:block');
        // dsr-6b: XkGvG connector 26×2; the mobile-only w-5 was dead code.
        expect(connector.className.split(' ')).toContain('sm:w-[26px]');
        expect(connector.className.split(' ')).not.toContain('w-5');
      }
    });

    it('desktop DOM structure is unchanged: 6 li rows, stage testids + states intact', () => {
      render(<GenerationProgressV2 phase="transcribing" />);

      const ol = screen.getByRole('list', { name: '字幕生成進度' });
      expect(ol.querySelectorAll('li')).toHaveLength(GENERATION_STAGES.length);
      expect(screen.getByTestId('gen-stage-提取音訊')).toHaveAttribute('data-state', 'done');
      expect(screen.getByTestId('gen-stage-轉錄中')).toHaveAttribute('data-state', 'active');
      expect(screen.getByTestId('gen-stage-完成')).toHaveAttribute('data-state', 'pending');
    });
  });
});
