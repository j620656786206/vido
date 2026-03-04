import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { healthService } from './healthService';

const mockServicesHealthResponse = {
  success: true,
  data: {
    degradationLevel: 'normal' as const,
    services: {
      tmdb: {
        name: 'tmdb',
        displayName: 'TMDb API',
        status: 'healthy' as const,
        lastCheck: '2026-03-04T12:00:00Z',
        lastSuccess: '2026-03-04T12:00:00Z',
        errorCount: 0,
      },
      douban: {
        name: 'douban',
        displayName: 'Douban Scraper',
        status: 'healthy' as const,
        lastCheck: '2026-03-04T12:00:00Z',
        lastSuccess: '2026-03-04T12:00:00Z',
        errorCount: 0,
      },
      wikipedia: {
        name: 'wikipedia',
        displayName: 'Wikipedia API',
        status: 'healthy' as const,
        lastCheck: '2026-03-04T12:00:00Z',
        lastSuccess: '2026-03-04T12:00:00Z',
        errorCount: 0,
      },
      ai: {
        name: 'ai',
        displayName: 'AI Parser',
        status: 'healthy' as const,
        lastCheck: '2026-03-04T12:00:00Z',
        lastSuccess: '2026-03-04T12:00:00Z',
        errorCount: 0,
      },
      qbittorrent: {
        name: 'qbittorrent',
        displayName: 'qBittorrent',
        status: 'healthy' as const,
        lastCheck: '2026-03-04T12:00:00Z',
        lastSuccess: '2026-03-04T12:00:00Z',
        errorCount: 0,
      },
    },
    message: '',
  },
};

const mockHistoryResponse = {
  success: true,
  data: [
    {
      id: 'evt-1',
      service: 'qbittorrent',
      eventType: 'disconnected' as const,
      status: 'down',
      message: 'connection refused',
      createdAt: '2026-03-04T11:55:00Z',
    },
    {
      id: 'evt-2',
      service: 'qbittorrent',
      eventType: 'connected' as const,
      status: 'healthy',
      createdAt: '2026-03-04T11:00:00Z',
    },
  ],
};

describe('healthService', () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  describe('getServicesHealth', () => {
    it('[P2] should return services health on success', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockServicesHealthResponse),
      });

      const result = await healthService.getServicesHealth();

      expect(result.degradationLevel).toBe('normal');
      expect(result.services.qbittorrent.status).toBe('healthy');
      expect(result.services.qbittorrent.displayName).toBe('qBittorrent');
    });

    it('[P2] should throw on HTTP error response', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: () =>
          Promise.resolve({
            error: { code: 'INTERNAL', message: 'Server error' },
          }),
      });

      await expect(healthService.getServicesHealth()).rejects.toThrow('Server error');
    });

    it('[P2] should throw on HTTP error with unparseable body', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 502,
        json: () => Promise.reject(new Error('invalid json')),
      });

      await expect(healthService.getServicesHealth()).rejects.toThrow('API request failed: 502');
    });

    it('[P2] should throw on API error response (success: false)', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            success: false,
            error: { code: 'SERVICE_UNAVAILABLE', message: '服務不可用' },
          }),
      });

      await expect(healthService.getServicesHealth()).rejects.toThrow('服務不可用');
    });
  });

  describe('getConnectionHistory', () => {
    it('[P2] should return connection history on success', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockHistoryResponse),
      });

      const result = await healthService.getConnectionHistory('qbittorrent');

      expect(result).toHaveLength(2);
      expect(result[0].eventType).toBe('disconnected');
      expect(result[1].eventType).toBe('connected');
    });

    it('[P2] should use default limit of 20', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockHistoryResponse),
      });

      await healthService.getConnectionHistory('qbittorrent');

      expect(globalThis.fetch).toHaveBeenCalledWith(expect.stringContaining('limit=20'));
    });

    it('[P2] should pass custom limit parameter', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockHistoryResponse),
      });

      await healthService.getConnectionHistory('qbittorrent', 5);

      expect(globalThis.fetch).toHaveBeenCalledWith(expect.stringContaining('limit=5'));
    });

    it('[P2] should encode service name in URL', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true, data: [] }),
      });

      await healthService.getConnectionHistory('qbittorrent');

      expect(globalThis.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/health/services/qbittorrent/history')
      );
    });

    it('[P2] should throw on HTTP error response', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 503,
        json: () =>
          Promise.resolve({
            error: { message: 'History not available' },
          }),
      });

      await expect(healthService.getConnectionHistory('qbittorrent')).rejects.toThrow(
        'History not available'
      );
    });
  });
});
