import { useState, useEffect } from 'react';
import { Loader2, Plug, Save } from 'lucide-react';
import { cn } from '../../lib/utils';
import {
  useQBittorrentConfig,
  useSaveQBConfig,
  useTestQBConnection,
} from '../../hooks/useQBittorrent';
import { ConnectionTestResult } from './ConnectionTestResult';

interface TestResult {
  success: boolean;
  message: string;
  version?: string;
  apiVersion?: string;
}

export function QBittorrentForm() {
  const { data: config, isLoading } = useQBittorrentConfig();
  const saveMutation = useSaveQBConfig();
  const testMutation = useTestQBConnection();

  const [host, setHost] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [basePath, setBasePath] = useState('');
  const [testResult, setTestResult] = useState<TestResult | null>(null);
  const [saveSuccess, setSaveSuccess] = useState(false);

  // Populate form when config loads
  useEffect(() => {
    if (config) {
      setHost(config.host || '');
      setUsername(config.username || '');
      setBasePath(config.basePath || '');
    }
  }, [config]);

  const handleTestConnection = () => {
    setTestResult(null);
    testMutation.mutate(
      { host, username, password, basePath },
      {
        onSuccess: (info) => {
          setTestResult({
            success: true,
            message: '連線成功！',
            version: info.appVersion,
            apiVersion: info.apiVersion,
          });
        },
        onError: (error) => {
          setTestResult({
            success: false,
            message: error.message || '連線失敗',
          });
        },
      }
    );
  };

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    setSaveSuccess(false);
    saveMutation.mutate(
      { host, username, password, basePath },
      {
        onSuccess: () => setSaveSuccess(true),
      }
    );
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-[var(--text-secondary)]" />
      </div>
    );
  }

  const isSubmitting = saveMutation.isPending || testMutation.isPending;

  return (
    <form onSubmit={handleSave} data-testid="qbittorrent-form">
      <div className="space-y-5">
        <div>
          <label
            htmlFor="qb-host"
            className="mb-1.5 block text-sm font-medium text-[var(--text-secondary)]"
          >
            主機位址
          </label>
          <input
            id="qb-host"
            type="text"
            value={host}
            onChange={(e) => setHost(e.target.value)}
            placeholder="http://192.168.1.100:8080"
            className="w-full rounded-md border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
            required
          />
        </div>

        <div>
          <label
            htmlFor="qb-username"
            className="mb-1.5 block text-sm font-medium text-[var(--text-secondary)]"
          >
            使用者名稱
          </label>
          <input
            id="qb-username"
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="admin"
            className="w-full rounded-md border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
            required
          />
        </div>

        <div>
          <label
            htmlFor="qb-password"
            className="mb-1.5 block text-sm font-medium text-[var(--text-secondary)]"
          >
            密碼
          </label>
          <input
            id="qb-password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••"
            className="w-full rounded-md border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
            required
          />
        </div>

        <div>
          <label
            htmlFor="qb-basepath"
            className="mb-1.5 block text-sm font-medium text-[var(--text-secondary)]"
          >
            Base Path <span className="text-[var(--text-muted)]">（選填，反向代理用）</span>
          </label>
          <input
            id="qb-basepath"
            type="text"
            value={basePath}
            onChange={(e) => setBasePath(e.target.value)}
            placeholder="/qbittorrent"
            className="w-full rounded-md border border-[var(--border-subtle)] bg-[var(--bg-secondary)] px-3 py-2 text-sm text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-[var(--accent-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--accent-primary)]"
          />
        </div>
      </div>

      {testResult && <ConnectionTestResult {...testResult} />}

      {saveSuccess && !testResult && (
        <p className="mt-4 text-sm text-[var(--success)]">設定已儲存</p>
      )}

      {saveMutation.isError && !testResult && (
        <p className="mt-4 text-sm text-[var(--error)]">{saveMutation.error.message}</p>
      )}

      <div className="mt-6 flex flex-col gap-3 md:flex-row md:justify-end">
        <button
          type="button"
          onClick={handleTestConnection}
          disabled={isSubmitting || !host || !username || !password}
          className={cn(
            'flex items-center gap-2 rounded-md px-4 py-2 text-sm font-medium transition-colors',
            isSubmitting || !host || !username || !password
              ? 'cursor-not-allowed bg-[var(--bg-tertiary)] text-[var(--text-muted)]'
              : 'bg-[var(--bg-tertiary)] text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]'
          )}
        >
          {testMutation.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Plug className="h-4 w-4" />
          )}
          測試連線
        </button>

        <button
          type="submit"
          disabled={isSubmitting || !host || !username || !password}
          className={cn(
            'flex items-center gap-2 rounded-md px-4 py-2 text-sm font-medium transition-colors',
            isSubmitting || !host || !username || !password
              ? 'cursor-not-allowed bg-blue-800 text-[var(--accent-primary)]'
              : 'bg-[var(--accent-primary)] text-white hover:bg-[var(--accent-hover)]'
          )}
        >
          {saveMutation.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Save className="h-4 w-4" />
          )}
          儲存設定
        </button>
      </div>
    </form>
  );
}
