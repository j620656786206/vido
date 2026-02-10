import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { NewMediaToast } from './NewMediaToast';

describe('NewMediaToast', () => {
  it('[P1] renders media title', () => {
    render(<NewMediaToast title="Test Movie" posterUrl="/poster.jpg" mediaType="movie" />);
    expect(screen.getByText('Test Movie')).toBeTruthy();
  });

  it('[P1] renders success message', () => {
    render(<NewMediaToast title="Test Movie" posterUrl="/poster.jpg" mediaType="movie" />);
    expect(screen.getByText('已新增至媒體庫')).toBeTruthy();
  });

  it('[P1] renders poster image', () => {
    render(<NewMediaToast title="Test Movie" posterUrl="/poster.jpg" mediaType="movie" />);
    const img = screen.getByAltText('Test Movie');
    expect(img).toBeTruthy();
    expect((img as HTMLImageElement).src).toContain('/poster.jpg');
  });

  it('[P2] shows placeholder when no poster', () => {
    render(<NewMediaToast title="No Poster Movie" mediaType="movie" />);
    expect(screen.getByTestId('poster-placeholder')).toBeTruthy();
  });

  it('[P2] shows media type label', () => {
    render(<NewMediaToast title="Test Series" posterUrl="/poster.jpg" mediaType="tv" />);
    expect(screen.getByText('影集')).toBeTruthy();
  });

  it('[P2] shows "電影" for movie type', () => {
    render(<NewMediaToast title="Test Movie" posterUrl="/poster.jpg" mediaType="movie" />);
    expect(screen.getByText('電影')).toBeTruthy();
  });
});
