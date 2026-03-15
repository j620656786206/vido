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
});
