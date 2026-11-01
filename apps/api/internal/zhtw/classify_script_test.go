package zhtw

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pairConverter maps a small fixed set of character pairs both ways, enough
// to model OpenCC on the test sentences below.
type pairConverter struct{ err error }

var (
	classifyS2T = strings.NewReplacer("这", "這", "个", "個", "软", "軟", "质", "質", "静", "靜", "们", "們", "说", "說", "话", "話", "师", "師", "门", "門", "长", "長")
	classifyT2S = strings.NewReplacer("這", "这", "個", "个", "軟", "软", "質", "质", "靜", "静", "們", "们", "說", "说", "話", "话", "師", "师", "門", "门", "長", "长")
)

func (c pairConverter) ConvertS2TWP(b []byte) ([]byte, error) { return c.ConvertS2TW(b) }
func (c pairConverter) ConvertS2TW(b []byte) ([]byte, error) {
	if c.err != nil {
		return nil, c.err
	}
	return []byte(classifyS2T.Replace(string(b))), nil
}
func (c pairConverter) ConvertT2S(b []byte) ([]byte, error) {
	if c.err != nil {
		return nil, c.err
	}
	return []byte(classifyT2S.Replace(string(b))), nil
}

func TestClassifyScript(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  string
	}{
		{"simplified", []string{"这个软件的质量很好", "他们说的话"}, ScriptSimplified},
		{"traditional", []string{"這個軟體的質量很好", "他們說的話"}, ScriptTraditional},
		// 《白髮》: Traditional burned subtitles with one stray simplified 静.
		{"traditional with one stray simplified char", []string{"師門長老們說", "此處很静"}, ScriptTraditional},
		{"no distinguishing characters", []string{"你好", "我在山上"}, ScriptUntold},
		{"tie", []string{"这個"}, ScriptUntold},
		{"no lines", nil, ScriptUntold},
		{"blank lines", []string{"", "  "}, ScriptUntold},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ClassifyScript(pairConverter{}, tt.lines)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClassifyScript_ConverterErrorIsReturned(t *testing.T) {
	_, err := ClassifyScript(pairConverter{err: errors.New("opencc missing")}, []string{"这个"})
	require.Error(t, err)
}
