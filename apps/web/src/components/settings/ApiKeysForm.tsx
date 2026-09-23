// Design ref: ux-design.pen Screen C7-D (PWvEX) · C7-M (f8Fda) · C21-D (AVUg2) · C22-D (t6FA4)
// C21 是未設定加密金鑰的唯讀態，C22 是 HTTP 明文警告
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
import { useEffect, useState } from 'react';
import {
  AlertTriangle,
  Check,
  Loader2,
  Lock,
  Plug,
  RefreshCw,
  Save,
  ShieldAlert,
  Trash2,
} from 'lucide-react';
import { cn } from '../../lib/utils';
import { TmdbAttribution } from '../ui/TmdbAttribution';
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
   * a `claude` field and nothing else). dsr-3b: the other rows draw NO 測試 —
   * a disabled button and the same note on every row said one fact four times.
   * It is said once, below the rows (C7-D pegsz).
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
    placeholder: 'TMDB API Key',
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
      return 'bg-[var(--success-tint)] text-[var(--success-text)]';
    case 'env':
      return 'bg-[var(--info-tint)] text-[var(--info-text)]';
    default:
      return 'bg-[var(--bg-tertiary)] text-[var(--text-muted)]';
  }
}

// C7-D ri9FI: 44 high, tertiary ground, mono (keys are machine strings).
// Disabled uses the disabled-text token (C21 y97vhg), not a blanket opacity.
const INPUT_CLASS =
  'min-h-11 w-full rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-tertiary)] px-3 font-mono text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)] disabled:cursor-not-allowed disabled:text-[var(--text-disabled)] disabled:placeholder-[var(--text-disabled)]';

const SECONDARY_BUTTON_CLASS =
  'flex min-h-9 items-center gap-1.5 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-4 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)] disabled:cursor-not-allowed disabled:opacity-50 max-sm:min-h-11';

// C7-D j6UfaO: the 測試 that lives inside the input box — 28 high, 12px, ruled.
const INLINE_TEST_BUTTON_CLASS =
  'absolute right-2 top-1/2 flex h-7 -translate-y-1/2 items-center gap-1 rounded-[var(--radius-sm)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3 text-xs font-semibold text-[var(--text-secondary)] transition-colors hover:text-[var(--text-primary)] disabled:cursor-not-allowed disabled:opacity-50';

export function ApiKeysForm() {
  const {
    data: settings,
    isLoading,
    isError,
    refetch,
    isFetching,
    errorUpdatedAt,
  } = useKeySettings();
  // A failed read with nothing cached goes back to "loading" the moment it is
  // refetched (TanStack v5 — project_tanstack_refetch_no_data_resets_pending).
  // Without this, 重試 would swap the whole page for a spinner and drop the
  // TMDB attribution — the two things fail-soft exists to keep on screen.
  const [retrying, setRetrying] = useState(false);
  useEffect(() => {
    if (retrying && !isFetching) setRetrying(false);
  }, [retrying, isFetching]);
  const readFailed = isError || retrying;
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
      // sub-6-6 (FE half, inherited by sub-6-8b): name the model the probe
      // actually reached. A valid key against a model this account cannot call
      // is a different failure with a different fix, and「金鑰驗證成功」alone
      // hides which one you have. Pre-sub-6-6 servers omit it — no model, no
      // claim.
      onSuccess: (result) =>
        setTestResults((r) => ({
          ...r,
          [name]: {
            ok: true,
            message: result?.model ? `金鑰驗證成功 · 已驗證：${result.model}` : '金鑰驗證成功',
          },
        })),
      // The backend already speaks zh-TW per Rule 3 (金鑰無效或已撤銷 / 連線逾時…),
      // and it distinguishes a bad key from a bad model id and from a quota
      // ceiling — re-writing those here would flatten three different next
      // actions into one.
      onError: (err) =>
        setTestResults((r) => ({ ...r, [name]: { ok: false, message: err.message } })),
      onSettled: () => setTestingName(null),
    });
  };

  if (isLoading && !retrying) {
    return (
      <div className="flex items-center justify-center py-12" data-testid="api-keys-loading">
        <Loader2 className="h-6 w-6 animate-spin text-[var(--text-secondary)]" />
      </div>
    );
  }

  // J7-D applies to the FORM BODY, not the page. The page header lives in the
  // route (keys.tsx), matching connection.tsx — otherwise the h1 gets narrowed
  // too and the heading jumps 160px as you move between settings tabs, which
  // makes a deliberate rule read as a bug.
  return (
    <div className="max-w-3xl">
      {/* Fail-soft, not SettingsErrorState: the form and the TMDB attribution
          must stay on screen (sub-2-1b CR — never a blank dead end; sub-6-9 —
          compliance is not conditional). What changed in dsr-3b is the words:
          no backend string, and a way to try again. */}
      {readFailed && (
        <div
          data-testid="api-keys-load-error"
          className="mb-6 flex items-start gap-3 rounded-[var(--radius-md)] bg-[var(--error-tint)] p-4"
        >
          <AlertTriangle
            className="mt-0.5 h-4 w-4 shrink-0 text-[var(--error-text)]"
            aria-hidden="true"
          />
          <div role="alert" key={errorUpdatedAt} className="flex-1">
            <p className="text-sm font-semibold text-[var(--error-text)]">無法讀取金鑰設定</p>
            <p className="mt-1 text-xs text-[var(--error-text)]">
              與後端的連線中斷了。已存的金鑰不受影響。
            </p>
          </div>
          <button
            type="button"
            onClick={() => {
              if (retrying) return;
              setRetrying(true);
              void refetch();
            }}
            aria-disabled={retrying || undefined}
            className="flex min-h-9 shrink-0 items-center gap-1.5 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-3 text-sm font-semibold text-[var(--text-primary)] max-sm:min-h-11"
          >
            <RefreshCw
              className={cn('h-4 w-4', retrying && 'motion-safe:animate-spin')}
              aria-hidden="true"
            />
            重試
          </button>
        </div>
      )}

      {/* AC #2 — read-only degradation. Controls stay VISIBLE and disabled with
          a reason; hiding them would leave the user with nothing to act on.
          dsr-3b (C21-D O9YinR): 硃砂, not 赭 — without ENCRYPTION_KEY nothing
          can be stored at all; that is a broken setup, not a request that
          quietly didn't happen. */}
      {settings && !writable && (
        <div
          data-testid="keys-not-writable"
          role="alert"
          className="mb-6 flex items-start gap-3 rounded-[var(--radius-md)] bg-[var(--error-tint)] p-4"
        >
          <Lock className="mt-0.5 h-4 w-4 shrink-0 text-[var(--error-text)]" aria-hidden="true" />
          {notWritableReason === REASON_ENCRYPTION_KEY_MISSING ? (
            <div>
              <p className="text-sm font-semibold text-[var(--error-text)]">
                未設定加密金鑰，無法安全儲存 API 金鑰
              </p>
              <p className="mt-1 text-xs text-[var(--error-text)]">
                請設定 ENCRYPTION_KEY 後重啟容器。在那之前，這一頁只能檢視，不能儲存。
              </p>
            </div>
          ) : (
            <p className="text-sm font-semibold text-[var(--error-text)]">
              目前無法儲存 API 金鑰，設定僅供檢視。
            </p>
          )}
        </div>
      )}

      {/* AC #3 — NFR-S3. Advisory, never a hard block: Vido ships over HTTP by
          default, so blocking outright would make the feature unusable for the
          audience it exists for. D9's own wording is warn + require confirmation.
          dsr-3b (C22-D b6rD2b): 赭 throughout — a title, a sentence, and an
          acknowledgement in the warning's own colour, never 泥金 (= 正在跑). */}
      {!isSecureContext && (
        <div
          data-testid="insecure-context-warning"
          role="status"
          aria-live="polite"
          className="mb-6 rounded-[var(--radius-md)] bg-[var(--warning-tint)] p-4"
        >
          <div className="flex items-start gap-3">
            <ShieldAlert
              className="mt-0.5 h-[18px] w-[18px] shrink-0 text-[var(--warning-text)]"
              aria-hidden="true"
            />
            <div>
              <p className="text-sm font-semibold text-[var(--warning-text)]">
                目前連線未加密（HTTP）
              </p>
              <p className="mt-1 text-xs text-[var(--warning-text)]">
                API 金鑰會以明文傳送到 NAS。建議先設定 HTTPS 反向代理。
              </p>
            </div>
          </div>
          <label className="mt-3 flex min-h-11 cursor-pointer items-center gap-2 pl-7 text-xs font-semibold text-[var(--warning-text)]">
            <input
              type="checkbox"
              checked={insecureAck}
              onChange={(e) => setInsecureAck(e.target.checked)}
              data-testid="insecure-context-ack"
              className="h-[18px] w-[18px] shrink-0 accent-[var(--warning-text)]"
            />
            我了解風險，仍要在未加密連線下儲存
          </label>
        </div>
      )}

      {/* The settings card. On a phone it dissolves and every key row becomes its
          own card (C7-M f8Fda) — same fields, same copy, nothing dropped. */}
      <div
        data-testid="api-keys-card"
        className="rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-6 max-sm:border-0 max-sm:bg-transparent max-sm:p-0"
      >
        <form onSubmit={handleSave} data-testid="api-keys-form">
          <div data-testid="key-rows" className="flex flex-col gap-4 max-sm:gap-3">
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
              const testBusy = testMutation.isPending && testingName === row.name;
              const labelClass = cn(
                'text-sm font-semibold',
                settings && !writable ? 'text-[var(--text-muted)]' : 'text-[var(--text-primary)]'
              );

              // One 測試 button, two homes: inside the input when there is one
              // (C7-D j6UfaO), beside 編輯 / 清除 when the key is stored and
              // there is no input to sit in.
              const testButton = row.testable ? (
                <button
                  type="button"
                  onClick={() => handleTest(row.name)}
                  // Writability is irrelevant here — probing stores nothing,
                  // so an operator without ENCRYPTION_KEY can still find out
                  // whether the env key they deployed actually works.
                  // The insecure gate applies only when a TYPED key would
                  // cross the wire; testing the resolved key sends none.
                  disabled={testBusy || (insecureBlocked && typedCandidate)}
                  data-testid={`key-test-${row.name}`}
                  className={showInput ? INLINE_TEST_BUTTON_CLASS : SECONDARY_BUTTON_CLASS}
                >
                  {testBusy ? (
                    <Loader2
                      className={cn('animate-spin', showInput ? 'h-3 w-3' : 'h-4 w-4')}
                      aria-hidden="true"
                    />
                  ) : (
                    !showInput && <Plug className="h-4 w-4" aria-hidden="true" />
                  )}
                  測試
                </button>
              ) : null;

              return (
                <div
                  key={row.name}
                  data-testid={`key-row-${row.name}`}
                  className="flex flex-col gap-2 max-sm:rounded-[var(--radius-lg)] max-sm:border max-sm:border-[var(--border-subtle)] max-sm:bg-[var(--bg-secondary)] max-sm:p-4"
                >
                  <div className="flex flex-wrap items-center gap-2">
                    {/* A stored key draws no input, so it must draw no <label for>
                        either — an orphan label is an a11y defect, not decoration. */}
                    {showInput ? (
                      <label htmlFor={inputId} className={labelClass}>
                        {row.label}
                      </label>
                    ) : (
                      <span className={labelClass}>{row.label}</span>
                    )}
                    <span
                      data-testid={`key-state-${row.name}`}
                      className={cn(
                        'rounded-full px-2 py-0.5 text-xs font-semibold',
                        stateToneClass(state)
                      )}
                    >
                      {/* CR sub-2-1b L2: a failed read is UNKNOWN, not "not
                          set" — a server outage must not badge a configured
                          key as 尚未設定. */}
                      {readFailed && !state ? '無法確認' : stateLabel(state)}
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

                  {showInput && (
                    <div className="flex gap-2 max-sm:flex-col">
                      <div className="relative flex-1">
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
                          // Room on the right for the inline 測試 (and the
                          // browser's own password-reveal icon beside it).
                          className={cn(INPUT_CLASS, row.testable && 'pr-24')}
                        />
                        {testButton}
                      </div>
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
                          className="min-h-11 shrink-0 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-4 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-primary)]"
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
                      className="flex flex-col gap-2 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] p-3 sm:flex-row sm:items-center"
                    >
                      <p className="flex-1 text-xs text-[var(--text-primary)]">
                        清除後將改用環境變數的金鑰；若環境變數也未設定，相關功能會停用。
                      </p>
                      <div className="flex shrink-0 gap-2">
                        <button
                          type="button"
                          onClick={() => handleClear(row.name)}
                          data-testid={`key-clear-confirm-yes-${row.name}`}
                          className="min-h-9 rounded-[var(--radius-md)] bg-[var(--error)] px-3 text-sm font-medium text-[var(--text-on-scrim)] transition-opacity hover:opacity-90 max-sm:min-h-11"
                        >
                          確認清除
                        </button>
                        <button
                          type="button"
                          onClick={() => setConfirmingClear(null)}
                          data-testid={`key-clear-cancel-${row.name}`}
                          className="min-h-9 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-3 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-primary)] max-sm:min-h-11"
                        >
                          取消
                        </button>
                      </div>
                    </div>
                  ) : (
                    !showInput && (
                      <div className="flex flex-wrap items-center gap-2">
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
                        {testButton}
                      </div>
                    )
                  )}

                  <p id={hintId} className="text-xs text-[var(--text-muted)]">
                    {row.hint}
                  </p>

                  {/* Honest about precedence: 2-1a resolves secret > env, so saving
                      here silently overrides the deploy-time variable. Say it. */}
                  {state?.source === 'env' && (
                    <p
                      data-testid={`key-env-override-note-${row.name}`}
                      className="text-xs text-[var(--info-text)]"
                    >
                      在此儲存的金鑰會覆蓋環境變數提供的設定。
                    </p>
                  )}

                  {result && (
                    <p
                      data-testid={`key-test-result-${row.name}`}
                      role="status"
                      aria-live="polite"
                      className={cn(
                        'text-sm',
                        result.ok ? 'text-[var(--success-text)]' : 'text-[var(--error-text)]'
                      )}
                    >
                      {result.message}
                    </p>
                  )}

                  {/* TMDB API Terms of Use §3 — the logo and the non-endorsement
                      notice belong wherever TMDB data is used, and this row is
                      where the operator accounts for that data source (sub-6-9). */}
                  {row.name === 'tmdb' && <TmdbAttribution className="mt-2" />}
                </div>
              );
            })}
          </div>

          {/* Said once, for the whole card (C7-D pegsz). Only Claude has a probe;
              nothing checks the other keys on save, so this says no more. */}
          <p className="mt-4 text-xs text-[var(--text-muted)]">僅 Claude 金鑰支援連線測試。</p>

          <div className="mt-4 flex flex-col items-end gap-3 max-sm:items-stretch">
            {saveSuccess && (
              // 固定詞彙: green means IN PROGRESS; a completed save is a neutral
              // report — same pattern as QBittorrentForm one tab away.
              <p
                role="status"
                aria-live="polite"
                className="flex items-center gap-1.5 text-sm text-[var(--text-secondary)]"
              >
                <Check className="h-4 w-4" aria-hidden="true" />
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
                'flex min-h-11 items-center justify-center gap-2 rounded-[var(--radius-md)] px-5 text-sm font-semibold transition-colors max-sm:w-full',
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
              儲存金鑰
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
