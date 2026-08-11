import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { CandidateListPanel, type CandidateListPanelProps } from './CandidateListPanel';
import { computeTotals } from './consentSelection';
import type { GenerationCandidate } from '../../../services/subtitleService';

const A = '0a54a9e2-3a67-4f3e-9f8e-a1c2d3e4f501';
const B = '1b65baf3-4b78-4a4f-8a9f-b2d3e4f5a602';
const EP = '8fa9fed7-8fbc-4e8d-8edc-f6b7c8d9e006';
const U = '2c76cba4-5c89-4b5a-9baf-c3e4f5a6b703';

const CANDIDATES: GenerationCandidate[] = [
  {
    mediaId: A,
    mediaType: 'movie',
    title: '沙丘：第二部',
    route: 'extract',
    runtimeMinutes: 166,
    runtimeKnown: true,
    estimatedUsd: 0.05,
  },
  {
    mediaId: B,
    mediaType: 'movie',
    title: '奧本海默',
    route: 'extract',
    runtimeMinutes: 180,
    runtimeKnown: true,
    estimatedUsd: 0.04,
  },
  {
    mediaId: EP,
    mediaType: 'episode',
    title: '怪奇物語 S04E07',
    route: 'asr',
    runtimeMinutes: 52,
    runtimeKnown: true,
    estimatedUsd: 0.31,
  },
  {
    mediaId: U,
    mediaType: 'movie',
    title: '未知片長的電影',
    route: 'asr',
    runtimeMinutes: 45,
    runtimeKnown: false,
    estimatedUsd: 0.27,
  },
];

function renderPanel(overrides?: Partial<CandidateListPanelProps>) {
  const selectedIds = overrides?.selectedIds ?? new Set([A, B]);
  const budgetText = overrides?.budgetText ?? '5.00';
  const budgetUsd = overrides?.budgetUsd !== undefined ? overrides.budgetUsd : 5;
  const props: CandidateListPanelProps = {
    candidates: CANDIDATES,
    selectedIds,
    filter: 'all',
    totals: computeTotals(CANDIDATES, selectedIds, budgetUsd),
    budgetText,
    budgetUsd,
    onToggle: vi.fn(),
    onToggleAll: vi.fn(),
    onSelectAllExtract: vi.fn(),
    onClearSelection: vi.fn(),
    onFilterChange: vi.fn(),
    onBudgetTextChange: vi.fn(),
    onStartClick: vi.fn(),
    ...overrides,
  };
  return { props, ...render(<CandidateListPanel {...props} />) };
}

describe('CandidateListPanel (F15/F18)', () => {
  it('[P0 §5-sexies] renders estimated_usd VERBATIM — the word 免費 never appears', () => {
    renderPanel();
    expect(screen.getByTestId(`consent-row-usd-${A}`).textContent).toBe('$0.05');
    expect(screen.getByTestId(`consent-row-usd-${B}`).textContent).toBe('$0.04');
    expect(screen.queryByText(/免費/)).toBeNull();
    // The extract chip carries the 僅翻譯費 marker instead.
    expect(screen.getByTestId('consent-chip-extract').textContent).toContain('僅翻譯費');
  });

  it('[P0] unknown-runtime rows prefix ≈ and explain the 45-minute fallback', () => {
    renderPanel();
    expect(screen.getByTestId(`consent-row-usd-${U}`).textContent).toBe('≈ $0.27');
    expect(screen.getByText('片長未知，以 45 分鐘估算')).toBeInTheDocument();
  });

  it('[P0 三處同源] summary bar, footer and detail line show the SAME total', () => {
    renderPanel();
    expect(screen.getByTestId('consent-summary-usd').textContent).toBe('$0.09');
    expect(screen.getByTestId('consent-footer-usd').textContent).toBe('$0.09');
    expect(screen.getByTestId('consent-footer-detail').textContent).toContain('$0.09');
    expect(screen.getByTestId('consent-footer-detail').textContent).toContain('$0.00');
  });

  it('[P0 F18] over-budget: banner with feasible count, warning amounts, button relabels and stays ENABLED', () => {
    renderPanel({ selectedIds: new Set([A, B, EP, U]), budgetText: '0.30', budgetUsd: 0.3 });
    const banner = screen.getByTestId('consent-over-budget-banner');
    expect(banner.textContent).toContain('已超過上限');
    expect(banner.textContent).toContain('$0.30');
    expect(screen.getByTestId('consent-feasible-count').textContent).toBe('3');
    const btn = screen.getByTestId('consent-start-btn');
    expect(btn.textContent).toContain('開始產生（將於上限暫停）');
    expect(btn).not.toBeDisabled();
  });

  it('[P0] invalid budget (<=0) disables start and shows the >0 hint — never "unlimited"', () => {
    renderPanel({ budgetText: '0', budgetUsd: null });
    expect(screen.getByTestId('consent-start-btn')).toBeDisabled();
    expect(screen.getByText('上限必須大於 0')).toBeInTheDocument();
    expect(screen.getByTestId('consent-budget-input')).toHaveAttribute('aria-invalid', 'true');
  });

  it('empty selection disables start', () => {
    renderPanel({ selectedIds: new Set() });
    expect(screen.getByTestId('consent-start-btn')).toBeDisabled();
  });

  it('select-all checkbox is indeterminate on a partial selection', () => {
    renderPanel();
    const box = screen.getByTestId('consent-select-all') as HTMLInputElement;
    expect(box.indeterminate).toBe(true);
    expect(box.checked).toBe(false);
  });

  it('route filter narrows the visible rows only — totals untouched', () => {
    renderPanel({ filter: 'asr' });
    expect(screen.queryByTestId(`consent-row-${A}`)).toBeNull();
    expect(screen.getByTestId(`consent-row-${EP}`)).toBeInTheDocument();
    expect(screen.getByTestId('consent-summary-usd').textContent).toBe('$0.09');
  });

  it('episode rows render the backend-composed SxxEyy title as-is', () => {
    renderPanel();
    expect(screen.getByText('怪奇物語 S04E07')).toBeInTheDocument();
  });

  it('toolbar actions dispatch', () => {
    const { props } = renderPanel();
    fireEvent.click(screen.getByTestId('consent-select-extract'));
    fireEvent.click(screen.getByTestId('consent-clear-selection'));
    fireEvent.click(screen.getByTestId('consent-select-all'));
    expect(props.onSelectAllExtract).toHaveBeenCalledOnce();
    expect(props.onClearSelection).toHaveBeenCalledOnce();
    expect(props.onToggleAll).toHaveBeenCalledOnce();
  });
});

describe('CandidateListPanel F18 footer hint (third design round)', () => {
  it('over-budget hides the small auto-pause hint — the banner already says it', () => {
    renderPanel({ selectedIds: new Set([A, B, EP, U]), budgetText: '0.30', budgetUsd: 0.3 });
    expect(screen.queryByText('達到上限會自動暫停，可稍後續跑')).toBeNull();
    expect(screen.getByTestId('consent-over-budget-banner')).toBeInTheDocument();
  });

  it('normal state keeps the hint', () => {
    renderPanel();
    expect(screen.getByText('達到上限會自動暫停，可稍後續跑')).toBeInTheDocument();
  });
});
