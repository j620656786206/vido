import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
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
  return render(
    <QueryClientProvider client={queryClient}>
      <GlossaryPanelV2 mediaId="42" mediaTitle="怪奇物語" open onOpenChange={() => undefined} />
    </QueryClientProvider>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
});

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
});
