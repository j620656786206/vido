import React from 'react';
import { render, screen, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const navigateMock = vi.fn();
vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => navigateMock,
}));

const loginMock = vi.fn();
vi.mock('../../services/authService', async () => {
  // AuthError is a real class the component instanceof-checks, so keep it.
  const actual = await vi.importActual<typeof import('../../services/authService')>(
    '../../services/authService'
  );
  return {
    ...actual,
    authService: { login: (...args: unknown[]) => loginMock(...args) },
  };
});

import { AuthError } from '../../services/authService';
import { LoginForm } from './LoginForm';

function renderForm(props: { justLoggedOut?: boolean } = {}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <LoginForm {...props} />
    </QueryClientProvider>
  );
}

async function submit(password: string) {
  await userEvent.type(screen.getByLabelText('密碼'), password);
  await userEvent.click(screen.getByRole('button', { name: /登入/ }));
}

describe('LoginForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    loginMock.mockResolvedValue({ authEnabled: true, authenticated: true });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('focuses the only field on the page', () => {
    renderForm();
    expect(document.activeElement).toBe(screen.getByLabelText('密碼'));
  });

  it('logs in and enters the app', async () => {
    renderForm();
    await submit('vido1234');

    await waitFor(() => expect(loginMock).toHaveBeenCalledWith('vido1234'));
    expect(navigateMock).toHaveBeenCalledWith({ to: '/' });
  });

  // The recovery path for a self-hosted install is a file on a machine the user
  // owns. It is the one thing this audience can act on, and it must not be
  // hidden behind a failed attempt.
  it('always names VIDO_AUTH_PASSWORD as the recovery path', () => {
    renderForm();
    const help = screen.getByTestId('password-help');
    expect(help).toHaveTextContent('VIDO_AUTH_PASSWORD');
    expect(help).toHaveTextContent('重啟容器');
  });

  // The server sends `suggestion` alongside `message` on every auth failure.
  // Dropping it is how "還可以再試 2 次" never reached anyone.
  it("renders the API's suggestion, not just its message", async () => {
    loginMock.mockRejectedValue(
      new AuthError({
        message: '密碼錯誤',
        code: 'INVALID_CREDENTIALS',
        suggestion: '還可以再試 2 次,之後會鎖定 60 秒。',
      })
    );
    renderForm();
    await submit('nope');

    // The message is the live region; the suggestion sits beside it so a ticking
    // countdown can be excluded from announcements without silencing the error.
    expect(await screen.findByRole('alert')).toHaveTextContent('密碼錯誤');
    expect(screen.getByText('還可以再試 2 次,之後會鎖定 60 秒。')).toBeInTheDocument();
  });

  it('returns focus to the field and selects the old value after a failure', async () => {
    loginMock.mockRejectedValue(
      new AuthError({ message: '密碼錯誤', code: 'INVALID_CREDENTIALS' })
    );
    renderForm();
    await submit('nope');

    const input = screen.getByLabelText('密碼') as HTMLInputElement;
    await waitFor(() => expect(document.activeElement).toBe(input));
    expect(input.selectionStart).toBe(0);
    expect(input.selectionEnd).toBe('nope'.length);
  });

  it('clears the error as soon as the user starts retyping', async () => {
    loginMock.mockRejectedValue(
      new AuthError({ message: '密碼錯誤', code: 'INVALID_CREDENTIALS' })
    );
    renderForm();
    await submit('nope');
    await screen.findByText('密碼錯誤');

    await userEvent.type(screen.getByLabelText('密碼'), 'x');
    expect(screen.queryByText('密碼錯誤')).toBeNull();
  });

  it('reveals and re-hides the password', async () => {
    renderForm();
    const input = screen.getByLabelText('密碼');
    expect(input).toHaveAttribute('type', 'password');

    await userEvent.click(screen.getByTestId('reveal-password'));
    expect(input).toHaveAttribute('type', 'text');

    await userEvent.click(screen.getByTestId('reveal-password'));
    expect(input).toHaveAttribute('type', 'password');
  });

  // The lockout is a real server-issued deadline. Showing it counting down is
  // the difference between "try again" and "you are locked out for 60 seconds".
  it('counts a lockout down and re-enables itself when it lifts', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    loginMock.mockRejectedValue(
      new AuthError({
        message: '嘗試次數過多',
        code: 'TOO_MANY_ATTEMPTS',
        suggestion: '密碼連續錯誤 5 次,請等 60 秒後再試。',
        retryAfterSeconds: 3,
      })
    );
    renderForm();
    await submit('nope');

    const button = await screen.findByRole('button', { name: /還要等/ });
    expect(button).toBeDisabled();
    expect(button).toHaveTextContent('還要等 3 秒');

    await act(async () => {
      vi.advanceTimersByTime(3000);
    });

    await waitFor(() => expect(screen.getByRole('button', { name: '登入' })).toBeEnabled());
  });

  // Inside role="alert" the per-second countdown interrupted a screen reader 60
  // times for one piece of news. The message announces; the seconds do not.
  it('does not re-announce the countdown every second', async () => {
    loginMock.mockRejectedValue(
      new AuthError({
        message: '嘗試次數過多',
        code: 'TOO_MANY_ATTEMPTS',
        suggestion: '密碼連續錯誤 5 次,請等 60 秒後再試。',
        retryAfterSeconds: 30,
      })
    );
    renderForm();
    await submit('nope');

    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent('嘗試次數過多');
    // The ticking line is present for sighted users but out of the live region.
    const ticking = screen.getByText(/請等 \d+ 秒後再試。/);
    expect(ticking).toHaveAttribute('aria-hidden', 'true');
    expect(alert).not.toContainElement(ticking);
  });

  // A non-lockout failure keeps its suggestion readable — aria-hidden is only for
  // the value that changes every second.
  it('leaves an ordinary suggestion announceable', async () => {
    loginMock.mockRejectedValue(
      new AuthError({
        message: '密碼錯誤',
        code: 'INVALID_CREDENTIALS',
        suggestion: '還可以再試 2 次,之後會鎖定 60 秒。',
      })
    );
    renderForm();
    await submit('nope');

    const suggestion = await screen.findByText('還可以再試 2 次,之後會鎖定 60 秒。');
    expect(suggestion).not.toHaveAttribute('aria-hidden', 'true');
  });

  it('acknowledges an intentional logout', () => {
    renderForm({ justLoggedOut: true });
    expect(screen.getByTestId('logged-out-note')).toHaveTextContent('已登出');
  });

  // The disabled CTA is this screen's OPENING state, so it cannot be the thing
  // that fails the contrast floor: a token swap, never opacity.
  it('disables the submit button with tokens rather than opacity', () => {
    renderForm();
    const button = screen.getByRole('button', { name: '登入' });
    expect(button).toBeDisabled();
    expect(button.className).not.toContain('disabled:opacity');
    expect(button.className).toContain('disabled:bg-[var(--bg-tertiary)]');
    expect(button.className).toContain('min-h-[44px]');
  });

  // ── dsr-12: the gate has to be the same card shape as the rest of the app ──
  // The .pen Login Card node (M1-D I6UAK) carries fill + 1px stroke + 12px radius
  // and an EMPTY `effects` array. DESIGN.md §Cards and Containers measured why:
  // in 夜行 the shadow is worth 1.056:1 against the page, the hairline 1.660:1.
  it('is a bordered card, not a floating one', () => {
    renderForm();
    const card = screen.getByTestId('login-form');
    expect(card.className).toContain('border-[var(--border-subtle)]');
    expect(card.className).toContain('rounded-[var(--radius-lg)]');
    expect(card.className).not.toContain('shadow-');
  });

  // Every string on this screen must sit on a rung of the type scale. 11px is not
  // a rung — $Type/Label/Size is 12, which is Tailwind's `text-xs`.
  it('puts every label on the type scale instead of an 11px one-off', () => {
    const { container } = renderForm();
    const offScale = Array.from(container.querySelectorAll<HTMLElement>('*')).filter((el) =>
      /text-\[\d+px\]/.test(el.className)
    );
    expect(offScale.map((el) => el.className)).toEqual([]);

    expect(screen.getByText('NAS 媒體庫').className).toContain('text-xs');
    expect(screen.getByTestId('password-help').className).toContain('text-xs');
  });

  // 固定詞彙 ruling (Alexyu 2026-09-11, PM/UX review of Flow M): a lockout is NOT
  // 硃砂. Nothing is broken — you asked to log in, the server declined for a while.
  // That is 赭: 「你要求了但沒發生」. A wrong password IS 硃砂.
  it('paints a lockout in ochre and a wrong password in cinnabar', async () => {
    loginMock.mockRejectedValue(
      new AuthError({ message: '密碼錯誤', code: 'INVALID_CREDENTIALS' })
    );
    const { unmount } = renderForm();
    await submit('nope');
    expect((await screen.findByRole('alert')).className).toContain('text-[var(--error-text)]');
    unmount();

    loginMock.mockRejectedValue(
      new AuthError({
        message: '嘗試次數過多',
        code: 'TOO_MANY_ATTEMPTS',
        suggestion: '請等 43 秒後再試。',
        retryAfterSeconds: 43,
      })
    );
    renderForm();
    await submit('nope');
    const locked = await screen.findByRole('alert');
    expect(locked).toHaveTextContent('嘗試次數過多');
    expect(locked.className).toContain('text-[var(--warning-text)]');
    expect(locked.className).not.toContain('text-[var(--error-text)]');
  });

  // $Type/H2/Size is themed by breakpoint: 24 on desktop, 20 on mobile. A fixed
  // text-2xl wordmark is 24 on a 390px phone, where M6-M draws it at 20.
  it('steps the wordmark down on a phone', () => {
    renderForm();
    const wordmark = screen.getByText('vido');
    expect(wordmark.className).toContain('text-xl');
    expect(wordmark.className).toContain('sm:text-2xl');
  });
});
