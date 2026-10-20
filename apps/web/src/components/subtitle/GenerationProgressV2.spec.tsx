import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import {
  GenerationProgressV2,
  GENERATION_STAGES,
  GENERATION_STAGES_EXTRACT,
} from './GenerationProgressV2';

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

    it('each stage is a full-width row [circle + Body label + spacer + Mono pct] at <sm, 72px column at sm+', () => {
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

      // dsr-6f-1: Body 14 on a phone (F3-M IdGB2), Label 12 / 1.5 on desktop —
      // the off-scale 13px is gone.
      const label = screen.getByText('翻譯中');
      expect(label.className.split(' ')).toEqual(
        expect.arrayContaining(['text-sm', 'sm:text-xs', 'sm:leading-normal'])
      );
      expect(label.className.split(' ')).not.toContain('text-[13px]');
      // F3-M rows carry 4px of padding; phone only.
      expect(stage.className.split(' ')).toContain('max-sm:p-1');

      // Mono pct right-aligned via the ml-auto spacer on mobile only.
      const pct = screen.getByText('63%');
      expect(pct.className).toContain('ml-auto');
      // [guard] 11px is FROZEN (disc-2026-09-11px-micro-label-not-on-type-scale).
      expect(pct.className.split(' ')).toContain('text-xs');
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

  // disc-2026-10-single-generate-ignores-embedded-english-b — J12-D (jYNkJ), the
  // five-stage variant for runs that read a subtitle track instead of listening.
  describe('embedded-track lane (route)', () => {
    it('renders the FIVE extract stages and none of 提取音訊 / 轉錄中', () => {
      render(<GenerationProgressV2 phase="extracting" route="extract" />);

      expect(GENERATION_STAGES_EXTRACT).toEqual(['抽取字幕', '翻譯中', '簡轉繁', 'AI校正', '完成']);
      for (const stage of GENERATION_STAGES_EXTRACT) {
        expect(screen.getByTestId(`gen-stage-${stage}`)).toBeInTheDocument();
      }
      expect(screen.queryByTestId('gen-stage-提取音訊')).toBeNull();
      expect(screen.queryByTestId('gen-stage-轉錄中')).toBeNull();
      expect(screen.getByTestId('gen-stage-抽取字幕')).toHaveAttribute('data-state', 'active');
    });

    it.each(['extract', 'translate', 'deliver_direct', 'convert_then_deliver'])(
      'route %s selects the five-stage variant',
      (route) => {
        render(<GenerationProgressV2 phase="translating" route={route} percentage={62} />);
        expect(screen.getAllByRole('listitem')).toHaveLength(5);
        expect(screen.getByTestId('gen-stage-翻譯中')).toHaveAttribute('data-state', 'active');
        expect(screen.getByTestId('gen-stage-抽取字幕')).toHaveAttribute('data-state', 'done');
      }
    );

    it.each(['asr', 'skip', undefined, null])('route %s keeps the six-stage stepper', (route) => {
      render(
        <GenerationProgressV2 phase="transcribing" route={route as string | null | undefined} />
      );
      expect(screen.getAllByRole('listitem')).toHaveLength(6);
      expect(screen.getByTestId('gen-stage-轉錄中')).toHaveAttribute('data-state', 'active');
    });

    it('a delivered Chinese track completes with 翻譯中 SKIPPED (minus), everything else done', () => {
      render(<GenerationProgressV2 phase="complete" route="deliver_direct" />);

      expect(screen.getByTestId('gen-stage-翻譯中')).toHaveAttribute('data-state', 'skipped');
      for (const stage of ['抽取字幕', '簡轉繁', 'AI校正', '完成']) {
        expect(screen.getByTestId(`gen-stage-${stage}`)).toHaveAttribute('data-state', 'done');
      }
      // Skipped is visually distinct from pending (dot) and done (check): a minus glyph.
      const mark = screen.getByTestId('gen-stage-翻譯中').querySelector('svg.lucide-minus');
      expect(mark).not.toBeNull();
    });

    it('a converted Simplified track also skips 翻譯中; a translated English track does not', () => {
      const { unmount } = render(
        <GenerationProgressV2 phase="complete" route="convert_then_deliver" />
      );
      expect(screen.getByTestId('gen-stage-翻譯中')).toHaveAttribute('data-state', 'skipped');
      unmount();
      render(<GenerationProgressV2 phase="complete" route="translate" />);
      expect(screen.getByTestId('gen-stage-翻譯中')).toHaveAttribute('data-state', 'done');
    });

    it('a predicted extract that actually transcribes shows the six stages (the route was wrong)', () => {
      render(<GenerationProgressV2 phase="transcribing" route="extract" />);
      expect(screen.getAllByRole('listitem')).toHaveLength(6);
      expect(screen.getByTestId('gen-stage-轉錄中')).toHaveAttribute('data-state', 'active');
    });

    it('names a failed first step 抽取字幕失敗; a failed translation stays 翻譯失敗', () => {
      const { unmount } = render(
        <GenerationProgressV2 phase="failed" failedPhase="extracting" route="extract" />
      );
      expect(screen.getByTestId('gen-failed-panel').querySelector('p')?.textContent).toBe(
        '抽取字幕失敗'
      );
      expect(screen.getByTestId('gen-stage-抽取字幕')).toHaveAttribute('data-state', 'failed');
      unmount();
      render(<GenerationProgressV2 phase="failed" failedPhase="translating" route="extract" />);
      expect(screen.getByTestId('gen-failed-panel').querySelector('p')?.textContent).toBe(
        '翻譯失敗'
      );
    });
  });

  // disc-2026-10-gen-live-column-squeeze (F11-SPEC-WIDTH V2oywc): the workspace
  // queue row picks horizontal / vertical from ITS OWN width (616 = 562 stepper +
  // 54 indent), not the viewport. jsdom has no layout — these pin the class
  // strings; the real widths are measured in a browser (story AC #3).
  describe('widthSource — viewport (dialogs, default) vs container (workspace queue row)', () => {
    const classesOf = () => {
      const ol = screen.getByRole('list', { name: '字幕生成進度' });
      const connector = Array.from(ol.querySelectorAll('span[aria-hidden="true"]')).find((el) =>
        el.className.includes('h-0.5')
      )!;
      return {
        ol: ol.className,
        li: ol.querySelector('li')!.className,
        connector: connector.className,
        stage: screen.getByTestId('gen-stage-翻譯中').className,
        label: screen.getByText('翻譯中').className,
        pct: screen.getByText('45%').className,
      };
    };

    it('[P0] no widthSource: every class string is VERBATIM what the dialogs draw today (F3/F4/F8 unchanged)', () => {
      render(<GenerationProgressV2 phase="translating" percentage={45} />);
      expect(classesOf()).toEqual({
        ol: 'flex flex-col gap-1.5 sm:flex-row sm:items-start sm:gap-0 sm:justify-center',
        li: 'flex w-full items-start sm:w-auto',
        connector: 'hidden sm:block mt-[10px] h-0.5 sm:w-[26px] bg-[var(--success)]',
        stage:
          'flex w-full flex-row items-center gap-2.5 max-sm:p-1 sm:w-[72px] sm:flex-col sm:gap-1.5',
        label: 'text-sm sm:text-xs sm:leading-normal font-semibold text-[var(--accent-text)]',
        pct: 'ml-auto font-mono text-xs tabular-nums text-[var(--accent-text)] sm:ml-0',
      });
    });

    it('[P0] widthSource="viewport" is the same as omitting it', () => {
      const { unmount } = render(<GenerationProgressV2 phase="translating" percentage={45} />);
      const omitted = classesOf();
      unmount();
      render(<GenerationProgressV2 phase="translating" percentage={45} widthSource="viewport" />);
      expect(classesOf()).toEqual(omitted);
    });

    it('[P0] widthSource="container": horizontal at a ≥616 container, the phone stack below it — no viewport class left', () => {
      render(
        <GenerationProgressV2
          phase="translating"
          percentage={45}
          align="start"
          widthSource="container"
        />
      );
      const c = classesOf();
      expect(c).toEqual({
        ol: 'flex flex-col gap-1.5 @min-[616px]:flex-row @min-[616px]:items-start @min-[616px]:gap-0 @min-[616px]:justify-start',
        li: 'flex w-full items-start @min-[616px]:w-auto',
        connector:
          'hidden @min-[616px]:block mt-[10px] h-0.5 @min-[616px]:w-[26px] bg-[var(--success)]',
        stage:
          'flex w-full flex-row items-center gap-2.5 @max-[616px]:p-1 @min-[616px]:w-[72px] @min-[616px]:flex-col @min-[616px]:gap-1.5',
        label:
          'text-sm @min-[616px]:text-xs @min-[616px]:leading-normal font-semibold text-[var(--accent-text)]',
        pct: 'ml-auto font-mono text-xs tabular-nums text-[var(--accent-text)] @min-[616px]:ml-0',
      });
      // A viewport breakpoint left behind would flip the row at 640 again.
      for (const cls of Object.values(c)) expect(cls).not.toMatch(/(^|\s)(max-)?sm:/);
    });

    it('[P1] the five-stage extract lane uses the same switch point', () => {
      render(
        <GenerationProgressV2
          phase="translating"
          percentage={45}
          route="extract"
          widthSource="container"
        />
      );
      expect(screen.getByTestId('gen-stage-抽取字幕').className).toContain('@min-[616px]:w-[72px]');
      expect(screen.getByTestId('gen-stage-翻譯中').className).toContain('@max-[616px]:p-1');
    });

    it('[P1] only the 翻譯中 step carries a percentage, in both layouts', () => {
      render(<GenerationProgressV2 phase="translating" percentage={45} widthSource="container" />);
      expect(screen.getAllByText('45%')).toHaveLength(1);
      expect(screen.getByTestId('gen-stage-翻譯中')).toHaveTextContent('45%');
    });
  });
});
