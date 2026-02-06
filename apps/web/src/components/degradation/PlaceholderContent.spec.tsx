import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import {
  PlaceholderContent,
  PlaceholderPoster,
  DegradationMessage,
} from './PlaceholderContent';

describe('PlaceholderContent', () => {
  it('renders title placeholder', () => {
    render(<PlaceholderContent field="title" />);
    expect(screen.getByText('未知標題')).toBeInTheDocument();
  });

  it('renders overview placeholder', () => {
    render(<PlaceholderContent field="overview" />);
    expect(screen.getByText('暫無簡介')).toBeInTheDocument();
  });

  it('renders year placeholder', () => {
    render(<PlaceholderContent field="year" />);
    expect(screen.getByText('—')).toBeInTheDocument();
  });

  it('applies custom className', () => {
    const { container } = render(
      <PlaceholderContent field="title" className="custom-class" />
    );
    expect(container.firstChild).toHaveClass('custom-class');
  });

  it('has title attribute explaining the placeholder', () => {
    render(<PlaceholderContent field="title" />);
    expect(screen.getByText('未知標題')).toHaveAttribute(
      'title',
      '標題暫時無法取得'
    );
  });
});

describe('PlaceholderPoster', () => {
  it('renders placeholder poster', () => {
    render(<PlaceholderPoster />);
    expect(screen.getByRole('img')).toBeInTheDocument();
  });

  it('has accessible label', () => {
    render(<PlaceholderPoster />);
    expect(screen.getByRole('img')).toHaveAttribute(
      'aria-label',
      '海報無法載入'
    );
  });

  it('applies size classes', () => {
    const { rerender } = render(<PlaceholderPoster size="sm" />);
    expect(screen.getByRole('img')).toHaveClass('h-24', 'w-16');

    rerender(<PlaceholderPoster size="lg" />);
    expect(screen.getByRole('img')).toHaveClass('h-72', 'w-48');
  });

  it('applies custom className', () => {
    render(<PlaceholderPoster className="custom-class" />);
    expect(screen.getByRole('img')).toHaveClass('custom-class');
  });
});

describe('DegradationMessage', () => {
  it('renders message', () => {
    render(<DegradationMessage message="測試訊息" />);
    expect(screen.getByText('測試訊息')).toBeInTheDocument();
  });

  it('renders missing fields', () => {
    render(
      <DegradationMessage
        message="部分資料遺失"
        missingFields={['title', 'overview']}
      />
    );
    expect(screen.getByText('部分資料遺失')).toBeInTheDocument();
    expect(screen.getByText(/無法取得：/)).toBeInTheDocument();
    expect(screen.getByText(/標題/)).toBeInTheDocument();
    expect(screen.getByText(/簡介/)).toBeInTheDocument();
  });

  it('applies custom className', () => {
    render(
      <DegradationMessage message="test" className="custom-class" />
    );
    expect(screen.getByRole('status')).toHaveClass('custom-class');
  });

  it('does not show missing fields section when empty', () => {
    render(<DegradationMessage message="test" missingFields={[]} />);
    expect(screen.queryByText(/無法取得/)).not.toBeInTheDocument();
  });
});
