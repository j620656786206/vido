/**
 * Global database-outage banner (bugfix-i-3-db-dead-returns-200).
 *
 * When the database dies, every data-backed call fails — historically each in
 * its own way, which made ONE data-layer incident look like ten distinct bugs
 * on the first NAS deploy. The backend now fails every /api/v1 call with a
 * uniform 503 DATABASE_UNAVAILABLE; this banner is the single comprehensible
 * message that explains all of them at once.
 *
 * It polls the ungated root /health endpoint (which answers even while the
 * database is down) and shows nothing while everything is fine. Recovery is
 * automatic server-side, so the banner clears itself on the next healthy poll.
 */
import { useQuery } from '@tanstack/react-query';

interface HealthResponse {
  status: string;
  database?: { status: string };
}

async function fetchHealth(): Promise<HealthResponse | null> {
  try {
    // The endpoint returns 503 with the same JSON body while unhealthy, so we
    // parse the body regardless of status instead of throwing.
    const res = await fetch('/health', { headers: { Accept: 'application/json' } });
    return (await res.json()) as HealthResponse;
  } catch {
    // Network-level failure is not the database's verdict — stay quiet rather
    // than cry wolf on a dropped Wi-Fi packet.
    return null;
  }
}

export function DatabaseUnavailableBanner() {
  const { data } = useQuery({
    queryKey: ['health', 'database-banner'],
    queryFn: fetchHealth,
    refetchInterval: 30_000,
    refetchIntervalInBackground: false,
    retry: false,
    staleTime: 0,
  });

  if (data?.database?.status !== 'unhealthy') return null;

  return (
    <div
      role="alert"
      data-testid="database-unavailable-banner"
      className="fixed inset-x-0 top-0 z-50 border-b border-[var(--error)] bg-[var(--error-tint)] px-4 py-2 text-center text-sm text-[var(--error-text)] backdrop-blur-sm"
    >
      <span className="font-semibold">資料庫目前無法使用</span>
      ｜大部分功能會暫時失效。請檢查 NAS 的儲存掛載與磁碟空間；系統每 30 秒會自動嘗試復原。
    </div>
  );
}
