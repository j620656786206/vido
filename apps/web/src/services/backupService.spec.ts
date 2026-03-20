import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { backupService } from './backupService';

const mockFetch = vi.fn();
global.fetch = mockFetch;

beforeEach(() => {
  mockFetch.mockReset();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe('backupService', () => {
  describe('listBackups', () => {
    it('[P1] returns backup list on success', async () => {
      const mockData = {
        backups: [{ id: 'b1', filename: 'backup.tar.gz', status: 'completed' }],
        totalSizeBytes: 1024,
      };
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true, data: mockData }),
      });

      const result = await backupService.listBackups();
      expect(result).toEqual(mockData);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/settings/backups'),
        expect.objectContaining({ headers: { 'Content-Type': 'application/json' } })
      );
    });

    it('[P1] throws error on API failure', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        json: () =>
          Promise.resolve({
            success: false,
            error: { code: 'BACKUP_CREATE_FAILED', message: 'Server error' },
          }),
      });

      await expect(backupService.listBackups()).rejects.toThrow('Server error');
    });
  });

  describe('createBackup', () => {
    it('[P1] sends POST request', async () => {
      const mockBackup = { id: 'b1', filename: 'backup.tar.gz', status: 'completed' };
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true, data: mockBackup }),
      });

      const result = await backupService.createBackup();
      expect(result).toEqual(mockBackup);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/settings/backups'),
        expect.objectContaining({ method: 'POST' })
      );
    });

    it('[P1] throws on backup in progress', async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 409,
        json: () =>
          Promise.resolve({
            success: false,
            error: { code: 'BACKUP_IN_PROGRESS', message: 'Another backup is already running' },
          }),
      });

      await expect(backupService.createBackup()).rejects.toThrow(
        'Another backup is already running'
      );
    });
  });

  describe('deleteBackup', () => {
    it('[P1] sends DELETE request with encoded ID', async () => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true, data: { deleted: true } }),
      });

      await backupService.deleteBackup('b1');
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/settings/backups/b1'),
        expect.objectContaining({ method: 'DELETE' })
      );
    });
  });

  describe('getDownloadUrl', () => {
    it('[P1] returns correct download URL', () => {
      const url = backupService.getDownloadUrl('b1');
      expect(url).toContain('/settings/backups/b1/download');
    });
  });
});
