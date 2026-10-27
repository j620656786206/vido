package wikipedia

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// CastEntry is one actor → character row of a zh.wikipedia cast table
// (disc-2026-10-glossary-seeds-actor-names-and-fan-surname-b).
type CastEntry struct {
	Actor     string
	Character string
}

// GetWikitext returns the wikitext of the first of titles that exists,
// following redirects and Chinese variant titles (百花杀 finds 百花殺), and the
// title of the page it came from. A title that does not exist is skipped; none
// existing is a NotFoundError.
func (c *Client) GetWikitext(ctx context.Context, titles ...string) (string, string, error) {
	for _, title := range titles {
		if strings.TrimSpace(title) == "" {
			continue
		}
		params := url.Values{
			"action":        {"query"},
			"titles":        {title},
			"prop":          {"revisions"},
			"rvprop":        {"content"},
			"rvslots":       {"main"},
			"redirects":     {"1"},
			"converttitles": {"1"},
			"format":        {"json"},
			"formatversion": {"2"},
			"utf8":          {"1"},
		}
		body, err := c.doRequest(ctx, params)
		if err != nil {
			return "", "", err
		}
		var resp struct {
			Query struct {
				Pages []struct {
					Title     string `json:"title"`
					Missing   bool   `json:"missing"`
					Revisions []struct {
						Slots struct {
							Main struct {
								Content string `json:"content"`
							} `json:"main"`
						} `json:"slots"`
					} `json:"revisions"`
				} `json:"pages"`
			} `json:"query"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return "", "", fmt.Errorf("parse wikitext response: %w", err)
		}
		for _, page := range resp.Query.Pages {
			if page.Missing || len(page.Revisions) == 0 {
				continue
			}
			if text := page.Revisions[0].Slots.Main.Content; text != "" {
				return text, page.Title, nil
			}
		}
	}
	return "", "", &NotFoundError{PageTitle: strings.Join(titles, " / ")}
}

// IsNotFound reports whether err is a NotFoundError.
func IsNotFound(err error) bool {
	var nf *NotFoundError
	return errors.As(err, &nf)
}

var (
	castRefPattern      = regexp.MustCompile(`(?s)<ref[^>]*/>|<ref[^>]*>.*?</ref>`)
	castTemplatePattern = regexp.MustCompile(`\{\{[^{}]*\}\}`)
	castLinkPattern     = regexp.MustCompile(`\[\[(?:[^\[\]|]*\|)?([^\[\]]*)\]\]`)
	castTagPattern      = regexp.MustCompile(`<[^>]+>`)
	castParenPattern    = regexp.MustCompile(`[（(].*$`)
	// -{…}- protects text from variant conversion; -{}- is an empty marker.
	castVariantPattern = regexp.MustCompile(`-\{([^{}]*)\}-`)
	// "* [[孟子義]] 飾 沈汐和" — the list form some pages use instead of a table.
	castListPattern = regexp.MustCompile(`^[*#]+\s*(.+?)\s*(?:飾演|饰演|飾|饰)\s*(.+)$`)
)

// ParseCast reads every actor → character row out of a page's wikitext: each
// wikitable whose header names an actor column (演員／演员) and a character
// column (角色), and list lines of the form 「演員 飾 角色」. A rowspan cell
// (one actor, two roles) fills the rows below it; rows that still do not line
// up are skipped rather than guessed.
func ParseCast(wikitext string) []CastEntry {
	var out []CastEntry
	seen := map[string]bool{}
	add := func(actor, character string) {
		actor, character = cleanCastCell(actor), cleanCastCell(character)
		if !isCastName(actor) || !isCastName(character) || seen[actor+"\x00"+character] {
			return
		}
		seen[actor+"\x00"+character] = true
		out = append(out, CastEntry{Actor: actor, Character: character})
	}

	for _, table := range splitWikitables(wikitext) {
		actorCol, characterCol := -1, -1
		spans := map[int]span{}
		for _, row := range table {
			// A header row is "! 演員 !! 角色", or on some pages a plain
			// "| '''演員''' || '''角色'''" row before any data.
			if row.header || actorCol < 0 {
				if a, c := castHeaderColumns(row.cells); a >= 0 && c >= 0 {
					actorCol, characterCol = a, c
					spans = map[int]span{}
					continue
				}
				if row.header || actorCol < 0 {
					continue
				}
			}
			cells := expandRowspans(row.cells, spans)
			if actorCol < 0 || characterCol < 0 || len(cells) <= actorCol || len(cells) <= characterCol {
				continue
			}
			add(cells[actorCol], cells[characterCol])
		}
	}

	for _, line := range strings.Split(wikitext, "\n") {
		if m := castListPattern.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			add(m[1], strings.FieldsFunc(m[2], func(r rune) bool { return r == '，' || r == ',' || r == '：' || r == ':' })[0])
		}
	}
	return out
}

// castHeaderColumns finds the actor and character columns of a header row.
func castHeaderColumns(cells []string) (actorCol, characterCol int) {
	actorCol, characterCol = -1, -1
	for i, cell := range cells {
		switch name := cleanCastCell(cell); {
		case actorCol < 0 && (name == "演員" || name == "演员"):
			actorCol = i
		case characterCol < 0 && name == "角色":
			characterCol = i
		}
	}
	return actorCol, characterCol
}

type span struct {
	rows  int // rows below this one still covered
	value string
}

var castRowspanPattern = regexp.MustCompile(`rowspan\s*=\s*"?(\d+)`)

// expandRowspans puts the cells a rowspan above covers back into this row,
// and records the rowspans this row starts.
func expandRowspans(raw []string, spans map[int]span) []string {
	var cells []string
	next := 0
	for col := 0; next < len(raw) || spans[col].rows > 0; col++ {
		if sp := spans[col]; sp.rows > 0 {
			cells = append(cells, sp.value)
			sp.rows--
			spans[col] = sp
			continue
		}
		cell := raw[next]
		next++
		if m := castRowspanPattern.FindStringSubmatch(attributesOf(cell)); m != nil {
			if n := atoi(m[1]); n > 1 {
				spans[col] = span{rows: n - 1, value: cell}
			}
		}
		cells = append(cells, cell)
	}
	return cells
}

// attributesOf is the `style=… rowspan=2` part before a cell's first
// top-level "|", or "" when the cell has none.
func attributesOf(cell string) string {
	if rest := dropCellAttributes(cell); rest != cell {
		return cell[:len(cell)-len(rest)]
	}
	return ""
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		n = n*10 + int(r-'0')
	}
	return n
}

type wikiRow struct {
	header bool
	cells  []string
}

// splitWikitables turns each {| … |} block into rows of cells. Continuation
// lines (a long description wrapped onto the next line) are dropped: only the
// leading cells matter here.
func splitWikitables(wikitext string) [][]wikiRow {
	var tables [][]wikiRow
	var rows []wikiRow
	var cur *wikiRow
	in := false
	flush := func() {
		if cur != nil && len(cur.cells) > 0 {
			rows = append(rows, *cur)
		}
		cur = nil
	}
	for _, raw := range strings.Split(wikitext, "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "{|"):
			in, rows, cur = true, nil, nil
		case !in:
		case strings.HasPrefix(line, "|}"):
			flush()
			tables = append(tables, rows)
			in = false
		case strings.HasPrefix(line, "|-"), strings.HasPrefix(line, "|+"):
			flush()
		case strings.HasPrefix(line, "!"):
			if cur == nil {
				cur = &wikiRow{header: true}
			}
			cur.cells = append(cur.cells, strings.Split(line[1:], "!!")...)
		case strings.HasPrefix(line, "|"):
			if cur == nil {
				cur = &wikiRow{}
			}
			cur.cells = append(cur.cells, strings.Split(line[1:], "||")...)
		}
	}
	return tables
}

// cleanCastCell strips a cell down to its text: attributes (style="…"|),
// references, templates, links, tags and a trailing parenthetical.
func cleanCastCell(cell string) string {
	cell = castRefPattern.ReplaceAllString(cell, "")
	for castTemplatePattern.MatchString(cell) {
		cell = castTemplatePattern.ReplaceAllString(cell, "")
	}
	cell = dropCellAttributes(cell)
	cell = castVariantPattern.ReplaceAllStringFunc(cell, func(m string) string {
		inner := castVariantPattern.FindStringSubmatch(m)[1]
		if i := strings.LastIndex(inner, ":"); i >= 0 { // zh-hans:x;zh-hant:y → y
			inner = strings.TrimSuffix(inner[i+1:], ";")
		}
		return inner
	})
	cell = castLinkPattern.ReplaceAllString(cell, "$1")
	cell = castTagPattern.ReplaceAllString(cell, "")
	cell = strings.NewReplacer("'''", "", "''", "").Replace(cell)
	cell = castParenPattern.ReplaceAllString(cell, "")
	return strings.TrimSpace(cell)
}

// dropCellAttributes removes a leading `style="…" |` / `rowspan=2|`: the
// first "|" outside [[ ]] ends the attributes when what precedes it has "=".
func dropCellAttributes(cell string) string {
	depth := 0
	for i := 0; i < len(cell); i++ {
		switch {
		case strings.HasPrefix(cell[i:], "[["):
			depth++
			i++
		case strings.HasPrefix(cell[i:], "]]"):
			depth--
			i++
		case cell[i] == '|' && depth == 0:
			if strings.Contains(cell[:i], "=") {
				return cell[i+1:]
			}
			return cell
		}
	}
	return cell
}

// isCastName keeps what looks like a person's or character's name: some CJK,
// short, no sentence punctuation.
func isCastName(s string) bool {
	n := utf8.RuneCountInString(s)
	if n == 0 || n > 20 || strings.ContainsAny(s, "。，；：、") {
		return false
	}
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
