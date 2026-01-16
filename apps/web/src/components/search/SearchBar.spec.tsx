import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { SearchBar } from './SearchBar';

describe('SearchBar', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('should render with placeholder text', () => {
    const onSearch = vi.fn();
    render(<SearchBar onSearch={onSearch} />);

    expect(screen.getByPlaceholderText('搜尋電影或影集...')).toBeInTheDocument();
  });

  it('should render with custom placeholder', () => {
    const onSearch = vi.fn();
    render(<SearchBar onSearch={onSearch} placeholder="Search movies..." />);

    expect(screen.getByPlaceholderText('Search movies...')).toBeInTheDocument();
  });

  it('should display initial query value', () => {
    const onSearch = vi.fn();
    render(<SearchBar onSearch={onSearch} initialQuery="鬼滅之刃" />);

    const input = screen.getByRole('textbox') as HTMLInputElement;
    expect(input.value).toBe('鬼滅之刃');
  });

  it('should call onSearch after debounce when query >= 2 chars', async () => {
    const onSearch = vi.fn();
    render(<SearchBar onSearch={onSearch} />);

    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: '鬼滅' } });

    // onSearch should not be called immediately
    expect(onSearch).not.toHaveBeenCalled();

    // Fast-forward timers by 300ms (debounce time)
    vi.advanceTimersByTime(300);

    expect(onSearch).toHaveBeenCalledWith('鬼滅');
    expect(onSearch).toHaveBeenCalledTimes(1);
  });

  it('should not call onSearch when query is 1 character', async () => {
    const onSearch = vi.fn();
    render(<SearchBar onSearch={onSearch} />);

    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: '鬼' } });

    vi.advanceTimersByTime(300);

    expect(onSearch).not.toHaveBeenCalled();
  });

  it('should call onSearch with empty string when cleared', async () => {
    const onSearch = vi.fn();
    render(<SearchBar onSearch={onSearch} initialQuery="test" />);

    // Clear button should be visible
    const clearButton = screen.getByLabelText('清除搜尋');
    expect(clearButton).toBeInTheDocument();

    fireEvent.click(clearButton);

    expect(onSearch).toHaveBeenCalledWith('');
  });

  it('should hide clear button when input is empty', () => {
    const onSearch = vi.fn();
    render(<SearchBar onSearch={onSearch} />);

    expect(screen.queryByLabelText('清除搜尋')).not.toBeInTheDocument();
  });

  it('should show clear button when input has value', () => {
    const onSearch = vi.fn();
    render(<SearchBar onSearch={onSearch} initialQuery="test" />);

    expect(screen.getByLabelText('清除搜尋')).toBeInTheDocument();
  });

  it('should clear input on Escape key', () => {
    const onSearch = vi.fn();
    render(<SearchBar onSearch={onSearch} initialQuery="test" />);

    const input = screen.getByRole('textbox') as HTMLInputElement;
    fireEvent.keyDown(input, { key: 'Escape' });

    expect(input.value).toBe('');
    expect(onSearch).toHaveBeenCalledWith('');
  });

  it('should have accessible label', () => {
    const onSearch = vi.fn();
    render(<SearchBar onSearch={onSearch} />);

    expect(screen.getByLabelText('搜尋')).toBeInTheDocument();
  });

  it('should support Traditional Chinese input', async () => {
    const onSearch = vi.fn();
    render(<SearchBar onSearch={onSearch} />);

    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: '進擊的巨人' } });

    vi.advanceTimersByTime(300);

    expect(onSearch).toHaveBeenCalledWith('進擊的巨人');
  });

  it('should support English input', async () => {
    const onSearch = vi.fn();
    render(<SearchBar onSearch={onSearch} />);

    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'Demon Slayer' } });

    vi.advanceTimersByTime(300);

    expect(onSearch).toHaveBeenCalledWith('Demon Slayer');
  });
});
