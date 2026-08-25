import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { QBittorrentForm } from './QBittorrentForm';
import { QBittorrentApiError } from '../../services/qbittorrent';

// The error banner links to 金鑰設定; stub Link so this stays a component test
// instead of dragging in a memory router.
vi.mock('@tanstack/react-router', () => ({
  Link: ({ to, children, ...rest }: { to: string; children: React.ReactNode }) => (
    <a href={to} {...rest}>
      {children}
    </a>
  ),
}));

// Mock the hooks
const mockGetConfig = vi.fn();
const mockSaveMutate = vi.fn();
const mockTestMutate = vi.fn();

vi.mock('../../hooks/useQBittorrent', () => ({
  useQBittorrentConfig: () => mockGetConfig(),
  useSaveQBConfig: () => ({
    mutate: mockSaveMutate,
    isPending: false,
    isError: false,
    error: null,
  }),
  useTestQBConnection: () => ({
    mutate: mockTestMutate,
    isPending: false,
  }),
}));

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

describe('QBittorrentForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetConfig.mockReturnValue({
      data: { host: '', username: '', basePath: '', configured: false },
      isLoading: false,
    });
  });

  // An empty form is an assertion that nothing is saved. Rendering one over a
  // failed read tells a user whose config exists that they never configured it,
  // and they retype credentials that were already there.
  describe('load failure', () => {
    it('replaces the form with a banner instead of rendering empty fields', () => {
      mockGetConfig.mockReturnValue({
        data: undefined,
        isLoading: false,
        isError: true,
        error: new QBittorrentApiError(
          'Failed to retrieve qBittorrent configuration',
          'INTERNAL_ERROR'
        ),
        refetch: vi.fn(),
        isFetching: false,
      });

      renderWithProviders(<QBittorrentForm />);

      expect(screen.getByTestId('qb-config-load-error')).toBeInTheDocument();
      expect(screen.queryByLabelText('主機位址')).not.toBeInTheDocument();
      expect(screen.queryByText('儲存設定')).not.toBeInTheDocument();
      expect(screen.getByTestId('qb-config-error-keys-link')).toHaveAttribute(
        'href',
        '/settings/keys'
      );
    });

    it('names the real cause when the backend reports a decrypt failure', () => {
      mockGetConfig.mockReturnValue({
        data: undefined,
        isLoading: false,
        isError: true,
        error: new QBittorrentApiError(
          'Stored qBittorrent password could not be decrypted',
          'QBITTORRENT_CONFIG_DECRYPT_FAILED',
          'ENCRYPTION_KEY is missing or has changed'
        ),
        refetch: vi.fn(),
        isFetching: false,
      });

      renderWithProviders(<QBittorrentForm />);

      const banner = screen.getByTestId('qb-config-load-error');
      expect(banner).toHaveTextContent('儲存的密碼解不開');
      expect(banner).toHaveTextContent('ENCRYPTION_KEY');
      // The whole point: do NOT let the user conclude nothing was ever saved.
      expect(banner).toHaveTextContent('設定已經存在，不是沒設定過');
    });

    it('offers a retry that refetches', async () => {
      const refetch = vi.fn();
      mockGetConfig.mockReturnValue({
        data: undefined,
        isLoading: false,
        isError: true,
        error: new QBittorrentApiError('boom', 'INTERNAL_ERROR'),
        refetch,
        isFetching: false,
      });

      renderWithProviders(<QBittorrentForm />);
      await userEvent.click(screen.getByTestId('qb-config-retry'));

      expect(refetch).toHaveBeenCalledTimes(1);
    });
  });

  it('renders the form fields', () => {
    renderWithProviders(<QBittorrentForm />);

    expect(screen.getByLabelText('主機位址')).toBeInTheDocument();
    expect(screen.getByLabelText('使用者名稱')).toBeInTheDocument();
    expect(screen.getByLabelText('密碼')).toBeInTheDocument();
    expect(screen.getByText('測試連線')).toBeInTheDocument();
    expect(screen.getByText('儲存設定')).toBeInTheDocument();
  });

  it('shows loading state', () => {
    mockGetConfig.mockReturnValue({
      data: undefined,
      isLoading: true,
    });

    renderWithProviders(<QBittorrentForm />);

    expect(screen.queryByTestId('qbittorrent-form')).toBeNull();
  });

  it('populates form with existing config', () => {
    mockGetConfig.mockReturnValue({
      data: {
        host: 'http://192.168.1.100:8080',
        username: 'admin',
        basePath: '/qbt',
        configured: true,
      },
      isLoading: false,
    });

    renderWithProviders(<QBittorrentForm />);

    expect((screen.getByLabelText('主機位址') as HTMLInputElement).value).toBe(
      'http://192.168.1.100:8080'
    );
    expect((screen.getByLabelText('使用者名稱') as HTMLInputElement).value).toBe('admin');
  });

  it('disables buttons when required fields are empty', () => {
    renderWithProviders(<QBittorrentForm />);

    const testBtn = screen.getByText('測試連線').closest('button');
    const saveBtn = screen.getByText('儲存設定').closest('button');

    expect(testBtn?.disabled).toBe(true);
    expect(saveBtn?.disabled).toBe(true);
  });

  it('enables buttons when form is filled', async () => {
    const user = userEvent.setup();
    renderWithProviders(<QBittorrentForm />);

    await user.type(screen.getByLabelText('主機位址'), 'http://host:8080');
    await user.type(screen.getByLabelText('使用者名稱'), 'admin');
    await user.type(screen.getByLabelText('密碼'), 'pass');

    const testBtn = screen.getByText('測試連線').closest('button');
    const saveBtn = screen.getByText('儲存設定').closest('button');

    expect(testBtn?.disabled).toBe(false);
    expect(saveBtn?.disabled).toBe(false);
  });

  it('calls test mutation directly on test connection click (AC3)', async () => {
    const user = userEvent.setup();
    renderWithProviders(<QBittorrentForm />);

    await user.type(screen.getByLabelText('主機位址'), 'http://host:8080');
    await user.type(screen.getByLabelText('使用者名稱'), 'admin');
    await user.type(screen.getByLabelText('密碼'), 'secret');
    await user.click(screen.getByText('測試連線'));

    expect(mockTestMutate).toHaveBeenCalledWith(
      {
        host: 'http://host:8080',
        username: 'admin',
        password: 'secret',
        basePath: '',
      },
      expect.any(Object)
    );
    // Should NOT call save when testing
    expect(mockSaveMutate).not.toHaveBeenCalled();
  });

  it('calls save mutation on form submit', async () => {
    const user = userEvent.setup();
    renderWithProviders(<QBittorrentForm />);

    await user.type(screen.getByLabelText('主機位址'), 'http://host:8080');
    await user.type(screen.getByLabelText('使用者名稱'), 'admin');
    await user.type(screen.getByLabelText('密碼'), 'secret');
    await user.click(screen.getByText('儲存設定'));

    expect(mockSaveMutate).toHaveBeenCalledWith(
      {
        host: 'http://host:8080',
        username: 'admin',
        password: 'secret',
        basePath: '',
      },
      expect.any(Object)
    );
  });

  // --- Label verification: 主機位址 (not "Host URL") ---

  it('uses zh-TW label "主機位址" for the host field', () => {
    renderWithProviders(<QBittorrentForm />);
    expect(screen.getByLabelText('主機位址')).toBeInTheDocument();
    expect(screen.queryByLabelText('Host URL')).not.toBeInTheDocument();
  });

  // --- Button alignment: justify-end on desktop ---

  it('button container has md:justify-end for desktop right-alignment', () => {
    renderWithProviders(<QBittorrentForm />);
    const testBtn = screen.getByText('測試連線').closest('button');
    const buttonContainer = testBtn?.parentElement;
    expect(buttonContainer).toHaveClass('md:justify-end');
  });

  it('button container uses flex-col on mobile and flex-row on desktop', () => {
    renderWithProviders(<QBittorrentForm />);
    const testBtn = screen.getByText('測試連線').closest('button');
    const buttonContainer = testBtn?.parentElement;
    expect(buttonContainer).toHaveClass('flex-col');
    expect(buttonContainer).toHaveClass('md:flex-row');
  });

  // --- Base path field is optional ---

  it('renders base path field with optional indicator', () => {
    renderWithProviders(<QBittorrentForm />);
    expect(screen.getByLabelText(/Base Path/)).toBeInTheDocument();
    expect(screen.getByText('（選填，反向代理用）')).toBeInTheDocument();
  });

  it('base path field is not required', () => {
    renderWithProviders(<QBittorrentForm />);
    const basePath = screen.getByLabelText(/Base Path/);
    expect(basePath).not.toBeRequired();
  });

  // --- Form element and test-id ---

  it('renders with data-testid qbittorrent-form', () => {
    renderWithProviders(<QBittorrentForm />);
    expect(screen.getByTestId('qbittorrent-form')).toBeInTheDocument();
  });

  it('form element wraps all inputs', () => {
    renderWithProviders(<QBittorrentForm />);
    const form = screen.getByTestId('qbittorrent-form');
    expect(form.tagName).toBe('FORM');
    expect(form).toContainElement(screen.getByLabelText('主機位址'));
    expect(form).toContainElement(screen.getByLabelText('使用者名稱'));
    expect(form).toContainElement(screen.getByLabelText('密碼'));
  });

  // --- Password field type ---

  it('password field has type password for security', () => {
    renderWithProviders(<QBittorrentForm />);
    const passwordField = screen.getByLabelText('密碼');
    expect(passwordField).toHaveAttribute('type', 'password');
  });

  // --- Required fields ---

  it('host, username, and password fields are required', () => {
    renderWithProviders(<QBittorrentForm />);
    expect(screen.getByLabelText('主機位址')).toBeRequired();
    expect(screen.getByLabelText('使用者名稱')).toBeRequired();
    expect(screen.getByLabelText('密碼')).toBeRequired();
  });
});
