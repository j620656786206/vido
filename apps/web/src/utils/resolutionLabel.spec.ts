import { describe, it, expect } from 'vitest';
import { resolutionTier, audioChannelLabel } from './resolutionLabel';

describe('resolutionTier', () => {
  it.each([
    // ffprobe frame sizes — the ones the old exact-match table could not read
    ['3840x2160', '4K'],
    ['3840x2076', '4K'],
    ['3840x1608', '4K'],
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

describe('audioChannelLabel', () => {
  it('maps 2 to Stereo', () => {
    expect(audioChannelLabel(2)).toBe('Stereo');
  });

  it('maps 6 to 5.1', () => {
    expect(audioChannelLabel(6)).toBe('5.1');
  });

  it('maps 8 to 7.1', () => {
    expect(audioChannelLabel(8)).toBe('7.1');
  });

  it('maps 1 to Mono', () => {
    expect(audioChannelLabel(1)).toBe('Mono');
  });

  it('returns the raw number for unknown channel counts', () => {
    expect(audioChannelLabel(4)).toBe('4');
    expect(audioChannelLabel(0)).toBe('0');
  });
});
