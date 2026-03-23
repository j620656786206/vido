import { describe, it, expect, vi, beforeEach } from 'vitest';
import { scannerService, ScannerApiError } from './scannerService';

const mockFetch = vi.fn();
global.fetch = mockFetch;

function mockSuccess<T>(data: T) {
  mockFetch.mockResolvedValueOnce({
    ok: true,
    json: async () => ({ success: true, data }),
  });
}

function mockError(status: number, code: string, message: string) {
  mockFetch.mockResolvedValueOnce({
    ok: false,
    status,
    json: async () => ({ success: false, error: { code, message } }),
  });
}

describe('scannerService', () => {
  beforeEach(() => {
    mockFetch.mockReset();
  });

  describe('triggerScan', () => {
    it('sends POST to /scanner/scan', async () => {
      const result = { files_found: 100, files_new: 10, errors: 0, duration: '30s' };
      mockSuccess(result);

      const data = await scannerService.triggerScan();
      expect(data).toEqual(result);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/scanner/scan'),
        expect.objectContaining({ method: 'POST' })
      );
    });

    it('throws ScannerApiError on 409 conflict', async () => {
      mockError(409, 'SCANNER_ALREADY_RUNNING', '掃描已在進行中');

      await expect(scannerService.triggerScan()).rejects.toThrow(ScannerApiError);
      await expect(scannerService.triggerScan().catch((e) => e.code)).resolves.toBe(undefined);
    });
  });

  describe('getScanStatus', () => {
    it('fetches scan status', async () => {
      const status = {
        is_scanning: false,
        files_found: 0,
        files_processed: 0,
        current_file: '',
        percent_done: 0,
        error_count: 0,
        estimated_time: '',
        last_scan_at: '2026-03-22T14:30:00Z',
        last_scan_files: 1247,
        last_scan_duration: '3m12s',
      };
      mockSuccess(status);

      const data = await scannerService.getScanStatus();
      expect(data.last_scan_files).toBe(1247);
    });
  });

  describe('getSchedule', () => {
    it('fetches schedule config', async () => {
      mockSuccess({ frequency: 'hourly' });

      const data = await scannerService.getSchedule();
      expect(data.frequency).toBe('hourly');
    });
  });

  describe('updateSchedule', () => {
    it('sends PUT with frequency', async () => {
      mockSuccess({ frequency: 'daily' });

      const data = await scannerService.updateSchedule('daily');
      expect(data.frequency).toBe('daily');
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/scanner/schedule'),
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify({ frequency: 'daily' }),
        })
      );
    });
  });

  describe('cancelScan', () => {
    it('sends POST to /scanner/cancel', async () => {
      mockSuccess({});

      await scannerService.cancelScan();
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/scanner/cancel'),
        expect.objectContaining({ method: 'POST' })
      );
    });
  });

  describe('error handling', () => {
    it('throws ScannerApiError with code and message', async () => {
      mockError(400, 'SCANNER_SCHEDULE_INVALID', 'Invalid schedule');

      try {
        await scannerService.getSchedule();
        expect.fail('should have thrown');
      } catch (e) {
        expect(e).toBeInstanceOf(ScannerApiError);
        expect((e as ScannerApiError).code).toBe('SCANNER_SCHEDULE_INVALID');
        expect((e as ScannerApiError).message).toBe('Invalid schedule');
      }
    });
  });
});
