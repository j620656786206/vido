import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ScanProgressCard } from './ScanProgressCard';
import type { ScanProgressState } from '../../hooks/useScanProgress';

const mockNavigate = vi.fn();
vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => mockNavigate,
}));

const baseScanningState: ScanProgressState = {
  isScanning: true,
  percentDone: 62,
  currentFile: '[Leopard-Raws] Demon Slayer S03E01.mkv',
  filesFound: 847,
  filesProcessed: 524,
  filesUnmatched: 323,
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

const minimizedState: ScanProgressState = {
  ...baseScanningState,
  isMinimized: true,
};

describe('ScanProgressCard', () => {
  const mockCancel = vi.fn();
  const mockToggleMinimize = vi.fn();
  const mockDismiss = vi.fn();

  beforeEach(() => {
    vi.useFakeTimers();
    mockCancel.mockReset();
    mockToggleMinimize.mockReset();
    mockDismiss.mockReset();
    mockNavigate.mockReset();
  });

  it('renders scanning state with progress and stats', () => {
    render(
      <ScanProgressCard
        state={baseScanningState}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    expect(screen.getByText('媒體庫掃描中')).toBeInTheDocument();
    expect(screen.getByText('62%')).toBeInTheDocument();
    expect(screen.getByText('找到 847')).toBeInTheDocument();
    expect(screen.getByText('解析 524')).toBeInTheDocument();
    expect(screen.getByTestId('scan-error-stat')).toHaveTextContent('錯誤 3');
    // dsr-5: no 比對 counter. The scanner never reports a matched count; the
    // card used to print filesProcessed a second time under that label.
    expect(screen.queryByText(/比對/)).not.toBeInTheDocument();
    expect(screen.getByTestId('scan-current-file')).toHaveTextContent('Demon Slayer');
    expect(screen.getByTestId('scan-eta')).toHaveTextContent('1 分 42 秒');
  });

  it('renders progress bar with correct width', () => {
    render(
      <ScanProgressCard
        state={baseScanningState}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    const bar = screen.getByTestId('scan-progress-bar');
    expect(bar.style.width).toBe('62%');
  });

  it('renders minimized pill when minimized', () => {
    render(
      <ScanProgressCard
        state={minimizedState}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    expect(screen.getByTestId('scan-progress-pill')).toBeInTheDocument();
    expect(screen.getByText('掃描中 62%')).toBeInTheDocument();
  });

  it('calls onToggleMinimize when minimize button clicked', () => {
    render(
      <ScanProgressCard
        state={baseScanningState}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    fireEvent.click(screen.getByTestId('scan-minimize-btn'));
    expect(mockToggleMinimize).toHaveBeenCalledTimes(1);
  });

  it('calls onToggleMinimize when pill clicked', () => {
    render(
      <ScanProgressCard
        state={minimizedState}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    fireEvent.click(screen.getByTestId('scan-progress-pill'));
    expect(mockToggleMinimize).toHaveBeenCalledTimes(1);
  });

  it('shows cancel confirmation on cancel button click', () => {
    render(
      <ScanProgressCard
        state={baseScanningState}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    fireEvent.click(screen.getByTestId('scan-cancel-btn'));
    expect(screen.getByTestId('cancel-confirm-dialog')).toBeInTheDocument();
    expect(screen.getByText('確定要取消掃描嗎？已處理的結果會保留。')).toBeInTheDocument();
  });

  it('calls onCancel when cancel is confirmed', () => {
    render(
      <ScanProgressCard
        state={baseScanningState}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    fireEvent.click(screen.getByTestId('scan-cancel-btn'));
    fireEvent.click(screen.getByTestId('cancel-confirm-btn'));
    expect(mockCancel).toHaveBeenCalledTimes(1);
  });

  it('hides cancel confirmation when continue is clicked', () => {
    render(
      <ScanProgressCard
        state={baseScanningState}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    fireEvent.click(screen.getByTestId('scan-cancel-btn'));
    expect(screen.getByTestId('cancel-confirm-dialog')).toBeInTheDocument();

    fireEvent.click(screen.getByTestId('cancel-continue-btn'));
    expect(screen.queryByTestId('cancel-confirm-dialog')).not.toBeInTheDocument();
  });

  it('shows X button triggers cancel confirmation', () => {
    render(
      <ScanProgressCard
        state={baseScanningState}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    fireEvent.click(screen.getByTestId('scan-close-btn'));
    expect(screen.getByTestId('cancel-confirm-dialog')).toBeInTheDocument();
  });

  it('renders completion state with summary', () => {
    render(
      <ScanProgressCard
        state={completeState}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    expect(screen.getByText('掃描完成')).toBeInTheDocument();
    // 新增／更新 are absent from this state (not reported) — omitted, not printed as 0.
    expect(screen.getByTestId('scan-summary')).toHaveTextContent(
      '找到 847 檔案 · 無法匯入 323 · 錯誤 3'
    );
    expect(screen.getByTestId('view-scan-problems-link')).toHaveTextContent('查看無法匯入與錯誤');
  });

  it('renders cancelled state', () => {
    render(
      <ScanProgressCard
        state={cancelledState}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    expect(screen.getByText('掃描已取消')).toBeInTheDocument();
  });

  it('prints only what the scanner reported — 新增／更新 from scan_complete, no 比對成功 (dsr-5)', () => {
    render(
      <ScanProgressCard
        state={{
          ...completeState,
          filesFound: 1247,
          filesCreated: 30,
          filesUpdated: 1168,
          filesUnmatched: 42,
          errorCount: 7,
        }}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    expect(screen.getByTestId('scan-summary')).toHaveTextContent(
      '找到 1,247 檔案 · 新增 30 · 更新 1,168 · 無法匯入 42 · 錯誤 7'
    );
    // 比對成功 was found − unmatched (errors counted as successes); 未比對 counted
    // files that were never imported, so no library filter could show them.
    expect(screen.queryByText(/比對成功|未比對/)).not.toBeInTheDocument();
  });

  it('sends 查看無法匯入與錯誤 to the system log — where those files are recorded', () => {
    render(
      <ScanProgressCard
        state={completeState}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    fireEvent.click(screen.getByTestId('view-scan-problems-link'));
    expect(mockDismiss).toHaveBeenCalled();
    expect(mockNavigate).toHaveBeenCalledWith({ to: '/settings/logs' });
  });

  it('keeps the link when files could not be imported even with zero errors', () => {
    render(
      <ScanProgressCard
        state={{ ...completeState, errorCount: 0, filesUnmatched: 5 }}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    expect(screen.getByTestId('view-scan-problems-link')).toBeInTheDocument();
  });

  it('calls onDismiss when dismiss button clicked on complete card', () => {
    render(
      <ScanProgressCard
        state={completeState}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    fireEvent.click(screen.getByTestId('scan-dismiss-btn'));
    expect(mockDismiss).toHaveBeenCalledTimes(1);
  });

  it('auto-dismisses after 10 seconds on completion', () => {
    render(
      <ScanProgressCard
        state={completeState}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    expect(mockDismiss).not.toHaveBeenCalled();
    vi.advanceTimersByTime(10000);
    expect(mockDismiss).toHaveBeenCalledTimes(1);
  });

  it('has no problems link after a clean scan', () => {
    render(
      <ScanProgressCard
        state={{ ...completeState, errorCount: 0, filesUnmatched: 0 }}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    expect(screen.queryByTestId('view-scan-problems-link')).not.toBeInTheDocument();
  });

  it('shows auto-dismiss progress bar on completion', () => {
    render(
      <ScanProgressCard
        state={completeState}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    expect(screen.getByTestId('auto-dismiss-bar')).toBeInTheDocument();
  });

  it('pauses auto-dismiss when user hovers on complete card', () => {
    render(
      <ScanProgressCard
        state={completeState}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    // Hover to pause auto-dismiss
    fireEvent.mouseEnter(screen.getByTestId('scan-progress-card'));

    // Advance past auto-dismiss time
    vi.advanceTimersByTime(15000);

    // Should NOT have been dismissed because user is interacting
    expect(mockDismiss).not.toHaveBeenCalled();
  });

  it('resumes the countdown after the pointer leaves (dsr-5)', () => {
    render(
      <ScanProgressCard
        state={completeState}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
      />
    );

    const card = screen.getByTestId('scan-progress-card');
    fireEvent.mouseEnter(card);
    vi.advanceTimersByTime(15000);
    fireEvent.mouseLeave(card);
    expect(mockDismiss).not.toHaveBeenCalled();

    // Used to stay forever once pointed at: leaving never restarted the timer.
    vi.advanceTimersByTime(10000);
    expect(mockDismiss).toHaveBeenCalledTimes(1);
  });

  it('shows cancelling state on confirm button', () => {
    render(
      <ScanProgressCard
        state={baseScanningState}
        onCancel={mockCancel}
        onToggleMinimize={mockToggleMinimize}
        onDismiss={mockDismiss}
        isCancelling={true}
      />
    );

    fireEvent.click(screen.getByTestId('scan-cancel-btn'));
    expect(screen.getByTestId('cancel-confirm-btn')).toHaveTextContent('取消中...');
    expect(screen.getByTestId('cancel-confirm-btn')).toBeDisabled();
  });
});
