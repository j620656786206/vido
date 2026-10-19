/**
 * One resolution word for a library item (disc-2026-10-library-list-triage-b1 AC #4).
 *
 * The backend stores `video_resolution` in two shapes:
 * - ffprobe writes the raw frame size `寬x高` (`ffprobe_service.go`, e.g. `3840x2076`),
 * - the NFO reader writes a label it already worked out (`4K`, `1440p`, `1080p`…).
 *
 * This turns the first into the second with the same thresholds as the Go side
 * (`resolveResolution`, apps/api/internal/services/nfo_reader_service.go:312). Height decides; width rescues
 * the letterboxed scope encodes whose height is short of the line (`3840x1608` is a
 * 4K file, not a 1440p one).
 *
 * ⚠️ One step past Go: Go has the width rescue for 4K and 1440p only, so it calls a
 * 1080p scope encode (`1920x800`) 720p and a 720p one (`1280x536`) 480p. The story's
 * own example says `1920x800` → 1080p — and the NAS had ten such films on 2026-10-19
 * (`1920x804` ×6, `1920x802`, `1920x800`, `1920x780`, `1920x1036`) — so the same rescue is applied
 * to the two lower tiers here. Until Go does the same
 * (`disc-2026-10-resolution-tier-scope-encodes`), a scope file whose NFO the Go side
 * labelled still reads 720p — today every film on the NAS is ffprobe-shaped.
 *
 * Anything that is not a frame size goes through untouched (`4K`, `1080p`, `SD`),
 * except `2160p`, which is the same thing as `4K` and is shown as 4K.
 */
export function resolutionTier(raw: string | null | undefined): string {
  const value = (raw ?? '').trim();
  if (!value) return '';

  const size = value.match(/^(\d+)\s*[x×X]\s*(\d+)$/);
  if (size) return tierFromSize(Number(size[1]), Number(size[2]));

  const label = value.match(/^(\d+)p$/i);
  if (label && Number(label[1]) >= 2160) return '4K';

  return value;
}

function tierFromSize(width: number, height: number): string {
  if (width <= 0 && height <= 0) return '';
  if (height >= 2160 || (width >= 3840 && height >= 1080)) return '4K';
  if (height >= 1440 || (width >= 2560 && height >= 1080)) return '1440p';
  if (height >= 1080 || (width >= 1920 && height >= 720)) return '1080p';
  if (height >= 720 || (width >= 1280 && height >= 480)) return '720p';
  if (height >= 480) return '480p';
  return `${height}p`;
}

const CHANNEL_MAP: Record<number, string> = {
  1: 'Mono',
  2: 'Stereo',
  6: '5.1',
  8: '7.1',
};

/**
 * Not used yet — kept on purpose: `disc-2026-10-detail-resolution-raw` (blocked by this
 * story) turns the detail page's `8ch` into `7.1` with it and changes the map there.
 */
export function audioChannelLabel(channels: number): string {
  return CHANNEL_MAP[channels] ?? String(channels);
}
