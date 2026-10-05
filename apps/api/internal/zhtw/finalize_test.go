package zhtw

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeConverter stands in for OpenCC: it swaps the simplified characters of
// the one test sentence and records that it ran.
type fakeConverter struct {
	err   error
	calls int
}

func (f *fakeConverter) ConvertS2TWP(b []byte) ([]byte, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	r := strings.NewReplacer("这", "這", "个", "個", "软", "軟", "质", "質")
	return []byte(r.Replace(string(b))), nil
}

const simplified = "这个软件的质量很好"

func TestFinalize_ScriptThenVocabulary(t *testing.T) {
	conv := &fakeConverter{}
	out, err := Finalize(conv, simplified, []string{"US"})
	require.NoError(t, err)
	assert.Equal(t, 1, conv.calls)
	assert.Equal(t, "這個軟體的品質很好", out)
}

func TestFinalize_MainlandKeepsItsVocabulary(t *testing.T) {
	out, err := Finalize(&fakeConverter{}, simplified, []string{"us", " cn "})
	require.NoError(t, err)
	assert.Equal(t, "這個軟件的質量很好", out, "script converted, vocabulary untouched")
}

func TestFinalize_NilConverterOnlyRewritesVocabulary(t *testing.T) {
	out, err := Finalize(nil, "這個軟件的質量很好", nil)
	require.NoError(t, err)
	assert.Equal(t, "這個軟體的品質很好", out)
}

func TestFinalize_ConverterErrorReturnsInputThroughVocabulary(t *testing.T) {
	boom := errors.New("opencc down")
	out, err := Finalize(&fakeConverter{err: boom}, "這個軟件的質量很好", nil)
	require.ErrorIs(t, err, boom)
	assert.Equal(t, "這個軟體的品質很好", out)
}

func TestIsMainland(t *testing.T) {
	assert.True(t, IsMainland([]string{"US", "cn"}))
	assert.True(t, IsMainland([]string{" CN "}))
	assert.False(t, IsMainland([]string{"TW", "HK", "MO"}))
	assert.False(t, IsMainland(nil))
}
