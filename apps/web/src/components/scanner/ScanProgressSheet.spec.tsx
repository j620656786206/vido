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
    expect(screen.getByText('524')).toBeInTheDocument();
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
});
