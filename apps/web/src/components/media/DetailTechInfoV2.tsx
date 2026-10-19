// Design ref: ux-design.pen Screen B3p-D (uRGu2) · B3p-M (SzNRb)
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
 *
 * disc-2026-10-detail-resolution-raw: the resolution badge reads `4K · 3840×2160`
 * (B3p-D) and only `4K` on a phone (B3p-M) — the frame size sits in its own span that
 * is hidden below `sm`, and the badge's `title` carries the whole string. The audio
 * badge writes the speaker layout (`TrueHD 7.1`), not the raw count (`8ch`).
 */
import { subtitleLangLabel, trackLangs } from '../../utils/libraryStatus';
import { audioChannelLabel, formatResolutionBadge } from '../../utils/resolutionLabel';

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
 * `und`) and sidecar suffixes as-is. The mapping is the SHARED `subtitleLangLabel`
 * (libraryStatus.ts, dsr-6b) so this row and the 管理字幕 dialog name a track the
 * same way — a bare `zh` reads 繁中 in both, and `chi`/`zho` stay raw.
 */
function trackLabel(lang: string): string {
  return subtitleLangLabel(lang).label;
}

function subtitleTrackSummary(subtitleTracks?: string): string | null {
  const langs = trackLangs({ subtitleTracks });
  if (!langs || langs.length === 0) return null;
  return [...new Set(langs.map(trackLabel))].join(' · ');
}

function Badge({ children, title }: { children: React.ReactNode; title?: string }) {
  return (
    <span
      data-testid="detail-tech-badge"
      title={title}
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
  const resolution = formatResolutionBadge(videoResolution);
  const audio = audioCodec
    ? `${audioCodec}${audioChannels ? ` ${audioChannelLabel(audioChannels)}` : ''}`
    : null;
  // Stable keys: the resolution badge's content is JSX, so content can no longer key it.
  const badges: { key: string; title?: string; content: React.ReactNode }[] = [];
  if (resolution) {
    badges.push({
      key: 'res',
      // Only when it adds something: a bare label (`1080p`) needs no tooltip repeating it.
      title: resolution.dims ? resolution.full : undefined,
      content: (
        <>
          <span>{resolution.tier}</span>
          {resolution.dims && <span className="hidden sm:inline">{` · ${resolution.dims}`}</span>}
        </>
      ),
    });
  }
  if (videoCodec) badges.push({ key: 'vcodec', content: videoCodec });
  if (audio) badges.push({ key: 'audio', content: audio });
  if (hdrFormat) badges.push({ key: 'hdr', content: hdrFormat });

  if (badges.length === 0 && !tracks && !size && !filePath) return null;

  return (
    <section data-testid="detail-tech-info">
      <h2 className="mb-3 text-lg font-semibold text-[var(--text-primary)]">檔案資訊</h2>
      {badges.length > 0 && (
        <div className="flex flex-wrap items-center gap-2">
          {badges.map((b) => (
            <Badge key={b.key} title={b.title}>
              {b.content}
            </Badge>
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
