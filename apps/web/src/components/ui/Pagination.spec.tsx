import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { Pagination } from './Pagination';

describe('Pagination', () => {
  it('should not render when totalPages is 1', () => {
    const onPageChange = vi.fn();
    const { container } = render(
      <Pagination currentPage={1} totalPages={1} onPageChange={onPageChange} />
    );

    expect(container.firstChild).toBeNull();
  });

  it('should not render when totalPages is 0', () => {
    const onPageChange = vi.fn();
    const { container } = render(
      <Pagination currentPage={1} totalPages={0} onPageChange={onPageChange} />
    );

    expect(container.firstChild).toBeNull();
  });

  it('should render all pages when totalPages <= 7', () => {
    const onPageChange = vi.fn();
    render(
      <Pagination currentPage={1} totalPages={5} onPageChange={onPageChange} />
    );

    for (let i = 1; i <= 5; i++) {
      expect(screen.getByLabelText(`第 ${i} 頁`)).toBeInTheDocument();
    }
  });

  it('should highlight current page', () => {
    const onPageChange = vi.fn();
    render(
      <Pagination currentPage={3} totalPages={5} onPageChange={onPageChange} />
    );

    const currentPageButton = screen.getByLabelText('第 3 頁');
    expect(currentPageButton).toHaveAttribute('aria-current', 'page');
    expect(currentPageButton).toHaveClass('bg-blue-600');
  });

  it('should call onPageChange when clicking page number', () => {
    const onPageChange = vi.fn();
    render(
      <Pagination currentPage={1} totalPages={5} onPageChange={onPageChange} />
    );

    fireEvent.click(screen.getByLabelText('第 3 頁'));
    expect(onPageChange).toHaveBeenCalledWith(3);
  });

  it('should disable previous button on first page', () => {
    const onPageChange = vi.fn();
    render(
      <Pagination currentPage={1} totalPages={5} onPageChange={onPageChange} />
    );

    const prevButton = screen.getByLabelText('上一頁');
    expect(prevButton).toBeDisabled();
  });

  it('should disable next button on last page', () => {
    const onPageChange = vi.fn();
    render(
      <Pagination currentPage={5} totalPages={5} onPageChange={onPageChange} />
    );

    const nextButton = screen.getByLabelText('下一頁');
    expect(nextButton).toBeDisabled();
  });

  it('should call onPageChange with previous page when clicking previous', () => {
    const onPageChange = vi.fn();
    render(
      <Pagination currentPage={3} totalPages={5} onPageChange={onPageChange} />
    );

    fireEvent.click(screen.getByLabelText('上一頁'));
    expect(onPageChange).toHaveBeenCalledWith(2);
  });

  it('should call onPageChange with next page when clicking next', () => {
    const onPageChange = vi.fn();
    render(
      <Pagination currentPage={3} totalPages={5} onPageChange={onPageChange} />
    );

    fireEvent.click(screen.getByLabelText('下一頁'));
    expect(onPageChange).toHaveBeenCalledWith(4);
  });

  it('should show ellipsis for large page counts', () => {
    const onPageChange = vi.fn();
    render(
      <Pagination currentPage={5} totalPages={10} onPageChange={onPageChange} />
    );

    // Should show: 1 ... 4 5 6 ... 10
    expect(screen.getByLabelText('第 1 頁')).toBeInTheDocument();
    expect(screen.getByLabelText('第 4 頁')).toBeInTheDocument();
    expect(screen.getByLabelText('第 5 頁')).toBeInTheDocument();
    expect(screen.getByLabelText('第 6 頁')).toBeInTheDocument();
    expect(screen.getByLabelText('第 10 頁')).toBeInTheDocument();
    expect(screen.getAllByText('...').length).toBe(2);
  });

  it('should have accessible navigation label', () => {
    const onPageChange = vi.fn();
    render(
      <Pagination currentPage={1} totalPages={5} onPageChange={onPageChange} />
    );

    expect(screen.getByLabelText('分頁導航')).toBeInTheDocument();
  });
});
