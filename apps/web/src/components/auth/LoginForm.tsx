// Time-bomb-exempt: the only wall-clock read is the lockout countdown, whose
// deadline comes from the server's Retry-After header — it renders only after a
// 429, a state no visual baseline captures (there is no login gallery fixture),
// so ambient Date.now() can never reach a snapshot. (Sally, critique 2026-09-01)
// Implements: <utility — no .pen counterpart>
/**
 * The password gate (V0.1.1). This is the first — and for a failed login, the
 * only — screen a self-hoster sees, so it follows the same rule as every readout
 * in the app: say what the server actually knows.
 *
 * The server knows three things this screen used to throw away:
 *   1. `suggestion` on the error body ("還可以再試 2 次…"),
 *   2. `Retry-After` in seconds when the per-IP lockout trips,
 *   3. that the only recovery path is editing VIDO_AUTH_PASSWORD and restarting.
 * (1) and (2) now render; (3) is the standing help line under the button,
 * because there is no account, no reset mail, and no support desk — the user
 * owns the machine, and that is the answer.
 *
 * Nothing above the submit button ever moves. The error lives in a reserved
 * slot under the field, so a failed try does not shift the input out from under
 * a user who is already reaching for it.
 */
import { useEffect, useRef, useState, type FormEvent } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useQueryClient } from '@tanstack/react-query';
import { Eye, EyeOff } from 'lucide-react';
import { AuthError, authService } from '../../services/authService';
import { authKeys } from '../../hooks/useAuthStatus';

interface LoginFormProps {
  /** True when the user arrived here by pressing 登出 rather than by being turned away. */
  justLoggedOut?: boolean;
}

export function LoginForm({ justLoggedOut = false }: LoginFormProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const inputRef = useRef<HTMLInputElement>(null);
  const [password, setPassword] = useState('');
  const [reveal, setReveal] = useState(false);
  const [error, setError] = useState<{ message: string; suggestion?: string } | null>(null);
  const [submitting, setSubmitting] = useState(false);
  /** Epoch ms when a per-IP lockout lifts, or null when we are not locked out. */
  const [lockedUntil, setLockedUntil] = useState<number | null>(null);
  const [secondsLeft, setSecondsLeft] = useState(0);

  // The lockout countdown. This is a real, server-issued deadline ticking down,
  // which is the one kind of number this app is allowed to animate — unlike a
  // progress bar for work nobody is measuring.
  useEffect(() => {
    if (lockedUntil === null) return;
    const tick = () => {
      const left = Math.max(0, Math.ceil((lockedUntil - Date.now()) / 1000));
      setSecondsLeft(left);
      if (left === 0) {
        setLockedUntil(null);
        setError(null);
        inputRef.current?.focus();
      }
    };
    tick();
    // eslint-disable-next-line local/no-hardcoded-duration -- a wall-clock tick, not a transition
    const id = window.setInterval(tick, 1000);
    return () => window.clearInterval(id);
  }, [lockedUntil]);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (lockedUntil !== null) return;
    setError(null);
    setSubmitting(true);
    try {
      await authService.login(password);
      // Refresh the cached auth status so the root guard lets us into the app.
      await queryClient.invalidateQueries({ queryKey: authKeys.status() });
      navigate({ to: '/' });
    } catch (err) {
      if (err instanceof AuthError) {
        setError({ message: err.message, suggestion: err.suggestion });
        if (err.isLockedOut && err.retryAfterSeconds) {
          setLockedUntil(Date.now() + err.retryAfterSeconds * 1000);
        }
      } else {
        setError({ message: '登入失敗,請再試一次。' });
      }
      setSubmitting(false);
      // Put the cursor back where the next attempt has to start, with the old
      // value selected so retyping replaces it.
      inputRef.current?.focus();
      inputRef.current?.select();
    }
  }

  const locked = lockedUntil !== null;
  const disabled = submitting || locked || password.length === 0;
  const buttonLabel = locked ? `還要等 ${secondsLeft} 秒` : submitting ? '登入中…' : '登入';

  return (
    <div
      className="w-full max-w-sm rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-6 shadow-[var(--shadow-md)] sm:p-8"
      data-testid="login-form"
    >
      {/* One wordmark for the whole product: lowercase, gold, with the same
          subtitle the sidebar carries three pixels away once you are inside. */}
      <div className="mb-6">
        <h1 className="text-2xl font-bold leading-none text-[var(--accent-text)]">vido</h1>
        <p className="mt-1 text-[11px] text-[var(--text-secondary)]">NAS 媒體庫</p>
      </div>

      {justLoggedOut && !error && (
        <p
          role="status"
          data-testid="logged-out-note"
          className="mb-4 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-3 py-2 text-sm text-[var(--text-secondary)]"
        >
          已登出。
        </p>
      )}

      <form onSubmit={handleSubmit}>
        <label
          htmlFor="password"
          className="mb-2 block text-sm font-medium text-[var(--text-secondary)]"
        >
          密碼
        </label>

        <div className="relative">
          <input
            id="password"
            ref={inputRef}
            // The rule guards against stealing focus on content pages. This is a
            // dedicated gate whose ONLY interactive job is this one field, and a
            // user who lands here has nothing else to read first.
            // eslint-disable-next-line jsx-a11y/no-autofocus
            autoFocus
            type={reveal ? 'text' : 'password'}
            autoComplete="current-password"
            value={password}
            onChange={(e) => {
              setPassword(e.target.value);
              // Clear the failure the moment the user starts fixing it, instead
              // of leaving "密碼錯誤" hanging over a field they have already retyped.
              if (error && !locked) setError(null);
            }}
            aria-invalid={error !== null}
            aria-describedby="password-status password-help"
            className="min-h-[44px] w-full rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-primary)] py-2.5 pl-4 pr-14 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
            placeholder="輸入密碼"
          />
          {/* The value here is a random string out of the user's own .env, typed
              on a phone keyboard more often than not. Being able to look at it
              is error PREVENTION, which is cheaper than the lockout it avoids. */}
          <button
            type="button"
            onClick={() => setReveal((v) => !v)}
            data-testid="reveal-password"
            aria-pressed={reveal}
            aria-label={reveal ? '隱藏密碼' : '顯示密碼'}
            className="absolute inset-y-0 right-0 flex h-11 w-11 items-center justify-center self-center rounded-[var(--radius-md)] text-[var(--text-muted)] transition-colors duration-[var(--motion-touch)] hover:text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]"
          >
            {reveal ? (
              <EyeOff className="h-4 w-4" aria-hidden="true" />
            ) : (
              <Eye className="h-4 w-4" aria-hidden="true" />
            )}
          </button>
        </div>

        {/* Reserved slot: the height is held whether or not there is an error, so
            a failed submit never shifts the field or the button. */}
        <div id="password-status" role="alert" className="min-h-[3.25rem] pt-2">
          {error && (
            <>
              <p className="text-sm text-[var(--error-text)]">{error.message}</p>
              {error.suggestion && (
                <p className="mt-0.5 text-[11px] text-[var(--text-secondary)]">
                  {locked ? `請等 ${secondsLeft} 秒後再試。` : error.suggestion}
                </p>
              )}
            </>
          )}
        </div>

        {/* Disabled state is a token swap, not opacity: half-transparent white on
            gold measures 2.54:1, and an empty field is this screen's OPENING
            state — so the very first thing a new user sees would have been a CTA
            that fails the contrast floor. Grey-on-grey reads as "not yet" and
            clears AA. */}
        <button
          type="submit"
          disabled={disabled}
          className="min-h-[44px] w-full rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-4 py-2.5 text-sm font-medium text-[var(--text-on-accent)] transition-[background-color,color] duration-[var(--motion-touch)] hover:bg-[var(--accent-pressed)] disabled:cursor-not-allowed disabled:bg-[var(--bg-tertiary)] disabled:text-[var(--text-muted)]"
        >
          {buttonLabel}
        </button>
      </form>

      {/* There is no account and no reset link — the recovery path is a variable
          on a machine the user owns, which they would never guess from "密碼錯誤".
          One sentence: the question, the variable, the restart. Naming where the
          variable lives (.env? compose?) is what made the first draft long, and
          the person who set it already knows. */}
      <p
        id="password-help"
        data-testid="password-help"
        className="mt-4 border-t border-[var(--border-subtle)] pt-4 text-[11px] leading-relaxed text-[var(--text-muted)]"
      >
        忘記密碼？改{' '}
        <code className="font-mono text-[var(--text-secondary)]">VIDO_AUTH_PASSWORD</code>{' '}
        再重啟容器。
      </p>
    </div>
  );
}
