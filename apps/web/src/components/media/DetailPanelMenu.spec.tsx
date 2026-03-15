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

  it('[P1] trigger button has accessible aria-label', () => {
    render(<DetailPanelMenu {...defaultProps} />);
    expect(screen.getByTestId('detail-menu-trigger')).toHaveAttribute('aria-label', '更多操作');
  });

  it('[P1] trigger has aria-expanded and aria-haspopup', () => {
    render(<DetailPanelMenu {...defaultProps} />);
    const trigger = screen.getByTestId('detail-menu-trigger');

    expect(trigger).toHaveAttribute('aria-haspopup', 'menu');
    expect(trigger).toHaveAttribute('aria-expanded', 'false');

    fireEvent.click(trigger);
    expect(trigger).toHaveAttribute('aria-expanded', 'true');
  });

  it('[P1] dropdown has role=menu and items have role=menuitem', () => {
    render(<DetailPanelMenu {...defaultProps} />);
    fireEvent.click(screen.getByTestId('detail-menu-trigger'));

    expect(screen.getByTestId('detail-menu-dropdown')).toHaveAttribute('role', 'menu');
    expect(screen.getByTestId('menu-reparse')).toHaveAttribute('role', 'menuitem');
    expect(screen.getByTestId('menu-export')).toHaveAttribute('role', 'menuitem');
    expect(screen.getByTestId('menu-delete')).toHaveAttribute('role', 'menuitem');
  });

  it('[P1] separator has role=separator', () => {
    render(<DetailPanelMenu {...defaultProps} />);
    fireEvent.click(screen.getByTestId('detail-menu-trigger'));

    const dropdown = screen.getByTestId('detail-menu-dropdown');
    const separator = dropdown.querySelector('[role="separator"]');
    expect(separator).toBeInTheDocument();
  });

  it('[P1] toggles menu closed on second click', () => {
    render(<DetailPanelMenu {...defaultProps} />);

    fireEvent.click(screen.getByTestId('detail-menu-trigger'));
    expect(screen.getByTestId('detail-menu-dropdown')).toBeInTheDocument();

    fireEvent.click(screen.getByTestId('detail-menu-trigger'));
    expect(screen.queryByTestId('detail-menu-dropdown')).not.toBeInTheDocument();
  });

  it('[P1] delete button has red text styling', () => {
    render(<DetailPanelMenu {...defaultProps} />);
    fireEvent.click(screen.getByTestId('detail-menu-trigger'));

    const deleteBtn = screen.getByTestId('menu-delete');
    expect(deleteBtn.className).toContain('text-red');
  });

  it('[P1] has separator between export and delete', () => {
    render(<DetailPanelMenu {...defaultProps} />);
    fireEvent.click(screen.getByTestId('detail-menu-trigger'));

    const dropdown = screen.getByTestId('detail-menu-dropdown');
    const separator = dropdown.querySelector('.border-t');
    expect(separator).toBeInTheDocument();
  });

  it('[P1] resets confirm state when menu is reopened', () => {
    render(<DetailPanelMenu {...defaultProps} />);

    // Open and click delete to show confirmation
    fireEvent.click(screen.getByTestId('detail-menu-trigger'));
    fireEvent.click(screen.getByTestId('menu-delete'));
    expect(screen.getByText('確定要刪除嗎？')).toBeInTheDocument();

    // Close via outside click
    fireEvent.mouseDown(document.body);
    expect(screen.queryByTestId('detail-menu-dropdown')).not.toBeInTheDocument();

    // Reopen — should show normal menu, not confirmation
    fireEvent.click(screen.getByTestId('detail-menu-trigger'));
    expect(screen.getByTestId('menu-delete')).toBeInTheDocument();
    expect(screen.queryByText('確定要刪除嗎？')).not.toBeInTheDocument();
  });
});
