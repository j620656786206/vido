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
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
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
    render(<LibraryCard library={libraryWith(true)} autoSubtitleSupported onEdit={vi.fn()} />);

    expect(screen.getByTestId('library-card-footer')).toHaveTextContent(
      '2 個資料夾 · 316 個項目 · 自動處理免費字幕'
    );
  });

  it('says nothing at all when the library is off', () => {
    render(<LibraryCard library={libraryWith(false)} autoSubtitleSupported onEdit={vi.fn()} />);

    const footer = screen.getByTestId('library-card-footer');
    expect(footer).toHaveTextContent('2 個資料夾 · 316 個項目');
    expect(footer.textContent).not.toContain('自動處理免費字幕');
  });

  it('never implies that scanning itself generates subtitles (sub-4-3 AC #6)', () => {
    render(<LibraryCard library={libraryWith(true)} autoSubtitleSupported onEdit={vi.fn()} />);

    expect(screen.getByTestId('library-card-footer').textContent).not.toContain('掃描');
  });

  // ─── 9R-10b-M4: the card must not claim something that is not happening ───
  //
  // `auto_subtitle` is writable in every mode, but the generator that honours
  // it exists only when the API runs in `pipeline` mode. A library left opted
  // in after the server was switched back to `legacy` kept showing the green
  // "自動處理免費字幕" — announcing work nobody was doing.
  //
  // Colour is a RULE, not three separate decisions:
  //   success = it is happening
  //   warning = you asked for it, and it is NOT happening
  //   absent  = you did not ask

  it('warns, rather than boasts, when the server cannot honour the opt-in', () => {
    render(
      <LibraryCard library={libraryWith(true)} autoSubtitleSupported={false} onEdit={vi.fn()} />
    );

    const status = screen.getByTestId('library-card-auto-subtitle-status');
    // The parenthetical names WHO is not enabled. Without it this reads as
    // "you didn't tick the box" — and the user did tick it, which is the
    // worst possible misreading.
    expect(status).toHaveTextContent('自動處理免費字幕（伺服器未啟用）');
    expect(status.className).toContain('text-[var(--warning-text)]');
  });

  it('stays green when the server DOES honour the opt-in', () => {
    render(<LibraryCard library={libraryWith(true)} autoSubtitleSupported onEdit={vi.fn()} />);

    const status = screen.getByTestId('library-card-auto-subtitle-status');
    expect(status).toHaveTextContent('自動處理免費字幕');
    expect(status.textContent).not.toContain('伺服器未啟用');
    expect(status.className).toContain('text-[var(--success-text)]');
  });

  it('says nothing when the library is off AND the server cannot honour it', () => {
    // Deliberately silent: the user never asked for anything here, so the card
    // has no reason to worry them about a capability they did not request.
    render(
      <LibraryCard library={libraryWith(false)} autoSubtitleSupported={false} onEdit={vi.fn()} />
    );

    expect(screen.queryByTestId('library-card-auto-subtitle-status')).not.toBeInTheDocument();
    expect(screen.getByTestId('library-card-footer')).toHaveTextContent('2 個資料夾 · 316 個項目');
  });

  it('never implies scanning generates subtitles in the unsupported state either', () => {
    render(
      <LibraryCard library={libraryWith(true)} autoSubtitleSupported={false} onEdit={vi.fn()} />
    );

    expect(screen.getByTestId('library-card-footer').textContent).not.toContain('掃描');
  });
});

// disc-2026-09-scanner-custom-modals-a11y: the ⋮ menu is a real menu — Esc and
// a click outside close it.
describe('LibraryCard — ⋮ menu', () => {
  async function openMenu() {
    const user = userEvent.setup();
    const onEdit = vi.fn();
    render(<LibraryCard library={libraryWith(false)} autoSubtitleSupported onEdit={onEdit} />);
    await user.click(screen.getByRole('button', { name: '我的電影 的操作' }));
    return { user, onEdit };
  }

  it('opens as a menu with 編輯 and 刪除', async () => {
    await openMenu();
    expect(screen.getByRole('menu')).toBeInTheDocument();
    expect(screen.getByRole('menuitem', { name: '編輯' })).toBeInTheDocument();
    expect(screen.getByRole('menuitem', { name: '刪除' })).toBeInTheDocument();
  });

  it('Esc closes it', async () => {
    const { user } = await openMenu();
    await user.keyboard('{Escape}');
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
  });

  it('a click outside closes it', async () => {
    const { user } = await openMenu();
    await user.click(document.body);
    expect(screen.queryByRole('menu')).not.toBeInTheDocument();
  });

  it('編輯 calls onEdit', async () => {
    const { user, onEdit } = await openMenu();
    await user.click(screen.getByRole('menuitem', { name: '編輯' }));
    expect(onEdit).toHaveBeenCalledTimes(1);
  });

  it('刪除 asks for confirmation', async () => {
    const { user } = await openMenu();
    await user.click(screen.getByRole('menuitem', { name: '刪除' }));
    expect(screen.getByTestId('confirm-delete-button')).toBeInTheDocument();
    fireEvent.click(screen.getByText('取消'));
    expect(screen.queryByTestId('confirm-delete-button')).not.toBeInTheDocument();
  });
});
