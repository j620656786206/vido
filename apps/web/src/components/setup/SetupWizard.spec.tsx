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

  it('shows step 1 of 5 on initial render', async () => {
    renderWithProviders();
    expect(await screen.findByText('步驟 1 / 5')).toBeInTheDocument();
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

  it('shows all 5 step dots', async () => {
    renderWithProviders();
    await screen.findByTestId('step-progress');
    expect(screen.getByTestId('step-dot-welcome')).toBeInTheDocument();
    expect(screen.getByTestId('step-dot-qbittorrent')).toBeInTheDocument();
    expect(screen.getByTestId('step-dot-media-folder')).toBeInTheDocument();
    expect(screen.getByTestId('step-dot-api-keys')).toBeInTheDocument();
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

    // Step 5: Complete
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
    fireEvent.click(await screen.findByTestId('skip-button')); // → complete

    expect(await screen.findByText('設定完成！')).toBeInTheDocument();
    expect(screen.getByText('zh-TW')).toBeInTheDocument();
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

    // Click finish
    fireEvent.click(await screen.findByTestId('finish-button'));

    await waitFor(() => {
      expect(setupService.completeSetup).toHaveBeenCalledWith(
        expect.objectContaining({
          language: 'zh-TW',
          libraries: expect.arrayContaining([
            expect.objectContaining({ path: '/media', contentType: 'movie' }),
          ]),
        })
      );
    });
  });

  it('shows error when validation fails', async () => {
    const { setupService } = await import('../../services/setupService');
    vi.mocked(setupService.validateStep).mockRejectedValueOnce(new Error('language is required'));

    renderWithProviders();
    // Change language to empty and try to proceed
    const select = await screen.findByTestId('language-select');
    fireEvent.change(select, { target: { value: '' } });
    fireEvent.click(screen.getByTestId('next-button'));

    expect(await screen.findByTestId('setup-error')).toBeInTheDocument();
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
});
