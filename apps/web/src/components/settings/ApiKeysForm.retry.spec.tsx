/**
 * dsr-3b CR #1 — 重試 against a REAL QueryClient.
 *
 * The main spec mocks useKeySettings, and a mock can say "isError + not
 * fetching" forever. A real failed query with nothing cached goes back to
 * pending the moment it is refetched (project_tanstack_refetch_no_data_resets_pending),
 * which used to swap the whole page for the spinner — dropping the form and the
 * TMDB attribution that fail-soft exists to keep on screen.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const getKeys = vi.fn();

vi.mock('../../services/keySettingsService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../services/keySettingsService')>();
  return {
    ...actual,
    keySettingsService: { ...actual.keySettingsService, getKeys: () => getKeys() },
  };
});

import { ApiKeysForm } from './ApiKeysForm';

function renderForm() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <ApiKeysForm />
    </QueryClientProvider>
  );
}

describe('ApiKeysForm — 重試 keeps the page on screen (real QueryClient)', () => {
  beforeEach(() => {
    getKeys.mockReset();
  });

  it('keeps the banner, the form and the TMDB attribution while the retry is in flight', async () => {
    getKeys.mockRejectedValueOnce(new Error('Failed to fetch'));
    let release: (v: unknown) => void = () => {};
    getKeys.mockImplementationOnce(() => new Promise((resolve) => (release = resolve)));
    renderForm();

    const retry = await screen.findByRole('button', { name: '重試' });
    fireEvent.click(retry);

    // In flight: nothing swapped for a spinner.
    expect(screen.queryByTestId('api-keys-loading')).toBeNull();
    expect(screen.getByTestId('api-keys-load-error')).toBeInTheDocument();
    expect(screen.getByTestId('api-keys-form')).toBeInTheDocument();
    expect(screen.getByTestId('tmdb-attribution')).toBeInTheDocument();
    expect(screen.getByTestId('key-state-claude')).toHaveTextContent('無法確認');
    expect(screen.getByRole('button', { name: '重試' })).toHaveAttribute('aria-disabled', 'true');

    release({
      writable: true,
      keys: [
        { name: 'claude', configured: false, source: 'none' },
        { name: 'tmdb', configured: false, source: 'none' },
        { name: 'openai', configured: false, source: 'none' },
      ],
    });
    await waitFor(() => expect(screen.queryByTestId('api-keys-load-error')).toBeNull());
    expect(screen.getByTestId('key-state-claude')).toHaveTextContent('尚未設定');
  });

  // Only a failed READ greys nothing — the labels are muted for writable:false,
  // which is a known state with its own banner, not for "we could not ask".
  it('does not grey the labels just because the read failed', async () => {
    getKeys.mockRejectedValue(new Error('Failed to fetch'));
    renderForm();
    await screen.findByTestId('api-keys-load-error');
    expect(screen.getByText('Claude（翻譯）', { selector: 'label' })).toHaveClass(
      'text-[var(--text-primary)]'
    );
  });
});
