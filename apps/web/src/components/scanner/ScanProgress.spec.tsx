import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ScanProgress } from './ScanProgress';

const mockUseScanProgress = vi.fn();
const mockUseCancelScan = vi.fn();

vi.mock('../../hooks/useScanProgress', () => ({
  useScanProgress: () => mockUseScanProgress(),
}));

vi.mock('../../hooks/useScanner', () => ({
  useCancelScan: () => mockUseCancelScan(),
}));

function renderWithProviders() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <ScanProgress />
    </QueryClientProvider>
  );
}

const scanningState = {
  isScanning: true,
  percentDone: 50,
  currentFile: 'file.mkv',
  filesFound: 100,
  filesProcessed: 50,
  errorCount: 0,
  estimatedTime: '1 分',
  isComplete: false,
  isCancelled: false,
  isMinimized: false,
  isDismissed: false,
  connectionStatus: 'sse' as const,
  isVisible: true,
  toggleMinimize: vi.fn(),
  dismiss: vi.fn(),
};

const originalMatchMedia = window.matchMedia;

describe('ScanProgress', () => {
  beforeEach(() => {
    mockUseCancelScan.mockReturnValue({ mutate: vi.fn(), isPending: false });
    // Mock matchMedia for all tests (jsdom doesn't provide it)
    window.matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: query.includes('min-width') && window.innerWidth >= 768,
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      onchange: null,
      dispatchEvent: vi.fn(),
    }));
  });

  afterEach(() => {
    window.matchMedia = originalMatchMedia;
  });

  it('renders nothing when not visible', () => {
    mockUseScanProgress.mockReturnValue({
      ...scanningState,
      isVisible: false,
    });

    const { container } = renderWithProviders();
    expect(container.innerHTML).toBe('');
  });

  it('renders desktop card on wide viewport', () => {
    Object.defineProperty(window, 'innerWidth', { value: 1024, writable: true });
    mockUseScanProgress.mockReturnValue(scanningState);

    renderWithProviders();
    expect(screen.getByTestId('scan-progress-wrapper')).toBeInTheDocument();
    expect(screen.getByTestId('scan-progress-card')).toBeInTheDocument();
  });

  it('renders mobile sheet on narrow viewport', () => {
    Object.defineProperty(window, 'innerWidth', { value: 375, writable: true });
    window.matchMedia = vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      onchange: null,
      dispatchEvent: vi.fn(),
    }));

    mockUseScanProgress.mockReturnValue(scanningState);

    renderWithProviders();
    expect(screen.getByTestId('scan-progress-wrapper')).toBeInTheDocument();
    expect(screen.getByTestId('scan-progress-sheet')).toBeInTheDocument();

    // Restore
    Object.defineProperty(window, 'innerWidth', { value: 1024, writable: true });
  });
});
