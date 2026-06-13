// Implements: <screen-section — pending epic-19-8 mapping>
/**
 * v2 "檔案資訊" tech-truth block (UX Redesign Phase 2 — UX2-3, AC #4). The hybrid
 * differentiator — codec/resolution/audio/subtitle/size/path at a glance on a
 * consumption-grade page. Tech values render as accent-tint Mono badges; size +
 * path as fact rows. Fail-soft: renders nothing when there's no tech data.
 */
import { deriveSubtitleStatus } from '../../utils/libraryStatus';

interface DetailTechInfoV2Props {
  videoResolution?: string;
  videoCodec?: string;
  audioCodec?: string;
  audioChannels?: number;
  hdrFormat?: string;
  subtitleTracks?: string;
  fileSize?: number;
  filePath?: string;
}

function formatSize(bytes?: number): string | null {
  if (!bytes || bytes <= 0) return null;
  const gb = bytes / 1024 ** 3;
  if (gb >= 1) return `${gb.toFixed(1)} GB`;
  return `${Math.round(bytes / 1024 ** 2)} MB`;
}

function Badge({ children }: { children: React.ReactNode }) {
  return (
    <span className="rounded-[var(--radius-sm)] bg-[var(--accent-tint)] px-2 py-0.5 font-mono text-xs text-[var(--accent-text)]">
      {children}
    </span>
  );
}

export function DetailTechInfoV2({
  videoResolution,
  videoCodec,
  audioCodec,
  audioChannels,
  hdrFormat,
  subtitleTracks,
  fileSize,
  filePath,
}: DetailTechInfoV2Props) {
  const subtitle = deriveSubtitleStatus({ parseStatus: 'success', subtitleTracks });
  const size = formatSize(fileSize);
  const badges = [
    videoResolution,
    videoCodec,
    audioCodec ? `${audioCodec}${audioChannels ? ` ${audioChannels}ch` : ''}` : null,
    hdrFormat,
  ].filter(Boolean) as string[];

  if (badges.length === 0 && !subtitle && !size && !filePath) return null;

  return (
    <section data-testid="detail-tech-info">
      <h2 className="mb-3 text-lg font-semibold text-[var(--text-primary)]">檔案資訊</h2>
      {(badges.length > 0 || subtitle) && (
        <div className="flex flex-wrap items-center gap-2">
          {badges.map((b) => (
            <Badge key={b}>{b}</Badge>
          ))}
          {subtitle && (
            <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${subtitle.className}`}>
              {subtitle.label}
            </span>
          )}
        </div>
      )}
      {(size || filePath) && (
        <dl className="mt-3 space-y-1.5 text-sm">
          {size && (
            <div className="flex gap-3">
              <dt className="w-16 shrink-0 text-[var(--text-muted)]">大小</dt>
              <dd className="font-mono text-[var(--text-secondary)]">{size}</dd>
            </div>
          )}
          {filePath && (
            <div className="flex gap-3">
              <dt className="w-16 shrink-0 text-[var(--text-muted)]">路徑</dt>
              <dd className="truncate font-mono text-[var(--text-secondary)]" title={filePath}>
                {filePath}
              </dd>
            </div>
          )}
        </dl>
      )}
    </section>
  );
}
