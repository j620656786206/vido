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
}));

vi.mock('../../hooks/useKeySettings', () => ({
  useKeySettings: () => h.query,
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

  it('states the reason and what to do about it', () => {
    renderForm();
    expect(screen.getByTestId('keys-not-writable')).toHaveTextContent(
      '未設定加密金鑰，無法安全儲存 API 金鑰 —— 請設定 ENCRYPTION_KEY 後重啟。'
    );
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

    expect(screen.getByTestId('insecure-context-warning')).toHaveTextContent(
      '目前連線未加密（HTTP），API 金鑰會以明文傳送到 NAS。建議先設定 HTTPS 反向代理。'
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

  it('draws 測試 DISABLED with a reason for the keys 2-1a exposes no probe for', () => {
    renderForm();

    expect(screen.getByTestId('key-test-tmdb')).toBeDisabled();
    expect(screen.getByTestId('key-test-openai')).toBeDisabled();
    expect(screen.getByTestId('key-row-tmdb')).toHaveTextContent('目前僅支援 Claude 金鑰測試');
  });
});

describe('ApiKeysForm — load states', () => {
  it('shows a spinner while the key state loads', () => {
    h.query = { data: undefined, isLoading: true, isError: false, error: null };
    renderForm();

    expect(screen.getByTestId('api-keys-loading')).toBeInTheDocument();
    expect(screen.queryByTestId('api-keys-form')).toBeNull();
  });

  it('fails soft when the key state cannot be read', () => {
    h.query = {
      data: undefined,
      isLoading: false,
      isError: true,
      error: new Error('伺服器沒有回應'),
    };
    renderForm();

    expect(screen.getByTestId('api-keys-load-error')).toHaveTextContent('無法讀取金鑰設定');
    // The form still renders so the page is never a blank dead end.
    expect(screen.getByTestId('api-keys-form')).toBeInTheDocument();
    // CR sub-2-1b L2: unknown ≠ not set — a server outage must not badge a
    // possibly-configured key as 尚未設定.
    expect(screen.getByTestId('key-state-claude')).toHaveTextContent('無法確認');
    expect(screen.getByTestId('key-state-tmdb')).toHaveTextContent('無法確認');
  });
});
