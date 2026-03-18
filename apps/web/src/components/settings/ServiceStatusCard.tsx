import { useState } from 'react';
import {
  CheckCircle,
  AlertTriangle,
  XCircle,
  WifiOff,
  Settings,
  Loader2,
  RefreshCw,
} from 'lucide-react';
import type { ServiceStatus, ServiceConnectionStatus } from '../../services/serviceStatusService';

const statusConfig: Record<
  ServiceConnectionStatus,
  {
    color: string;
    bg: string;
    icon: React.ElementType;
    label: string;
  }
> = {
  connected: {
    color: 'text-green-400',
    bg: 'bg-green-400/10',
    icon: CheckCircle,
    label: '已連線',
  },
  rate_limited: {
    color: 'text-yellow-400',
    bg: 'bg-yellow-400/10',
    icon: AlertTriangle,
    label: '速率限制',
  },
  error: {
    color: 'text-red-400',
    bg: 'bg-red-400/10',
    icon: XCircle,
    label: '錯誤',
  },
  disconnected: {
    color: 'text-red-400',
    bg: 'bg-red-400/10',
    icon: WifiOff,
    label: '已斷線',
  },
  unconfigured: {
    color: 'text-gray-400',
    bg: 'bg-gray-400/10',
    icon: Settings,
    label: '未設定',
  },
};

interface ServiceStatusCardProps {
  service: ServiceStatus;
  onTest: (serviceName: string) => void;
  isTesting: boolean;
}

export function ServiceStatusCard({ service, onTest, isTesting }: ServiceStatusCardProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const config = statusConfig[service.status] || statusConfig.error;
  const StatusIcon = config.icon;

  const showDetail = service.status !== 'connected' && service.status !== 'unconfigured';

  return (
    <div
      className="rounded-lg border border-slate-700 bg-slate-800 p-4"
      data-testid={`service-card-${service.name}`}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className={`rounded-full p-2 ${config.bg}`}>
            <StatusIcon className={`h-5 w-5 ${config.color}`} />
          </div>
          <div>
            <h3 className="font-medium text-slate-200">{service.displayName}</h3>
            <div className="flex items-center gap-2 text-sm">
              <span className={config.color}>{config.label}</span>
              {service.responseTimeMs > 0 && service.status === 'connected' && (
                <span className="text-slate-500">{service.responseTimeMs}ms</span>
              )}
            </div>
          </div>
        </div>

        <button
          onClick={() => onTest(service.name)}
          disabled={isTesting}
          className="flex items-center gap-1.5 rounded-lg bg-slate-700 px-3 py-1.5 text-xs font-medium text-slate-300 transition-colors hover:bg-slate-600 disabled:opacity-50"
          data-testid={`test-btn-${service.name}`}
        >
          {isTesting ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <RefreshCw className="h-3.5 w-3.5" />
          )}
          測試連線
        </button>
      </div>

      {/* Expandable detail panel */}
      {showDetail && (
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="mt-2 text-xs text-slate-500 hover:text-slate-400"
          data-testid={`detail-toggle-${service.name}`}
        >
          {isExpanded ? '隱藏詳情' : '顯示詳情'}
        </button>
      )}

      {isExpanded && (
        <div
          className="mt-3 space-y-1 rounded-md bg-slate-900/50 p-3 text-xs text-slate-400"
          data-testid={`detail-panel-${service.name}`}
        >
          {service.errorMessage && (
            <p>
              <span className="text-slate-500">錯誤訊息：</span> {service.errorMessage}
            </p>
          )}
          {service.lastSuccessAt && (
            <p>
              <span className="text-slate-500">最後成功：</span>{' '}
              {new Date(service.lastSuccessAt).toLocaleString('zh-TW')}
            </p>
          )}
          <p>
            <span className="text-slate-500">最後檢查：</span>{' '}
            {service.lastCheckAt
              ? new Date(service.lastCheckAt).toLocaleString('zh-TW')
              : '尚未檢查'}
          </p>
        </div>
      )}
    </div>
  );
}
