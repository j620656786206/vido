import { useMemo } from 'react';

interface HighlightTextProps {
  text: string;
  query?: string;
  className?: string;
}

/**
 * Highlights matching substrings in text by wrapping them in <mark> tags.
 * Case-insensitive matching. Returns plain text when no query provided.
 */
export function HighlightText({ text, query, className }: HighlightTextProps) {
  const parts = useMemo(() => {
    if (!query || query.length < 2) return null;

    // Escape special regex characters in query
    const escaped = query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const regex = new RegExp(`(${escaped})`, 'gi');
    return text.split(regex);
  }, [text, query]);

  if (!parts) {
    return <span className={className}>{text}</span>;
  }

  return (
    <span className={className}>
      {parts.map((part, i) =>
        part.toLowerCase() === query!.toLowerCase() ? (
          <mark key={i} className="bg-[var(--warning)]/30 text-inherit rounded-sm px-0.5">
            {part}
          </mark>
        ) : (
          part
        )
      )}
    </span>
  );
}
