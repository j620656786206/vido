import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { PosterCard } from './PosterCard';
import { useMovieDetails, useTVShowDetails } from '../../hooks/useMediaDetails';

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
    <a href={`${to.replace('$type', params.type).replace('$id', params.id)}`} {...props}>
      {children}
    </a>
  ),
}));

// Mock the TMDb detail hooks — PosterCard calls useMovieDetails/useTVShowDetails for the
// lazy-on-hover metadata line (bugfix-10-7 AC #1). They are disabled (id=0) until hover-intent.
// Default per-test: no data (year-only). Per-test overrides via mockReturnValue/mockImplementation.
// Typed-mock via double-cast through Partial (bugfix-10-2 CR M3 pattern) — ZERO `as any`.
vi.mock('../../hooks/useMediaDetails', () => ({
  useMovieDetails: vi.fn(),
  useTVShowDetails: vi.fn(),
}));

const mockUseMovieDetails = vi.mocked(useMovieDetails);
const mockUseTVShowDetails = vi.mocked(useTVShowDetails);

type MovieDetailsResult = ReturnType<typeof useMovieDetails>;
type TVShowDetailsResult = ReturnType<typeof useTVShowDetails>;

const movieResult = (runtime?: number): MovieDetailsResult =>
  ({
    data: runtime === undefined ? undefined : { runtime },
  }) as Partial<MovieDetailsResult> as MovieDetailsResult;

const tvResult = (numberOfSeasons?: number, numberOfEpisodes?: number): TVShowDetailsResult =>
  ({
    data: numberOfSeasons === undefined ? undefined : { numberOfSeasons, numberOfEpisodes },
  }) as Partial<TVShowDetailsResult> as TVShowDetailsResult;

describe('PosterCard', () => {
  const defaultProps = {
    id: 'movie-123',
    type: 'movie' as const,
    title: '鬼滅之刃',
    posterPath: '/test-poster.jpg',
    releaseDate: '2020-10-16',
    voteAverage: 8.5,
  };

  beforeEach(() => {
    vi.clearAllMocks();
    vi.useRealTimers();
    // Default: detail hooks disabled / no data → metadata line stays year-only.
    mockUseMovieDetails.mockReturnValue(movieResult());
    mockUseTVShowDetails.mockReturnValue(tvResult());
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
      expect(img).toHaveAttribute('src', 'https://image.tmdb.org/t/p/w342/test-poster.jpg');
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
      expect(link).toHaveAttribute('href', '/media/movie/movie-123');
    });

    it('links to correct tv detail page', () => {
      render(<PosterCard {...defaultProps} type="tv" />);
      const link = screen.getByRole('link');
      expect(link).toHaveAttribute('href', '/media/tv/movie-123');
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
      expect(link).toHaveAttribute('href', '/media/movie/movie-123');
      // Verify focus is visible (focus-visible ring classes)
      expect(link).toHaveClass('focus-visible:ring-2');
    });
  });

  describe('Hover Interaction (in-card overlay per Component/PosterCardHover MQbvp)', () => {
    // bugfix-10-4: hover state is now CSS-driven via lg:group-hover: classes (no React state).
    // RTL cannot fire CSS :hover events, so we assert presence + correct hover-gating classes
    // per Rule 16 (toBeInTheDocument / toHaveClass over toBeVisible for hover-CSS-dependent
    // elements). Runtime opacity transition is exercised by tests/e2e/poster-card-hover.spec.ts.

    it('[P0] center play overlay is in DOM with hover-only visibility classes (AC #1)', () => {
      render(<PosterCard {...defaultProps} />);
      const overlay = screen.getByTestId('hover-play-overlay');
      expect(overlay).toBeInTheDocument();
      expect(overlay).toHaveClass('hidden', 'lg:flex', 'opacity-0', 'lg:group-hover:opacity-100');
    });

    it('[P1] center play overlay is NOT rendered in selection mode (AC #1)', () => {
      render(<PosterCard {...defaultProps} selectable={true} />);
      expect(screen.queryByTestId('hover-play-overlay')).not.toBeInTheDocument();
    });

    it('[P0] kebab menu repositioned to top-right (AC #1)', () => {
      const onMenuClick = vi.fn();
      render(<PosterCard {...defaultProps} onMenuClick={onMenuClick} />);
      const kebab = screen.getByTestId('poster-menu-button');
      expect(kebab).toHaveClass('right-2', 'top-2');
      expect(kebab).not.toHaveClass('left-2');
    });

    it('[P0] rating badge repositioned to bottom-right (AC #1)', () => {
      const { container } = render(<PosterCard {...defaultProps} voteAverage={8.5} />);
      const ratingWrapper = container.querySelector('.absolute.bottom-2.right-2');
      expect(ratingWrapper).toBeInTheDocument();
      const ratingWrapperLeft = container.querySelector('.absolute.bottom-2.left-2');
      expect(ratingWrapperLeft).not.toBeInTheDocument();
    });

    it('[P0] in-card title overlay is intentionally NOT rendered (Party Mode 2026-05-08 design correction)', () => {
      // Note: MQbvp originally specified a bottom-left title/year overlay.
      // Party Mode (Sally + Alexyu) determined this duplicates the below-image title
      // and has legibility issues against varying poster backgrounds.
      // In-card info-density redesign deferred to feature-X-postercard-info-density.
      render(<PosterCard {...defaultProps} />);
      expect(screen.queryByTestId('hover-title-overlay')).not.toBeInTheDocument();
    });

    it('[P1] HoverPreviewCard is no longer in the DOM (AC #2 — deletion regression guard)', () => {
      render(<PosterCard {...defaultProps} />);
      expect(screen.queryByTestId('hover-preview-card')).not.toBeInTheDocument();
    });
  });

  describe('New Badge (Story 5-8)', () => {
    it('renders "新增" badge when isNew is true', () => {
      render(<PosterCard {...defaultProps} isNew={true} />);
      expect(screen.getByTestId('new-badge')).toBeInTheDocument();
      expect(screen.getByText('新增')).toBeInTheDocument();
    });

    it('does not render "新增" badge when isNew is false', () => {
      render(<PosterCard {...defaultProps} isNew={false} />);
      expect(screen.queryByTestId('new-badge')).not.toBeInTheDocument();
    });

    it('does not render "新增" badge when isNew is undefined', () => {
      render(<PosterCard {...defaultProps} />);
      expect(screen.queryByTestId('new-badge')).not.toBeInTheDocument();
    });
  });

  describe('Availability Badges (Story 10-4)', () => {
    it('renders 已有 badge when isOwned is true (AC #1)', () => {
      render(<PosterCard {...defaultProps} isOwned={true} />);
      expect(screen.getByTestId('availability-badge-owned')).toBeInTheDocument();
      expect(screen.getByText('已有')).toBeInTheDocument();
    });

    it('renders 已請求 badge when isRequested is true (AC #2)', () => {
      render(<PosterCard {...defaultProps} isRequested={true} />);
      expect(screen.getByTestId('availability-badge-requested')).toBeInTheDocument();
      expect(screen.getByText('已請求')).toBeInTheDocument();
    });

    it('does not render any availability badge when neither flag is set', () => {
      render(<PosterCard {...defaultProps} />);
      expect(screen.queryByTestId('availability-badge-owned')).not.toBeInTheDocument();
      expect(screen.queryByTestId('availability-badge-requested')).not.toBeInTheDocument();
    });

    it('prefers 已有 over 已請求 when both flags are set (owned wins)', () => {
      render(<PosterCard {...defaultProps} isOwned={true} isRequested={true} />);
      expect(screen.getByTestId('availability-badge-owned')).toBeInTheDocument();
      expect(screen.queryByTestId('availability-badge-requested')).not.toBeInTheDocument();
    });

    it('availability badge coexists with isNew and type badges without overlap', () => {
      render(<PosterCard {...defaultProps} isOwned={true} isNew={true} />);
      // All three badges must be in the DOM — owned, new, and type.
      expect(screen.getByTestId('availability-badge-owned')).toBeInTheDocument();
      expect(screen.getByTestId('new-badge')).toBeInTheDocument();
      expect(screen.getByText('電影')).toBeInTheDocument();
    });
  });

  describe('Selection Mode (Story 5-7)', () => {
    it('[P0] renders selection checkbox when selectable is true', () => {
      render(<PosterCard {...defaultProps} selectable={true} />);
      expect(screen.getByTestId('selection-checkbox')).toBeInTheDocument();
    });

    it('[P0] hides selection checkbox when selectable is false', () => {
      render(<PosterCard {...defaultProps} selectable={false} />);
      expect(screen.queryByTestId('selection-checkbox')).not.toBeInTheDocument();
    });

    it('[P0] hides selection checkbox when selectable is undefined', () => {
      render(<PosterCard {...defaultProps} />);
      expect(screen.queryByTestId('selection-checkbox')).not.toBeInTheDocument();
    });

    it('[P0] calls onSelect when card is clicked in selection mode', () => {
      const onSelect = vi.fn();
      render(<PosterCard {...defaultProps} selectable={true} onSelect={onSelect} />);

      fireEvent.click(screen.getByTestId('poster-card'));
      expect(onSelect).toHaveBeenCalledOnce();
    });

    it('[P0] does not call onSelect when card is clicked outside selection mode', () => {
      const onSelect = vi.fn();
      render(<PosterCard {...defaultProps} selectable={false} onSelect={onSelect} />);

      fireEvent.click(screen.getByTestId('poster-card'));
      expect(onSelect).not.toHaveBeenCalled();
    });

    it('[P1] shows check icon when selected', () => {
      render(<PosterCard {...defaultProps} selectable={true} selected={true} />);
      // Check icon rendered inside selection-checkbox
      const checkbox = screen.getByTestId('selection-checkbox');
      expect(checkbox.querySelector('svg')).toBeInTheDocument();
    });

    it('[P1] does not show check icon when not selected', () => {
      render(<PosterCard {...defaultProps} selectable={true} selected={false} />);
      const checkbox = screen.getByTestId('selection-checkbox');
      expect(checkbox.querySelector('svg')).not.toBeInTheDocument();
    });

    it('[P1] applies ring-2 styling when selected', () => {
      const { container } = render(
        <PosterCard {...defaultProps} selectable={true} selected={true} />
      );
      const posterWrapper = container.querySelector('.aspect-\\[2\\/3\\]');
      expect(posterWrapper?.className).toContain('ring-2');
      expect(posterWrapper?.className).toContain('ring-[var(--accent-primary)]');
    });

    it('[P1] applies opacity-70 when selectable but not selected', () => {
      const { container } = render(
        <PosterCard {...defaultProps} selectable={true} selected={false} />
      );
      const posterWrapper = container.querySelector('.aspect-\\[2\\/3\\]');
      expect(posterWrapper?.className).toContain('opacity-70');
    });

    it('[P1] does not apply opacity-70 when selected', () => {
      const { container } = render(
        <PosterCard {...defaultProps} selectable={true} selected={true} />
      );
      const posterWrapper = container.querySelector('.aspect-\\[2\\/3\\]');
      expect(posterWrapper?.className).not.toContain('opacity-70');
    });
  });

  describe('Library-specific Props (Story 5-1)', () => {
    it('renders metadata source badge when metadataSource is provided', () => {
      render(<PosterCard {...defaultProps} metadataSource="TMDb" />);
      expect(screen.getByText('TMDb')).toBeInTheDocument();
    });

    it('does not render metadata source badge when not provided', () => {
      render(<PosterCard {...defaultProps} />);
      expect(screen.queryByText('TMDb')).not.toBeInTheDocument();
    });

    it('renders menu button when onMenuClick is provided', () => {
      const onMenuClick = vi.fn();
      render(<PosterCard {...defaultProps} onMenuClick={onMenuClick} />);
      expect(screen.getByTestId('poster-menu-button')).toBeInTheDocument();
    });

    it('does not render menu button when onMenuClick is not provided', () => {
      render(<PosterCard {...defaultProps} />);
      expect(screen.queryByTestId('poster-menu-button')).not.toBeInTheDocument();
    });

    it('calls onMenuClick when menu button is clicked', () => {
      const onMenuClick = vi.fn();
      render(<PosterCard {...defaultProps} onMenuClick={onMenuClick} />);

      fireEvent.click(screen.getByTestId('poster-menu-button'));
      expect(onMenuClick).toHaveBeenCalledTimes(1);
    });

    it('menu button has accessible label', () => {
      const onMenuClick = vi.fn();
      render(<PosterCard {...defaultProps} onMenuClick={onMenuClick} />);
      expect(screen.getByLabelText('更多選項')).toBeInTheDocument();
    });
  });

  describe('Metadata line (bugfix-10-7 AC #1 — info-density, lazy-on-hover)', () => {
    it('[P0] shows year only before hover (movie — detail hook disabled with id=0)', () => {
      render(<PosterCard {...defaultProps} type="movie" id="550" releaseDate="2022-03-25" />);
      expect(screen.getByText('2022')).toBeInTheDocument();
      expect(mockUseMovieDetails).toHaveBeenLastCalledWith(0);
    });

    it('[P0] shows year + runtime after the ~200 ms hover-intent debounce resolves (movie)', () => {
      vi.useFakeTimers();
      // Simulate the hooks' built-in `enabled: id > 0` gating: id=0 ⇒ no data, id>0 ⇒ runtime.
      mockUseMovieDetails.mockImplementation((id: number) => movieResult(id > 0 ? 139 : undefined));
      render(<PosterCard {...defaultProps} type="movie" id="550" releaseDate="2022-03-25" />);
      // Before hover-intent fires, only the year is shown (hook called with id=0 ⇒ disabled).
      expect(screen.getByText('2022')).toBeInTheDocument();
      expect(mockUseMovieDetails).toHaveBeenLastCalledWith(0);
      fireEvent.mouseEnter(screen.getByTestId('poster-card'));
      act(() => {
        vi.advanceTimersByTime(200);
      });
      expect(screen.getByText('2022 · 2 小時 19 分')).toBeInTheDocument();
      expect(mockUseMovieDetails).toHaveBeenLastCalledWith(550);
      vi.useRealTimers();
    });

    it('[P0] shows year + season/episode count after hover (tv)', () => {
      vi.useFakeTimers();
      mockUseTVShowDetails.mockImplementation((id: number) =>
        id > 0 ? tvResult(4, 34) : tvResult()
      );
      render(
        <PosterCard
          {...defaultProps}
          type="tv"
          id="66732"
          releaseDate="2016-07-15"
          voteAverage={8.6}
        />
      );
      expect(screen.getByText('2016')).toBeInTheDocument();
      expect(mockUseTVShowDetails).toHaveBeenLastCalledWith(0);
      fireEvent.mouseEnter(screen.getByTestId('poster-card'));
      act(() => {
        vi.advanceTimersByTime(200);
      });
      expect(screen.getByText('2016 · 4 季 34 集')).toBeInTheDocument();
      expect(mockUseTVShowDetails).toHaveBeenLastCalledWith(66732);
      vi.useRealTimers();
    });

    it('[P1] owned-library UUID id never triggers a detail fetch — stays year-only after hover', () => {
      vi.useFakeTimers();
      // Realistic mock: id=0 (disabled) returns no data, id>0 returns data. A UUID id derives to 0.
      mockUseMovieDetails.mockImplementation((id: number) => movieResult(id > 0 ? 139 : undefined));
      render(
        <PosterCard
          {...defaultProps}
          type="movie"
          id="0ce73c75-a742-4f3a-9b21-2c8e1f0a4d55"
          releaseDate="2022-03-25"
        />
      );
      fireEvent.mouseEnter(screen.getByTestId('poster-card'));
      act(() => {
        vi.advanceTimersByTime(200);
      });
      // hoverIntent flipped true, but fetchId stays 0 (UUID ⇒ tmdbId 0) ⇒ no enrichment.
      expect(mockUseMovieDetails).toHaveBeenLastCalledWith(0);
      expect(screen.getByText('2022')).toBeInTheDocument();
      expect(screen.queryByText(/小時/)).not.toBeInTheDocument();
      vi.useRealTimers();
    });

    it('[P1] does not render a metadata line when there is no year and no fetched extra', () => {
      // No year (releaseDate undefined), no fetched runtime ⇒ formatPosterMeta(null, '') ⇒ '' ⇒ <p> not rendered.
      const { container } = render(
        <PosterCard {...defaultProps} type="movie" id="550" releaseDate={undefined} />
      );
      expect(container.querySelector('.mt-2 p')).toBeNull();
    });
  });

  describe('Rating badge glyph (bugfix-10-7 AC #3 — lucide <Star>, not the ⭐ emoji)', () => {
    it('[P0] renders a lucide <Star> SVG inside the rating chip and no ⭐ emoji', () => {
      const { container } = render(<PosterCard {...defaultProps} voteAverage={8.4} />);
      const ratingChip = container.querySelector('.absolute.bottom-2.right-2');
      expect(ratingChip).not.toBeNull();
      expect(ratingChip?.querySelector('svg')).not.toBeNull();
      expect(screen.queryByText(/⭐/)).toBeNull();
      expect(screen.getByText('8.4')).toBeInTheDocument();
    });
  });

  describe('Hover badge-cluster recede (bugfix-10-7 AC #2 — scale-95 on fade)', () => {
    it('[P1] top-right badge cluster carries transition-all + origin-top-right + lg:group-hover:scale-95', () => {
      const { container } = render(<PosterCard {...defaultProps} />);
      const badgeCluster = container.querySelector('.absolute.right-2.top-2.origin-top-right');
      expect(badgeCluster).not.toBeNull();
      expect(badgeCluster).toHaveClass(
        'transition-all',
        'duration-300',
        'origin-top-right',
        'lg:group-hover:scale-95',
        'lg:group-hover:opacity-0'
      );
    });
  });

  // ---------------------------------------------------------------------------
  // TEA /testarch-automate pass (bugfix-10-7) — P2 regression guards for the
  // hover-intent debounce timing logic (AC #1). The DEV story covered the happy
  // path (enter → 200 ms → line resolves) + the no-fetch UUID case; these close
  // the cancel-path, no-flicker, and Rule-14-cleanup gaps. All deterministic
  // (fake timers + act), no hard waits, atomic — test-quality.md / test-priorities-matrix.md (P2: UI-polish edge cases).
  // ---------------------------------------------------------------------------
  describe('Hover-intent debounce edge cases (bugfix-10-7 AC #1 — TEA regression guards)', () => {
    it('[P2] mouseLeave before the ~200 ms debounce fires ⇒ timer cancelled, no detail fetch, line stays year-only', () => {
      vi.useFakeTimers();
      mockUseMovieDetails.mockImplementation((id: number) => movieResult(id > 0 ? 139 : undefined));
      render(<PosterCard {...defaultProps} type="movie" id="550" releaseDate="2022-03-25" />);
      const card = screen.getByTestId('poster-card');

      fireEvent.mouseEnter(card);
      act(() => {
        vi.advanceTimersByTime(100); // not yet — debounce is 200 ms
      });
      fireEvent.mouseLeave(card); // cancels the pending timer
      act(() => {
        vi.advanceTimersByTime(300); // well past when it would have fired
      });

      expect(screen.getByText('2022')).toBeInTheDocument();
      expect(screen.queryByText(/小時/)).not.toBeInTheDocument();
      // The detail hook was never asked for a real id ⇒ no network call would have happened.
      expect(mockUseMovieDetails.mock.calls.every(([id]) => id === 0)).toBe(true);
      vi.useRealTimers();
    });

    it('[P2] once the line has resolved, mouseLeave then re-enter does NOT flicker back to year-only', () => {
      vi.useFakeTimers();
      mockUseMovieDetails.mockImplementation((id: number) => movieResult(id > 0 ? 139 : undefined));
      render(<PosterCard {...defaultProps} type="movie" id="550" releaseDate="2022-03-25" />);
      const card = screen.getByTestId('poster-card');

      fireEvent.mouseEnter(card);
      act(() => {
        vi.advanceTimersByTime(200);
      });
      expect(screen.getByText('2022 · 2 小時 19 分')).toBeInTheDocument();

      // Leaving must NOT reset hoverIntent — the data is loaded, keep showing it.
      fireEvent.mouseLeave(card);
      expect(screen.getByText('2022 · 2 小時 19 分')).toBeInTheDocument();
      expect(screen.queryByText('2022')).not.toBeInTheDocument(); // the bare year is gone — the composed line stuck

      // Re-entering is idempotent — re-arms the (now-null) timer harmlessly, line unchanged.
      fireEvent.mouseEnter(card);
      act(() => {
        vi.advanceTimersByTime(200);
      });
      expect(screen.getByText('2022 · 2 小時 19 分')).toBeInTheDocument();
      vi.useRealTimers();
    });

    it('[P2] unmount with a pending hover-intent timer clears it (Rule 14 — no leaked timer)', () => {
      vi.useFakeTimers();
      const setSpy = vi.spyOn(globalThis, 'setTimeout');
      const clearSpy = vi.spyOn(globalThis, 'clearTimeout');
      mockUseMovieDetails.mockImplementation((id: number) => movieResult(id > 0 ? 139 : undefined));
      const { unmount } = render(
        <PosterCard {...defaultProps} type="movie" id="550" releaseDate="2022-03-25" />
      );

      fireEvent.mouseEnter(screen.getByTestId('poster-card')); // arms the 200 ms hover-intent timer
      // handleMouseEnter is the only setTimeout caller triggered by mouseEnter (no re-render) ⇒ last call is ours.
      const hoverTimerId = setSpy.mock.results[setSpy.mock.results.length - 1].value;

      unmount();

      // The useEffect cleanup must clearTimeout the still-pending hover-intent timer.
      expect(clearSpy).toHaveBeenCalledWith(hoverTimerId);

      setSpy.mockRestore();
      clearSpy.mockRestore();
      vi.useRealTimers();
    });
  });
});
