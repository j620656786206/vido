import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { MediaTypeTabs } from './MediaTypeTabs';

describe('MediaTypeTabs', () => {
  it('should render all three tabs', () => {
    const onTypeChange = vi.fn();
    render(<MediaTypeTabs activeType="all" onTypeChange={onTypeChange} />);

    expect(screen.getByText('全部')).toBeInTheDocument();
    expect(screen.getByText('電影')).toBeInTheDocument();
    expect(screen.getByText('影集')).toBeInTheDocument();
  });

  it('should highlight the active tab', () => {
    const onTypeChange = vi.fn();
    render(<MediaTypeTabs activeType="movie" onTypeChange={onTypeChange} />);

    const movieTab = screen.getByRole('tab', { name: /電影/ });
    expect(movieTab).toHaveAttribute('aria-selected', 'true');
    expect(movieTab).toHaveClass('bg-blue-600');
  });

  it('should call onTypeChange when clicking a tab', () => {
    const onTypeChange = vi.fn();
    render(<MediaTypeTabs activeType="all" onTypeChange={onTypeChange} />);

    fireEvent.click(screen.getByText('電影'));
    expect(onTypeChange).toHaveBeenCalledWith('movie');
  });

  it('should show result counts when provided', () => {
    const onTypeChange = vi.fn();
    render(
      <MediaTypeTabs activeType="all" onTypeChange={onTypeChange} movieCount={50} tvCount={30} />
    );

    expect(screen.getByText('80')).toBeInTheDocument(); // Total
    expect(screen.getByText('50')).toBeInTheDocument(); // Movies
    expect(screen.getByText('30')).toBeInTheDocument(); // TV
  });

  it('should not show count badges when counts are undefined', () => {
    const onTypeChange = vi.fn();
    render(<MediaTypeTabs activeType="all" onTypeChange={onTypeChange} />);

    // Should only have the label text, no count badges
    const tabs = screen.getAllByRole('tab');
    tabs.forEach((tab) => {
      expect(tab.textContent).toMatch(/^(全部|電影|影集)$/);
    });
  });

  it('should have accessible labels', () => {
    const onTypeChange = vi.fn();
    render(<MediaTypeTabs activeType="all" onTypeChange={onTypeChange} />);

    expect(screen.getByLabelText('媒體類型篩選')).toBeInTheDocument();
  });

  it('should set correct aria-selected for each tab', () => {
    const onTypeChange = vi.fn();
    render(<MediaTypeTabs activeType="tv" onTypeChange={onTypeChange} />);

    expect(screen.getByRole('tab', { name: /全部/ })).toHaveAttribute('aria-selected', 'false');
    expect(screen.getByRole('tab', { name: /電影/ })).toHaveAttribute('aria-selected', 'false');
    expect(screen.getByRole('tab', { name: /影集/ })).toHaveAttribute('aria-selected', 'true');
  });
});
