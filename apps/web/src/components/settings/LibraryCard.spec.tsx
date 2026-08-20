/**
 * LibraryCard — Story 9R-10b: the auto-generation opt-in must be visible from
 * the list, not only from inside the edit modal.
 *
 * WHY THIS MATTERS. `media_libraries` already carries one per-library boolean
 * (`auto_detect`) that has full CRUD behind it and NO user interface at all —
 * an invisible setting nobody can see or change. Sally's design ruling for
 * 9R-10b (J4-D, sPzZT) exists specifically so the opt-in does not become the
 * second one: a user must be able to tell, at a glance, which libraries will
 * process new files automatically.
 */
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { LibraryCard } from './LibraryCard';
import type { MediaLibraryWithPaths } from '../../services/mediaLibraryService';

vi.mock('../../hooks/useMediaLibrary', () => ({
  useDeleteLibrary: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

function libraryWith(autoSubtitle: boolean): MediaLibraryWithPaths {
  return {
    id: 'lib-1',
    name: '我的電影',
    contentType: 'movie',
    autoDetect: false,
    autoSubtitle,
    sortOrder: 0,
    createdAt: '2026-08-20T00:00:00Z',
    updatedAt: '2026-08-20T00:00:00Z',
    mediaCount: 316,
    paths: [
      {
        id: 'p1',
        libraryId: 'lib-1',
        path: '/volume1/media/movies',
        status: 'accessible',
        lastCheckedAt: null,
        createdAt: '2026-08-20T00:00:00Z',
      },
      {
        id: 'p2',
        libraryId: 'lib-1',
        path: '/volume1/media/movies-4k',
        status: 'accessible',
        lastCheckedAt: null,
        createdAt: '2026-08-20T00:00:00Z',
      },
    ],
  };
}

describe('LibraryCard auto-subtitle state', () => {
  it('shows the opt-in in the footer when the library is on', () => {
    render(<LibraryCard library={libraryWith(true)} onEdit={vi.fn()} />);

    expect(screen.getByTestId('library-card-footer')).toHaveTextContent(
      '2 個資料夾 · 316 個項目 · 自動處理免費字幕'
    );
  });

  it('says nothing at all when the library is off', () => {
    render(<LibraryCard library={libraryWith(false)} onEdit={vi.fn()} />);

    const footer = screen.getByTestId('library-card-footer');
    expect(footer).toHaveTextContent('2 個資料夾 · 316 個項目');
    expect(footer.textContent).not.toContain('自動處理免費字幕');
  });

  it('never implies that scanning itself generates subtitles (sub-4-3 AC #6)', () => {
    render(<LibraryCard library={libraryWith(true)} onEdit={vi.fn()} />);

    expect(screen.getByTestId('library-card-footer').textContent).not.toContain('掃描');
  });
});
