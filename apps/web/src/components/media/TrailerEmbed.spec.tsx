import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { TrailerEmbed } from './TrailerEmbed';

describe('TrailerEmbed', () => {
  it('renders button initially, not iframe', () => {
    render(<TrailerEmbed videoKey="abc123" title="Test Movie" />);

    expect(screen.getByTestId('trailer-button')).toBeInTheDocument();
    expect(screen.getByText('觀看預告片')).toBeInTheDocument();
    expect(screen.queryByTestId('trailer-player')).not.toBeInTheDocument();
  });

  it('shows iframe with youtube-nocookie after clicking button', () => {
    render(<TrailerEmbed videoKey="abc123" title="Test Movie" />);

    fireEvent.click(screen.getByTestId('trailer-button'));

    expect(screen.getByTestId('trailer-player')).toBeInTheDocument();
    const iframe = screen.getByTitle('Test Movie 預告片');
    expect(iframe).toBeInTheDocument();
    expect(iframe).toHaveAttribute('src', 'https://www.youtube-nocookie.com/embed/abc123');
  });

  it('uses privacy-enhanced mode (youtube-nocookie)', () => {
    render(<TrailerEmbed videoKey="xyz789" title="Privacy Test" />);
    fireEvent.click(screen.getByTestId('trailer-button'));

    const iframe = screen.getByTitle('Privacy Test 預告片');
    expect(iframe.getAttribute('src')).toContain('youtube-nocookie.com');
    expect(iframe.getAttribute('src')).not.toContain('youtube.com/embed');
  });

  it('has responsive aspect ratio container', () => {
    render(<TrailerEmbed videoKey="abc123" title="Test" />);
    fireEvent.click(screen.getByTestId('trailer-button'));

    const container = screen.getByTestId('trailer-player');
    expect(container.className).toContain('aspect-video');
  });

  it('[P1] iframe has correct allow attributes for security', () => {
    render(<TrailerEmbed videoKey="sec-test" title="Security Test" />);
    fireEvent.click(screen.getByTestId('trailer-button'));

    const iframe = screen.getByTitle('Security Test 預告片');
    expect(iframe).toHaveAttribute(
      'allow',
      'accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope'
    );
    expect(iframe).toHaveAttribute('allowFullScreen');
  });

  it('[P1] constructs correct title with zh-TW suffix', () => {
    render(<TrailerEmbed videoKey="key1" title="蜘蛛人" />);
    fireEvent.click(screen.getByTestId('trailer-button'));

    expect(screen.getByTitle('蜘蛛人 預告片')).toBeInTheDocument();
  });

  it('[P1] button hides after iframe is shown', () => {
    render(<TrailerEmbed videoKey="abc123" title="Test" />);
    fireEvent.click(screen.getByTestId('trailer-button'));

    expect(screen.queryByTestId('trailer-button')).not.toBeInTheDocument();
    expect(screen.getByTestId('trailer-player')).toBeInTheDocument();
  });

  it('[P2] embeds different video keys correctly', () => {
    render(<TrailerEmbed videoKey="DIFFERENT_KEY_123" title="Diff" />);
    fireEvent.click(screen.getByTestId('trailer-button'));

    const iframe = screen.getByTitle('Diff 預告片');
    expect(iframe).toHaveAttribute(
      'src',
      'https://www.youtube-nocookie.com/embed/DIFFERENT_KEY_123'
    );
  });
});
