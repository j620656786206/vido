import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BackupManagement } from './BackupManagement';

vi.mock('../../hooks/useBackups', () => ({
  useBackups: vi.fn(),
  useCreateBackup: vi.fn(),
  useDeleteBackup: vi.fn(),
  useVerifyBackup: vi.fn(),
  useRestoreBackup: vi.fn(),
  useBackupSchedule: vi.fn(),
  useUpdateSchedule: vi.fn(),
}));

import {
  useBackups,
  useCreateBackup,
  useDeleteBackup,
  useVerifyBackup,
  useRestoreBackup,
  useBackupSchedule,
  useUpdateSchedule,
} from '../../hooks/useBackups';

const mockUseBackups = vi.mocked(useBackups);
const mockUseCreateBackup = vi.mocked(useCreateBackup);
const mockUseDeleteBackup = vi.mocked(useDeleteBackup);
const mockUseVerifyBackup = vi.mocked(useVerifyBackup);
const mockUseRestoreBackup = vi.mocked(useRestoreBackup);
const mockUseBackupSchedule = vi.mocked(useBackupSchedule);
const mockUseUpdateSchedule = vi.mocked(useUpdateSchedule);

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(React.createElement(QueryClientProvider, { client: queryClient }, ui));
}

beforeEach(() => {
  mockUseCreateBackup.mockReturnValue({
    mutateAsync: vi.fn().mockResolvedValue({}),
    isPending: false,
  } as any);
  mockUseDeleteBackup.mockReturnValue({
    mutateAsync: vi.fn().mockResolvedValue(undefined),
    isPending: false,
  } as any);
  mockUseVerifyBackup.mockReturnValue({
    mutateAsync: vi.fn().mockResolvedValue({ match: true, status: 'verified' }),
    isPending: false,
  } as any);
  mockUseRestoreBackup.mockReturnValue({
    mutateAsync: vi.fn().mockResolvedValue({ status: 'completed', message: '還原完成' }),
    isPending: false,
  } as any);
  mockUseBackupSchedule.mockReturnValue({
    data: { enabled: false, frequency: 'disabled', hour: 3, dayOfWeek: 0 },
    isLoading: false,
  } as any);
  mockUseUpdateSchedule.mockReturnValue({
    mutateAsync: vi.fn().mockResolvedValue({}),
    isPending: false,
  } as any);
});

describe('BackupManagement', () => {
  it('renders loading state', () => {
    mockUseBackups.mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    } as any);

    renderWithQuery(React.createElement(BackupManagement));
    expect(screen.getByTestId('backup-loading')).toBeInTheDocument();
  });

  it('renders error state', () => {
    mockUseBackups.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('Network error'),
    } as any);

    renderWithQuery(React.createElement(BackupManagement));
    expect(screen.getByTestId('backup-error')).toBeInTheDocument();
    expect(screen.getByText('無法載入備份資料')).toBeInTheDocument();
  });

  it('renders empty state when no backups', () => {
    mockUseBackups.mockReturnValue({
      data: { backups: [], totalSizeBytes: 0 },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(BackupManagement));
    expect(screen.getByTestId('backup-empty')).toBeInTheDocument();
    expect(screen.getByText('尚未建立任何備份')).toBeInTheDocument();
  });

  it('renders backup table when backups exist', () => {
    mockUseBackups.mockReturnValue({
      data: {
        backups: [
          {
            id: 'b1',
            filename: 'vido-backup-20260320-140000-v17.tar.gz',
            sizeBytes: 52428800,
            schemaVersion: 17,
            checksum: 'abc123',
            status: 'completed',
            createdAt: '2026-03-20T14:00:00Z',
          },
        ],
        totalSizeBytes: 52428800,
      },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(BackupManagement));
    expect(screen.getByTestId('backup-management')).toBeInTheDocument();
    expect(screen.getByTestId('backup-table')).toBeInTheDocument();
    expect(screen.getByTestId('backup-summary')).toHaveTextContent('50.0 MB');
    expect(screen.getByTestId('backup-summary')).toHaveTextContent('1 個備份');
  });

  it('renders header text', () => {
    mockUseBackups.mockReturnValue({
      data: { backups: [], totalSizeBytes: 0 },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(BackupManagement));
    expect(screen.getByText('備份與還原')).toBeInTheDocument();
    expect(screen.getByText('建立與管理 Vido 資料庫備份，確保資料安全')).toBeInTheDocument();
  });

  it('calls createBackup when button is clicked', async () => {
    const user = userEvent.setup();
    const mockMutateAsync = vi.fn().mockResolvedValue({});
    mockUseCreateBackup.mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
    } as any);
    mockUseBackups.mockReturnValue({
      data: { backups: [], totalSizeBytes: 0 },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(BackupManagement));
    await user.click(screen.getByTestId('create-backup-btn'));
    expect(mockMutateAsync).toHaveBeenCalled();
  });

  it('shows error when backup creation fails', async () => {
    const user = userEvent.setup();
    mockUseCreateBackup.mockReturnValue({
      mutateAsync: vi.fn().mockRejectedValue(new Error('Disk full')),
      isPending: false,
    } as any);
    mockUseBackups.mockReturnValue({
      data: { backups: [], totalSizeBytes: 0 },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(BackupManagement));
    await user.click(screen.getByTestId('create-backup-btn'));
    expect(screen.getByTestId('create-error')).toBeInTheDocument();
    expect(screen.getByText('Disk full')).toBeInTheDocument();
  });

  it('[P1] disables create button when backup is in progress', () => {
    mockUseCreateBackup.mockReturnValue({
      mutateAsync: vi.fn(),
      isPending: true,
    } as any);
    mockUseBackups.mockReturnValue({
      data: { backups: [], totalSizeBytes: 0 },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(BackupManagement));
    expect(screen.getByTestId('create-backup-btn')).toBeDisabled();
  });

  it('[P2] shows fallback error message for non-Error rejection', async () => {
    const user = userEvent.setup();
    mockUseCreateBackup.mockReturnValue({
      mutateAsync: vi.fn().mockRejectedValue('string error'),
      isPending: false,
    } as any);
    mockUseBackups.mockReturnValue({
      data: { backups: [], totalSizeBytes: 0 },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(BackupManagement));
    await user.click(screen.getByTestId('create-backup-btn'));
    expect(screen.getByText('建立備份失敗')).toBeInTheDocument();
  });

  it('[P2] shows error message text from API error', () => {
    mockUseBackups.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('Connection refused'),
    } as any);

    renderWithQuery(React.createElement(BackupManagement));
    expect(screen.getByText('Connection refused')).toBeInTheDocument();
  });

  it('[P1] renders correct summary for multiple backups', () => {
    mockUseBackups.mockReturnValue({
      data: {
        backups: [
          {
            id: 'b1',
            filename: 'backup1.tar.gz',
            sizeBytes: 52428800,
            schemaVersion: 17,
            checksum: 'a',
            status: 'completed',
            createdAt: '2026-03-20T14:00:00Z',
          },
          {
            id: 'b2',
            filename: 'backup2.tar.gz',
            sizeBytes: 41943040,
            schemaVersion: 17,
            checksum: 'b',
            status: 'completed',
            createdAt: '2026-03-19T14:00:00Z',
          },
        ],
        totalSizeBytes: 94371840,
      },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(BackupManagement));
    expect(screen.getByTestId('backup-summary')).toHaveTextContent('2 個備份');
  });

  describe('AC1-4: Data restore', () => {
    const backupData = {
      backups: [
        {
          id: 'b1',
          filename: 'vido-backup-20260320-140000-v17.tar.gz',
          sizeBytes: 52428800,
          schemaVersion: 17,
          checksum: 'abc123',
          status: 'completed' as const,
          createdAt: '2026-03-20T14:00:00Z',
        },
      ],
      totalSizeBytes: 52428800,
    };

    it('[P1] shows restore button for completed backups', () => {
      mockUseBackups.mockReturnValue({
        data: backupData,
        isLoading: false,
        error: null,
      } as any);

      renderWithQuery(React.createElement(BackupManagement));
      expect(screen.getByTestId('restore-btn-b1')).toBeInTheDocument();
    });

    it('[P1] opens confirm dialog when restore is clicked', async () => {
      const user = userEvent.setup();
      mockUseBackups.mockReturnValue({
        data: backupData,
        isLoading: false,
        error: null,
      } as any);

      renderWithQuery(React.createElement(BackupManagement));
      await user.click(screen.getByTestId('restore-btn-b1'));
      expect(screen.getByTestId('restore-confirm-dialog')).toBeInTheDocument();
      expect(screen.getByTestId('restore-filename')).toHaveTextContent(
        'vido-backup-20260320-140000-v17.tar.gz'
      );
    });

    it('[P1] closes dialog on cancel', async () => {
      const user = userEvent.setup();
      mockUseBackups.mockReturnValue({
        data: backupData,
        isLoading: false,
        error: null,
      } as any);

      renderWithQuery(React.createElement(BackupManagement));
      await user.click(screen.getByTestId('restore-btn-b1'));
      expect(screen.getByTestId('restore-confirm-dialog')).toBeInTheDocument();

      await user.click(screen.getByTestId('restore-cancel-btn'));
      expect(screen.queryByTestId('restore-confirm-dialog')).not.toBeInTheDocument();
    });

    it('[P1] shows success message after restore completes', async () => {
      const user = userEvent.setup();
      mockUseRestoreBackup.mockReturnValue({
        mutateAsync: vi.fn().mockResolvedValue({ status: 'completed', message: '還原完成' }),
        isPending: false,
      } as any);
      mockUseBackups.mockReturnValue({
        data: backupData,
        isLoading: false,
        error: null,
      } as any);

      renderWithQuery(React.createElement(BackupManagement));
      await user.click(screen.getByTestId('restore-btn-b1'));
      await user.click(screen.getByTestId('restore-confirm-btn'));

      expect(screen.getByTestId('restore-message')).toBeInTheDocument();
      expect(screen.getByText(/還原完成/)).toBeInTheDocument();
    });

    it('[P1] shows error message when restore fails', async () => {
      const user = userEvent.setup();
      mockUseRestoreBackup.mockReturnValue({
        mutateAsync: vi
          .fn()
          .mockResolvedValue({ status: 'failed', error: 'RESTORE_VERIFY_FAILED' }),
        isPending: false,
      } as any);
      mockUseBackups.mockReturnValue({
        data: backupData,
        isLoading: false,
        error: null,
      } as any);

      renderWithQuery(React.createElement(BackupManagement));
      await user.click(screen.getByTestId('restore-btn-b1'));
      await user.click(screen.getByTestId('restore-confirm-btn'));

      expect(screen.getByTestId('restore-message')).toBeInTheDocument();
      expect(screen.getByText(/還原失敗/)).toBeInTheDocument();
    });

    it('[P2] shows error when restore API throws', async () => {
      const user = userEvent.setup();
      mockUseRestoreBackup.mockReturnValue({
        mutateAsync: vi.fn().mockRejectedValue(new Error('Network error')),
        isPending: false,
      } as any);
      mockUseBackups.mockReturnValue({
        data: backupData,
        isLoading: false,
        error: null,
      } as any);

      renderWithQuery(React.createElement(BackupManagement));
      await user.click(screen.getByTestId('restore-btn-b1'));
      await user.click(screen.getByTestId('restore-confirm-btn'));

      expect(screen.getByTestId('restore-message')).toBeInTheDocument();
      expect(screen.getByText('Network error')).toBeInTheDocument();
    });
  });

  describe('AC2/AC3: Backup verification', () => {
    const backupData = {
      backups: [
        {
          id: 'b1',
          filename: 'vido-backup-20260320-140000-v17.tar.gz',
          sizeBytes: 52428800,
          schemaVersion: 17,
          checksum: 'abc123',
          status: 'completed' as const,
          createdAt: '2026-03-20T14:00:00Z',
        },
      ],
      totalSizeBytes: 52428800,
    };

    it('[P1] shows success message when verification passes', async () => {
      const user = userEvent.setup();
      mockUseVerifyBackup.mockReturnValue({
        mutateAsync: vi.fn().mockResolvedValue({ match: true, status: 'verified' }),
        isPending: false,
      } as any);
      mockUseBackups.mockReturnValue({
        data: backupData,
        isLoading: false,
        error: null,
      } as any);

      renderWithQuery(React.createElement(BackupManagement));
      await user.click(screen.getByTestId('verify-btn-b1'));
      expect(screen.getByTestId('verify-message')).toBeInTheDocument();
      expect(screen.getByText(/備份驗證通過/)).toBeInTheDocument();
    });

    it('[P1] shows warning message when verification detects corruption', async () => {
      const user = userEvent.setup();
      mockUseVerifyBackup.mockReturnValue({
        mutateAsync: vi.fn().mockResolvedValue({ match: false, status: 'corrupted' }),
        isPending: false,
      } as any);
      mockUseBackups.mockReturnValue({
        data: backupData,
        isLoading: false,
        error: null,
      } as any);

      renderWithQuery(React.createElement(BackupManagement));
      await user.click(screen.getByTestId('verify-btn-b1'));
      expect(screen.getByTestId('verify-message')).toBeInTheDocument();
      expect(screen.getByText(/備份校驗碼不符/)).toBeInTheDocument();
    });

    it('[P2] shows error message when verification API fails', async () => {
      const user = userEvent.setup();
      mockUseVerifyBackup.mockReturnValue({
        mutateAsync: vi.fn().mockRejectedValue(new Error('File missing')),
        isPending: false,
      } as any);
      mockUseBackups.mockReturnValue({
        data: backupData,
        isLoading: false,
        error: null,
      } as any);

      renderWithQuery(React.createElement(BackupManagement));
      await user.click(screen.getByTestId('verify-btn-b1'));
      expect(screen.getByTestId('verify-message')).toBeInTheDocument();
      expect(screen.getByText('File missing')).toBeInTheDocument();
    });
  });
});
