import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BackupManagement } from './BackupManagement';

// 建立失敗 links to 系統日誌; stub Link so this stays a component test.
vi.mock('@tanstack/react-router', () => ({
  Link: ({ to, children, ...rest }: { to: string; children: React.ReactNode }) =>
    React.createElement('a', { href: to, ...rest }, children),
}));

vi.mock('../../hooks/useBackups', () => ({
  useBackups: vi.fn(),
  useCreateBackup: vi.fn(),
  useDeleteBackup: vi.fn(),
  useVerifyBackup: vi.fn(),
  useRestoreBackup: vi.fn(),
  useBackupSchedule: vi.fn(),
  useUpdateSchedule: vi.fn(),
  useExport: vi.fn(),
}));

import {
  useBackups,
  useCreateBackup,
  useDeleteBackup,
  useVerifyBackup,
  useRestoreBackup,
  useBackupSchedule,
  useUpdateSchedule,
  useExport,
} from '../../hooks/useBackups';

const mockUseBackups = vi.mocked(useBackups);
const mockUseCreateBackup = vi.mocked(useCreateBackup);
const mockUseDeleteBackup = vi.mocked(useDeleteBackup);
const mockUseVerifyBackup = vi.mocked(useVerifyBackup);
const mockUseRestoreBackup = vi.mocked(useRestoreBackup);
const mockUseBackupSchedule = vi.mocked(useBackupSchedule);
const mockUseUpdateSchedule = vi.mocked(useUpdateSchedule);
const mockUseExport = vi.mocked(useExport);

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
  mockUseExport.mockReturnValue({
    mutateAsync: vi.fn().mockResolvedValue({ status: 'completed', itemCount: 5 }),
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
    expect(screen.getByText('無法載入備份資訊')).toBeInTheDocument();
    expect(screen.getByText('與後端的連線中斷了。已存在的備份檔不受影響。')).toBeInTheDocument();
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
    expect(screen.getByTestId('backup-summary')).toHaveTextContent('1 份備份');
  });

  it('renders header text', () => {
    mockUseBackups.mockReturnValue({
      data: { backups: [], totalSizeBytes: 0 },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(BackupManagement));
    // Title/description live at the route level now (one header contract);
    // the component's own anchor is its testid + primary action.
    expect(screen.getByTestId('backup-management')).toBeInTheDocument();
    expect(screen.getByTestId('create-backup-btn')).toBeInTheDocument();
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
    // dsr-3f: the backend's (English, unclassified) message stays off the page.
    expect(screen.queryByText('Disk full')).toBeNull();
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

  // dsr-3f: inverted — the load failure says so in plain words (C16 shape).
  it('[P2] does not show the error message text from the API error', () => {
    mockUseBackups.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('Connection refused'),
    } as any);

    renderWithQuery(React.createElement(BackupManagement));
    expect(screen.queryByText('Connection refused')).toBeNull();
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
    expect(screen.getByTestId('backup-summary')).toHaveTextContent('2 份備份');
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

    it('dsr-3f: Esc closes the dialog and focus goes back to the restore button', async () => {
      const user = userEvent.setup();
      mockUseBackups.mockReturnValue({ data: backupData, isLoading: false, error: null } as any);
      renderWithQuery(React.createElement(BackupManagement));
      await user.click(screen.getByTestId('restore-btn-b1'));
      expect(screen.getByRole('dialog')).toBeInTheDocument();
      await user.keyboard('{Escape}');
      expect(screen.queryByRole('dialog')).toBeNull();
      await waitFor(() => expect(screen.getByTestId('restore-btn-b1')).toHaveFocus());
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

  describe('dsr-3f', () => {
    const one = {
      backups: [
        {
          id: 'b1',
          filename: 'vido-backup.tar.gz',
          sizeBytes: 52428800,
          schemaVersion: 17,
          checksum: 'x',
          status: 'completed' as const,
          createdAt: '2026-09-11T03:00:00',
        },
      ],
      totalSizeBytes: 52428800,
    };

    it('summary reads「N 份備份 · 已使用 X」with a drive icon', () => {
      mockUseBackups.mockReturnValue({ data: one, isLoading: false, error: null } as any);
      renderWithQuery(React.createElement(BackupManagement));
      const summary = screen.getByTestId('backup-summary');
      expect(summary).toHaveTextContent(/^1 份備份 · 已使用 50\.0 MB$/);
      expect(summary.querySelector('svg')).not.toBeNull();
    });

    it('建立失敗: above the summary, plain words, a link to 系統日誌, and 重試 runs create again', async () => {
      const user = userEvent.setup();
      const mutateAsync = vi.fn().mockRejectedValue(new Error('BACKUP_CREATE_FAILED'));
      mockUseCreateBackup.mockReturnValue({ mutateAsync, isPending: false } as any);
      mockUseBackups.mockReturnValue({ data: one, isLoading: false, error: null } as any);
      renderWithQuery(React.createElement(BackupManagement));
      await user.click(screen.getByTestId('create-backup-btn'));

      const bar = screen.getByRole('alert');
      expect(bar).toHaveAttribute('data-testid', 'create-error');
      expect(bar).toHaveTextContent('建立備份失敗');
      expect(bar).toHaveTextContent(
        '備份沒有建立成功。請稍後再試；若持續失敗，到「系統日誌」查看原因。'
      );
      expect(bar).not.toHaveTextContent('BACKUP_CREATE_FAILED');
      expect(screen.getByRole('link', { name: '系統日誌' })).toHaveAttribute(
        'href',
        '/settings/logs'
      );
      expect(
        bar.compareDocumentPosition(screen.getByTestId('backup-summary')) &
          Node.DOCUMENT_POSITION_FOLLOWING
      ).toBeTruthy();
      // The table still shows under the failure.
      expect(screen.getByTestId('backup-table')).toBeInTheDocument();

      await user.click(screen.getByTestId('create-retry-btn'));
      expect(mutateAsync).toHaveBeenCalledTimes(2);
    });

    it('a successful retry clears the failure', async () => {
      const user = userEvent.setup();
      const mutateAsync = vi.fn().mockRejectedValueOnce(new Error('x')).mockResolvedValueOnce({});
      mockUseCreateBackup.mockReturnValue({ mutateAsync, isPending: false } as any);
      mockUseBackups.mockReturnValue({ data: one, isLoading: false, error: null } as any);
      renderWithQuery(React.createElement(BackupManagement));
      await user.click(screen.getByTestId('create-backup-btn'));
      await user.click(screen.getByTestId('create-retry-btn'));
      expect(screen.queryByTestId('create-error')).toBeNull();
    });

    it('load failure: 重試 refetches; mid-refetch stays on the error page as 重試中…', async () => {
      const user = userEvent.setup();
      const refetch = vi.fn();
      mockUseBackups.mockReturnValue({
        data: undefined,
        isFetched: true,
        isFetching: false,
        error: new Error('boom'),
        refetch,
      } as any);
      const { rerender } = renderWithQuery(React.createElement(BackupManagement));
      await user.click(screen.getByRole('button', { name: '重試' }));
      expect(refetch).toHaveBeenCalled();

      // What TanStack reports while refetching with nothing cached.
      mockUseBackups.mockReturnValue({
        data: undefined,
        isLoading: true,
        isFetched: true,
        isFetching: true,
        error: null,
        refetch,
      } as any);
      rerender(
        React.createElement(
          QueryClientProvider,
          { client: new QueryClient() },
          React.createElement(BackupManagement)
        )
      );
      expect(screen.queryByTestId('backup-loading')).toBeNull();
      expect(screen.getByRole('button', { name: '重試中…' })).toBeInTheDocument();
    });
  });
});
