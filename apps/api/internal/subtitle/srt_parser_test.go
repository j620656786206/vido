package subtitle

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSRT_Basic(t *testing.T) {
	input := `1
00:00:01,000 --> 00:00:04,000
Hello, how are you?

2
00:00:05,000 --> 00:00:08,000
I'm doing fine, thanks.
`
	blocks, err := ParseSRT(input)
	require.NoError(t, err)
	require.Len(t, blocks, 2)

	assert.Equal(t, 1, blocks[0].Index)
	assert.Equal(t, "00:00:01,000", blocks[0].Start)
	assert.Equal(t, "00:00:04,000", blocks[0].End)
	assert.Equal(t, "Hello, how are you?", blocks[0].Text)

	assert.Equal(t, 2, blocks[1].Index)
	assert.Equal(t, "00:00:05,000", blocks[1].Start)
	assert.Equal(t, "00:00:08,000", blocks[1].End)
	assert.Equal(t, "I'm doing fine, thanks.", blocks[1].Text)
}

func TestParseSRT_MultiLineBlock(t *testing.T) {
	input := `1
00:00:01,000 --> 00:00:04,000
Line one
Line two

2
00:00:05,000 --> 00:00:08,000
Single line
`
	blocks, err := ParseSRT(input)
	require.NoError(t, err)
	require.Len(t, blocks, 2)

	assert.Equal(t, "Line one\nLine two", blocks[0].Text)
	assert.Equal(t, "Single line", blocks[1].Text)
}

func TestParseSRT_HTMLTags(t *testing.T) {
	input := `1
00:00:01,000 --> 00:00:04,000
<i>Italic text</i>

2
00:00:05,000 --> 00:00:08,000
<b>Bold</b> and <i>italic</i>
`
	blocks, err := ParseSRT(input)
	require.NoError(t, err)
	require.Len(t, blocks, 2)

	// HTML tags should be preserved
	assert.Equal(t, "<i>Italic text</i>", blocks[0].Text)
	assert.Equal(t, "<b>Bold</b> and <i>italic</i>", blocks[1].Text)
}

func TestParseSRT_EmptyInput(t *testing.T) {
	blocks, err := ParseSRT("")
	require.NoError(t, err)
	assert.Empty(t, blocks)
}

func TestParseSRT_WithBOM(t *testing.T) {
	// UTF-8 BOM + SRT content
	input := "\xEF\xBB\xBF1\n00:00:01,000 --> 00:00:04,000\nHello\n"
	blocks, err := ParseSRT(input)
	require.NoError(t, err)
	require.Len(t, blocks, 1)
	assert.Equal(t, 1, blocks[0].Index)
	assert.Equal(t, "Hello", blocks[0].Text)
}

func TestParseSRT_WindowsLineEndings(t *testing.T) {
	input := "1\r\n00:00:01,000 --> 00:00:04,000\r\nHello\r\n\r\n2\r\n00:00:05,000 --> 00:00:08,000\r\nWorld\r\n"
	blocks, err := ParseSRT(input)
	require.NoError(t, err)
	require.Len(t, blocks, 2)
	assert.Equal(t, "Hello", blocks[0].Text)
	assert.Equal(t, "World", blocks[1].Text)
}

func TestSerializeSRT_Basic(t *testing.T) {
	blocks := []SubtitleBlock{
		{Index: 1, Start: "00:00:01,000", End: "00:00:04,000", Text: "Hello"},
		{Index: 2, Start: "00:00:05,000", End: "00:00:08,000", Text: "World"},
	}

	result := SerializeSRT(blocks)

	expected := "1\n00:00:01,000 --> 00:00:04,000\nHello\n\n2\n00:00:05,000 --> 00:00:08,000\nWorld\n"
	assert.Equal(t, expected, result)
}

func TestSerializeSRT_MultiLine(t *testing.T) {
	blocks := []SubtitleBlock{
		{Index: 1, Start: "00:00:01,000", End: "00:00:04,000", Text: "Line one\nLine two"},
	}

	result := SerializeSRT(blocks)
	assert.Contains(t, result, "Line one\nLine two")
}

func TestParseSRT_RoundTrip(t *testing.T) {
	original := `1
00:00:01,000 --> 00:00:04,000
Hello, how are you?

2
00:00:05,500 --> 00:00:08,200
I'm doing fine, thanks.

3
00:00:10,000 --> 00:00:14,000
Line one
Line two
`
	blocks, err := ParseSRT(original)
	require.NoError(t, err)

	serialized := SerializeSRT(blocks)

	// Re-parse and compare
	blocks2, err := ParseSRT(serialized)
	require.NoError(t, err)

	require.Len(t, blocks2, len(blocks))
	for i := range blocks {
		assert.Equal(t, blocks[i].Index, blocks2[i].Index, "block %d index", i)
		assert.Equal(t, blocks[i].Start, blocks2[i].Start, "block %d start", i)
		assert.Equal(t, blocks[i].End, blocks2[i].End, "block %d end", i)
		assert.Equal(t, blocks[i].Text, blocks2[i].Text, "block %d text", i)
	}
}

func TestSerializeSRT_Empty(t *testing.T) {
	result := SerializeSRT(nil)
	assert.Equal(t, "", result)
}

func TestParseSRT_ExtraBlankLines(t *testing.T) {
	// Some SRT files have extra blank lines between blocks
	input := `1
00:00:01,000 --> 00:00:04,000
Hello


2
00:00:05,000 --> 00:00:08,000
World

`
	blocks, err := ParseSRT(input)
	require.NoError(t, err)
	require.Len(t, blocks, 2)
	assert.Equal(t, "Hello", blocks[0].Text)
	assert.Equal(t, "World", blocks[1].Text)
}

func TestParseSRT_TimestampPreservation(t *testing.T) {
	// AC #3: timestamps must be preserved exactly
	input := `1
00:01:23,456 --> 00:01:27,890
Test

2
01:30:00,000 --> 01:30:05,500
Another test
`
	blocks, err := ParseSRT(input)
	require.NoError(t, err)

	serialized := SerializeSRT(blocks)

	assert.True(t, strings.Contains(serialized, "00:01:23,456 --> 00:01:27,890"),
		"timestamps must be preserved exactly")
	assert.True(t, strings.Contains(serialized, "01:30:00,000 --> 01:30:05,500"),
		"timestamps must be preserved exactly")
}
