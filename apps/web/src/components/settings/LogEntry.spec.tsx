import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect } from 'vitest';
import { LogEntry } from './LogEntry';
import type { SystemLog } from '../../services/logService';

const makeLog = (overrides: Partial<SystemLog> = {}): SystemLog => ({
  id: 1,
  level: 'INFO',
  message: 'Test log message',
  createdAt: '2026-03-18T10:30:00Z',
  ...overrides,
});

describe('LogEntry', () => {
  it('renders level badge, message, and timestamp', () => {
    render(<LogEntry log={makeLog()} />);

    expect(screen.getByTestId('log-level')).toHaveTextContent('INFO');
    expect(screen.getByTestId('log-message')).toHaveTextContent('Test log message');
    expect(screen.getByTestId('log-timestamp')).toBeInTheDocument();
  });

  it('renders source when present', () => {
    render(<LogEntry log={makeLog({ source: 'tmdb' })} />);
    expect(screen.getByTestId('log-source')).toHaveTextContent('[tmdb]');
  });

  it('does not render source when absent', () => {
    render(<LogEntry log={makeLog({ source: undefined })} />);
    expect(screen.queryByTestId('log-source')).not.toBeInTheDocument();
  });

  it('renders color-coded badge for ERROR level', () => {
    render(<LogEntry log={makeLog({ level: 'ERROR' })} />);
    const badge = screen.getByTestId('log-level');
    expect(badge).toHaveTextContent('ERROR');
    expect(badge.className).toContain('text-[var(--error-text)]');
  });

  it('renders color-coded badge for WARN level', () => {
    render(<LogEntry log={makeLog({ level: 'WARN' })} />);
    const badge = screen.getByTestId('log-level');
    expect(badge.className).toContain('text-[var(--warning-text)]');
  });

  it('renders color-coded badge for DEBUG level', () => {
    render(<LogEntry log={makeLog({ level: 'DEBUG' })} />);
    const badge = screen.getByTestId('log-level');
    // dsr-3e: C12 draws DEBUG as text-muted on bg-tertiary (was secondary on muted/10).
    expect(badge.className).toContain('text-[var(--text-muted)]');
    expect(badge.className).toContain('bg-[var(--bg-tertiary)]');
  });

  it('disables expand button when no context or hint', () => {
    render(<LogEntry log={makeLog()} />);
    expect(screen.getByTestId('log-expand-btn')).toBeDisabled();
  });

  it('enables expand button when context exists', () => {
    render(<LogEntry log={makeLog({ context: { key: 'value' } })} />);
    expect(screen.getByTestId('log-expand-btn')).toBeEnabled();
  });

  it('shows context JSON when expanded', async () => {
    const user = userEvent.setup();
    render(<LogEntry log={makeLog({ context: { error_code: 'TMDB_TIMEOUT' } })} />);

    await user.click(screen.getByTestId('log-expand-btn'));

    expect(screen.getByTestId('log-details')).toBeInTheDocument();
    expect(screen.getByTestId('log-context')).toHaveTextContent('TMDB_TIMEOUT');
  });

  it('shows hint when expanded and hint exists', async () => {
    const user = userEvent.setup();
    render(
      <LogEntry
        log={makeLog({
          level: 'ERROR',
          hint: '檢查網路連線',
          context: { code: 'TMDB_TIMEOUT' },
        })}
      />
    );

    await user.click(screen.getByTestId('log-expand-btn'));

    expect(screen.getByTestId('log-hint')).toHaveTextContent('檢查網路連線');
  });

  it('collapses details on second click', async () => {
    const user = userEvent.setup();
    render(<LogEntry log={makeLog({ context: { key: 'value' } })} />);

    await user.click(screen.getByTestId('log-expand-btn'));
    expect(screen.getByTestId('log-details')).toBeInTheDocument();

    await user.click(screen.getByTestId('log-expand-btn'));
    expect(screen.queryByTestId('log-details')).not.toBeInTheDocument();
  });

  it('does not show hint section when hint is absent', async () => {
    const user = userEvent.setup();
    render(<LogEntry log={makeLog({ context: { key: 'value' } })} />);

    await user.click(screen.getByTestId('log-expand-btn'));

    expect(screen.queryByTestId('log-hint')).not.toBeInTheDocument();
    expect(screen.getByTestId('log-context')).toBeInTheDocument();
  });

  describe('dsr-3e', () => {
    it('orders badge → time → message → source', () => {
      const { container } = render(<LogEntry log={makeLog({ source: 'scanner' })} />);
      const ids = [...container.querySelectorAll('[data-testid]')]
        .map((el) => el.getAttribute('data-testid'))
        .filter((id) => ['log-level', 'log-timestamp', 'log-message', 'log-source'].includes(id!));
      expect(ids).toEqual(['log-level', 'log-timestamp', 'log-message', 'log-source']);
    });

    it('prints the time in the LOCAL zone as YYYY-MM-DD HH:mm:ss inside <time dateTime>', () => {
      const iso = '2026-09-11T01:42:18Z';
      render(<LogEntry log={makeLog({ createdAt: iso })} />);
      const full = screen.getByTestId('log-timestamp');
      expect(full.textContent).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/);
      const d = new Date(iso);
      expect(full.textContent!.slice(11, 13)).toBe(String(d.getHours()).padStart(2, '0'));
      const time = full.closest('time')!;
      expect(time).toHaveAttribute('dateTime', iso);
      // The phone copy is the time of day only.
      expect(time.textContent).toContain(full.textContent!.slice(11));
    });

    it('the badge is a fixed 64 in mono', () => {
      render(<LogEntry log={makeLog()} />);
      const badge = screen.getByTestId('log-level');
      expect(badge.className).toContain('w-16');
      expect(badge.className).toContain('font-mono');
    });

    it('rows have no divider, and on a phone the message takes its own full line', () => {
      render(<LogEntry log={makeLog()} />);
      expect(screen.getByTestId('log-entry').className).not.toContain('border-b');
      const msg = screen.getByTestId('log-message').className;
      expect(msg).toContain('basis-full');
      expect(msg).toContain('order-last');
      expect(msg).toContain('text-xs');
      expect(msg).toContain('sm:text-sm');
    });
  });
});
