import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { EmptySearchResults } from './EmptySearchResults';

describe('EmptySearchResults', () => {
  it('should render the no results message', () => {
    render(<EmptySearchResults query="test" onClear={vi.fn()} />);
    expect(screen.getByText('找不到相關結果')).toBeInTheDocument();
  });

  it('should display the search query in context message', () => {
    render(<EmptySearchResults query="駭客任務" onClear={vi.fn()} />);
    expect(screen.getByText(/搜尋「駭客任務」沒有找到匹配的電影或影集/)).toBeInTheDocument();
  });

  it('should show suggestion bullets', () => {
    render(<EmptySearchResults query="test" onClear={vi.fn()} />);
    expect(screen.getByText(/試試不同的關鍵字/)).toBeInTheDocument();
    expect(screen.getByText(/嘗試使用繁體中文或英文搜尋/)).toBeInTheDocument();
    expect(screen.getByText(/檢查拼寫是否正確/)).toBeInTheDocument();
  });

  it('should show clear search button', () => {
    render(<EmptySearchResults query="test" onClear={vi.fn()} />);
    expect(screen.getByText('清除搜尋')).toBeInTheDocument();
  });

  it('should call onClear when clear button is clicked', () => {
    const onClear = vi.fn();
    render(<EmptySearchResults query="test" onClear={onClear} />);

    fireEvent.click(screen.getByText('清除搜尋'));
    expect(onClear).toHaveBeenCalledTimes(1);
  });

  it('should have the data-testid attribute', () => {
    render(<EmptySearchResults query="test" onClear={vi.fn()} />);
    expect(screen.getByTestId('empty-search-results')).toBeInTheDocument();
  });
});
