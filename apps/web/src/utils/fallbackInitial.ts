/**
 * First LETTER or CJK char for a no-poster tile — 「[FanSub] 未知電影」 used to
 * render a giant 「[」 (critique R3 minor). Falls back to the raw first char when
 * the title is all symbols. Shared by PosterCardV2, DetailHeroV2 and the manual
 * match dialog (dsr-2b-b AC #6): an unmatched item's title IS its file name, so
 * the detail hero hit the same bracket.
 */
export function fallbackInitial(title: string): string {
  const m = title.match(/[\p{L}\p{N}]/u);
  return m ? m[0] : title.slice(0, 1) || '?';
}
