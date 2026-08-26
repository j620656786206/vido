// Implements: <utility — no .pen counterpart>
/**
 * ColorPlaceholder — generates a deterministic gradient poster placeholder
 * from a filename, displaying the first character as a large centered letter.
 * Used when no TMDB poster is available.
 */

interface ColorPlaceholderProps {
  /** Filename or title used to generate the gradient color */
  filename: string;
  /** Override the displayed initial character (defaults to first char of filename) */
  initial?: string;
  /** Pixel height for the placeholder (width follows 2:3 aspect ratio) */
  height?: number;
  className?: string;
}

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

export function ColorPlaceholder({
  filename,
  initial,
  height,
  className = '',
}: ColorPlaceholderProps) {
  const [colorA, colorB] = filenameToGradient(filename);
  const displayChar = initial || filename.charAt(0) || '?';

  const style: React.CSSProperties = {
    background: `linear-gradient(135deg, ${colorA}, ${colorB})`,
  };
  if (height != null) {
    style.height = height;
    style.aspectRatio = '2 / 3';
  }

  return (
    <div
      data-testid="color-placeholder"
      className={`relative flex items-center justify-center overflow-hidden rounded-lg ${className}`}
      style={style}
    >
      <span
        // --text-on-scrim: the tile is hash-derived and clamped dark in BOTH
        // themes, so its letter must not invert with the theme. --text-primary
        // measured 1.45:1 here in 日巡. See PosterCardV2 for the full note.
        className="select-none text-5xl font-bold text-[var(--text-on-scrim)] drop-shadow-lg"
        aria-hidden="true"
      >
        {displayChar}
      </span>
    </div>
  );
}

export default ColorPlaceholder;
