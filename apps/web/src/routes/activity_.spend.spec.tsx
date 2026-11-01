import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { Route } from './activity_.spend';

type Validate = (search: Record<string, unknown>) => { month?: string; category?: string };
const validate = Route.options.validateSearch as unknown as Validate;

describe('/activity/spend search params (disc-2026-11-spend-history-d)', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(2026, 10, 15, 12, 0, 0)); // 2026-11-15 local
  });
  afterEach(() => vi.useRealTimers());

  it('keeps a past month', () => {
    expect(validate({ month: '2026-09' }).month).toBe('2026-09');
  });

  it('drops this month (the bare URL is this month) and any future month', () => {
    expect(validate({ month: '2026-11' }).month).toBeUndefined();
    expect(validate({ month: '2026-12' }).month).toBeUndefined();
    expect(validate({ month: '2027-01' }).month).toBeUndefined();
  });

  it('drops malformed months and unknown categories (Rule 26: no numeric coercion)', () => {
    expect(validate({ month: 202609 }).month).toBeUndefined();
    expect(validate({ month: '2026-9' }).month).toBeUndefined();
    expect(validate({ category: 'no_subtitle' }).category).toBeUndefined();
    expect(validate({ category: 'asr' }).category).toBe('asr');
  });
});
