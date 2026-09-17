import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ButtonCost } from './ButtonCost';

// Story dsr-6a AC #4 — a paid control carries its estimated amount; without an
// amount it cannot be clicked (DESIGN.md「會花錢的動作要有記號」, J9-D).

describe('ButtonCost', () => {
  it('ready: shows the label and the amount verbatim, and clicks through', async () => {
    const onClick = vi.fn();
    render(
      <ButtonCost
        label="生成字幕"
        cost={{ status: 'ready', usd: 0.42, approximate: false }}
        onClick={onClick}
        data-testid="btn"
      />
    );
    const button = screen.getByTestId('btn');
    expect(screen.getByTestId('btn-amount').textContent).toBe('$0.42');
    expect(button).toHaveAccessibleName('生成字幕 $0.42');
    expect(button).toBeEnabled();
    await userEvent.click(button);
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('ready + assumed runtime: one ≈ on the amount', () => {
    render(
      <ButtonCost
        label="生成字幕"
        cost={{ status: 'ready', usd: 0.42, approximate: true }}
        data-testid="btn"
      />
    );
    expect(screen.getByTestId('btn-amount').textContent).toBe('≈ $0.42');
  });

  it('ready at zero: $0.00, never 「免費」', () => {
    render(
      <ButtonCost
        label="生成字幕"
        cost={{ status: 'ready', usd: 0, approximate: false }}
        data-testid="btn"
      />
    );
    expect(screen.getByTestId('btn-amount').textContent).toBe('$0.00');
    expect(screen.queryByText(/免費/)).toBeNull();
  });

  it('loading: skeleton in the amount slot, not clickable, announced', async () => {
    const onClick = vi.fn();
    render(
      <ButtonCost
        label="生成字幕"
        cost={{ status: 'loading' }}
        onClick={onClick}
        data-testid="btn"
      />
    );
    const button = screen.getByTestId('btn');
    expect(button).toHaveAttribute('aria-disabled', 'true');
    expect(screen.getByTestId('btn-amount-skeleton')).toBeInTheDocument();
    expect(screen.queryByTestId('btn-amount')).toBeNull();
    expect(button).toHaveTextContent('正在估算費用');
    await userEvent.click(button);
    expect(onClick).not.toHaveBeenCalled();
  });

  it('unavailable: disabled and shows no amount at all', async () => {
    const onClick = vi.fn();
    render(
      <ButtonCost
        label="生成字幕"
        cost={{ status: 'unavailable' }}
        onClick={onClick}
        data-testid="btn"
      />
    );
    const button = screen.getByTestId('btn');
    expect(button).toBeDisabled();
    expect(button.textContent).toBe('生成字幕');
    expect(button.textContent).not.toMatch(/\$/);
    await userEvent.click(button);
    expect(onClick).not.toHaveBeenCalled();
  });

  it('busy: keeps its content (no spinner, no width jump) and cannot be clicked twice', async () => {
    const onClick = vi.fn();
    render(
      <ButtonCost
        label="生成字幕"
        cost={{ status: 'ready', usd: 0.42, approximate: false }}
        busy
        onClick={onClick}
        data-testid="btn"
      />
    );
    const button = screen.getByTestId('btn');
    expect(button).toBeDisabled();
    expect(button).toHaveAttribute('aria-busy', 'true');
    expect(screen.getByTestId('btn-amount').textContent).toBe('$0.42');
    expect(button.querySelector('.animate-spin')).toBeNull();
    await userEvent.click(button);
    expect(onClick).not.toHaveBeenCalled();
  });

  it('keeps the same box in every state: skeleton measured in the amount font, border always present', () => {
    const { rerender } = render(
      <ButtonCost label="生成字幕" cost={{ status: 'loading' }} data-testid="btn" />
    );
    expect(screen.getByTestId('btn-amount-skeleton').className).toContain('font-mono');
    expect(screen.getByTestId('btn').className).toMatch(/(^|\s)border(\s|$)/);

    rerender(
      <ButtonCost
        label="生成字幕"
        cost={{ status: 'ready', usd: 0.42, approximate: false }}
        data-testid="btn"
      />
    );
    expect(screen.getByTestId('btn').className).toMatch(/(^|\s)border(\s|$)/);
    expect(screen.getByTestId('btn').className).toContain('border-transparent');
  });

  it('passes aria-describedby through so the reason line is announced', () => {
    render(
      <>
        <ButtonCost
          label="生成字幕"
          cost={{ status: 'unavailable' }}
          aria-describedby="why"
          data-testid="btn"
        />
        <p id="why">暫時算不出費用</p>
      </>
    );
    expect(screen.getByTestId('btn')).toHaveAccessibleDescription('暫時算不出費用');
  });

  it('the amount carries no colour of its own (money is a fact, not a state)', () => {
    render(
      <ButtonCost
        label="生成字幕"
        cost={{ status: 'ready', usd: 0.42, approximate: false }}
        data-testid="btn"
      />
    );
    expect(screen.getByTestId('btn-amount').className).not.toMatch(/text-\[var/);
  });
});
