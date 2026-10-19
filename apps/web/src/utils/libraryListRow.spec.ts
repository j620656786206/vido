import { describe, it, expect } from 'vitest';
import { formatSpecLine, fileNameOf, splitFileNameTail } from './libraryListRow';

describe('formatSpecLine', () => {
  it('joins tier, video codec and audio codec with a middle dot', () => {
    expect(
      formatSpecLine({ videoResolution: '3840x2076', videoCodec: 'H.265', audioCodec: 'DTS' })
    ).toBe('4K · H.265 · DTS');
  });

  it('skips the parts the file did not tell us', () => {
    expect(formatSpecLine({ audioCodec: 'AAC' })).toBe('AAC');
    expect(formatSpecLine({ videoResolution: '1920x1080', audioCodec: 'AC-3' })).toBe(
      '1080p · AC-3'
    );
  });

  it('returns null when there is nothing to show', () => {
    expect(formatSpecLine({})).toBeNull();
    expect(formatSpecLine({ videoResolution: '', videoCodec: ' ', audioCodec: '' })).toBeNull();
  });
});

describe('fileNameOf', () => {
  it('takes the last segment of a POSIX path', () => {
    expect(fileNameOf('/media/movies/Reze/[Itachi] Chainsaw Man – Reze-hen.mkv', 't')).toBe(
      '[Itachi] Chainsaw Man – Reze-hen.mkv'
    );
  });

  it('takes the last segment of a Windows path', () => {
    expect(fileNameOf('D:\\movies\\Film.2025.mkv', 't')).toBe('Film.2025.mkv');
  });

  it('keeps a series folder name (no extension) and ignores a trailing slash', () => {
    expect(fileNameOf('/media/tv/Some Show (2024)/', 't')).toBe('Some Show (2024)');
  });

  it('falls back to the title when there is no path', () => {
    expect(fileNameOf(undefined, '原始標題')).toBe('原始標題');
    expect(fileNameOf('', '原始標題')).toBe('原始標題');
  });
});

describe('splitFileNameTail', () => {
  it('keeps the last 12 characters as the tail', () => {
    expect(splitFileNameTail('[ItachiUchiha] Chainsaw Man – The Movie – Reze-hen.mkv')).toEqual({
      head: '[ItachiUchiha] Chainsaw Man – The Movie – ',
      tail: 'Reze-hen.mkv',
    });
  });

  it('puts a short name entirely in the tail', () => {
    expect(splitFileNameTail('short.mkv')).toEqual({ head: '', tail: 'short.mkv' });
    expect(splitFileNameTail('exactly12chr')).toEqual({ head: '', tail: 'exactly12chr' });
  });

  it('counts code points, not UTF-16 units', () => {
    const { head, tail } = splitFileNameTail('電影🎬名稱一二三四五六七八九十.mkv');
    expect(tail).toBe('三四五六七八九十.mkv');
    expect(head).toBe('電影🎬名稱一二');
  });
});
