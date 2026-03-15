import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { DetailPanelMenu } from './DetailPanelMenu';

describe('DetailPanelMenu', () => {
  const defaultProps = {
    onReparse: vi.fn(),
    onExport: vi.fn(),
    onDelete: vi.fn(),
  };

  it('renders trigger button', () => {
    render(<DetailPanelMenu {...defaultProps} />);
    expect(screen.getByTestId('detail-menu-trigger')).toBeInTheDocument();
  });

  it('opens dropdown on click', () => {
    render(<DetailPanelMenu {...defaultProps} />);

    expect(screen.queryByTestId('detail-menu-dropdown')).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId('detail-menu-trigger'));
    expect(screen.getByTestId('detail-menu-dropdown')).toBeInTheDocument();
  });

  it('shows reparse, export, and delete menu items', () => {
    render(<DetailPanelMenu {...defaultProps} />);
    fireEvent.click(screen.getByTestId('detail-menu-trigger'));

    expect(screen.getByTestId('menu-reparse')).toHaveTextContent('重新解析');
    expect(screen.getByTestId('menu-export')).toHaveTextContent('匯出 Metadata');
    expect(screen.getByTestId('menu-delete')).toHaveTextContent('刪除');
  });

  it('calls onReparse and closes menu', () => {
    const onReparse = vi.fn();
    render(<DetailPanelMenu {...defaultProps} onReparse={onReparse} />);

    fireEvent.click(screen.getByTestId('detail-menu-trigger'));
    fireEvent.click(screen.getByTestId('menu-reparse'));

    expect(onReparse).toHaveBeenCalledOnce();
    expect(screen.queryByTestId('detail-menu-dropdown')).not.toBeInTheDocument();
  });

  it('calls onExport and closes menu', () => {
    const onExport = vi.fn();
    render(<DetailPanelMenu {...defaultProps} onExport={onExport} />);

    fireEvent.click(screen.getByTestId('detail-menu-trigger'));
    fireEvent.click(screen.getByTestId('menu-export'));

    expect(onExport).toHaveBeenCalledOnce();
    expect(screen.queryByTestId('detail-menu-dropdown')).not.toBeInTheDocument();
  });

  it('shows confirmation dialog before delete', () => {
    render(<DetailPanelMenu {...defaultProps} />);

    fireEvent.click(screen.getByTestId('detail-menu-trigger'));
    fireEvent.click(screen.getByTestId('menu-delete'));

    expect(screen.getByText('確定要刪除嗎？')).toBeInTheDocument();
    expect(screen.getByTestId('confirm-delete')).toBeInTheDocument();
    expect(screen.getByTestId('cancel-delete')).toBeInTheDocument();
  });

  it('calls onDelete after confirmation', () => {
    const onDelete = vi.fn();
    render(<DetailPanelMenu {...defaultProps} onDelete={onDelete} />);

    fireEvent.click(screen.getByTestId('detail-menu-trigger'));
    fireEvent.click(screen.getByTestId('menu-delete'));
    fireEvent.click(screen.getByTestId('confirm-delete'));

    expect(onDelete).toHaveBeenCalledOnce();
  });

  it('cancels delete and shows menu items again', () => {
    render(<DetailPanelMenu {...defaultProps} />);

    fireEvent.click(screen.getByTestId('detail-menu-trigger'));
    fireEvent.click(screen.getByTestId('menu-delete'));
    fireEvent.click(screen.getByTestId('cancel-delete'));

    expect(screen.getByTestId('menu-delete')).toBeInTheDocument();
    expect(defaultProps.onDelete).not.toHaveBeenCalled();
  });

  it('closes menu on outside click', () => {
    render(
      <div>
        <div data-testid="outside">outside</div>
        <DetailPanelMenu {...defaultProps} />
      </div>
    );

    fireEvent.click(screen.getByTestId('detail-menu-trigger'));
    expect(screen.getByTestId('detail-menu-dropdown')).toBeInTheDocument();

    fireEvent.mouseDown(screen.getByTestId('outside'));
    expect(screen.queryByTestId('detail-menu-dropdown')).not.toBeInTheDocument();
  });
});
