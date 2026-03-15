import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { LibrarySearchBar } from './LibrarySearchBar';

describe('LibrarySearchBar', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('should render with default placeholder', () => {
    render(<LibrarySearchBar onSearch={vi.fn()} />);
    expect(screen.getByPlaceholderText('搜尋媒體標題...')).toBeInTheDocument();
  });

  it('should display initial query value', () => {
    render(<LibrarySearchBar onSearch={vi.fn()} initialQuery="鬼滅之刃" />);
    const input = screen.getByRole('textbox') as HTMLInputElement;
    expect(input.value).toBe('鬼滅之刃');
  });

  it('should debounce search with 500ms delay', () => {
    const onSearch = vi.fn();
    render(<LibrarySearchBar onSearch={onSearch} />);

    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: '駭客任務' } });

    // Should not fire before 500ms
    vi.advanceTimersByTime(400);
    expect(onSearch).not.toHaveBeenCalled();

    // Should fire at 500ms
    vi.advanceTimersByTime(100);
    expect(onSearch).toHaveBeenCalledWith('駭客任務');
    expect(onSearch).toHaveBeenCalledTimes(1);
  });

  it('should not trigger search for single character input', () => {
    const onSearch = vi.fn();
    render(<LibrarySearchBar onSearch={onSearch} />);

    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'a' } });

    vi.advanceTimersByTime(500);
    expect(onSearch).not.toHaveBeenCalled();
  });

  it('should trigger search for 2+ character input', () => {
    const onSearch = vi.fn();
    render(<LibrarySearchBar onSearch={onSearch} />);

    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'ab' } });

    vi.advanceTimersByTime(500);
    expect(onSearch).toHaveBeenCalledWith('ab');
  });

  it('should show clear button when input has value', () => {
    render(<LibrarySearchBar onSearch={vi.fn()} initialQuery="test" />);
    expect(screen.getByLabelText('清除搜尋')).toBeInTheDocument();
  });

  it('should hide clear button when input is empty', () => {
    render(<LibrarySearchBar onSearch={vi.fn()} />);
    expect(screen.queryByLabelText('清除搜尋')).not.toBeInTheDocument();
  });

  it('should clear input and call onSearch with empty string on clear click', () => {
    const onSearch = vi.fn();
    render(<LibrarySearchBar onSearch={onSearch} initialQuery="test" />);

    fireEvent.click(screen.getByLabelText('清除搜尋'));

    const input = screen.getByRole('textbox') as HTMLInputElement;
    expect(input.value).toBe('');
    expect(onSearch).toHaveBeenCalledWith('');
  });

  it('should clear on Escape key', () => {
    const onSearch = vi.fn();
    render(<LibrarySearchBar onSearch={onSearch} initialQuery="test" />);

    const input = screen.getByRole('textbox');
    fireEvent.keyDown(input, { key: 'Escape' });

    expect((input as HTMLInputElement).value).toBe('');
    expect(onSearch).toHaveBeenCalledWith('');
  });

  it('should show result count when query ≥ 2 chars and resultCount provided', () => {
    render(<LibrarySearchBar onSearch={vi.fn()} initialQuery="test" resultCount={15} />);
    expect(screen.getByTestId('search-result-count')).toHaveTextContent('找到 15 個結果');
  });

  it('should not show result count when query is short', () => {
    render(<LibrarySearchBar onSearch={vi.fn()} initialQuery="t" resultCount={0} />);
    expect(screen.queryByTestId('search-result-count')).not.toBeInTheDocument();
  });

  it('should not show result count when resultCount is undefined', () => {
    render(<LibrarySearchBar onSearch={vi.fn()} initialQuery="test" />);
    expect(screen.queryByTestId('search-result-count')).not.toBeInTheDocument();
  });

  it('should focus input on Ctrl+K', () => {
    render(<LibrarySearchBar onSearch={vi.fn()} />);

    const input = screen.getByRole('textbox');
    fireEvent.keyDown(document, { key: 'k', ctrlKey: true });

    expect(document.activeElement).toBe(input);
  });

  it('should focus input on Cmd+K (macOS)', () => {
    render(<LibrarySearchBar onSearch={vi.fn()} />);

    const input = screen.getByRole('textbox');
    fireEvent.keyDown(document, { key: 'k', metaKey: true });

    expect(document.activeElement).toBe(input);
  });

  it('should have accessible label', () => {
    render(<LibrarySearchBar onSearch={vi.fn()} />);
    expect(screen.getByLabelText('搜尋媒體標題')).toBeInTheDocument();
  });

  it('should support Chinese input', () => {
    const onSearch = vi.fn();
    render(<LibrarySearchBar onSearch={onSearch} />);

    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: '進擊的巨人' } });

    vi.advanceTimersByTime(500);
    expect(onSearch).toHaveBeenCalledWith('進擊的巨人');
  });

  it('should support English input', () => {
    const onSearch = vi.fn();
    render(<LibrarySearchBar onSearch={onSearch} />);

    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'avatar' } });

    vi.advanceTimersByTime(500);
    expect(onSearch).toHaveBeenCalledWith('avatar');
  });

  it('should restart debounce on rapid typing', () => {
    const onSearch = vi.fn();
    render(<LibrarySearchBar onSearch={onSearch} />);

    const input = screen.getByRole('textbox');

    // Type first value
    fireEvent.change(input, { target: { value: 'av' } });
    vi.advanceTimersByTime(300);

    // Type again before debounce fires — should restart timer
    fireEvent.change(input, { target: { value: 'avatar' } });
    vi.advanceTimersByTime(300);

    // First debounce window passed (600ms total) but second hasn't (only 300ms)
    expect(onSearch).not.toHaveBeenCalled();

    // Complete the second debounce
    vi.advanceTimersByTime(200);
    expect(onSearch).toHaveBeenCalledTimes(1);
    expect(onSearch).toHaveBeenCalledWith('avatar');
  });

  it('should show result count of 0 when provided', () => {
    render(<LibrarySearchBar onSearch={vi.fn()} initialQuery="nonexistent" resultCount={0} />);
    expect(screen.getByTestId('search-result-count')).toHaveTextContent('找到 0 個結果');
  });

  it('should show result count of 1', () => {
    render(<LibrarySearchBar onSearch={vi.fn()} initialQuery="unique" resultCount={1} />);
    expect(screen.getByTestId('search-result-count')).toHaveTextContent('找到 1 個結果');
  });

  it('should call onSearch with empty string immediately on clear (no debounce)', () => {
    const onSearch = vi.fn();
    render(<LibrarySearchBar onSearch={onSearch} initialQuery="test" />);

    fireEvent.click(screen.getByLabelText('清除搜尋'));

    // Should fire immediately without waiting for debounce
    expect(onSearch).toHaveBeenCalledWith('');
  });
});
