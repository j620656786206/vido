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
      const result = { filesFound: 100, filesNew: 10, errors: 0, duration: '30s' };
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

      try {
        await scannerService.triggerScan();
        expect.fail('should have thrown');
      } catch (e) {
        expect(e).toBeInstanceOf(ScannerApiError);
        expect((e as ScannerApiError).code).toBe('SCANNER_ALREADY_RUNNING');
        expect((e as ScannerApiError).message).toBe('掃描已在進行中');
      }
    });
  });

  describe('getScanStatus', () => {
    // bugfix-last-scan-never-shown AC #4: the backend's REAL snake_case body
    // (scanner_handler.go scanStatusResponse — ScanProgress flat + last_scan),
    // run through fetchApi's snake→camel transform. The old mock used field
    // names the backend never sent (isScanning, lastScanAt…).
    it('reads the real status body, including last_scan', async () => {
      mockSuccess({
        files_found: 0,
        files_created: 0,
        files_updated: 0,
        files_skipped: 0,
        files_removed: 0,
        files_unmatched: 0,
        error_count: 0,
        current_file: '',
        percent_done: 0,
        is_active: false,
        started_at: '0001-01-01T00:00:00Z',
        last_scan: { completed_at: '2026-03-22T14:30:00Z', files_found: 1247, duration_ms: 192000 },
      });

      const data = await scannerService.getScanStatus();
      expect(data.isActive).toBe(false);
      expect(data.lastScan).toEqual({
        completedAt: '2026-03-22T14:30:00Z',
        filesFound: 1247,
        durationMs: 192000,
      });
    });

    it('never scanned → lastScan is null', async () => {
      mockSuccess({ is_active: false, files_found: 0, last_scan: null });
      const data = await scannerService.getScanStatus();
      expect(data.lastScan).toBeNull();
    });
  });

  describe('getSchedule', () => {
    // bugfix-scan-schedule-field-mismatch: the backend's field is `interval`
    // (scanner_handler.go scheduleRequest / GetSchedule) — the mocks mirror the
    // real response, not a name the frontend invented.
    it('fetches schedule config', async () => {
      mockSuccess({ interval: 'hourly' });

      const data = await scannerService.getSchedule();
      expect(data.interval).toBe('hourly');
    });
  });

  describe('updateSchedule', () => {
    it('sends PUT with interval', async () => {
      mockSuccess({ interval: 'daily' });

      const data = await scannerService.updateSchedule('daily');
      expect(data.interval).toBe('daily');
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/scanner/schedule'),
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify({ interval: 'daily' }),
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
