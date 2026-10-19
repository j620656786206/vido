/**
 * Small pure helpers for the library list's rows (disc-2026-10-library-list-triage-b1).
 */
import { resolutionTier } from './resolutionLabel';

type SpecFields = {
  videoResolution?: string;
  videoCodec?: string;
  audioCodec?: string;
};

/**
 * The 規格 column as one quiet line — `4K · H.265 · DTS` (AC #4). The resolution is
 * shown as a tier, never the raw frame size. `null` when the file told us none of
 * the three; the row then shows a muted 「—」.
 */
export function formatSpecLine(media: SpecFields): string | null {
  const parts = [resolutionTier(media.videoResolution), media.videoCodec, media.audioCodec]
    .map((p) => (p ?? '').trim())
    .filter(Boolean);
  return parts.length ? parts.join(' · ') : null;
}

/**
 * The file name a failed row shows (AC #7): the last segment of `filePath` (a series'
 * path can be its folder — the last segment is still the useful part), or the title
 * when the list did not send a path.
 */
export function fileNameOf(filePath: string | undefined, fallback: string): string {
  const segments = (filePath ?? '').split(/[\\/]/).filter(Boolean);
  return segments.length ? segments[segments.length - 1] : fallback;
}

/** How many characters a middle-truncated file name always keeps at its end. */
export const FILE_NAME_TAIL = 12;

/**
 * Split a file name for middle truncation: the browser clips `head` with an
 * ellipsis and `tail` (the last 12 characters — the episode / release tag and the
 * extension) always stays. Splits on code points so a CJK or emoji name is never
 * cut through a surrogate pair.
 */
export function splitFileNameTail(name: string): { head: string; tail: string } {
  const chars = Array.from(name);
  if (chars.length <= FILE_NAME_TAIL) return { head: '', tail: name };
  return {
    head: chars.slice(0, -FILE_NAME_TAIL).join(''),
    tail: chars.slice(-FILE_NAME_TAIL).join(''),
  };
}
