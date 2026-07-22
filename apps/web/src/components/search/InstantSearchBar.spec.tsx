import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { InstantSearchBar } from './InstantSearchBar';
import { tmdbService } from '../../services/tmdb';
import type { UnifiedSearchResult } from '../../types/tmdb';

vi.mock('../../services/tmdb', () => ({
  tmdbService: { unifiedSearch: vi.fn() },
}));

const sample: UnifiedSearchResult = {
  query: '你的名字',
  page: 1,
  localMovies: [],
  localTv: [],
  movies: [
    {
      id: 1,
      title: '你的名字',
      originalTitle: 'Your Name',
      overview: '',
      releaseDate: '2016-08-26',
      posterPath: null,
      backdropPath: null,
      voteAverage: 8.4,
      voteCount: 100,
      genreIds: [],
    },
  ],
  tvShows: [
    {
      id: 2,
      name: '進擊的巨人',
      originalName: 'Attack on Titan',
      overview: '',
      firstAirDate: '2013-04-07',
      posterPath: null,
      backdropPath: null,
      voteAverage: 8.6,
      voteCount: 50,
      genreIds: [],
    },
  ],
  people: [],
};

function setup(initialPath = '/', barProps: { focusOnMount?: boolean } = {}) {
  const rootRoute = createRootRoute({
    component: () => <InstantSearchBar variant="desktop" {...barProps} />,
  });
  const indexRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/',
    component: () => <div>home</div>,
  });
  const mediaRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/media/$type/$id',
    component: () => <div>media</div>,
  });
  const searchRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/search',
    component: () => <div>search</div>,
  });
  const routeTree = rootRoute.addChildren([indexRoute, mediaRoute, searchRoute]);
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: [initialPath] }),
  });
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  );
  return router;
}

describe('InstantSearchBar', () => {
  beforeEach(() => {
    vi.mocked(tmdbService.unifiedSearch).mockReset();
    vi.mocked(tmdbService.unifiedSearch).mockResolvedValue(sample);
  });

  it('focuses the input on mount when focusOnMount is set (retro-11-AI1b — replaces autoFocus)', async () => {
    setup('/', { focusOnMount: true });
    const input = await screen.findByTestId('instant-search-input');
    await waitFor(() => expect(input).toHaveFocus());
  });

  it('does not steal focus on mount by default', async () => {
    setup();
    const input = await screen.findByTestId('instant-search-input');
    expect(input).not.toHaveFocus();
  });

  it('debounces input — suggestions do not appear synchronously but do after the debounce window', async () => {
    setup();
    const input = await screen.findByTestId('instant-search-input');

    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: '你的名字' } });

    // Synchronously: debounced query is still empty → no dropdown, no fetch yet.
    expect(screen.queryByTestId('search-suggestions')).not.toBeInTheDocument();
    expect(tmdbService.unifiedSearch).not.toHaveBeenCalled();

    // After the 300ms debounce + query resolution, suggestions render.
    await waitFor(() => expect(screen.getByTestId('search-suggestions')).toBeInTheDocument());
    expect(tmdbService.unifiedSearch).toHaveBeenCalledWith('你的名字');
  });

  it('does not search for queries shorter than 2 characters', async () => {
    setup();
    const input = await screen.findByTestId('instant-search-input');

    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: '你' } });

    // Wait past the debounce window inside act() so the resulting state update
    // (debouncedQuery → '你') is flushed without an act warning.
    await act(async () => {
      await new Promise((r) => setTimeout(r, 350));
    });
    expect(tmdbService.unifiedSearch).not.toHaveBeenCalled();
    expect(screen.queryByTestId('search-suggestions')).not.toBeInTheDocument();
  });

  it('navigates to the media detail page when a suggestion is clicked (AC #4)', async () => {
    const router = setup();
    const input = await screen.findByTestId('instant-search-input');

    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: '你的名字' } });

    await waitFor(() => expect(screen.getByText('你的名字')).toBeInTheDocument());
    fireEvent.click(screen.getByText('你的名字'));

    await waitFor(() => expect(router.state.location.pathname).toBe('/media/movie/1'));
  });

  it('supports arrow-key navigation + Enter to open the highlighted result', async () => {
    const router = setup();
    const input = await screen.findByTestId('instant-search-input');

    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: '你的名字' } });
    // Wait for an actual result row (not just the container, which renders while
    // results are still loading) so the navigable list is populated.
    await waitFor(() => expect(screen.getByText('進擊的巨人')).toBeInTheDocument());

    // ArrowDown highlights the first navigable (the movie), ArrowDown again the TV show.
    fireEvent.keyDown(input, { key: 'ArrowDown' });
    // aria-activedescendant points the combobox at the highlighted option so
    // screen readers announce it (a11y — Story 11-3 CR fix).
    expect(input).toHaveAttribute('aria-activedescendant', 'search-option-0');
    fireEvent.keyDown(input, { key: 'ArrowDown' });
    expect(input).toHaveAttribute('aria-activedescendant', 'search-option-1');
    fireEvent.keyDown(input, { key: 'Enter' });

    await waitFor(() => expect(router.state.location.pathname).toBe('/media/tv/2'));
  });

  it('navigates to the full /search results page on Enter with no highlight', async () => {
    const router = setup();
    const input = await screen.findByTestId('instant-search-input');

    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: '你的名字' } });
    fireEvent.keyDown(input, { key: 'Enter' });

    await waitFor(() => {
      expect(router.state.location.pathname).toBe('/search');
      expect(router.state.location.search).toEqual({ q: '你的名字' });
    });
  });

  it('clears the input via the clear button', async () => {
    setup();
    const input = (await screen.findByTestId('instant-search-input')) as HTMLInputElement;

    fireEvent.change(input, { target: { value: 'iron' } });
    expect(input.value).toBe('iron');

    fireEvent.click(screen.getByTestId('instant-search-clear'));
    expect(input.value).toBe('');
  });
});
