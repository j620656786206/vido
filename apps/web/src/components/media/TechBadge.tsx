// Implements: Component/TechBadge-Video (L9m19) + Component/TechBadge-Audio (9iTW3) + Component/TechBadge-Subtitle (f84BM) + Component/TechBadge-HDR (cUjyv)
// Source: ux-design.pen (Pencil app)
import { cn } from '@/lib/utils';

export type TechBadgeCategory = 'video' | 'audio' | 'hdr' | 'subtitle';

/**
 * Tech badges are INFORMATION badges, not status: video/audio/hdr/subtitle are
 * categories, so they may keep distinct hues — but as tokens, not raw palette.
 * The *-500 text shades measured 3.1–4.4:1 on their own tints; the readable
 * variants clear AA. Converted whole (all four variants at once) rather than
 * one hue per slice, so the component never ships half-migrated.
 */
const CATEGORY_CLASSES: Record<TechBadgeCategory, string> = {
  video: 'bg-[var(--accent-tint)] text-[var(--accent-text)]',
  audio: 'bg-[var(--info-tint)] text-[var(--info-text)]',
  hdr: 'bg-[var(--warning-tint)] text-[var(--warning-text)]',
  subtitle: 'bg-[var(--success-tint)] text-[var(--success-text)]',
};

export interface TechBadgeProps {
  label: string;
  category: TechBadgeCategory;
  className?: string;
}

const CATEGORY_LABELS: Record<TechBadgeCategory, string> = {
  video: 'Video',
  audio: 'Audio',
  hdr: 'HDR',
  subtitle: 'Subtitle',
};

export function TechBadge({ label, category, className }: TechBadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
        CATEGORY_CLASSES[category],
        className
      )}
      data-testid="tech-badge"
      aria-label={`${CATEGORY_LABELS[category]}: ${label}`}
    >
      {label}
    </span>
  );
}
