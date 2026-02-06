import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DegradationBadge } from './DegradationBadge';

describe('DegradationBadge', () => {
  it('returns null for normal level', () => {
    const { container } = render(<DegradationBadge level="normal" />);
    expect(container.firstChild).toBeNull();
  });

  it('renders partial degradation badge', () => {
    render(<DegradationBadge level="partial" />);
    expect(screen.getByRole('status')).toBeInTheDocument();
    expect(screen.getByText('部分降級')).toBeInTheDocument();
  });

  it('renders minimal degradation badge', () => {
    render(<DegradationBadge level="minimal" />);
    expect(screen.getByRole('status')).toBeInTheDocument();
    expect(screen.getByText('功能受限')).toBeInTheDocument();
  });

  it('renders offline badge', () => {
    render(<DegradationBadge level="offline" />);
    expect(screen.getByRole('status')).toBeInTheDocument();
    expect(screen.getByText('離線模式')).toBeInTheDocument();
  });

  it('hides label when showLabel is false', () => {
    render(<DegradationBadge level="partial" showLabel={false} />);
    expect(screen.queryByText('部分降級')).not.toBeInTheDocument();
  });

  it('applies custom className', () => {
    render(<DegradationBadge level="partial" className="custom-class" />);
    expect(screen.getByRole('status')).toHaveClass('custom-class');
  });

  it('has accessible label', () => {
    render(<DegradationBadge level="offline" />);
    expect(screen.getByRole('status')).toHaveAttribute(
      'aria-label',
      '系統狀態：離線模式'
    );
  });
});
