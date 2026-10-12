// A deterministic gradient for a poster tile that has no image: the same title
// always gets the same colours. Moved here from the deleted v1
// `components/media/ColorPlaceholder.tsx` (disc-2026-09-unmounted-v1-components).

/**
 * Deterministic hash function that maps a filename to a pair of HSL gradient stops.
 * Uses djb2-style hashing for even distribution across the hue wheel.
 */
export function filenameToGradient(filename: string): [string, string] {
  let hash = 0;
  for (let i = 0; i < filename.length; i++) {
    hash = filename.charCodeAt(i) + ((hash << 5) - hash);
  }
  const hue = Math.abs(hash) % 360;
  // Lightness is clamped DARK (26%/32%, was 35%/45%) so the light initial
  // letter (--text-primary) clears 3:1 on every hue — with hash-derived hues
  // the old 45% stop let a yellow tile drop dark ink to 2.4:1 (critique R1).
  // Darker stops also sit the tiles inside the 夜行 grounds instead of
  // signal-era brights.
  return [`hsl(${hue}, 45%, 26%)`, `hsl(${(hue + 40) % 360}, 40%, 32%)`];
}
