import { useState } from 'react';
import { Activity, Loader2 } from 'lucide-react';
import { useServiceStatuses, useTestServiceConnection } from '../../hooks/useServiceStatus';
import { ServiceStatusCard } from './ServiceStatusCard';

export function ServiceStatusDashboard() {
  const { data, isLoading, error } = useServiceStatuses();
  const testConnection = useTestServiceConnection();
  const [testingService, setTestingService] = useState<string | null>(null);

  const handleTest = async (serviceName: string) => {
    setTestingService(serviceName);
    try {
      await testConnection.mutateAsync(serviceName);
    } catch {
      // Error handled by TanStack Query
    } finally {
      setTestingService(null);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20" data-testid="status-loading">
        <Loader2 className="h-8 w-8 animate-spin text-slate-400" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="py-10 text-center" data-testid="status-error">
        <p className="text-red-400">無法載入服務狀態</p>
        <p className="mt-1 text-sm text-slate-500">{error.message}</p>
      </div>
    );
  }

  const services = data?.services ?? [];

  return (
    <div className="space-y-6" data-testid="service-status-dashboard">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Activity className="h-5 w-5 text-slate-400" />
        <div>
          <h2 className="text-lg font-semibold text-slate-200">服務狀態</h2>
          <p className="text-sm text-slate-400">監控外部服務連線狀態</p>
        </div>
      </div>

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

      {services.length === 0 && (
        <div className="py-10 text-center text-sm text-slate-500" data-testid="status-empty">
          沒有已設定的服務
        </div>
      )}
    </div>
  );
}
