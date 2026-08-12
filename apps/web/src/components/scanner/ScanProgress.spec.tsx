import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ScanProgress } from './ScanProgress';

const mockUseScanProgress = vi.fn();
const mockUseCancelScan = vi.fn();

vi.mock('../../hooks/useScanProgress', () => ({
  useScanProgress: () => mockUseScanProgress(),
  subscribeScanTracking: () => () => {},
  requestScanTracking: vi.fn(),
}));

vi.mock('../../hooks/useScanner', () => ({
  useCancelScan: () => mockUseCancelScan(),
}));

const mockPreview = vi.fn();
vi.mock('../../services/subtitleService', () => ({
  subtitleService: {
    previewGenerationBatch: () => mockPreview(),
  },
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
  startTracking: vi.fn(),
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

  // sub-5-1 AC #7: the F17 toast count includes episodes — its number must
  // match the consent list the toast links to, which lists episodes too.
  it('completion toast reads totalItemsIncludingEpisodes over the movies-only count', async () => {
    Object.defineProperty(window, 'innerWidth', { value: 1024, writable: true });
    mockPreview.mockResolvedValue({ totalItems: 38, totalItemsIncludingEpisodes: 142 });
    mockUseScanProgress.mockReturnValue({
      ...scanningState,
      isScanning: false,
      isComplete: true,
      percentDone: 100,
    });

    renderWithProviders();
    expect(await screen.findByText('142')).toBeInTheDocument();
    expect(screen.queryByText('38')).not.toBeInTheDocument();
  });

  it('completion toast falls back to totalItems on a pre-sub-5-1 server', async () => {
    Object.defineProperty(window, 'innerWidth', { value: 1024, writable: true });
    mockPreview.mockResolvedValue({ totalItems: 38 });
    mockUseScanProgress.mockReturnValue({
      ...scanningState,
      isScanning: false,
      isComplete: true,
      percentDone: 100,
    });

    renderWithProviders();
    expect(await screen.findByText('38')).toBeInTheDocument();
  });
});
