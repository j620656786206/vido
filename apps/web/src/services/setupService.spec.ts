import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { setupService } from './setupService';

const API_BASE = '/api/v1';

describe('setupService', () => {
  const mockFetch = vi.fn();

  beforeEach(() => {
    vi.stubGlobal('fetch', mockFetch);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('getStatus', () => {
    it('returns setup status when API succeeds', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ success: true, data: { needsSetup: true } }),
      });

      const result = await setupService.getStatus();
      expect(result).toEqual({ needsSetup: true });
      expect(mockFetch).toHaveBeenCalledWith(`${API_BASE}/setup/status`, expect.any(Object));
    });

    it('returns needsSetup false when setup completed', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ success: true, data: { needsSetup: false } }),
      });

      const result = await setupService.getStatus();
      expect(result).toEqual({ needsSetup: false });
    });

    it('throws on API error response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: () =>
          Promise.resolve({
            success: false,
            error: { code: 'INTERNAL_ERROR', message: 'DB down' },
          }),
      });

      await expect(setupService.getStatus()).rejects.toThrow('DB down');
    });

    it('throws on non-success response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () =>
          Promise.resolve({ success: false, error: { code: 'ERR', message: 'Something failed' } }),
      });

      await expect(setupService.getStatus()).rejects.toThrow('Something failed');
    });
  });

  describe('completeSetup', () => {
    it('sends POST with config and returns success message', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () =>
          Promise.resolve({ success: true, data: { message: 'Setup completed successfully' } }),
      });

      const config = { language: 'zh-TW', mediaFolderPath: '/media' };
      const result = await setupService.completeSetup(config);

      expect(result).toEqual({ message: 'Setup completed successfully' });
      expect(mockFetch).toHaveBeenCalledWith(
        `${API_BASE}/setup/complete`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ language: 'zh-TW', media_folder_path: '/media' }),
        })
      );
    });

    it('throws when setup already completed', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        json: () =>
          Promise.resolve({
            success: false,
            error: {
              code: 'SETUP_ALREADY_COMPLETED',
              message: 'Setup wizard has already been completed',
            },
          }),
      });

      await expect(
        setupService.completeSetup({ language: 'en', mediaFolderPath: '/m' })
      ).rejects.toThrow('Setup wizard has already been completed');
    });
  });

  describe('validateStep', () => {
    it('sends POST with step and data, returns valid=true', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ success: true, data: { valid: true } }),
      });

      const result = await setupService.validateStep('welcome', { language: 'zh-TW' });

      expect(result).toEqual({ valid: true });
      expect(mockFetch).toHaveBeenCalledWith(
        `${API_BASE}/setup/validate-step`,
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ step: 'welcome', data: { language: 'zh-TW' } }),
        })
      );
    });

    it('throws on validation failure', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        json: () =>
          Promise.resolve({
            success: false,
            error: { code: 'SETUP_VALIDATION_FAILED', message: 'language is required' },
          }),
      });

      await expect(setupService.validateStep('welcome', {})).rejects.toThrow(
        'language is required'
      );
    });
  });
});
