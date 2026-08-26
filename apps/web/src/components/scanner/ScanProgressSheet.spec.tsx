import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ScanProgressSheet } from './ScanProgressSheet';
import type { ScanProgressState } from '../../hooks/useScanProgress';

const baseScanningState: ScanProgressState = {
  isScanning: true,
  percentDone: 62,
  currentFile: 'test.mkv',
  filesFound: 847,
  filesProcessed: 524,
  errorCount: 3,
  estimatedTime: '1 分 42 秒',
  isComplete: false,
  isCancelled: false,
  isMinimized: false,
  isDismissed: false,
  connectionStatus: 'sse',
};

const completeState: ScanProgressState = {
  ...baseScanningState,
  isScanning: false,
  percentDone: 100,
  isComplete: true,
  currentFile: '',
  estimatedTime: '',
};

const cancelledState: ScanProgressState = {
  ...baseScanningState,
  isScanning: false,
  isCancelled: true,
  isComplete: false,
  currentFile: '',
  estimatedTime: '',
};

describe('ScanProgressSheet', () => {
  const mockCancel = vi.fn();
  const mockDismiss = vi.fn();

  beforeEach(() => {
    vi.useFakeTimers();
    mockCancel.mockReset();
    mockDismiss.mockReset();
  });

  it('renders peek state by default', () => {
    render(
      <ScanProgressSheet state={baseScanningState} onCancel={mockCancel} onDismiss={mockDismiss} />
    );

    const sheet = screen.getByTestId('scan-progress-sheet');
    expect(sheet).toBeInTheDocument();
    expect(screen.getByText('掃描中 62%')).toBeInTheDocument();
    expect(screen.getByText('847 檔案')).toBeInTheDocument();
  });

  it('expands on click', () => {
    render(
      <ScanProgressSheet state={baseScanningState} onCancel={mockCancel} onDismiss={mockDismiss} />
    );

    fireEvent.click(screen.getByTestId('scan-progress-sheet'));

    expect(screen.getByText('媒體庫掃描中')).toBeInTheDocument();
    expect(screen.getByTestId('sheet-drag-handle')).toBeInTheDocument();
    expect(screen.getByTestId('sheet-progress-bar')).toBeInTheDocument();
  });

  it('shows stats in expanded state', () => {
    render(
      <ScanProgressSheet state={baseScanningState} onCancel={mockCancel} onDismiss={mockDismiss} />
    );

    // Expand
    fireEvent.click(screen.getByTestId('scan-progress-sheet'));

    expect(screen.getByText('847')).toBeInTheDocument();
    // 524 appears twice: 解析 and 比對 both show filesProcessed
    expect(screen.getAllByText('524')).toHaveLength(2);
    expect(screen.getByText('3')).toBeInTheDocument();
  });

  it('shows cancel button in expanded state', () => {
    render(
      <ScanProgressSheet state={baseScanningState} onCancel={mockCancel} onDismiss={mockDismiss} />
    );

    fireEvent.click(screen.getByTestId('scan-progress-sheet'));
    expect(screen.getByTestId('sheet-cancel-btn')).toBeInTheDocument();
  });

  it('shows cancel confirmation dialog', () => {
    render(
      <ScanProgressSheet state={baseScanningState} onCancel={mockCancel} onDismiss={mockDismiss} />
    );

    fireEvent.click(screen.getByTestId('scan-progress-sheet'));
    fireEvent.click(screen.getByTestId('sheet-cancel-btn'));

    expect(screen.getByTestId('sheet-cancel-confirm')).toBeInTheDocument();
    expect(screen.getByText('確定要取消掃描嗎？已處理的結果會保留。')).toBeInTheDocument();
  });

  it('calls onCancel when cancel is confirmed', () => {
    render(
      <ScanProgressSheet state={baseScanningState} onCancel={mockCancel} onDismiss={mockDismiss} />
    );

    fireEvent.click(screen.getByTestId('scan-progress-sheet'));
    fireEvent.click(screen.getByTestId('sheet-cancel-btn'));
    fireEvent.click(screen.getByTestId('sheet-cancel-confirm-btn'));

    expect(mockCancel).toHaveBeenCalledTimes(1);
  });

  it('renders completion state', () => {
    render(
      <ScanProgressSheet state={completeState} onCancel={mockCancel} onDismiss={mockDismiss} />
    );

    expect(screen.getByText('掃描完成')).toBeInTheDocument();
    expect(screen.getByText(/847 檔案/)).toBeInTheDocument();
  });

  it('renders cancelled state', () => {
    render(
      <ScanProgressSheet state={cancelledState} onCancel={mockCancel} onDismiss={mockDismiss} />
    );

    expect(screen.getByText('掃描已取消')).toBeInTheDocument();
  });

  it('calls onDismiss on dismiss button click', () => {
    render(
      <ScanProgressSheet state={completeState} onCancel={mockCancel} onDismiss={mockDismiss} />
    );

    fireEvent.click(screen.getByTestId('sheet-dismiss-btn'));
    expect(mockDismiss).toHaveBeenCalledTimes(1);
  });

  it('auto-dismisses after 10 seconds on completion', () => {
    render(
      <ScanProgressSheet state={completeState} onCancel={mockCancel} onDismiss={mockDismiss} />
    );

    expect(mockDismiss).not.toHaveBeenCalled();
    vi.advanceTimersByTime(10000);
    expect(mockDismiss).toHaveBeenCalledTimes(1);
  });

  it('shows ETA in expanded state', () => {
    render(
      <ScanProgressSheet state={baseScanningState} onCancel={mockCancel} onDismiss={mockDismiss} />
    );

    fireEvent.click(screen.getByTestId('scan-progress-sheet'));
    expect(screen.getByText(/1 分 42 秒/)).toBeInTheDocument();
  });
  /**
   * ⚖️ Alexyu 2026-08-27 ruling, on the mobile sheet specifically.
   *
   * He asked for the countdown bar AND explicitly refused pause-on-touch:
   *「當我按下掃描媒體庫之後，我不希望畫面一直停留在那個地方不動。我手機可能
   * 還要做其他事情，它不像桌機一樣，我開個網頁就可以去忙別的事了。」
   *
   * That makes the two surfaces deliberately DIFFERENT — the desktop card
   * pauses on hover, this one must not pause at all — which is exactly the kind
   * of asymmetry a future reader would "fix" by copying the desktop behaviour
   * across. These guard the ruling, not the implementation.
   */
  describe('auto-dismiss countdown (mobile ruling)', () => {
    it('[P1] shows a countdown bar so the dismissal is predictable', () => {
      render(
        <ScanProgressSheet state={completeState} onCancel={mockCancel} onDismiss={mockDismiss} />
      );
      const bar = screen.getByTestId('sheet-auto-dismiss-bar');
      expect(bar.className).toContain('animate-countdown');
      // Reduced motion must leave a STILL bar rather than an instantly-empty
      // one: `forwards` plus the global 1ms clamp would claim「時間到」with ten
      // seconds still on the real timer.
      expect(bar.className).toContain('motion-reduce:animate-none');
    });

    it('[P1] the bar and the real timer read the same constant', () => {
      render(
        <ScanProgressSheet state={completeState} onCancel={mockCancel} onDismiss={mockDismiss} />
      );
      const bar = screen.getByTestId('sheet-auto-dismiss-bar');
      expect(bar.style.animationDuration).toBe('10000ms');

      // …and that is genuinely when it goes.
      expect(mockDismiss).not.toHaveBeenCalled();
      vi.advanceTimersByTime(10000);
      expect(mockDismiss).toHaveBeenCalledTimes(1);
    });

    it('[P1] touching it does NOT pause the countdown — mobile must not be held hostage', () => {
      render(
        <ScanProgressSheet state={completeState} onCancel={mockCancel} onDismiss={mockDismiss} />
      );
      const sheet = screen.getByTestId('scan-progress-sheet');

      // Every gesture a reader might make while looking at it.
      fireEvent.touchStart(sheet, { touches: [{ clientY: 200 }] });
      fireEvent.pointerDown(sheet);
      fireEvent.mouseEnter(sheet);
      vi.advanceTimersByTime(10000);

      expect(mockDismiss).toHaveBeenCalledTimes(1);
      expect(screen.getByTestId('sheet-auto-dismiss-bar').style.animationPlayState).toBe('');
    });

    it('[P2] the cancelled toast counts down too — it is the same self-destruct', () => {
      render(
        <ScanProgressSheet state={cancelledState} onCancel={mockCancel} onDismiss={mockDismiss} />
      );
      expect(screen.getByTestId('sheet-auto-dismiss-bar')).toBeInTheDocument();
    });
  });
});
