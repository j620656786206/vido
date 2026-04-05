import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { TechBadge } from './TechBadge';

describe('TechBadge', () => {
  it('renders with label text', () => {
    render(<TechBadge label="H.265" category="video" />);
    expect(screen.getByText('H.265')).toBeInTheDocument();
  });

  it('renders with video category styling', () => {
    render(<TechBadge label="4K" category="video" />);
    const badge = screen.getByTestId('tech-badge');
    expect(badge).toHaveClass('bg-blue-500/20', 'text-blue-400');
  });

  it('renders with audio category styling', () => {
    render(<TechBadge label="DTS 5.1" category="audio" />);
    const badge = screen.getByTestId('tech-badge');
    expect(badge).toHaveClass('bg-purple-500/20', 'text-purple-400');
  });

  it('renders with hdr category styling', () => {
    render(<TechBadge label="HDR10" category="hdr" />);
    const badge = screen.getByTestId('tech-badge');
    expect(badge).toHaveClass('bg-amber-500/20', 'text-amber-400');
  });

  it('renders with subtitle category styling', () => {
    render(<TechBadge label="3 字幕" category="subtitle" />);
    const badge = screen.getByTestId('tech-badge');
    expect(badge).toHaveClass('bg-emerald-500/20', 'text-emerald-400');
  });

  it('applies custom className', () => {
    render(<TechBadge label="H.265" category="video" className="mt-2" />);
    const badge = screen.getByTestId('tech-badge');
    expect(badge).toHaveClass('mt-2');
  });
});
