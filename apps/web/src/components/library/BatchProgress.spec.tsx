import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { BatchProgress } from './BatchProgress';

describe('BatchProgress', () => {
  const defaultProps = {
    isOpen: true,
    current: 5,
    total: 20,
    action: '刪除中...',
    isComplete: false,
    onClose: vi.fn(),
  };

  it('renders when open', () => {
    render(<BatchProgress {...defaultProps} />);
    expect(screen.getByTestId('batch-progress')).toBeInTheDocument();
  });

  it('does not render when closed', () => {
    render(<BatchProgress {...defaultProps} isOpen={false} />);
    expect(screen.queryByTestId('batch-progress')).not.toBeInTheDocument();
  });

  it('shows progress text', () => {
    render(<BatchProgress {...defaultProps} />);
    expect(screen.getByTestId('progress-text')).toHaveTextContent('處理中 5 / 20...');
  });

  it('shows completion text when complete', () => {
    render(<BatchProgress {...defaultProps} current={20} isComplete={true} />);
    expect(screen.getByTestId('progress-text')).toHaveTextContent('已完成 20 / 20');
  });

  it('shows the note only once complete (batch 重新解析 keeps working after the request)', () => {
    const note = '已排入比對，比對完成後清單會自動更新';
    const { rerender } = render(<BatchProgress {...defaultProps} note={note} />);
    expect(screen.queryByTestId('progress-note')).not.toBeInTheDocument();
    rerender(<BatchProgress {...defaultProps} current={20} isComplete={true} note={note} />);
    expect(screen.getByTestId('progress-note')).toHaveTextContent(note);
  });

  it('shows close button when complete', () => {
    render(<BatchProgress {...defaultProps} isComplete={true} />);
    expect(screen.getByTestId('progress-close-btn')).toBeInTheDocument();
  });

  it('calls onClose when close button clicked', () => {
    render(<BatchProgress {...defaultProps} isComplete={true} />);
    fireEvent.click(screen.getByTestId('progress-close-btn'));
    expect(defaultProps.onClose).toHaveBeenCalledOnce();
  });

  it('shows cancel button when not complete and onCancel provided', () => {
    const onCancel = vi.fn();
    render(<BatchProgress {...defaultProps} onCancel={onCancel} />);
    expect(screen.getByTestId('progress-cancel-btn')).toBeInTheDocument();
  });

  it('shows errors after completion', () => {
    const errors = [
      { id: 'm1', message: 'not found' },
      { id: 'm2', message: 'permission denied' },
    ];
    render(<BatchProgress {...defaultProps} isComplete={true} errors={errors} />);
    expect(screen.getByText('2 個項目失敗：')).toBeInTheDocument();
    expect(screen.getByText('m1: not found')).toBeInTheDocument();
    expect(screen.getByText('m2: permission denied')).toBeInTheDocument();
  });

  it('renders progress bar with correct width', () => {
    render(<BatchProgress {...defaultProps} current={10} total={20} />);
    const bar = screen.getByTestId('progress-bar');
    expect(bar).toHaveStyle({ width: '50%' });
  });
});
