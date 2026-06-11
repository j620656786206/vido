import { describe, it, expect } from 'vitest';
import { formatVoteCount } from './formatVoteCount';

describe('formatVoteCount', () => {
  it('shows counts under 1000 as-is', () => {
    expect(formatVoteCount(856)).toBe('856');
    expect(formatVoteCount(1)).toBe('1');
    expect(formatVoteCount(999)).toBe('999');
  });

  it('abbreviates thousands with one decimal under 10k', () => {
    expect(formatVoteCount(1200)).toBe('1.2k');
    expect(formatVoteCount(9900)).toBe('9.9k');
  });

  it('abbreviates thousands as integers at/above 10k', () => {
    expect(formatVoteCount(15000)).toBe('15k');
    expect(formatVoteCount(150000)).toBe('150k');
  });

  it('abbreviates millions', () => {
    expect(formatVoteCount(1_300_000)).toBe('1.3M');
    expect(formatVoteCount(2_130_000)).toBe('2.1M');
    expect(formatVoteCount(15_000_000)).toBe('15M');
  });

  it('trims trailing .0', () => {
    expect(formatVoteCount(2000)).toBe('2k');
    expect(formatVoteCount(3_000_000)).toBe('3M');
  });

  it('promotes the sub-million boundary to M instead of "1000k"', () => {
    expect(formatVoteCount(999_500)).toBe('1M');
    expect(formatVoteCount(999_999)).toBe('1M');
    expect(formatVoteCount(995_000)).toBe('995k');
  });

  it('handles zero and invalid input', () => {
    expect(formatVoteCount(0)).toBe('0');
    expect(formatVoteCount(-5)).toBe('0');
    expect(formatVoteCount(NaN)).toBe('0');
  });
});
