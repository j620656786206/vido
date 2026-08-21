/**
 * MediaLibraryManager — 9R-10b-M4: the capability WIRING.
 *
 * WHY THIS FILE EXISTS. `LibraryCard` renders three honest states, and
 * `LibraryEditModal` reads the capability off its own query — but nothing
 * connected the capability to the CARD. The manager is the only place that
 * hands it over, and a `autoSubtitleSupported={true}` hardcoded there would
 * bring the whole "card claims work nobody is doing" bug straight back with
 * every other test still green.
 *
 * These tests fail if that wire is cut. They are the only ones that can.
 */
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { MediaLibraryManager } from './MediaLibraryManager';
import { useMediaLibraries } from '../../hooks/useMediaLibrary';

vi.mock('../../hooks/useMediaLibrary', () => ({
  useMediaLibraries: vi.fn(),
  useDeleteLibrary: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useCreateLibrary: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateLibrary: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useAddLibraryPath: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useRemoveLibraryPath: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

const optedInLibrary = {
  id: 'lib-1',
  name: '我的電影',
  contentType: 'movie' as const,
  autoDetect: false,
  autoSubtitle: true,
  sortOrder: 0,
  createdAt: '2026-08-20T00:00:00Z',
  updatedAt: '2026-08-20T00:00:00Z',
  mediaCount: 316,
  paths: [],
};

function mockQuery(autoSubtitleSupported?: boolean) {
  vi.mocked(useMediaLibraries).mockReturnValue({
    data: { libraries: [optedInLibrary], autoSubtitleSupported },
    isLoading: false,
    error: null,
  } as unknown as ReturnType<typeof useMediaLibraries>);
}

describe('MediaLibraryManager auto-subtitle capability wiring', () => {
  it('passes the UNSUPPORTED capability down so the card warns instead of boasting', () => {
    mockQuery(false);

    render(<MediaLibraryManager />);

    expect(screen.getByTestId('library-card-auto-subtitle-status')).toHaveTextContent(
      '自動處理免費字幕（伺服器未啟用）'
    );
  });

  it('passes the SUPPORTED capability down so the card stays green', () => {
    mockQuery(true);

    render(<MediaLibraryManager />);

    const status = screen.getByTestId('library-card-auto-subtitle-status');
    expect(status).toHaveTextContent('自動處理免費字幕');
    expect(status.textContent).not.toContain('伺服器未啟用');
  });

  it('treats a missing capability field as supported, not unsupported', () => {
    // Same `!== false` semantics as the edit modal. An API that omits the key
    // is UNKNOWN — hiding or down-grading a shipped feature on a missing field
    // would be the worse of the two failures.
    mockQuery(undefined);

    render(<MediaLibraryManager />);

    expect(screen.getByTestId('library-card-auto-subtitle-status').textContent).not.toContain(
      '伺服器未啟用'
    );
  });
});
