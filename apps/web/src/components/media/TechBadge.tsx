import { cn } from '@/lib/utils';

export type TechBadgeCategory = 'video' | 'audio' | 'hdr' | 'subtitle';

const CATEGORY_CLASSES: Record<TechBadgeCategory, string> = {
  video: 'bg-blue-500/20 text-blue-500',
  audio: 'bg-purple-500/20 text-purple-500',
  hdr: 'bg-amber-500/20 text-amber-500',
  subtitle: 'bg-emerald-500/20 text-emerald-500',
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
