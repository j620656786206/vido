import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { SpendCard, SpendSectionView, spendHasContent, spendMonthLabel } from './SpendSection';
import type { SubtitleSpendSummary } from '../../services/subtitleSpendService';

/** The K5-D numbers (xgYKA) — the "everything measured" normal state. */
function month(over: Partial<SubtitleSpendSummary> = {}): SubtitleSpendSummary {
  return {
    period: 'month',
    from: '2026-10-01T00:00:00+08:00',
    to: '2026-11-01T00:00:00+08:00',
    translatedRuns: 12,
    translatedUsd: 3.48,
    asrRuns: 2,
    asrUsd: 1.9,
    unpricedRuns: 0,
    unroutedRuns: 0,
    unroutedUsd: 0,
    skippedDeliverCount: 8,
    skippedSavedUsdEstimate: 2.14,
    skippedSavedRuntimeAssumed: false,
    cacheHitCues: 101,
    cacheMeasuredRuns: 12,
    cacheSavedUsdEstimate: 0.31,
    byModel: [
      { modelId: 'claude-sonnet-5', runs: 9, usd: 4.2 },
      { modelId: 'claude-haiku-4-5', runs: 5, usd: 1.18 },
    ],
    ...over,
  };
}

describe('SpendCard — 本月 AI 花費 (sub-7-6c AC #1/#4, K5-D xgYKA)', () => {
  it('[P1] draws the two paid lanes, both savings and the per-model table', () => {
    render(<SpendCard summary={month()} />);
    expect(screen.getByTestId('activity-spend-translated')).toHaveTextContent('翻譯');
    expect(screen.getByTestId('activity-spend-translated')).toHaveTextContent('12 次 · $3.48');
    expect(screen.getByTestId('activity-spend-asr')).toHaveTextContent('語音辨識');
    expect(screen.getByTestId('activity-spend-asr')).toHaveTextContent('2 次 · $1.90');
    expect(screen.getByTestId('activity-spend-skipped')).toHaveTextContent(
      '略過 8 集，估算省下 $2.14'
    );
    expect(screen.getByTestId('activity-spend-cache')).toHaveTextContent('快取估算省下 $0.31');
    const rows = screen.getAllByTestId('activity-spend-model-row');
    expect(rows).toHaveLength(2);
    expect(rows[0]).toHaveTextContent('claude-sonnet-5');
    expect(rows[0]).toHaveTextContent('9 次 · $4.20');
    expect(rows[1]).toHaveTextContent('claude-haiku-4-5');
    expect(rows[1]).toHaveTextContent('5 次 · $1.18');
    // Nothing unrecorded this month → no footnote claiming otherwise.
    expect(screen.queryByTestId('activity-spend-footnote')).toBeNull();
  });

  it('[P1] a lane with no runs reads 「—」, never $0.00 (absent ≠ $0)', () => {
    render(<SpendCard summary={month({ asrRuns: 0, asrUsd: 0 })} />);
    expect(screen.getByTestId('activity-spend-asr')).toHaveTextContent(/—$/);
    expect(screen.getByTestId('activity-spend-asr')).not.toHaveTextContent('$0.00');
  });

  it('[P1] a cache no run measured reads 「—」; a measured one shows the estimate', () => {
    render(
      <SpendCard
        summary={month({ cacheMeasuredRuns: 0, cacheHitCues: 0, cacheSavedUsdEstimate: null })}
      />
    );
    expect(screen.getByTestId('activity-spend-cache')).toHaveTextContent(/快取估算省下 —$/);
    expect(screen.getByTestId('activity-spend-cache')).not.toHaveTextContent('$0');
  });

  // ⚖️ Sally 2026-09-05 / 7-6 AA: ≈ means ONE thing — the amount rests on an
  // assumed runtime. The word 估算 is always there; ≈ only when the flag is.
  it('[P1] ≈ appears on the skipped saving ONLY when a runtime was assumed', () => {
    const { rerender } = render(<SpendCard summary={month()} />);
    expect(screen.getByTestId('activity-spend-skipped')).not.toHaveTextContent('≈');

    rerender(<SpendCard summary={month({ skippedSavedRuntimeAssumed: true })} />);
    expect(screen.getByTestId('activity-spend-skipped')).toHaveTextContent(
      '略過 8 集，估算省下 ≈ $2.14'
    );
    // The cache saving never wears it — its estimate has nothing to do with runtime.
    expect(screen.getByTestId('activity-spend-cache')).not.toHaveTextContent('≈');
  });

  it('[P2] nothing skipped → no skipped line; the cache line stays', () => {
    render(<SpendCard summary={month({ skippedDeliverCount: 0, skippedSavedUsdEstimate: 0 })} />);
    expect(screen.queryByTestId('activity-spend-skipped')).toBeNull();
    expect(screen.getByTestId('activity-spend-cache')).toBeInTheDocument();
  });

  it('[P1] unpriced and pre-ledger rows go to the footnote, not the lane totals', () => {
    render(<SpendCard summary={month({ unpricedRuns: 2, unroutedRuns: 3, unroutedUsd: 0.77 })} />);
    const note = screen.getByTestId('activity-spend-footnote');
    expect(note).toHaveTextContent('另有 2 次沒記到金額');
    expect(note).toHaveTextContent('舊紀錄 3 次 · $0.77，未分翻譯／語音辨識');
    // The headline lanes are untouched by either.
    expect(screen.getByTestId('activity-spend-translated')).toHaveTextContent('12 次 · $3.48');
  });

  it('[P2] no model rows → no 依模型 block', () => {
    render(<SpendCard summary={month({ byModel: [] })} />);
    expect(screen.queryByTestId('activity-spend-by-model')).toBeNull();
    expect(screen.queryByText('依模型')).toBeNull();
  });
});

describe('spendMonthLabel / spendHasContent', () => {
  // Rule 23: the label is the SERVER's month (its `from`), not the browser clock.
  it('reads the month from `from` in the server zone — no wall-clock read', () => {
    expect(spendMonthLabel({ from: '2026-10-01T00:00:00+08:00' })).toBe('10 月');
    // A UTC browser would say 9 月 for this instant; the server said October.
    expect(spendMonthLabel({ from: '2026-10-01T00:00:00+08:00' })).not.toBe('9 月');
    expect(spendMonthLabel({ from: '2026-01-01T00:00:00Z' })).toBe('1 月');
    expect(spendMonthLabel({ from: 'garbage' })).toBe('');
  });

  it('an untouched month has no content; any recorded thing does', () => {
    const empty = month({
      translatedRuns: 0,
      asrRuns: 0,
      unpricedRuns: 0,
      unroutedRuns: 0,
      skippedDeliverCount: 0,
      cacheMeasuredRuns: 0,
      byModel: [],
    });
    expect(spendHasContent(empty)).toBe(false);
    expect(spendHasContent({ ...empty, skippedDeliverCount: 1 })).toBe(true);
    expect(spendHasContent({ ...empty, unroutedRuns: 1 })).toBe(true);
    expect(spendHasContent({ ...empty, cacheMeasuredRuns: 1 })).toBe(true);
  });
});

describe('SpendSectionView — the hub section around the card', () => {
  it('[P1] renders the title with the month at the right, and the card', () => {
    render(<SpendSectionView summary={month()} failed={false} onRetry={vi.fn()} />);
    expect(screen.getByRole('heading', { name: '本月 AI 花費' })).toBeInTheDocument();
    expect(screen.getByTestId('activity-spend-month')).toHaveTextContent('10 月');
    expect(screen.getByTestId('activity-spend-card')).toBeInTheDocument();
  });

  it('[P1] fail-soft: a failed endpoint shows its own 無法載入 + 重試, nothing else', () => {
    const onRetry = vi.fn();
    render(<SpendSectionView summary={undefined} failed onRetry={onRetry} />);
    expect(screen.getByTestId('activity-spend-error')).toBeInTheDocument();
    expect(screen.queryByTestId('activity-spend-card')).toBeNull();
    fireEvent.click(screen.getByTestId('activity-section-retry'));
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it('[P2] an untouched month renders no section at all', () => {
    const { container } = render(
      <SpendSectionView
        summary={month({
          translatedRuns: 0,
          asrRuns: 0,
          skippedDeliverCount: 0,
          cacheMeasuredRuns: 0,
          byModel: [],
        })}
        failed={false}
        onRetry={vi.fn()}
      />
    );
    expect(container).toBeEmptyDOMElement();
  });
});
