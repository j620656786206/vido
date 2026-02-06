/**
 * ParseStatusBadge Tests (Story 3.10 - Task 6)
 */

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ParseStatusBadge, ParsingStatusBadge } from './ParseStatusBadge';
import type { ParseStatus } from './types';

describe('ParseStatusBadge', () => {
  it('renders with pending status', () => {
    render(<ParseStatusBadge status="pending" />);

    const badge = screen.getByTestId('parse-status-badge');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveAttribute('data-status', 'pending');
    expect(badge).toHaveAttribute('title', '等待中');
  });

  it('renders with success status', () => {
    render(<ParseStatusBadge status="success" />);

    const badge = screen.getByTestId('parse-status-badge');
    expect(badge).toHaveAttribute('data-status', 'success');
    expect(badge).toHaveAttribute('title', '已完成');
  });

  it('renders with needs_ai status', () => {
    render(<ParseStatusBadge status="needs_ai" />);

    const badge = screen.getByTestId('parse-status-badge');
    expect(badge).toHaveAttribute('data-status', 'needs_ai');
    expect(badge).toHaveAttribute('title', '需要處理');
  });

  it('renders with failed status', () => {
    render(<ParseStatusBadge status="failed" />);

    const badge = screen.getByTestId('parse-status-badge');
    expect(badge).toHaveAttribute('data-status', 'failed');
    expect(badge).toHaveAttribute('title', '失敗');
  });

  it('uses custom tooltip when provided', () => {
    render(<ParseStatusBadge status="success" tooltip="解析成功！" />);

    const badge = screen.getByTestId('parse-status-badge');
    expect(badge).toHaveAttribute('title', '解析成功！');
  });

  it('shows label when showLabel is true', () => {
    render(<ParseStatusBadge status="success" showLabel />);

    expect(screen.getByText('已完成')).toBeInTheDocument();
  });

  it('does not show label by default', () => {
    render(<ParseStatusBadge status="success" />);

    expect(screen.queryByText('已完成')).not.toBeInTheDocument();
  });

  it('applies custom className', () => {
    render(<ParseStatusBadge status="success" className="custom-class" />);

    const badge = screen.getByTestId('parse-status-badge');
    expect(badge).toHaveClass('custom-class');
  });

  it('has correct ARIA attributes', () => {
    render(<ParseStatusBadge status="success" />);

    const badge = screen.getByTestId('parse-status-badge');
    expect(badge).toHaveAttribute('role', 'status');
    expect(badge).toHaveAttribute('aria-label', '解析狀態: 已完成');
  });

  it('renders with small size', () => {
    render(<ParseStatusBadge status="success" size="sm" />);

    const badge = screen.getByTestId('parse-status-badge');
    expect(badge).toHaveClass('px-1.5');
  });

  it('renders with large size', () => {
    render(<ParseStatusBadge status="success" size="lg" />);

    const badge = screen.getByTestId('parse-status-badge');
    expect(badge).toHaveClass('px-2.5');
  });

  it('renders all status types without crashing', () => {
    const statuses: ParseStatus[] = ['pending', 'success', 'needs_ai', 'failed'];

    statuses.forEach((status) => {
      const { unmount } = render(<ParseStatusBadge status={status} />);
      expect(screen.getByTestId('parse-status-badge')).toBeInTheDocument();
      unmount();
    });
  });
});

describe('ParsingStatusBadge', () => {
  it('renders with spinning animation', () => {
    render(<ParsingStatusBadge />);

    const badge = screen.getByTestId('parsing-status-badge');
    expect(badge).toBeInTheDocument();

    // Check for animate-spin class on the icon
    const icon = badge.querySelector('svg');
    expect(icon).toHaveClass('animate-spin');
  });

  it('has correct default tooltip', () => {
    render(<ParsingStatusBadge />);

    const badge = screen.getByTestId('parsing-status-badge');
    expect(badge).toHaveAttribute('title', '解析中...');
  });

  it('uses custom tooltip when provided', () => {
    render(<ParsingStatusBadge tooltip="正在搜尋 TMDb..." />);

    const badge = screen.getByTestId('parsing-status-badge');
    expect(badge).toHaveAttribute('title', '正在搜尋 TMDb...');
  });

  it('shows label when showLabel is true', () => {
    render(<ParsingStatusBadge showLabel />);

    expect(screen.getByText('解析中')).toBeInTheDocument();
  });

  it('has aria-live attribute for accessibility', () => {
    render(<ParsingStatusBadge />);

    const badge = screen.getByTestId('parsing-status-badge');
    expect(badge).toHaveAttribute('aria-live', 'polite');
  });
});
