/**
 * Responsive scan progress wrapper (Story 7.4, Task 5)
 * ≥768px → ScanProgressCard (floating bottom-right)
 * <768px → ScanProgressSheet (bottom sheet)
 * Visible on all pages during active scan.
 */

import { useEffect, useState } from 'react';
import { useScanProgress } from '../../hooks/useScanProgress';
import { useCancelScan } from '../../hooks/useScanner';
import { ScanProgressCard } from './ScanProgressCard';
import { ScanProgressSheet } from './ScanProgressSheet';

const DESKTOP_BREAKPOINT = 768;

function useIsDesktop() {
  const [isDesktop, setIsDesktop] = useState(
    typeof window !== 'undefined' ? window.innerWidth >= DESKTOP_BREAKPOINT : true
  );

  useEffect(() => {
    const mq = window.matchMedia(`(min-width: ${DESKTOP_BREAKPOINT}px)`);
    const handler = (e: MediaQueryListEvent) => setIsDesktop(e.matches);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, []);

  return isDesktop;
}

export function ScanProgress() {
  const scanProgress = useScanProgress();
  const cancelScan = useCancelScan();
  const isDesktop = useIsDesktop();

  if (!scanProgress.isVisible) return null;

  const handleCancel = () => {
    cancelScan.mutate();
  };

  if (isDesktop) {
    return (
      <div className="fixed bottom-4 right-4 z-50" data-testid="scan-progress-wrapper">
        <ScanProgressCard
          state={scanProgress}
          onCancel={handleCancel}
          onToggleMinimize={scanProgress.toggleMinimize}
          onDismiss={scanProgress.dismiss}
          isCancelling={cancelScan.isPending}
        />
      </div>
    );
  }

  return (
    <div className="fixed inset-x-0 bottom-0 z-50" data-testid="scan-progress-wrapper">
      <ScanProgressSheet
        state={scanProgress}
        onCancel={handleCancel}
        onDismiss={scanProgress.dismiss}
        isCancelling={cancelScan.isPending}
      />
    </div>
  );
}
