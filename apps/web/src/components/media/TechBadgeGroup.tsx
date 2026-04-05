import { TechBadge } from './TechBadge';
import { resolutionLabel, audioChannelLabel } from '@/utils/resolutionLabel';

export interface TechBadgeGroupProps {
  videoCodec?: string;
  videoResolution?: string;
  audioCodec?: string;
  audioChannels?: number;
  hdrFormat?: string;
  subtitleTracks?: string;
  className?: string;
}

export function TechBadgeGroup({
  videoCodec,
  videoResolution,
  audioCodec,
  audioChannels,
  hdrFormat,
  subtitleTracks,
  className,
}: TechBadgeGroupProps) {
  const badges: { label: string; category: 'video' | 'audio' | 'hdr' | 'subtitle' }[] = [];

  if (videoCodec) {
    badges.push({ label: videoCodec, category: 'video' });
  }

  if (videoResolution) {
    badges.push({ label: resolutionLabel(videoResolution), category: 'video' });
  }

  if (audioCodec) {
    const audioLabel =
      audioChannels != null ? `${audioCodec} ${audioChannelLabel(audioChannels)}` : audioCodec;
    badges.push({ label: audioLabel, category: 'audio' });
  }

  if (hdrFormat) {
    badges.push({ label: hdrFormat, category: 'hdr' });
  }

  if (subtitleTracks) {
    // subtitleTracks is a JSON string of subtitle track info
    try {
      const tracks = JSON.parse(subtitleTracks);
      if (Array.isArray(tracks) && tracks.length > 0) {
        const external = tracks.filter(
          (t) => typeof t === 'object' && t !== null && t.source === 'external'
        ).length;
        const embedded = tracks.filter(
          (t) => typeof t === 'object' && t !== null && t.source === 'embedded'
        ).length;
        const hasSourceInfo = external > 0 || embedded > 0;

        if (hasSourceInfo && external > 0 && embedded > 0) {
          badges.push({ label: `${embedded} 內嵌`, category: 'subtitle' });
          badges.push({ label: `${external} 外掛`, category: 'subtitle' });
        } else if (hasSourceInfo && external > 0) {
          badges.push({ label: `${external} 外掛字幕`, category: 'subtitle' });
        } else if (hasSourceInfo && embedded > 0) {
          badges.push({ label: `${embedded} 內嵌字幕`, category: 'subtitle' });
        } else {
          badges.push({ label: `${tracks.length} 字幕`, category: 'subtitle' });
        }
      }
    } catch {
      // If not JSON, treat as a simple label
      badges.push({ label: subtitleTracks, category: 'subtitle' });
    }
  }

  if (badges.length === 0) return null;

  return (
    <div className={className} data-testid="tech-badge-group">
      <div className="flex flex-wrap gap-1.5">
        {badges.map((badge, i) => (
          <TechBadge key={`${badge.category}-${badge.label}-${i}`} {...badge} />
        ))}
      </div>
    </div>
  );
}
