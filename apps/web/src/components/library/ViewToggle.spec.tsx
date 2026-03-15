import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { ViewToggle } from './ViewToggle';

describe('ViewToggle', () => {
  it('renders grid and list buttons', () => {
    render(<ViewToggle view="grid" onViewChange={vi.fn()} />);
    expect(screen.getByRole('radiogroup')).toBeInTheDocument();
    expect(screen.getByLabelText('格狀檢視')).toBeInTheDocument();
    expect(screen.getByLabelText('列表檢視')).toBeInTheDocument();
  });

  it('has correct aria-label on radiogroup', () => {
    render(<ViewToggle view="grid" onViewChange={vi.fn()} />);
    expect(screen.getByRole('radiogroup')).toHaveAttribute('aria-label', '切換檢視模式');
  });

  it('marks grid as checked when view is grid', () => {
    render(<ViewToggle view="grid" onViewChange={vi.fn()} />);
    expect(screen.getByLabelText('格狀檢視')).toHaveAttribute('aria-checked', 'true');
    expect(screen.getByLabelText('列表檢視')).toHaveAttribute('aria-checked', 'false');
  });

  it('marks list as checked when view is list', () => {
    render(<ViewToggle view="list" onViewChange={vi.fn()} />);
    expect(screen.getByLabelText('格狀檢視')).toHaveAttribute('aria-checked', 'false');
    expect(screen.getByLabelText('列表檢視')).toHaveAttribute('aria-checked', 'true');
  });

  it('applies active styling to grid button when grid is selected', () => {
    render(<ViewToggle view="grid" onViewChange={vi.fn()} />);
    const gridButton = screen.getByLabelText('格狀檢視');
    expect(gridButton.className).toContain('bg-blue-600');
    expect(gridButton.className).toContain('text-white');
  });

  it('applies inactive styling to list button when grid is selected', () => {
    render(<ViewToggle view="grid" onViewChange={vi.fn()} />);
    const listButton = screen.getByLabelText('列表檢視');
    expect(listButton.className).toContain('text-slate-400');
  });

  it('calls onViewChange with "list" when list button clicked', () => {
    const onViewChange = vi.fn();
    render(<ViewToggle view="grid" onViewChange={onViewChange} />);
    fireEvent.click(screen.getByLabelText('列表檢視'));
    expect(onViewChange).toHaveBeenCalledWith('list');
  });

  it('calls onViewChange with "grid" when grid button clicked', () => {
    const onViewChange = vi.fn();
    render(<ViewToggle view="list" onViewChange={onViewChange} />);
    fireEvent.click(screen.getByLabelText('格狀檢視'));
    expect(onViewChange).toHaveBeenCalledWith('grid');
  });

  it('has data-testid for test targeting', () => {
    render(<ViewToggle view="grid" onViewChange={vi.fn()} />);
    expect(screen.getByTestId('view-toggle')).toBeInTheDocument();
  });
});
