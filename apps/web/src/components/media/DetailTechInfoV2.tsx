// Design ref: ux-design.pen Screen B3p-D (uRGu2)
/**
 * v2 "檔案資訊" tech-truth block (UX Redesign Phase 2 — UX2-3, AC #4). The hybrid
 * differentiator — codec/resolution/audio/HDR at a glance on a consumption-grade
 * page, then size / subtitle tracks / path as fact rows. Fail-soft: renders
 * nothing when there's no tech data.
 *
 * dsr-2 AC #4 (P0, ⚖️ 2026-09-10): tech values are attributes of the FILE, not
 * something that happened — so the badges are neutral (bg-tertiary /
 * text-secondary), never a status tint, and pill-shaped (DESIGN.md §Shapes names
 * TechBadge a removable data mark). The subtitle STATUS already lives in the hero
 * badge; here the tracks are stated as a fact row instead of a second, differently
 * coloured status pill.
 */
import { HANS, HANT, trackLangs } from '../../utils/libraryStatus';

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

/**
 * One track tag → display label. ffprobe hands us raw ISO 639-2 tags (`chi`, `eng`,
 * `und`) and sidecar suffixes as-is. Script classification follows the SHARED
 * HANT/HANS sets (libraryStatus.ts) so this row agrees with every other subtitle
 * surface — which means a bare `zh` reads 繁中 there and here. `chi`/`zho` are not in
 * either set, so they stay raw rather than inventing a script the file never stated.
 */
function trackLabel(lang: string): string {
  if (HANT.has(lang)) return '繁中';
  if (HANS.has(lang)) return '簡中';
  if (lang === 'en' || lang === 'eng' || lang.startsWith('en-')) return '英文';
  if (lang === '' || lang === 'und') return '未標示';
  return lang;
}

function subtitleTrackSummary(subtitleTracks?: string): string | null {
  const langs = trackLangs({ subtitleTracks });
  if (!langs || langs.length === 0) return null;
  return [...new Set(langs.map(trackLabel))].join(' · ');
}

function Badge({ children }: { children: React.ReactNode }) {
  return (
    <span
      data-testid="detail-tech-badge"
      className="rounded-full bg-[var(--bg-tertiary)] px-2 py-0.5 font-mono text-xs text-[var(--text-secondary)]"
    >
      {children}
    </span>
  );
}

function Fact({
  label,
  children,
  testId,
}: {
  label: string;
  children: React.ReactNode;
  testId?: string;
}) {
  return (
    <div className="flex gap-3" data-testid={testId}>
      <dt className="w-16 shrink-0 text-[var(--text-secondary)]">{label}</dt>
      {children}
    </div>
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
  const tracks = subtitleTrackSummary(subtitleTracks);
  const size = formatSize(fileSize);
  const badges = [
    videoResolution,
    videoCodec,
    audioCodec ? `${audioCodec}${audioChannels ? ` ${audioChannels}ch` : ''}` : null,
    hdrFormat,
  ].filter(Boolean) as string[];

  if (badges.length === 0 && !tracks && !size && !filePath) return null;

  return (
    <section data-testid="detail-tech-info">
      <h2 className="mb-3 text-lg font-semibold text-[var(--text-primary)]">檔案資訊</h2>
      {badges.length > 0 && (
        <div className="flex flex-wrap items-center gap-2">
          {badges.map((b) => (
            <Badge key={b}>{b}</Badge>
          ))}
        </div>
      )}
      {(size || tracks || filePath) && (
        <dl className="mt-3 space-y-1.5 text-sm">
          {size && (
            <Fact label="檔案大小">
              <dd className="font-mono text-[var(--text-primary)]">{size}</dd>
            </Fact>
          )}
          {tracks && (
            <Fact label="字幕軌" testId="detail-subtitle-tracks">
              <dd className="text-[var(--text-primary)]">{tracks}</dd>
            </Fact>
          )}
          {filePath && (
            <Fact label="路徑">
              <dd className="truncate font-mono text-[var(--text-primary)]" title={filePath}>
                {filePath}
              </dd>
            </Fact>
          )}
        </dl>
      )}
    </section>
  );
}
