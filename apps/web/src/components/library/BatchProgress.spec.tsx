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

  // disc-2026-09-batch-reparse-progress-in-dialog — C24-D's three states.
  describe('batch 重新解析 matching states (C24-D)', () => {
    const base = {
      ...defaultProps,
      action: '重新解析中...',
      current: 5,
      total: 5,
      isComplete: true,
    };
    const m = {
      total: 20,
      processed: 3,
      succeeded: 2,
      failed: 1,
      skipped: 0,
      currentTitle: '你的名字',
    };

    it('① queued: 重新解析中, 已排入 N 項, says the pass covers every pending row', () => {
      render(
        <BatchProgress {...base} matching={{ ...m, phase: 'queued', total: 0, processed: 0 }} />
      );
      expect(screen.getByRole('heading')).toHaveTextContent('重新解析中');
      expect(screen.getByTestId('progress-text')).toHaveTextContent('已排入比對 5 項，等待開始…');
      expect(screen.getByTestId('matching-note')).toHaveTextContent('不只你勾的 5 部');
      expect(screen.getByTestId('progress-bar')).toHaveStyle({ width: '0%' });
      expect(screen.getByTestId('batch-progress')).toHaveAttribute('data-matching-phase', 'queued');
    });

    it('② running: 比對中, whole-pass total with 含你勾的 N 部, current title, count, tally', () => {
      render(<BatchProgress {...base} matching={{ ...m, phase: 'running' }} />);
      expect(screen.getByRole('heading')).toHaveTextContent('比對中');
      expect(screen.getByTestId('progress-text')).toHaveTextContent(
        '本輪整理 20 部（含你勾的 5 部）'
      );
      expect(screen.getByTestId('matching-current')).toHaveTextContent('目前：你的名字');
      expect(screen.getByTestId('matching-count')).toHaveTextContent('3 / 20');
      expect(screen.getByTestId('matching-tally')).toHaveTextContent('成功 2・失敗 1・略過 0');
      expect(screen.getByTestId('progress-bar')).toHaveStyle({ width: '15%' });
      expect(screen.queryByTestId('progress-cancel-btn')).not.toBeInTheDocument();
      expect(screen.getByTestId('progress-close-btn')).toBeInTheDocument();
    });

    it('③ done: 比對完成 with the result, the errors list stays', () => {
      const errors = [{ id: '寄生上流', message: '找不到符合的作品' }];
      render(
        <BatchProgress
          {...base}
          errors={errors}
          matching={{
            ...m,
            phase: 'done',
            processed: 20,
            succeeded: 18,
            failed: 2,
            currentTitle: '',
          }}
        />
      );
      expect(screen.getByRole('heading')).toHaveTextContent('比對完成');
      expect(screen.getByTestId('progress-text')).toHaveTextContent('成功 18・失敗 2 — 清單已更新');
      expect(screen.getByTestId('progress-bar')).toHaveStyle({ width: '100%' });
      expect(screen.getByText('寄生上流: 找不到符合的作品')).toBeInTheDocument();
    });

    it('matching is ignored while the request itself is still in flight', () => {
      render(<BatchProgress {...base} isComplete={false} matching={{ ...m, phase: 'running' }} />);
      expect(screen.getByRole('heading')).toHaveTextContent('重新解析中...');
      expect(screen.getByTestId('progress-text')).toHaveTextContent('處理中 5 / 5...');
    });
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

  // disc-2026-09-batch-error-shows-id-not-title
  it('names a failed row by its title, falling back to the id when the row is gone', () => {
    const errors = [
      { id: '3f1c9a2e', title: '寄生上流', message: '找不到符合的作品' },
      { id: 'ghost-id', message: 'movie not found' },
    ];
    render(<BatchProgress {...defaultProps} isComplete={true} errors={errors} />);
    expect(screen.getByText('寄生上流: 找不到符合的作品')).toBeInTheDocument();
    expect(screen.queryByText(/3f1c9a2e/)).not.toBeInTheDocument();
    expect(screen.getByText('ghost-id: movie not found')).toBeInTheDocument();
  });

  it('renders progress bar with correct width', () => {
    render(<BatchProgress {...defaultProps} current={10} total={20} />);
    const bar = screen.getByTestId('progress-bar');
    expect(bar).toHaveStyle({ width: '50%' });
  });
});
