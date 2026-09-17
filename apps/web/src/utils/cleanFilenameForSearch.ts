/**
 * Turn a media file name into a search query a person would type (dsr-2b-b
 * AC #4): the manual-match dialog is prefilled with it. An unmatched item's
 * title is its raw file name — 「[Leopard-Raws] Kimi no Na wa (BD).mkv」 — and
 * searching TMDb for that verbatim finds nothing.
 *
 * Release names put the title first and the junk after it, so the name is CUT at
 * the first release token (resolution, source, codec, audio, HDR, SxxEyy) rather
 * than having words removed one by one — that is what leaves `-SPARKS`, `H 265`
 * or `DDP5 1` behind. A trailing year goes too: TMDb's search matches title
 * words, and 「The Matrix 1999」 finds less than 「The Matrix」.
 *
 * Ambiguous words that are also title words (`web`, `bd`, `dv`) are NOT cut
 * tokens — 「Charlotte's.Web.2006」 keeps its Web. Bracketed groups are dropped
 * first, which is where fansub tags like (BD) live.
 */
const VIDEO_EXTENSION = /\.(?:mkv|mp4|m4v|avi|ts|m2ts|wmv|mov|iso|rmvb|flv|webm)$/i;

const CUT_TOKEN = new RegExp(
  '(?:^|[\\s._-])(?:' +
    [
      '\\d{3,4}[pi]',
      '4k',
      'uhd',
      's\\d{1,2}(?:e\\d{1,3})?',
      'hdr10\\+?|hdr|dolby[\\s._-]?vision',
      'x26[45]|h\\.?26[45]|hevc|avc',
      'blu-?ray|bdrip|brrip|web-?dl|webrip|hdtv|dvdrip|remux',
      'aac(?:\\d\\.\\d)?|ac3|e-?ac-?3|ddp?\\+?\\d?\\.?\\d?|dts(?:-hd)?|truehd|atmos|flac',
      '10bit|8bit',
    ].join('|') +
    ')(?=$|[\\s._\\-\\])])',
  'i'
);

const YEAR = /^(?:19|20)\d{2}$/;

export function cleanFilenameForSearch(input: string): string {
  const base = (input.split(/[\\/]/).pop() ?? input).replace(VIDEO_EXTENSION, '').trim();

  let name = base.replace(/\[[^\]]*\]|\([^)]*\)|【[^】]*】/g, ' '); // [group] (BD) 【字幕組】
  const cut = name.search(CUT_TOKEN);
  if (cut > 0) name = name.slice(0, cut);

  const words = name
    .replace(/[._]+/g, ' ')
    .replace(/\s-\S*$/, '') // a dangling "-GROUP" left before the cut
    .split(/\s+/)
    .filter(Boolean);
  if (words.length > 1 && YEAR.test(words[words.length - 1])) words.pop();

  return words.join(' ') || base;
}
