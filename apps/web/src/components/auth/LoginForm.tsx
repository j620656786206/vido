// Implements: <utility — no .pen counterpart>
import { useState, type FormEvent } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useQueryClient } from '@tanstack/react-query';
import { authService } from '../../services/authService';
import { authKeys } from '../../hooks/useAuthStatus';

export function LoginForm() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await authService.login(password);
      // Refresh the cached auth status so the root guard lets us into the app.
      await queryClient.invalidateQueries({ queryKey: authKeys.status() });
      navigate({ to: '/' });
    } catch (err) {
      setError(err instanceof Error ? err.message : '登入失敗,請再試一次。');
      setSubmitting(false);
    }
  }

  return (
    <div
      className="w-full max-w-sm rounded-2xl border border-[var(--border-subtle)]/50 bg-[var(--bg-primary)] p-8 shadow-2xl"
      data-testid="login-form"
    >
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-[var(--text-primary)]">Vido</h1>
        <p className="mt-1 text-sm text-[var(--text-secondary)]">請輸入密碼以繼續</p>
      </div>

      {error && (
        <div
          role="alert"
          className="mb-4 rounded-lg border border-[var(--error)]/30 bg-[var(--error)]/10 px-4 py-3 text-sm text-[var(--error-text)]"
        >
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit}>
        <label
          htmlFor="password"
          className="mb-2 block text-sm font-medium text-[var(--text-secondary)]"
        >
          密碼
        </label>
        <input
          id="password"
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          className="w-full rounded-lg border border-[var(--border-subtle)]/50 bg-[var(--bg-secondary)]/60 px-4 py-2.5 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
          placeholder="輸入密碼"
        />

        <button
          type="submit"
          disabled={submitting || password.length === 0}
          className="mt-6 w-full rounded-lg bg-[var(--accent-primary)] px-4 py-2.5 text-sm font-medium text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-pressed)] disabled:cursor-not-allowed disabled:opacity-50"
        >
          {submitting ? '登入中…' : '登入'}
        </button>
      </form>
    </div>
  );
}
