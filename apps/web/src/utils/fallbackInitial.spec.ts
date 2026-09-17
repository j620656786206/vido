import { describe, it, expect } from 'vitest';
import { fallbackInitial } from './fallbackInitial';

describe('fallbackInitial', () => {
  it('skips leading brackets and symbols to the first letter', () => {
    expect(fallbackInitial('[FanSub] 未知電影')).toBe('F');
    expect(fallbackInitial('[Leopard-Raws] Kimi no Na wa (BD).mkv')).toBe('L');
  });

  it('takes a CJK character as a letter', () => {
    expect(fallbackInitial('你的名字')).toBe('你');
  });

  it('falls back to the raw first char for an all-symbol title, and ? for empty', () => {
    expect(fallbackInitial('###')).toBe('#');
    expect(fallbackInitial('')).toBe('?');
  });
});
