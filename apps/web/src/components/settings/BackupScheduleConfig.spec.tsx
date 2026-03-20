import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BackupScheduleConfig } from './BackupScheduleConfig';

vi.mock('../../hooks/useBackups', () => ({
  useBackupSchedule: vi.fn(),
  useUpdateSchedule: vi.fn(),
}));

import { useBackupSchedule, useUpdateSchedule } from '../../hooks/useBackups';

const mockUseBackupSchedule = vi.mocked(useBackupSchedule);
const mockUseUpdateSchedule = vi.mocked(useUpdateSchedule);

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(React.createElement(QueryClientProvider, { client: queryClient }, ui));
}

beforeEach(() => {
  mockUseUpdateSchedule.mockReturnValue({
    mutateAsync: vi.fn().mockResolvedValue({}),
    isPending: false,
  } as any);
});

describe('BackupScheduleConfig', () => {
  it('renders loading state', () => {
    mockUseBackupSchedule.mockReturnValue({
      data: undefined,
      isLoading: true,
    } as any);

    renderWithQuery(React.createElement(BackupScheduleConfig));
    expect(screen.getByTestId('schedule-loading')).toBeInTheDocument();
  });

  it('renders disabled state with toggle', () => {
    mockUseBackupSchedule.mockReturnValue({
      data: { enabled: false, frequency: 'disabled', hour: 3, dayOfWeek: 0 },
      isLoading: false,
    } as any);

    renderWithQuery(React.createElement(BackupScheduleConfig));
    expect(screen.getByTestId('backup-schedule-config')).toBeInTheDocument();
    expect(screen.getByTestId('schedule-toggle')).toBeInTheDocument();
    expect(screen.queryByTestId('schedule-options')).not.toBeInTheDocument();
  });

  it('renders enabled state with options', () => {
    mockUseBackupSchedule.mockReturnValue({
      data: { enabled: true, frequency: 'daily', hour: 3, dayOfWeek: 0 },
      isLoading: false,
    } as any);

    renderWithQuery(React.createElement(BackupScheduleConfig));
    expect(screen.getByTestId('schedule-options')).toBeInTheDocument();
    expect(screen.getByTestId('schedule-frequency')).toBeInTheDocument();
    expect(screen.getByTestId('schedule-hour')).toBeInTheDocument();
    expect(screen.getByTestId('retention-info')).toBeInTheDocument();
  });

  it('shows day selector for weekly frequency', () => {
    mockUseBackupSchedule.mockReturnValue({
      data: { enabled: true, frequency: 'weekly', hour: 3, dayOfWeek: 1 },
      isLoading: false,
    } as any);

    renderWithQuery(React.createElement(BackupScheduleConfig));
    expect(screen.getByTestId('schedule-day')).toBeInTheDocument();
  });

  it('does not show day selector for daily frequency', () => {
    mockUseBackupSchedule.mockReturnValue({
      data: { enabled: true, frequency: 'daily', hour: 3, dayOfWeek: 0 },
      isLoading: false,
    } as any);

    renderWithQuery(React.createElement(BackupScheduleConfig));
    expect(screen.queryByTestId('schedule-day')).not.toBeInTheDocument();
  });

  it('calls updateSchedule when save is clicked', async () => {
    const user = userEvent.setup();
    const mockMutateAsync = vi.fn().mockResolvedValue({});
    mockUseUpdateSchedule.mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
    } as any);
    mockUseBackupSchedule.mockReturnValue({
      data: { enabled: true, frequency: 'daily', hour: 3, dayOfWeek: 0 },
      isLoading: false,
    } as any);

    renderWithQuery(React.createElement(BackupScheduleConfig));
    await user.click(screen.getByTestId('schedule-save-btn'));
    expect(mockMutateAsync).toHaveBeenCalled();
  });

  it('shows success message after save', async () => {
    const user = userEvent.setup();
    mockUseUpdateSchedule.mockReturnValue({
      mutateAsync: vi.fn().mockResolvedValue({}),
      isPending: false,
    } as any);
    mockUseBackupSchedule.mockReturnValue({
      data: { enabled: true, frequency: 'daily', hour: 3, dayOfWeek: 0 },
      isLoading: false,
    } as any);

    renderWithQuery(React.createElement(BackupScheduleConfig));
    await user.click(screen.getByTestId('schedule-save-btn'));
    expect(screen.getByTestId('schedule-message')).toBeInTheDocument();
    expect(screen.getByText(/排程設定已儲存/)).toBeInTheDocument();
  });

  it('shows retention policy info', () => {
    mockUseBackupSchedule.mockReturnValue({
      data: { enabled: true, frequency: 'daily', hour: 3, dayOfWeek: 0 },
      isLoading: false,
    } as any);

    renderWithQuery(React.createElement(BackupScheduleConfig));
    expect(screen.getByText(/7 個每日備份/)).toBeInTheDocument();
    expect(screen.getByText(/4 個每週備份/)).toBeInTheDocument();
  });

  it('shows next backup time when available', () => {
    mockUseBackupSchedule.mockReturnValue({
      data: {
        enabled: true,
        frequency: 'daily',
        hour: 3,
        dayOfWeek: 0,
        nextBackupAt: '2026-03-21T03:00:00Z',
      },
      isLoading: false,
    } as any);

    renderWithQuery(React.createElement(BackupScheduleConfig));
    expect(screen.getByText(/下次備份/)).toBeInTheDocument();
  });
});
