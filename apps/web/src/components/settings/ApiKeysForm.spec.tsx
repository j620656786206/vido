import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { KeySettings } from '../../services/keySettingsService';

const h = vi.hoisted(() => ({
  query: {
    data: undefined as KeySettings | undefined,
    isLoading: false,
    isError: false,
    error: null as Error | null,
  },
  saveMutate: vi.fn(),
  testMutate: vi.fn(),
  refetch: vi.fn(),
}));

vi.mock('../../hooks/useKeySettings', () => ({
  useKeySettings: () => ({ refetch: h.refetch, isFetching: false, ...h.query }),
  useSaveKeys: () => ({ mutate: h.saveMutate, isPending: false }),
  useTestClaudeKey: () => ({ mutate: h.testMutate, isPending: false }),
}));

import { ApiKeysForm } from './ApiKeysForm';

/** All three keys unset and storage available — the first-run baseline. */
const ALL_NONE: KeySettings = {
  writable: true,
  keys: [
    { name: 'claude', configured: false, source: 'none' },
    { name: 'tmdb', configured: false, source: 'none' },
    { name: 'openai', configured: false, source: 'none' },
  ],
};

/** One of each source, so the three distinct affordances are visible at once. */
const MIXED_SOURCES: KeySettings = {
  writable: true,
  keys: [
    { name: 'claude', configured: true, source: 'secret', masked: 'sk-ant…7f3a' },
    { name: 'tmdb', configured: true, source: 'env' },
    { name: 'openai', configured: false, source: 'none' },
  ],
};

function setSecureContext(value: boolean) {
  Object.defineProperty(window, 'isSecureContext', {
    value,
    configurable: true,
    writable: true,
  });
}

function renderForm(ui: React.ReactElement = <ApiKeysForm />) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

beforeEach(() => {
  vi.clearAllMocks();
  // clearAllMocks keeps implementations; these mocks get per-test ones, so they
  // must be reset or a callback stub leaks into every later test.
  h.saveMutate.mockReset();
  h.testMutate.mockReset();
  h.refetch.mockReset();
  h.query = { data: ALL_NONE, isLoading: false, isError: false, error: null };
  // Default to a secure context so only the NFR-S3 tests exercise the warning.
  setSecureContext(true);
});

afterEach(() => {
  setSecureContext(true);
});

describe('ApiKeysForm — the three source states (AC #1)', () => {
  it('renders one row per key with the AC-specified labels', () => {
    renderForm();

    expect(screen.getByTestId('key-row-claude')).toBeInTheDocument();
    expect(screen.getByTestId('key-row-tmdb')).toBeInTheDocument();
    expect(screen.getByTestId('key-row-openai')).toBeInTheDocument();
    expect(screen.getByLabelText('Claude（翻譯）')).toBeInTheDocument();
    expect(screen.getByLabelText('TMDB')).toBeInTheDocument();
    expect(screen.getByLabelText('雲端 ASR（選配）')).toBeInTheDocument();
  });

  it('source=secret shows 已設定 + the mask + 編輯/清除 instead of an input', () => {
    h.query.data = MIXED_SOURCES;
    renderForm();

    expect(screen.getByTestId('key-state-claude')).toHaveTextContent('已設定');
    expect(screen.getByTestId('key-masked-claude')).toHaveTextContent('sk-ant…7f3a');
    expect(screen.getByTestId('key-edit-claude')).toBeInTheDocument();
    expect(screen.getByTestId('key-clear-claude')).toBeInTheDocument();
    // No editable field is drawn until 編輯 is pressed.
    expect(screen.queryByLabelText('Claude（翻譯）')).toBeNull();
  });

  it('source=env is honest about precedence: it says so AND warns that saving overrides', () => {
    h.query.data = MIXED_SOURCES;
    renderForm();

    expect(screen.getByTestId('key-state-tmdb')).toHaveTextContent('目前由環境變數提供');
    expect(screen.getByTestId('key-env-override-note-tmdb')).toHaveTextContent(
      '在此儲存的金鑰會覆蓋環境變數提供的設定。'
    );
    expect(screen.getByLabelText('TMDB')).toBeInTheDocument();
  });

  it('source=none shows 尚未設定 with an empty input and no override note', () => {
    h.query.data = MIXED_SOURCES;
    renderForm();

    expect(screen.getByTestId('key-state-openai')).toHaveTextContent('尚未設定');
    expect(screen.getByLabelText('雲端 ASR（選配）')).toHaveValue('');
    expect(screen.queryByTestId('key-env-override-note-openai')).toBeNull();
  });

  it('every input is a password field, and 編輯 never seeds the mask into it', () => {
    h.query.data = MIXED_SOURCES;
    renderForm();

    fireEvent.click(screen.getByTestId('key-edit-claude'));

    const input = screen.getByLabelText('Claude（翻譯）');
    expect(input).toHaveAttribute('type', 'password');
    // The mask is a display hint, never a value — 2-1a returns no value to seed.
    expect(input).toHaveValue('');
    expect(screen.getByLabelText('TMDB')).toHaveAttribute('type', 'password');
    expect(screen.getByLabelText('雲端 ASR（選配）')).toHaveAttribute('type', 'password');
  });

  it('the TMDb row states the restart caveat (backlog-tmdb-runtime-key-resolution)', () => {
    renderForm();
    expect(screen.getByTestId('key-row-tmdb')).toHaveTextContent('儲存後需重啟伺服器才會生效');
  });

  // sub-5-2 AC #5: the cloud-ASR key hot-reloads now, exactly like Claude. The
  // TMDb row above is the deliberate contrast — it is the ONLY row that still
  // needs a restart, so these two tests together pin that the copy distinguishes
  // them instead of blanket-claiming either behaviour.
  it('the cloud-ASR row promises immediate effect, not a restart', () => {
    renderForm();
    const row = screen.getByTestId('key-row-openai');
    expect(row).toHaveTextContent('儲存後立即生效');
    // Guard against the TMDb row's affirmative wording leaking over. Matching a
    // bare 「需重啟」 would be wrong: this row legitimately says 「無需重啟伺服器」.
    expect(row).not.toHaveTextContent('儲存後需重啟伺服器才會生效');
  });
});

describe('ApiKeysForm — saving (AC #1)', () => {
  it('PUTs only the CHANGED rows, never the untouched ones', async () => {
    renderForm();

    fireEvent.change(screen.getByLabelText('雲端 ASR（選配）'), { target: { value: 'sk-openai' } });
    fireEvent.click(screen.getByTestId('key-save'));

    await waitFor(() => expect(h.saveMutate).toHaveBeenCalledTimes(1));
    expect(h.saveMutate.mock.calls[0][0]).toEqual({ openai: 'sk-openai' });
  });

  it('儲存 stays disabled while nothing has been typed', () => {
    renderForm();
    expect(screen.getByTestId('key-save')).toBeDisabled();
  });

  it('surfaces the backend message when the save is refused', async () => {
    h.saveMutate.mockImplementation((_updates, opts: { onError: (e: Error) => void }) =>
      opts.onError(new Error('未設定加密金鑰，無法安全儲存 API 金鑰'))
    );
    renderForm();

    fireEvent.change(screen.getByLabelText('TMDB'), { target: { value: 'tmdb-key' } });
    fireEvent.click(screen.getByTestId('key-save'));

    expect(await screen.findByTestId('key-save-error')).toHaveTextContent(
      '未設定加密金鑰，無法安全儲存 API 金鑰'
    );
  });
});

describe('ApiKeysForm — 清除 (AC #1)', () => {
  it('asks for confirmation first and sends nothing until 確認清除', () => {
    h.query.data = MIXED_SOURCES;
    renderForm();

    fireEvent.click(screen.getByTestId('key-clear-claude'));

    expect(screen.getByTestId('key-clear-confirm-claude')).toBeInTheDocument();
    expect(h.saveMutate).not.toHaveBeenCalled();
  });

  it('確認清除 sends an explicit empty string — 2-1a’s delete path', async () => {
    h.query.data = MIXED_SOURCES;
    renderForm();

    fireEvent.click(screen.getByTestId('key-clear-claude'));
    fireEvent.click(screen.getByTestId('key-clear-confirm-yes-claude'));

    await waitFor(() => expect(h.saveMutate).toHaveBeenCalledTimes(1));
    expect(h.saveMutate.mock.calls[0][0]).toEqual({ claude: '' });
  });

  it('取消 abandons the clear without writing', () => {
    h.query.data = MIXED_SOURCES;
    renderForm();

    fireEvent.click(screen.getByTestId('key-clear-claude'));
    fireEvent.click(screen.getByTestId('key-clear-cancel-claude'));

    expect(screen.queryByTestId('key-clear-confirm-claude')).toBeNull();
    expect(h.saveMutate).not.toHaveBeenCalled();
  });
});

describe('ApiKeysForm — writable:false degrades honestly (AC #2)', () => {
  const READ_ONLY: KeySettings = {
    writable: false,
    reason: 'encryption_key_missing',
    keys: [
      { name: 'claude', configured: true, source: 'env' },
      { name: 'tmdb', configured: false, source: 'none' },
      { name: 'openai', configured: false, source: 'none' },
    ],
  };

  beforeEach(() => {
    h.query.data = READ_ONLY;
  });

  // dsr-3b — C21-D (AVUg2): a missing ENCRYPTION_KEY is a broken setup, not a
  // request that didn't happen — 硃砂, not 赭 — and it says what happened
  // and what to do in two lines.
  it('states the reason and what to do about it, in 硃砂, as an alert', () => {
    renderForm();
    const banner = screen.getByTestId('keys-not-writable');
    expect(banner).toHaveAttribute('role', 'alert');
    expect(banner.className).toContain('bg-[var(--error-tint)]');
    expect(banner.className).not.toContain('warning');
    expect(screen.getByText('未設定加密金鑰，無法安全儲存 API 金鑰')).toHaveClass(
      'font-semibold',
      'text-[var(--error-text)]'
    );
    expect(
      screen.getByText('請設定 ENCRYPTION_KEY 後重啟容器。在那之前，這一頁只能檢視，不能儲存。')
    ).toHaveClass('text-xs', 'text-[var(--error-text)]');
  });

  it('greys the labels and draws disabled inputs in the disabled text token', () => {
    renderForm();
    expect(screen.getByText('Claude（翻譯）', { selector: 'label' })).toHaveClass(
      'text-[var(--text-muted)]'
    );
    expect(screen.getByLabelText('TMDB').className).toContain(
      'disabled:text-[var(--text-disabled)]'
    );
    expect(screen.getByLabelText('TMDB').className).not.toContain('disabled:opacity-50');
  });

  it('DISABLES inputs and 儲存 rather than hiding them (Rule 24 capability honor)', () => {
    renderForm();

    expect(screen.getByLabelText('Claude（翻譯）')).toBeDisabled();
    expect(screen.getByLabelText('TMDB')).toBeDisabled();
    expect(screen.getByLabelText('雲端 ASR（選配）')).toBeDisabled();
    expect(screen.getByTestId('key-save')).toBeDisabled();
  });

  it('keeps current state visible — read-only is not blank', () => {
    renderForm();
    expect(screen.getByTestId('key-state-claude')).toHaveTextContent('目前由環境變數提供');
  });

  it('still allows 測試 — probing stores nothing, so it is not gated on writability', () => {
    renderForm();
    expect(screen.getByTestId('key-test-claude')).toBeEnabled();
  });

  // CR sub-2-1b M2: the READ_ONLY fixture above carries no secret-sourced key,
  // so the 編輯/清除 half of read-only was previously unexercised.
  it('a secret-sourced row under writable:false draws 編輯/清除 disabled, not hidden', () => {
    h.query.data = {
      writable: false,
      reason: 'encryption_key_missing',
      keys: [
        { name: 'claude', configured: true, source: 'secret', masked: 'sk-ant…7f3a' },
        { name: 'tmdb', configured: false, source: 'none' },
        { name: 'openai', configured: false, source: 'none' },
      ],
    };
    renderForm();

    expect(screen.getByTestId('key-edit-claude')).toBeDisabled();
    expect(screen.getByTestId('key-clear-claude')).toBeDisabled();
    // Still visible with its state — read-only is not blank.
    expect(screen.getByTestId('key-state-claude')).toHaveTextContent('已設定');
    expect(screen.getByTestId('key-masked-claude')).toHaveTextContent('sk-ant…7f3a');
  });
});

describe('ApiKeysForm — NFR-S3 HTTPS gate (AC #3)', () => {
  it('insecure context ⇒ warning + 儲存 blocked until the checkbox is ticked', async () => {
    setSecureContext(false);
    renderForm();

    // dsr-3b — C22-D: a title and a sentence, both in 赭's text token.
    const warning = screen.getByTestId('insecure-context-warning');
    expect(screen.getByText('目前連線未加密（HTTP）')).toHaveClass(
      'font-semibold',
      'text-[var(--warning-text)]'
    );
    expect(warning).toHaveTextContent('API 金鑰會以明文傳送到 NAS。建議先設定 HTTPS 反向代理。');
    // 泥金 means 正在跑; this checkbox is an acknowledgement of a warning.
    const ack = screen.getByTestId('insecure-context-ack');
    expect(ack.className).toContain('accent-[var(--warning-text)]');
    expect(ack.className).not.toContain('accent-primary');
    expect(ack.closest('label')).toHaveClass(
      'text-xs',
      'font-semibold',
      'text-[var(--warning-text)]'
    );

    fireEvent.change(screen.getByLabelText('Claude（翻譯）'), { target: { value: 'sk-ant-x' } });
    expect(screen.getByTestId('key-save')).toBeDisabled();

    fireEvent.click(screen.getByTestId('insecure-context-ack'));
    await waitFor(() => expect(screen.getByTestId('key-save')).toBeEnabled());

    fireEvent.click(screen.getByTestId('key-save'));
    await waitFor(() =>
      expect(h.saveMutate).toHaveBeenCalledWith({ claude: 'sk-ant-x' }, expect.anything())
    );
  });

  it('secure context ⇒ neither warning nor checkbox, and 儲存 needs no acknowledgement', async () => {
    setSecureContext(true);
    renderForm();

    expect(screen.queryByTestId('insecure-context-warning')).toBeNull();
    expect(screen.queryByTestId('insecure-context-ack')).toBeNull();

    fireEvent.change(screen.getByLabelText('Claude（翻譯）'), { target: { value: 'sk-ant-x' } });
    await waitFor(() => expect(screen.getByTestId('key-save')).toBeEnabled());
  });

  it('localhost over http is treated as SECURE — the gate reads isSecureContext, not the protocol', () => {
    // jsdom serves the suite from http://localhost, i.e. exactly the dev/first-run
    // path. A `location.protocol === "https:"` test would warn here and train
    // users to dismiss the one warning that matters.
    expect(window.location.protocol).toBe('http:');
    setSecureContext(true);
    renderForm();

    expect(screen.queryByTestId('insecure-context-warning')).toBeNull();
  });

  // CR sub-2-1b M3: the gate extension to 測試 — a typed candidate crosses the
  // wire in cleartext exactly as 儲存 does, so it is gated the same way; the
  // no-candidate probe transmits no secret and stays available.
  it('insecure context gates 測試 only while a candidate has been typed', async () => {
    setSecureContext(false);
    renderForm();

    // No typed candidate → probing the resolved key sends nothing → enabled.
    expect(screen.getByTestId('key-test-claude')).toBeEnabled();

    fireEvent.change(screen.getByLabelText('Claude（翻譯）'), { target: { value: 'sk-ant-x' } });
    expect(screen.getByTestId('key-test-claude')).toBeDisabled();

    fireEvent.click(screen.getByTestId('insecure-context-ack'));
    await waitFor(() => expect(screen.getByTestId('key-test-claude')).toBeEnabled());
  });

  it('the acknowledgement is per visit — a remount starts unticked', () => {
    setSecureContext(false);
    const first = renderForm();
    fireEvent.click(screen.getByTestId('insecure-context-ack'));
    expect(screen.getByTestId('insecure-context-ack')).toBeChecked();
    first.unmount();

    renderForm();
    expect(screen.getByTestId('insecure-context-ack')).not.toBeChecked();
  });
});

describe('ApiKeysForm — 測試 (AC #1, AC #5.5)', () => {
  it('renders 成功 inline, not as a toast-only signal', async () => {
    h.testMutate.mockImplementation(
      (_candidate, opts: { onSuccess: () => void; onSettled: () => void }) => {
        opts.onSuccess();
        opts.onSettled();
      }
    );
    renderForm();

    fireEvent.click(screen.getByTestId('key-test-claude'));

    expect(await screen.findByTestId('key-test-result-claude')).toHaveTextContent('金鑰驗證成功');
  });

  it('[sub-6-6 FE half] names the model the probe actually reached', async () => {
    // 「金鑰驗證成功」 alone leaves a real failure invisible: a valid key against
    // a model this account cannot call. Naming the verified model separates them.
    h.testMutate.mockImplementation(
      (
        _candidate,
        opts: { onSuccess: (r: { valid: boolean; model?: string }) => void; onSettled: () => void }
      ) => {
        opts.onSuccess({ valid: true, model: 'claude-sonnet-5' });
        opts.onSettled();
      }
    );
    renderForm();

    fireEvent.click(screen.getByTestId('key-test-claude'));

    expect(await screen.findByTestId('key-test-result-claude')).toHaveTextContent(
      '已驗證：claude-sonnet-5'
    );
  });

  it('a pre-sub-6-6 server sends no model — the form claims nothing about one', async () => {
    h.testMutate.mockImplementation(
      (_candidate, opts: { onSuccess: (r: { valid: boolean }) => void; onSettled: () => void }) => {
        opts.onSuccess({ valid: true });
        opts.onSettled();
      }
    );
    renderForm();

    fireEvent.click(screen.getByTestId('key-test-claude'));

    const result = await screen.findByTestId('key-test-result-claude');
    expect(result).toHaveTextContent('金鑰驗證成功');
    expect(result.textContent).not.toContain('已驗證');
  });

  it('a 401-shaped response renders 金鑰無效 without throwing', async () => {
    h.testMutate.mockImplementation(
      (_candidate, opts: { onError: (e: Error) => void; onSettled: () => void }) => {
        opts.onError(new Error('金鑰無效或已撤銷'));
        opts.onSettled();
      }
    );
    renderForm();

    fireEvent.click(screen.getByTestId('key-test-claude'));

    expect(await screen.findByTestId('key-test-result-claude')).toHaveTextContent('金鑰無效');
  });

  it('a timeout renders 逾時 without throwing', async () => {
    h.testMutate.mockImplementation(
      (_candidate, opts: { onError: (e: Error) => void; onSettled: () => void }) => {
        opts.onError(new Error('連線逾時，無法驗證金鑰'));
        opts.onSettled();
      }
    );
    renderForm();

    fireEvent.click(screen.getByTestId('key-test-claude'));

    expect(await screen.findByTestId('key-test-result-claude')).toHaveTextContent('逾時');
  });

  it('sends the TYPED candidate so a key can be verified before it is saved', async () => {
    renderForm();

    fireEvent.change(screen.getByLabelText('Claude（翻譯）'), { target: { value: 'sk-ant-new' } });
    fireEvent.click(screen.getByTestId('key-test-claude'));

    await waitFor(() => expect(h.testMutate).toHaveBeenCalledTimes(1));
    expect(h.testMutate.mock.calls[0][0]).toBe('sk-ant-new');
  });

  it('sends undefined when nothing is typed, i.e. "test whatever currently resolves"', async () => {
    h.query.data = MIXED_SOURCES;
    renderForm();

    fireEvent.click(screen.getByTestId('key-test-claude'));

    await waitFor(() => expect(h.testMutate).toHaveBeenCalledTimes(1));
    expect(h.testMutate.mock.calls[0][0]).toBeUndefined();
  });

  // CR sub-2-1b M1: a verdict vouches only for the exact string it tested.
  it('編輯 → 測試成功 → 取消 clears the verdict — it must not vouch for the stored key', async () => {
    h.testMutate.mockImplementation(
      (_candidate, opts: { onSuccess: () => void; onSettled: () => void }) => {
        opts.onSuccess();
        opts.onSettled();
      }
    );
    h.query.data = MIXED_SOURCES;
    renderForm();

    fireEvent.click(screen.getByTestId('key-edit-claude'));
    fireEvent.change(screen.getByLabelText('Claude（翻譯）'), { target: { value: 'sk-ant-new' } });
    fireEvent.click(screen.getByTestId('key-test-claude'));
    expect(await screen.findByTestId('key-test-result-claude')).toHaveTextContent('金鑰驗證成功');

    fireEvent.click(screen.getByTestId('key-edit-cancel-claude'));
    expect(screen.queryByTestId('key-test-result-claude')).toBeNull();
  });

  it('editing the candidate voids the previous verdict', async () => {
    h.testMutate.mockImplementation(
      (_candidate, opts: { onSuccess: () => void; onSettled: () => void }) => {
        opts.onSuccess();
        opts.onSettled();
      }
    );
    renderForm();

    fireEvent.change(screen.getByLabelText('Claude（翻譯）'), { target: { value: 'sk-ant-a' } });
    fireEvent.click(screen.getByTestId('key-test-claude'));
    expect(await screen.findByTestId('key-test-result-claude')).toHaveTextContent('金鑰驗證成功');

    fireEvent.change(screen.getByLabelText('Claude（翻譯）'), { target: { value: 'sk-ant-b' } });
    expect(screen.queryByTestId('key-test-result-claude')).toBeNull();
  });

  // dsr-3b: only Claude has a probe (2-1a), so the other rows draw no 測試
  // at all — a disabled button on every row plus a note on every row was the
  // same fact said four times. It is said once, below the rows.
  it('draws 測試 only for Claude and says so once, below the rows', () => {
    renderForm();
    expect(screen.getByTestId('key-test-claude')).toBeInTheDocument();
    expect(screen.queryByTestId('key-test-tmdb')).toBeNull();
    expect(screen.queryByTestId('key-test-openai')).toBeNull();
    expect(screen.getAllByText('僅 Claude 金鑰支援連線測試。')).toHaveLength(1);
    expect(screen.getByTestId('key-row-tmdb')).not.toHaveTextContent('僅支援');
    // No claim that the other keys are checked some other way — nothing says so.
    expect(screen.queryByText(/自行驗證/)).toBeNull();
  });
});

describe('ApiKeysForm — load states', () => {
  it('shows a spinner while the key state loads', () => {
    h.query = { data: undefined, isLoading: true, isError: false, error: null };
    renderForm();

    expect(screen.getByTestId('api-keys-loading')).toBeInTheDocument();
    expect(screen.queryByTestId('api-keys-form')).toBeNull();
  });

  it('fails soft when the key state cannot be read — in plain words, with 重試', () => {
    h.query = {
      data: undefined,
      isLoading: false,
      isError: true,
      error: new Error('Failed to fetch'),
    };
    renderForm();

    const banner = screen.getByTestId('api-keys-load-error');
    expect(banner).toHaveTextContent('無法讀取金鑰設定');
    expect(banner).toHaveTextContent('與後端的連線中斷了。已存的金鑰不受影響。');
    // dsr-3b: the backend's own words never reach the page.
    expect(banner).not.toHaveTextContent('Failed to fetch');
    fireEvent.click(screen.getByRole('button', { name: '重試' }));
    expect(h.refetch).toHaveBeenCalledTimes(1);
    // The form still renders so the page is never a blank dead end — and the
    // TMDB attribution below must stay (sub-6-9, compliance is not conditional).
    expect(screen.getByTestId('api-keys-form')).toBeInTheDocument();
    // CR sub-2-1b L2: unknown ≠ not set — a server outage must not badge a
    // possibly-configured key as 尚未設定.
    expect(screen.getByTestId('key-state-claude')).toHaveTextContent('無法確認');
    expect(screen.getByTestId('key-state-tmdb')).toHaveTextContent('無法確認');
  });
});

describe('ApiKeysForm — TMDB attribution (sub-6-9, TMDB terms §3)', () => {
  it('carries the §3 notice and logo inside the TMDB row', () => {
    renderForm();

    const attribution = screen.getByTestId('tmdb-attribution');
    // Inside the TMDB row specifically — an attribution floating at the bottom
    // of the page would not say WHICH data source it accounts for.
    expect(screen.getByTestId('key-row-tmdb')).toContainElement(attribution);
    expect(attribution).toHaveTextContent(
      'This application uses TMDB and the TMDB APIs but is not endorsed, certified, or otherwise approved by TMDB.'
    );
    expect(attribution).toHaveTextContent(
      '本應用程式使用 TMDB 與 TMDB API，但未經 TMDB 認可、認證或核准。'
    );
    expect(screen.getByAltText('TMDB')).toBeInTheDocument();
  });

  it('shows it once — not per key row', () => {
    renderForm();

    expect(screen.getAllByTestId('tmdb-attribution')).toHaveLength(1);
    expect(screen.getByTestId('key-row-claude')).not.toContainElement(
      screen.getByTestId('tmdb-attribution')
    );
  });

  it('stays on screen when the key state cannot be read (compliance is not conditional)', () => {
    h.query = { data: undefined, isLoading: false, isError: true, error: new Error('boom') };
    renderForm();

    expect(screen.getByTestId('tmdb-attribution')).toBeInTheDocument();
  });
});

// dsr-3b — C7-D (PWvEX) / C7-M (f8Fda): each row is label + state → input (the
// 測試 button sits INSIDE it, right) → hint; a phone gives every key its own card.
describe('ApiKeysForm — C7 row layout', () => {
  it('puts Claude’s 測試 inside the input box', () => {
    renderForm();
    const input = screen.getByLabelText('Claude（翻譯）');
    const test = screen.getByTestId('key-test-claude');
    expect(input.parentElement).toContainElement(test);
    expect(test).toHaveClass('h-7', 'text-xs');
    expect(input).toHaveClass('min-h-11', 'font-mono', 'bg-[var(--bg-tertiary)]');
  });

  it('keeps 測試 with 編輯 / 清除 when the key is stored (there is no input)', () => {
    h.query.data = MIXED_SOURCES;
    renderForm();
    const test = screen.getByTestId('key-test-claude');
    expect(test.parentElement).toContainElement(screen.getByTestId('key-edit-claude'));
  });

  it('puts the hint under the input, not above it', () => {
    renderForm();
    const input = screen.getByLabelText('Claude（翻譯）');
    const hint = screen.getByText(/用於字幕翻譯與 AI 檔名解析/);
    expect(input.compareDocumentPosition(hint) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it('sets row labels at 14 / 600 in the primary text colour, and the state pill at 12', () => {
    renderForm();
    expect(screen.getByText('Claude（翻譯）', { selector: 'label' })).toHaveClass(
      'text-sm',
      'font-semibold',
      'text-[var(--text-primary)]'
    );
    const pill = screen.getByTestId('key-state-claude');
    expect(pill).toHaveClass('text-xs', 'font-semibold');
    expect(pill.className).not.toContain('text-[11px]');
  });

  it('draws the card solid, separates rows by space not rules, and says 儲存金鑰', () => {
    renderForm();
    const card = screen.getByTestId('api-keys-card');
    expect(card.className).toContain('bg-[var(--bg-secondary)]');
    expect(card.className).not.toContain('/50');
    expect(card.className).toContain('rounded-[var(--radius-lg)]');
    expect(screen.getByTestId('key-rows').className).not.toContain('divide-y');
    const save = screen.getByTestId('key-save');
    expect(save).toHaveTextContent('儲存金鑰');
    expect(save).toHaveClass('min-h-11', 'font-semibold', 'max-sm:w-full');
  });

  it('gives every key its own card on a phone', () => {
    renderForm();
    expect(screen.getByTestId('api-keys-card').className).toContain('max-sm:bg-transparent');
    for (const name of ['claude', 'tmdb', 'openai']) {
      const row = screen.getByTestId(`key-row-${name}`);
      expect(row.className).toContain('max-sm:bg-[var(--bg-secondary)]');
      expect(row.className).toContain('max-sm:p-4');
    }
  });

  it('spells the brand TMDB in the placeholder too', () => {
    renderForm();
    expect(screen.getByLabelText('TMDB')).toHaveAttribute('placeholder', 'TMDB API Key');
  });
});
