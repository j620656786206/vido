import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { RestoreConfirmDialog } from './RestoreConfirmDialog';
import type { Backup } from '../../services/backupService';

const testBackup: Backup = {
  id: 'b1',
  filename: 'vido-backup-20260320-140000-v17.tar.gz',
  sizeBytes: 52428800,
  schemaVersion: 17,
  checksum: 'abc123',
  status: 'completed',
  createdAt: '2026-03-20T14:00:00Z',
};

describe('RestoreConfirmDialog', () => {
  it('renders dialog with backup filename', () => {
    render(
      React.createElement(RestoreConfirmDialog, {
        backup: testBackup,
        isRestoring: false,
        onConfirm: vi.fn(),
        onCancel: vi.fn(),
      })
    );
    expect(screen.getByTestId('restore-confirm-dialog')).toBeInTheDocument();
    expect(screen.getByTestId('restore-filename')).toHaveTextContent(testBackup.filename);
    expect(screen.getByText('確認還原', { selector: 'h3' })).toBeInTheDocument();
  });

  it('calls onConfirm when confirm button is clicked', async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    render(
      React.createElement(RestoreConfirmDialog, {
        backup: testBackup,
        isRestoring: false,
        onConfirm,
        onCancel: vi.fn(),
      })
    );
    await user.click(screen.getByTestId('restore-confirm-btn'));
    expect(onConfirm).toHaveBeenCalled();
  });

  it('calls onCancel when cancel button is clicked', async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    render(
      React.createElement(RestoreConfirmDialog, {
        backup: testBackup,
        isRestoring: false,
        onConfirm: vi.fn(),
        onCancel,
      })
    );
    await user.click(screen.getByTestId('restore-cancel-btn'));
    expect(onCancel).toHaveBeenCalled();
  });

  it('disables buttons when isRestoring is true', () => {
    render(
      React.createElement(RestoreConfirmDialog, {
        backup: testBackup,
        isRestoring: true,
        onConfirm: vi.fn(),
        onCancel: vi.fn(),
      })
    );
    expect(screen.getByTestId('restore-confirm-btn')).toBeDisabled();
    expect(screen.getByTestId('restore-cancel-btn')).toBeDisabled();
  });

  it('shows loading state when restoring', () => {
    render(
      React.createElement(RestoreConfirmDialog, {
        backup: testBackup,
        isRestoring: true,
        onConfirm: vi.fn(),
        onCancel: vi.fn(),
      })
    );
    expect(screen.getByText('還原中...')).toBeInTheDocument();
  });

  it('shows warning about data replacement', () => {
    render(
      React.createElement(RestoreConfirmDialog, {
        backup: testBackup,
        isRestoring: false,
        onConfirm: vi.fn(),
        onCancel: vi.fn(),
      })
    );
    expect(screen.getByText(/取代目前所有的資料/)).toBeInTheDocument();
    expect(screen.getByText(/自動建立目前資料的快照/)).toBeInTheDocument();
  });
});
