import { describe, it, expect } from 'vitest';
import { usd, sumUsd, addUsd, subUsd, gtUsd, ltUsd, percentOfUsd, roundUsd } from './currency';

describe('usd (display)', () => {
  it('renders two decimal places verbatim', () => {
    expect(usd(4.5)).toBe('$4.50');
    expect(usd(0)).toBe('$0.00');
    expect(usd(25.8)).toBe('$25.80');
  });

  it("rounds half UP, the rule an invoice states — not toFixed's binary accident", () => {
    // (1.005).toFixed(2) === "1.00" in V8, because the double behind the
    // literal 1.005 is 1.00499999…. Reading the value as the decimal it was
    // written as gives the answer a person doing it by hand would.
    expect(usd(1.005)).toBe('$1.01');
    expect(usd(2.675)).toBe('$2.68');
  });
});

describe('sumUsd (the invariant this module exists for)', () => {
  it('0.1 + 0.2 is 0.3, not 0.30000000000000004', () => {
    expect(sumUsd([0.1, 0.2])).toBe(0.3);
    expect(0.1 + 0.2).not.toBe(0.3); // …which is what the naive version does
  });

  it('the displayed parts always add up to the displayed total', () => {
    // The concrete screen bug: two components that each round UP while their
    // exact sum rounds DOWN. Adding the raw floats and formatting each
    // independently prints 「$0.01 + $0.01 = $0.01」.
    const extract = 0.005;
    const asr = 0.005;
    const total = sumUsd([extract, asr]);
    expect(usd(extract)).toBe('$0.01');
    expect(usd(asr)).toBe('$0.01');
    expect(usd(total)).toBe('$0.01');
    // …so the honest fix is to round the PARTS first and sum those.
    const parts = [roundUsd(extract), roundUsd(asr)];
    expect(usd(sumUsd(parts))).toBe('$0.02');
    expect(usd(parts[0])).toBe('$0.01');
    expect(usd(parts[1])).toBe('$0.01');
  });

  it('stays exact across a library-sized sum', () => {
    // 1,200 items at one cent each. A float accumulator lands on
    // 11.999999999999831; the total a user is asked to approve must not
    // depend on how many rows happened to be in the list.
    const rows = Array.from({ length: 1200 }, () => 0.01);
    expect(sumUsd(rows)).toBe(12);
    expect(rows.reduce((a, b) => a + b, 0)).not.toBe(12);
  });

  it('is empty-safe', () => {
    expect(sumUsd([])).toBe(0);
  });
});

describe('addUsd / subUsd', () => {
  it('add and subtract without drift', () => {
    expect(addUsd(0.07, 1.63)).toBe(1.7);
    expect(subUsd(0.3, 0.1)).toBe(0.2);
    expect(0.3 - 0.1).not.toBe(0.2);
  });
});

describe('gtUsd / ltUsd (the budget verdict)', () => {
  it('does not flip a ceiling on representation error', () => {
    // The float sum of 0.1 + 0.7 is 0.7999999999999999, so a naive
    // `total > 0.8` reads false and the run bills past the approved ceiling.
    const total = sumUsd([0.1, 0.7]);
    expect(total).toBe(0.8);
    expect(gtUsd(total, 0.8)).toBe(false);
    expect(ltUsd(total, 0.8)).toBe(false);
    expect(gtUsd(total, 0.79)).toBe(true);
  });
});

describe('percentOfUsd', () => {
  it('reports a whole percent of the reference amount', () => {
    expect(percentOfUsd(2.8, 4.5)).toBe(62);
    expect(percentOfUsd(0.22, 0.36)).toBe(61);
  });

  it('takes the magnitude, so a dearer option reports its premium', () => {
    expect(percentOfUsd(-1.87, 0.53)).toBe(353);
  });

  it('refuses to divide by zero rather than printing ∞%', () => {
    expect(percentOfUsd(1, 0)).toBeUndefined();
  });
});

describe('roundUsd', () => {
  it('rounds to whole cents, half up', () => {
    expect(roundUsd(0.005)).toBe(0.01);
    expect(roundUsd(1.004)).toBe(1);
    expect(roundUsd(2.345)).toBe(2.35);
  });
});
