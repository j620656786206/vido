import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { SidePanel } from './SidePanel';

describe('SidePanel', () => {
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    children: <div>Panel content</div>,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    // Clean up body overflow style
    document.body.style.overflow = '';
  });

  it('should not render when closed', () => {
    render(<SidePanel {...defaultProps} isOpen={false} />);
    expect(screen.queryByTestId('side-panel')).not.toBeInTheDocument();
  });

  it('should render when open', () => {
    render(<SidePanel {...defaultProps} />);
    expect(screen.getByTestId('side-panel')).toBeInTheDocument();
  });

  it('should render children content', () => {
    render(<SidePanel {...defaultProps} />);
    expect(screen.getByText('Panel content')).toBeInTheDocument();
  });

  it('should render title when provided', () => {
    render(<SidePanel {...defaultProps} title="Test Title" />);
    expect(screen.getByText('Test Title')).toBeInTheDocument();
  });

  it('should call onClose when close button is clicked', () => {
    const onClose = vi.fn();
    render(<SidePanel {...defaultProps} onClose={onClose} />);

    fireEvent.click(screen.getByTestId('side-panel-close'));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('should call onClose when backdrop area is clicked', () => {
    const onClose = vi.fn();
    render(<SidePanel {...defaultProps} onClose={onClose} />);

    // Click on the dialog container (which wraps both backdrop and panel)
    // The backdrop click handler is on the container, not the backdrop element itself
    const dialog = screen.getByRole('dialog');
    fireEvent.click(dialog);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('should call onClose when Escape key is pressed', () => {
    const onClose = vi.fn();
    render(<SidePanel {...defaultProps} onClose={onClose} />);

    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('should not call onClose for other keys', () => {
    const onClose = vi.fn();
    render(<SidePanel {...defaultProps} onClose={onClose} />);

    fireEvent.keyDown(document, { key: 'Enter' });
    expect(onClose).not.toHaveBeenCalled();
  });

  it('should set body overflow to hidden when open', () => {
    render(<SidePanel {...defaultProps} />);
    expect(document.body.style.overflow).toBe('hidden');
  });

  it('should restore body overflow when closed', () => {
    const { rerender } = render(<SidePanel {...defaultProps} />);
    expect(document.body.style.overflow).toBe('hidden');

    rerender(<SidePanel {...defaultProps} isOpen={false} />);
    expect(document.body.style.overflow).toBe('');
  });

  it('should have aria-modal attribute', () => {
    render(<SidePanel {...defaultProps} />);
    expect(screen.getByRole('dialog')).toHaveAttribute('aria-modal', 'true');
  });

  it('should have close button with accessible label', () => {
    render(<SidePanel {...defaultProps} />);
    expect(screen.getByLabelText('關閉')).toBeInTheDocument();
  });

  it('should apply slide-in animation class when open', () => {
    render(<SidePanel {...defaultProps} />);
    const panel = screen.getByTestId('side-panel');
    expect(panel).toHaveClass('translate-x-0');
  });
});
