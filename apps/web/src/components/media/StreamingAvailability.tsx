// Design ref: ux-design.pen — no current screen frame; Epic 12 detail-page streaming-availability section postdates the .pen design
import { getImageUrl } from '../../lib/image';
import type { WatchProvider, WatchProviderRegion } from '../../types/library';

export interface StreamingAvailabilityProps {
  /** The region's provider groups (already filtered to one region upstream). */
  region?: WatchProviderRegion;
  isLoading?: boolean;
  isError?: boolean;
  onRetry?: () => void;
}

// Single source of truth for the section heading id so the <section> landmark is
// labelled by the visible <h2> (aria-labelledby) — mirrors RelatedContent (12-3).
const HEADING_ID = 'streaming-availability-heading';

// Monetization groups in display order. flatrate (subscription) is primary.
const GROUPS: {
  key: keyof Pick<WatchProviderRegion, 'flatrate' | 'rent' | 'buy'>;
  label: string;
}[] = [
  { key: 'flatrate', label: '訂閱' },
  { key: 'rent', label: '租借' },
  { key: 'buy', label: '購買' },
];

function Heading() {
  return (
    <h2 id={HEADING_ID} className="text-lg font-semibold text-[var(--text-primary)]">
      可在哪裡觀看
    </h2>
  );
}

function ProviderLogo({ provider }: { provider: WatchProvider }) {
  const url = getImageUrl(provider.logoPath, 'w92');
  if (!url) {
    // Logo missing — fall back to a readable name chip (never a broken image).
    return (
      <span className="rounded-md bg-[var(--bg-secondary)] px-2 py-1 text-xs text-[var(--text-primary)]">
        {provider.providerName}
      </span>
    );
  }
  return (
    <img
      src={url}
      alt={provider.providerName}
      title={provider.providerName}
      width={40}
      height={40}
      loading="lazy"
      className="h-10 w-10 rounded-md object-cover"
    />
  );
}

function ProviderGroup({ label, providers }: { label: string; providers: WatchProvider[] }) {
  // Sort each group by TMDB displayPriority (lower = shown first).
  const sorted = [...providers].sort((a, b) => a.displayPriority - b.displayPriority);
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-xs font-medium text-[var(--text-secondary)]">{label}</span>
      <div className="flex flex-wrap gap-2">
        {sorted.map((p) => (
          <ProviderLogo key={`${label}-${p.providerId}`} provider={p} />
        ))}
      </div>
    </div>
  );
}

/**
 * StreamingAvailability renders the "可在哪裡觀看" section on a media detail page
 * (Story 12-4). It is fail-soft (Rule 27 Pillar 3): a load error or empty result
 * NEVER throws or breaks the rest of the page — it shows a quiet retry affordance
 * or a muted empty-state. Provider logos come from TMDB (JustWatch-sourced), which
 * is why the mandatory "資料來源：JustWatch" attribution is rendered beneath them.
 */
export function StreamingAvailability({
  region,
  isLoading,
  isError,
  onRetry,
}: StreamingAvailabilityProps) {
  if (isLoading) {
    return (
      <section
        aria-labelledby={HEADING_ID}
        className="flex flex-col gap-3"
        data-testid="streaming-availability"
      >
        <Heading />
        <div className="flex flex-wrap gap-2" data-testid="streaming-availability-skeleton">
          {[0, 1, 2, 3].map((i) => (
            <div key={i} className="h-10 w-10 animate-pulse rounded-md bg-[var(--bg-secondary)]" />
          ))}
        </div>
      </section>
    );
  }

  if (isError) {
    return (
      <section
        aria-labelledby={HEADING_ID}
        className="flex flex-col gap-3"
        data-testid="streaming-availability"
      >
        <Heading />
        <div
          role="alert"
          className="flex flex-col items-center gap-3 rounded-lg border border-[var(--border-subtle)] px-4 py-6 text-center"
          data-testid="streaming-availability-error"
        >
          <p className="text-sm text-[var(--text-secondary)]">無法載入串流資訊,請稍後再試。</p>
          {onRetry && (
            <button
              type="button"
              onClick={onRetry}
              className="rounded-md border border-[var(--border-subtle)] px-3 py-1.5 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-secondary)]"
            >
              重試
            </button>
          )}
        </div>
      </section>
    );
  }

  const groups = GROUPS.map((g) => ({ ...g, providers: region?.[g.key] ?? [] })).filter(
    (g) => g.providers.length > 0
  );

  // No provider data for this region — quiet muted empty-state (AC #4). Never an error.
  if (groups.length === 0) {
    return (
      <section
        aria-labelledby={HEADING_ID}
        className="flex flex-col gap-3"
        data-testid="streaming-availability"
      >
        <Heading />
        <p
          className="text-sm text-[var(--text-secondary)]"
          data-testid="streaming-availability-empty"
        >
          此區域暫無串流資訊
        </p>
      </section>
    );
  }

  return (
    <section
      aria-labelledby={HEADING_ID}
      className="flex flex-col gap-3"
      data-testid="streaming-availability"
    >
      <Heading />
      <div className="flex flex-col gap-4">
        {groups.map((g) => (
          <ProviderGroup key={g.key} label={g.label} providers={g.providers} />
        ))}
      </div>

      {region?.link && (
        <a
          href={region.link}
          target="_blank"
          rel="noopener noreferrer"
          className="text-sm font-medium text-[var(--accent-primary)] hover:underline"
          data-testid="streaming-availability-link"
        >
          前往 TMDB 觀看頁
        </a>
      )}

      {/* JustWatch attribution — mandatory licensing requirement (AC #6). */}
      <p
        className="text-xs text-[var(--text-muted)]"
        data-testid="streaming-availability-attribution"
      >
        資料來源：JustWatch
      </p>
    </section>
  );
}
