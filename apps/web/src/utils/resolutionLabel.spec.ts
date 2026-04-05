import { describe, it, expect } from 'vitest';
import { resolutionLabel, audioChannelLabel } from './resolutionLabel';

describe('resolutionLabel', () => {
  it('maps 3840x2160 to 4K', () => {
    expect(resolutionLabel('3840x2160')).toBe('4K');
  });

  it('maps 2560x1440 to 2K', () => {
    expect(resolutionLabel('2560x1440')).toBe('2K');
  });

  it('maps 1920x1080 to 1080p', () => {
    expect(resolutionLabel('1920x1080')).toBe('1080p');
  });

  it('maps 1280x720 to 720p', () => {
    expect(resolutionLabel('1280x720')).toBe('720p');
  });

  it('returns raw value for unknown resolution', () => {
    expect(resolutionLabel('720x480')).toBe('720x480');
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

  it('returns raw number for unknown channels', () => {
    expect(audioChannelLabel(4)).toBe('4');
  });
});
