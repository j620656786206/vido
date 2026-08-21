/**
 * LibraryEditModal label-association coverage (retro-11-AI1b).
 * Asserts the jsx-a11y htmlFor/id fixes: every visible form label resolves to
 * its control via getByLabelText, and icon-only buttons carry accessible names.
 */
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { LibraryEditModal } from './LibraryEditModal';
import { useMediaLibraries } from '../../hooks/useMediaLibrary';

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

// Same referential-stability rule as librariesQuery above: one constant per
// capability state, never a literal rebuilt inside a test.
const unsupportedQuery = {
  data: { ...librariesQuery.data, autoSubtitleSupported: false },
};

const supportedQuery = {
  data: { ...librariesQuery.data, autoSubtitleSupported: true },
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
    // The component renders more than once (the hydrate effect re-renders it),
    // so the capability must be pinned for the whole test, not handed out once.
    vi.mocked(useMediaLibraries).mockReturnValue(
      librariesQuery as ReturnType<typeof useMediaLibraries>
    );
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

  // ─── CR M2: create mode must carry the opt-in too ──────────────────────

  it('sends autoSubtitle when CREATING a library, not only when editing', async () => {
    // Regression pin: the create branch used to drop the field entirely, so a
    // user could tick the box, press 建立, and silently get a library that was
    // off — no error, nothing on screen to say the choice had been discarded.
    render(<LibraryEditModal onClose={vi.fn()} />);

    fireEvent.change(screen.getByTestId('library-name-input'), { target: { value: '新媒體庫' } });
    fireEvent.click(screen.getByTestId('library-auto-subtitle-checkbox'));
    fireEvent.click(screen.getByTestId('library-save-button'));

    await vi.waitFor(() => expect(mutation.mutateAsync).toHaveBeenCalled());
    expect(mutation.mutateAsync).toHaveBeenCalledWith(
      expect.objectContaining({ name: '新媒體庫', autoSubtitle: true })
    );
  });

  it('leaves the opt-in off on create when the box is untouched', async () => {
    render(<LibraryEditModal onClose={vi.fn()} />);

    fireEvent.change(screen.getByTestId('library-name-input'), { target: { value: '新媒體庫' } });
    fireEvent.click(screen.getByTestId('library-save-button'));

    await vi.waitFor(() => expect(mutation.mutateAsync).toHaveBeenCalled());
    expect(mutation.mutateAsync).toHaveBeenCalledWith(
      expect.objectContaining({ autoSubtitle: false })
    );
  });

  // ─── 補審 M4: capability honor ───────────────────────────────────────────
  //
  // The generator that honours this opt-in is built only when the API runs in
  // `pipeline` mode, and the shipped default is `legacy`. Offering a checkbox
  // that does nothing is a promise nothing keeps — the user ticks it, the save
  // succeeds, and no subtitle is ever produced.
  //
  // 9R-10b-M4: the first fix HID the whole field. Honest, but it also stopped
  // telling the user the feature exists and how to switch it on. Sally's
  // 2026-08-21 ruling (design J5-D) replaces hiding with a DISABLED state that
  // names the one thing the user can act on.

  it('keeps the opt-in on screen — disabled — when the deployment cannot honour it', () => {
    vi.mocked(useMediaLibraries).mockReturnValue(
      unsupportedQuery as ReturnType<typeof useMediaLibraries>
    );

    render(<LibraryEditModal libraryId="lib-1" onClose={vi.fn()} />);

    expect(screen.getByTestId('library-auto-subtitle-field')).toBeInTheDocument();
    expect(screen.getByTestId('library-auto-subtitle-checkbox')).toBeDisabled();
  });

  it('says WHY the option is disabled and WHAT to do about it', () => {
    // Both halves matter on their own: "why is it grey" and "what do I type".
    // The second sentence is verbatim from the API's own 409 suggestion
    // (subtitle_pipeline_handler.go:113) so a user who hit that error over the
    // API reads the same words here.
    vi.mocked(useMediaLibraries).mockReturnValue(
      unsupportedQuery as ReturnType<typeof useMediaLibraries>
    );

    render(<LibraryEditModal libraryId="lib-1" onClose={vi.fn()} />);

    const notice = screen.getByTestId('library-auto-subtitle-unsupported-notice');
    expect(notice).toHaveTextContent('字幕生成管線尚未啟用，這個選項無法變更。');
    expect(notice).toHaveTextContent(
      '請將 VIDO_SUBTITLE_PIPELINE_MODE 設為 pipeline 後重啟伺服器。'
    );
  });

  it('puts the notice between the control and the description, in that order', () => {
    // AC #2 ordering, and it has a reason: the eye lands on a greyed control
    // and asks "why", so the answer comes first; the description below then
    // answers "what would it even do", which is what makes the user decide
    // whether to go and change the variable at all. Reversed, the user reads a
    // pitch for a feature before learning they cannot switch it on.
    vi.mocked(useMediaLibraries).mockReturnValue(
      unsupportedQuery as ReturnType<typeof useMediaLibraries>
    );

    render(<LibraryEditModal libraryId="lib-1" onClose={vi.fn()} />);

    const checkbox = screen.getByTestId('library-auto-subtitle-checkbox');
    const notice = screen.getByTestId('library-auto-subtitle-unsupported-notice');
    const description = screen.getByText(
      '影片內建繁體中文字幕會直接沿用，簡體字幕自動轉成繁體。這些都在本機執行，不會產生費用。'
    );

    expect(checkbox.compareDocumentPosition(notice) & Node.DOCUMENT_POSITION_FOLLOWING).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING
    );
    expect(notice.compareDocumentPosition(description) & Node.DOCUMENT_POSITION_FOLLOWING).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING
    );
  });

  it('renders the env var as a copyable monospace token, not prose', () => {
    // It is a string the user must type EXACTLY. Prose styling invites typos.
    vi.mocked(useMediaLibraries).mockReturnValue(
      unsupportedQuery as ReturnType<typeof useMediaLibraries>
    );

    render(<LibraryEditModal libraryId="lib-1" onClose={vi.fn()} />);

    const token = screen.getByTestId('library-auto-subtitle-env-var');
    expect(token).toHaveTextContent('VIDO_SUBTITLE_PIPELINE_MODE');
    expect(token.className).toContain('font-mono');
  });

  it('ties the disabled checkbox to its explanation for screen readers', () => {
    // A disabled input is skipped by the tab order entirely, so without
    // aria-describedby the reason it is disabled never reaches a
    // keyboard/screen-reader user at all.
    vi.mocked(useMediaLibraries).mockReturnValue(
      unsupportedQuery as ReturnType<typeof useMediaLibraries>
    );

    render(<LibraryEditModal libraryId="lib-1" onClose={vi.fn()} />);

    const checkbox = screen.getByTestId('library-auto-subtitle-checkbox');
    const notice = screen.getByTestId('library-auto-subtitle-unsupported-notice');
    expect(notice).toHaveAttribute('id', 'library-auto-subtitle-unsupported-notice');
    expect(checkbox).toHaveAttribute(
      'aria-describedby',
      'library-auto-subtitle-unsupported-notice'
    );
  });

  it('dims the label and the frozen description in the disabled state', () => {
    // The copy itself is FROZEN (2026-08-19 ruling) — only its colour may
    // change. Dimming is what tells the eye the whole block is inert; without
    // it a greyed checkbox sits under full-strength text and reads as a
    // rendering bug rather than a state.
    vi.mocked(useMediaLibraries).mockReturnValue(
      unsupportedQuery as ReturnType<typeof useMediaLibraries>
    );

    render(<LibraryEditModal libraryId="lib-1" onClose={vi.fn()} />);

    const label = screen.getByText('新檔入庫後，自動完成免費的字幕處理');
    const description = screen.getByText(
      '影片內建繁體中文字幕會直接沿用，簡體字幕自動轉成繁體。這些都在本機執行，不會產生費用。'
    );

    expect(label.className).toContain('text-[var(--text-disabled)]');
    expect(description.className).toContain('text-[var(--text-disabled)]');
  });

  it('keeps the label and description at full strength when supported', () => {
    vi.mocked(useMediaLibraries).mockReturnValue(
      supportedQuery as ReturnType<typeof useMediaLibraries>
    );

    render(<LibraryEditModal libraryId="lib-1" onClose={vi.fn()} />);

    expect(screen.getByText('新檔入庫後，自動完成免費的字幕處理').className).toContain(
      'text-[var(--text-primary)]'
    );
  });

  it('shows no unsupported notice when the deployment DOES run the auto lane', () => {
    vi.mocked(useMediaLibraries).mockReturnValue(
      supportedQuery as ReturnType<typeof useMediaLibraries>
    );

    render(<LibraryEditModal libraryId="lib-1" onClose={vi.fn()} />);

    expect(
      screen.queryByTestId('library-auto-subtitle-unsupported-notice')
    ).not.toBeInTheDocument();
    expect(screen.getByTestId('library-auto-subtitle-checkbox')).toBeEnabled();
  });

  it('shows the opt-in when the deployment reports it supported', () => {
    vi.mocked(useMediaLibraries).mockReturnValue(
      supportedQuery as ReturnType<typeof useMediaLibraries>
    );

    render(<LibraryEditModal libraryId="lib-1" onClose={vi.fn()} />);

    expect(screen.getByTestId('library-auto-subtitle-field')).toBeInTheDocument();
  });

  it('keeps the opt-in visible when the API does not report the capability', () => {
    // An API that omits the field is UNKNOWN, not unsupported. Hiding a shipped
    // control on a missing key would be the worse failure of the two.
    render(<LibraryEditModal libraryId="lib-1" onClose={vi.fn()} />);

    expect(screen.getByTestId('library-auto-subtitle-field')).toBeInTheDocument();
  });

  it('omits autoSubtitle from the update payload when unsupported', async () => {
    // Omitted, not `false`: the field is optional on update, so omitting leaves
    // an opt-in made while the pipeline was enabled exactly as the user left it
    // instead of silently clearing it from a screen that never showed it.
    vi.mocked(useMediaLibraries).mockReturnValue(
      unsupportedQuery as ReturnType<typeof useMediaLibraries>
    );

    render(<LibraryEditModal libraryId="lib-1" onClose={vi.fn()} />);
    fireEvent.click(screen.getByTestId('library-save-button'));

    await vi.waitFor(() => expect(mutation.mutateAsync).toHaveBeenCalled());
    expect(mutation.mutateAsync).toHaveBeenCalledWith(
      expect.not.objectContaining({ autoSubtitle: expect.anything() })
    );
  });
});
