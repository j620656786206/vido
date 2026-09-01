// Implements: <utility — no .pen counterpart>
/**
 * 登出 control for the V0.1.1 shared-password gate.
 *
 * The gate shipped (PR #365) with a working `POST /api/v1/auth/logout` that no
 * UI ever called: once you were in, the only way out was clearing the cookie by
 * hand. On a shared NAS screen that is the difference between "I'm done" and
 * "the next person is me".
 *
 * ⚖️ It sits BELOW the ambient status strip, behind its own rule, at the same
 * weight as a nav destination — not beside ThemeToggle at 11px. The first draft
 * copied ThemeToggle's row verbatim, which made an irreversible session action
 * pixel-identical to an appearance preference 44px away, and left it lighter and
 * smaller than 探索/設定 in the mobile sheet. Visual weight follows CONSEQUENCE,
 * not whatever component happens to be next door; and `SidebarFooter` calls its
 * own region an "ambient status strip", which is a place for readouts, not for
 * things that fire.
 *
 * Renders NOTHING when the server has no password set (`authEnabled === false`).
 * A LAN-only install has no session to end, and a logout button that cannot log
 * you out is worse than no button.
 */
import { useState } from 'react';
import { LogOut } from 'lucide-react';
import { useNavigate } from '@tanstack/react-router';
import { useQueryClient } from '@tanstack/react-query';
import { authService } from '../../services/authService';
import { useAuthStatus } from '../../hooks/useAuthStatus';
import { Tooltip } from '../ui/Tooltip';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/Dialog';
import { cn } from '../../lib/utils';

interface LogoutButtonProps {
  /** 'rail' = icon only (collapsed sidebar); 'row' = icon + label. */
  variant?: 'rail' | 'row';
  className?: string;
}

export function LogoutButton({ variant = 'rail', className }: LogoutButtonProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { data: authStatus } = useAuthStatus();
  const [confirming, setConfirming] = useState(false);
  const [pending, setPending] = useState(false);

  // No password configured → no session to end.
  if (authStatus?.authEnabled !== true) return null;

  async function handleLogout() {
    setPending(true);
    try {
      await authService.logout();
    } catch {
      // The cookie clear is server-side and best-effort; even if the request
      // failed (offline, API restarting) we still drop local state and send the
      // user to /login, where a stale cookie is re-checked against the server.
    }
    // Wipe every cached response, not just the auth key: the library, settings
    // and download queues in the cache belong to the session that just ended.
    queryClient.clear();
    setConfirming(false);
    // `loggedOut` makes the gate acknowledge the action — otherwise the last
    // frame of "I logged out" is identical to the one a stranger gets, and
    // nothing on screen confirms that anything happened.
    navigate({ to: '/login', search: { loggedOut: true } });
  }

  const label = '登出';

  const trigger =
    variant === 'row' ? (
      <button
        type="button"
        onClick={() => setConfirming(true)}
        disabled={pending}
        data-testid="logout-button"
        aria-label={label}
        className={cn(
          'flex min-h-[44px] w-full items-center gap-3 rounded-[var(--radius-md)] px-2.5 py-2 text-sm font-medium text-[var(--text-secondary)] transition-colors duration-[var(--motion-touch)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] disabled:cursor-not-allowed disabled:text-[var(--text-muted)]',
          className
        )}
      >
        <LogOut className="h-5 w-5 shrink-0 text-[var(--text-muted)]" aria-hidden="true" />
        <span>{label}</span>
      </button>
    ) : (
      <Tooltip content={label}>
        <button
          type="button"
          onClick={() => setConfirming(true)}
          disabled={pending}
          data-testid="logout-button"
          aria-label={label}
          className={cn(
            'flex h-11 w-11 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-secondary)] transition-colors duration-[var(--motion-touch)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] disabled:cursor-not-allowed disabled:text-[var(--text-muted)]',
            className
          )}
        >
          <LogOut className="h-5 w-5 shrink-0" aria-hidden="true" />
        </button>
      </Tooltip>
    );

  return (
    <>
      {trigger}
      <Dialog open={confirming} onOpenChange={(open) => !pending && setConfirming(open)}>
        {/* z-[80]: this button also lives inside the mobile 更多 sheet, whose
            backdrop is z-[70] and panel z-[71] (Sheet.tsx). At DialogContent's
            default z-50 the dialog asking for a decision was rendered UNDER the
            sheet's scrim — the dimmest thing on screen. Closing the sheet first
            is NOT the fix: the sheet owns this component, so unmounting it takes
            the dialog with it before it can ever paint. */}
        <DialogContent className="z-[80] max-w-sm" data-testid="logout-confirm">
          <DialogHeader>
            <DialogTitle>要登出嗎？</DialogTitle>
            <DialogDescription>
              下次進來要再輸入一次密碼。密碼是安裝時設定的,不是這裡能改的。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <button
              type="button"
              onClick={() => setConfirming(false)}
              disabled={pending}
              data-testid="logout-cancel"
              className="min-h-[44px] rounded-[var(--radius-md)] border border-[var(--border-subtle)] px-4 text-sm font-medium text-[var(--text-primary)] transition-colors duration-[var(--motion-touch)] hover:bg-[var(--bg-tertiary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]"
            >
              取消
            </button>
            <button
              type="button"
              onClick={handleLogout}
              disabled={pending}
              data-testid="logout-confirm-button"
              className="min-h-[44px] rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-4 text-sm font-medium text-[var(--text-on-accent)] transition-colors duration-[var(--motion-touch)] hover:bg-[var(--accent-pressed)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] disabled:cursor-not-allowed disabled:bg-[var(--bg-tertiary)] disabled:text-[var(--text-muted)]"
            >
              {pending ? '登出中…' : '登出'}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
