import { useState } from 'react';
import { Trash2, Loader2 } from 'lucide-react';
import type { CacheTypeInfo } from '../../services/cacheService';

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const k = 1024;
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(k)), units.length - 1);
  const value = bytes / Math.pow(k, i);
  return `${value.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

interface CacheTypeCardProps {
  cacheType: CacheTypeInfo;
  onClear: (type: string) => Promise<void>;
}

export function CacheTypeCard({ cacheType, onClear }: CacheTypeCardProps) {
  const [confirming, setConfirming] = useState(false);
  const [clearing, setClearing] = useState(false);

  const handleClear = async () => {
    if (!confirming) {
      setConfirming(true);
      return;
    }
    setClearing(true);
    try {
      await onClear(cacheType.type);
    } finally {
      setClearing(false);
      setConfirming(false);
    }
  };

  const handleCancel = () => {
    setConfirming(false);
  };

  return (
    <div
      className="flex items-center justify-between rounded-lg border border-slate-700 bg-slate-800/50 px-4 py-3"
      data-testid={`cache-type-${cacheType.type}`}
    >
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium text-slate-200" data-testid="cache-type-label">
          {cacheType.label}
        </p>
        <p className="text-xs text-slate-400" data-testid="cache-type-size">
          {formatBytes(cacheType.sizeBytes)} · {cacheType.entryCount.toLocaleString()} 筆
        </p>
      </div>

      <div className="ml-4 flex items-center gap-2">
        {confirming && !clearing && (
          <button
            onClick={handleCancel}
            className="rounded px-2 py-1 text-xs text-slate-400 hover:text-slate-200"
            data-testid="cache-cancel-btn"
          >
            取消
          </button>
        )}
        <button
          onClick={handleClear}
          disabled={clearing}
          className={`flex items-center gap-1.5 rounded px-3 py-1.5 text-xs font-medium transition-colors ${
            confirming
              ? 'bg-red-600 text-white hover:bg-red-700'
              : 'bg-slate-700 text-slate-300 hover:bg-slate-600'
          } disabled:opacity-50`}
          data-testid="cache-clear-btn"
        >
          {clearing ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Trash2 className="h-3.5 w-3.5" />
          )}
          {confirming ? '確認清除' : '清除'}
        </button>
      </div>
    </div>
  );
}

export { formatBytes };
