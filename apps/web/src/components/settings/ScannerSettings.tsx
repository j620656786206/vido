/**
 * Scanner Settings component (Story 7.3)
 * Displays media folder paths, scan schedule, last scan info, and scan trigger button.
 */

import { useState, useRef, useEffect } from 'react';
import { ScanLine, Loader, AlertCircle } from 'lucide-react';
import { MediaLibraryManager } from './MediaLibraryManager';
import { cn } from '../../lib/utils';
import {
  useScanStatus,
  useTriggerScan,
  useScanSchedule,
  useUpdateScanSchedule,
} from '../../hooks/useScanner';
import type { ScannerApiError } from '../../services/scannerService';
import type { ScheduleFrequency } from '../../services/scannerService';

const SCHEDULE_OPTIONS: { value: ScheduleFrequency; label: string }[] = [
  { value: 'hourly', label: '每小時' },
  { value: 'daily', label: '每天' },
  { value: 'manual', label: '僅手動' },
];

function formatLastScan(lastAt: string, files: number, duration: string): string {
  if (!lastAt) return '尚未執行過掃描';
  const date = new Date(lastAt);
  const formatted = date.toLocaleString('zh-TW', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
  return `${formatted} · ${files.toLocaleString()} 檔案 · 耗時 ${duration}`;
}

export function ScannerSettings() {
  const { data: status, isLoading: statusLoading } = useScanStatus();
  const { data: schedule, isLoading: scheduleLoading } = useScanSchedule();
  const triggerScan = useTriggerScan();
  const updateSchedule = useUpdateScanSchedule();
  const [notification, setNotification] = useState<{
    type: 'success' | 'warning' | 'error';
    message: string;
  } | null>(null);
  const dismissTimerRef = useRef<ReturnType<typeof setTimeout>>();

  // Cleanup timer on unmount
  useEffect(() => {
    return () => {
      if (dismissTimerRef.current) clearTimeout(dismissTimerRef.current);
    };
  }, []);

  const showNotification = (type: 'success' | 'warning' | 'error', message: string) => {
    if (dismissTimerRef.current) clearTimeout(dismissTimerRef.current);
    setNotification({ type, message });
    dismissTimerRef.current = setTimeout(() => setNotification(null), 5000);
  };

  const isScanning = status?.isScanning ?? false;

  const handleScan = async () => {
    setNotification(null);
    try {
      await triggerScan.mutateAsync();
    } catch (err) {
      const apiErr = err as ScannerApiError;
      if (apiErr.code === 'SCANNER_ALREADY_RUNNING') {
        showNotification('warning', '掃描已在進行中');
      } else {
        showNotification('error', apiErr.message || '掃描觸發失敗');
      }
    }
  };

  const handleScheduleChange = async (frequency: ScheduleFrequency) => {
    try {
      await updateSchedule.mutateAsync(frequency);
    } catch (err) {
      const apiErr = err as ScannerApiError;
      showNotification('error', apiErr.message || '排程更新失敗');
    }
  };

  if (statusLoading || scheduleLoading) {
    return (
      <div
        className="flex items-center gap-2 text-[var(--text-secondary)]"
        data-testid="scanner-loading"
      >
        <Loader className="h-4 w-4 animate-spin" />
        <span>載入中...</span>
      </div>
    );
  }

  return (
    <div className="space-y-6" data-testid="scanner-settings">
      {/* Header */}
      <div>
        <h2 className="text-xl font-semibold text-[var(--text-primary)]">媒體庫掃描</h2>
        <p className="mt-1 text-sm text-[var(--text-secondary)]">
          設定掃描資料夾、排程，以及手動觸發媒體庫掃描
        </p>
      </div>

      {/* Notification */}
      {notification && (
        <div
          className={cn(
            'flex items-center gap-2 rounded-lg px-4 py-3 text-sm',
            notification.type === 'success' && 'bg-green-900/30 text-[var(--success)]',
            notification.type === 'warning' && 'bg-yellow-900/30 text-[var(--warning)]',
            notification.type === 'error' && 'bg-red-900/30 text-[var(--error)]'
          )}
          data-testid="scanner-notification"
          role="alert"
        >
          <AlertCircle className="h-4 w-4 shrink-0" />
          {notification.message}
        </div>
      )}

      {/* Settings card */}
      <div className="space-y-6 rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-6">
        {/* Media Libraries (Story 7b-4) */}
        <MediaLibraryManager />

        <hr className="border-[var(--border-subtle)]" />

        {/* Schedule selector */}
        <div className="space-y-3">
          <label
            htmlFor="scan-schedule"
            className="text-sm font-medium text-[var(--text-secondary)]"
          >
            掃描排程
          </label>
          <select
            id="scan-schedule"
            value={schedule?.frequency ?? 'manual'}
            onChange={(e) => handleScheduleChange(e.target.value as ScheduleFrequency)}
            disabled={updateSchedule.isPending}
            className="w-48 rounded-md border border-[var(--border-subtle)] bg-[var(--bg-primary)] px-3 py-2.5 text-sm text-[var(--text-primary)] focus:border-blue-400 focus:outline-none focus:ring-1 focus:ring-blue-400"
            data-testid="schedule-select"
          >
            {SCHEDULE_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>

        <hr className="border-[var(--border-subtle)]" />

        {/* Last scan info */}
        <div className="space-y-2">
          <label className="text-sm font-medium text-[var(--text-secondary)]">上次掃描</label>
          <p
            className="font-mono text-sm text-[var(--text-secondary)]"
            data-testid="last-scan-info"
          >
            {status
              ? formatLastScan(status.lastScanAt, status.lastScanFiles, status.lastScanDuration)
              : '載入中...'}
          </p>
        </div>

        <hr className="border-[var(--border-subtle)]" />

        {/* Scan button */}
        <button
          type="button"
          onClick={handleScan}
          disabled={isScanning || triggerScan.isPending}
          className={cn(
            'flex w-full items-center justify-center gap-2 rounded-lg px-4 py-3.5 text-sm font-semibold transition-colors',
            isScanning || triggerScan.isPending
              ? 'cursor-not-allowed bg-[var(--accent-primary)]/50 text-blue-200'
              : 'bg-[var(--accent-primary)] text-white hover:bg-[var(--accent-primary)]'
          )}
          data-testid="scan-trigger-button"
        >
          {isScanning || triggerScan.isPending ? (
            <>
              <Loader className="h-4 w-4 animate-spin" />
              掃描進行中...
            </>
          ) : (
            <>
              <ScanLine className="h-4 w-4" />
              掃描媒體庫
            </>
          )}
        </button>
      </div>
    </div>
  );
}
