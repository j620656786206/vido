import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { BatchConfirmDialog } from './BatchConfirmDialog';

describe('BatchConfirmDialog', () => {
  const defaultProps = {
    isOpen: true,
    itemCount: 5,
    action: 'delete' as const,
    onConfirm: vi.fn(),
    onCancel: vi.fn(),
  };

  it('renders when open', () => {
    render(<BatchConfirmDialog {...defaultProps} />);
    expect(screen.getByTestId('batch-confirm-dialog')).toBeInTheDocument();
  });

  it('does not render when closed', () => {
    render(<BatchConfirmDialog {...defaultProps} isOpen={false} />);
    expect(screen.queryByTestId('batch-confirm-dialog')).not.toBeInTheDocument();
  });

  it('shows item count in delete message', () => {
    render(<BatchConfirmDialog {...defaultProps} />);
    expect(screen.getByTestId('confirm-message')).toHaveTextContent('確定要刪除 5 個項目嗎？');
  });

  it('shows warning for delete action', () => {
    render(<BatchConfirmDialog {...defaultProps} />);
    expect(screen.getByTestId('confirm-warning')).toHaveTextContent('此操作無法復原');
  });

  it('shows reparse message for reparse action', () => {
    render(<BatchConfirmDialog {...defaultProps} action="reparse" />);
    expect(screen.getByTestId('confirm-message')).toHaveTextContent('確定要重新解析 5 個項目嗎？');
  });

  it('calls onConfirm when confirm button clicked', () => {
    render(<BatchConfirmDialog {...defaultProps} />);
    fireEvent.click(screen.getByTestId('confirm-action-btn'));
    expect(defaultProps.onConfirm).toHaveBeenCalledOnce();
  });

  it('calls onCancel when cancel button clicked', () => {
    render(<BatchConfirmDialog {...defaultProps} />);
    fireEvent.click(screen.getByTestId('confirm-cancel-btn'));
    expect(defaultProps.onCancel).toHaveBeenCalledOnce();
  });

  it('calls onCancel when backdrop clicked', () => {
    render(<BatchConfirmDialog {...defaultProps} />);
    fireEvent.click(screen.getByTestId('batch-confirm-dialog'));
    expect(defaultProps.onCancel).toHaveBeenCalled();
  });

  it('calls onCancel on Escape key', () => {
    render(<BatchConfirmDialog {...defaultProps} />);
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(defaultProps.onCancel).toHaveBeenCalled();
  });
});
