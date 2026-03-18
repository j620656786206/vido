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
    expect(badge.className).toContain('text-red-400');
  });

  it('renders color-coded badge for WARN level', () => {
    render(<LogEntry log={makeLog({ level: 'WARN' })} />);
    const badge = screen.getByTestId('log-level');
    expect(badge.className).toContain('text-yellow-400');
  });

  it('renders color-coded badge for DEBUG level', () => {
    render(<LogEntry log={makeLog({ level: 'DEBUG' })} />);
    const badge = screen.getByTestId('log-level');
    expect(badge.className).toContain('text-gray-400');
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
});
