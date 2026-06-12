import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DoubanSection } from './DoubanSection';
import type { DoubanReviewSummary } from '../../types/library';

function summary(overrides: Partial<DoubanReviewSummary> = {}): DoubanReviewSummary {
  return {
    id: '1292052',
    totalComments: 152340,
    topComments: [
      { author: '影評人甲', rating: 5, text: '這部電影太棒了' },
      { author: '觀眾乙', rating: 4, text: '敘事流暢' },
    ],
    ...overrides,
  };
}

describe('DoubanSection', () => {
  it('renders the direct Douban link when doubanId is present (AC #1)', () => {
    render(<DoubanSection doubanId="1292052" />);

    const link = screen.getByTestId('douban-page-link');
    expect(link).toBeInTheDocument();
    expect(link).toHaveAttribute('href', 'https://movie.douban.com/subject/1292052/');
    expect(link).toHaveAttribute('target', '_blank');
    expect(link).toHaveAttribute('rel', 'noopener noreferrer');
    expect(link).toHaveTextContent('查看豆瓣頁面');
  });

  it('omits the entire section when no doubanId is known (AC #4)', () => {
    const { container } = render(<DoubanSection doubanId={undefined} summary={summary()} />);

    expect(container).toBeEmptyDOMElement();
    expect(screen.queryByTestId('douban-section')).not.toBeInTheDocument();
  });

  it('renders comments with Traditional text, author, stars and total count (AC #2/#3)', () => {
    render(<DoubanSection doubanId="1292052" summary={summary()} />);

    expect(screen.getByText('影評人甲')).toBeInTheDocument();
    expect(screen.getByText('這部電影太棒了')).toBeInTheDocument();
    expect(screen.getByText('觀眾乙')).toBeInTheDocument();
    expect(screen.getByTestId('douban-reviews-count')).toHaveTextContent('152,340');
    expect(screen.getAllByTestId('douban-comment')).toHaveLength(2);
    expect(screen.getAllByTestId('douban-comment-rating')[0]).toHaveAttribute('aria-label', '5 星');
  });

  it('keeps the direct link but drops the review block on error (AC #5)', () => {
    render(<DoubanSection doubanId="1292052" summary={summary()} isError />);

    expect(screen.getByTestId('douban-page-link')).toBeInTheDocument();
    expect(screen.queryByTestId('douban-reviews')).not.toBeInTheDocument();
  });

  it('keeps the direct link with no review block when the summary is empty (AC #5)', () => {
    render(
      <DoubanSection
        doubanId="1292052"
        summary={{ id: '1292052', totalComments: 0, topComments: [] }}
      />
    );

    expect(screen.getByTestId('douban-page-link')).toBeInTheDocument();
    expect(screen.queryByTestId('douban-reviews')).not.toBeInTheDocument();
  });

  it('shows a quiet skeleton while loading and still renders the link', () => {
    render(<DoubanSection doubanId="1292052" isLoading />);

    expect(screen.getByTestId('douban-page-link')).toBeInTheDocument();
    expect(screen.getByTestId('douban-reviews-skeleton')).toBeInTheDocument();
  });

  it('caps the rendered comments at five', () => {
    const many = Array.from({ length: 8 }, (_, i) => ({
      author: `使用者${i}`,
      rating: 3,
      text: `短評${i}`,
    }));
    render(<DoubanSection doubanId="1292052" summary={summary({ topComments: many })} />);

    expect(screen.getAllByTestId('douban-comment')).toHaveLength(5);
  });

  it('omits the star glyphs for an unrated comment', () => {
    render(
      <DoubanSection
        doubanId="1292052"
        summary={summary({ topComments: [{ author: '無評分', rating: 0, text: '只留言' }] })}
      />
    );

    expect(screen.getByText('無評分')).toBeInTheDocument();
    expect(screen.queryByTestId('douban-comment-rating')).not.toBeInTheDocument();
  });
});
