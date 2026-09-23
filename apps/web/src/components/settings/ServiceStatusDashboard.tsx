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
  const { data, isFetched, refetch, isFetching } = useServiceStatuses();
  const testConnection = useTestServiceConnection();
  const [testingService, setTestingService] = useState<string | null>(null);
  const [retestingAll, setRetestingAll] = useState(false);
  const [testError, setTestError] = useState<string | null>(null);
  const [statusChanges, setStatusChanges] = useState<StatusChange[]>([]);
  const previousStatusesRef = useRef<Map<string, ServiceConnectionStatus> | null>(null);
  const listRef = useRef<HTMLDivElement>(null);

  const services = data?.services ?? [];
  const broken = services.filter((s) => isBroken(s.status));
  // The same set the old per-card 顯示詳情 covered: anything neither fine nor
  // deliberately off. rate_limited is not "broken", but its raw error is still
  // the clue someone opening this page came for (6-4 AC #2).
  const detailed = services.filter(
    (s) => s.status !== 'connected' && s.status !== 'unconfigured' && (s.errorMessage || s.message)
  );

  // A retest that fixes everything unmounts the banner under the focused
  // 重新檢查 — hand focus to the list instead of dropping it on <body>.
  const hadBrokenRef = useRef(false);
  useEffect(() => {
    const lostFocus = document.activeElement === document.body;
    if (hadBrokenRef.current && broken.length === 0 && lostFocus) listRef.current?.focus();
    hadBrokenRef.current = broken.length > 0;
  }, [broken.length]);

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

  /** Resolves true when the test request itself went through. */
  const runTest = async (serviceName: string): Promise<boolean> => {
    setTestingService(serviceName);
    try {
      await testConnection.mutateAsync(serviceName);
      return true;
    } catch {
      return false;
    } finally {
      setTestingService(null);
    }
  };

  // The request failing is not the service failing — and its message is the
  // backend's (or the browser's) English, so it stays off the page.
  const reportFailures = (names: string[]) => {
    if (names.length === 0) return setTestError(null);
    const zh = names.map(
      (n) => getServiceLabel(services.find((s) => s.name === n) ?? { name: n, displayName: n }).name
    );
    setTestError(`無法重新檢查 ${zh.join('、')}，請稍後再試。`);
  };

  const handleTest = async (serviceName: string) => {
    if (testingService || retestingAll) return; // Prevent concurrent test requests
    setTestError(null);
    reportFailures((await runTest(serviceName)) ? [] : [serviceName]);
  };

  // One at a time through the same per-service test — no new endpoint. Failures
  // are collected and reported once, so a later success cannot erase an earlier miss.
  const handleRetestBroken = async () => {
    if (retestingAll || testingService) return;
    setRetestingAll(true);
    setTestError(null);
    const failed: string[] = [];
    try {
      for (const svc of broken) if (!(await runTest(svc.name))) failed.push(svc.name);
    } finally {
      setRetestingAll(false);
      reportFailures(failed);
    }
  };

  // Keyed on data, not on error/isLoading. A failed read with nothing cached
  // drops back to pending on EVERY refetch — the 30s poll included
  // (project_tanstack_refetch_no_data_resets_pending) — so keying the skeleton
  // on isLoading made the error page flash to the skeleton and back, taking
  // keyboard focus with it. isFetched stays true once any attempt has finished.
  if (!data && !isFetched) return <ServiceStatusSkeleton />;

  // Only when there is nothing to show. A failed background poll keeps the last
  // good list on screen instead of replacing it with a full-page error.
  if (!data) {
    return (
      <SettingsErrorState
        testId="status-error"
        title="無法載入服務狀態"
        description="與後端的連線中斷了。這不影響已在執行的背景工作。"
        isRetrying={isFetching}
        onRetry={() => void refetch()}
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
      <div
        ref={listRef}
        tabIndex={-1}
        className="space-y-3 focus:outline-none"
        data-testid="service-cards-list"
      >
        {services.map((service) => (
          <ServiceStatusCard
            key={service.name}
            service={service}
            onTest={handleTest}
            isTesting={testingService === service.name}
          />
        ))}
      </div>

      {/* The announcement is a short, always-mounted sentence, not the banner:
          a region that mounts already filled is not reliably read, role="alert"
          would re-announce on every 30s poll, and the banner itself would read
          out 技術細節 and the button label too. Announces when the set of broken
          services changes after the page is up; on first load the banner is
          simply there to read. */}
      <p className="sr-only" aria-live="polite" data-testid="service-error-announcement">
        {broken.length > 0 &&
          `${broken.map((s) => getServiceLabel(s).name).join('、')} 目前無法連線。`}
      </p>
      {broken.length > 0 ? (
        <BrokenServicesBanner
          broken={broken}
          detailed={detailed}
          onRetest={handleRetestBroken}
          isRetesting={retestingAll}
        />
      ) : (
        detailed.length > 0 && (
          <div className="px-1 text-xs text-[var(--text-muted)]">
            <TechnicalDetails services={detailed} />
          </div>
        )
      )}

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
  const marker = hint.link ? `「${hint.link.label}」` : '';
  const at = marker ? hint.text.indexOf(marker) : -1;
  // A sentence that does not quote its link label stays plain text rather than
  // growing a stray link (serviceLabels.spec pins that every label is quoted).
  if (!hint.link || at < 0) return <>{hint.text}</>;
  const before = hint.text.slice(0, at);
  const after = hint.text.slice(at + marker.length);
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

/** The backend's raw words, kept for troubleshooting but collapsed by default. */
function TechnicalDetails({ services }: { services: ServiceStatus[] }) {
  return (
    <details className="group" data-testid="service-error-details">
      <summary className="inline-flex cursor-pointer list-none items-center gap-1 font-semibold [&::-webkit-details-marker]:hidden">
        <ChevronRight
          className="size-3 transition-transform group-open:rotate-90"
          aria-hidden="true"
        />
        技術細節
      </summary>
      <dl className="mt-2 grid grid-cols-[auto_1fr] gap-x-2 gap-y-1 font-mono break-all">
        {services.map((svc) => (
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
  );
}

interface BrokenServicesBannerProps {
  broken: ServiceStatus[];
  detailed: ServiceStatus[];
  onRetest: () => void;
  isRetesting: boolean;
}

// C8-D blwxn / C8-M yNVad: one banner under the list, one line per broken
// service with advice that is true whatever the cause. The backend's raw error
// stays reachable in 技術細節 — it is the last clue when the advice is not enough.
function BrokenServicesBanner({
  broken,
  detailed,
  onRetest,
  isRetesting,
}: BrokenServicesBannerProps) {
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
        {detailed.length > 0 && <TechnicalDetails services={detailed} />}
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
