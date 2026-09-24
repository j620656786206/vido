import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, afterEach } from 'vitest';
import { RestoreConfirmDialog } from './RestoreConfirmDialog';

const h = vi.hoisted(() => ({ isPhone: false }));
vi.mock('../../hooks/useIsPhone', () => ({ useIsPhone: () => h.isPhone }));
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
    const onConfirm = vi.fn();
    const onCancel = vi.fn();
    render(
      React.createElement(RestoreConfirmDialog, {
        backup: testBackup,
        isRestoring: true,
        onConfirm,
        onCancel,
      })
    );
    // dsr-3f CR: aria-disabled (keeps focus) instead of disabled; clicks do nothing.
    for (const id of ['restore-confirm-btn', 'restore-cancel-btn']) {
      const b = screen.getByTestId(id);
      expect(b).toHaveAttribute('aria-disabled', 'true');
      b.click();
    }
    expect(onConfirm).not.toHaveBeenCalled();
    expect(onCancel).not.toHaveBeenCalled();
    // ✕ could not close it anyway, so it is not offered.
    expect(screen.getByRole('button', { name: 'Close' }).className).toContain('hidden');
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

  describe('dsr-3f: a real dialog', () => {
    const renderDialog = (over = {}) => {
      const props = {
        backup: { ...testBackup, createdAt: '2026-09-10T03:00:00' },
        isRestoring: false,
        onConfirm: vi.fn(),
        onCancel: vi.fn(),
        ...over,
      };
      render(React.createElement(RestoreConfirmDialog, props));
      return props;
    };
    afterEach(() => {
      h.isPhone = false;
    });

    it('is a named modal dialog that starts on 取消, not on the destructive button', () => {
      renderDialog();
      const dialog = screen.getByRole('dialog', { name: '確認還原' });
      expect(dialog).toBeInTheDocument();
      expect(screen.getByTestId('restore-cancel-btn')).toHaveFocus();
    });

    it('Esc cancels', async () => {
      const user = userEvent.setup();
      const p = renderDialog();
      await user.keyboard('{Escape}');
      expect(p.onCancel).toHaveBeenCalledTimes(1);
      expect(p.onConfirm).not.toHaveBeenCalled();
    });

    it('Esc does nothing while a restore is running', async () => {
      const user = userEvent.setup();
      const p = renderDialog({ isRestoring: true });
      await user.keyboard('{Escape}');
      expect(p.onCancel).not.toHaveBeenCalled();
    });

    it('shows which backup: file name, then size · date, in mono', () => {
      renderDialog();
      expect(screen.getByTestId('restore-file').className).toContain('font-mono');
      expect(screen.getByTestId('restore-file-meta')).toHaveTextContent(
        '50.0 MB · 2026-09-10 03:00'
      );
    });

    it('the snapshot note is an info callout and 取消 is a filled neutral button', () => {
      renderDialog();
      expect(screen.getByTestId('restore-snapshot-note').className).toContain(
        'bg-[var(--info-tint)]'
      );
      expect(screen.getByTestId('restore-cancel-btn').className).toContain(
        'bg-[var(--bg-tertiary)]'
      );
    });

    it('keeps the consequence emphasised', () => {
      renderDialog();
      expect(screen.getByText('這將會取代目前所有的資料').tagName).toBe('STRONG');
    });

    it('desktop: 取消 then 確認還原; no grabber', () => {
      renderDialog();
      const buttons = screen.getByTestId('restore-actions').querySelectorAll('button');
      expect([...buttons].map((b) => b.textContent)).toEqual(['取消', '確認還原']);
      expect(screen.queryByTestId('restore-sheet-grabber')).toBeNull();
    });

    it('phone: a sheet with a grabber, stacked buttons, 確認還原 on top', () => {
      h.isPhone = true;
      renderDialog();
      expect(screen.getByTestId('restore-sheet-grabber')).toBeInTheDocument();
      // 確認還原 is first in the DOM here, yet focus still starts on 取消.
      expect(screen.getByTestId('restore-cancel-btn')).toHaveFocus();
      const actions = screen.getByTestId('restore-actions');
      expect(actions.className).toContain('flex-col');
      expect([...actions.querySelectorAll('button')].map((b) => b.textContent)).toEqual([
        '確認還原',
        '取消',
      ]);
    });
  });

  it('dsr-3f CR: returns focus to the same backup’s 還原 even if the list re-rendered', async () => {
    const user = userEvent.setup();
    const first = document.createElement('button');
    first.dataset.testid = 'restore-btn-b1';
    document.body.appendChild(first);
    first.focus();
    const onCancel = vi.fn();
    const { unmount } = render(
      React.createElement(RestoreConfirmDialog, {
        backup: testBackup,
        isRestoring: false,
        onConfirm: vi.fn(),
        onCancel,
      })
    );
    // The layout flips: the original button goes, an equivalent one appears.
    first.remove();
    const second = document.createElement('button');
    second.dataset.testid = 'restore-btn-b1';
    document.body.appendChild(second);
    await user.keyboard('{Escape}');
    unmount();
    await new Promise((r) => setTimeout(r, 0));
    expect(second).toHaveFocus();
    second.remove();
  });
});
