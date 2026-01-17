import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { PosterCard } from './PosterCard';

// Mock TanStack Router
vi.mock('@tanstack/react-router', () => ({
  Link: ({
    children,
    to,
    params,
    ...props
  }: {
    children: React.ReactNode;
    to: string;
    params: Record<string, string>;
  }) => (
    <a
      href={`${to.replace('$type', params.type).replace('$id', params.id)}`}
      {...props}
    >
      {children}
    </a>
  ),
}));

describe('PosterCard', () => {
  const defaultProps = {
    id: 123,
    type: 'movie' as const,
    title: '鬼滅之刃',
    posterPath: '/test-poster.jpg',
    releaseDate: '2020-10-16',
    voteAverage: 8.5,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Basic Rendering', () => {
    it('renders title correctly', () => {
      render(<PosterCard {...defaultProps} />);
      expect(screen.getByText('鬼滅之刃')).toBeInTheDocument();
    });

    it('renders year from release date', () => {
      render(<PosterCard {...defaultProps} />);
      expect(screen.getByText('2020')).toBeInTheDocument();
    });

    it('renders rating badge with correct format', () => {
      render(<PosterCard {...defaultProps} />);
      expect(screen.getByText(/8\.5/)).toBeInTheDocument();
    });

    it('renders movie type badge in zh-TW', () => {
      render(<PosterCard {...defaultProps} type="movie" />);
      expect(screen.getByText('電影')).toBeInTheDocument();
    });

    it('renders tv type badge in zh-TW', () => {
      render(<PosterCard {...defaultProps} type="tv" />);
      expect(screen.getByText('影集')).toBeInTheDocument();
    });

    it('does not render year when release date is missing', () => {
      render(<PosterCard {...defaultProps} releaseDate={undefined} />);
      expect(screen.queryByText('2020')).not.toBeInTheDocument();
    });

    it('does not render rating badge when vote average is 0', () => {
      render(<PosterCard {...defaultProps} voteAverage={0} />);
      expect(screen.queryByText('0.0')).not.toBeInTheDocument();
    });

    it('does not render rating badge when vote average is undefined', () => {
      render(<PosterCard {...defaultProps} voteAverage={undefined} />);
      expect(screen.queryByRole('img', { name: /star/i })).not.toBeInTheDocument();
    });
  });

  describe('Poster Image', () => {
    it('renders poster image with lazy loading', () => {
      render(<PosterCard {...defaultProps} />);
      const img = screen.getByRole('img', { name: '鬼滅之刃' });
      expect(img).toHaveAttribute('loading', 'lazy');
    });

    it('constructs correct TMDb image URL', () => {
      render(<PosterCard {...defaultProps} />);
      const img = screen.getByRole('img', { name: '鬼滅之刃' });
      expect(img).toHaveAttribute(
        'src',
        'https://image.tmdb.org/t/p/w342/test-poster.jpg'
      );
    });

    it('includes srcSet for responsive images', () => {
      render(<PosterCard {...defaultProps} />);
      const img = screen.getByRole('img', { name: '鬼滅之刃' });
      const srcSet = img.getAttribute('srcset');
      expect(srcSet).toContain('185w');
      expect(srcSet).toContain('342w');
      expect(srcSet).toContain('500w');
    });

    it('includes sizes attribute for responsive rendering', () => {
      render(<PosterCard {...defaultProps} />);
      const img = screen.getByRole('img', { name: '鬼滅之刃' });
      expect(img).toHaveAttribute('sizes');
    });

    it('shows loading skeleton initially', () => {
      render(<PosterCard {...defaultProps} />);
      expect(screen.getByTestId('poster-skeleton')).toBeInTheDocument();
    });

    it('hides skeleton after image loads', async () => {
      render(<PosterCard {...defaultProps} />);
      const img = screen.getByRole('img', { name: '鬼滅之刃' });

      fireEvent.load(img);

      await waitFor(() => {
        expect(screen.queryByTestId('poster-skeleton')).not.toBeInTheDocument();
      });
    });

    it('shows fallback placeholder when poster path is null', () => {
      render(<PosterCard {...defaultProps} posterPath={null} />);
      expect(screen.getByTestId('poster-fallback')).toBeInTheDocument();
    });

    it('shows fallback placeholder on image error', async () => {
      render(<PosterCard {...defaultProps} />);
      const img = screen.getByRole('img', { name: '鬼滅之刃' });

      fireEvent.error(img);

      await waitFor(() => {
        expect(screen.getByTestId('poster-fallback')).toBeInTheDocument();
      });
    });
  });

  describe('Navigation', () => {
    it('links to correct movie detail page', () => {
      render(<PosterCard {...defaultProps} />);
      const link = screen.getByRole('link');
      expect(link).toHaveAttribute('href', '/media/movie/123');
    });

    it('links to correct tv detail page', () => {
      render(<PosterCard {...defaultProps} type="tv" />);
      const link = screen.getByRole('link');
      expect(link).toHaveAttribute('href', '/media/tv/123');
    });
  });

  describe('Accessibility', () => {
    it('has accessible image alt text', () => {
      render(<PosterCard {...defaultProps} />);
      expect(screen.getByRole('img', { name: '鬼滅之刃' })).toBeInTheDocument();
    });

    it('card is focusable via link', () => {
      render(<PosterCard {...defaultProps} />);
      const link = screen.getByRole('link');
      link.focus();
      expect(link).toHaveFocus();
    });

    it('supports keyboard navigation (Enter to select)', () => {
      render(<PosterCard {...defaultProps} />);
      const link = screen.getByRole('link');
      // Link elements natively support Enter key for activation
      // Verify the link has correct href for keyboard navigation
      expect(link).toHaveAttribute('href', '/media/movie/123');
      // Verify focus is visible (focus-visible ring classes)
      expect(link).toHaveClass('focus-visible:ring-2');
    });
  });

  describe('Hover Interaction', () => {
    it('shows HoverPreviewCard on mouse enter', () => {
      render(
        <PosterCard
          {...defaultProps}
          overview="這是一部關於鬼殺隊的動畫"
          genreIds={[16, 28]}
        />
      );
      const link = screen.getByRole('link');

      // Initially, hover preview should not be visible
      expect(screen.queryByTestId('hover-preview-card')).not.toBeInTheDocument();

      // Trigger mouse enter
      fireEvent.mouseEnter(link);

      // Hover preview should now be visible
      expect(screen.getByTestId('hover-preview-card')).toBeInTheDocument();
    });

    it('hides HoverPreviewCard on mouse leave', () => {
      render(
        <PosterCard
          {...defaultProps}
          overview="這是一部關於鬼殺隊的動畫"
          genreIds={[16, 28]}
        />
      );
      const link = screen.getByRole('link');

      // Show the hover preview
      fireEvent.mouseEnter(link);
      expect(screen.getByTestId('hover-preview-card')).toBeInTheDocument();

      // Hide the hover preview
      fireEvent.mouseLeave(link);
      expect(screen.queryByTestId('hover-preview-card')).not.toBeInTheDocument();
    });

    it('displays overview in hover preview', () => {
      render(
        <PosterCard
          {...defaultProps}
          overview="這是一部關於鬼殺隊的動畫"
          genreIds={[16, 28]}
        />
      );
      const link = screen.getByRole('link');

      fireEvent.mouseEnter(link);

      expect(screen.getByText('這是一部關於鬼殺隊的動畫')).toBeInTheDocument();
    });

    it('displays genres in hover preview', () => {
      render(
        <PosterCard
          {...defaultProps}
          overview="Test overview"
          genreIds={[16, 28]}
        />
      );
      const link = screen.getByRole('link');

      fireEvent.mouseEnter(link);

      expect(screen.getByText('動畫')).toBeInTheDocument();
      expect(screen.getByText('動作')).toBeInTheDocument();
    });
  });
});
