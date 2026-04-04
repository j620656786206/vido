import { useEffect, useRef, useState } from 'react';
import { Activity, Bell, Loader2 } from 'lucide-react';
import { useServiceStatuses, useTestServiceConnection } from '../../hooks/useServiceStatus';
import { ServiceStatusCard } from './ServiceStatusCard';
import type { ServiceConnectionStatus } from '../../services/serviceStatusService';

const STATUS_LABELS: Record<ServiceConnectionStatus, string> = {
  connected: '已連線',
  rate_limited: '速率限制',
  error: '錯誤',
  disconnected: '已斷線',
  unconfigured: '未設定',
};

interface StatusChange {
  displayName: string;
  from: string;
  to: string;
}

export function ServiceStatusDashboard() {
  const { data, isLoading, error } = useServiceStatuses();
  const testConnection = useTestServiceConnection();
  const [testingService, setTestingService] = useState<string | null>(null);
  const [testError, setTestError] = useState<string | null>(null);
  const [statusChanges, setStatusChanges] = useState<StatusChange[]>([]);
  const previousStatusesRef = useRef<Map<string, ServiceConnectionStatus> | null>(null);

  const services = data?.services ?? [];

  // Detect status changes between polling intervals
  useEffect(() => {
    if (services.length === 0) return;

    const currentMap = new Map<string, ServiceConnectionStatus>();
    for (const svc of services) {
      currentMap.set(svc.name, svc.status);
    }

    const prev = previousStatusesRef.current;
    if (prev !== null) {
      const changes: StatusChange[] = [];
      for (const svc of services) {
        const prevStatus = prev.get(svc.name);
        if (prevStatus && prevStatus !== svc.status) {
          changes.push({
            displayName: svc.displayName,
            from: STATUS_LABELS[prevStatus] || prevStatus,
            to: STATUS_LABELS[svc.status] || svc.status,
          });
        }
      }
      if (changes.length > 0) {
        setStatusChanges(changes);
      }
    }

    previousStatusesRef.current = currentMap;
  }, [services]);

  // Auto-dismiss notifications after 5 seconds
  useEffect(() => {
    if (statusChanges.length === 0) return;
    const timer = setTimeout(() => setStatusChanges([]), 5000);
    return () => clearTimeout(timer);
  }, [statusChanges]);

  const handleTest = async (serviceName: string) => {
    if (testingService) return; // Prevent concurrent test requests
    setTestingService(serviceName);
    setTestError(null);
    try {
      await testConnection.mutateAsync(serviceName);
    } catch (err) {
      setTestError(err instanceof Error ? err.message : '測試連線失敗');
    } finally {
      setTestingService(null);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20" data-testid="status-loading">
        <Loader2 className="h-8 w-8 animate-spin text-[var(--text-secondary)]" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="py-10 text-center" data-testid="status-error">
        <p className="text-[var(--error)]">無法載入服務狀態</p>
        <p className="mt-1 text-sm text-[var(--text-muted)]">{error.message}</p>
      </div>
    );
  }

  return (
    <div className="space-y-6" data-testid="service-status-dashboard">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Activity className="h-5 w-5 text-[var(--text-secondary)]" />
        <div>
          <h2 className="text-lg font-semibold text-[var(--text-primary)]">服務狀態</h2>
          <p className="text-sm text-[var(--text-secondary)]">監控外部服務連線狀態</p>
        </div>
      </div>

      {/* Status change notifications (AC3) */}
      {statusChanges.length > 0 && (
        <div
          className="flex items-start gap-3 rounded-lg border border-blue-800 bg-blue-900/20 px-4 py-3"
          role="status"
          aria-live="polite"
          data-testid="status-change-notification"
        >
          <Bell className="mt-0.5 h-4 w-4 shrink-0 text-[var(--accent-primary)]" />
          <div className="space-y-1 text-sm text-blue-300">
            {statusChanges.map((change, i) => (
              <p key={i}>
                {change.displayName}：{change.from} → {change.to}
              </p>
            ))}
          </div>
          <button
            onClick={() => setStatusChanges([])}
            className="ml-auto shrink-0 text-xs text-[var(--text-muted)] hover:text-[var(--text-secondary)]"
            data-testid="dismiss-notification"
          >
            關閉
          </button>
        </div>
      )}

      {/* Service cards */}
      <div className="space-y-3" data-testid="service-cards-list">
        {services.map((service) => (
          <ServiceStatusCard
            key={service.name}
            service={service}
            onTest={handleTest}
            isTesting={testingService === service.name}
          />
        ))}
      </div>

      {testError && (
        <div
          className="rounded-lg border border-red-800 bg-red-900/20 px-4 py-3 text-sm text-[var(--error)]"
          role="alert"
          data-testid="test-error"
        >
          {testError}
        </div>
      )}

      {services.length === 0 && (
        <div
          className="py-10 text-center text-sm text-[var(--text-muted)]"
          data-testid="status-empty"
        >
          沒有已設定的服務
        </div>
      )}
    </div>
  );
}
