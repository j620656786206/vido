import { render, screen, within } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { DetailTechInfoV2 } from './DetailTechInfoV2';

describe('DetailTechInfoV2', () => {
  it('renders tech badges and the size / subtitle-track / path fact rows', () => {
    render(
      <DetailTechInfoV2
        videoResolution="1080p"
        videoCodec="HEVC"
        audioCodec="DTS"
        audioChannels={6}
        subtitleTracks={JSON.stringify([{ language: 'zh-Hant' }])}
        fileSize={3 * 1024 ** 3}
        filePath="/media/movies/yourname.mkv"
      />
    );
    const section = screen.getByTestId('detail-tech-info');
    expect(section).toHaveTextContent('1080p');
    expect(section).toHaveTextContent('HEVC');
    expect(section).toHaveTextContent('DTS 6ch');
    expect(section).toHaveTextContent('3.0 GB');
    expect(section).toHaveTextContent('/media/movies/yourname.mkv');
    // B3p-D labels the three facts 檔案大小／字幕軌／路徑.
    expect(within(section).getByText('檔案大小')).toBeInTheDocument();
    expect(within(section).getByText('路徑')).toBeInTheDocument();
    expect(screen.getByTestId('detail-subtitle-tracks')).toHaveTextContent('繁中');
  });

  // dsr-2 AC #4 (P0): tech badges are file attributes, not events — neutral, pill-shaped.
  it('renders tech badges neutral — no status tint, pill radius', () => {
    render(<DetailTechInfoV2 videoResolution="2160p" videoCodec="HEVC" hdrFormat="HDR10" />);
    const badges = screen.getAllByTestId('detail-tech-badge');
    expect(badges).toHaveLength(3);
    for (const badge of badges) {
      expect(badge.className).not.toMatch(/-tint/);
      expect(badge.className).toContain('bg-[var(--bg-tertiary)]');
      expect(badge.className).toContain('text-[var(--text-secondary)]');
      expect(badge.className).toContain('rounded-full');
    }
  });

  // The hero already carries the subtitle STATUS badge; this block states the tracks
  // as a fact, so the row must not wear a status tint.
  it('states subtitle tracks as a plain fact row, not a status-coloured pill', () => {
    render(<DetailTechInfoV2 subtitleTracks={JSON.stringify([{ language: 'zh-Hant' }])} />);
    const row = screen.getByTestId('detail-subtitle-tracks');
    expect(row.innerHTML).not.toMatch(/-tint/);
    expect(screen.queryAllByTestId('detail-tech-badge')).toHaveLength(0);
  });

  it('labels tracks, de-duplicates, and keeps chi/zho raw — no script guess from a bare language code', () => {
    render(
      <DetailTechInfoV2
        subtitleTracks={JSON.stringify([
          { language: 'zh-TW' },
          { language: 'zh-CN' },
          { language: 'eng' },
          { language: 'en' },
          { language: 'chi' },
          { language: 'und' },
          { lang: 'jpn' },
        ])}
      />
    );
    expect(screen.getByTestId('detail-subtitle-tracks')).toHaveTextContent(
      '繁中 · 簡中 · 英文 · chi · 未標示 · jpn'
    );
  });

  it('hides the subtitle-track row when there are no tracks or the value is not JSON', () => {
    const { rerender } = render(<DetailTechInfoV2 subtitleTracks="[]" filePath="/media/a.mkv" />);
    expect(screen.queryByTestId('detail-subtitle-tracks')).not.toBeInTheDocument();
    rerender(<DetailTechInfoV2 subtitleTracks="legacy-not-json" filePath="/media/a.mkv" />);
    expect(screen.queryByTestId('detail-subtitle-tracks')).not.toBeInTheDocument();
  });

  it('renders nothing when there is no tech data', () => {
    const { container } = render(<DetailTechInfoV2 />);
    expect(container).toBeEmptyDOMElement();
  });

  // The canonical HANT set (libraryStatus.ts, shared with every subtitle surface)
  // counts a bare `zh` as 繁中. Documented here so nobody reads the chi/zho test above
  // as "this row never maps a bare zh" (review #7).
  it('follows the shared HANT set: a bare zh reads 繁中', () => {
    render(<DetailTechInfoV2 subtitleTracks={JSON.stringify([{ language: 'zh' }])} />);
    expect(screen.getByTestId('detail-subtitle-tracks')).toHaveTextContent('繁中');
  });
});
