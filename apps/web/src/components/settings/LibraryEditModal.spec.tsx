/**
 * LibraryEditModal label-association coverage (retro-11-AI1b).
 * Asserts the jsx-a11y htmlFor/id fixes: every visible form label resolves to
 * its control via getByLabelText, and icon-only buttons carry accessible names.
 */
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { LibraryEditModal } from './LibraryEditModal';

const mutation = { mutateAsync: vi.fn().mockResolvedValue({}), isPending: false };

// The mocked query result is a MODULE-LEVEL constant, not a fresh literal per
// call. TanStack Query hands back referentially stable data between renders, and
// LibraryEditModal's hydrate effect keys on that identity — a mock that rebuilt
// the object every render would re-run the effect on every render and silently
// reset form state mid-test, which looks exactly like a broken component.
const librariesQuery = {
  data: {
    libraries: [
      {
        id: 'lib-1',
        name: '我的電影',
        contentType: 'movie' as const,
        autoSubtitle: false,
        paths: [{ id: 'path-1', path: '/media/movies' }],
      },
    ],
  },
};

vi.mock('../../hooks/useMediaLibrary', () => ({
  useMediaLibraries: vi.fn(() => librariesQuery),
  useCreateLibrary: () => mutation,
  useUpdateLibrary: () => mutation,
  useAddLibraryPath: () => mutation,
  useRemoveLibraryPath: () => mutation,
}));

describe('LibraryEditModal', () => {
  beforeEach(() => {
    mutation.mutateAsync.mockClear();
  });

  it('associates every form label with its control (retro-11-AI1b htmlFor/id)', () => {
    render(<LibraryEditModal onClose={vi.fn()} />);

    expect(screen.getByLabelText('名稱')).toBe(screen.getByTestId('library-name-input'));
    expect(screen.getByLabelText('類型')).toBe(screen.getByTestId('library-type-select'));
    expect(screen.getByLabelText('資料夾路徑')).toBe(screen.getByTestId('library-path-input'));
  });

  it('gives icon-only buttons accessible names (retro-11-AI1b)', () => {
    render(<LibraryEditModal libraryId="lib-1" onClose={vi.fn()} />);

    expect(screen.getByRole('button', { name: '關閉' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '移除路徑 /media/movies' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '新增路徑' })).toBeInTheDocument();
  });

  // ─── Story 9R-10b AC #2: the free auto-generation opt-in ────────────────
  //
  // The copy is FROZEN by the 2026-08-19 ruling 「9R-10b 花錢須同意」 and
  // defined on design screen E5-D (hUVYm). It has to say two things the user
  // can act on — only free work happens automatically, and anything that costs
  // money waits for them — without ever implying that scanning itself produces
  // subtitles (the 2026-08-07 misunderstanding). These assertions exist so a
  // future copy edit has to be a deliberate one.

  it('renders the auto-subtitle opt-in with its label bound to the control', () => {
    render(<LibraryEditModal libraryId="lib-1" onClose={vi.fn()} />);

    const checkbox = screen.getByTestId('library-auto-subtitle-checkbox');
    expect(screen.getByLabelText('新檔入庫後，自動完成免費的字幕處理')).toBe(checkbox);
    expect(checkbox).not.toBeChecked();
  });

  it('states what runs free and what waits for consent', () => {
    render(<LibraryEditModal libraryId="lib-1" onClose={vi.fn()} />);

    expect(
      screen.getByText(
        '影片內建繁體中文字幕會直接沿用，簡體字幕自動轉成繁體。這些都在本機執行，不會產生費用。'
      )
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        '需要 AI 翻譯或語音辨識的影片不會自動處理，它們會留在「產生字幕」清單裡，標好預估金額等你確認。'
      )
    ).toBeInTheDocument();
  });

  it('never implies that scanning itself generates subtitles (sub-4-3 AC #6)', () => {
    const { container } = render(<LibraryEditModal libraryId="lib-1" onClose={vi.fn()} />);

    const block = container.querySelector('[data-testid="library-auto-subtitle-field"]');
    expect(block?.textContent).not.toContain('掃描');
  });

  it('places the opt-in LAST, after the fields that describe the library itself', () => {
    // Design E5-D (hUVYm) orders the modal 名稱 → 類型 → 資料夾路徑 → opt-in.
    // The first three say what the library IS; the opt-in says what Vido DOES
    // with it, and reads as a separate commitment rather than another attribute.
    render(<LibraryEditModal libraryId="lib-1" onClose={vi.fn()} />);

    const pathInput = screen.getByTestId('library-path-input');
    const optIn = screen.getByTestId('library-auto-subtitle-field');

    expect(
      pathInput.compareDocumentPosition(optIn) & Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy();
  });

  it('sends auto_subtitle in the update payload when toggled on', async () => {
    render(<LibraryEditModal libraryId="lib-1" onClose={vi.fn()} />);

    fireEvent.click(screen.getByTestId('library-auto-subtitle-checkbox'));
    fireEvent.click(screen.getByTestId('library-save-button'));

    await vi.waitFor(() => expect(mutation.mutateAsync).toHaveBeenCalled());
    expect(mutation.mutateAsync).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'lib-1', autoSubtitle: true })
    );
  });
});
