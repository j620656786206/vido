// Abbreviates a vote/rating count for compact display (Story 12-1, Dev Note #4).
//   < 1,000        → as-is            ("856")
//   < 1,000,000    → thousands        ("1.2k", "15k")
//   >= 1,000,000   → millions         ("1.3M", "2.1M")
// Values < 10 in a tier keep one decimal (1.2k); larger values round to an
// integer (15k). A trailing ".0" is trimmed (2000 → "2k").
export function formatVoteCount(count: number): string {
  if (!Number.isFinite(count) || count <= 0) {
    return '0';
  }
  if (count < 1000) {
    return String(Math.floor(count));
  }
  if (count < 1_000_000) {
    // Guard the upper edge: 999_500 rounds to "1000k" — promote to "1M" instead.
    if (Math.round(count / 1000) >= 1000) {
      return `${abbreviate(count / 1_000_000)}M`;
    }
    return `${abbreviate(count / 1000)}k`;
  }
  return `${abbreviate(count / 1_000_000)}M`;
}

function abbreviate(value: number): string {
  const fixed = value < 10 ? value.toFixed(1) : String(Math.round(value));
  return fixed.replace(/\.0$/, '');
}
