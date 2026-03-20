import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { BackupTable } from './BackupTable';
import type { Backup } from '../../services/backupService';

const completedBackup: Backup = {
  id: 'b1',
  filename: 'vido-backup-20260320-140000-v17.tar.gz',
  sizeBytes: 52428800,
  schemaVersion: 17,
  checksum: 'abc123',
  status: 'completed',
  createdAt: '2026-03-20T14:00:00Z',
};

const failedBackup: Backup = {
  id: 'b2',
  filename: 'vido-backup-20260319-030000-v17.tar.gz',
  sizeBytes: 0,
  schemaVersion: 17,
  checksum: '',
  status: 'failed',
  errorMessage: 'disk full',
  createdAt: '2026-03-19T03:00:00Z',
};

const runningBackup: Backup = {
  id: 'b3',
  filename: 'vido-backup-20260320-150000-v17.tar.gz',
  sizeBytes: 0,
  schemaVersion: 17,
  checksum: '',
  status: 'running',
  createdAt: '2026-03-20T15:00:00Z',
};

describe('BackupTable', () => {
  it('renders table header', () => {
    const onDelete = vi.fn();
    render(
      React.createElement(BackupTable, {
        backups: [completedBackup],
        onDelete,
        onVerify: vi.fn(),
        onRestore: vi.fn(),
        isDeleting: false,
        isVerifying: false,
        isRestoring: false,
      })
    );
    expect(screen.getByText('檔案名稱')).toBeInTheDocument();
    expect(screen.getByText('大小')).toBeInTheDocument();
    expect(screen.getByText('建立時間')).toBeInTheDocument();
    expect(screen.getByText('狀態')).toBeInTheDocument();
    expect(screen.getByText('操作')).toBeInTheDocument();
  });

  it('renders completed backup row with download button', () => {
    const onDelete = vi.fn();
    render(
      React.createElement(BackupTable, {
        backups: [completedBackup],
        onDelete,
        onVerify: vi.fn(),
        onRestore: vi.fn(),
        isDeleting: false,
        isVerifying: false,
        isRestoring: false,
      })
    );
    expect(screen.getByTestId('backup-row-b1')).toBeInTheDocument();
    expect(screen.getByText(completedBackup.filename)).toBeInTheDocument();
    expect(screen.getByText('50.0 MB')).toBeInTheDocument();
    expect(screen.getByText('完成')).toBeInTheDocument();
    expect(screen.getByTestId('download-btn-b1')).toBeInTheDocument();
    expect(screen.getByTestId('delete-btn-b1')).toBeInTheDocument();
  });

  it('renders failed backup without download button', () => {
    const onDelete = vi.fn();
    render(
      React.createElement(BackupTable, {
        backups: [failedBackup],
        onDelete,
        onVerify: vi.fn(),
        onRestore: vi.fn(),
        isDeleting: false,
        isVerifying: false,
        isRestoring: false,
      })
    );
    expect(screen.getByText('失敗')).toBeInTheDocument();
    expect(screen.queryByTestId('download-btn-b2')).not.toBeInTheDocument();
    expect(screen.getByTestId('delete-btn-b2')).toBeInTheDocument();
  });

  it('renders running backup status', () => {
    const onDelete = vi.fn();
    render(
      React.createElement(BackupTable, {
        backups: [runningBackup],
        onDelete,
        onVerify: vi.fn(),
        onRestore: vi.fn(),
        isDeleting: false,
        isVerifying: false,
        isRestoring: false,
      })
    );
    expect(screen.getByText('執行中')).toBeInTheDocument();
  });

  it('calls onDelete when delete button is clicked', async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn();
    render(
      React.createElement(BackupTable, {
        backups: [completedBackup],
        onDelete,
        onVerify: vi.fn(),
        onRestore: vi.fn(),
        isDeleting: false,
        isVerifying: false,
        isRestoring: false,
      })
    );
    await user.click(screen.getByTestId('delete-btn-b1'));
    expect(onDelete).toHaveBeenCalledWith('b1');
  });

  it('renders multiple backup rows', () => {
    const onDelete = vi.fn();
    render(
      React.createElement(BackupTable, {
        backups: [completedBackup, failedBackup, runningBackup],
        onDelete,
        onVerify: vi.fn(),
        onRestore: vi.fn(),
        isDeleting: false,
        isVerifying: false,
        isRestoring: false,
      })
    );
    expect(screen.getByTestId('backup-row-b1')).toBeInTheDocument();
    expect(screen.getByTestId('backup-row-b2')).toBeInTheDocument();
    expect(screen.getByTestId('backup-row-b3')).toBeInTheDocument();
  });

  it('disables delete button when isDeleting is true', () => {
    const onDelete = vi.fn();
    render(
      React.createElement(BackupTable, {
        backups: [completedBackup],
        onDelete,
        onVerify: vi.fn(),
        isDeleting: true,
        isVerifying: false,
      })
    );
    expect(screen.getByTestId('delete-btn-b1')).toBeDisabled();
  });

  it('[P2] renders pending backup status', () => {
    const onDelete = vi.fn();
    const pendingBackup: Backup = {
      id: 'b4',
      filename: 'vido-backup-20260320-160000-v17.tar.gz',
      sizeBytes: 0,
      schemaVersion: 17,
      checksum: '',
      status: 'pending',
      createdAt: '2026-03-20T16:00:00Z',
    };
    render(
      React.createElement(BackupTable, {
        backups: [pendingBackup],
        onDelete,
        onVerify: vi.fn(),
        onRestore: vi.fn(),
        isDeleting: false,
        isVerifying: false,
        isRestoring: false,
      })
    );
    expect(screen.getByText('等待中')).toBeInTheDocument();
    expect(screen.queryByTestId('download-btn-b4')).not.toBeInTheDocument();
  });

  it('[P1] download link points to correct API endpoint', () => {
    const onDelete = vi.fn();
    render(
      React.createElement(BackupTable, {
        backups: [completedBackup],
        onDelete,
        onVerify: vi.fn(),
        onRestore: vi.fn(),
        isDeleting: false,
        isVerifying: false,
        isRestoring: false,
      })
    );
    const downloadLink = screen.getByTestId('download-btn-b1');
    expect(downloadLink).toHaveAttribute(
      'href',
      expect.stringContaining('/settings/backups/b1/download')
    );
  });

  it('[P1] calls onVerify when verify button is clicked', async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn();
    const onVerify = vi.fn();
    render(
      React.createElement(BackupTable, {
        backups: [completedBackup],
        onDelete,
        onVerify,
        isDeleting: false,
        isVerifying: false,
      })
    );
    await user.click(screen.getByTestId('verify-btn-b1'));
    expect(onVerify).toHaveBeenCalledWith('b1');
  });

  it('[P1] renders corrupted backup status', () => {
    const onDelete = vi.fn();
    const corruptedBackup: Backup = {
      id: 'b5',
      filename: 'vido-backup-20260318-030000-v17.tar.gz',
      sizeBytes: 52000000,
      schemaVersion: 17,
      checksum: 'abc123',
      status: 'corrupted',
      errorMessage: 'Checksum mismatch detected',
      createdAt: '2026-03-18T03:00:00Z',
    };
    render(
      React.createElement(BackupTable, {
        backups: [corruptedBackup],
        onDelete,
        onVerify: vi.fn(),
        onRestore: vi.fn(),
        isDeleting: false,
        isVerifying: false,
        isRestoring: false,
      })
    );
    expect(screen.getByText('已損壞')).toBeInTheDocument();
    expect(screen.queryByTestId('verify-btn-b5')).not.toBeInTheDocument();
    expect(screen.queryByTestId('download-btn-b5')).not.toBeInTheDocument();
  });

  it('[P2] disables verify button when isVerifying is true', () => {
    const onDelete = vi.fn();
    render(
      React.createElement(BackupTable, {
        backups: [completedBackup],
        onDelete,
        onVerify: vi.fn(),
        onRestore: vi.fn(),
        isDeleting: false,
        isVerifying: true,
        isRestoring: false,
      })
    );
    expect(screen.getByTestId('verify-btn-b1')).toBeDisabled();
  });

  it('[P1] renders restore button for completed backups', () => {
    const onDelete = vi.fn();
    render(
      React.createElement(BackupTable, {
        backups: [completedBackup],
        onDelete,
        onVerify: vi.fn(),
        onRestore: vi.fn(),
        isDeleting: false,
        isVerifying: false,
        isRestoring: false,
      })
    );
    expect(screen.getByTestId('restore-btn-b1')).toBeInTheDocument();
  });

  it('[P1] calls onRestore when restore button is clicked', async () => {
    const user = userEvent.setup();
    const onRestore = vi.fn();
    render(
      React.createElement(BackupTable, {
        backups: [completedBackup],
        onDelete: vi.fn(),
        onVerify: vi.fn(),
        onRestore,
        isDeleting: false,
        isVerifying: false,
        isRestoring: false,
      })
    );
    await user.click(screen.getByTestId('restore-btn-b1'));
    expect(onRestore).toHaveBeenCalledWith('b1');
  });

  it('[P2] does not show restore button for failed backups', () => {
    render(
      React.createElement(BackupTable, {
        backups: [failedBackup],
        onDelete: vi.fn(),
        onVerify: vi.fn(),
        onRestore: vi.fn(),
        isDeleting: false,
        isVerifying: false,
        isRestoring: false,
      })
    );
    expect(screen.queryByTestId('restore-btn-b2')).not.toBeInTheDocument();
  });

  it('[P2] disables restore button when isRestoring is true', () => {
    render(
      React.createElement(BackupTable, {
        backups: [completedBackup],
        onDelete: vi.fn(),
        onVerify: vi.fn(),
        onRestore: vi.fn(),
        isDeleting: false,
        isVerifying: false,
        isRestoring: true,
      })
    );
    expect(screen.getByTestId('restore-btn-b1')).toBeDisabled();
  });
});
