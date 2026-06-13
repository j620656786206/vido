import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { DetailTechInfoV2 } from './DetailTechInfoV2';

describe('DetailTechInfoV2', () => {
  it('renders tech badges, subtitle pill and size/path fact rows', () => {
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
    expect(section).toHaveTextContent('繁中');
    expect(section).toHaveTextContent('3.0 GB');
    expect(section).toHaveTextContent('/media/movies/yourname.mkv');
  });

  it('renders nothing when there is no tech data', () => {
    const { container } = render(<DetailTechInfoV2 />);
    expect(container).toBeEmptyDOMElement();
  });
});
