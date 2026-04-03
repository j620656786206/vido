import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { ColorPlaceholder, filenameToGradient } from './ColorPlaceholder';

describe('filenameToGradient', () => {
  it('returns two HSL color strings', () => {
    const [a, b] = filenameToGradient('test-movie.mkv');
    expect(a).toMatch(/^hsl\(\d+, 65%, 35%\)$/);
    expect(b).toMatch(/^hsl\(\d+, 55%, 45%\)$/);
  });

  it('is deterministic — same input produces same output', () => {
    const first = filenameToGradient('kimi-no-na-wa.mkv');
    const second = filenameToGradient('kimi-no-na-wa.mkv');
    expect(first).toEqual(second);
  });

  it('produces different gradients for different filenames', () => {
    const a = filenameToGradient('movie-a.mkv');
    const b = filenameToGradient('movie-b.mkv');
    expect(a).not.toEqual(b);
  });

  it('handles empty string without throwing', () => {
    const [a, b] = filenameToGradient('');
    expect(a).toMatch(/^hsl\(\d+, 65%, 35%\)$/);
    expect(b).toMatch(/^hsl\(\d+, 55%, 45%\)$/);
  });
});

describe('ColorPlaceholder', () => {
  it('renders with gradient background and initial letter', () => {
    render(<ColorPlaceholder filename="Inception.2010.1080p.mkv" />);

    const el = screen.getByTestId('color-placeholder');
    expect(el).toBeInTheDocument();
    // First character of filename
    expect(el).toHaveTextContent('I');
  });

  it('uses custom initial when provided', () => {
    render(<ColorPlaceholder filename="test.mkv" initial="未" />);

    const el = screen.getByTestId('color-placeholder');
    expect(el).toHaveTextContent('未');
  });

  it('applies custom height', () => {
    render(<ColorPlaceholder filename="test.mkv" height={200} />);

    const el = screen.getByTestId('color-placeholder');
    expect(el.style.height).toBe('200px');
  });

  it('defaults to no inline height when height prop omitted', () => {
    render(<ColorPlaceholder filename="test.mkv" />);

    const el = screen.getByTestId('color-placeholder');
    expect(el.style.height).toBe('');
  });

  it('applies additional className', () => {
    render(<ColorPlaceholder filename="test.mkv" className="w-full" />);

    const el = screen.getByTestId('color-placeholder');
    expect(el.className).toContain('w-full');
  });

  it('maintains 2:3 aspect ratio when height is set', () => {
    render(<ColorPlaceholder filename="test.mkv" height={240} />);

    const el = screen.getByTestId('color-placeholder');
    expect(el.getAttribute('style')).toContain('aspect-ratio: 2 / 3');
  });

  it('renders fallback "?" for empty filename', () => {
    render(<ColorPlaceholder filename="" />);

    const el = screen.getByTestId('color-placeholder');
    expect(el).toHaveTextContent('?');
  });
});
