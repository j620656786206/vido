import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ParseFailedActions } from './ParseFailedActions';

describe('ParseFailedActions', () => {
  it('renders retry button and manual search button when onManualSearch is provided', () => {
    render(<ParseFailedActions torrentHash="abc123" onRetry={vi.fn()} onManualSearch={vi.fn()} />);
    expect(screen.getByRole('button', { name: /重試/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /手動搜尋/i })).toBeInTheDocument();
  });

  it('does not render manual search button when onManualSearch is not provided', () => {
    render(<ParseFailedActions torrentHash="abc123" onRetry={vi.fn()} />);
    expect(screen.getByRole('button', { name: /重試/i })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /手動搜尋/i })).not.toBeInTheDocument();
  });

  it('calls onRetry when retry button is clicked', async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();
    render(<ParseFailedActions torrentHash="abc123" onRetry={onRetry} />);

    await user.click(screen.getByRole('button', { name: /重試/i }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('calls onManualSearch when manual search button is clicked', async () => {
    const user = userEvent.setup();
    const onManualSearch = vi.fn();
    render(
      <ParseFailedActions torrentHash="abc123" onRetry={vi.fn()} onManualSearch={onManualSearch} />
    );

    await user.click(screen.getByRole('button', { name: /手動搜尋/i }));
    expect(onManualSearch).toHaveBeenCalledWith('abc123');
  });

  it('shows error message when provided', () => {
    render(
      <ParseFailedActions
        torrentHash="abc123"
        errorMessage="could not parse filename"
        onRetry={vi.fn()}
      />
    );
    expect(screen.getByText(/could not parse filename/i)).toBeInTheDocument();
  });

  it('disables retry button when retrying', () => {
    render(<ParseFailedActions torrentHash="abc123" onRetry={vi.fn()} isRetrying={true} />);
    const retryBtn = screen.getByRole('button', { name: /重試/i });
    expect(retryBtn).toBeDisabled();
  });

  it('has proper data-testid attributes', () => {
    render(<ParseFailedActions torrentHash="abc123" onRetry={vi.fn()} />);
    expect(screen.getByTestId('parse-failed-actions')).toBeInTheDocument();
  });
});
