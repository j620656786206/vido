import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { SortSelector } from './SortSelector';
import type { SortField, SortOrder } from './SortSelector';

describe('SortSelector', () => {
  const defaultProps = {
    sortBy: 'created_at' as SortField,
    sortOrder: 'desc' as SortOrder,
    onSortChange: vi.fn(),
  };

  it('renders sort button with current sort label', () => {
    render(<SortSelector {...defaultProps} />);
    expect(screen.getByTestId('sort-selector-button')).toBeInTheDocument();
    expect(screen.getByText('新增日期')).toBeInTheDocument();
  });

  it('shows direction arrow indicator', () => {
    const { rerender } = render(<SortSelector {...defaultProps} sortOrder="desc" />);
    expect(screen.getByTestId('sort-direction-indicator')).toBeInTheDocument();

    rerender(<SortSelector {...defaultProps} sortOrder="asc" />);
    expect(screen.getByTestId('sort-direction-indicator')).toBeInTheDocument();
  });

  it('opens dropdown on click', async () => {
    render(<SortSelector {...defaultProps} />);
    expect(screen.queryByTestId('sort-dropdown')).not.toBeInTheDocument();

    await userEvent.click(screen.getByTestId('sort-selector-button'));
    expect(screen.getByTestId('sort-dropdown')).toBeInTheDocument();
  });

  it('renders all sort options with zh-TW labels', async () => {
    render(<SortSelector {...defaultProps} />);
    await userEvent.click(screen.getByTestId('sort-selector-button'));

    expect(screen.getByTestId('sort-option-created_at')).toHaveTextContent('新增日期');
    expect(screen.getByTestId('sort-option-title')).toHaveTextContent('標題');
    expect(screen.getByTestId('sort-option-release_date')).toHaveTextContent('年份');
    expect(screen.getByTestId('sort-option-rating')).toHaveTextContent('評分');
  });

  it('calls onSortChange when selecting a different sort option', async () => {
    const onSortChange = vi.fn();
    render(<SortSelector {...defaultProps} onSortChange={onSortChange} />);
    await userEvent.click(screen.getByTestId('sort-selector-button'));

    await userEvent.click(screen.getByTestId('sort-option-title'));
    expect(onSortChange).toHaveBeenCalledWith('title', 'asc');
  });

  it('toggles direction when clicking same sort option', async () => {
    const onSortChange = vi.fn();
    render(<SortSelector sortBy="created_at" sortOrder="desc" onSortChange={onSortChange} />);
    await userEvent.click(screen.getByTestId('sort-selector-button'));

    await userEvent.click(screen.getByTestId('sort-option-created_at'));
    expect(onSortChange).toHaveBeenCalledWith('created_at', 'asc');
  });

  it('uses default direction for new sort field selection', async () => {
    const onSortChange = vi.fn();
    render(<SortSelector sortBy="created_at" sortOrder="desc" onSortChange={onSortChange} />);
    await userEvent.click(screen.getByTestId('sort-selector-button'));

    // Title defaults to asc, rating defaults to desc
    await userEvent.click(screen.getByTestId('sort-option-rating'));
    expect(onSortChange).toHaveBeenCalledWith('rating', 'desc');
  });

  it('highlights active sort option', async () => {
    render(<SortSelector {...defaultProps} sortBy="title" sortOrder="asc" />);
    await userEvent.click(screen.getByTestId('sort-selector-button'));

    const titleOption = screen.getByTestId('sort-option-title');
    expect(titleOption.className).toContain('bg-blue-600');
  });

  it('closes dropdown after selection', async () => {
    const onSortChange = vi.fn();
    render(<SortSelector {...defaultProps} onSortChange={onSortChange} />);
    await userEvent.click(screen.getByTestId('sort-selector-button'));
    expect(screen.getByTestId('sort-dropdown')).toBeInTheDocument();

    await userEvent.click(screen.getByTestId('sort-option-title'));
    expect(screen.queryByTestId('sort-dropdown')).not.toBeInTheDocument();
  });

  it('closes dropdown on outside click', async () => {
    render(
      <div>
        <div data-testid="outside">Outside</div>
        <SortSelector {...defaultProps} />
      </div>
    );
    await userEvent.click(screen.getByTestId('sort-selector-button'));
    expect(screen.getByTestId('sort-dropdown')).toBeInTheDocument();

    fireEvent.mouseDown(screen.getByTestId('outside'));
    expect(screen.queryByTestId('sort-dropdown')).not.toBeInTheDocument();
  });

  it('has accessible aria-label', () => {
    render(<SortSelector {...defaultProps} />);
    expect(screen.getByLabelText('排序方式')).toBeInTheDocument();
  });

  it('supports keyboard navigation with Escape to close', async () => {
    render(<SortSelector {...defaultProps} />);
    await userEvent.click(screen.getByTestId('sort-selector-button'));
    expect(screen.getByTestId('sort-dropdown')).toBeInTheDocument();

    await userEvent.keyboard('{Escape}');
    expect(screen.queryByTestId('sort-dropdown')).not.toBeInTheDocument();
  });

  it('shows correct label for each sortBy value', () => {
    const labels: Record<SortField, string> = {
      created_at: '新增日期',
      title: '標題',
      release_date: '年份',
      rating: '評分',
    };

    for (const [field, label] of Object.entries(labels)) {
      const { unmount } = render(
        <SortSelector sortBy={field as SortField} sortOrder="desc" onSortChange={vi.fn()} />
      );
      expect(screen.getByTestId('sort-selector-button')).toHaveTextContent(label);
      unmount();
    }
  });
});
