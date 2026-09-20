import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { AnalysisProgressPanel } from './AnalysisProgressPanel';

/**
 * F14 產生字幕．分析中 (dsr-6e-2). Two of these guard people who cannot see the
 * screen: the counter used to be an aria-live region fed by a 250ms SSE tick,
 * and the cancel button used `disabled`, which drops focus to <body>.
 */
describe('AnalysisProgressPanel (F14)', () => {
  it('the counter is NOT a live region — the progressbar carries the numbers', () => {
    render(<AnalysisProgressPanel analyzed={234} total={1247} onCancel={vi.fn()} />);
    const counter = screen.getByTestId('consent-analysis-counter');
    expect(counter).toHaveTextContent('234 / 1,247');
    expect(counter.closest('[aria-live]')).toBeNull();
    const bar = screen.getByRole('progressbar', { name: '字幕軌分析進度' });
    expect(bar).toHaveAttribute('aria-valuenow', '234');
    expect(bar).toHaveAttribute('aria-valuemax', '1247');
  });

  it('取消 lives in a footer, outside the content block', () => {
    render(<AnalysisProgressPanel analyzed={1} total={2} onCancel={vi.fn()} />);
    const footer = screen.getByTestId('consent-analysis-footer');
    const cancel = screen.getByTestId('consent-analysis-cancel');
    expect(footer).toContainElement(cancel);
    expect(screen.getByTestId('consent-analysis-panel')).not.toContainElement(cancel);
    expect(footer.className).toContain('border-t');
    expect(footer.className).toContain('justify-end');
  });

  it('cancelling: aria-disabled (never `disabled`), says 取消中…, dims, and ignores a second click', () => {
    const onCancel = vi.fn();
    render(<AnalysisProgressPanel analyzed={1} total={2} cancelling onCancel={onCancel} />);
    const cancel = screen.getByTestId('consent-analysis-cancel');
    expect(cancel).toHaveAttribute('aria-disabled', 'true');
    expect(cancel).not.toBeDisabled();
    expect(cancel).toHaveTextContent('取消中…');
    // Whole TOKENS: `aria-disabled:opacity-50` contains the substring
    // `disabled:opacity-50`, so a substring check is either vacuous or
    // order-dependent (CR M1).
    const tokens = cancel.className.split(/\s+/);
    expect(tokens).toContain('aria-disabled:opacity-50');
    expect(tokens).toContain('aria-disabled:cursor-not-allowed');
    expect(tokens.some((t) => t.startsWith('disabled:'))).toBe(false);
    fireEvent.click(cancel);
    expect(onCancel).not.toHaveBeenCalled();
  });

  it('idle: 取消 fires onCancel', () => {
    const onCancel = vi.fn();
    render(<AnalysisProgressPanel analyzed={1} total={2} onCancel={onCancel} />);
    const cancel = screen.getByTestId('consent-analysis-cancel');
    expect(cancel).toHaveTextContent(/^取消$/);
    fireEvent.click(cancel);
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it('track and fill are pills; label is Body 600; no 13px', () => {
    const { container } = render(
      <AnalysisProgressPanel analyzed={1} total={2} onCancel={vi.fn()} />
    );
    const bar = screen.getByRole('progressbar');
    expect(bar.className).toContain('rounded-full');
    expect(screen.getByTestId('consent-analysis-bar').className).toContain('rounded-full');
    expect(screen.getByTestId('consent-analysis-counter').parentElement?.className).toContain(
      'font-semibold'
    );
    expect(container.innerHTML).not.toContain('text-[13px]');
    expect(screen.getByText(/這個步驟在本機執行/).className).toContain('text-sm');
    // CR L2: the content block can shrink and scroll, so a short viewport never
    // pushes the 取消 footer out of the 85vh dialog.
    const panel = screen.getByTestId('consent-analysis-panel');
    expect(panel.className).toContain('min-h-0');
    expect(panel.className).toContain('overflow-y-auto');
    expect(panel.firstElementChild?.className).toContain('max-w-[480px]');
  });
});
