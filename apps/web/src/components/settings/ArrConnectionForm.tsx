// Design ref: ux-design.pen Screen C23-D (Qva0y) · C23-M (p37q9)
import { useEffect, useId, useState } from 'react';
import {
  AlertTriangle,
  Check,
  ChevronDown,
  CircleAlert,
  CircleCheck,
  CircleDashed,
  Loader2,
  Plug,
  PowerOff,
  RefreshCw,
  Save,
  TriangleAlert,
  type LucideIcon,
} from 'lucide-react';
import { cn } from '../../lib/utils';
import {
  useDvrConfig,
  useDvrQualityProfiles,
  useDvrRootFolders,
  useSaveDvrConfig,
  useTestDvrConnection,
} from '../../hooks/useDvrSettings';
import {
  DVR_AUTH_FAILED,
  DVR_CONNECTION_FAILED,
  DVR_NOT_CONFIGURED,
  DVR_TEST_FAILED,
  DVR_TIMEOUT,
  DvrSettingsApiError,
  type DvrConfig,
  type DvrPlugin,
} from '../../services/dvrSettings';

const PLUGIN_COPY: Record<
  DvrPlugin,
  { name: string; role: string; urlPlaceholder: string; listsHint: string }
> = {
  sonarr: {
    name: 'Sonarr',
    role: '影集的搜尋與匯入',
    urlPlaceholder: 'http://192.168.1.100:8989',
    listsHint: '送出影集請求時，Vido 用這兩個設定把它加進 Sonarr。',
  },
  radarr: {
    name: 'Radarr',
    role: '電影的搜尋與匯入',
    urlPlaceholder: 'http://192.168.1.100:7878',
    listsHint: '送出電影請求時，Vido 用這兩個設定把它加進 Radarr。',
  },
};

const HAS_CJK = /[㐀-鿿]/;

function reasonFor(code: string, name: string, detail?: string): string | null {
  switch (code) {
    case DVR_AUTH_FAILED:
      return `API 金鑰不對。到 ${name} 的 Settings → General 重新複製一次。`;
    case DVR_CONNECTION_FAILED:
      return `連不到這個網址。確認 ${name} 有在執行，網址和連接埠都正確。`;
    case DVR_TIMEOUT:
      return `${name} 太久沒有回應。`;
    case DVR_NOT_CONFIGURED:
      return `還沒有填 ${name} 的網址和 API 金鑰。`;
    case DVR_TEST_FAILED:
      // The plugin's own refusal (e.g. Sonarr v3) is written for people; keep its zh-TW part.
      return detail && HAS_CJK.test(detail) ? detail.split('—')[0].trim() : null;
    default:
      return null;
  }
}

/**
 * What the user can do about a failed test or save. A refused save arrives as DVR_TEST_FAILED
 * wrapping the real reason (`causeCode`), so the message names both: that nothing was stored,
 * and why.
 */
export function describeDvrError(error: Error, name: string, action: 'test' | 'save'): string {
  if (!(error instanceof DvrSettingsApiError)) return error.message || `無法連線到 ${name}`;
  const code = error.code === DVR_TEST_FAILED && error.causeCode ? error.causeCode : error.code;
  const why =
    reasonFor(code, name, error.suggestion) ??
    (action === 'save' && error.code === DVR_TEST_FAILED
      ? '按「測試連線」看看原因。'
      : error.message || `無法連線到 ${name}`);
  return action === 'save' && error.code === DVR_TEST_FAILED
    ? `連線測試沒有通過，設定沒有儲存。${why}`
    : why;
}

interface BadgeSpec {
  label: string;
  icon: LucideIcon;
  className: string;
}

const NEUTRAL = 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)]';

/**
 * The header badge says what is true right now. 已連線 is 青碧 (the check came back with an
 * answer), 連不上 is 硃砂; everything that is not a fault — never set up, switched off, not
 * checked yet — stays neutral.
 */
function healthBadge(config: DvrConfig): BadgeSpec {
  if (!config.url || !config.hasApiKey) {
    return { label: '未設定', icon: CircleDashed, className: NEUTRAL };
  }
  if (!config.enabled) {
    return { label: '已停用', icon: PowerOff, className: NEUTRAL };
  }
  switch (config.health.status) {
    case 'healthy':
      return {
        label: '已連線',
        icon: CircleCheck,
        className: 'bg-[var(--success-tint)] text-[var(--success-text)]',
      };
    case 'unhealthy':
      return {
        label: '連不上',
        icon: CircleAlert,
        className: 'bg-[var(--error-tint)] text-[var(--error-text)]',
      };
    default:
      return { label: '尚未檢查', icon: CircleDashed, className: NEUTRAL };
  }
}

const FIELD_LABEL = 'mb-2 block text-sm font-medium text-[var(--text-secondary)]';
const FIELD_HINT = 'mt-2 text-xs text-[var(--text-muted)]';
const CONTROL =
  'h-11 w-full rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-tertiary)] px-3 text-sm text-[var(--text-primary)] placeholder:text-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)] disabled:cursor-not-allowed disabled:bg-[var(--bg-secondary)] disabled:text-[var(--text-disabled)]';
const BUTTON =
  'inline-flex h-11 items-center justify-center gap-2 rounded-[var(--radius-md)] px-5 text-sm font-semibold transition-colors disabled:cursor-not-allowed disabled:bg-[var(--bg-tertiary)] disabled:text-[var(--text-disabled)] [&_svg]:size-4';

/** A warning about the saved setup, not a fault: no semantic colour (DESIGN.md §赭, 2026-09-11). */
function SetupNote({ children }: { children: React.ReactNode }) {
  return (
    <p
      data-testid="arr-setup-note"
      className="flex items-start gap-2 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] p-3 text-sm text-[var(--text-primary)]"
    >
      <TriangleAlert
        className="mt-0.5 size-4 shrink-0 text-[var(--text-secondary)]"
        aria-hidden="true"
      />
      {children}
    </p>
  );
}

/**
 * One Sonarr or Radarr connection card on 連線設定. Fields mirror the 13-4a config:
 * enabled, URL, API key (write-only), quality profile, root folder. The profile and folder
 * lists come from the plugin itself, through the SAVED and ENABLED config — so they unlock
 * after the first successful save, and the card says so instead of showing an empty list.
 */
export function ArrConnectionForm({ plugin }: { plugin: DvrPlugin }) {
  const copy = PLUGIN_COPY[plugin];
  const uid = useId();
  const {
    data: config,
    isLoading,
    isError,
    error: loadError,
    refetch,
    isFetching,
  } = useDvrConfig(plugin);
  const save = useSaveDvrConfig(plugin);
  const test = useTestDvrConnection(plugin);

  const [enabled, setEnabled] = useState(false);
  const [url, setUrl] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [qualityProfileId, setQualityProfileId] = useState(0);
  const [rootFolderPath, setRootFolderPath] = useState('');
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    if (!config) return;
    // A card that was never set up opens switched ON: typing a URL and key means you want it.
    const neverSetUp = !config.url && !config.hasApiKey;
    setEnabled(neverSetUp ? true : config.enabled);
    setUrl(config.url);
    setQualityProfileId(config.qualityProfileId);
    setRootFolderPath(config.rootFolderPath);
  }, [config]);

  const listsReady = Boolean(config?.enabled && config.url && config.hasApiKey);
  const profiles = useDvrQualityProfiles(plugin, listsReady);
  const folders = useDvrRootFolders(plugin, listsReady);
  const listsLoaded =
    listsReady &&
    profiles.data !== undefined &&
    folders.data !== undefined &&
    !profiles.isError &&
    !folders.isError;

  const titleId = `${uid}-title`;
  const enableLabelId = `${uid}-enable`;
  const badge = config ? healthBadge(config) : null;

  // The stored key only stands in for the server it was saved with. A new URL needs the key again,
  // so a typo'd address never receives it.
  const urlChanged = Boolean(config?.hasApiKey) && url.trim() !== (config?.url ?? '');
  const hasKey = apiKey !== '' || (Boolean(config?.hasApiKey) && !urlChanged);
  const busy = save.isPending || test.isPending;
  const canSubmit = url.trim() !== '' && hasKey && !busy;

  const profileMissing =
    listsLoaded &&
    qualityProfileId !== 0 &&
    !(profiles.data ?? []).some((p) => p.id === qualityProfileId);
  const folderMissing =
    listsLoaded &&
    rootFolderPath !== '' &&
    !(folders.data ?? []).some((f) => f.path === rootFolderPath);
  const savedIncomplete =
    listsLoaded &&
    Boolean(config?.enabled) &&
    (!config?.qualityProfileId || !config?.rootFolderPath);

  const touch = () => setSaved(false);

  const handleTest = () => {
    setTestResult(null);
    touch();
    test.mutate(
      { url: url.trim(), apiKey },
      {
        onSuccess: () => setTestResult({ success: true, message: `連得到 ${copy.name}` }),
        onError: (error) =>
          setTestResult({ success: false, message: describeDvrError(error, copy.name, 'test') }),
      }
    );
  };

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    setTestResult(null);
    touch();
    save.mutate(
      { url: url.trim(), apiKey, enabled, qualityProfileId, rootFolderPath },
      {
        onSuccess: () => {
          setSaved(true);
          setApiKey('');
        },
      }
    );
  };

  const listPlaceholder = !listsReady
    ? '啟用並儲存連線後可以選擇'
    : profiles.isError || folders.isError
      ? `讀不到 ${copy.name} 的清單`
      : null;

  const keyHint = urlChanged
    ? '換了網址，要重新貼上 API 金鑰。'
    : config?.hasApiKey
      ? '已儲存。留空表示不變更。'
      : `在 ${copy.name} 的 Settings → General 可以找到。`;

  return (
    <section
      aria-labelledby={titleId}
      data-testid={`arr-card-${plugin}`}
      className="max-w-3xl rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4 md:p-6"
    >
      <header className="mb-6 flex items-center justify-between gap-4">
        <div>
          <h2 id={titleId} className="text-base font-semibold text-[var(--text-primary)]">
            {copy.name}
          </h2>
          <p className="text-xs text-[var(--text-muted)]">{copy.role}</p>
        </div>
        {badge && (
          <span
            data-testid={`arr-health-${plugin}`}
            className={cn(
              'inline-flex shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-semibold',
              badge.className
            )}
          >
            <badge.icon className="size-3" aria-hidden="true" />
            {badge.label}
          </span>
        )}
      </header>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2
            className="size-6 animate-spin text-[var(--text-secondary)]"
            aria-label="載入中"
          />
        </div>
      ) : isError ? (
        // Same rule as the qBittorrent card: a failed read replaces the form. An empty form
        // would claim nothing is saved.
        <div
          role="status"
          className="flex items-start gap-3 rounded-[var(--radius-md)] bg-[var(--error-tint)] p-4"
        >
          <AlertTriangle
            className="mt-0.5 size-4 shrink-0 text-[var(--error-text)]"
            aria-hidden="true"
          />
          <div className="space-y-2">
            <p className="text-sm font-medium text-[var(--error-text)]">
              無法讀取 {copy.name} 設定
            </p>
            <p className="text-sm text-[var(--text-primary)]">
              {loadError?.message || '伺服器沒有回傳設定內容。設定可能仍然存在，這裡只是讀不到。'}
            </p>
            <button
              type="button"
              onClick={() => refetch()}
              disabled={isFetching}
              className={cn(BUTTON, 'bg-[var(--bg-tertiary)] text-[var(--text-primary)]')}
            >
              {isFetching ? <Loader2 className="animate-spin" /> : <RefreshCw />}
              重新載入
            </button>
          </div>
        </div>
      ) : (
        <form onSubmit={handleSave} data-testid={`arr-form-${plugin}`}>
          {/* Locked while a test or save is in flight, so the form cannot change under the request. */}
          <fieldset disabled={busy} className="m-0 min-w-0 space-y-4 border-0 p-0">
            <div className="flex items-center justify-between gap-4">
              <div>
                <span
                  id={enableLabelId}
                  className="block text-sm font-medium text-[var(--text-secondary)]"
                >
                  啟用
                </span>
                <span className="text-xs text-[var(--text-muted)]">
                  {enabled ? '每分鐘檢查一次連線。' : '關閉時 Vido 不會連線到它。'}
                </span>
              </div>
              <button
                type="button"
                role="switch"
                aria-checked={enabled}
                aria-labelledby={enableLabelId}
                onClick={() => {
                  setEnabled((v) => !v);
                  touch();
                }}
                className="flex size-11 shrink-0 items-center justify-center disabled:cursor-not-allowed"
              >
                <span
                  className={cn(
                    'flex h-6 w-11 items-center rounded-full p-1 transition-colors',
                    enabled ? 'bg-[var(--accent-primary)]' : 'bg-[var(--bg-tertiary)]'
                  )}
                >
                  <span
                    className={cn(
                      'size-4 rounded-full transition-transform',
                      enabled
                        ? 'translate-x-5 bg-[var(--text-on-accent)]'
                        : 'translate-x-0 bg-[var(--text-muted)]'
                    )}
                  />
                </span>
              </button>
            </div>

            <div>
              <label htmlFor={`${uid}-url`} className={FIELD_LABEL}>
                網址
              </label>
              {/* type="text": a native url field refuses「192.168.1.100:8989」with a browser bubble. */}
              <input
                id={`${uid}-url`}
                type="text"
                inputMode="url"
                value={url}
                onChange={(e) => {
                  setUrl(e.target.value);
                  setTestResult(null);
                  touch();
                }}
                placeholder={copy.urlPlaceholder}
                autoComplete="off"
                spellCheck={false}
                className={CONTROL}
              />
            </div>

            <div>
              <label htmlFor={`${uid}-api-key`} className={FIELD_LABEL}>
                API 金鑰
              </label>
              <input
                id={`${uid}-api-key`}
                type="password"
                value={apiKey}
                onChange={(e) => {
                  setApiKey(e.target.value);
                  setTestResult(null);
                  touch();
                }}
                placeholder={
                  config?.hasApiKey && !urlChanged
                    ? '••••••••••••••••'
                    : `貼上 ${copy.name} 的 API 金鑰`
                }
                autoComplete="off"
                className={CONTROL}
              />
              <p className={FIELD_HINT}>{keyHint}</p>
            </div>

            <div>
              <label htmlFor={`${uid}-profile`} className={FIELD_LABEL}>
                品質設定檔
              </label>
              <div className="relative">
                <select
                  id={`${uid}-profile`}
                  value={listPlaceholder ? '' : String(qualityProfileId)}
                  onChange={(e) => {
                    setQualityProfileId(Number(e.target.value));
                    touch();
                  }}
                  disabled={Boolean(listPlaceholder) || profiles.isLoading}
                  className={cn(CONTROL, 'appearance-none pr-9')}
                >
                  {listPlaceholder ? (
                    <option value="">{listPlaceholder}</option>
                  ) : (
                    <>
                      <option value="0">請選擇</option>
                      {profileMissing && (
                        <option value={String(qualityProfileId)}>
                          {`找不到這個設定檔（#${qualityProfileId}）`}
                        </option>
                      )}
                      {(profiles.data ?? []).map((p) => (
                        <option key={p.id} value={String(p.id)}>
                          {p.name}
                        </option>
                      ))}
                    </>
                  )}
                </select>
                <ChevronDown
                  className="pointer-events-none absolute right-3 top-1/2 size-4 -translate-y-1/2 text-[var(--text-muted)]"
                  aria-hidden="true"
                />
              </div>
            </div>

            <div>
              <label htmlFor={`${uid}-folder`} className={FIELD_LABEL}>
                根資料夾
              </label>
              <div className="relative">
                <select
                  id={`${uid}-folder`}
                  value={listPlaceholder ? '' : rootFolderPath}
                  onChange={(e) => {
                    setRootFolderPath(e.target.value);
                    touch();
                  }}
                  disabled={Boolean(listPlaceholder) || folders.isLoading}
                  className={cn(CONTROL, 'appearance-none pr-9')}
                >
                  {listPlaceholder ? (
                    <option value="">{listPlaceholder}</option>
                  ) : (
                    <>
                      <option value="">請選擇</option>
                      {folderMissing && (
                        <option value={rootFolderPath}>
                          {`${rootFolderPath}（在 ${copy.name} 找不到）`}
                        </option>
                      )}
                      {(folders.data ?? []).map((f) => (
                        <option key={f.id} value={f.path}>
                          {f.path}
                        </option>
                      ))}
                    </>
                  )}
                </select>
                <ChevronDown
                  className="pointer-events-none absolute right-3 top-1/2 size-4 -translate-y-1/2 text-[var(--text-muted)]"
                  aria-hidden="true"
                />
              </div>
              <p className={FIELD_HINT}>
                {listsLoaded && (folders.data ?? []).length === 0
                  ? `${copy.name} 裡還沒有根資料夾，先到 ${copy.name} 的 Settings → Media Management 新增。`
                  : copy.listsHint}
              </p>
            </div>

            {(profileMissing || folderMissing) && (
              <SetupNote>
                已儲存的品質設定檔或根資料夾在 {copy.name} 裡找不到了，請重新選擇後儲存。
              </SetupNote>
            )}
            {savedIncomplete && !profileMissing && !folderMissing && (
              <SetupNote>
                還沒選品質設定檔或根資料夾：Vido 送出的請求會停在等待中。選好之後再儲存一次。
              </SetupNote>
            )}

            {testResult && (
              <p
                role="status"
                data-testid={`arr-test-result-${plugin}`}
                className={cn(
                  'flex items-start gap-2 rounded-[var(--radius-md)] p-3 text-sm',
                  testResult.success
                    ? 'bg-[var(--success-tint)] text-[var(--success-text)]'
                    : 'bg-[var(--error-tint)] text-[var(--error-text)]'
                )}
              >
                {testResult.success ? (
                  <CircleCheck className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
                ) : (
                  <CircleAlert className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
                )}
                {testResult.message}
              </p>
            )}

            {saved && !testResult && (
              <p
                role="status"
                className="flex items-center gap-1.5 text-sm text-[var(--text-secondary)]"
              >
                <Check className="size-4" aria-hidden="true" />
                設定已儲存
              </p>
            )}

            {save.isError && !testResult && (
              // 赭: you asked for a save and it did not happen.
              <p role="alert" className="text-sm text-[var(--warning-text)]">
                {describeDvrError(save.error, copy.name, 'save')}
              </p>
            )}

            <div className="grid grid-cols-2 gap-2 md:flex md:justify-end md:gap-3">
              <button
                type="button"
                onClick={handleTest}
                disabled={!canSubmit}
                className={cn(BUTTON, 'bg-[var(--bg-tertiary)] text-[var(--text-primary)]')}
              >
                {test.isPending ? <Loader2 className="animate-spin" /> : <Plug />}
                測試連線
              </button>
              <button
                type="submit"
                disabled={!canSubmit}
                className={cn(
                  BUTTON,
                  'bg-[var(--accent-primary)] text-[var(--text-on-accent)] hover:bg-[var(--accent-hover)]'
                )}
              >
                {save.isPending ? <Loader2 className="animate-spin" /> : <Save />}
                儲存設定
              </button>
            </div>
          </fieldset>
        </form>
      )}
    </section>
  );
}
