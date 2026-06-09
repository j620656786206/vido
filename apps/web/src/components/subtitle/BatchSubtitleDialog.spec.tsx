import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BatchSubtitlePanel, BatchSubtitleDialog } from './BatchSubtitleDialog';
import type { BatchProgressState } from '../../hooks/useSubtitleBatchProgress';

// --- Mocks ---
const mockNavigate = vi.fn();
vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => mockNavigate,
}));

const mockStartBatch = vi.fn();
const mockCancelBatch = vi.fn();
vi.mock('../../services/subtitleService', () => ({
  subtitleService: {
    startBatch: (...args: unknown[]) => mockStartBatch(...args),
    cancelBatch: (...args: unknown[]) => mockCancelBatch(...args),
  },
}));

const mockStartTracking = vi.fn();
const mockReset = vi.fn();
let hookState: { progress: BatchProgressState; status: string };
vi.mock('../../hooks/useSubtitleBatchProgress', () => ({
  useSubtitleBatchProgress: () => ({
    progress: hookState.progress,
    status: hookState.status,
    startTracking: mockStartTracking,
    reset: mockReset,
  }),
}));

const baseProgress: BatchProgressState = {
  batchId: 'b-1',
  totalItems: 10,
  currentIndex: 4,
  currentItem: '電影 A',
  successCount: 3,
  failCount: 1,
  status: 'running',
};

beforeEach(() => {
  vi.clearAllMocks();
  hookState = {
    progress: { ...baseProgress, status: 'idle', currentIndex: 0, successCount: 0, failCount: 0 },
    status: 'idle',
  };
});

describe('BatchSubtitlePanel (presentational state machine)', () => {
  const handlers = {
    onStart: vi.fn(),
    onConfirmCancel: vi.fn(),
    onViewNotFound: vi.fn(),
    onClose: vi.fn(),
  };

  it('idle: shows scope selector + start button; season hidden without seasonId', () => {
    render(
      <BatchSubtitlePanel
        status="idle"
        progress={{ ...baseProgress, status: 'idle' }}
        {...handlers}
      />
    );
    expect(screen.getByTestId('batch-subtitle-scope-library')).toBeInTheDocument();
    expect(screen.queryByTestId('batch-subtitle-scope-season')).not.toBeInTheDocument();
    expect(screen.getByTestId('batch-subtitle-start-btn')).toBeInTheDocument();
  });

  it('idle: season scope visible when seasonId provided', () => {
    render(
      <BatchSubtitlePanel
        status="idle"
        progress={{ ...baseProgress, status: 'idle' }}
        seasonId="season-1"
        {...handlers}
      />
    );
    expect(screen.getByTestId('batch-subtitle-scope-season')).toBeInTheDocument();
  });

  it('idle: start button invokes onStart with the selected scope', () => {
    render(
      <BatchSubtitlePanel
        status="idle"
        progress={{ ...baseProgress, status: 'idle' }}
        seasonId="season-1"
        {...handlers}
      />
    );
    fireEvent.click(screen.getByTestId('batch-subtitle-scope-season'));
    fireEvent.click(screen.getByTestId('batch-subtitle-start-btn'));
    expect(handlers.onStart).toHaveBeenCalledWith('season');
  });

  it('processing: shows progress bar, counter, found/not-found stats', () => {
    render(<BatchSubtitlePanel status="running" progress={baseProgress} {...handlers} />);
    expect(screen.getByTestId('batch-subtitle-progress-bar')).toBeInTheDocument();
    expect(screen.getByTestId('batch-subtitle-counter')).toHaveTextContent('4 / 10');
    expect(screen.getByTestId('batch-subtitle-found')).toHaveTextContent('找到 3');
    expect(screen.getByTestId('batch-subtitle-notfound')).toHaveTextContent('未找到 1');
  });

  it('processing: cancel requires confirmation before firing onConfirmCancel', () => {
    render(<BatchSubtitlePanel status="running" progress={baseProgress} {...handlers} />);
    fireEvent.click(screen.getByTestId('batch-subtitle-cancel-btn'));
    expect(screen.getByTestId('batch-subtitle-cancel-confirm')).toBeInTheDocument();
    expect(handlers.onConfirmCancel).not.toHaveBeenCalled();

    fireEvent.click(screen.getByTestId('batch-subtitle-cancel-confirm-btn'));
    expect(handlers.onConfirmCancel).toHaveBeenCalledOnce();
  });

  it('complete: shows summary, close, and view-not-found link', () => {
    render(
      <BatchSubtitlePanel
        status="complete"
        progress={{ ...baseProgress, currentIndex: 10, successCount: 8, failCount: 2 }}
        {...handlers}
      />
    );
    expect(screen.getByTestId('batch-subtitle-summary')).toHaveTextContent(
      '找到 8 · 未找到 2 · 共 10'
    );
    fireEvent.click(screen.getByTestId('batch-subtitle-view-notfound'));
    expect(handlers.onViewNotFound).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByTestId('batch-subtitle-close-btn'));
    expect(handlers.onClose).toHaveBeenCalledOnce();
  });
});

describe('BatchSubtitleDialog (container wiring)', () => {
  it('returns null when closed', () => {
    const { container } = render(<BatchSubtitleDialog open={false} onOpenChange={vi.fn()} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('start: calls startBatch then startTracking with the 202 result', async () => {
    mockStartBatch.mockResolvedValueOnce({
      conflict: false,
      result: { batchId: 'b-99', totalItems: 25 },
    });

    render(<BatchSubtitleDialog open onOpenChange={vi.fn()} />);
    fireEvent.click(screen.getByTestId('batch-subtitle-start-btn'));

    await waitFor(() =>
      expect(mockStartBatch).toHaveBeenCalledWith({ scope: 'library', seasonId: undefined })
    );
    await waitFor(() =>
      expect(mockStartTracking).toHaveBeenCalledWith({ batchId: 'b-99', totalItems: 25 })
    );
  });

  it('start: recovers from a 409 conflict by tracking the in-progress batch', async () => {
    const inProgress = { ...baseProgress, batchId: 'b-running' };
    mockStartBatch.mockResolvedValueOnce({ conflict: true, progress: inProgress });

    render(<BatchSubtitleDialog open onOpenChange={vi.fn()} />);
    fireEvent.click(screen.getByTestId('batch-subtitle-start-btn'));

    await waitFor(() => expect(mockStartTracking).toHaveBeenCalledWith(inProgress));
  });

  it('view-not-found navigates to /library with the subtitleStatus param', () => {
    hookState = { progress: { ...baseProgress, status: 'complete' }, status: 'complete' };
    render(<BatchSubtitleDialog open onOpenChange={vi.fn()} />);

    fireEvent.click(screen.getByTestId('batch-subtitle-view-notfound'));
    expect(mockNavigate).toHaveBeenCalledTimes(1);
    const arg = mockNavigate.mock.calls[0][0];
    expect(arg.to).toBe('/library');
    expect(arg.search({})).toEqual({ subtitleStatus: 'not_found' });
  });

  it('confirming cancel calls cancelBatch', async () => {
    mockCancelBatch.mockResolvedValueOnce({ cancelled: true });
    hookState = { progress: baseProgress, status: 'running' };
    render(<BatchSubtitleDialog open onOpenChange={vi.fn()} />);

    fireEvent.click(screen.getByTestId('batch-subtitle-cancel-btn'));
    fireEvent.click(screen.getByTestId('batch-subtitle-cancel-confirm-btn'));
    await waitFor(() => expect(mockCancelBatch).toHaveBeenCalledOnce());
  });
});
