// Design ref: ux-design.pen Screen C8-D (wqcqY) · C8-M (qx8Ma)；載入骨架見 C15-D (XwdOH) · C15-M (wkUNt)，整頁載入失敗見 C16-D (uYGBU)
import { Fragment, useEffect, useRef, useState } from 'react';
import { Link } from '@tanstack/react-router';
import { Bell, ChevronRight, CircleAlert, RefreshCw } from 'lucide-react';
import { useServiceStatuses, useTestServiceConnection } from '../../hooks/useServiceStatus';
import { ServiceStatusCard } from './ServiceStatusCard';
import { SettingsErrorState } from './SettingsErrorState';
import { Skeleton } from '../ui/Skeleton';
import { cn } from '../../lib/utils';
import { STATUS_LABELS, getServiceLabel, isBroken, type ServiceFixHint } from './serviceLabels';
import type { ServiceConnectionStatus, ServiceStatus } from '../../services/serviceStatusService';

interface StatusChange {
  name: string;
  from: string;
  to: string;
}

export function ServiceStatusDashboard() {
  const { data, isLoading, error, refetch, isFetching } = useServiceStatuses();
  const testConnection = useTestServiceConnection();
  const [testingService, setTestingService] = useState<string | null>(null);
  const [retestingAll, setRetestingAll] = useState(false);
  const [testError, setTestError] = useState<string | null>(null);
  const [statusChanges, setStatusChanges] = useState<StatusChange[]>([]);
  const previousStatusesRef = useRef<Map<string, ServiceConnectionStatus> | null>(null);
  // A failed read with nothing cached goes back to pending the moment it is
  // refetched (project_tanstack_refetch_no_data_resets_pending). Without this,
  // 重試 would swap the error state for the skeleton and 重試中… never shows.
  const [retrying, setRetrying] = useState(false);
  useEffect(() => {
    if (retrying && !isFetching) setRetrying(false);
  }, [retrying, isFetching]);

  const services = data?.services ?? [];
  const broken = services.filter((s) => isBroken(s.status));

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
            name: getServiceLabel(svc).name,
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

  // One at a time through the same per-service test — no new endpoint.
  const handleRetestBroken = async () => {
    if (retestingAll || testingService) return;
    setRetestingAll(true);
    try {
      for (const svc of broken) await handleTest(svc.name);
    } finally {
      setRetestingAll(false);
    }
  };

  if (isLoading && !retrying) return <ServiceStatusSkeleton />;

  if (error || retrying) {
    return (
      <SettingsErrorState
        testId="status-error"
        title="無法載入服務狀態"
        description="與後端的連線中斷了。這不影響已在執行的背景工作。"
        isRetrying={retrying}
        onRetry={() => {
          setRetrying(true);
          void refetch();
        }}
      />
    );
  }

  return (
    <div className="space-y-4" data-testid="service-status-dashboard">
      {/* Status change notifications (AC3) */}
      {statusChanges.length > 0 && (
        <div
          className="flex items-start gap-3 rounded-lg bg-[var(--accent-tint)] px-4 py-3"
          role="status"
          aria-live="polite"
          data-testid="status-change-notification"
        >
          <Bell className="mt-0.5 h-4 w-4 shrink-0 text-[var(--accent-text)]" />
          <div className="space-y-1 text-sm text-[var(--accent-text)]">
            {statusChanges.map((change, i) => (
              <p key={i}>
                {change.name}：{change.from} → {change.to}
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

      {/* The live region is always mounted and only its CONTENT is conditional:
          a region that appears already filled is not reliably announced, and
          role="alert" would re-announce on every 30s poll that re-renders it.
          polite + mount-once = read out once, when the first service breaks. */}
      <div aria-live="polite" data-testid="service-error-banner-region">
        {broken.length > 0 && (
          <BrokenServicesBanner
            broken={broken}
            onRetest={handleRetestBroken}
            isRetesting={retestingAll}
          />
        )}
      </div>

      {testError && (
        <div
          className="rounded-lg bg-[var(--error-tint)] px-4 py-3 text-sm text-[var(--error-text)]"
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

const SKELETON_ROWS = 5;

/** C15-D UTfMO / C15-M: five rows the shape of the cards (the 36px button box only on desktop). */
export function ServiceStatusSkeleton() {
  return (
    <div className="space-y-3" aria-busy="true" aria-label="載入中" data-testid="status-loading">
      {Array.from({ length: SKELETON_ROWS }, (_, i) => (
        <div
          key={i}
          className="flex h-16 items-center gap-4 rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4 sm:h-[72px]"
          data-testid="status-skeleton-row"
        >
          <div className="flex-1 space-y-2">
            <Skeleton className="h-[18px] w-40 rounded-[var(--radius-sm)]" />
            <Skeleton className="h-3.5 w-60 max-w-full rounded-[var(--radius-sm)]" />
          </div>
          <Skeleton className="h-6 w-[84px] shrink-0 rounded-[var(--radius-sm)]" />
          <Skeleton className="hidden size-9 shrink-0 rounded-[var(--radius-sm)] sm:block" />
        </div>
      ))}
    </div>
  );
}

function HintText({ hint }: { hint: ServiceFixHint }) {
  if (!hint.link) return <>{hint.text}</>;
  const marker = `「${hint.link.label}」`;
  const [before, after] = hint.text.split(marker);
  return (
    <>
      {before}「
      <Link
        to={hint.link.to}
        className="underline underline-offset-2 hover:text-[var(--text-primary)]"
      >
        {hint.link.label}
      </Link>
      」{after}
    </>
  );
}

interface BrokenServicesBannerProps {
  broken: ServiceStatus[];
  onRetest: () => void;
  isRetesting: boolean;
}

// C8-D blwxn / C8-M yNVad: one banner under the list, one line per broken
// service with advice that is true whatever the cause. The backend's raw error
// stays reachable in 技術細節 — it is the last clue when the advice is not enough.
function BrokenServicesBanner({ broken, onRetest, isRetesting }: BrokenServicesBannerProps) {
  const details = broken.filter((s) => s.errorMessage || s.message);
  return (
    <div
      className="grid grid-cols-[16px_1fr] gap-x-2 gap-y-2 rounded-[var(--radius-md)] bg-[var(--error-tint)] p-3 text-xs text-[var(--error-text)] sm:grid-cols-[16px_1fr_auto] sm:gap-x-3 sm:px-4"
      data-testid="service-error-banner"
    >
      <CircleAlert className="mt-0.5 size-4" aria-hidden="true" />
      <div className="min-w-0 space-y-2">
        <ul className="space-y-1">
          {broken.map((svc) => {
            const label = getServiceLabel(svc);
            return (
              <li key={svc.name} data-testid={`broken-line-${svc.name}`}>
                {label.name}：目前無法連線。
                {label.fixHint && <HintText hint={label.fixHint} />}
              </li>
            );
          })}
        </ul>
        {details.length > 0 && (
          <details className="group" data-testid="service-error-details">
            <summary className="inline-flex cursor-pointer list-none items-center gap-1 font-semibold [&::-webkit-details-marker]:hidden">
              <ChevronRight
                className="size-3 transition-transform group-open:rotate-90"
                aria-hidden="true"
              />
              技術細節
            </summary>
            <dl className="mt-2 grid grid-cols-[auto_1fr] gap-x-2 gap-y-1 font-mono break-all">
              {details.map((svc) => (
                <Fragment key={svc.name}>
                  <dt>{getServiceLabel(svc).name}</dt>
                  <dd>
                    {svc.errorMessage || svc.message}
                    {/* 6-4 AC #2: an error still says when it last worked. */}
                    {svc.lastSuccessAt && (
                      <span className="block">
                        最後成功：{new Date(svc.lastSuccessAt).toLocaleString('zh-TW')}
                      </span>
                    )}
                  </dd>
                </Fragment>
              ))}
            </dl>
          </details>
        )}
      </div>
      <button
        type="button"
        onClick={onRetest}
        aria-disabled={isRetesting || undefined}
        className="col-start-2 inline-flex h-8 items-center gap-1 justify-self-start rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-3 font-semibold text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] aria-disabled:cursor-not-allowed aria-disabled:text-[var(--text-muted)] max-sm:h-11 sm:col-start-3 sm:row-start-1"
        data-testid="retest-broken"
      >
        <RefreshCw
          className={cn('size-3.5', isRetesting && 'motion-safe:animate-spin')}
          aria-hidden="true"
        />
        重新檢查
      </button>
    </div>
  );
}
