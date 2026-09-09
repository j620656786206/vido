import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
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
    search: '',
    searchQuery: '',
    sort: 'group',
    onSearchChange: vi.fn(),
    onSortChange: vi.fn(),
    ...overrides,
  };
  return { props, ...render(<CandidateListPanel {...props} />) };
}

/**
 * Open the series section. sub-6-11 AC #4 made shows COLLAPSED by default, so
 * every assertion about an episode row or a season header has to open the show
 * first — exactly what the user does.
 */
function expandSeries(seriesId = SRS) {
  const disclosure = screen.queryByTestId(`consent-group-${seriesId}-disclosure`);
  if (disclosure && disclosure.getAttribute('aria-expanded') === 'false') {
    fireEvent.click(disclosure);
  }
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

  it('[P0] unknown-runtime rows prefix ≈ on the amount and state the 45-minute assumption', () => {
    renderPanel();
    expect(screen.getByTestId(`consent-row-usd-${U}`).textContent).toBe('≈ $0.27');
    // sub-6-10b AC #2: the fallback no longer REPLACES the route line — the
    // two sit side by side, because 「為什麼這列要收錢」 is the sentence the
    // 45-minute caveat used to erase.
    const row = screen.getByTestId(`consent-row-${U}`);
    expect(row.textContent).toContain('無文字字幕軌 → 語音辨識 + 翻譯');
    expect(row.textContent).toContain('片長未知（估 45 分）');
  });

  it('[P0] Sally 裁定 1 — at most one ≈ per row, and it belongs to the amount', () => {
    renderPanel();
    // Two ≈ in one row meant two different things: the amount's ≈ says "this
    // figure rests on an assumed length"; the runtime's said "we do not know".
    // 45 minutes is an ASSUMPTION, not an approximate measurement, so it is
    // stated as a fact with the assumption in brackets instead.
    const row = screen.getByTestId(`consent-row-${U}`);
    expect((row.textContent ?? '').match(/≈/g)).toHaveLength(1);
    expect(screen.getByTestId(`consent-row-usd-${U}`).textContent).toContain('≈');
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

  it('[CR H1 · sub-6-11 AC #4] the header reports the SECTION route composition, and the subtotal the selection', () => {
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

    // 1 extract + 2 asr across the WHOLE show — not the 1+1 that is ticked.
    const routes = screen.getByTestId(`consent-group-${SRS}-routes`);
    expect(routes).toHaveTextContent('抽取 1');
    expect(routes).toHaveTextContent('語音辨識 2');
    expect(screen.getByTestId(`consent-group-${SRS}-selected`)).toHaveTextContent(
      '已選 2/3 · $0.34'
    );
  });

  it('[sub-6-11 AC #4] the route composition shows with NOTHING selected — that is when the user needs it', () => {
    // Pre-sub-6-11 this row was blank until something was ticked, so 「這部劇
    // 要花錢嗎」 could only be answered after the user had already ticked it.
    renderPanel({ candidates: GROUPED, selectedIds: new Set() });
    const routes = screen.getByTestId(`consent-group-${SRS}-routes`);
    expect(routes).toHaveTextContent('語音辨識 3');
    // GROUPED is 100% asr, so the extract badge is still correctly absent.
    expect(routes).not.toHaveTextContent('抽取');
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
    expandSeries();
    expect(screen.getByTestId(`consent-season-${SRS}-1`)).toHaveTextContent('第 1 季');
    expect(screen.getByTestId(`consent-season-${SRS}-2`)).toHaveTextContent('第 2 季');

    cleanup();
    // Single-season show → series header only, no season row.
    renderPanel({
      candidates: [seriesEp(S1E1, 1, 1), seriesEp(S1E2, 1, 2)],
      selectedIds: new Set(),
    });
    expect(screen.getByTestId(`consent-group-${SRS}`)).toBeInTheDocument();
    expandSeries();
    expect(screen.queryByTestId(`consent-season-${SRS}-1`)).toBeNull();
  });

  it('S00 renders as 特別篇', () => {
    renderPanel({
      candidates: [seriesEp(S1E1, 0, 1), seriesEp(S2E1, 1, 1)],
      selectedIds: new Set(),
    });
    expandSeries();
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

    expandSeries();
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
      blocker: 'SUBTITLE_TARGET_NOT_WRITABLE',
      blockerDir: 'ro-folder',
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
    expect(badge).toHaveAttribute('title', '資料夾無法寫入：ro-folder');
    expect(screen.getByTestId('consent-unwritable-count')).toHaveTextContent('1 部資料夾無法寫入');
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

describe('sub-6-1 group header with an unwritable member (CR H3)', () => {
  const S = '5fa9fed7-8fbc-4e8d-8edc-f6b7c8d9e011';
  const E1 = '6ab0afe8-9acd-4f9e-9fed-a7c8d9e0f112';
  const E2 = '7bc1b0f9-abde-4a0f-8afe-b8d9e0f1a213';
  const season: GenerationCandidate[] = [
    {
      mediaId: E1,
      mediaType: 'episode',
      title: '劇 S01E01',
      route: 'extract',
      runtimeMinutes: 45,
      runtimeKnown: true,
      estimatedUsd: 0.02,
      seriesId: S,
      seriesTitle: '劇',
      seasonNumber: 1,
      episodeNumber: 1,
    },
    {
      mediaId: E2,
      mediaType: 'episode',
      title: '劇 S01E02',
      route: 'extract',
      runtimeMinutes: 45,
      runtimeKnown: true,
      estimatedUsd: 0.02,
      seriesId: S,
      seriesTitle: '劇',
      seasonNumber: 1,
      episodeNumber: 2,
      writable: false,
      blocker: 'SUBTITLE_TARGET_NOT_WRITABLE',
      blockerDir: 'S01',
    },
  ];

  it('reaches "all" with only the selectable member selected and toggles only selectable ids', () => {
    cleanup();
    const selectedIds = new Set([E1]);
    const onToggleGroup = vi.fn();
    renderPanel({
      candidates: season,
      selectedIds,
      totals: computeTotals(season, selectedIds, 5),
      onToggleGroup,
    });
    const header = screen
      .getByTestId(`consent-group-${S}`)
      .querySelector('input[type="checkbox"]') as HTMLInputElement;
    expect(header.checked).toBe(true);
    expect(header.indeterminate).toBe(false);
    fireEvent.click(header);
    expect(onToggleGroup).toHaveBeenCalledWith([E1], false);
  });
});

// ─── sub-6-10b: row identity ───────────────────────────────────────────────
//
// The critique this closes was a screenshot of 2,399 identical grey squares
// titled with release filenames. Every assertion below is about a row being
// recognisable enough to consent to.

describe('CandidateListPanel — row identity (sub-6-10b)', () => {
  const M = '3d87dcb5-6d9a-4c6b-8cbf-d4f5a6b7c804';

  function row(over: Partial<GenerationCandidate> = {}): GenerationCandidate {
    return {
      mediaId: M,
      mediaType: 'movie',
      title: '沙丘：第二部',
      route: 'extract',
      runtimeMinutes: 166,
      runtimeKnown: true,
      estimatedUsd: 0.05,
      ...over,
    };
  }

  function renderOne(over: Partial<GenerationCandidate> = {}) {
    cleanup();
    const candidates = [row(over)];
    return renderPanel({
      candidates,
      selectedIds: new Set<string>(),
      totals: computeTotals(candidates, new Set<string>(), 5),
    });
  }

  it('AC #1 — renders the poster at w92, decorative and lazy', () => {
    renderOne({ posterPath: '/dune.jpg' });
    const img = screen.getByTestId(`consent-row-poster-${M}`) as HTMLImageElement;
    expect(img.getAttribute('src')).toBe('https://image.tmdb.org/t/p/w92/dune.jpg');
    // Decorative: the title is right beside it, so announcing it would make a
    // screen reader read every row's name twice.
    expect(img.getAttribute('alt')).toBe('');
    expect(img.getAttribute('loading')).toBe('lazy');
  });

  it('AC #1 — no poster falls back to the title initial, never an empty square', () => {
    renderOne();
    expect(screen.queryByTestId(`consent-row-poster-${M}`)).toBeNull();
    expect(screen.getByTestId(`consent-row-poster-fallback-${M}`).textContent).toBe('沙');
  });

  it('Sally 裁定 3 — the initial is drawn at the Title tier, not as muted small text', () => {
    renderOne();
    const fallback = screen.getByTestId(`consent-row-poster-fallback-${M}`);
    // A DELIBERATE exception to DESIGN.md's Small-By-Default rule, locked here
    // so nobody "fixes" it back: this is a graphic stand-in for a poster, not a
    // text label. At 12px muted on --bg-tertiary it is grey-on-grey and cannot
    // do its only job — telling row 47 apart from row 48.
    expect(fallback).toHaveClass('text-lg', 'font-semibold', 'text-[var(--text-secondary)]');
  });

  it('AC #1 — a poster that fails to load degrades to the initial', () => {
    renderOne({ posterPath: '/gone.jpg' });
    fireEvent.error(screen.getByTestId(`consent-row-poster-${M}`));
    expect(screen.queryByTestId(`consent-row-poster-${M}`)).toBeNull();
    expect(screen.getByTestId(`consent-row-poster-fallback-${M}`).textContent).toBe('沙');
  });

  it('AC #2 — route and runtime sit side by side, in the shipped zh-TW form', () => {
    renderOne({ runtimeSource: 'ffprobe' });
    const text = screen.getByTestId(`consent-row-${M}`).textContent ?? '';
    expect(text).toContain('內嵌英文字幕 → 翻譯');
    // formatRuntime's form — the same one the PosterCard and detail page use.
    expect(text).toContain('2 小時 46 分');
  });

  it('AC #2 — only runtime_source=fallback gets the ≈ marker', () => {
    for (const source of ['ffprobe', 'tmdb'] as const) {
      renderOne({ runtimeSource: source });
      expect(screen.getByTestId(`consent-row-usd-${M}`).textContent).toBe('$0.05');
    }
    renderOne({ runtimeSource: 'fallback', runtimeKnown: false, runtimeMinutes: 45 });
    expect(screen.getByTestId(`consent-row-usd-${M}`).textContent).toBe('≈ $0.05');
    expect(screen.getByTestId(`consent-row-${M}`).textContent).toContain('片長未知（估 45 分）');
  });

  it('AC #2 — a pre-sub-6-10a server (no runtime_source) behaves exactly as before', () => {
    renderOne({ runtimeKnown: false, runtimeMinutes: 45 });
    expect(screen.getByTestId(`consent-row-usd-${M}`).textContent).toBe('≈ $0.05');
    renderOne({ runtimeKnown: true });
    expect(screen.getByTestId(`consent-row-usd-${M}`).textContent).toBe('$0.05');
  });

  it('AC #3 — an unmatched row is marked, titled by display_title, and keeps the filename on hover', () => {
    renderOne({
      title: '[bitsearch.to] Predator.Badlands.2025.2160p.WEB-DL',
      displayTitle: 'Predator Badlands (2025)',
      tmdbMatched: false,
    });

    expect(screen.getByText('Predator Badlands (2025)')).toBeInTheDocument();
    expect(screen.queryByText('[bitsearch.to] Predator.Badlands.2025.2160p.WEB-DL')).toBeNull();

    const badge = screen.getByTestId(`consent-row-unmatched-${M}`);
    expect(badge.textContent).toBe('未匹配');
    expect(badge).toHaveAttribute('title', 'TMDb 沒有比對到，片名由檔名解析');

    // The raw filename is what the user will look for on disk — keep it reachable.
    expect(screen.getByText('Predator Badlands (2025)')).toHaveAttribute(
      'title',
      '[bitsearch.to] Predator.Badlands.2025.2160p.WEB-DL'
    );
  });

  it('Sally 裁定 2 — 未匹配 sits beside the title it doubts, and survives a long title', () => {
    renderOne({
      title: '[bitsearch.to] Predator.Badlands.2025.2160p.WEB-DL.DDP5.1.Atmos.H.265-FLUX',
      displayTitle: 'Predator Badlands (2025)',
      tmdbMatched: false,
    });

    const badge = screen.getByTestId(`consent-row-unmatched-${M}`);
    const title = screen.getByText('Predator Badlands (2025)');
    // Doubt belongs next to the thing being doubted. The right-hand cluster is
    // state → kind → cost; 「未匹配」 is none of those, and half a row away it
    // no longer reads as being ABOUT the title.
    expect(title.parentElement).toBe(badge.parentElement);
    // The title yields, the badge does not — otherwise a long filename pushes
    // the one mark that explains it off the row.
    expect(title).toHaveClass('truncate');
    expect(badge).toHaveClass('shrink-0');
  });

  it('AC #3 — a matched row carries no 未匹配 mark, and an old server never gets one', () => {
    renderOne({ tmdbMatched: true });
    expect(screen.queryByTestId(`consent-row-unmatched-${M}`)).toBeNull();
    // Field absent = "the server never told us", which is NOT "TMDb found nothing".
    renderOne();
    expect(screen.queryByTestId(`consent-row-unmatched-${M}`)).toBeNull();
  });

  it('AC #3 — the checkbox label follows the display title', () => {
    renderOne({ displayTitle: 'Predator Badlands (2025)', tmdbMatched: false });
    expect(screen.getByLabelText('選取 Predator Badlands (2025)')).toBeInTheDocument();
  });
});

describe('CandidateListPanel — phone-width row (bugfix-f15-row-mobile-identity-collapse)', () => {
  const M = '4e98edc6-7eab-4d7c-9dc0-e5a6b7c8d905';

  function renderOne(over: Partial<GenerationCandidate> = {}) {
    cleanup();
    const candidates: GenerationCandidate[] = [
      {
        mediaId: M,
        mediaType: 'movie',
        title: '星際效應',
        route: 'asr',
        runtimeMinutes: 45,
        runtimeKnown: false,
        estimatedUsd: 0.24,
        ...over,
      },
    ];
    return renderPanel({
      candidates,
      selectedIds: new Set<string>(),
      totals: computeTotals(candidates, new Set<string>(), 5),
    });
  }

  it('the row re-flows on the LIST width, not the viewport', () => {
    renderOne();
    // A container query: the drawn phone row must also hold in a narrow
    // desktop dialog, and a 390px gallery fixture must be able to pin it
    // under the harness's fixed 1280px viewport.
    expect(screen.getByTestId('consent-candidate-list')).toHaveClass('@container');
  });

  it('below 36rem the cost cluster sits on the title line; above it spans both lines', () => {
    renderOne();
    const cluster = screen.getByTestId(`consent-row-cluster-${M}`);
    expect(cluster).toHaveClass('row-start-1');
    expect(cluster).toHaveClass('@xl:row-end-3');
    // ONE cost node serves both layouts — no hidden duplicate that could drift.
    expect(screen.getAllByTestId(`consent-row-usd-${M}`)).toHaveLength(1);
  });

  it('below 36rem the subtitle takes both columns and wraps instead of truncating', () => {
    renderOne();
    const subtitle = screen.getByTestId(`consent-row-subtitle-${M}`);
    expect(subtitle).toHaveTextContent('無文字字幕軌 → 語音辨識 + 翻譯 · 片長未知（估 45 分）');
    expect(subtitle).toHaveClass('truncate');
    expect(subtitle).toHaveClass('@max-xl:col-end-3');
    expect(subtitle).toHaveClass('@max-xl:whitespace-normal');
  });

  it('below 36rem the kind badge is gone — the amount colour already says the route', () => {
    renderOne();
    const kind = screen.getByTestId(`consent-row-kind-${M}`);
    expect(kind).toHaveTextContent('語音辨識');
    expect(kind).toHaveClass('hidden');
    expect(kind).toHaveClass('@xl:inline');
    // The unwritable state stays visible in both layouts (F15-M-v2 draws it
    // beside the amount): it changes what the user may do, the kind does not.
    renderOne({ writable: false, blocker: 'folder_not_writable' });
    expect(screen.getByTestId(`consent-row-unwritable-${M}`)).not.toHaveClass('hidden');
  });

  it('the unmatched mark still sits beside the title in the phone layout', () => {
    renderOne({ tmdbMatched: false, displayTitle: '星際效應', title: 'Interstellar.2014' });
    const badge = screen.getByTestId(`consent-row-unmatched-${M}`);
    const title = screen.getByText('星際效應');
    expect(title.parentElement).toBe(badge.parentElement);
    expect(title.parentElement).toHaveClass('row-start-1');
  });
});

// ─── sub-6-11: search, sort, virtualization, collapse ───────────────────────

describe('CandidateListPanel — search (sub-6-11 AC #1)', () => {
  it('typing reports up to the container; the panel itself filters on the DEBOUNCED copy', () => {
    const onSearchChange = vi.fn();
    // `search` is mid-keystroke, `searchQuery` is what the 200ms debounce has
    // committed — so the list still shows everything while the user types.
    renderPanel({ search: '沙', searchQuery: '', onSearchChange });
    fireEvent.change(screen.getByTestId('consent-search-input'), { target: { value: '沙丘' } });
    expect(onSearchChange).toHaveBeenCalledWith('沙丘');
    expect(screen.getByTestId(`consent-row-${B}`)).toBeInTheDocument();
  });

  it('a committed query narrows the rows and leaves the totals alone', () => {
    renderPanel({ search: '沙丘', searchQuery: '沙丘' });
    expect(screen.getByTestId(`consent-row-${A}`)).toBeInTheDocument();
    expect(screen.queryByTestId(`consent-row-${B}`)).toBeNull();
    // The summary bar still counts the whole library — search is a VIEW filter.
    expect(screen.getByTestId('consent-summary-usd')).toHaveTextContent('$0.09');
    expect(screen.getByTestId('consent-chip-all')).toHaveTextContent('4');
  });

  it('search multiplies with the route chip rather than replacing it', () => {
    renderPanel({ search: '電影', searchQuery: '電影', filter: 'extract' });
    // 未知片長的電影 matches the text but is an ASR row, which the chip hides.
    expect(screen.queryByTestId(`consent-row-${U}`)).toBeNull();
    expect(screen.getByTestId('consent-search-empty')).toBeInTheDocument();
  });

  it('no hit shows 沒有符合的候選 and a way back out', () => {
    const onSearchChange = vi.fn();
    renderPanel({ search: '不存在的片', searchQuery: '不存在的片', onSearchChange });
    expect(screen.getByTestId('consent-search-empty')).toHaveTextContent('沒有符合的候選');
    fireEvent.click(screen.getByTestId('consent-search-empty-clear'));
    expect(onSearchChange).toHaveBeenCalledWith('');
  });

  it('the clear button appears only once there is something to clear', () => {
    const { rerender, props } = renderPanel({ search: '', searchQuery: '' });
    expect(screen.queryByTestId('consent-search-clear')).toBeNull();
    rerender(<CandidateListPanel {...props} search="沙" searchQuery="" />);
    fireEvent.click(screen.getByTestId('consent-search-clear'));
    expect(props.onSearchChange).toHaveBeenCalledWith('');
  });

  it('an empty list with NO search shows no empty state — nothing was searched for', () => {
    renderPanel({
      candidates: [],
      selectedIds: new Set(),
      totals: computeTotals([], new Set(), 5),
    });
    expect(screen.queryByTestId('consent-search-empty')).toBeNull();
  });
});

describe('CandidateListPanel — sort (sub-6-11 AC #2)', () => {
  it('the menu offers the five documented orders and reports a change', () => {
    const onSortChange = vi.fn();
    renderPanel({ onSortChange });
    const select = screen.getByTestId('consent-sort-select') as HTMLSelectElement;
    expect([...select.options].map((o) => o.value)).toEqual([
      'group',
      'cost-desc',
      'cost-asc',
      'title-asc',
      'unmatched-first',
    ]);
    fireEvent.change(select, { target: { value: 'cost-desc' } });
    expect(onSortChange).toHaveBeenCalledWith('cost-desc');
  });

  it('金額高→低 reorders the DRAWN rows and drops every group header', () => {
    renderPanel({ candidates: GROUPED, selectedIds: new Set(), sort: 'cost-desc' });
    expect(screen.queryByTestId(`consent-group-${SRS}`)).toBeNull();
    const drawn = [...screen.getByTestId('consent-candidate-list').children].map((el) =>
      el.getAttribute('data-testid')
    );
    // The three $0.30 episodes lead (tie broken by submission order); the
    // $0.05 film that heads the grouped view sinks to the bottom.
    expect(drawn).toEqual([
      `consent-row-${S1E1}`,
      `consent-row-${S1E2}`,
      `consent-row-${S2E1}`,
      `consent-row-${A}`,
    ]);
  });

  it('the over-budget banner says the N is counted in SUBMISSION order once sorted', () => {
    const all = new Set(CANDIDATES.map((x) => x.mediaId));
    renderPanel({
      selectedIds: all,
      totals: computeTotals(CANDIDATES, all, 0.3),
      budgetText: '0.30',
      budgetUsd: 0.3,
      sort: 'cost-desc',
    });
    expect(screen.getByTestId('consent-feasible-order-note')).toHaveTextContent('依提交順序');
  });

  it('the note is absent in the default 群組 order, where the two orders agree', () => {
    const all = new Set(CANDIDATES.map((x) => x.mediaId));
    renderPanel({
      selectedIds: all,
      totals: computeTotals(CANDIDATES, all, 0.3),
      budgetText: '0.30',
      budgetUsd: 0.3,
    });
    expect(screen.queryByTestId('consent-feasible-order-note')).toBeNull();
  });
});

describe('CandidateListPanel — collapse (sub-6-11 AC #4)', () => {
  it('a show is CLOSED on arrival: header yes, episodes no', () => {
    renderPanel({ candidates: GROUPED, selectedIds: new Set() });
    expect(screen.getByTestId(`consent-group-${SRS}`)).toHaveAttribute('data-expanded', 'false');
    expect(screen.queryByTestId(`consent-row-${S1E1}`)).toBeNull();
  });

  it('the disclosure opens it, and opening is not the same gesture as ticking it', () => {
    const onToggleGroup = vi.fn();
    renderPanel({ candidates: GROUPED, selectedIds: new Set(), onToggleGroup });
    fireEvent.click(screen.getByTestId(`consent-group-${SRS}-disclosure`));
    expect(screen.getByTestId(`consent-row-${S1E1}`)).toBeInTheDocument();
    expect(onToggleGroup).not.toHaveBeenCalled();
  });

  it('a search opens the show it hit, without the user touching the disclosure', () => {
    renderPanel({
      candidates: GROUPED,
      selectedIds: new Set(),
      search: '怪奇',
      searchQuery: '怪奇',
    });
    expect(screen.getByTestId(`consent-row-${S1E1}`)).toBeInTheDocument();
  });

  it('a collapsed header still says what the show would cost, with nothing ticked', () => {
    renderPanel({ candidates: GROUPED, selectedIds: new Set() });
    expect(screen.getByTestId(`consent-group-${SRS}-routes`)).toHaveTextContent('語音辨識 3');
    expect(screen.getByTestId(`consent-group-${SRS}-selected`)).toHaveTextContent('已選 0/3');
  });

  it('movies split into 已匹配 / 未匹配, matched first, both collapsible', () => {
    const split = [
      { ...CANDIDATES[0], tmdbMatched: true },
      { ...CANDIDATES[1], tmdbMatched: false },
    ];
    renderPanel({
      candidates: split,
      selectedIds: new Set(),
      totals: computeTotals(split, new Set(), 5),
    });
    const list = screen.getByTestId('consent-candidate-list');
    const drawn = [...list.children].map((el) => el.getAttribute('data-testid'));
    expect(drawn).toEqual([
      'consent-movies-matched',
      `consent-row-${A}`,
      'consent-movies-unmatched',
      `consent-row-${B}`,
    ]);

    fireEvent.click(screen.getByTestId('consent-movies-unmatched-disclosure'));
    expect(screen.queryByTestId(`consent-row-${B}`)).toBeNull();
    expect(screen.getByTestId(`consent-row-${A}`)).toBeInTheDocument();
  });

  it('an all-matched library keeps the flat, header-less list it shipped with', () => {
    renderPanel({ candidates: CANDIDATES.map((x) => ({ ...x, tmdbMatched: true })) });
    expect(screen.queryByTestId('consent-movies-matched')).toBeNull();
  });
});

describe('CandidateListPanel — 2,400 rows (sub-6-11 AC #3)', () => {
  const MANY: GenerationCandidate[] = Array.from({ length: 2400 }, (_, i) => ({
    mediaId: `9f000000-0000-4000-8000-${String(i).padStart(12, '0')}`,
    mediaType: 'movie' as const,
    title: `電影 ${i}`,
    route: 'extract' as const,
    runtimeMinutes: 100,
    runtimeKnown: true,
    estimatedUsd: 0.05,
  }));

  // jsdom lays nothing out — every box is 0×0, so the virtualizer would read
  // the viewport as empty and this suite would pass for the wrong reason. Give
  // the scroller a real 600px viewport and each row the height a browser
  // reports; then the assertions are about the windowing, not about jsdom.
  // The library measures the SCROLLER with offsetHeight and each ROW with the
  // panel's own measureElement (getBoundingClientRect), so both are stubbed.
  const HEIGHT_OF = (el: Element) => (el.tagName === 'LI' ? 86 : 600);
  beforeEach(() => {
    Object.defineProperty(HTMLElement.prototype, 'offsetHeight', {
      configurable: true,
      get(this: HTMLElement) {
        return HEIGHT_OF(this);
      },
    });
    Object.defineProperty(HTMLElement.prototype, 'offsetWidth', {
      configurable: true,
      get: () => 800,
    });
    vi.spyOn(Element.prototype, 'getBoundingClientRect').mockImplementation(function (
      this: Element
    ) {
      const height = HEIGHT_OF(this);
      return {
        height,
        width: 800,
        top: 0,
        left: 0,
        right: 800,
        bottom: height,
        x: 0,
        y: 0,
        toJSON: () => ({}),
      } as ReturnType<Element['getBoundingClientRect']>;
    });
  });
  afterEach(() => {
    vi.restoreAllMocks();
    delete (HTMLElement.prototype as Partial<HTMLElement>).offsetHeight;
    delete (HTMLElement.prototype as Partial<HTMLElement>).offsetWidth;
  });

  it('draws a WINDOW, not 2,400 rows', () => {
    renderPanel({
      candidates: MANY,
      selectedIds: new Set(),
      totals: computeTotals(MANY, new Set(), 5),
    });
    const list = screen.getByTestId('consent-candidate-list');
    expect(list).toHaveAttribute('data-virtualized', 'true');
    expect(list.children.length).toBeGreaterThan(0);
    expect(list.children.length).toBeLessThan(100);
  });

  it('a small library is NOT virtualized — it keeps the plain list', () => {
    renderPanel();
    expect(screen.getByTestId('consent-candidate-list')).toHaveAttribute(
      'data-virtualized',
      'false'
    );
    expect(screen.getByTestId(`consent-row-${A}`)).toBeInTheDocument();
  });

  it('a new search, filter or sort returns the scroll to the top', () => {
    const { rerender, props } = renderPanel({
      candidates: MANY,
      selectedIds: new Set(),
      totals: computeTotals(MANY, new Set(), 5),
    });
    const scroller = screen.getByTestId('consent-list-scroll');
    scroller.scrollTop = 14000;
    rerender(<CandidateListPanel {...props} sort="cost-desc" />);
    expect(scroller.scrollTop).toBe(0);
  });
});

describe('CandidateListPanel — CR fixes (sub-6-11)', () => {
  it('collapsing a show DURING a search actually collapses it', () => {
    renderPanel({
      candidates: GROUPED,
      selectedIds: new Set(),
      search: '怪奇',
      searchQuery: '怪奇',
    });
    // The search forced it open…
    expect(screen.getByTestId(`consent-row-${S1E1}`)).toBeInTheDocument();
    // …and one click closes it, instead of silently recording the opposite.
    fireEvent.click(screen.getByTestId(`consent-group-${SRS}-disclosure`));
    expect(screen.queryByTestId(`consent-row-${S1E1}`)).toBeNull();
    expect(screen.getByTestId(`consent-group-${SRS}`)).toHaveAttribute('data-expanded', 'false');
  });

  it('the header denominator counts the SAME rows the route badges do', () => {
    // 2 of 3 episodes are writable → 「語音辨識 2」 must sit beside 「已選 0/2」,
    // not beside 「已選 0/3」.
    const mixed = [
      seriesEp(S1E1, 1, 1),
      seriesEp(S1E2, 1, 2),
      seriesEp(S2E1, 2, 1, { writable: false, blocker: 'folder_not_writable' }),
    ];
    renderPanel({
      candidates: mixed,
      selectedIds: new Set(),
      totals: computeTotals(mixed, new Set(), 5),
    });
    expect(screen.getByTestId(`consent-group-${SRS}-routes`)).toHaveTextContent('語音辨識 2');
    expect(screen.getByTestId(`consent-group-${SRS}-selected`)).toHaveTextContent('已選 0/2');
  });

  it('a movie section checkbox is not announced as 「選取整部」', () => {
    const split = [
      { ...CANDIDATES[0], tmdbMatched: true },
      { ...CANDIDATES[1], tmdbMatched: false },
    ];
    renderPanel({
      candidates: split,
      selectedIds: new Set(),
      totals: computeTotals(split, new Set(), 5),
    });
    expect(screen.getByLabelText('選取所有已匹配的電影')).toBeInTheDocument();
    expect(screen.getByLabelText('選取所有未匹配的電影')).toBeInTheDocument();
  });

  it('the list keeps a minimum height so a short dialog cannot collapse it to nothing', () => {
    renderPanel();
    expect(screen.getByTestId('consent-list-scroll').className).toContain('min-h-[10rem]');
  });
});
