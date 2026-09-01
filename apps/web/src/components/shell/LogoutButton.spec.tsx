import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const navigateMock = vi.fn();
vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => navigateMock,
}));

const useAuthStatusMock = vi.fn();
vi.mock('../../hooks/useAuthStatus', () => ({
  useAuthStatus: () => useAuthStatusMock(),
  authKeys: { all: ['auth'], status: () => ['auth', 'status'] },
}));

const logoutMock = vi.fn();
vi.mock('../../services/authService', () => ({
  authService: { logout: (...args: unknown[]) => logoutMock(...args) },
}));

// The Base UI tooltip wrapper renders its trigger child; passthrough keeps the
// button queryable without a floating-ui layer in jsdom.
vi.mock('../ui/Tooltip', () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => children,
}));

import { LogoutButton } from './LogoutButton';

function renderButton(variant: 'rail' | 'row' = 'row') {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const utils = render(
    <QueryClientProvider client={client}>
      <LogoutButton variant={variant} />
    </QueryClientProvider>
  );
  return { ...utils, client };
}

/** Open the confirm dialog and press 登出 in it. */
async function logOut() {
  await userEvent.click(screen.getByTestId('logout-button'));
  await screen.findByTestId('logout-confirm');
  await userEvent.click(screen.getByTestId('logout-confirm-button'));
}

describe('LogoutButton', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    logoutMock.mockResolvedValue(undefined);
    useAuthStatusMock.mockReturnValue({ data: { authEnabled: true, authenticated: true } });
  });

  // A LAN-only install (no VIDO_AUTH_PASSWORD) has no session to end. A logout
  // button that cannot log you out is worse than no button at all.
  it('renders nothing when the server has no password configured', () => {
    useAuthStatusMock.mockReturnValue({ data: { authEnabled: false, authenticated: true } });
    renderButton();
    expect(screen.queryByTestId('logout-button')).toBeNull();
  });

  it('renders nothing while auth status is still unknown', () => {
    useAuthStatusMock.mockReturnValue({ data: undefined });
    renderButton();
    expect(screen.queryByTestId('logout-button')).toBeNull();
  });

  // 14px / --text-secondary, the weight of a nav destination — NOT the 11px
  // muted row it was first copied from. Ending a session has more consequence
  // than changing a theme, and the type has to say so.
  it('carries destination weight in the row variant', () => {
    renderButton('row');
    const btn = screen.getByTestId('logout-button');
    expect(btn.className).toContain('text-sm');
    expect(btn.className).toContain('text-[var(--text-secondary)]');
    expect(btn.className).toContain('min-h-[44px]');
  });

  it('renders an icon-only 44px target on the collapsed rail', () => {
    renderButton('rail');
    const btn = screen.getByRole('button', { name: '登出' });
    expect(btn.className).toContain('h-11');
    expect(btn.className).toContain('w-11');
  });

  // On a shared NAS screen this control sits one row from a theme switch. A
  // mis-tap must not silently end the session.
  it('asks for confirmation instead of logging out on the first click', async () => {
    renderButton();
    await userEvent.click(screen.getByTestId('logout-button'));

    expect(await screen.findByTestId('logout-confirm')).toBeInTheDocument();
    expect(logoutMock).not.toHaveBeenCalled();
    expect(navigateMock).not.toHaveBeenCalled();
  });

  it('cancelling leaves the session alone', async () => {
    renderButton();
    await userEvent.click(screen.getByTestId('logout-button'));
    await userEvent.click(await screen.findByTestId('logout-cancel'));

    expect(logoutMock).not.toHaveBeenCalled();
    expect(navigateMock).not.toHaveBeenCalled();
  });

  it('calls the logout endpoint and sends the user to an acknowledged /login', async () => {
    renderButton();
    await logOut();

    await waitFor(() => expect(logoutMock).toHaveBeenCalledTimes(1));
    // `loggedOut` is what makes the gate say 已登出 rather than showing the
    // anonymous card a stranger would get.
    expect(navigateMock).toHaveBeenCalledWith({ to: '/login', search: { loggedOut: true } });
  });

  // Everything cached belongs to the session that just ended — the library, the
  // settings, the download queue. Leaving it in memory means the next person at
  // this screen sees the previous session's data behind the login form.
  it('clears the whole query cache, not just the auth key', async () => {
    const { client } = renderButton();
    client.setQueryData(['movies'], [{ id: 1, title: 'seeded' }]);
    const clearSpy = vi.spyOn(client, 'clear');

    await logOut();

    await waitFor(() => expect(clearSpy).toHaveBeenCalled());
    expect(client.getQueryData(['movies'])).toBeUndefined();
  });

  // The cookie clear is server-side and best-effort. If the API is down or
  // restarting, the user must still end up at the login screen rather than being
  // stranded in a shell whose every request 401s.
  it('still navigates to /login when the logout request fails', async () => {
    logoutMock.mockRejectedValue(new Error('network down'));
    renderButton();

    await logOut();

    await waitFor(() =>
      expect(navigateMock).toHaveBeenCalledWith({ to: '/login', search: { loggedOut: true } })
    );
  });
});
