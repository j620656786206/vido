import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ContinueWatchingSlot } from './ContinueWatchingSlot';

describe('ContinueWatchingSlot (ux3-1-3 — reserved slot, blocked on Epic 17)', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('[P1] renders the quiet 「連接 Plex / Jellyfin 後顯示」 affordance, never a broken tile', () => {
    render(<ContinueWatchingSlot />);
    expect(screen.getByTestId('home-continue-watching')).toBeInTheDocument();
    expect(screen.getByText('繼續觀看')).toBeInTheDocument();
    expect(screen.getByText('連接 Plex / Jellyfin 後顯示')).toBeInTheDocument();
  });

  it('[P2] emits no console error/warning (no media server = no noise)', () => {
    const err = vi.spyOn(console, 'error').mockImplementation(() => {});
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    render(<ContinueWatchingSlot />);
    expect(err).not.toHaveBeenCalled();
    expect(warn).not.toHaveBeenCalled();
  });
});
