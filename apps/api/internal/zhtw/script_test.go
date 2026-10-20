package zhtw

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scriptConverter adds the reverse profile to fakeConverter.
type scriptConverter struct{ fakeConverter }

var t2sChars = strings.NewReplacer("這", "这", "個", "个", "軟", "软", "質", "质", "蘇", "苏", "裡", "里")

func (s *scriptConverter) ConvertT2S(b []byte) ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	return []byte(t2sChars.Replace(string(b))), nil
}

func (s *scriptConverter) ConvertS2TW(b []byte) ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	// Like real OpenCC, s2tw also "fixes" a character in Traditional text.
	return []byte(strings.ReplaceAll(s2twChars.Replace(string(b)), "蘇里", "蘇裡")), nil
}

func TestSimplifiedFields(t *testing.T) {
	got, err := SimplifiedFields(&scriptConverter{}, []string{
		simplified,       // Simplified
		"這個軟體",           // Traditional, s2tw leaves it
		"傑克蘇里",           // Traditional that s2tw would still change
		"Tears of Steel", // no Chinese
		"这個",             // mixed scripts: left alone
	})
	require.NoError(t, err)
	assert.Equal(t, []bool{true, false, false, false, false}, got)
}

func TestSimplifiedFields_ErrorAndEmpty(t *testing.T) {
	got, err := SimplifiedFields(&scriptConverter{}, nil)
	require.NoError(t, err)
	assert.Empty(t, got)

	_, err = SimplifiedFields(&scriptConverter{fakeConverter{err: errors.New("boom")}}, []string{simplified})
	require.Error(t, err)
}

func TestConvertJoined_FieldCountMustSurvive(t *testing.T) {
	out, err := ConvertJoined(func(s string) (string, error) { return s2twChars.Replace(s), nil }, []string{"这", "个\n个"})
	require.NoError(t, err)
	assert.Equal(t, []string{"這", "個\n個"}, out, "a newline inside a field does not split it")

	_, err = ConvertJoined(func(string) (string, error) { return "merged", nil }, []string{"a", "b"})
	require.Error(t, err, "a conversion that swallows the separator is refused, not half-applied")
}
