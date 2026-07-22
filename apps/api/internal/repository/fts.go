package repository

import "strings"

// ftsPrefixQuery turns raw user input into a safe FTS5 prefix query.
//
// Two problems with passing raw input to MATCH:
//  1. CJK partial queries never hit: the unicode61 tokenizer keeps a CJK run as
//     ONE token ("駭客任務"), so MATCH '駭客' finds nothing — a zh-TW library
//     was only searchable by exact full titles (testsprite-round1 TC092 chain).
//  2. FTS5 query syntax is live in raw input: quotes, '-', 'AND', '(' etc.
//     raise fts5 syntax errors → 500s on innocent queries.
//
// Each whitespace-separated term is double-quote-escaped, quoted (disabling all
// operator syntax), and given a '*' prefix marker: 駭客 → "駭客"* which matches
// the 駭客任務 token. Multiple terms join with implicit AND.
func ftsPrefixQuery(raw string) string {
	terms := strings.Fields(raw)
	parts := make([]string, 0, len(terms))
	for _, t := range terms {
		t = strings.ReplaceAll(t, `"`, `""`)
		parts = append(parts, `"`+t+`"*`)
	}
	return strings.Join(parts, " ")
}
