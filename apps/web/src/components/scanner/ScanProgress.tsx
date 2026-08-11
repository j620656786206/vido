// Design ref: ux-design.pen Screen H2 Scan Progress Desktop (wyuhF)
/**
 * Responsive scan progress wrapper (Story 7.4, Task 5)
 * ≥768px → ScanProgressCard (floating bottom-right)
 * <768px → ScanProgressSheet (bottom sheet)
 * Visible on all pages during active scan.
 */

import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useScanProgress, subscribeScanTracking } from '../../hooks/useScanProgress';
import { useCancelScan } from '../../hooks/useScanner';
import { subtitleService } from '../../services/subtitleService';
import { generationBatchPreviewKey } from '../subtitle/GenerationBatchDialogV2';
import { ScanProgressCard } from './ScanProgressCard';
import { ScanProgressSheet } from './ScanProgressSheet';

const DESKTOP_BREAKPOINT = 768;

function useIsDesktop() {
  const [isDesktop, setIsDesktop] = useState(
    typeof window !== 'undefined' ? window.innerWidth >= DESKTOP_BREAKPOINT : true
  );

  useEffect(() => {
    const mq = window.matchMedia(`(min-width: ${DESKTOP_BREAKPOINT}px)`);
    const handler = (e: { matches: boolean }) => setIsDesktop(e.matches);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, []);

  return isDesktop;
}

export function ScanProgress() {
  const scanProgress = useScanProgress();
  const cancelScan = useCancelScan();
  const isDesktop = useIsDesktop();

  // Lazily open the scan-progress SSE only when a scan is actually triggered.
  // The trigger (掃描媒體庫) lives in a SEPARATE hook instance (ScannerSettings)
  // and cannot reach this shell-level instance directly, so it broadcasts a
  // module-level signal that this effect listens for. Staying lazy (no
  // connect-on-mount) preserves the app's networkidle-based e2e stability —
  // the EventSource is never opened until a scan begins.
  const { startTracking } = scanProgress;
  useEffect(() => subscribeScanTracking(startTracking), [startTracking]);

  // sub-4-3 F17: the 缺繁中字幕 count for the completion toast. The scan
  // payload carries no such count — fetched from the frozen preview endpoint
  // (movies-only; the episode undercount is tracked as
  // backlog-consent-toast-count-episodes). Hook runs unconditionally (before
  // the visibility early-return — rules-of-hooks) but only FETCHES once the
  // toast is actually showing; the card stays prop-driven for fixtures.
  const missingQuery = useQuery({
    queryKey: generationBatchPreviewKey,
    queryFn: () => subtitleService.previewGenerationBatch(),
    enabled: scanProgress.isVisible && scanProgress.isComplete,
    // CR sub-4-3 M6: the app-wide 5-min staleTime would show a PREVIOUS scan's
    // count (scan completion — the event that changes this number — never
    // invalidates the key; only a batch terminal does). Always refetch on the
    // toast appearing.
    staleTime: 0,
  });
  const missingSubtitleCount = scanProgress.isComplete ? missingQuery.data?.totalItems : undefined;

  if (!scanProgress.isVisible) return null;

  const handleCancel = () => {
    cancelScan.mutate();
  };

  // Completion toast: top-center per H3 spec; progress card: bottom-right per H2 spec
  const isComplete = scanProgress.isComplete || scanProgress.isCancelled;

  if (isDesktop) {
    return (
      <div
        className={
          isComplete ? 'fixed left-1/2 top-4 z-50 -translate-x-1/2' : 'fixed bottom-4 right-4 z-50'
        }
        data-testid="scan-progress-wrapper"
      >
        <ScanProgressCard
          state={scanProgress}
          onCancel={handleCancel}
          onToggleMinimize={scanProgress.toggleMinimize}
          onDismiss={scanProgress.dismiss}
          isCancelling={cancelScan.isPending}
          missingSubtitleCount={missingSubtitleCount}
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
        missingSubtitleCount={missingSubtitleCount}
      />
    </div>
  );
}
