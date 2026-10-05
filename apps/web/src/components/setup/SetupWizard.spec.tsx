import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router';
import { SetupWizard } from './SetupWizard';

// Mock setup service
vi.mock('../../services/setupService', () => ({
  setupService: {
    getStatus: vi.fn().mockResolvedValue({ needsSetup: true }),
    completeSetup: vi.fn().mockResolvedValue({ message: 'ok' }),
    validateStep: vi.fn().mockResolvedValue({ valid: true }),
  },
}));

function createTestRouter() {
  const rootRoute = createRootRoute();
  const setupRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/setup',
    component: () => <SetupWizard />,
  });
  const indexRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/',
    component: () => <div data-testid="dashboard">Dashboard</div>,
  });

  const router = createRouter({
    routeTree: rootRoute.addChildren([setupRoute, indexRoute]),
    history: createMemoryHistory({ initialEntries: ['/setup'] }),
  });

  return router;
}

function renderWithProviders() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const router = createTestRouter();

  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  );
}

describe('SetupWizard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the wizard container', async () => {
    renderWithProviders();
    expect(await screen.findByTestId('setup-wizard')).toBeInTheDocument();
  });

  it('shows step 1 of 6 on initial render', async () => {
    renderWithProviders();
    expect(await screen.findByText('步驟 1 / 6')).toBeInTheDocument();
  });

  it('shows the welcome step first', async () => {
    renderWithProviders();
    expect(await screen.findByTestId('welcome-step')).toBeInTheDocument();
    expect(screen.getByTestId('language-select')).toBeInTheDocument();
  });

  it('renders step progress dots', async () => {
    renderWithProviders();
    expect(await screen.findByTestId('step-progress')).toBeInTheDocument();
    expect(screen.getByTestId('step-dot-welcome')).toBeInTheDocument();
    expect(screen.getByTestId('step-dot-complete')).toBeInTheDocument();
  });

  it('navigates to next step on Next click', async () => {
    renderWithProviders();
    const nextBtn = await screen.findByTestId('next-button');
    fireEvent.click(nextBtn);
    expect(await screen.findByTestId('qbittorrent-step')).toBeInTheDocument();
  });

  it('navigates back from qbittorrent step', async () => {
    renderWithProviders();
    // Go to step 2
    fireEvent.click(await screen.findByTestId('next-button'));
    expect(await screen.findByTestId('qbittorrent-step')).toBeInTheDocument();
    // Go back
    fireEvent.click(screen.getByTestId('back-button'));
    expect(await screen.findByTestId('welcome-step')).toBeInTheDocument();
  });

  it('shows skip button on optional steps (qbittorrent)', async () => {
    renderWithProviders();
    fireEvent.click(await screen.findByTestId('next-button'));
    expect(await screen.findByTestId('skip-button')).toBeInTheDocument();
  });

  it('can skip qbittorrent step', async () => {
    renderWithProviders();
    fireEvent.click(await screen.findByTestId('next-button'));
    fireEvent.click(await screen.findByTestId('skip-button'));
    expect(await screen.findByTestId('media-library-step')).toBeInTheDocument();
  });

  it('shows all 6 step dots', async () => {
    renderWithProviders();
    await screen.findByTestId('step-progress');
    expect(screen.getByTestId('step-dot-welcome')).toBeInTheDocument();
    expect(screen.getByTestId('step-dot-qbittorrent')).toBeInTheDocument();
    expect(screen.getByTestId('step-dot-media-folder')).toBeInTheDocument();
    expect(screen.getByTestId('step-dot-api-keys')).toBeInTheDocument();
    expect(screen.getByTestId('step-dot-usage-report')).toBeInTheDocument();
    expect(screen.getByTestId('step-dot-complete')).toBeInTheDocument();
  });

  it('navigates through all steps to complete', async () => {
    renderWithProviders();

    // Step 1: Welcome → Next
    fireEvent.click(await screen.findByTestId('next-button'));

    // Step 2: qBittorrent → Skip
    fireEvent.click(await screen.findByTestId('skip-button'));

    // Step 3: Media Library → enter path then Next
    const libraryPath = await screen.findByTestId('library-path-0');
    fireEvent.change(libraryPath, { target: { value: '/media/videos' } });
    fireEvent.click(screen.getByTestId('next-button'));

    // Step 4: API Keys → Skip
    expect(await screen.findByTestId('api-keys-step')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('skip-button'));

    // Step 5: Usage report → Next (no 跳過 on this step)
    expect(await screen.findByTestId('usage-report-step')).toBeInTheDocument();
    expect(screen.queryByTestId('skip-button')).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId('next-button'));

    // Step 6: Complete
    expect(await screen.findByTestId('complete-step')).toBeInTheDocument();
    expect(screen.getByTestId('finish-button')).toBeInTheDocument();
  });

  it('shows summary on complete step', async () => {
    renderWithProviders();

    // Navigate to complete
    fireEvent.click(await screen.findByTestId('next-button')); // → qbt
    fireEvent.click(await screen.findByTestId('skip-button')); // → media library
    const libraryPath = await screen.findByTestId('library-path-0');
    fireEvent.change(libraryPath, { target: { value: '/media' } });
    fireEvent.click(screen.getByTestId('next-button')); // → api-keys
    fireEvent.click(await screen.findByTestId('skip-button')); // → usage-report
    fireEvent.click(await screen.findByTestId('next-button')); // → complete

    expect(await screen.findByText('設定完成！')).toBeInTheDocument();
    expect(screen.getByText('繁體中文')).toBeInTheDocument();
  });

  it('submits setup on finish click', async () => {
    const { setupService } = await import('../../services/setupService');
    renderWithProviders();

    // Navigate to complete
    fireEvent.click(await screen.findByTestId('next-button'));
    fireEvent.click(await screen.findByTestId('skip-button'));
    const libraryPath = await screen.findByTestId('library-path-0');
    fireEvent.change(libraryPath, { target: { value: '/media' } });
    fireEvent.click(screen.getByTestId('next-button'));
    fireEvent.click(await screen.findByTestId('skip-button'));
    await screen.findByTestId('usage-report-step');
    fireEvent.click(screen.getByTestId('next-button'));

    // Click finish
    fireEvent.click(await screen.findByTestId('finish-button'));

    await waitFor(() => {
      expect(setupService.completeSetup).toHaveBeenCalledWith(
        expect.objectContaining({
          language: 'zh-TW',
          libraries: expect.arrayContaining([
            expect.objectContaining({ path: '/media', contentType: 'movie' }),
          ]),
          usageReportEnabled: false,
        })
      );
    });
  });

  it('sends the usage-report opt-in when the switch was turned on', async () => {
    const { setupService } = await import('../../services/setupService');
    renderWithProviders();

    fireEvent.click(await screen.findByTestId('next-button'));
    fireEvent.click(await screen.findByTestId('skip-button'));
    const libraryPath = await screen.findByTestId('library-path-0');
    fireEvent.change(libraryPath, { target: { value: '/media' } });
    fireEvent.click(screen.getByTestId('next-button'));
    fireEvent.click(await screen.findByTestId('skip-button'));
    fireEvent.click(await screen.findByRole('switch', { name: '每週送一次匿名計數' }));
    fireEvent.click(screen.getByTestId('next-button'));

    expect(await screen.findByText('開啟')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('finish-button'));
    await waitFor(() => {
      expect(setupService.completeSetup).toHaveBeenCalledWith(
        expect.objectContaining({ usageReportEnabled: true })
      );
    });
  });

  it('going back from the summary keeps the switch where it was left', async () => {
    renderWithProviders();

    fireEvent.click(await screen.findByTestId('next-button'));
    fireEvent.click(await screen.findByTestId('skip-button'));
    const libraryPath = await screen.findByTestId('library-path-0');
    fireEvent.change(libraryPath, { target: { value: '/media' } });
    fireEvent.click(screen.getByTestId('next-button'));
    fireEvent.click(await screen.findByTestId('skip-button'));
    fireEvent.click(await screen.findByRole('switch', { name: '每週送一次匿名計數' }));
    fireEvent.click(screen.getByTestId('next-button'));
    await screen.findByText('開啟');

    fireEvent.click(screen.getByTestId('back-button'));

    expect(await screen.findByRole('switch', { name: '每週送一次匿名計數' })).toHaveAttribute(
      'aria-checked',
      'true'
    );
  });

  it('shows the server’s (zh-TW) reason when validation fails', async () => {
    const { setupService } = await import('../../services/setupService');
    vi.mocked(setupService.validateStep).mockRejectedValueOnce(new Error('請選擇語言。'));

    renderWithProviders();
    // Change language to empty and try to proceed
    const select = await screen.findByTestId('language-select');
    fireEvent.change(select, { target: { value: '' } });
    fireEvent.click(screen.getByTestId('next-button'));

    expect(await screen.findByTestId('setup-error')).toHaveTextContent('請選擇語言。');
  });

  // disc-setup-wizard-container-path-hint AC #4: no English fallbacks.
  it('a validation failure without a message reads in zh-TW', async () => {
    const { setupService } = await import('../../services/setupService');
    vi.mocked(setupService.validateStep).mockRejectedValueOnce({});

    renderWithProviders();
    fireEvent.click(await screen.findByTestId('next-button'));

    expect(await screen.findByTestId('setup-error')).toHaveTextContent(
      '這一步沒有通過檢查，請再試一次。'
    );
  });

  it('a failed 完成設定 reads in zh-TW, not the server’s English', async () => {
    const { setupService } = await import('../../services/setupService');
    vi.mocked(setupService.completeSetup).mockRejectedValueOnce(
      new Error('Failed to complete setup')
    );
    renderWithProviders();

    fireEvent.click(await screen.findByTestId('next-button'));
    fireEvent.click(await screen.findByTestId('skip-button'));
    const libraryPath = await screen.findByTestId('library-path-0');
    fireEvent.change(libraryPath, { target: { value: '/media' } });
    fireEvent.click(screen.getByTestId('next-button'));
    fireEvent.click(await screen.findByTestId('skip-button'));
    await screen.findByTestId('usage-report-step');
    fireEvent.click(screen.getByTestId('next-button')); // usage-report → complete
    fireEvent.click(await screen.findByTestId('finish-button'));

    expect(await screen.findByTestId('setup-error')).toHaveTextContent(
      '設定沒有完成，請再試一次。'
    );
    expect(screen.getByTestId('setup-error')).not.toHaveTextContent('Failed');
  });

  it('a 完成設定 refusal the server wrote in zh-TW is shown as is', async () => {
    const { setupService } = await import('../../services/setupService');
    vi.mocked(setupService.completeSetup).mockRejectedValueOnce(
      new Error('設定已經完成過了，請重新整理頁面。')
    );
    renderWithProviders();

    fireEvent.click(await screen.findByTestId('next-button'));
    fireEvent.click(await screen.findByTestId('skip-button'));
    const libraryPath = await screen.findByTestId('library-path-0');
    fireEvent.change(libraryPath, { target: { value: '/media' } });
    fireEvent.click(screen.getByTestId('next-button'));
    fireEvent.click(await screen.findByTestId('skip-button'));
    await screen.findByTestId('usage-report-step');
    fireEvent.click(screen.getByTestId('next-button')); // usage-report → complete
    fireEvent.click(await screen.findByTestId('finish-button'));

    expect(await screen.findByTestId('setup-error')).toHaveTextContent(
      '設定已經完成過了，請重新整理頁面。'
    );
  });

  it('shows skip warning on API keys step when no keys entered', async () => {
    renderWithProviders();

    fireEvent.click(await screen.findByTestId('next-button')); // → qbt
    fireEvent.click(await screen.findByTestId('skip-button')); // → media library
    const libraryPath = await screen.findByTestId('library-path-0');
    fireEvent.change(libraryPath, { target: { value: '/media' } });
    fireEvent.click(screen.getByTestId('next-button')); // → api-keys

    expect(await screen.findByTestId('skip-warning')).toBeInTheDocument();
  });

  it('sends a Claude key, not a provider + key pair the server never read (dsr-13)', async () => {
    const { setupService } = await import('../../services/setupService');
    renderWithProviders();

    fireEvent.click(await screen.findByTestId('next-button')); // → qbt
    fireEvent.click(await screen.findByTestId('skip-button')); // → media library
    fireEvent.change(await screen.findByTestId('library-path-0'), { target: { value: '/media' } });
    fireEvent.click(screen.getByTestId('next-button')); // → api-keys
    fireEvent.change(await screen.findByTestId('claude-key-input'), {
      target: { value: 'sk-ant-test' },
    });
    fireEvent.click(screen.getByTestId('next-button')); // → usage-report
    await screen.findByTestId('usage-report-step');
    fireEvent.click(screen.getByTestId('next-button')); // usage-report → complete
    fireEvent.click(await screen.findByTestId('finish-button'));

    // The api-keys step sends the Claude key for validation too, so a server
    // that cannot store keys (no ENCRYPTION_KEY) can refuse on this step.
    expect(setupService.validateStep).toHaveBeenCalledWith('api-keys', {
      tmdbApiKey: '',
      claudeApiKey: 'sk-ant-test',
    });
    await waitFor(() => {
      expect(setupService.completeSetup).toHaveBeenCalledWith(
        expect.objectContaining({ claudeApiKey: 'sk-ant-test' })
      );
    });
  });

  it('跳過 throws away a half-typed key instead of submitting it unvalidated', async () => {
    const { setupService } = await import('../../services/setupService');
    renderWithProviders();

    fireEvent.click(await screen.findByTestId('next-button')); // → qbt
    fireEvent.click(await screen.findByTestId('skip-button')); // → media library
    fireEvent.change(await screen.findByTestId('library-path-0'), { target: { value: '/media' } });
    fireEvent.click(screen.getByTestId('next-button')); // → api-keys
    fireEvent.change(await screen.findByTestId('tmdb-key-input'), {
      target: { value: 'too-short' },
    });
    fireEvent.click(screen.getByTestId('skip-button')); // → usage-report
    await screen.findByTestId('usage-report-step');
    fireEvent.click(screen.getByTestId('next-button')); // usage-report → complete

    expect(await screen.findByTestId('complete-step')).toBeInTheDocument();
    expect(screen.getAllByText('未設定')).toHaveLength(3); // qBittorrent · TMDb · Claude
    fireEvent.click(screen.getByTestId('finish-button'));
    await waitFor(() => {
      expect(setupService.completeSetup).toHaveBeenCalledWith(
        expect.objectContaining({ tmdbApiKey: undefined })
      );
    });
  });

  it('keeps the step count for screen readers now that the visible line is gone', async () => {
    renderWithProviders();
    expect(await screen.findByText('步驟 1 / 6')).toHaveClass('sr-only');
  });
});
