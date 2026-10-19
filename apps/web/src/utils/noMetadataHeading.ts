/**
 * The sentence for a title that automatic matching could not find (B10p). The detail
 * page (`DetailNoMetadataV2`) and the library list's failed row both read it from
 * here, so the list never says something the page it links to does not
 * (disc-2026-10-library-list-triage-b1 AC #7). The noun follows the type.
 */
export function noMetadataHeading(mediaType: 'movie' | 'series' | 'tv'): string {
  return `沒有找到這部${mediaType === 'movie' ? '電影' : '影集'}的資料`;
}
