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
    }),
    getSchedule: vi.fn().mockResolvedValue({ frequency: 'hourly' }),
    triggerScan: vi
      .fn()
      .mockResolvedValue({ files_found: 10, files_new: 5, errors: 0, duration: '10s' }),
    cancelScan: vi.fn().mockResolvedValue(undefined),
    updateSchedule: vi.fn().mockResolvedValue({ frequency: 'daily' }),
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
      expect(result.current.data?.last_scan_files).toBe(1247);
      expect(result.current.data?.is_scanning).toBe(false);
    });
  });

  describe('useScanSchedule', () => {
    it('fetches schedule config', async () => {
      const { result } = renderHook(() => useScanSchedule(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data?.frequency).toBe('hourly');
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
    it('calls scannerService.updateSchedule with frequency', async () => {
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
