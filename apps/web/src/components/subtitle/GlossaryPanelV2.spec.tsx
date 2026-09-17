import { describe, it, expect, vi, beforeEach } from 'vitest';
import { act, render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { GlossaryPanelV2 } from './GlossaryPanelV2';
import { glossaryService, type GlossaryTerm } from '../../services/glossaryService';

vi.mock('../../services/glossaryService', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../../services/glossaryService')>();
  return {
    ...mod,
    glossaryService: {
      listTerms: vi.fn(),
      addTerm: vi.fn(),
      editTerm: vi.fn(),
      confirmTerm: vi.fn(),
      confirmAll: vi.fn(),
      deleteTerm: vi.fn(),
    },
  };
});

const mocked = vi.mocked(glossaryService);

const term = (overrides: Partial<GlossaryTerm> = {}): GlossaryTerm => ({
  id: 't1',
  mediaId: '42',
  termSrc: 'Demogorgon',
  termZh: '魔王獸',
  language: 'zh-Hant',
  source: 'subtitle',
  confirmed: false,
  createdAt: '2026-07-01T00:00:00Z',
  updatedAt: '2026-07-01T00:00:00Z',
  ...overrides,
});

function renderPanel() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const onOpenChange = vi.fn();
  const ui = (open: boolean) => (
    <QueryClientProvider client={queryClient}>
      <GlossaryPanelV2 mediaId="42" mediaTitle="怪奇物語" open={open} onOpenChange={onOpenChange} />
    </QueryClientProvider>
  );
  const utils = render(ui(true));
  return { ...utils, onOpenChange, setOpen: (open: boolean) => utils.rerender(ui(open)) };
}

beforeEach(() => {
  vi.clearAllMocks();
});

/** A promise the test settles by hand — for "while it is still on its way" cases. */
function deferred<T>() {
  let resolve: (value: T) => void = () => undefined;
  let reject: (err: Error) => void = () => undefined;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

/** TanStack runs a mutationFn a tick after `mutate`, so "was NOT called" needs a flush first. */
const flush = () => new Promise((resolve) => setTimeout(resolve, 30));

/** A promise that never settles — keeps a mutation pending so `busy` stays on. */
const pending = () => new Promise<never>(() => undefined);

async function openAddForm(src: string, zh: string) {
  fireEvent.click(await screen.findByTestId('glossary-add-term'));
  fireEvent.change(screen.getByTestId('glossary-add-src'), { target: { value: src } });
  fireEvent.change(screen.getByTestId('glossary-add-zh'), { target: { value: zh } });
}

describe('GlossaryPanelV2', () => {
  it('renders the term list with the footer count（共 N 條 · M 條未確認）', async () => {
    mocked.listTerms.mockResolvedValue([
      term(),
      term({ id: 't2', termSrc: 'Vecna', termZh: '維克那', source: 'manual', confirmed: true }),
    ]);

    renderPanel();

    expect(await screen.findByText('Demogorgon')).toBeInTheDocument();
    expect(screen.getByText('Vecna')).toBeInTheDocument();
    const footer = screen.getByTestId('glossary-footer-count');
    expect(footer).toHaveTextContent('共 2 條 · 1 條未確認');
    expect(mocked.listTerms).toHaveBeenCalledWith('42');
  });

  it('shows the loading skeleton while the list query is in flight', () => {
    mocked.listTerms.mockReturnValue(new Promise(() => undefined) as never);

    renderPanel();

    expect(screen.getByTestId('glossary-loading')).toBeInTheDocument();
  });

  it('empty state is distinct from failure: 尚無詞彙 — 生成字幕時自動累積', async () => {
    mocked.listTerms.mockResolvedValue([]);

    renderPanel();

    expect(await screen.findByTestId('glossary-empty')).toBeInTheDocument();
    expect(screen.getByText('尚無詞彙')).toBeInTheDocument();
    expect(screen.getByText('生成字幕時自動累積')).toBeInTheDocument();
    expect(screen.queryByTestId('glossary-error')).not.toBeInTheDocument();
  });

  it('fail-soft error state renders 重試 and refetches', async () => {
    mocked.listTerms.mockRejectedValueOnce(new Error('boom'));
    mocked.listTerms.mockResolvedValueOnce([term()]);

    renderPanel();

    expect(await screen.findByTestId('glossary-error')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('glossary-retry'));

    expect(await screen.findByText('Demogorgon')).toBeInTheDocument();
  });

  it('全部確認 posts confirm-all and refreshes the list', async () => {
    mocked.listTerms.mockResolvedValue([term()]);
    mocked.confirmAll.mockResolvedValue({ confirmed: 1 });

    renderPanel();

    fireEvent.click(await screen.findByTestId('glossary-confirm-all'));

    await waitFor(() => expect(mocked.confirmAll).toHaveBeenCalledWith('42'));
    // invalidation refetches the list
    await waitFor(() => expect(mocked.listTerms.mock.calls.length).toBeGreaterThanOrEqual(2));
  });

  it('add flow: 新增詞彙 → form → submit posts a manual confirmed term', async () => {
    mocked.listTerms.mockResolvedValue([term()]);
    mocked.addTerm.mockResolvedValue(term({ id: 't9', termSrc: 'Hawkins', termZh: '霍金斯鎮' }));

    renderPanel();

    fireEvent.click(await screen.findByTestId('glossary-add-term'));
    fireEvent.change(screen.getByTestId('glossary-add-src'), { target: { value: 'Hawkins' } });
    fireEvent.change(screen.getByTestId('glossary-add-zh'), { target: { value: '霍金斯鎮' } });
    fireEvent.click(screen.getByTestId('glossary-add-submit'));

    await waitFor(() =>
      expect(mocked.addTerm).toHaveBeenCalledWith('42', {
        termSrc: 'Hawkins',
        termZh: '霍金斯鎮',
        source: 'manual',
        confirmed: true,
      })
    );
  });

  it('per-row confirm posts the confirm route', async () => {
    mocked.listTerms.mockResolvedValue([term()]);
    mocked.confirmTerm.mockResolvedValue(undefined);

    renderPanel();

    fireEvent.click(await screen.findByTestId('glossary-confirm-t1'));

    await waitFor(() => expect(mocked.confirmTerm).toHaveBeenCalledWith('42', 't1'));
  });

  it('per-row delete (after Radix confirm) calls the delete route', async () => {
    mocked.listTerms.mockResolvedValue([term()]);
    mocked.deleteTerm.mockResolvedValue(undefined);

    renderPanel();

    fireEvent.click(await screen.findByTestId('glossary-delete-t1'));
    fireEvent.click(screen.getByTestId('glossary-delete-confirm-t1'));

    await waitFor(() => expect(mocked.deleteTerm).toHaveBeenCalledWith('42', 't1'));
  });

  it('per-row edit PUTs {termZh, confirmed} preserving the row confirmed flag', async () => {
    mocked.listTerms.mockResolvedValue([term({ confirmed: true })]);
    mocked.editTerm.mockResolvedValue(undefined);

    renderPanel();

    fireEvent.click(await screen.findByTestId('glossary-edit-t1'));
    fireEvent.change(screen.getByTestId('glossary-edit-input-t1'), {
      target: { value: '魔神獸' },
    });
    fireEvent.click(screen.getByTestId('glossary-save-t1'));

    await waitFor(() =>
      expect(mocked.editTerm).toHaveBeenCalledWith('42', 't1', {
        termZh: '魔神獸',
        confirmed: true,
      })
    );
  });
  describe('dsr-6c alignment (F6-D-v2 dlfMR / F7-D-v2 A85GFD)', () => {
    it('the panel is 880 wide with radius-lg and a hairline frame on sm+ only', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      renderPanel();

      const panel = await screen.findByTestId('glossary-panel-v2');
      expect(panel).toHaveClass(
        'max-w-[880px]',
        'sm:rounded-[var(--radius-lg)]',
        'sm:border',
        'sm:border-[var(--border-subtle)]'
      );
      expect(panel).not.toHaveClass('max-w-3xl');
    });

    it('全部確認 and 新增詞彙 are Secondary buttons with no icon', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      renderPanel();

      for (const testId of ['glossary-confirm-all', 'glossary-add-term']) {
        const button = await screen.findByTestId(testId);
        expect(button.querySelector('svg')).toBeNull();
        expect(button).toHaveClass('px-5');
      }
    });

    it('the empty-state 新增詞彙 has no icon either', async () => {
      mocked.listTerms.mockResolvedValue([]);
      renderPanel();

      const button = await screen.findByTestId('glossary-empty-add');
      expect(button.querySelector('svg')).toBeNull();
      expect(button).toHaveClass('px-5');
    });
  });

  describe('write failures say so (dsr-6c AC #5)', () => {
    it('a failed add names the term, keeps the form open and the drafts', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      mocked.addTerm.mockRejectedValue(new Error('500'));
      renderPanel();

      await openAddForm('Hawkins', '霍金斯鎮');
      fireEvent.click(screen.getByTestId('glossary-add-submit'));

      const alert = await screen.findByTestId('glossary-write-error');
      expect(alert).toHaveAttribute('role', 'alert');
      expect(alert).toHaveTextContent(/^新增「Hawkins」失敗，請再試一次$/);
      expect(screen.getByTestId('glossary-add-src')).toHaveValue('Hawkins');
      expect(screen.getByTestId('glossary-add-zh')).toHaveValue('霍金斯鎮');
    });

    it('the write error is the error-tint box of F6-SPEC-STATES ③, docked so it stays in view (CR LOW-4, LOW-7)', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      mocked.confirmTerm.mockRejectedValue(new Error('500'));
      renderPanel();

      fireEvent.click(await screen.findByTestId('glossary-confirm-t1'));

      const alert = await screen.findByTestId('glossary-write-error');
      expect(alert).toHaveClass(
        'flex',
        'items-center',
        'gap-2',
        'rounded-[var(--radius-md)]',
        'bg-[var(--error-tint)]',
        'p-3'
      );
      expect(alert.querySelector('svg')).toHaveClass('text-[var(--error-text)]');
      expect(alert.querySelector('p')).toHaveClass('text-sm', 'text-[var(--error-text)]');
      expect(screen.getByTestId('glossary-write-error-dock')).toHaveClass(
        'sticky',
        'top-0',
        'bg-[var(--bg-secondary)]'
      );
      expect(screen.getByTestId('glossary-write-error-dock')).toContainElement(alert);
    });

    it('a row save still in flight: Esc waits, and the failure keeps the typed text (CR MED-1)', async () => {
      const save = deferred<void>();
      mocked.listTerms.mockResolvedValue([term()]);
      mocked.editTerm.mockReturnValue(save.promise);
      const { onOpenChange } = renderPanel();

      fireEvent.click(await screen.findByTestId('glossary-edit-t1'));
      const input = screen.getByTestId('glossary-edit-input-t1');
      fireEvent.change(input, { target: { value: '魔神獸' } });
      fireEvent.keyDown(input, { key: 'Enter' });
      await waitFor(() => expect(mocked.editTerm).toHaveBeenCalledTimes(1));

      fireEvent.keyDown(input, { key: 'Escape' });
      expect(screen.getByTestId('glossary-edit-input-t1')).toHaveValue('魔神獸');
      expect(onOpenChange).not.toHaveBeenCalledWith(false);

      await act(async () => save.reject(new Error('500')));
      expect(await screen.findByTestId('glossary-write-error')).toHaveTextContent(
        /^「Demogorgon」的新譯名沒有存到，請再試一次$/
      );
      expect(screen.getByTestId('glossary-edit-input-t1')).toHaveValue('魔神獸');
    });

    it('an add still in flight: Esc and 取消 wait, and the failure keeps the drafts (CR MED-1)', async () => {
      const addCall = deferred<GlossaryTerm>();
      mocked.listTerms.mockResolvedValue([term()]);
      mocked.addTerm.mockReturnValue(addCall.promise);
      renderPanel();

      await openAddForm('Hawkins', '霍金斯鎮');
      fireEvent.keyDown(screen.getByTestId('glossary-add-zh'), { key: 'Enter' });
      await waitFor(() => expect(mocked.addTerm).toHaveBeenCalledTimes(1));

      fireEvent.keyDown(screen.getByTestId('glossary-add-zh'), { key: 'Escape' });
      fireEvent.click(screen.getByTestId('glossary-add-cancel'));
      expect(screen.getByTestId('glossary-add-form')).toBeInTheDocument();
      expect(screen.getByTestId('glossary-add-cancel')).toHaveAttribute('aria-disabled', 'true');

      await act(async () => addCall.reject(new Error('500')));
      expect(await screen.findByTestId('glossary-write-error')).toHaveTextContent(
        /^新增「Hawkins」失敗，請再試一次$/
      );
      expect(screen.getByTestId('glossary-add-src')).toHaveValue('Hawkins');
      expect(screen.getByTestId('glossary-add-zh')).toHaveValue('霍金斯鎮');
    });

    it('a write that fails after the panel was closed and reopened does not report into the new session (CR LOW-3)', async () => {
      const confirmCall = deferred<void>();
      mocked.listTerms.mockResolvedValue([term()]);
      mocked.confirmTerm.mockReturnValue(confirmCall.promise);
      const { setOpen } = renderPanel();

      fireEvent.click(await screen.findByTestId('glossary-confirm-t1'));
      await waitFor(() => expect(mocked.confirmTerm).toHaveBeenCalledTimes(1));
      await act(async () => setOpen(false));
      await act(async () => setOpen(true));

      await act(async () => confirmCall.reject(new Error('500')));
      await flush();
      expect(screen.queryByTestId('glossary-write-error')).not.toBeInTheDocument();
    });

    it('a failed edit names the term and the row keeps the typed text', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      mocked.editTerm.mockRejectedValue(new Error('500'));
      renderPanel();

      fireEvent.click(await screen.findByTestId('glossary-edit-t1'));
      fireEvent.change(screen.getByTestId('glossary-edit-input-t1'), {
        target: { value: '魔神獸' },
      });
      fireEvent.click(screen.getByTestId('glossary-save-t1'));

      expect(await screen.findByTestId('glossary-write-error')).toHaveTextContent(
        /^「Demogorgon」的新譯名沒有存到，請再試一次$/
      );
      expect(screen.getByTestId('glossary-edit-input-t1')).toHaveValue('魔神獸');
    });

    it('a failed confirm names the term', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      mocked.confirmTerm.mockRejectedValue(new Error('500'));
      renderPanel();

      fireEvent.click(await screen.findByTestId('glossary-confirm-t1'));

      expect(await screen.findByTestId('glossary-write-error')).toHaveTextContent(
        /^確認「Demogorgon」失敗，請再試一次$/
      );
    });

    it('a failed 全部確認 says so', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      mocked.confirmAll.mockRejectedValue(new Error('500'));
      renderPanel();

      fireEvent.click(await screen.findByTestId('glossary-confirm-all'));

      expect(await screen.findByTestId('glossary-write-error')).toHaveTextContent(
        /^全部確認失敗，請再試一次$/
      );
    });

    it('a failed delete names the term', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      mocked.deleteTerm.mockRejectedValue(new Error('500'));
      renderPanel();

      fireEvent.click(await screen.findByTestId('glossary-delete-t1'));
      fireEvent.click(screen.getByTestId('glossary-delete-confirm-t1'));

      expect(await screen.findByTestId('glossary-write-error')).toHaveTextContent(
        /^刪除「Demogorgon」失敗，請再試一次$/
      );
    });

    it('the message clears as soon as the next write starts', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      mocked.confirmTerm.mockRejectedValueOnce(new Error('500')).mockReturnValueOnce(pending());
      renderPanel();

      fireEvent.click(await screen.findByTestId('glossary-confirm-t1'));
      await screen.findByTestId('glossary-write-error');

      fireEvent.click(screen.getByTestId('glossary-confirm-t1'));
      await waitFor(() =>
        expect(screen.queryByTestId('glossary-write-error')).not.toBeInTheDocument()
      );
    });
  });

  describe('Esc only cancels what is in progress (dsr-6c AC #6, real Radix Dialog)', () => {
    async function startRowEdit() {
      fireEvent.click(await screen.findByTestId('glossary-edit-t1'));
      const input = screen.getByTestId('glossary-edit-input-t1');
      fireEvent.change(input, { target: { value: '魔神獸' } });
      return input;
    }

    it('① Esc in the row edit input cancels the edit and keeps the panel open', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      const { onOpenChange } = renderPanel();

      fireEvent.keyDown(await startRowEdit(), { key: 'Escape' });

      expect(onOpenChange).not.toHaveBeenCalledWith(false);
      expect(screen.queryByTestId('glossary-edit-input-t1')).not.toBeInTheDocument();
      expect(screen.getByText('魔王獸')).toBeInTheDocument();
    });

    it('② Esc with focus on 儲存 also cancels the edit and keeps the panel open', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      const { onOpenChange } = renderPanel();

      await startRowEdit();
      const save = screen.getByTestId('glossary-save-t1');
      save.focus();
      fireEvent.keyDown(save, { key: 'Escape' });

      expect(onOpenChange).not.toHaveBeenCalledWith(false);
      expect(screen.queryByTestId('glossary-edit-input-t1')).not.toBeInTheDocument();
    });

    it('③ Esc on the add form 取消 closes the form, not the panel', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      const { onOpenChange } = renderPanel();

      await openAddForm('Hawkins', '霍金斯鎮');
      fireEvent.keyDown(screen.getByTestId('glossary-add-cancel'), { key: 'Escape' });

      expect(onOpenChange).not.toHaveBeenCalledWith(false);
      expect(screen.queryByTestId('glossary-add-form')).not.toBeInTheDocument();
    });

    it('an Esc that only drops an IME candidate in the add form cancels nothing (CR LOW-4)', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      const { onOpenChange } = renderPanel();

      await openAddForm('Hawkins', '霍金ㄙ');
      fireEvent.keyDown(screen.getByTestId('glossary-add-zh'), {
        key: 'Escape',
        isComposing: true,
      });

      expect(onOpenChange).not.toHaveBeenCalledWith(false);
      expect(screen.getByTestId('glossary-add-zh')).toHaveValue('霍金ㄙ');
    });

    it('Esc inside the delete confirm only closes that dialog — the add form and the panel stay (CR LOW-4)', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      const { onOpenChange } = renderPanel();

      await openAddForm('Hawkins', '霍金斯鎮');
      fireEvent.click(screen.getByTestId('glossary-delete-t1'));
      fireEvent.keyDown(screen.getByTestId('glossary-delete-confirm-t1'), { key: 'Escape' });

      await waitFor(() =>
        expect(screen.queryByTestId('glossary-delete-dialog-t1')).not.toBeInTheDocument()
      );
      expect(onOpenChange).not.toHaveBeenCalledWith(false);
      expect(screen.getByTestId('glossary-add-src')).toHaveValue('Hawkins');
      expect(mocked.deleteTerm).not.toHaveBeenCalled();
    });

    it('④ an Esc that only drops an IME candidate cancels nothing', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      const { onOpenChange } = renderPanel();

      fireEvent.keyDown(await startRowEdit(), { key: 'Escape', isComposing: true });

      expect(onOpenChange).not.toHaveBeenCalledWith(false);
      expect(screen.getByTestId('glossary-edit-input-t1')).toHaveValue('魔神獸');
    });

    it('⑤ Esc outside any editor closes the panel', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      const { onOpenChange } = renderPanel();

      fireEvent.keyDown(await screen.findByTestId('glossary-edit-t1'), { key: 'Escape' });

      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });

  describe('Enter (dsr-6c AC #7)', () => {
    it.each([
      ['isComposing', { key: 'Enter', isComposing: true }],
      ['keyCode 229 (Safari)', { key: 'Enter', keyCode: 229 }],
    ])('an Enter that only picks an IME candidate (%s) does not add', async (_label, init) => {
      mocked.listTerms.mockResolvedValue([term()]);
      renderPanel();

      await openAddForm('Hawkins', '霍金ㄙ');
      fireEvent.keyDown(screen.getByTestId('glossary-add-zh'), init);

      await flush();
      expect(mocked.addTerm).not.toHaveBeenCalled();
    });

    it('a plain Enter adds', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      mocked.addTerm.mockReturnValue(pending());
      renderPanel();

      await openAddForm('Hawkins', '霍金斯鎮');
      fireEvent.keyDown(screen.getByTestId('glossary-add-zh'), { key: 'Enter' });

      await waitFor(() => expect(mocked.addTerm).toHaveBeenCalledTimes(1));
    });

    it('Enter while another write is in flight does not add', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      mocked.confirmTerm.mockReturnValue(pending());
      renderPanel();

      await openAddForm('Hawkins', '霍金斯鎮');
      fireEvent.click(screen.getByTestId('glossary-confirm-t1'));
      await waitFor(() =>
        expect(screen.getByTestId('glossary-add-submit')).toHaveAttribute('aria-disabled', 'true')
      );

      fireEvent.keyDown(screen.getByTestId('glossary-add-zh'), { key: 'Enter' });
      await flush();
      expect(mocked.addTerm).not.toHaveBeenCalled();
    });

    it('Enter in a row edit while another write is in flight does not save (CR LOW-4)', async () => {
      mocked.listTerms.mockResolvedValue([
        term(),
        term({ id: 't2', termSrc: 'Vecna', termZh: '維克那' }),
      ]);
      mocked.confirmTerm.mockReturnValue(pending());
      renderPanel();

      fireEvent.click(await screen.findByTestId('glossary-edit-t1'));
      const input = screen.getByTestId('glossary-edit-input-t1');
      fireEvent.change(input, { target: { value: '魔神獸' } });
      fireEvent.click(screen.getByTestId('glossary-confirm-t2'));
      await waitFor(() =>
        expect(screen.getByTestId('glossary-save-t1')).toHaveAttribute('aria-disabled', 'true')
      );

      fireEvent.keyDown(input, { key: 'Enter' });
      await flush();
      expect(mocked.editTerm).not.toHaveBeenCalled();
    });

    it('the add form takes focus on the 原文 input when it opens', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      renderPanel();

      fireEvent.click(await screen.findByTestId('glossary-add-term'));
      expect(screen.getByTestId('glossary-add-src')).toHaveFocus();
    });
  });

  describe('adding a term that is already in the table (dsr-6c AC #8)', () => {
    it('blocks it (ASCII case-insensitive, like SQLite NOCASE) and points at 編輯', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      renderPanel();

      await openAddForm('  demogorgon ', '魔神獸');
      fireEvent.click(screen.getByTestId('glossary-add-submit'));

      const note = await screen.findByTestId('glossary-add-duplicate');
      expect(note).toHaveTextContent(
        /^「Demogorgon」已經在表裡了（→ 魔王獸）。要改譯名，請按那一列的「編輯」。$/
      );
      expect(note).toHaveClass('text-sm', 'text-[var(--warning-text)]');
      await flush();
      expect(mocked.addTerm).not.toHaveBeenCalled();
    });

    it('a second blocked submit remounts the note so a screen reader announces it again (CR LOW-8)', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      renderPanel();

      await openAddForm('Demogorgon', '魔神獸');
      fireEvent.click(screen.getByTestId('glossary-add-submit'));
      const first = await screen.findByTestId('glossary-add-duplicate');

      fireEvent.click(screen.getByTestId('glossary-add-submit'));
      expect(screen.getByTestId('glossary-add-duplicate')).not.toBe(first);
    });

    it('the note follows the live list: once that row is deleted it goes, and a failed add is the only alert (CR LOW-2)', async () => {
      mocked.listTerms.mockResolvedValueOnce([term()]).mockResolvedValue([]);
      mocked.deleteTerm.mockResolvedValue(undefined);
      mocked.addTerm.mockRejectedValue(new Error('500'));
      renderPanel();

      await openAddForm('Demogorgon', '魔神獸');
      fireEvent.click(screen.getByTestId('glossary-add-submit'));
      await screen.findByTestId('glossary-add-duplicate');

      fireEvent.click(screen.getByTestId('glossary-delete-t1'));
      fireEvent.click(screen.getByTestId('glossary-delete-confirm-t1'));
      await waitFor(() =>
        expect(screen.queryByTestId('glossary-add-duplicate')).not.toBeInTheDocument()
      );

      fireEvent.click(screen.getByTestId('glossary-add-submit'));
      await screen.findByTestId('glossary-write-error');
      expect(screen.getAllByRole('alert')).toHaveLength(1);
    });

    it('取消 also clears the note — reopening the form starts without it (CR LOW-4)', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      renderPanel();

      await openAddForm('Demogorgon', '魔神獸');
      fireEvent.click(screen.getByTestId('glossary-add-submit'));
      await screen.findByTestId('glossary-add-duplicate');

      fireEvent.click(screen.getByTestId('glossary-add-cancel'));
      fireEvent.click(screen.getByTestId('glossary-add-term'));
      expect(screen.queryByTestId('glossary-add-duplicate')).not.toBeInTheDocument();
    });

    it('clears the note as soon as the 原文 changes', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      renderPanel();

      await openAddForm('Demogorgon', '魔神獸');
      fireEvent.click(screen.getByTestId('glossary-add-submit'));
      await screen.findByTestId('glossary-add-duplicate');

      fireEvent.change(screen.getByTestId('glossary-add-src'), {
        target: { value: 'Demogorgons' },
      });
      expect(screen.queryByTestId('glossary-add-duplicate')).not.toBeInTheDocument();
    });

    it('does not fold non-ASCII case (É vs é are different rows to SQLite NOCASE)', async () => {
      mocked.listTerms.mockResolvedValue([term({ termSrc: 'Élan', termZh: '活力' })]);
      mocked.addTerm.mockReturnValue(pending());
      renderPanel();

      await openAddForm('élan', '衝勁');
      fireEvent.click(screen.getByTestId('glossary-add-submit'));

      await waitFor(() => expect(mocked.addTerm).toHaveBeenCalledTimes(1));
      expect(screen.queryByTestId('glossary-add-duplicate')).not.toBeInTheDocument();
    });

    it('does not block a term stored under another language', async () => {
      mocked.listTerms.mockResolvedValue([term({ language: 'ja' })]);
      mocked.addTerm.mockReturnValue(pending());
      renderPanel();

      await openAddForm('Demogorgon', '魔神獸');
      fireEvent.click(screen.getByTestId('glossary-add-submit'));

      await waitFor(() => expect(mocked.addTerm).toHaveBeenCalledTimes(1));
    });

    it('does not block while the list never loaded (nothing to compare against)', async () => {
      mocked.listTerms.mockRejectedValue(new Error('boom'));
      mocked.addTerm.mockReturnValue(pending());
      renderPanel();

      await screen.findByTestId('glossary-error');
      await openAddForm('Demogorgon', '魔神獸');
      fireEvent.click(screen.getByTestId('glossary-add-submit'));

      await waitFor(() => expect(mocked.addTerm).toHaveBeenCalledTimes(1));
    });
  });

  describe('state fixes (dsr-6c AC #9)', () => {
    it('a failed background refetch keeps the loaded list and shows the error above it', async () => {
      mocked.listTerms.mockResolvedValueOnce([term()]).mockRejectedValue(new Error('boom'));
      mocked.confirmTerm.mockResolvedValue(undefined);
      renderPanel();

      fireEvent.click(await screen.findByTestId('glossary-confirm-t1'));

      expect(await screen.findByTestId('glossary-error')).toBeInTheDocument();
      expect(screen.getByTestId('glossary-list')).toBeInTheDocument();
      expect(screen.getByText('Demogorgon')).toBeInTheDocument();
      // The error sits above the list.
      expect(
        screen
          .getByTestId('glossary-error')
          .compareDocumentPosition(screen.getByTestId('glossary-list')) &
          Node.DOCUMENT_POSITION_FOLLOWING
      ).toBeTruthy();
    });

    it('an empty list whose refetch failed shows exactly one 新增詞彙', async () => {
      mocked.listTerms.mockResolvedValueOnce([]).mockRejectedValue(new Error('boom'));
      mocked.addTerm.mockResolvedValue(term({ id: 't9' }));
      renderPanel();

      fireEvent.click(await screen.findByTestId('glossary-empty-add'));
      fireEvent.change(screen.getByTestId('glossary-add-src'), { target: { value: 'Hawkins' } });
      fireEvent.change(screen.getByTestId('glossary-add-zh'), { target: { value: '霍金斯鎮' } });
      fireEvent.click(screen.getByTestId('glossary-add-submit'));

      await screen.findByTestId('glossary-error');
      expect(screen.getAllByRole('button', { name: '新增詞彙' })).toHaveLength(1);
    });

    it('while adding into an empty list, the 尚無詞彙 illustration steps aside', async () => {
      mocked.listTerms.mockResolvedValue([]);
      renderPanel();

      fireEvent.click(await screen.findByTestId('glossary-empty-add'));

      expect(screen.getByTestId('glossary-add-form')).toBeInTheDocument();
      expect(screen.queryByTestId('glossary-empty')).not.toBeInTheDocument();
    });

    it('取消 clears the drafts', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      renderPanel();

      await openAddForm('Hawkins', '霍金斯鎮');
      fireEvent.click(screen.getByTestId('glossary-add-cancel'));
      fireEvent.click(screen.getByTestId('glossary-add-term'));

      expect(screen.getByTestId('glossary-add-src')).toHaveValue('');
      expect(screen.getByTestId('glossary-add-zh')).toHaveValue('');
    });

    it('closing and reopening the panel starts clean (no open form, drafts or old error)', async () => {
      mocked.listTerms.mockResolvedValue([term()]);
      mocked.confirmTerm.mockRejectedValue(new Error('500'));
      const { setOpen } = renderPanel();

      fireEvent.click(await screen.findByTestId('glossary-confirm-t1'));
      await screen.findByTestId('glossary-write-error');
      await openAddForm('Hawkins', '霍金斯鎮');

      setOpen(false);
      setOpen(true);

      await screen.findByTestId('glossary-list');
      expect(screen.queryByTestId('glossary-add-form')).not.toBeInTheDocument();
      expect(screen.queryByTestId('glossary-write-error')).not.toBeInTheDocument();
      fireEvent.click(screen.getByTestId('glossary-add-term'));
      expect(screen.getByTestId('glossary-add-src')).toHaveValue('');
    });
  });
});
