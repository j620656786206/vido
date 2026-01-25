import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { HoverPreviewCard } from './HoverPreviewCard';

describe('HoverPreviewCard', () => {
  const defaultProps = {
    title: '鬼滅之刃',
    originalTitle: 'Demon Slayer',
    overview:
      '在大正時代的日本，炭治郎是一個善良的賣炭少年，與家人過著平凡而幸福的生活。某天，他下山賣炭回家後，發現家人全被鬼殺害，唯一存活的妹妹禰豆子也變成了鬼。',
    genreIds: [16, 28, 14],
  };

  describe('Original Title Display', () => {
    it('shows original title when different from title', () => {
      render(<HoverPreviewCard {...defaultProps} />);
      expect(screen.getByText('Demon Slayer')).toBeInTheDocument();
    });

    it('does not show original title when same as title', () => {
      render(<HoverPreviewCard {...defaultProps} originalTitle="鬼滅之刃" />);
      expect(screen.queryByText('鬼滅之刃')).not.toBeInTheDocument();
    });

    it('does not show original title when undefined', () => {
      render(<HoverPreviewCard {...defaultProps} originalTitle={undefined} />);
      // Only the genres and overview should be visible
      expect(screen.queryByTestId('original-title')).not.toBeInTheDocument();
    });
  });

  describe('Genre Display', () => {
    it('displays genres in Traditional Chinese', () => {
      render(<HoverPreviewCard {...defaultProps} />);
      expect(screen.getByText('動畫')).toBeInTheDocument();
      expect(screen.getByText('動作')).toBeInTheDocument();
      expect(screen.getByText('奇幻')).toBeInTheDocument();
    });

    it('limits genres to 3 maximum', () => {
      const props = {
        ...defaultProps,
        genreIds: [16, 28, 14, 35, 18], // 5 genres
      };
      render(<HoverPreviewCard {...props} />);
      expect(screen.getByText('動畫')).toBeInTheDocument();
      expect(screen.getByText('動作')).toBeInTheDocument();
      expect(screen.getByText('奇幻')).toBeInTheDocument();
      expect(screen.queryByText('喜劇')).not.toBeInTheDocument();
    });

    it('handles empty genre list', () => {
      render(<HoverPreviewCard {...defaultProps} genreIds={[]} />);
      // Should not crash and genres section should not be visible
      expect(screen.queryByTestId('genres-container')).not.toBeInTheDocument();
    });

    it('handles unknown genre IDs', () => {
      render(<HoverPreviewCard {...defaultProps} genreIds={[99999]} />);
      // Unknown genre should be filtered out
      expect(screen.queryByTestId('genres-container')).not.toBeInTheDocument();
    });
  });

  describe('Overview Display', () => {
    it('displays overview text', () => {
      render(<HoverPreviewCard {...defaultProps} />);
      expect(screen.getByText(/在大正時代的日本/)).toBeInTheDocument();
    });

    it('truncates long overview with line-clamp', () => {
      render(<HoverPreviewCard {...defaultProps} />);
      const overview = screen.getByTestId('overview');
      expect(overview).toHaveClass('line-clamp-3');
    });

    it('handles missing overview', () => {
      render(<HoverPreviewCard {...defaultProps} overview={undefined} />);
      expect(screen.queryByTestId('overview')).not.toBeInTheDocument();
    });
  });

  describe('Desktop-only Display', () => {
    it('has hidden class for non-desktop viewports', () => {
      render(<HoverPreviewCard {...defaultProps} />);
      const container = screen.getByTestId('hover-preview-card');
      expect(container).toHaveClass('hidden');
      expect(container).toHaveClass('lg:block');
    });
  });
});
