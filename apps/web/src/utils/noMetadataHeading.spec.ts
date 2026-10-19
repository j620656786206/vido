import { describe, it, expect } from 'vitest';
import { noMetadataHeading } from './noMetadataHeading';

describe('noMetadataHeading', () => {
  it('names a film 電影', () => {
    expect(noMetadataHeading('movie')).toBe('沒有找到這部電影的資料');
  });

  it('names a series 影集, whichever spelling the caller uses', () => {
    expect(noMetadataHeading('series')).toBe('沒有找到這部影集的資料');
    expect(noMetadataHeading('tv')).toBe('沒有找到這部影集的資料');
  });
});
