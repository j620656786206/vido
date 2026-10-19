import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, it, expect } from 'vitest';
import { resolutionTier, audioChannelLabel, formatResolutionBadge } from './resolutionLabel';

/**
 * The table Go's `TestResolveResolution_SharedTable` walks too
 * (disc-2026-10-resolution-tier-scope-encodes AC #3): one frame size, one tier, on both
 * sides. Change the rule on one side only and that side goes red.
 */
const SHARED_TABLE = join(
  __dirname,
  '..',
  '..',
  '..',
  'api',
  'internal',
  'services',
  'testdata',
  'resolution_tiers.json'
);
const SHARED_ROWS: { width: number; height: number; tier: string; why: string }[] = JSON.parse(
  readFileSync(SHARED_TABLE, 'utf8')
);

describe('resolutionTier — shared table with Go resolveResolution', () => {
  it('reads the whole shared table', () => {
    expect(SHARED_ROWS.length).toBeGreaterThanOrEqual(40);
  });

  it.each(SHARED_ROWS.map((r) => [`${r.width}x${r.height}`, r.tier, r.why]))(
    '%s → %s (%s)',
    (raw, want) => {
      expect(resolutionTier(raw)).toBe(want);
    }
  );
});

describe('resolutionTier', () => {
  it.each([
    // ffprobe frame sizes — the ones the old exact-match table could not read
    ['3840x2160', '4K'],
    ['3840x2076', '4K'],
    ['3840x1608', '4K'],
    // ⚖️ Alexyu 2026-10-19: a scope 4K with a few pixels off the sides is still 4K.
    ['3828x1588', '4K'],
    ['2560x1440', '1440p'],
    ['2560x1080', '1440p'],
    ['1920x1080', '1080p'],
    ['1920x800', '1080p'],
    ['1920x804', '1080p'],
    ['1920x1036', '1080p'],
    ['1280x720', '720p'],
    ['1280x536', '720p'],
    ['720x576', '480p'],
    ['720x480', '480p'],
    ['640x360', '360p'],
  ])('turns the frame size %s into %s', (raw, want) => {
    expect(resolutionTier(raw)).toBe(want);
  });

  it.each([
    ['4K', '4K'],
    ['1440p', '1440p'],
    ['1080p', '1080p'],
    ['720p', '720p'],
    ['SD', 'SD'],
  ])('leaves the NFO label %s as %s', (raw, want) => {
    expect(resolutionTier(raw)).toBe(want);
  });

  it('calls 2160p 4K', () => {
    expect(resolutionTier('2160p')).toBe('4K');
  });

  it('reads a × sign and spaces as a frame size', () => {
    expect(resolutionTier('3840 × 2160')).toBe('4K');
  });

  it('returns an empty string when there is nothing to read', () => {
    expect(resolutionTier('')).toBe('');
    expect(resolutionTier('   ')).toBe('');
    expect(resolutionTier(undefined)).toBe('');
    expect(resolutionTier(null)).toBe('');
    expect(resolutionTier('0x0')).toBe('');
  });
});

// disc-2026-10-detail-resolution-raw AC #1 — the detail page's resolution badge.
describe('formatResolutionBadge', () => {
  it.each([
    ['3840x2160', '4K', '3840×2160', '4K · 3840×2160'],
    ['3840x2076', '4K', '3840×2076', '4K · 3840×2076'],
    ['2880x2160', '4K', '2880×2160', '4K · 2880×2160'],
    // disc-2026-10-resolution-tier-scope-encodes: was 1440p until the scope rule.
    ['3828x1588', '4K', '3828×1588', '4K · 3828×1588'],
    ['1920x804', '1080p', '1920×804', '1080p · 1920×804'],
    ['1280x720', '720p', '1280×720', '720p · 1280×720'],
  ])('writes the frame size %s as tier %s and dims %s', (raw, tier, dims, full) => {
    expect(formatResolutionBadge(raw)).toEqual({ tier, dims, full });
  });

  it.each([
    ['4K', '4K'],
    ['1080p', '1080p'],
    ['2160p', '4K'],
    ['abc', 'abc'],
  ])('shows the label %s as %s with no frame size', (raw, tier) => {
    expect(formatResolutionBadge(raw)).toEqual({ tier, dims: null, full: tier });
  });

  it('normalises an upper-case X and spaces to a bare × with no spaces', () => {
    expect(formatResolutionBadge(' 3840 X 2160 ')).toEqual({
      tier: '4K',
      dims: '3840×2160',
      full: '4K · 3840×2160',
    });
  });

  it('leaves a side of zero out of the frame size', () => {
    expect(formatResolutionBadge('0x1080')).toEqual({ tier: '1080p', dims: null, full: '1080p' });
  });

  it('returns null when there is nothing to show', () => {
    expect(formatResolutionBadge('')).toBeNull();
    expect(formatResolutionBadge('   ')).toBeNull();
    expect(formatResolutionBadge(undefined)).toBeNull();
    expect(formatResolutionBadge(null)).toBeNull();
    expect(formatResolutionBadge('0x0')).toBeNull();
  });
});

// disc-2026-10-detail-resolution-raw AC #3 — speaker layouts, not Mono／Stereo.
describe('audioChannelLabel', () => {
  it.each([
    [1, '1.0'],
    [2, '2.0'],
    [6, '5.1'],
    [8, '7.1'],
  ])('maps %i channels to %s', (n, want) => {
    expect(audioChannelLabel(n)).toBe(want);
  });

  it('keeps the raw count with "ch" for counts with no common layout', () => {
    expect(audioChannelLabel(4)).toBe('4ch');
    expect(audioChannelLabel(3)).toBe('3ch');
  });
});
