import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ConsentEmptyState } from './ConsentEmptyState';

/**
 * This screen shipped a false claim for months and a homepage critique caught
 * it: the phase means「0 listable candidates」, but the copy asserted
 *「所有影片都有繁中字幕了」 — a claim only ONE of the two possible causes
 * supports. Verified live: analyzed 15/15, candidates [], against 18 titles
 * with 0 covered, and the sentence was printed anyway, in GREEN.
 *
 * These guard the direction that lies. An unknown reason must never render as
 * completion.
 */
describe('ConsentEmptyState', () => {
  it('[P0] with no props it makes NO coverage claim — the default must not lie', () => {
    render(<ConsentEmptyState />);
    expect(screen.queryByText('所有影片都有繁中字幕了')).toBeNull();
    expect(screen.getByText('沒有找到可以產生字幕的項目')).toBeInTheDocument();
    expect(screen.getByTestId('consent-empty-state')).toHaveAttribute(
      'data-empty-reason',
      'no-candidates'
    );
  });

  it('[P0] the green check is EARNED — it appears only when everything really is covered', () => {
    const { container: notCovered } = render(<ConsentEmptyState allCovered={false} />);
    expect(notCovered.querySelector('.text-\\[var\\(--success-text\\)\\]')).toBeNull();

    const { container: covered } = render(<ConsentEmptyState allCovered />);
    expect(covered.querySelector('.text-\\[var\\(--success-text\\)\\]')).not.toBeNull();
    expect(screen.getByText('所有影片都有繁中字幕了')).toBeInTheDocument();
  });

  it('[P1] reports what was actually analysed instead of asserting a conclusion', () => {
    render(<ConsentEmptyState analyzed={15} />);
    expect(screen.getByText(/分析了 15 部/)).toBeInTheDocument();
    // The most common real cause of a non-empty library landing here.
    expect(screen.getByText(/影集的語音辨識還在開發中/)).toBeInTheDocument();
  });

  it('[P2] omits the count rather than printing a fake 0 when it is unknown', () => {
    render(<ConsentEmptyState />);
    expect(screen.queryByText(/分析了 0 部/)).toBeNull();
    expect(screen.getByText(/這次分析沒有可以抽取內嵌字幕/)).toBeInTheDocument();
  });
});
