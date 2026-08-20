const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p';

export type ImageSize = 'w92' | 'w154' | 'w185' | 'w342' | 'w500' | 'w780' | 'original';

/**
 * True when the stored value is already a complete URL rather than a TMDb path.
 *
 * bugfix-d D2: the canonical storage form is the TMDb-relative path ("/abc.jpg"),
 * but two kinds of absolute value legitimately reach us — rows written before the
 * D2 fix (a sampled 222 rows held 103 absolute URLs), and artwork from providers
 * whose images live on their own host (Douban, Wikipedia), which have no relative
 * path to give. Prefixing either produced ".../w342https://…" and a broken image,
 * so absolute values pass through untouched.
 */
function isAbsoluteUrl(path: string): boolean {
  return /^(https?:)?\/\//i.test(path);
}

export function getImageUrl(path: string | null, size: ImageSize = 'w342'): string | null {
  if (!path) return null;
  if (isAbsoluteUrl(path)) return path;
  return `${TMDB_IMAGE_BASE}/${size}${path}`;
}

export function getImageSrcSet(path: string | null): string | null {
  if (!path) return null;
  // An absolute URL is one fixed rendition — there is no size ladder to offer,
  // so let the caller's plain `src` serve it instead of emitting a bogus srcset.
  if (isAbsoluteUrl(path)) return null;
  return [
    `${TMDB_IMAGE_BASE}/w185${path} 185w`,
    `${TMDB_IMAGE_BASE}/w342${path} 342w`,
    `${TMDB_IMAGE_BASE}/w500${path} 500w`,
  ].join(', ');
}

export function getImageSizes(): string {
  return '(max-width: 640px) 45vw, (max-width: 1024px) 25vw, 200px';
}

// Backdrop-specific responsive set — covers mobile (w780), tablet (w1280) and
// desktop (original). Used by hero banners that span the full viewport width
// and would otherwise pay desktop-sized payload (3–5MB) on mobile.
export function getBackdropSrcSet(path: string | null): string | null {
  if (!path) return null;
  // bugfix-d D2 — same passthrough rule as getImageSrcSet.
  if (isAbsoluteUrl(path)) return null;
  return [
    `${TMDB_IMAGE_BASE}/w780${path} 780w`,
    `${TMDB_IMAGE_BASE}/w1280${path} 1280w`,
    `${TMDB_IMAGE_BASE}/original${path} 1920w`,
  ].join(', ');
}

export function getBackdropSizes(): string {
  // Banner is full-width at every breakpoint.
  return '(max-width: 640px) 100vw, (max-width: 1024px) 100vw, 100vw';
}
