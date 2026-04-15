import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { TrailerModal, pickBestTrailer } from './TrailerModal';
import type { Video } from '../../types/tmdb';

vi.mock('../../services/tmdb', () => ({
  default: {
    getMovieVideos: vi.fn(),
    getTVShowVideos: vi.fn(),
  },
}));

import tmdbService from '../../services/tmdb';

const mockGetMovieVideos = vi.mocked(tmdbService.getMovieVideos);
const mockGetTVShowVideos = vi.mocked(tmdbService.getTVShowVideos);

function video(overrides: Partial<Video> = {}): Video {
  return {
    key: 'abc123',
    name: 'Trailer',
    site: 'YouTube',
    type: 'Trailer',
    official: true,
    publishedAt: '2024-01-01T00:00:00.000Z',
    ...overrides,
  };
}

function renderModal(props: Partial<React.ComponentProps<typeof TrailerModal>> = {}) {
  const defaults: React.ComponentProps<typeof TrailerModal> = {
    open: true,
    onClose: vi.fn(),
    mediaType: 'movie',
    tmdbId: 550,
    title: 'Fight Club',
  };
  const merged = { ...defaults, ...props };
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return {
    ...render(
      React.createElement(
        QueryClientProvider,
        { client: queryClient },
        React.createElement(TrailerModal, merged)
      )
    ),
    onClose: merged.onClose,
  };
}

describe('pickBestTrailer', () => {
  it('returns null for empty/undefined results', () => {
    expect(pickBestTrailer(undefined)).toBeNull();
    expect(pickBestTrailer([])).toBeNull();
  });

  it('filters out non-YouTube trailers', () => {
    const results = [
      video({ key: 'vimeo1', site: 'Vimeo' }),
      video({ key: 'tt', site: 'YouTube' }),
    ];
    expect(pickBestTrailer(results)?.key).toBe('tt');
  });

  it('filters out non-Trailer types (Teaser, Featurette)', () => {
    const results = [
      video({ key: 'teaser1', type: 'Teaser' }),
      video({ key: 'real-trailer', type: 'Trailer' }),
    ];
    expect(pickBestTrailer(results)?.key).toBe('real-trailer');
  });

  it('rejects keys with invalid characters (XSS guard)', () => {
    const results = [
      video({ key: '<script>', type: 'Trailer' }),
      video({ key: 'safe_key-123', type: 'Trailer' }),
    ];
    expect(pickBestTrailer(results)?.key).toBe('safe_key-123');
  });

  it('prefers official over unofficial', () => {
    const results = [
      video({ key: 'fan', official: false, publishedAt: '2025-01-01' }),
      video({ key: 'official', official: true, publishedAt: '2020-01-01' }),
    ];
    expect(pickBestTrailer(results)?.key).toBe('official');
  });

  it('among same officiality, prefers most recent', () => {
    const results = [
      video({ key: 'old', official: true, publishedAt: '2020-01-01' }),
      video({ key: 'new', official: true, publishedAt: '2025-06-01' }),
    ];
    expect(pickBestTrailer(results)?.key).toBe('new');
  });
});

describe('TrailerModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('[P1] renders nothing when open is false', () => {
    renderModal({ open: false });
    expect(screen.queryByTestId('trailer-modal')).toBeNull();
    expect(mockGetMovieVideos).not.toHaveBeenCalled();
  });

  it('[P1] renders YouTube iframe with autoplay when trailer found (AC #6)', async () => {
    mockGetMovieVideos.mockResolvedValue({
      id: 550,
      results: [video({ key: 'SUXWAEX2jlg' })],
    });

    renderModal();

    const iframe = await screen.findByTestId('trailer-modal-iframe');
    expect(iframe).toBeInTheDocument();
    expect(iframe.getAttribute('src')).toBe(
      'https://www.youtube-nocookie.com/embed/SUXWAEX2jlg?autoplay=1'
    );
  });

  it('[P1] uses TV endpoint when mediaType is tv', async () => {
    mockGetTVShowVideos.mockResolvedValue({ id: 1396, results: [] });

    renderModal({ mediaType: 'tv', tmdbId: 1396 });

    await waitFor(() => expect(mockGetTVShowVideos).toHaveBeenCalledWith(1396));
    expect(mockGetMovieVideos).not.toHaveBeenCalled();
  });

  it('[P1] closes on Escape key (AC #6)', async () => {
    mockGetMovieVideos.mockResolvedValue({ id: 550, results: [video()] });

    const { onClose } = renderModal();

    await screen.findByTestId('trailer-modal-iframe');
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('[P1] closes on backdrop click (AC #6)', async () => {
    mockGetMovieVideos.mockResolvedValue({ id: 550, results: [video()] });

    const { onClose } = renderModal();

    await screen.findByTestId('trailer-modal-iframe');
    const modal = screen.getByTestId('trailer-modal');
    fireEvent.click(modal);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('[P1] does NOT close when clicking inside the modal content', async () => {
    mockGetMovieVideos.mockResolvedValue({ id: 550, results: [video()] });

    const { onClose } = renderModal();

    const iframe = await screen.findByTestId('trailer-modal-iframe');
    fireEvent.click(iframe);
    expect(onClose).not.toHaveBeenCalled();
  });

  it('[P1] close button triggers onClose', async () => {
    mockGetMovieVideos.mockResolvedValue({ id: 550, results: [video()] });

    const { onClose } = renderModal();

    await screen.findByTestId('trailer-modal-iframe');
    fireEvent.click(screen.getByTestId('trailer-modal-close'));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('[P1] shows empty state when no embeddable trailer found', async () => {
    mockGetMovieVideos.mockResolvedValue({ id: 550, results: [] });

    renderModal();

    expect(await screen.findByTestId('trailer-modal-empty')).toBeInTheDocument();
    expect(screen.getByText('找不到預告片')).toBeInTheDocument();
  });

  it('[P2] shows empty state on API error (graceful degradation)', async () => {
    mockGetMovieVideos.mockRejectedValue(new Error('API down'));

    renderModal();

    // L1 fix changed retry from default (3 retries) to 1, so the empty state
    // surfaces after roughly one retry's backoff. Bumped from default 1000ms.
    expect(
      await screen.findByTestId('trailer-modal-empty', {}, { timeout: 3000 })
    ).toBeInTheDocument();
  });

  it('[P2] dialog has aria-modal and accessible label', async () => {
    mockGetMovieVideos.mockResolvedValue({ id: 550, results: [video()] });

    renderModal({ title: '鬥陣俱樂部' });

    const dialog = await screen.findByRole('dialog');
    expect(dialog.getAttribute('aria-modal')).toBe('true');
    expect(dialog.getAttribute('aria-label')).toBe('鬥陣俱樂部 預告片');
  });

  // H2 fix — focus management for aria-modal dialog (WCAG 2.4.3).

  it('[P1] moves focus to close button when dialog opens (H2 fix)', async () => {
    mockGetMovieVideos.mockResolvedValue({ id: 550, results: [video()] });
    renderModal();
    const closeBtn = await screen.findByTestId('trailer-modal-close');
    await waitFor(() => expect(document.activeElement).toBe(closeBtn));
  });

  it('[P1] restores focus to the previously-focused element on close (H2 fix)', async () => {
    mockGetMovieVideos.mockResolvedValue({ id: 550, results: [video()] });
    // Set up an external trigger that "owns" focus before the modal opens.
    const trigger = document.createElement('button');
    trigger.textContent = 'Open';
    document.body.appendChild(trigger);
    trigger.focus();
    expect(document.activeElement).toBe(trigger);

    const { rerender } = render(
      React.createElement(
        QueryClientProvider,
        { client: new QueryClient({ defaultOptions: { queries: { retry: false } } }) },
        React.createElement(TrailerModal, {
          open: true,
          onClose: vi.fn(),
          mediaType: 'movie',
          tmdbId: 550,
          title: 'X',
        })
      )
    );

    await screen.findByTestId('trailer-modal-close');
    // Close the modal by re-rendering with open=false.
    rerender(
      React.createElement(
        QueryClientProvider,
        { client: new QueryClient({ defaultOptions: { queries: { retry: false } } }) },
        React.createElement(TrailerModal, {
          open: false,
          onClose: vi.fn(),
          mediaType: 'movie',
          tmdbId: 550,
          title: 'X',
        })
      )
    );

    await waitFor(() => expect(document.activeElement).toBe(trigger));
    document.body.removeChild(trigger);
  });

  it('[P1] traps Tab forward at the last focusable element (H2 fix)', async () => {
    mockGetMovieVideos.mockResolvedValue({ id: 550, results: [video()] });
    renderModal();
    const closeBtn = await screen.findByTestId('trailer-modal-close');
    const iframe = await screen.findByTestId('trailer-modal-iframe');

    // Move focus to the LAST focusable element (iframe), then Tab forward.
    iframe.focus();
    expect(document.activeElement).toBe(iframe);

    fireEvent.keyDown(document, { key: 'Tab' });
    // Trap should send focus back to the first focusable element (close button).
    await waitFor(() => expect(document.activeElement).toBe(closeBtn));
  });

  it('[P1] traps Shift+Tab backward at the first focusable element (H2 fix)', async () => {
    mockGetMovieVideos.mockResolvedValue({ id: 550, results: [video()] });
    renderModal();
    const closeBtn = await screen.findByTestId('trailer-modal-close');
    const iframe = await screen.findByTestId('trailer-modal-iframe');

    // Currently on first focusable (close); Shift+Tab should wrap to last (iframe).
    closeBtn.focus();
    expect(document.activeElement).toBe(closeBtn);

    fireEvent.keyDown(document, { key: 'Tab', shiftKey: true });
    await waitFor(() => expect(document.activeElement).toBe(iframe));
  });
});
