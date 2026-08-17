// Design ref: ux-design.pen — no current screen frame; FR25 金鑰設定頁 is M1.5 net-new (story sub-2-1b) and rides the designed settings shell (Screen 10 Settings Desktop, 6UCtX) rather than adding a frame of its own
/**
 * 金鑰設定 — in-app provider API key configuration (Story sub-2-1b, FR25 /
 * architecture D9 NFR-S3).
 *
 * Mirrors the shipped secrets-backed settings page (QBittorrentForm, rendered by
 * /settings/connection) rather than inventing a shell: same card wrapper, same
 * label/input/action rhythm, same tokens, same SettingsLayout sidebar entry.
 *
 * Three things this page refuses to lie about:
 *  - PRECEDENCE. A key set here (`secret`) beats a `CLAUDE_API_KEY`-style env
 *    var, so an env-sourced row says so AND warns that saving overrides it.
 *  - WRITABILITY. No ENCRYPTION_KEY ⇒ `writable: false` ⇒ inputs and 儲存 are
 *    DISABLED WITH A REASON, never hidden (Rule 24 capability honor — the same
 *    "never draw a dead control as live" rule ManageSubtitleDialogV2 follows).
 *  - TRANSPORT. Over plain HTTP the key crosses the wire in cleartext, so D9's
 *    warn-and-confirm gate blocks 儲存 until the user acknowledges it. The
 *    signal is `window.isSecureContext`, NOT a `location.protocol` string test,
 *    so localhost — secure by definition — never trains users to dismiss it.
 */
import { useState } from 'react';
import { AlertTriangle, Loader2, Plug, Save, ShieldAlert, Trash2 } from 'lucide-react';
import { cn } from '../../lib/utils';
import { useKeySettings, useSaveKeys, useTestClaudeKey } from '../../hooks/useKeySettings';
import {
  REASON_ENCRYPTION_KEY_MISSING,
  type KeyName,
  type KeyState,
  type KeyUpdates,
} from '../../services/keySettingsService';

interface KeyRowSpec {
  name: KeyName;
  label: string;
  placeholder: string;
  hint: string;
  /**
   * Only Claude has a probe endpoint (sub-2-1a's POST /settings/keys/test takes
   * a `claude` field and nothing else). The other rows still SHOW 測試, disabled
   * with a reason — hiding it would leave the user hunting for a control that
   * simply does not exist yet.
   */
  testable: boolean;
}

const KEY_ROWS: KeyRowSpec[] = [
  {
    name: 'claude',
    label: 'Claude（翻譯）',
    placeholder: 'sk-ant-…',
    hint: '用於字幕翻譯與 AI 檔名解析。儲存後立即生效，無需重啟伺服器。',
    testable: true,
  },
  {
    name: 'tmdb',
    label: 'TMDB',
    placeholder: 'TMDb API Key',
    // backlog-tmdb-runtime-key-resolution: the resolver EXPOSES the TMDb key so
    // it can be stored here, but the running TMDb client keeps its env value
    // until restart. Saying so is that backlog entry's explicit ask of 2-1b.
    hint: '用於中繼資料與海報。儲存後需重啟伺服器才會生效。',
    testable: false,
  },
  {
    name: 'openai',
    label: '雲端 ASR（選配）',
    placeholder: 'sk-…',
    // sub-5-2: the ASR client resolves its key per call (ASRProviderHolder), so
    // this row joins Claude in taking effect on save. The self-hosted
    // no-key-needed fact deliberately lives in docs/deployment.md, NOT here
    // (sub-5-2 CR M1): a first-draft clause bolted it onto the fallback
    // sentence and read as "self-hosted ⇒ ASR unavailable, use built-in
    // sources" — the opposite of the truth. This is the AC #5 ratified string.
    hint: '選配：雲端語音辨識。儲存後立即生效，無需重啟伺服器。未設定時仍可使用內建的字幕來源。',
    testable: false,
  },
];

const EMPTY_VALUES: Record<KeyName, string> = { claude: '', tmdb: '', openai: '' };

function stateLabel(state: KeyState | undefined): string {
  switch (state?.source) {
    case 'secret':
      return '已設定';
    case 'env':
      return '目前由環境變數提供';
    default:
      return '尚未設定';
  }
}

function stateToneClass(state: KeyState | undefined): string {
  switch (state?.source) {
    case 'secret':
      return 'bg-[var(--success-tint)] text-[var(--success)]';
    case 'env':
      return 'bg-[var(--info-tint)] text-[var(--info)]';
    default:
      return 'bg-[var(--bg-tertiary)] text-[var(--text-muted)]';
  }
}

const INPUT_CLASS =
  'w-full rounded-md border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)] disabled:cursor-not-allowed disabled:opacity-50';

const SECONDARY_BUTTON_CLASS =
  'flex items-center gap-1.5 rounded-md bg-[var(--bg-tertiary)] px-4 py-2 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)] disabled:cursor-not-allowed disabled:opacity-50';

export function ApiKeysForm() {
  const { data: settings, isLoading, isError, error } = useKeySettings();
  const saveMutation = useSaveKeys();
  const testMutation = useTestClaudeKey();

  const [values, setValues] = useState<Record<KeyName, string>>(EMPTY_VALUES);
  const [editing, setEditing] = useState<Partial<Record<KeyName, boolean>>>({});
  const [confirmingClear, setConfirmingClear] = useState<KeyName | null>(null);
  const [testingName, setTestingName] = useState<KeyName | null>(null);
  const [testResults, setTestResults] = useState<
    Partial<Record<KeyName, { ok: boolean; message: string }>>
  >({});
  // Per PAGE VISIT — deliberately not persisted anywhere. A remembered
  // acknowledgement is an acknowledgement nobody reads the second time.
  const [insecureAck, setInsecureAck] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveSuccess, setSaveSuccess] = useState(false);

  // `window.isSecureContext` is true for HTTPS *and* localhost. A backend check
  // would have to trust X-Forwarded-Proto, which a misconfigured reverse proxy
  // sets wrong — so this judgement belongs in the browser (AC #3).
  const isSecureContext = typeof window === 'undefined' || window.isSecureContext;

  const writable = settings?.writable ?? false;
  const notWritableReason = settings?.reason;

  const byName = new Map<KeyName, KeyState>((settings?.keys ?? []).map((k) => [k.name, k]));

  const pendingUpdates: KeyUpdates = {};
  for (const row of KEY_ROWS) {
    // Only CHANGED rows travel — an untouched field is absent from the payload,
    // which 2-1a reads as "leave this key alone" (a blank string would DELETE).
    if (values[row.name].trim() !== '') pendingUpdates[row.name] = values[row.name];
  }
  const hasPendingUpdates = Object.keys(pendingUpdates).length > 0;
  const insecureBlocked = !isSecureContext && !insecureAck;
  const canSave = writable && hasPendingUpdates && !insecureBlocked && !saveMutation.isPending;

  const resetSaveFeedback = () => {
    setSaveError(null);
    setSaveSuccess(false);
  };

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSave) return;
    resetSaveFeedback();
    saveMutation.mutate(pendingUpdates, {
      onSuccess: () => {
        setValues(EMPTY_VALUES);
        setEditing({});
        setSaveSuccess(true);
      },
      onError: (err) => setSaveError(err.message),
    });
  };

  const handleClear = (name: KeyName) => {
    setConfirmingClear(null);
    resetSaveFeedback();
    // An explicit empty string is 2-1a's delete path: it removes the stored
    // secret and lets resolution fall back to the env var (or to nothing).
    saveMutation.mutate(
      { [name]: '' },
      {
        onSuccess: () => {
          setValues((v) => ({ ...v, [name]: '' }));
          setEditing((e) => ({ ...e, [name]: false }));
          setTestResults((r) => ({ ...r, [name]: undefined }));
        },
        onError: (err) => setSaveError(err.message),
      }
    );
  };

  const handleTest = (name: KeyName) => {
    // CR sub-2-1b L1: the probe endpoint takes a `claude` field and nothing
    // else, so this handler must refuse any other row even if a future edit
    // flips its `testable` flag — otherwise that row's key would be POSTed as
    // a Claude candidate.
    if (name !== 'claude') return;
    const candidate = values[name].trim();
    setTestingName(name);
    setTestResults((r) => ({ ...r, [name]: undefined }));
    testMutation.mutate(candidate === '' ? undefined : candidate, {
      onSuccess: () =>
        setTestResults((r) => ({ ...r, [name]: { ok: true, message: '金鑰驗證成功' } })),
      // The backend already speaks zh-TW per Rule 3 (金鑰無效或已撤銷 / 連線逾時…),
      // and it distinguishes a bad key from a bad model id and from a quota
      // ceiling — re-writing those here would flatten three different next
      // actions into one.
      onError: (err) =>
        setTestResults((r) => ({ ...r, [name]: { ok: false, message: err.message } })),
      onSettled: () => setTestingName(null),
    });
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12" data-testid="api-keys-loading">
        <Loader2 className="h-6 w-6 animate-spin text-[var(--text-secondary)]" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl">
      <h1 className="mb-2 text-2xl font-bold text-[var(--text-primary)]">金鑰設定</h1>
      <p className="mb-6 text-sm text-[var(--text-secondary)]">
        設定 Vido 使用的第三方服務 API 金鑰。金鑰會加密後儲存於 NAS，並優先於環境變數。
      </p>

      {isError && (
        <div
          data-testid="api-keys-load-error"
          role="status"
          aria-live="polite"
          className="mb-6 flex items-start gap-3 rounded-md bg-[var(--error-tint)] p-4"
        >
          <AlertTriangle
            className="mt-0.5 h-4 w-4 shrink-0 text-[var(--error)]"
            aria-hidden="true"
          />
          <p className="text-sm text-[var(--error-text)]">
            無法讀取金鑰設定{error?.message ? `：${error.message}` : ''}
          </p>
        </div>
      )}

      {/* AC #2 — read-only degradation. Controls stay VISIBLE and disabled with
          a reason; hiding them would leave the user with nothing to act on. */}
      {settings && !writable && (
        <div
          data-testid="keys-not-writable"
          role="status"
          aria-live="polite"
          className="mb-6 flex items-start gap-3 rounded-md bg-[var(--warning-tint)] p-4"
        >
          <AlertTriangle
            className="mt-0.5 h-4 w-4 shrink-0 text-[var(--warning)]"
            aria-hidden="true"
          />
          <p className="text-sm text-[var(--text-primary)]">
            {notWritableReason === REASON_ENCRYPTION_KEY_MISSING
              ? '未設定加密金鑰，無法安全儲存 API 金鑰 —— 請設定 ENCRYPTION_KEY 後重啟。'
              : '目前無法儲存 API 金鑰，設定僅供檢視。'}
          </p>
        </div>
      )}

      {/* AC #3 — NFR-S3. Advisory, never a hard block: Vido ships over HTTP by
          default, so blocking outright would make the feature unusable for the
          audience it exists for. D9's own wording is warn + require confirmation. */}
      {!isSecureContext && (
        <div
          data-testid="insecure-context-warning"
          role="status"
          aria-live="polite"
          className="mb-6 rounded-md bg-[var(--warning-tint)] p-4"
        >
          <div className="flex items-start gap-3">
            <ShieldAlert
              className="mt-0.5 h-4 w-4 shrink-0 text-[var(--warning)]"
              aria-hidden="true"
            />
            <p className="text-sm text-[var(--text-primary)]">
              目前連線未加密（HTTP），API 金鑰會以明文傳送到 NAS。建議先設定 HTTPS 反向代理。
            </p>
          </div>
          <label className="mt-3 flex cursor-pointer items-center gap-2 pl-7 text-sm text-[var(--text-secondary)]">
            <input
              type="checkbox"
              checked={insecureAck}
              onChange={(e) => setInsecureAck(e.target.checked)}
              data-testid="insecure-context-ack"
              className="h-4 w-4 shrink-0 accent-[var(--accent-primary)]"
            />
            我了解風險，仍要在未加密連線下儲存
          </label>
        </div>
      )}

      {/* The settings card is the shipped shell (routes/settings/connection.tsx) —
          same classes, so the two pages sit at the same tone and radius. */}
      <div className="rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-secondary)]/50 p-6">
        <form onSubmit={handleSave} data-testid="api-keys-form">
          <div className="divide-y divide-[var(--border-subtle)]">
            {KEY_ROWS.map((row) => {
              const state = byName.get(row.name);
              const inputId = `key-input-${row.name}`;
              const hintId = `key-hint-${row.name}`;
              const isStored = state?.source === 'secret';
              const isEditing = editing[row.name] === true;
              const showInput = !isStored || isEditing;
              const isConfirmingClear = confirmingClear === row.name;
              const typedCandidate = values[row.name].trim() !== '';
              const result = testResults[row.name];

              return (
                <div
                  key={row.name}
                  data-testid={`key-row-${row.name}`}
                  className="py-5 first:pt-0 last:pb-0"
                >
                  <div className="mb-1.5 flex flex-wrap items-center gap-2">
                    {/* A stored key draws no input, so it must draw no <label for>
                        either — an orphan label is an a11y defect, not decoration. */}
                    {showInput ? (
                      <label
                        htmlFor={inputId}
                        className="text-sm font-medium text-[var(--text-secondary)]"
                      >
                        {row.label}
                      </label>
                    ) : (
                      <span className="text-sm font-medium text-[var(--text-secondary)]">
                        {row.label}
                      </span>
                    )}
                    <span
                      data-testid={`key-state-${row.name}`}
                      className={cn(
                        'rounded-full px-2 py-0.5 text-[11px] font-medium',
                        stateToneClass(state)
                      )}
                    >
                      {/* CR sub-2-1b L2: a failed read is UNKNOWN, not "not
                          set" — a server outage must not badge a configured
                          key as 尚未設定. */}
                      {isError && !state ? '無法確認' : stateLabel(state)}
                    </span>
                    {isStored && state?.masked && (
                      <span
                        data-testid={`key-masked-${row.name}`}
                        className="font-mono text-xs text-[var(--text-muted)]"
                      >
                        {state.masked}
                      </span>
                    )}
                  </div>

                  <p id={hintId} className="mb-3 text-xs text-[var(--text-muted)]">
                    {row.hint}
                  </p>

                  {/* Honest about precedence: 2-1a resolves secret > env, so saving
                      here silently overrides the deploy-time variable. Say it. */}
                  {state?.source === 'env' && (
                    <p
                      data-testid={`key-env-override-note-${row.name}`}
                      className="mb-3 text-xs text-[var(--info)]"
                    >
                      在此儲存的金鑰會覆蓋環境變數提供的設定。
                    </p>
                  )}

                  {showInput && (
                    <div className="mb-3 flex flex-col gap-2 sm:flex-row">
                      <input
                        id={inputId}
                        // Never a text field, and never seeded from the server —
                        // 2-1a returns masks only, so there is no value to fetch.
                        type="password"
                        autoComplete="off"
                        value={values[row.name]}
                        aria-describedby={hintId}
                        onChange={(e) => {
                          resetSaveFeedback();
                          // CR sub-2-1b M1: a verdict only vouches for the exact
                          // string it tested — editing the candidate voids it.
                          setTestResults((r) => ({ ...r, [row.name]: undefined }));
                          setValues((v) => ({ ...v, [row.name]: e.target.value }));
                        }}
                        disabled={!writable}
                        placeholder={row.placeholder}
                        className={INPUT_CLASS}
                      />
                      {isStored && (
                        <button
                          type="button"
                          onClick={() => {
                            setEditing((e) => ({ ...e, [row.name]: false }));
                            setValues((v) => ({ ...v, [row.name]: '' }));
                            // CR sub-2-1b M1: the abandoned candidate's verdict
                            // must not stand next to the STORED key it never
                            // tested (清除 already resets — keep the paths equal).
                            setTestResults((r) => ({ ...r, [row.name]: undefined }));
                          }}
                          data-testid={`key-edit-cancel-${row.name}`}
                          className="shrink-0 rounded-md bg-[var(--bg-tertiary)] px-4 py-2 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-primary)]"
                        >
                          取消
                        </button>
                      )}
                    </div>
                  )}

                  {isConfirmingClear ? (
                    /* Inline two-step confirm rather than a modal: it needs no
                       focus trap to be keyboard-correct, and the consequence is
                       one sentence — a dialog would be heavier than the decision. */
                    <div
                      data-testid={`key-clear-confirm-${row.name}`}
                      className="flex flex-col gap-2 rounded-md bg-[var(--warning-tint)] p-3 sm:flex-row sm:items-center"
                    >
                      <p className="flex-1 text-xs text-[var(--text-primary)]">
                        清除後將改用環境變數的金鑰；若環境變數也未設定，相關功能會停用。
                      </p>
                      <div className="flex shrink-0 gap-2">
                        <button
                          type="button"
                          onClick={() => handleClear(row.name)}
                          data-testid={`key-clear-confirm-yes-${row.name}`}
                          className="rounded-md bg-[var(--error)] px-3 py-1.5 text-sm font-medium text-white transition-opacity hover:opacity-90"
                        >
                          確認清除
                        </button>
                        <button
                          type="button"
                          onClick={() => setConfirmingClear(null)}
                          data-testid={`key-clear-cancel-${row.name}`}
                          className="rounded-md bg-[var(--bg-tertiary)] px-3 py-1.5 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-primary)]"
                        >
                          取消
                        </button>
                      </div>
                    </div>
                  ) : (
                    <div className="flex flex-wrap items-center gap-2">
                      {!showInput && (
                        <>
                          <button
                            type="button"
                            onClick={() => {
                              resetSaveFeedback();
                              setEditing((e) => ({ ...e, [row.name]: true }));
                            }}
                            disabled={!writable}
                            data-testid={`key-edit-${row.name}`}
                            className={SECONDARY_BUTTON_CLASS}
                          >
                            編輯
                          </button>
                          <button
                            type="button"
                            onClick={() => setConfirmingClear(row.name)}
                            disabled={!writable}
                            data-testid={`key-clear-${row.name}`}
                            className={SECONDARY_BUTTON_CLASS}
                          >
                            <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                            清除
                          </button>
                        </>
                      )}
                      <button
                        type="button"
                        onClick={() => handleTest(row.name)}
                        // Writability is irrelevant here — probing stores nothing,
                        // so an operator without ENCRYPTION_KEY can still find out
                        // whether the env key they deployed actually works.
                        // The insecure gate applies only when a TYPED key would
                        // cross the wire; testing the resolved key sends none.
                        disabled={
                          !row.testable ||
                          (testMutation.isPending && testingName === row.name) ||
                          (insecureBlocked && typedCandidate)
                        }
                        title={row.testable ? undefined : '目前僅支援 Claude 金鑰測試'}
                        data-testid={`key-test-${row.name}`}
                        className={SECONDARY_BUTTON_CLASS}
                      >
                        {testMutation.isPending && testingName === row.name ? (
                          <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
                        ) : (
                          <Plug className="h-4 w-4" aria-hidden="true" />
                        )}
                        測試
                      </button>
                      {!row.testable && (
                        <span className="text-xs text-[var(--text-muted)]">
                          目前僅支援 Claude 金鑰測試
                        </span>
                      )}
                      {result && (
                        <span
                          data-testid={`key-test-result-${row.name}`}
                          role="status"
                          aria-live="polite"
                          className={cn(
                            'text-sm',
                            result.ok ? 'text-[var(--success)]' : 'text-[var(--error-text)]'
                          )}
                        >
                          {result.message}
                        </span>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>

          <div className="mt-6 flex flex-col items-end gap-3">
            {saveSuccess && (
              <p role="status" aria-live="polite" className="text-sm text-[var(--success)]">
                金鑰已儲存
              </p>
            )}
            {saveError && (
              <p
                data-testid="key-save-error"
                role="status"
                aria-live="polite"
                className="text-sm text-[var(--error-text)]"
              >
                {saveError}
              </p>
            )}
            <button
              type="submit"
              disabled={!canSave}
              data-testid="key-save"
              className={cn(
                'flex items-center gap-2 rounded-md px-4 py-2 text-sm font-medium transition-colors',
                canSave
                  ? 'bg-[var(--accent-primary)] text-[var(--text-on-accent)] hover:bg-[var(--accent-hover)]'
                  : 'cursor-not-allowed bg-[var(--bg-tertiary)] text-[var(--text-muted)]'
              )}
            >
              {saveMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
              ) : (
                <Save className="h-4 w-4" aria-hidden="true" />
              )}
              儲存
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
