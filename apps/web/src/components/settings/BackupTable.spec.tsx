import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, afterEach } from 'vitest';
import { BackupTable, BACKUP_GRID } from './BackupTable';

const h = vi.hoisted(() => ({ isPhone: false }));
vi.mock('../../hooks/useIsPhone', () => ({ useIsPhone: () => h.isPhone }));
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
  it('完成 wears neutral — green is reserved for live states', () => {
    render(
      React.createElement(BackupTable, {
        backups: [completedBackup],
        onDelete: vi.fn(),
        onVerify: vi.fn(),
        onRestore: vi.fn(),
        isDeleting: false,
        isVerifying: false,
        isRestoring: false,
      })
    );
    const label = screen.getByText('完成');
    expect(label.className).toContain('text-[var(--text-secondary)]');
    expect(label.className).not.toContain('success');
  });

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

  describe('dsr-3f', () => {
    const props = (backups: Backup[]) => ({
      backups,
      onDelete: vi.fn(),
      onVerify: vi.fn(),
      onRestore: vi.fn(),
      isDeleting: false,
      isVerifying: false,
      isRestoring: false,
    });
    const local = { ...completedBackup, createdAt: '2026-09-11T03:00:00' }; // no offset = local time
    afterEach(() => {
      h.isPhone = false;
    });

    it('phone: one card per backup, restore and delete 44px tall, running only deletable', () => {
      h.isPhone = true;
      render(React.createElement(BackupTable, props([local, runningBackup])));
      expect(screen.getByTestId('backup-table').tagName).toBe('UL');
      expect(screen.queryByTestId('backup-table-head')).toBeNull();
      expect(screen.getByTestId('restore-btn-b1').className).toContain('h-11');
      expect(screen.getByTestId('delete-btn-b1').className).toContain('h-11');
      expect(screen.getByTestId('backup-row-b1')).toHaveTextContent('2026-09-11 03:00');
      expect(screen.queryByTestId('restore-btn-b3')).toBeNull();
      expect(screen.getByTestId('delete-btn-b3')).toBeInTheDocument();
    });

    it('desktop: header and rows share one column template', () => {
      render(React.createElement(BackupTable, props([local])));
      const cols = BACKUP_GRID.split(' ').filter((c) => c.includes('grid-cols'));
      for (const c of cols) {
        expect(screen.getByTestId('backup-table-head').className).toContain(c);
        expect(screen.getByTestId('backup-row-b1').className).toContain(c);
      }
      expect(screen.getByTestId('backup-table-head').className).toContain(
        'bg-[var(--bg-tertiary)]'
      );
      expect(screen.getByTestId('backup-table-head').className).toContain(
        'text-[var(--text-muted)]'
      );
    });

    it('stamps the time as YYYY-MM-DD HH:mm in a <time>', () => {
      render(React.createElement(BackupTable, props([local])));
      const t = screen.getByText('2026-09-11 03:00');
      expect(t.tagName).toBe('TIME');
      expect(t).toHaveAttribute('dateTime', '2026-09-11T03:00:00');
    });

    it('the 完成 pill is 12px semibold with a check, never the old 11px', () => {
      render(React.createElement(BackupTable, props([local])));
      const pill = screen.getByText('完成');
      expect(pill.className).toContain('text-xs');
      expect(pill.className).toContain('font-semibold');
      expect(pill.className).not.toContain('text-[11px]');
      expect(pill.className).toContain('bg-[var(--bg-tertiary)]');
      expect(pill.querySelector('svg')).not.toBeNull();
    });

    it('the delete button is error-coloured and every action names its backup', () => {
      render(React.createElement(BackupTable, props([local])));
      expect(screen.getByTestId('delete-btn-b1').className).toContain('text-[var(--error-text)]');
      for (const verb of ['還原', '驗證完整性', '下載', '刪除']) {
        expect(
          screen.getByRole(verb === '下載' ? 'link' : 'button', {
            name: `${verb} ${local.filename}`,
          })
        ).toBeInTheDocument();
      }
    });

    it('a narrow column gets cards even when the viewport is not a phone (sidebar at 768)', () => {
      const RO = window.ResizeObserver;
      const rect = HTMLElement.prototype.getBoundingClientRect;
      class FakeRO {
        observe() {}
        disconnect() {}
      }
      window.ResizeObserver = FakeRO as unknown as typeof window.ResizeObserver;
      HTMLElement.prototype.getBoundingClientRect = function () {
        return { width: 480 } as ReturnType<HTMLElement['getBoundingClientRect']>;
      };
      try {
        render(React.createElement(BackupTable, props([local])));
        expect(screen.getByTestId('backup-list')).toHaveAttribute('data-layout', 'cards');
        expect(screen.getByTestId('restore-btn-b1').className).toContain('h-11');
      } finally {
        window.ResizeObserver = RO;
        HTMLElement.prototype.getBoundingClientRect = rect;
      }
    });

    it('a wide column keeps the table', () => {
      const RO = window.ResizeObserver;
      const rect = HTMLElement.prototype.getBoundingClientRect;
      window.ResizeObserver = class {
        observe() {}
        disconnect() {}
      } as unknown as typeof window.ResizeObserver;
      HTMLElement.prototype.getBoundingClientRect = function () {
        return { width: 900 } as ReturnType<HTMLElement['getBoundingClientRect']>;
      };
      try {
        render(React.createElement(BackupTable, props([local])));
        expect(screen.getByTestId('backup-list')).toHaveAttribute('data-layout', 'table');
      } finally {
        window.ResizeObserver = RO;
        HTMLElement.prototype.getBoundingClientRect = rect;
      }
    });
  });
});
