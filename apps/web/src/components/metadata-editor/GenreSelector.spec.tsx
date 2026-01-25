/**
 * GenreSelector Tests (Story 3.8 - AC1)
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { GenreSelector, GENRE_OPTIONS } from './GenreSelector';

describe('GenreSelector', () => {
  const defaultProps = {
    selectedGenres: [],
    onToggle: vi.fn(),
  };

  it('renders all genre options', () => {
    render(<GenreSelector {...defaultProps} />);

    GENRE_OPTIONS.forEach((genre) => {
      expect(screen.getByRole('button', { name: genre.label })).toBeTruthy();
    });
  });

  it('renders custom label', () => {
    render(<GenreSelector {...defaultProps} label="分類" />);
    expect(screen.getByText('分類')).toBeTruthy();
  });

  it('highlights selected genres', () => {
    render(<GenreSelector {...defaultProps} selectedGenres={['action', 'drama']} />);

    const actionButton = screen.getByRole('button', { name: '動作' });
    const dramaButton = screen.getByRole('button', { name: '劇情' });
    const comedyButton = screen.getByRole('button', { name: '喜劇' });

    expect(actionButton.className).toContain('bg-blue-600');
    expect(dramaButton.className).toContain('bg-blue-600');
    expect(comedyButton.className).not.toContain('bg-blue-600');
  });

  it('calls onToggle when genre is clicked', async () => {
    const onToggle = vi.fn();
    render(<GenreSelector {...defaultProps} onToggle={onToggle} />);

    await userEvent.click(screen.getByRole('button', { name: '動作' }));

    expect(onToggle).toHaveBeenCalledWith('action');
  });

  it('sets aria-pressed attribute correctly', () => {
    render(<GenreSelector {...defaultProps} selectedGenres={['action']} />);

    const actionButton = screen.getByRole('button', { name: '動作' });
    const comedyButton = screen.getByRole('button', { name: '喜劇' });

    expect(actionButton.getAttribute('aria-pressed')).toBe('true');
    expect(comedyButton.getAttribute('aria-pressed')).toBe('false');
  });

  it('renders with custom options', () => {
    const customOptions = [
      { value: 'custom1', label: '自訂1' },
      { value: 'custom2', label: '自訂2' },
    ];

    render(<GenreSelector {...defaultProps} options={customOptions} />);

    expect(screen.getByRole('button', { name: '自訂1' })).toBeTruthy();
    expect(screen.getByRole('button', { name: '自訂2' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: '動作' })).toBeNull();
  });
});
