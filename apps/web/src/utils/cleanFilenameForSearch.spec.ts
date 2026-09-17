import { describe, it, expect } from 'vitest';
import { cleanFilenameForSearch } from './cleanFilenameForSearch';

describe('cleanFilenameForSearch', () => {
  it.each([
    // fansub groups and bracketed tags
    ['[Leopard-Raws] Kimi no Na wa (BD).mkv', 'Kimi no Na wa'],
    // scene release names: cut at the first release token, drop the trailing year
    ['The.Matrix.1999.1080p.BluRay.x264-SPARKS.mkv', 'The Matrix'],
    ['Inception.2010.1080p.BluRay.HDR.x265-GROUP.mkv', 'Inception'],
    ['Dune.Part.Two.2024.2160p.WEB-DL.DDP5.1.H.265-FLUX.mkv', 'Dune Part Two'],
    ['Spirited_Away_2160p_WEB-DL_HEVC.mp4', 'Spirited Away'],
    // words that are also release tags stay when they are part of the title
    ["Charlotte's.Web.2006.1080p.mkv", "Charlotte's Web"],
    ['Spider-Man.No.Way.Home.2021.2160p.mkv', 'Spider-Man No Way Home'],
    // a title that is a year keeps it; only a TRAILING year after words goes
    ['1917.2019.1080p.mkv', '1917'],
    ['Blade.Runner.2049.2017.1080p.mkv', 'Blade Runner 2049'],
    // episodes / season packs
    ['Mr.Robot.S01E01.720p.mkv', 'Mr Robot'],
    // Chinese and mixed names
    ['你的名字.mkv', '你的名字'],
    ['你的名字', '你的名字'],
    ['你的名字.Your.Name.2016.1080p.mkv', '你的名字 Your Name'],
  ])('%s → %s', (input, expected) => {
    expect(cleanFilenameForSearch(input)).toBe(expected);
  });

  it('only strips real video extensions, never the end of a title', () => {
    expect(cleanFilenameForSearch('The.Last.of.Us')).toBe('The Last of Us');
  });

  it('uses only the last path segment', () => {
    expect(cleanFilenameForSearch('/volume1/Movies/Anime/Tenki.no.Ko.2019.mkv')).toBe(
      'Tenki no Ko'
    );
  });

  it('falls back to the bare name when cleaning leaves nothing', () => {
    expect(cleanFilenameForSearch('[x264].mkv')).toBe('[x264]');
  });
});
