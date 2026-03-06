import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { PosterCardMenu } from './PosterCardMenu';

describe('PosterCardMenu', () => {
  const defaultProps = {
    onViewDetails: vi.fn(),
    onReparse: vi.fn(),
    onExport: vi.fn(),
    onDelete: vi.fn(),
    isOpen: true,
    onClose: vi.fn(),
  };

  it('renders menu when isOpen is true', () => {
    render(<PosterCardMenu {...defaultProps} />);
    expect(screen.getByTestId('poster-card-menu')).toBeInTheDocument();
  });

  it('does not render when isOpen is false', () => {
    render(<PosterCardMenu {...defaultProps} isOpen={false} />);
    expect(screen.queryByTestId('poster-card-menu')).not.toBeInTheDocument();
  });

  it('renders all menu items', () => {
    render(<PosterCardMenu {...defaultProps} />);
    expect(screen.getByText('查看詳情')).toBeInTheDocument();
    expect(screen.getByText('重新解析')).toBeInTheDocument();
    expect(screen.getByText('匯出中繼資料')).toBeInTheDocument();
    expect(screen.getByText('刪除')).toBeInTheDocument();
  });

  it('calls onViewDetails and onClose when View Details is clicked', () => {
    render(<PosterCardMenu {...defaultProps} />);
    fireEvent.click(screen.getByText('查看詳情'));
    expect(defaultProps.onViewDetails).toHaveBeenCalled();
    expect(defaultProps.onClose).toHaveBeenCalled();
  });

  it('calls onReparse and onClose when Re-parse is clicked', () => {
    render(<PosterCardMenu {...defaultProps} />);
    fireEvent.click(screen.getByText('重新解析'));
    expect(defaultProps.onReparse).toHaveBeenCalled();
    expect(defaultProps.onClose).toHaveBeenCalled();
  });

  it('requires confirmation for delete', () => {
    render(<PosterCardMenu {...defaultProps} />);
    // First click shows confirmation
    fireEvent.click(screen.getByText('刪除'));
    expect(defaultProps.onDelete).not.toHaveBeenCalled();
    expect(screen.getByText('確認刪除')).toBeInTheDocument();

    // Second click executes delete
    fireEvent.click(screen.getByText('確認刪除'));
    expect(defaultProps.onDelete).toHaveBeenCalled();
  });

  it('renders as bottom sheet on mobile', () => {
    render(<PosterCardMenu {...defaultProps} isMobile={true} />);
    expect(screen.getByTestId('poster-card-menu')).toBeInTheDocument();
    // Bottom sheet has different styling (fixed positioning)
    const menu = screen.getByTestId('poster-card-menu');
    expect(menu.className).toContain('fixed');
  });
});
