import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { SelectionToolbar } from './SelectionToolbar';

describe('SelectionToolbar', () => {
  const defaultProps = {
    selectedCount: 5,
    onDelete: vi.fn(),
    onReparse: vi.fn(),
    onExport: vi.fn(),
    onCancel: vi.fn(),
  };

  it('renders selected count', () => {
    render(<SelectionToolbar {...defaultProps} />);
    expect(screen.getByTestId('selected-count')).toHaveTextContent('已選取 5 項');
  });

  it('renders all action buttons', () => {
    render(<SelectionToolbar {...defaultProps} />);
    expect(screen.getByTestId('batch-delete-btn')).toBeInTheDocument();
    expect(screen.getByTestId('batch-reparse-btn')).toBeInTheDocument();
    expect(screen.getByTestId('batch-export-btn')).toBeInTheDocument();
    expect(screen.getByTestId('batch-cancel-btn')).toBeInTheDocument();
  });

  it('calls onDelete when delete clicked', () => {
    render(<SelectionToolbar {...defaultProps} />);
    fireEvent.click(screen.getByTestId('batch-delete-btn'));
    expect(defaultProps.onDelete).toHaveBeenCalledOnce();
  });

  it('calls onReparse when reparse clicked', () => {
    render(<SelectionToolbar {...defaultProps} />);
    fireEvent.click(screen.getByTestId('batch-reparse-btn'));
    expect(defaultProps.onReparse).toHaveBeenCalledOnce();
  });

  it('calls onExport when export clicked', () => {
    render(<SelectionToolbar {...defaultProps} />);
    fireEvent.click(screen.getByTestId('batch-export-btn'));
    expect(defaultProps.onExport).toHaveBeenCalledOnce();
  });

  it('calls onCancel when cancel clicked', () => {
    render(<SelectionToolbar {...defaultProps} />);
    fireEvent.click(screen.getByTestId('batch-cancel-btn'));
    expect(defaultProps.onCancel).toHaveBeenCalledOnce();
  });

  it('disables action buttons when processing', () => {
    render(<SelectionToolbar {...defaultProps} isProcessing={true} />);
    expect(screen.getByTestId('batch-delete-btn')).toBeDisabled();
    expect(screen.getByTestId('batch-reparse-btn')).toBeDisabled();
    expect(screen.getByTestId('batch-export-btn')).toBeDisabled();
  });

  it('updates count dynamically', () => {
    const { rerender } = render(<SelectionToolbar {...defaultProps} selectedCount={3} />);
    expect(screen.getByTestId('selected-count')).toHaveTextContent('已選取 3 項');

    rerender(<SelectionToolbar {...defaultProps} selectedCount={10} />);
    expect(screen.getByTestId('selected-count')).toHaveTextContent('已選取 10 項');
  });
});
