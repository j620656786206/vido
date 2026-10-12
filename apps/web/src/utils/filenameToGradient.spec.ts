import { describe, it, expect } from 'vitest';
import { filenameToGradient } from './filenameToGradient';

describe('filenameToGradient', () => {
  it('returns two HSL color strings', () => {
    const [a, b] = filenameToGradient('test-movie.mkv');
    // Lightness stops are part of the contract: clamped dark (26%/32%) so the
    // light initial letter clears 3:1 on every hash-derived hue (critique R1).
    expect(a).toMatch(/^hsl\(\d+, 45%, 26%\)$/);
    expect(b).toMatch(/^hsl\(\d+, 40%, 32%\)$/);
  });

  it('is deterministic — same input produces same output', () => {
    const first = filenameToGradient('kimi-no-na-wa.mkv');
    const second = filenameToGradient('kimi-no-na-wa.mkv');
    expect(first).toEqual(second);
  });

  it('produces different gradients for different filenames', () => {
    const a = filenameToGradient('movie-a.mkv');
    const b = filenameToGradient('movie-b.mkv');
    expect(a).not.toEqual(b);
  });

  it('handles empty string without throwing', () => {
    const [a, b] = filenameToGradient('');
    expect(a).toMatch(/^hsl\(\d+, 45%, 26%\)$/);
    expect(b).toMatch(/^hsl\(\d+, 40%, 32%\)$/);
  });
});
