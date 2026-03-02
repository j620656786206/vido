import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ParseCompleteToast } from './ParseCompleteToast';

describe('ParseCompleteToast', () => {
  it('renders media title', () => {
    render(
      <ParseCompleteToast
        title="Test Movie"
        posterUrl="/poster.jpg"
        mediaType="movie"
      />
    );
    expect(screen.getByText('Test Movie')).toBeTruthy();
  });

  it('renders success message', () => {
    render(
      <ParseCompleteToast
        title="Test Movie"
        posterUrl="/poster.jpg"
        mediaType="movie"
      />
    );
    expect(screen.getByText('解析完成')).toBeTruthy();
  });

  it('renders poster image', () => {
    render(
      <ParseCompleteToast
        title="Test Movie"
        posterUrl="/poster.jpg"
        mediaType="movie"
      />
    );
    const img = screen.getByAltText('Test Movie');
    expect(img).toBeTruthy();
    expect((img as HTMLImageElement).src).toContain('/poster.jpg');
  });

  it('shows placeholder when no poster', () => {
    render(
      <ParseCompleteToast title="No Poster" mediaType="movie" />
    );
    expect(screen.getByTestId('parse-complete-poster-placeholder')).toBeTruthy();
  });

  it('shows media type label for movie', () => {
    render(
      <ParseCompleteToast title="Movie" posterUrl="/p.jpg" mediaType="movie" />
    );
    expect(screen.getByText('電影')).toBeTruthy();
  });

  it('shows media type label for tv', () => {
    render(
      <ParseCompleteToast title="Series" posterUrl="/p.jpg" mediaType="tv" />
    );
    expect(screen.getByText('影集')).toBeTruthy();
  });

  it('renders failed state when status is failed', () => {
    render(
      <ParseCompleteToast
        title="Failed Movie"
        mediaType="movie"
        status="failed"
        errorMessage="could not parse filename"
      />
    );
    expect(screen.getByText('解析失敗')).toBeTruthy();
    expect(screen.getByText(/could not parse filename/)).toBeTruthy();
  });

  it('has proper data-testid attribute', () => {
    render(
      <ParseCompleteToast title="Test" mediaType="movie" />
    );
    expect(screen.getByTestId('parse-complete-toast')).toBeTruthy();
  });
});
