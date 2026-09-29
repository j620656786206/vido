import { describe, it, expect, vi } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { createElement } from 'react';
import {
  useScanStatus,
  useScanSchedule,
  useTriggerScan,
  useCancelScan,
  useUpdateScanSchedule,
  scannerKeys,
} from './useScanner';

vi.mock('../services/scannerService', () => ({
  scannerService: {
    getScanStatus: vi.fn().mockResolvedValue({
      isActive: false,
      filesFound: 0,
      filesCreated: 0,
      filesUpdated: 0,
      filesSkipped: 0,
      filesRemoved: 0,
      errorCount: 0,
      currentFile: '',
      percentDone: 0,
      lastScan: { completedAt: '2026-03-22T14:30:00Z', filesFound: 1247, durationMs: 192000 },
    }),
    getSchedule: vi.fn().mockResolvedValue({ interval: 'hourly' }),
    triggerScan: vi
      .fn()
      .mockResolvedValue({ filesFound: 10, filesNew: 5, errors: 0, duration: '10s' }),
    cancelScan: vi.fn().mockResolvedValue(undefined),
    updateSchedule: vi.fn().mockResolvedValue({ interval: 'daily' }),
  },
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
}

describe('useScanner hooks', () => {
  describe('scannerKeys', () => {
    it('generates correct query keys', () => {
      expect(scannerKeys.all).toEqual(['scanner']);
      expect(scannerKeys.status()).toEqual(['scanner', 'status']);
      expect(scannerKeys.schedule()).toEqual(['scanner', 'schedule']);
    });
  });

  describe('useScanStatus', () => {
    it('fetches scan status successfully', async () => {
      const { result } = renderHook(() => useScanStatus(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data?.lastScan?.filesFound).toBe(1247);
      expect(result.current.data?.isActive).toBe(false);
    });
  });

  describe('useScanSchedule', () => {
    it('fetches schedule config', async () => {
      const { result } = renderHook(() => useScanSchedule(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data?.interval).toBe('hourly');
    });
  });

  describe('useTriggerScan', () => {
    it('calls scannerService.triggerScan on mutate', async () => {
      const { scannerService } = await import('../services/scannerService');
      const { result } = renderHook(() => useTriggerScan(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        await result.current.mutateAsync();
      });

      expect(scannerService.triggerScan).toHaveBeenCalled();
    });
  });

  describe('useCancelScan', () => {
    it('calls scannerService.cancelScan on mutate', async () => {
      const { scannerService } = await import('../services/scannerService');
      const { result } = renderHook(() => useCancelScan(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        await result.current.mutateAsync();
      });

      expect(scannerService.cancelScan).toHaveBeenCalled();
    });
  });

  describe('useUpdateScanSchedule', () => {
    it('calls scannerService.updateSchedule with the interval', async () => {
      const { scannerService } = await import('../services/scannerService');
      const { result } = renderHook(() => useUpdateScanSchedule(), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        await result.current.mutateAsync('daily');
      });

      expect(scannerService.updateSchedule).toHaveBeenCalledWith('daily');
    });
  });
});
