/**
 * RetryNotifications Component Tests (Story 3.11 - Task 9)
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, act, renderHook } from '@testing-library/react';
import { RetryNotifications, useRetryNotifications } from './RetryNotifications';

describe('RetryNotifications', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders nothing when no notifications', () => {
    const onDismiss = vi.fn();
    const { container } = render(
      <RetryNotifications notifications={[]} onDismiss={onDismiss} />
    );

    expect(container).toBeEmptyDOMElement();
  });

  it('renders notifications', () => {
    const onDismiss = vi.fn();
    const notifications = [
      { id: 'n1', type: 'success' as const, message: 'Success message' },
      { id: 'n2', type: 'error' as const, message: 'Error message' },
    ];

    render(<RetryNotifications notifications={notifications} onDismiss={onDismiss} />);

    expect(screen.getByTestId('retry-notifications')).toBeInTheDocument();
    expect(screen.getByTestId('notification-n1')).toBeInTheDocument();
    expect(screen.getByTestId('notification-n2')).toBeInTheDocument();
    expect(screen.getByText('Success message')).toBeInTheDocument();
    expect(screen.getByText('Error message')).toBeInTheDocument();
  });

  it('renders notification with description', () => {
    const onDismiss = vi.fn();
    const notifications = [
      {
        id: 'n1',
        type: 'success' as const,
        message: 'Success',
        description: 'Task completed',
      },
    ];

    render(<RetryNotifications notifications={notifications} onDismiss={onDismiss} />);

    expect(screen.getByText('Success')).toBeInTheDocument();
    expect(screen.getByText('Task completed')).toBeInTheDocument();
  });

  it('calls onDismiss when close button clicked', async () => {
    const onDismiss = vi.fn();
    const notifications = [
      { id: 'n1', type: 'success' as const, message: 'Success' },
    ];

    render(<RetryNotifications notifications={notifications} onDismiss={onDismiss} />);

    const closeButton = screen.getByRole('button', { name: '關閉通知' });
    fireEvent.click(closeButton);

    // Wait for the dismiss animation
    act(() => {
      vi.advanceTimersByTime(500);
    });

    expect(onDismiss).toHaveBeenCalledWith('n1');
  });

  it('auto-dismisses after duration', () => {
    const onDismiss = vi.fn();
    const notifications = [
      { id: 'n1', type: 'success' as const, message: 'Success', duration: 3000 },
    ];

    render(<RetryNotifications notifications={notifications} onDismiss={onDismiss} />);

    expect(onDismiss).not.toHaveBeenCalled();

    // Fast-forward past duration + animation time
    act(() => {
      vi.advanceTimersByTime(3500);
    });

    expect(onDismiss).toHaveBeenCalledWith('n1');
  });

  it('renders different notification types with correct styles', () => {
    const onDismiss = vi.fn();
    const notifications = [
      { id: 'success', type: 'success' as const, message: 'Success' },
      { id: 'error', type: 'error' as const, message: 'Error' },
      { id: 'warning', type: 'warning' as const, message: 'Warning' },
      { id: 'info', type: 'info' as const, message: 'Info' },
    ];

    render(<RetryNotifications notifications={notifications} onDismiss={onDismiss} />);

    expect(screen.getByTestId('notification-success')).toHaveClass('bg-green-500/10');
    expect(screen.getByTestId('notification-error')).toHaveClass('bg-red-500/10');
    expect(screen.getByTestId('notification-warning')).toHaveClass('bg-yellow-500/10');
    expect(screen.getByTestId('notification-info')).toHaveClass('bg-blue-500/10');
  });

  it('applies custom className', () => {
    const onDismiss = vi.fn();
    const notifications = [
      { id: 'n1', type: 'success' as const, message: 'Success' },
    ];

    render(
      <RetryNotifications
        notifications={notifications}
        onDismiss={onDismiss}
        className="custom-class"
      />
    );

    expect(screen.getByTestId('retry-notifications')).toHaveClass('custom-class');
  });
});

describe('useRetryNotifications', () => {
  it('starts with empty notifications', () => {
    const { result } = renderHook(() => useRetryNotifications());
    expect(result.current.notifications).toHaveLength(0);
  });

  it('adds notifications', () => {
    const { result } = renderHook(() => useRetryNotifications());

    act(() => {
      result.current.addNotification('success', 'Test message');
    });

    expect(result.current.notifications).toHaveLength(1);
    expect(result.current.notifications[0].type).toBe('success');
    expect(result.current.notifications[0].message).toBe('Test message');
  });

  it('dismisses notifications', () => {
    const { result } = renderHook(() => useRetryNotifications());

    let notificationId: string;
    act(() => {
      notificationId = result.current.addNotification('success', 'Test');
    });

    expect(result.current.notifications).toHaveLength(1);

    act(() => {
      result.current.dismissNotification(notificationId);
    });

    expect(result.current.notifications).toHaveLength(0);
  });

  it('shows retry success notification', () => {
    const { result } = renderHook(() => useRetryNotifications());

    act(() => {
      result.current.showRetrySuccess('task-123');
    });

    expect(result.current.notifications).toHaveLength(1);
    expect(result.current.notifications[0].type).toBe('success');
    expect(result.current.notifications[0].message).toBe('重試成功');
    expect(result.current.notifications[0].description).toContain('task-123');
  });

  it('shows retry exhausted notification', () => {
    const { result } = renderHook(() => useRetryNotifications());

    act(() => {
      result.current.showRetryExhausted('task-123');
    });

    expect(result.current.notifications).toHaveLength(1);
    expect(result.current.notifications[0].type).toBe('warning');
    expect(result.current.notifications[0].message).toBe('重試次數已用盡');
    expect(result.current.notifications[0].description).toContain('task-123');
  });

  it('shows retry cancelled notification', () => {
    const { result } = renderHook(() => useRetryNotifications());

    act(() => {
      result.current.showRetryCancelled();
    });

    expect(result.current.notifications).toHaveLength(1);
    expect(result.current.notifications[0].message).toBe('已取消重試');
  });

  it('shows retry triggered notification', () => {
    const { result } = renderHook(() => useRetryNotifications());

    act(() => {
      result.current.showRetryTriggered();
    });

    expect(result.current.notifications).toHaveLength(1);
    expect(result.current.notifications[0].message).toBe('已觸發立即重試');
  });
});
