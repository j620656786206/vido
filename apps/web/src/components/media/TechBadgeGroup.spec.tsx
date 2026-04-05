import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { TechBadgeGroup } from './TechBadgeGroup';

describe('TechBadgeGroup', () => {
  it('renders all badge types when all fields provided', () => {
    render(
      <TechBadgeGroup
        videoCodec="H.265"
        videoResolution="3840x2160"
        audioCodec="DTS"
        audioChannels={6}
        hdrFormat="HDR10"
        subtitleTracks='[{"lang":"zh"},{"lang":"en"},{"lang":"ja"}]'
      />
    );

    expect(screen.getByText('H.265')).toBeInTheDocument();
    expect(screen.getByText('4K')).toBeInTheDocument();
    expect(screen.getByText('DTS 5.1')).toBeInTheDocument();
    expect(screen.getByText('HDR10')).toBeInTheDocument();
    expect(screen.getByText('3 字幕')).toBeInTheDocument();
  });

  it('renders only video badges when only video fields provided', () => {
    render(<TechBadgeGroup videoCodec="H.264" videoResolution="1920x1080" />);

    expect(screen.getByText('H.264')).toBeInTheDocument();
    expect(screen.getByText('1080p')).toBeInTheDocument();

    const badges = screen.getAllByTestId('tech-badge');
    expect(badges).toHaveLength(2);
  });

  it('returns null when all fields are null/undefined', () => {
    const { container } = render(<TechBadgeGroup />);
    expect(container.firstChild).toBeNull();
  });

  it('shows audio codec without channels when channels not provided', () => {
    render(<TechBadgeGroup audioCodec="AAC" />);
    expect(screen.getByText('AAC')).toBeInTheDocument();
  });

  it('formats audio channels correctly', () => {
    render(<TechBadgeGroup audioCodec="DTS" audioChannels={8} />);
    expect(screen.getByText('DTS 7.1')).toBeInTheDocument();
  });

  it('handles non-JSON subtitle tracks gracefully', () => {
    render(<TechBadgeGroup subtitleTracks="zh-Hant" />);
    expect(screen.getByText('zh-Hant')).toBeInTheDocument();
  });

  it('does not render subtitle badge for empty tracks array', () => {
    render(<TechBadgeGroup subtitleTracks="[]" videoCodec="H.265" />);
    const badges = screen.getAllByTestId('tech-badge');
    expect(badges).toHaveLength(1);
  });
});
