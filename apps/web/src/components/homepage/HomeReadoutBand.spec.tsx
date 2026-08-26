import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import React from 'react';
import type { HomeSummary } from '../../services/homeSummaryService';

vi.mock('../../hooks/useHomeSummary', () => ({
  useHomeSummary: vi.fn(),
}));
// The band's cells are router <Link>s; this spec has no router, so stub Link
// as a plain anchor (the RecentlyAddedRowV2 spec precedent).
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = (await importOriginal()) as Record<string, unknown>;
  return {
    ...actual,
    Link: ({ to, children, ...rest }: { to: string; children: React.ReactNode }) => (
      <a href={to} {...rest}>
        {children}
      </a>
    ),
  };
});

import { useHomeSummary } from '../../hooks/useHomeSummary';
import { HomeReadoutBand } from './HomeReadoutBand';

const mockUseHomeSummary = vi.mocked(useHomeSummary);

function summary(over: Partial<HomeSummary> = {}): HomeSummary {
  return {
    coverage: { status: 'ok', covered: 42, total: 55 },
    processedToday: { status: 'ok', count: 3 },
    attention: {
      status: 'ok',
      failedCount: 2,
      spentUsd: 1.2,
      budgetUsd: 5,
      spendSource: 'live_batch',
    },
    inFlight: { status: 'ok', count: 2 },
    ...over,
  };
}

function result(over: Record<string, unknown> = {}) {
  return {
    data: undefined,
    isLoading: false,
    isError: false,
    ...over,
  } as unknown as ReturnType<typeof useHomeSummary>;
}

describe('HomeReadoutBand (Home v3 讀數帶 — ux3-1-7)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('[P1] renders all four cells with their values (H1-D-v3)', () => {
    mockUseHomeSummary.mockReturnValue(result({ data: summary() }));
    render(<HomeReadoutBand />);
    expect(screen.getByTestId('readout-coverage-value')).toHaveTextContent('42/55');
    expect(screen.getByTestId('readout-processed-value')).toHaveTextContent('3 部');
    expect(screen.getByTestId('readout-attention-value')).toHaveTextContent('2 部失敗 · $1.2/$5');
    expect(screen.getByTestId('readout-inflight-value')).toHaveTextContent('2 個任務');
  });

  it('[P1] every cell is a door — coverage → /library, the rest → /activity', () => {
    mockUseHomeSummary.mockReturnValue(result({ data: summary() }));
    render(<HomeReadoutBand />);
    expect(screen.getByTestId('readout-coverage')).toHaveAttribute('href', '/library');
    expect(screen.getByTestId('readout-processed')).toHaveAttribute('href', '/activity');
    expect(screen.getByTestId('readout-attention')).toHaveAttribute('href', '/activity');
    expect(screen.getByTestId('readout-inflight')).toHaveAttribute('href', '/activity');
  });

  it('[P1] a degraded cell hides its NUMBER, not its label — siblings keep serving (量不到的格不顯示數字)', () => {
    mockUseHomeSummary.mockReturnValue(
      result({
        data: summary({ coverage: { status: 'unavailable', covered: 0, total: 0, error: 'db' } }),
      })
    );
    render(<HomeReadoutBand />);
    expect(screen.getByTestId('readout-coverage')).toBeInTheDocument();
    expect(screen.queryByTestId('readout-coverage-value')).toBeNull();
    expect(screen.getByTestId('readout-processed-value')).toHaveTextContent('3 部');
  });

  it('[P1] 0 failures renders 一切正常 — the cell never disappears, and wears NO amber', () => {
    mockUseHomeSummary.mockReturnValue(
      result({ data: summary({ attention: { status: 'ok', failedCount: 0 } }) })
    );
    render(<HomeReadoutBand />);
    const value = screen.getByTestId('readout-attention-value');
    expect(value).toHaveTextContent('一切正常');
    expect(value.className).not.toContain('warning');
  });

  it('[P1] failures > 0 wears --warning-text amber (固定詞彙: 要求了但沒發生)', () => {
    mockUseHomeSummary.mockReturnValue(
      result({ data: summary({ attention: { status: 'ok', failedCount: 2 } }) })
    );
    render(<HomeReadoutBand />);
    const value = screen.getByTestId('readout-attention-value');
    expect(value).toHaveTextContent('2 部失敗');
    expect(value.className).toContain('warning-text');
  });

  it('[P2] spend trio absent → no amount text (absent ≠ $0); last_run spend with 0 failures stays quiet', () => {
    mockUseHomeSummary.mockReturnValue(
      result({
        data: summary({
          attention: {
            status: 'ok',
            failedCount: 0,
            spentUsd: 0.42,
            budgetUsd: 5,
            spendSource: 'last_run',
          },
        }),
      })
    );
    render(<HomeReadoutBand />);
    // Historical spend beside 一切正常 would be noise, not an exception.
    expect(screen.getByTestId('readout-attention-value')).toHaveTextContent(/^一切正常$/);
  });

  it('[P2] last_run spend DOES surface when there are failures', () => {
    mockUseHomeSummary.mockReturnValue(
      result({
        data: summary({
          attention: {
            status: 'ok',
            failedCount: 1,
            spentUsd: 0.42,
            budgetUsd: 5,
            spendSource: 'last_run',
          },
        }),
      })
    );
    render(<HomeReadoutBand />);
    expect(screen.getByTestId('readout-attention-value')).toHaveTextContent('1 部失敗 · $0.4/$5');
  });

  it('[P1] first-run 0/0 — the coverage cell becomes the 開始掃描 door to scanner settings', () => {
    mockUseHomeSummary.mockReturnValue(
      result({ data: summary({ coverage: { status: 'ok', covered: 0, total: 0 } }) })
    );
    render(<HomeReadoutBand />);
    const cell = screen.getByTestId('readout-coverage');
    expect(cell).toHaveAttribute('href', '/settings/scanner');
    expect(cell).toHaveTextContent('開始掃描');
    expect(screen.getByTestId('readout-coverage-value')).toHaveTextContent('0/0');
    // H5-D-v3 sets the CTA half apart in accent — a flat muted label hides
    // that this cell is the one door a brand-new user needs.
    expect(screen.getByTestId('readout-coverage-action').className).toContain('accent-text');
  });

  it('[P2] the accent CTA half appears ONLY on first run — a stocked library has a plain label', () => {
    mockUseHomeSummary.mockReturnValue(result({ data: summary() }));
    render(<HomeReadoutBand />);
    expect(screen.queryByTestId('readout-coverage-action')).toBeNull();
  });

  it('[P2] 0 renders as 0 — a quiet day is a readout, not an absence (0 是資訊)', () => {
    mockUseHomeSummary.mockReturnValue(
      result({
        data: summary({
          processedToday: { status: 'ok', count: 0 },
          inFlight: { status: 'ok', count: 0 },
        }),
      })
    );
    render(<HomeReadoutBand />);
    expect(screen.getByTestId('readout-processed-value')).toHaveTextContent('0 部');
    expect(screen.getByTestId('readout-inflight-value')).toHaveTextContent('0 個任務');
  });

  it('[P2] loading renders the band-shaped skeleton; whole-request error renders NOTHING (fail-soft)', () => {
    mockUseHomeSummary.mockReturnValue(result({ isLoading: true }));
    const { rerender } = render(<HomeReadoutBand />);
    expect(screen.getByTestId('home-readout-skeleton')).toBeInTheDocument();

    mockUseHomeSummary.mockReturnValue(result({ isError: true }));
    rerender(<HomeReadoutBand />);
    expect(screen.queryByTestId('home-readout-band')).toBeNull();
    expect(screen.queryByTestId('home-readout-skeleton')).toBeNull();
  });

  /**
   * Motion licence ③ (styles.css):「動的東西＝正在發生的事」. The breath is a
   * CLAIM that jobs are running, so it is bound to the same honesty contract
   * as the colour vocabulary — a still 進行中 cell and a moving one must mean
   * different things, and only the backend gets to say which.
   */
  describe('[P1] 進行中 breathes only while something is actually in flight', () => {
    const icon = () => screen.getByTestId('readout-inflight').querySelector('svg')!;

    it('breathes when the count is above zero', () => {
      mockUseHomeSummary.mockReturnValue(result({ data: summary() })); // count: 2
      render(<HomeReadoutBand />);
      expect(icon().getAttribute('class')).toContain('motion-safe:animate-breathe');
    });

    it('is perfectly still at 0 — the stillness IS the readout', () => {
      mockUseHomeSummary.mockReturnValue(
        result({ data: summary({ inFlight: { status: 'ok', count: 0 } }) })
      );
      render(<HomeReadoutBand />);
      expect(icon().getAttribute('class')).not.toContain('animate-breathe');
    });

    it('is still when the count is UNMEASURABLE — unknown is not the same as busy', () => {
      mockUseHomeSummary.mockReturnValue(
        result({ data: summary({ inFlight: { status: 'unavailable' } }) })
      );
      render(<HomeReadoutBand />);
      expect(icon().getAttribute('class')).not.toContain('animate-breathe');
    });

    it('no OTHER cell ever breathes — three of the four are settled facts', () => {
      mockUseHomeSummary.mockReturnValue(result({ data: summary() }));
      render(<HomeReadoutBand />);
      for (const id of ['readout-coverage', 'readout-processed', 'readout-attention']) {
        expect(screen.getByTestId(id).innerHTML, `${id} must not move`).not.toContain(
          'animate-breathe'
        );
      }
    });
  });
});
