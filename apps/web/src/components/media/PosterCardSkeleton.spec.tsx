import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { PosterCardSkeleton } from './PosterCardSkeleton';

describe('PosterCardSkeleton', () => {
  it('renders skeleton container with animation', () => {
    const { container } = render(<PosterCardSkeleton />);
    const skeleton = container.firstChild as HTMLElement;
    expect(skeleton).toHaveClass('animate-pulse');
  });

  it('renders poster placeholder with correct aspect ratio', () => {
    const { container } = render(<PosterCardSkeleton />);
    const posterPlaceholder = container.querySelector('.aspect-\\[2\\/3\\]');
    expect(posterPlaceholder).toBeInTheDocument();
    expect(posterPlaceholder).toHaveClass('rounded-lg', 'bg-gray-700');
  });

  it('renders title placeholder', () => {
    const { container } = render(<PosterCardSkeleton />);
    const titlePlaceholder = container.querySelector('.h-4.w-3\\/4');
    expect(titlePlaceholder).toBeInTheDocument();
    expect(titlePlaceholder).toHaveClass('rounded', 'bg-gray-700');
  });

  it('renders year placeholder', () => {
    const { container } = render(<PosterCardSkeleton />);
    const yearPlaceholder = container.querySelector('.h-3.w-1\\/4');
    expect(yearPlaceholder).toBeInTheDocument();
    expect(yearPlaceholder).toHaveClass('rounded', 'bg-gray-700');
  });

  it('has proper spacing between poster and text', () => {
    const { container } = render(<PosterCardSkeleton />);
    const textContainer = container.querySelector('.mt-2');
    expect(textContainer).toBeInTheDocument();
  });

  it('has proper spacing between title and year placeholders', () => {
    const { container } = render(<PosterCardSkeleton />);
    const textContainer = container.querySelector('.space-y-1');
    expect(textContainer).toBeInTheDocument();
  });
});
