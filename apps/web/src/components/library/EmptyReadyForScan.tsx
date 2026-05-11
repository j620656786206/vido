// Implements: Component/EmptyLibrary-ReadyForScan (mfKgm)
import { useEffect, useRef, useState } from 'react';
import { Link } from '@tanstack/react-router';
import { AlertCircle, ScanSearch } from 'lucide-react';
import { useTriggerScan } from '../../hooks/useScanner';
import type { ScannerApiError } from '../../services/scannerService';

type NotificationKind = 'success' | 'error';

export function EmptyReadyForScan() {
  const triggerScan = useTriggerScan();
  const [notification, setNotification] = useState<{
    type: NotificationKind;
    message: string;
  } | null>(null);
  const dismissTimerRef = useRef<ReturnType<typeof setTimeout>>();

  useEffect(() => {
    return () => {
      if (dismissTimerRef.current) clearTimeout(dismissTimerRef.current);
    };
  }, []);

  const showNotification = (type: NotificationKind, message: string) => {
    if (dismissTimerRef.current) clearTimeout(dismissTimerRef.current);
    setNotification({ type, message });
    dismissTimerRef.current = setTimeout(() => setNotification(null), 5000);
  };

  const handleScan = async () => {
    setNotification(null);
    try {
      await triggerScan.mutateAsync();
      showNotification('success', '掃描已啟動');
    } catch (err) {
      const apiErr = err as ScannerApiError;
      showNotification('error', apiErr?.message || '掃描觸發失敗');
    }
  };

  const isPending = triggerScan.isPending;

  return (
    <div
      className="flex flex-col items-center justify-center py-24 text-center"
      data-testid="empty-ready-for-scan"
    >
      <div className="mb-6 flex items-center gap-3 text-[var(--text-muted)]">
        <ScanSearch className="h-10 w-10" />
      </div>

      <h2 className="mb-3 text-xl font-semibold text-[var(--text-primary)]">
        準備好了，等待第一筆媒體
      </h2>
      <p className="mb-8 max-w-sm text-sm text-[var(--text-secondary)]">
        下載完成或掃描到檔案後會自動出現在這裡
      </p>

      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={handleScan}
          disabled={isPending}
          className="rounded-lg bg-[var(--accent-primary)] px-5 py-2.5 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-pressed)] disabled:cursor-not-allowed disabled:opacity-60"
          data-testid="empty-ready-for-scan-trigger-btn"
        >
          {isPending ? '掃描中…' : '立即掃描'}
        </button>
        <Link
          to="/downloads"
          className="rounded-lg border border-[var(--border-subtle)] px-5 py-2.5 text-sm font-medium text-[var(--text-secondary)] transition-colors hover:border-[var(--text-muted)] hover:text-white"
          data-testid="empty-ready-for-scan-downloads-btn"
        >
          前往下載中
        </Link>
      </div>

      {notification && (
        <div
          className={`mt-6 flex items-center gap-2 rounded-lg px-4 py-3 text-sm ${
            notification.type === 'success'
              ? 'bg-green-900/30 text-[var(--success)]'
              : 'bg-red-900/30 text-[var(--error)]'
          }`}
          data-testid="empty-ready-for-scan-notification"
          role="alert"
        >
          <AlertCircle className="h-4 w-4 shrink-0" />
          {notification.message}
        </div>
      )}
    </div>
  );
}
