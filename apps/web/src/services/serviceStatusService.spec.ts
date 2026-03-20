import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { serviceStatusService } from './serviceStatusService';

const mockFetch = vi.fn();
global.fetch = mockFetch;

beforeEach(() => {
  mockFetch.mockReset();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe('serviceStatusService', () => {
  describe('getAllStatuses', () => {
    it('[P1] returns service statuses on successful response', async () => {
      // GIVEN: API returns successful response with services
      const mockServices = {
        services: [
          {
            name: 'tmdb',
            displayName: 'TMDb API',
            status: 'connected',
            message: '已連線',
            lastSuccessAt: '2026-02-10T14:30:00Z',
            lastCheckAt: '2026-02-10T14:30:00Z',
            responseTimeMs: 45,
          },
        ],
      };
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true, data: mockServices }),
      });

      // WHEN: Fetching all statuses
      const result = await serviceStatusService.getAllStatuses();

      // THEN: Returns parsed service data
      expect(result).toEqual(mockServices);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/settings/services'),
        expect.objectContaining({
          headers: { 'Content-Type': 'application/json' },
        })
      );
    });

    it('[P1] throws error on API failure response', async () => {
      // GIVEN: API returns error response
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        json: () =>
          Promise.resolve({
            success: false,
            error: { code: 'INTERNAL_ERROR', message: 'Server error' },
          }),
      });

      // WHEN/THEN: Should throw with error message
      await expect(serviceStatusService.getAllStatuses()).rejects.toThrow('Server error');
    });

    it('[P2] throws generic error when no error message in response', async () => {
      // GIVEN: API returns failure without error details
      mockFetch.mockResolvedValue({
        ok: false,
        status: 503,
        json: () => Promise.resolve({ success: false }),
      });

      // WHEN/THEN: Should throw with status code
      await expect(serviceStatusService.getAllStatuses()).rejects.toThrow(
        'API request failed: 503'
      );
    });

    it('[P1] throws error when success is true but data field is missing', async () => {
      // GIVEN: API returns success without data
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true }),
      });

      // WHEN/THEN: Should throw for missing data
      await expect(serviceStatusService.getAllStatuses()).rejects.toThrow(
        'API response missing data field'
      );
    });
  });

  describe('testService', () => {
    it('[P1] sends POST request with encoded service name', async () => {
      // GIVEN: API returns successful test result
      const mockResult = {
        name: 'qbittorrent',
        displayName: 'qBittorrent',
        status: 'connected',
        message: '已連線，版本 4.6.1',
        lastSuccessAt: '2026-02-10T14:30:00Z',
        lastCheckAt: '2026-02-10T14:30:00Z',
        responseTimeMs: 30,
      };
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true, data: mockResult }),
      });

      // WHEN: Testing a service connection
      const result = await serviceStatusService.testService('qbittorrent');

      // THEN: Returns test result and sends POST
      expect(result).toEqual(mockResult);
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/settings/services/qbittorrent/test'),
        expect.objectContaining({ method: 'POST' })
      );
    });

    it('[P1] throws error when service test fails', async () => {
      // GIVEN: API returns SERVICE_NOT_FOUND error
      mockFetch.mockResolvedValue({
        ok: false,
        status: 404,
        json: () =>
          Promise.resolve({
            success: false,
            error: { code: 'SERVICE_NOT_FOUND', message: 'Unknown service: invalid' },
          }),
      });

      // WHEN/THEN: Should throw with error message
      await expect(serviceStatusService.testService('invalid')).rejects.toThrow(
        'Unknown service: invalid'
      );
    });
  });
});
