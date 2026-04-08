package subtitle

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SubtitleBlock represents a single SRT subtitle entry.
type SubtitleBlock struct {
	Index int
	Start string
	End   string
	Text  string
}

// timestampPattern matches SRT timestamp lines: 00:00:01,000 --> 00:00:04,000
var timestampPattern = regexp.MustCompile(`^(\d{2}:\d{2}:\d{2},\d{3})\s*-->\s*(\d{2}:\d{2}:\d{2},\d{3})`)

// ParseSRT parses SRT content into a slice of SubtitleBlocks.
// Handles UTF-8 BOM, Windows line endings, and extra blank lines.
func ParseSRT(content string) ([]SubtitleBlock, error) {
	if content == "" {
		return nil, nil
	}

	// Strip UTF-8 BOM
	content = strings.TrimPrefix(content, "\xEF\xBB\xBF")

	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	lines := strings.Split(content, "\n")
	var blocks []SubtitleBlock

	i := 0
	for i < len(lines) {
		// Skip empty lines between blocks
		line := strings.TrimSpace(lines[i])
		if line == "" {
			i++
			continue
		}

		// Parse block index
		index, err := strconv.Atoi(line)
		if err != nil {
			i++
			continue
		}
		i++

		// Parse timestamp line
		if i >= len(lines) {
			break
		}
		tsLine := strings.TrimSpace(lines[i])
		matches := timestampPattern.FindStringSubmatch(tsLine)
		if matches == nil {
			continue
		}
		start := matches[1]
		end := matches[2]
		i++

		// Collect text lines until empty line or EOF
		var textLines []string
		for i < len(lines) {
			tl := lines[i]
			trimmed := strings.TrimSpace(tl)
			if trimmed == "" {
				i++
				break
			}
			textLines = append(textLines, trimmed)
			i++
		}

		blocks = append(blocks, SubtitleBlock{
			Index: index,
			Start: start,
			End:   end,
			Text:  strings.Join(textLines, "\n"),
		})
	}

	return blocks, nil
}

// SerializeSRT converts a slice of SubtitleBlocks back to SRT format.
// Timestamps are preserved exactly as stored in the blocks (AC #3).
func SerializeSRT(blocks []SubtitleBlock) string {
	if len(blocks) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, b := range blocks {
		sb.WriteString(fmt.Sprintf("%d\n", b.Index))
		sb.WriteString(fmt.Sprintf("%s --> %s\n", b.Start, b.End))
		sb.WriteString(b.Text)
		sb.WriteString("\n")
		if i < len(blocks)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
