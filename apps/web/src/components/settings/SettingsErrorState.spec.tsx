import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { SettingsErrorState } from './SettingsErrorState';

const TITLE = '無法載入服務狀態';
const DESCRIPTION = '與後端的連線中斷了。這不影響已在執行的背景工作。';

describe('SettingsErrorState', () => {
  it('announces itself as an alert with the title and description verbatim', () => {
    render(<SettingsErrorState title={TITLE} description={DESCRIPTION} onRetry={() => {}} />);
    const alert = screen.getByRole('alert');
    expect(alert).toHaveTextContent(TITLE);
    expect(alert).toHaveTextContent(DESCRIPTION);
  });

  it('draws the C16-D recipe: error-tint circle, 18→20 title, secondary description', () => {
    render(<SettingsErrorState title={TITLE} description={DESCRIPTION} onRetry={() => {}} />);
    const icon = screen.getByTestId('settings-error-state-icon');
    expect(icon.className).toContain('bg-[var(--error-tint)]');
    expect(icon.className).toContain('size-16');
    const title = screen.getByText(TITLE);
    expect(title.className).toContain('text-lg');
    expect(title.className).toContain('sm:text-xl');
    expect(title.className).toContain('font-bold');
    expect(screen.getByText(DESCRIPTION).className).toContain('text-[var(--text-secondary)]');
  });

  it('retries once per press', async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();
    render(<SettingsErrorState title={TITLE} description={DESCRIPTION} onRetry={onRetry} />);
    const button = screen.getByRole('button', { name: '重試' });
    expect(button.className).toContain('min-h-11');
    await user.click(button);
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('disables the button and says so while a retry is in flight', async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();
    render(
      <SettingsErrorState title={TITLE} description={DESCRIPTION} onRetry={onRetry} isRetrying />
    );
    const button = screen.getByRole('button', { name: '重試中…' });
    // aria-disabled, not disabled: the button keeps keyboard focus.
    expect(button).toHaveAttribute('aria-disabled', 'true');
    expect(button).not.toBeDisabled();
    button.focus();
    await user.keyboard('{Enter}');
    await user.click(button);
    expect(onRetry).not.toHaveBeenCalled();
    expect(button).toHaveFocus();
  });

  // The component exists so that a raw backend error never reaches the page.
  // It has no prop that could carry one.
  it('has no way to receive a raw error', () => {
    render(
      <SettingsErrorState
        title={TITLE}
        description={DESCRIPTION}
        onRetry={() => {}}
        // @ts-expect-error — `error` is deliberately not a prop
        error={new Error('Failed to fetch')}
      />
    );
    expect(screen.queryByText(/Failed to fetch/)).toBeNull();
  });

  it('accepts a test id for the page that hosts it', () => {
    render(
      <SettingsErrorState
        title={TITLE}
        description={DESCRIPTION}
        onRetry={() => {}}
        testId="service-status-error"
      />
    );
    expect(screen.getByTestId('service-status-error')).toContainElement(screen.getByRole('alert'));
  });

  // The live region is the message only; the button outside it means the
  // 重試 → 重試中… label change is not re-announced as a new alert.
  it('keeps the retry button outside the alert region', () => {
    render(<SettingsErrorState title={TITLE} description={DESCRIPTION} onRetry={() => {}} />);
    const alert = screen.getByRole('alert');
    expect(alert).not.toContainElement(screen.getByRole('button', { name: '重試' }));
  });
});
