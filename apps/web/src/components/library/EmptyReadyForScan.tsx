// Implements: Component/EmptyLibrary-ReadyForScan (mfKgm)
import { useEffect, useRef, useState } from 'react';
import { Link } from '@tanstack/react-router';
import { AlertCircle, ScanSearch } from 'lucide-react';
import { useTriggerScan } from '../../hooks/useScanner';
import { requestScanTracking } from '../../hooks/useScanProgress';
import type { ScannerApiError } from '../../services/scannerService';

type NotificationKind = 'success' | 'warning' | 'error';

// A server message written for people (zh-TW) contains CJK characters.
const HAS_CJK = /[\u3400-\u9fff]/;

export function EmptyReadyForScan() {
  const triggerScan = useTriggerScan();
  const [notification, setNotification] = useState<{
    type: NotificationKind;
    message: string;
  } | null>(null);
  const dismissTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  // True while waiting for the shell card's progress stream (see handleScan).
  const [connecting, setConnecting] = useState(false);

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
    // Connect the shell ScanProgress card FIRST, then start: without this the
    // first scan from an empty library showed no progress and no 完成 toast
    // (bugfix-scan-instant-completion-no-feedback). Never rejects; times out.
    setConnecting(true);
    try {
      await requestScanTracking();
    } finally {
      setConnecting(false);
    }
    try {
      await triggerScan.mutateAsync();
      showNotification('success', '掃描已啟動');
    } catch (err) {
      const apiErr = err as ScannerApiError;
      if (apiErr?.code === 'SCANNER_ALREADY_RUNNING') {
        showNotification('warning', '掃描已在進行中');
      } else {
        // disc-2026-09-scan-trigger-error-english: no developer English here.
        showNotification(
          'error',
          HAS_CJK.test(apiErr?.message ?? '') ? apiErr.message : '掃描沒有開始，請再試一次。'
        );
      }
    }
  };

  const isPending = connecting || triggerScan.isPending;

  return (
    <div
      className="flex flex-col items-center justify-center py-24 text-center"
      data-testid="empty-ready-for-scan"
    >
      <div className="mb-6 flex items-center gap-3 text-[var(--text-muted)]">
        <ScanSearch className="h-10 w-10" />
      </div>

      <h2 className="mb-3 text-lg sm:text-xl font-semibold text-[var(--text-primary)]">
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
          className="max-sm:inline-flex max-sm:min-h-11 max-sm:items-center rounded-lg bg-[var(--accent-primary)] px-5 py-2.5 text-sm font-medium text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-pressed)] disabled:cursor-not-allowed disabled:opacity-60"
          data-testid="empty-ready-for-scan-trigger-btn"
        >
          {isPending ? '掃描中…' : '立即掃描'}
        </button>
        <Link
          to="/downloads"
          className="max-sm:inline-flex max-sm:min-h-11 max-sm:items-center rounded-lg border border-[var(--border-subtle)] px-5 py-2.5 text-sm font-medium text-[var(--text-secondary)] transition-colors hover:border-[var(--text-muted)] hover:text-[var(--text-primary)]"
          data-testid="empty-ready-for-scan-downloads-btn"
        >
          前往下載中
        </Link>
      </div>

      {notification && (
        <div
          className={`mt-6 flex items-center gap-2 rounded-lg px-4 py-3 text-sm ${
            notification.type === 'success'
              ? 'bg-[var(--success-tint)] text-[var(--success-text)]'
              : notification.type === 'warning'
                ? 'bg-[var(--warning-tint)] text-[var(--warning-text)]'
                : 'bg-[var(--error-tint)] text-[var(--error-text)]'
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
