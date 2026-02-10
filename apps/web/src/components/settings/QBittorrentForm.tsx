import { useState } from 'react';
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
  const [initialized, setInitialized] = useState(false);
  const [testResult, setTestResult] = useState<TestResult | null>(null);
  const [saveSuccess, setSaveSuccess] = useState(false);

  // Populate form when config loads
  if (config && !initialized) {
    setHost(config.host || '');
    setUsername(config.username || '');
    setBasePath(config.basePath || '');
    setInitialized(true);
  }

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
        <Loader2 className="h-6 w-6 animate-spin text-slate-400" />
      </div>
    );
  }

  const isSubmitting = saveMutation.isPending || testMutation.isPending;

  return (
    <form onSubmit={handleSave} data-testid="qbittorrent-form">
      <div className="space-y-5">
        <div>
          <label htmlFor="qb-host" className="mb-1.5 block text-sm font-medium text-slate-300">
            Host URL
          </label>
          <input
            id="qb-host"
            type="text"
            value={host}
            onChange={(e) => setHost(e.target.value)}
            placeholder="http://192.168.1.100:8080"
            className="w-full rounded-md border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            required
          />
        </div>

        <div>
          <label htmlFor="qb-username" className="mb-1.5 block text-sm font-medium text-slate-300">
            使用者名稱
          </label>
          <input
            id="qb-username"
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="admin"
            className="w-full rounded-md border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            required
          />
        </div>

        <div>
          <label htmlFor="qb-password" className="mb-1.5 block text-sm font-medium text-slate-300">
            密碼
          </label>
          <input
            id="qb-password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••"
            className="w-full rounded-md border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            required
          />
        </div>

        <div>
          <label htmlFor="qb-basepath" className="mb-1.5 block text-sm font-medium text-slate-300">
            Base Path <span className="text-slate-500">（選填，反向代理用）</span>
          </label>
          <input
            id="qb-basepath"
            type="text"
            value={basePath}
            onChange={(e) => setBasePath(e.target.value)}
            placeholder="/qbittorrent"
            className="w-full rounded-md border border-slate-700 bg-slate-800 px-3 py-2 text-sm text-slate-100 placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
      </div>

      {testResult && <ConnectionTestResult {...testResult} />}

      {saveSuccess && !testResult && <p className="mt-4 text-sm text-green-400">設定已儲存</p>}

      {saveMutation.isError && !testResult && (
        <p className="mt-4 text-sm text-red-400">{saveMutation.error.message}</p>
      )}

      <div className="mt-6 flex gap-3">
        <button
          type="button"
          onClick={handleTestConnection}
          disabled={isSubmitting || !host || !username || !password}
          className={cn(
            'flex items-center gap-2 rounded-md px-4 py-2 text-sm font-medium transition-colors',
            isSubmitting || !host || !username || !password
              ? 'cursor-not-allowed bg-slate-700 text-slate-500'
              : 'bg-slate-700 text-slate-200 hover:bg-slate-600'
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
              ? 'cursor-not-allowed bg-blue-800 text-blue-400'
              : 'bg-blue-600 text-white hover:bg-blue-500'
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
