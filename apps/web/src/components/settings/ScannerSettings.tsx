// Design ref: ux-design.pen Screen E1-D (KvZSc) · E1-M (uABWl)
// 媒體庫區塊由 MediaLibraryManager／LibraryCard 畫；E1 已在 dsr-5 依多媒體庫改版重畫（drift-e1 關閉）。
// 掃描排程（每小時／每天／僅手動）前後端都有實作——原本這裡寫「前端未實作」是錯的。
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
import { requestScanTracking } from '../../hooks/useScanProgress';
import type { ScannerApiError } from '../../services/scannerService';
import type { LastScan, ScheduleInterval } from '../../services/scannerService';

const SCHEDULE_OPTIONS: { value: ScheduleInterval; label: string }[] = [
  { value: 'hourly', label: '每小時' },
  { value: 'daily', label: '每天' },
  { value: 'manual', label: '僅手動' },
];

function formatScanDuration(ms: number): string {
  if (ms < 1000) return '不到 1 秒';
  const totalSeconds = Math.floor(ms / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes === 0) return `${seconds} 秒`;
  if (minutes >= 60) {
    const rest = minutes % 60;
    return rest === 0
      ? `${Math.floor(minutes / 60)} 小時`
      : `${Math.floor(minutes / 60)} 小時 ${rest} 分`;
  }
  return seconds === 0 ? `${minutes} 分` : `${minutes} 分 ${seconds} 秒`;
}

/** E1-D: 「2026-03-22 14:30 · 1,247 檔案 · 耗時 3 分 12 秒」 (local time). */
function formatLastScan(lastScan: LastScan | null | undefined): string {
  if (!lastScan) return '尚未執行過掃描';
  const d = new Date(lastScan.completedAt);
  const pad = (n: number) => String(n).padStart(2, '0');
  const when = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
  return `${when} · ${lastScan.filesFound.toLocaleString('zh-TW')} 檔案 · 耗時 ${formatScanDuration(lastScan.durationMs)}`;
}

// A server message written for people (zh-TW) contains CJK characters.
const HAS_CJK = /[\u3400-\u9fff]/;

export function ScannerSettings() {
  const { data: status, isLoading: statusLoading } = useScanStatus();
  const { data: schedule, isLoading: scheduleLoading } = useScanSchedule();
  const triggerScan = useTriggerScan();
  const updateSchedule = useUpdateScanSchedule();
  // True while waiting for the progress stream before the scan request.
  const [connecting, setConnecting] = useState(false);
  const [notification, setNotification] = useState<{
    type: 'success' | 'warning' | 'error';
    message: string;
  } | null>(null);
  const dismissTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

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

  const isScanning = status?.isActive ?? false;

  const handleScan = async () => {
    setNotification(null);
    // Open the shell card's progress stream and wait until it is connected
    // BEFORE starting: a small library scans in milliseconds, and events sent
    // before the stream registers are lost — no card, no 完成 toast
    // (bugfix-scan-instant-completion-no-feedback). Never rejects; times out.
    setConnecting(true);
    try {
      await requestScanTracking();
    } finally {
      setConnecting(false);
    }
    try {
      await triggerScan.mutateAsync();
    } catch (err) {
      const apiErr = err as ScannerApiError;
      if (apiErr.code === 'SCANNER_ALREADY_RUNNING') {
        showNotification('warning', '掃描已在進行中');
      } else {
        // Server/network messages are mostly developer English
        // (disc-2026-09-scan-trigger-error-english): pass a zh-TW one through,
        // otherwise say it plainly.
        showNotification(
          'error',
          HAS_CJK.test(apiErr?.message ?? '') ? apiErr.message : '掃描沒有開始，請再試一次。'
        );
      }
    }
  };

  const handleScheduleChange = async (interval: ScheduleInterval) => {
    try {
      await updateSchedule.mutateAsync(interval);
    } catch {
      // The API's message is English for developers; the user gets Chinese.
      showNotification('error', '排程沒有存成功，請再試一次。');
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
      {/* Notification */}
      {notification && (
        <div
          className={cn(
            'flex items-center gap-2 rounded-lg px-4 py-3 text-sm',
            // 固定詞彙: completion notices are neutral; green stays with
            // states that are LIVE right now.
            notification.type === 'success' && 'bg-[var(--bg-tertiary)] text-[var(--text-primary)]',
            notification.type === 'warning' &&
              'bg-[var(--warning-tint)] text-[var(--warning-text)]',
            notification.type === 'error' && 'bg-[var(--error-tint)] text-[var(--error-text)]'
          )}
          data-testid="scanner-notification"
          role="alert"
        >
          <AlertCircle className="h-4 w-4 shrink-0" />
          {notification.message}
        </div>
      )}

      {/* Settings card */}
      <div className="space-y-4 rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4 sm:space-y-6 sm:p-6">
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
            value={schedule?.interval ?? 'manual'}
            onChange={(e) => handleScheduleChange(e.target.value as ScheduleInterval)}
            disabled={updateSchedule.isPending}
            className="block w-full rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-primary)] px-3 py-2.5 text-sm text-[var(--text-primary)] focus:border-[var(--focus-ring)] focus:outline-none focus:ring-1 focus:ring-[var(--focus-ring)] sm:w-50"
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
          <span className="text-sm font-medium text-[var(--text-secondary)]">上次掃描</span>
          <p
            className="font-mono text-xs text-[var(--text-muted)] sm:text-sm"
            data-testid="last-scan-info"
          >
            {status ? formatLastScan(status.lastScan) : '載入中...'}
          </p>
        </div>

        <hr className="border-[var(--border-subtle)]" />

        {/* Scan button */}
        <button
          type="button"
          onClick={handleScan}
          disabled={isScanning || connecting || triggerScan.isPending}
          className={cn(
            'flex w-full items-center justify-center gap-2 rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-4 py-3.5 text-sm font-semibold text-[var(--text-on-accent)] transition-colors sm:text-base',
            // Was a gold label on a half-transparent gold fill while scanning,
            // and a hover that set the same colour it already had.
            isScanning || connecting || triggerScan.isPending
              ? 'cursor-not-allowed opacity-50'
              : 'hover:bg-[var(--accent-hover)] active:bg-[var(--accent-pressed)]'
          )}
          data-testid="scan-trigger-button"
        >
          {isScanning || connecting || triggerScan.isPending ? (
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
