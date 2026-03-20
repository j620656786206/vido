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
      React.createElement(BackupTable, { backups: [completedBackup], onDelete, isDeleting: false })
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
      React.createElement(BackupTable, { backups: [completedBackup], onDelete, isDeleting: false })
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
      React.createElement(BackupTable, { backups: [failedBackup], onDelete, isDeleting: false })
    );
    expect(screen.getByText('失敗')).toBeInTheDocument();
    expect(screen.queryByTestId('download-btn-b2')).not.toBeInTheDocument();
    expect(screen.getByTestId('delete-btn-b2')).toBeInTheDocument();
  });

  it('renders running backup status', () => {
    const onDelete = vi.fn();
    render(
      React.createElement(BackupTable, { backups: [runningBackup], onDelete, isDeleting: false })
    );
    expect(screen.getByText('執行中')).toBeInTheDocument();
  });

  it('calls onDelete when delete button is clicked', async () => {
    const user = userEvent.setup();
    const onDelete = vi.fn();
    render(
      React.createElement(BackupTable, { backups: [completedBackup], onDelete, isDeleting: false })
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
        isDeleting: false,
      })
    );
    expect(screen.getByTestId('backup-row-b1')).toBeInTheDocument();
    expect(screen.getByTestId('backup-row-b2')).toBeInTheDocument();
    expect(screen.getByTestId('backup-row-b3')).toBeInTheDocument();
  });

  it('disables delete button when isDeleting is true', () => {
    const onDelete = vi.fn();
    render(
      React.createElement(BackupTable, { backups: [completedBackup], onDelete, isDeleting: true })
    );
    expect(screen.getByTestId('delete-btn-b1')).toBeDisabled();
  });
});
