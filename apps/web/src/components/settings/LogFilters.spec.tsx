import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { LogFilters } from './LogFilters';

describe('LogFilters', () => {
  it('renders all level filter chips', () => {
    render(<LogFilters level="" keyword="" onLevelChange={vi.fn()} onKeywordChange={vi.fn()} />);

    expect(screen.getByTestId('log-filter-all')).toBeInTheDocument();
    expect(screen.getByTestId('log-filter-error')).toBeInTheDocument();
    expect(screen.getByTestId('log-filter-warn')).toBeInTheDocument();
    expect(screen.getByTestId('log-filter-info')).toBeInTheDocument();
    expect(screen.getByTestId('log-filter-debug')).toBeInTheDocument();
  });

  it('calls onLevelChange when chip is clicked', async () => {
    const user = userEvent.setup();
    const onLevelChange = vi.fn();

    render(
      <LogFilters level="" keyword="" onLevelChange={onLevelChange} onKeywordChange={vi.fn()} />
    );

    await user.click(screen.getByTestId('log-filter-error'));
    expect(onLevelChange).toHaveBeenCalledWith('ERROR');
  });

  it('toggles level off when same chip clicked', async () => {
    const user = userEvent.setup();
    const onLevelChange = vi.fn();

    render(
      <LogFilters
        level="ERROR"
        keyword=""
        onLevelChange={onLevelChange}
        onKeywordChange={vi.fn()}
      />
    );

    await user.click(screen.getByTestId('log-filter-error'));
    expect(onLevelChange).toHaveBeenCalledWith('');
  });

  it('calls onKeywordChange on Enter', async () => {
    const user = userEvent.setup();
    const onKeywordChange = vi.fn();

    render(
      <LogFilters level="" keyword="" onLevelChange={vi.fn()} onKeywordChange={onKeywordChange} />
    );

    const input = screen.getByTestId('log-keyword-input');
    await user.type(input, 'test{Enter}');
    expect(onKeywordChange).toHaveBeenCalledWith('test');
  });

  it('clears keyword when X is clicked', async () => {
    const user = userEvent.setup();
    const onKeywordChange = vi.fn();

    render(
      <LogFilters
        level=""
        keyword="existing"
        onLevelChange={vi.fn()}
        onKeywordChange={onKeywordChange}
      />
    );

    // The clear button should be visible since keyword is pre-filled
    // Type to populate the internal state
    const input = screen.getByTestId('log-keyword-input');
    await user.clear(input);
    await user.type(input, 'something');

    const clearBtn = screen.getByTestId('log-keyword-clear');
    await user.click(clearBtn);

    expect(onKeywordChange).toHaveBeenCalledWith('');
  });

  it('renders keyword search input with placeholder', () => {
    render(<LogFilters level="" keyword="" onLevelChange={vi.fn()} onKeywordChange={vi.fn()} />);

    expect(screen.getByPlaceholderText('搜尋關鍵字...')).toBeInTheDocument();
  });
});
