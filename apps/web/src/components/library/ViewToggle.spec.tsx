import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { ViewToggle } from './ViewToggle';
import { expectTooltipOnFocus } from '../../test-utils/tooltip';

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
    expect(gridButton.className).toContain('bg-[var(--accent-primary)]');
    expect(gridButton.className).toContain('text-[var(--text-on-accent)]');
  });

  it('applies inactive styling to list button when grid is selected', () => {
    render(<ViewToggle view="grid" onViewChange={vi.fn()} />);
    const listButton = screen.getByLabelText('列表檢視');
    expect(listButton.className).toContain('text-[var(--text-secondary)]');
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

// disc-2026-10-icon-only-controls-tooltip-a — icon-only controls name themselves in the shared tooltip.
describe('ViewToggle — icon-only', () => {
  it('both modes show their names on keyboard focus', async () => {
    render(<ViewToggle view="grid" onViewChange={vi.fn()} />);
    await expectTooltipOnFocus(screen.getByLabelText('格狀檢視'), '格狀檢視');
    await expectTooltipOnFocus(screen.getByLabelText('列表檢視'), '列表檢視');
  });
});
