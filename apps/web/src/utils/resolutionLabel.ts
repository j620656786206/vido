/** `寬x高` as ffprobe writes it; also reads `×` and spaces around the sign. */
const FRAME_SIZE = /^(\d+)\s*[x×X]\s*(\d+)$/;

/**
 * One resolution word for a library item (disc-2026-10-library-list-triage-b1 AC #4,
 * rule from disc-2026-10-resolution-tier-scope-encodes).
 *
 * The backend stores `video_resolution` in two shapes:
 * - ffprobe writes the raw frame size `寬x高` (`ffprobe_service.go`, e.g. `3840x2076`),
 * - the NFO reader writes a label it already worked out (`4K`, `1440p`, `1080p`…) with
 *   Go `resolveResolution` (apps/api/internal/services/nfo_reader_service.go).
 *
 * This turns the first into the second with the SAME table as Go — both sides are
 * tested against apps/api/internal/services/testdata/resolution_tiers.json, so a
 * frame size reads the same whichever way it came in. See `RESOLUTION_TIERS`.
 *
 * Anything that is not a frame size goes through untouched (`4K`, `1080p`, `SD`),
 * except `2160p`, which is the same thing as `4K` and is shown as 4K.
 */
export function resolutionTier(raw: string | null | undefined): string {
  const value = (raw ?? '').trim();
  if (!value) return '';

  const size = value.match(FRAME_SIZE);
  if (size) return tierFromSize(Number(size[1]), Number(size[2]));

  const label = value.match(/^(\d+)p$/i);
  if (label && Number(label[1]) >= 2160) return '4K';

  return value;
}

/**
 * The frame is turned sideways first (long side = width). A tier is reached when the
 * long side OR the short side is at least 90% of that tier's 16:9 frame, checked from
 * 4K down — so a 4K film with its black bars cropped (`3840x1608`, `3828x1588`) is 4K
 * (⚖️ Alexyu 2026-10-19), a 4:3 one (`2880x2160`) is 4K, and a 21:9 monitor capture
 * (`3440x1440`) stays 1440p. Below 480p the short side is written as `${n}p`.
 *
 * The lines are whole pixels, not `× 0.9`, so this and Go's `resolutionTiers` cannot
 * round them differently. Change both, and the shared table, together.
 */
const RESOLUTION_TIERS: ReadonlyArray<{ label: string; minLong: number; minShort: number }> = [
  { label: '4K', minLong: 3456, minShort: 1944 }, // 3840×2160
  { label: '1440p', minLong: 2304, minShort: 1296 }, // 2560×1440
  { label: '1080p', minLong: 1728, minShort: 972 }, // 1920×1080
  { label: '720p', minLong: 1152, minShort: 648 }, // 1280×720
  { label: '480p', minLong: 769, minShort: 432 }, // 854×480
];

function tierFromSize(width: number, height: number): string {
  if (width <= 0 && height <= 0) return '';
  // Only a frame with both sides known can be turned sideways; `0x1080` keeps its
  // 1080 as the short side, as it read before the rule change.
  const portrait = width > 0 && height > width;
  const long = portrait ? height : width;
  const short = portrait ? width : height;
  const tier = RESOLUTION_TIERS.find((t) => long >= t.minLong || short >= t.minShort);
  return tier ? tier.label : `${short}p`;
}

/**
 * The detail page's resolution badge (disc-2026-10-detail-resolution-raw AC #1–#2):
 * the tier first, then the frame size the file really has — `4K · 3840×2160` — the
 * way B3p-D / B10p-D draw it. The tier comes from `resolutionTier`, so the badge and
 * the library list never disagree about what a file is.
 *
 * Returned in two parts because the phone screens (B3p-M / B10p-M) show the tier
 * alone: the component hides `dims` below the `sm` breakpoint.
 *
 * - `dims` is `寬×高` with U+00D7 and no spaces, only when the value IS a frame size
 *   with both sides above zero; an NFO label (`4K`, `1080p`, `SD`) has none.
 * - `full` is what the badge's `title` carries (`4K · 3840×2160`, or the tier alone).
 * - `null` when there is nothing to show — the badge is left out, as before.
 */
export interface ResolutionBadge {
  tier: string;
  dims: string | null;
  full: string;
}

export function formatResolutionBadge(raw: string | null | undefined): ResolutionBadge | null {
  const tier = resolutionTier(raw);
  if (!tier) return null;

  const size = (raw ?? '').trim().match(FRAME_SIZE);
  const width = size ? Number(size[1]) : 0;
  const height = size ? Number(size[2]) : 0;
  const dims = width > 0 && height > 0 ? `${width}×${height}` : null;

  return { tier, dims, full: dims ? `${tier} · ${dims}` : tier };
}

/**
 * Channel count → the speaker layout the detail page's audio badge writes after the
 * codec (`TrueHD 7.1`, disc-2026-10-detail-resolution-raw AC #3). Numbers, not
 * Mono／Stereo: the design draws `7.1`, and the badges in that row are monospace
 * read-outs. A count with no common layout stays `${n}ch`, as the page wrote before.
 */
const CHANNEL_MAP: Record<number, string> = {
  1: '1.0',
  2: '2.0',
  6: '5.1',
  8: '7.1',
};

export function audioChannelLabel(channels: number): string {
  return CHANNEL_MAP[channels] ?? `${channels}ch`;
}
