import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/react';
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
    onToggleGroup: vi.fn(),
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

// ---------------------------------------------------------------------------
// sub-5-3 AC #2 — series/season group headers
// ---------------------------------------------------------------------------

const S1E1 = '9a0bfe08-1acd-4f9e-9fed-a7c8d9e0f107';
const S1E2 = '9a0bfe08-1acd-4f9e-9fed-a7c8d9e0f108';
const S2E1 = '9a0bfe08-1acd-4f9e-9fed-a7c8d9e0f109';
const SRS = 'b1c2d3e4-f5a6-4b7c-8d9e-0f1a2b3c4d5e';

function seriesEp(
  mediaId: string,
  season: number,
  episode: number,
  overrides: Partial<GenerationCandidate> = {}
): GenerationCandidate {
  return {
    mediaId,
    mediaType: 'episode',
    title: `怪奇物語 S0${season}E0${episode}`,
    route: 'asr',
    runtimeMinutes: 50,
    runtimeKnown: true,
    estimatedUsd: 0.3,
    seriesId: SRS,
    seriesTitle: '怪奇物語',
    seasonNumber: season,
    episodeNumber: episode,
    ...overrides,
  };
}

const GROUPED = [CANDIDATES[0], seriesEp(S1E1, 1, 1), seriesEp(S1E2, 1, 2), seriesEp(S2E1, 2, 1)];

describe('CandidateListPanel — series/season grouping (sub-5-3 AC #2)', () => {
  it('renders a series header with title, 已選 n/N and the SELECTED subtotal', () => {
    renderPanel({
      candidates: GROUPED,
      selectedIds: new Set([S1E1, S1E2]),
      totals: computeTotals(GROUPED, new Set([S1E1, S1E2]), 5),
    });

    const header = screen.getByTestId(`consent-group-${SRS}`);
    expect(header).toHaveTextContent('怪奇物語');
    expect(screen.getByTestId(`consent-group-${SRS}-selected`)).toHaveTextContent(
      '已選 2/3 · $0.60'
    );
  });

  it('[CR H1] the header reports the SELECTED route composition from computeTotals', () => {
    const mixed = [
      seriesEp(S1E1, 1, 1),
      seriesEp(S1E2, 1, 2, { route: 'extract', estimatedUsd: 0.04 }),
      seriesEp(S2E1, 2, 1),
    ];
    renderPanel({
      candidates: mixed,
      selectedIds: new Set([S1E1, S1E2]),
      totals: computeTotals(mixed, new Set([S1E1, S1E2]), 5),
    });

    const routes = screen.getByTestId(`consent-group-${SRS}-routes`);
    expect(routes).toHaveTextContent('抽取 1');
    expect(routes).toHaveTextContent('語音辨識 1');
    expect(screen.getByTestId(`consent-group-${SRS}-selected`)).toHaveTextContent(
      '已選 2/3 · $0.34'
    );
  });

  it('[CR H1] a route badge is omitted when that route has nothing selected', () => {
    renderPanel({ candidates: GROUPED, selectedIds: new Set() });
    const routes = screen.getByTestId(`consent-group-${SRS}-routes`);
    expect(routes).not.toHaveTextContent('抽取');
    expect(routes).not.toHaveTextContent('語音辨識');
  });

  it('[CR L5] a group whose every row is filtered out renders no header at all', () => {
    // Series is 100% asr; the extract chip hides every row → the header would
    // be a checkbox governing nothing.
    renderPanel({
      candidates: [CANDIDATES[0], seriesEp(S1E1, 1, 1), seriesEp(S1E2, 1, 2)],
      selectedIds: new Set(),
      filter: 'extract',
    });
    expect(screen.queryByTestId(`consent-group-${SRS}`)).toBeNull();
    expect(screen.getByTestId(`consent-row-${A}`)).toBeInTheDocument();
  });

  it('season headers appear only for a multi-season show, labelled 第 n 季', () => {
    renderPanel({ candidates: GROUPED, selectedIds: new Set() });
    expect(screen.getByTestId(`consent-season-${SRS}-1`)).toHaveTextContent('第 1 季');
    expect(screen.getByTestId(`consent-season-${SRS}-2`)).toHaveTextContent('第 2 季');

    cleanup();
    // Single-season show → series header only, no season row.
    renderPanel({
      candidates: [seriesEp(S1E1, 1, 1), seriesEp(S1E2, 1, 2)],
      selectedIds: new Set(),
    });
    expect(screen.getByTestId(`consent-group-${SRS}`)).toBeInTheDocument();
    expect(screen.queryByTestId(`consent-season-${SRS}-1`)).toBeNull();
  });

  it('S00 renders as 特別篇', () => {
    renderPanel({
      candidates: [seriesEp(S1E1, 0, 1), seriesEp(S2E1, 1, 1)],
      selectedIds: new Set(),
    });
    expect(screen.getByTestId(`consent-season-${SRS}-0`)).toHaveTextContent('特別篇');
  });

  it('the series checkbox toggles the WHOLE group and reports mixed state honestly', () => {
    const onToggleGroup = vi.fn();
    renderPanel({
      candidates: GROUPED,
      selectedIds: new Set([S1E1]),
      onToggleGroup,
    });

    const box = screen.getByLabelText('選取整部 怪奇物語') as HTMLInputElement;
    // CR L3: mixed state rides the NATIVE indeterminate property (the shipped
    // consent-select-all idiom) — no redundant aria-checked on a native input.
    expect(box.indeterminate).toBe(true);
    expect(box.checked).toBe(false);
    expect(box).not.toHaveAttribute('aria-checked');

    fireEvent.click(box);
    expect(onToggleGroup).toHaveBeenCalledWith([S1E1, S1E2, S2E1], true);
  });

  it('a fully-selected group unchecks on toggle (next=false)', () => {
    const onToggleGroup = vi.fn();
    renderPanel({
      candidates: GROUPED,
      selectedIds: new Set([S1E1, S1E2, S2E1]),
      onToggleGroup,
    });

    fireEvent.click(screen.getByLabelText('選取整部 怪奇物語'));
    expect(onToggleGroup).toHaveBeenCalledWith([S1E1, S1E2, S2E1], false);
  });

  it('the season checkbox toggles only that season', () => {
    const onToggleGroup = vi.fn();
    renderPanel({ candidates: GROUPED, selectedIds: new Set(), onToggleGroup });

    fireEvent.click(screen.getByLabelText('選取第 1 季'));
    expect(onToggleGroup).toHaveBeenCalledWith([S1E1, S1E2], true);
  });

  it('group toggle semantics ignore the route filter — chips are a VIEW filter only', () => {
    const onToggleGroup = vi.fn();
    const mixed = [
      seriesEp(S1E1, 1, 1),
      seriesEp(S1E2, 1, 2, { route: 'extract' }),
      seriesEp(S2E1, 2, 1),
    ];
    renderPanel({
      candidates: mixed,
      selectedIds: new Set(),
      filter: 'asr',
      onToggleGroup,
    });

    // Only asr rows are VISIBLE, but the header operates on ALL group items —
    // the same semantics 全選 already ships.
    expect(screen.queryByTestId(`consent-row-${S1E2}`)).toBeNull();
    fireEvent.click(screen.getByLabelText('選取整部 怪奇物語'));
    expect(onToggleGroup).toHaveBeenCalledWith([S1E1, S1E2, S2E1], true);
  });

  it('a degraded (empty) series title renders 未知影集 and still groups', () => {
    renderPanel({
      candidates: [
        seriesEp(S1E1, 1, 1, { seriesTitle: '' }),
        seriesEp(S1E2, 1, 2, { seriesTitle: '' }),
      ],
      selectedIds: new Set(),
    });
    expect(screen.getByTestId(`consent-group-${SRS}`)).toHaveTextContent('未知影集');
  });

  it('pre-sub-5-3 rows (no seriesId) keep the flat shipped rendering — zero headers', () => {
    renderPanel({ candidates: CANDIDATES, selectedIds: new Set() });
    expect(screen.queryByText('未知影集')).toBeNull();
    expect(document.querySelector('[data-testid^="consent-group-"]')).toBeNull();
  });
});

describe('sub-6-1 unwritable rows', () => {
  const RO = '4e98edc6-7eab-4d7c-9dcb-e5a6b7c8d905';
  const withUnwritable: GenerationCandidate[] = [
    ...CANDIDATES,
    {
      mediaId: RO,
      mediaType: 'movie',
      title: '唯讀資料夾裡的電影',
      route: 'extract',
      runtimeMinutes: 100,
      runtimeKnown: true,
      estimatedUsd: 0.03,
      writable: false,
      blocker: '資料夾無法寫入：permission denied',
    },
  ];

  it('disables the checkbox and shows the blocker badge', () => {
    cleanup();
    const selectedIds = new Set([A, B]);
    renderPanel({
      candidates: withUnwritable,
      selectedIds,
      totals: computeTotals(withUnwritable, selectedIds, 5),
    });
    const row = screen.getByTestId(`consent-row-${RO}`);
    expect(row).toHaveAttribute('data-writable', 'false');
    const box = row.querySelector('input[type="checkbox"]') as HTMLInputElement;
    expect(box).toBeDisabled();
    const badge = screen.getByTestId(`consent-row-unwritable-${RO}`);
    expect(badge).toHaveTextContent('資料夾無法寫入');
    expect(badge).toHaveAttribute('title', '資料夾無法寫入：permission denied');
  });

  it('全選 counts only selectable rows as "all"', () => {
    cleanup();
    // Every writable row selected (A, B, EP, U) — the unwritable RO is not part of "all".
    const selectedIds = new Set([A, B, EP, U]);
    renderPanel({
      candidates: withUnwritable,
      selectedIds,
      totals: computeTotals(withUnwritable, selectedIds, 5),
    });
    const all = screen.getByTestId('consent-select-all') as HTMLInputElement;
    expect(all.checked).toBe(true);
    expect(all.indeterminate).toBe(false);
  });
});
