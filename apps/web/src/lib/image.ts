const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p';

export type ImageSize = 'w92' | 'w154' | 'w185' | 'w342' | 'w500' | 'w780' | 'original';

export function getImageUrl(path: string | null, size: ImageSize = 'w342'): string | null {
  if (!path) return null;
  return `${TMDB_IMAGE_BASE}/${size}${path}`;
}

export function getImageSrcSet(path: string | null): string | null {
  if (!path) return null;
  return [
    `${TMDB_IMAGE_BASE}/w185${path} 185w`,
    `${TMDB_IMAGE_BASE}/w342${path} 342w`,
    `${TMDB_IMAGE_BASE}/w500${path} 500w`,
  ].join(', ');
}

export function getImageSizes(): string {
  return '(max-width: 640px) 45vw, (max-width: 1024px) 25vw, 200px';
}
