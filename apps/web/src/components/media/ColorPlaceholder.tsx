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
  return [`hsl(${hue}, 65%, 35%)`, `hsl(${(hue + 40) % 360}, 55%, 45%)`];
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
        className="select-none text-5xl font-bold text-white/90 drop-shadow-lg"
        aria-hidden="true"
      >
        {displayChar}
      </span>
    </div>
  );
}

export default ColorPlaceholder;
